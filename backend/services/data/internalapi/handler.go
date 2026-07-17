// Package internalapi owns the versioned Data Service HTTP transport.
package internalapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	eventapp "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/apps/miniappapi"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	domainimport "github.com/meierlink88/tidewise-ai/backend/internal/domain/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
	"github.com/meierlink88/tidewise-ai/backend/services/data/rawimport"
)

const (
	Namespace           = "/internal/data/v1"
	MaxRequestBodyBytes = 1_048_576

	ScopeResearchRead        = "data.research.read"
	ScopeAdminRead           = "data.admin.read"
	ScopeSourceMetadataRead  = "data.source-metadata.read"
	ScopeRawImport           = "data.raw-documents.import"
	ScopeReviewedEventImport = "data.reviewed-events.import"
)

type Principal struct {
	Identity string
	Scopes   []string
}

func (p Principal) HasScope(scope string) bool {
	for _, candidate := range p.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

type Credential struct {
	Secret    string
	Principal Principal
}

type Authenticator struct {
	credentials []Credential
}

func NewAuthenticator(credentials []Credential) (*Authenticator, error) {
	result := &Authenticator{credentials: make([]Credential, 0, len(credentials))}
	seenSecret := map[string]struct{}{}
	for _, credential := range credentials {
		credential.Secret = strings.TrimSpace(credential.Secret)
		credential.Principal.Identity = strings.TrimSpace(credential.Principal.Identity)
		if credential.Secret == "" || credential.Principal.Identity == "" || len(credential.Principal.Scopes) == 0 {
			return nil, fmt.Errorf("service credential, identity and scopes are required")
		}
		if _, duplicate := seenSecret[credential.Secret]; duplicate {
			return nil, fmt.Errorf("service credentials must be unique")
		}
		seenSecret[credential.Secret] = struct{}{}
		result.credentials = append(result.credentials, credential)
	}
	return result, nil
}

func (a *Authenticator) Authenticate(header string) (Principal, bool) {
	const prefix = "Bearer "
	if a == nil || !strings.HasPrefix(header, prefix) {
		return Principal{}, false
	}
	presented := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	for _, credential := range a.credentials {
		if len(presented) == len(credential.Secret) && subtle.ConstantTimeCompare([]byte(presented), []byte(credential.Secret)) == 1 {
			return credential.Principal, true
		}
	}
	return Principal{}, false
}

type RawImportService interface {
	Import(context.Context, string, string, rawimport.Batch) (rawimport.Result, error)
	Status(context.Context, string, string) (rawimport.ImportStatus, error)
}

type ReviewedEventService interface {
	Import(context.Context, domainimport.Package) (eventapp.Result, error)
}

type ResearchService interface {
	ListThemes(context.Context, miniappapi.ResearchListRequest) (miniappapi.ResearchThemeListResponse, error)
	GetTheme(context.Context, string, miniappapi.ResearchDetailRequest) (miniappapi.ResearchThemeDetailResponse, error)
	ListAnchors(context.Context, miniappapi.ResearchListRequest) (miniappapi.ResearchAnchorListResponse, error)
	GetAnchor(context.Context, string, miniappapi.ResearchDetailRequest) (miniappapi.ResearchAnchorDetailResponse, error)
}

type AdminStore interface {
	repositories.AdminQueryRepository
}

type SourceMetadataStore interface {
	ListSourceCatalogs(context.Context, repositories.SourceCatalogListFilter) ([]domain.SourceCatalog, error)
}

type Dependencies struct {
	Authenticator  *Authenticator
	RawImports     RawImportService
	ReviewedEvents ReviewedEventService
	Research       ResearchService
	Admin          AdminStore
	SourceMetadata SourceMetadataStore
	NewRequestID   func() string
}

type operation func(http.ResponseWriter, *http.Request, Principal, string)

func NewHandler(dependencies Dependencies) http.Handler {
	if dependencies.NewRequestID == nil {
		dependencies.NewRequestID = func() string { return fmt.Sprintf("data-%d", time.Now().UTC().UnixNano()) }
	}
	mux := http.NewServeMux()
	mux.Handle("POST "+Namespace+"/raw-document-imports", dependencies.authorize(ScopeRawImport, dependencies.importRawDocuments))
	mux.Handle("GET "+Namespace+"/raw-document-imports/{idempotency_key}", dependencies.authorize(ScopeRawImport, dependencies.rawImportStatus))
	mux.Handle("POST "+Namespace+"/reviewed-event-imports", dependencies.authorize(ScopeReviewedEventImport, dependencies.importReviewedEvent))
	mux.Handle("GET "+Namespace+"/research/themes", dependencies.authorize(ScopeResearchRead, dependencies.listResearchThemes))
	mux.Handle("GET "+Namespace+"/research/themes/{theme_id}", dependencies.authorize(ScopeResearchRead, dependencies.getResearchTheme))
	mux.Handle("GET "+Namespace+"/research/anchors", dependencies.authorize(ScopeResearchRead, dependencies.listResearchAnchors))
	mux.Handle("GET "+Namespace+"/research/anchors/{anchor_id}", dependencies.authorize(ScopeResearchRead, dependencies.getResearchAnchor))
	mux.Handle("GET "+Namespace+"/admin/raw-documents", dependencies.authorize(ScopeAdminRead, dependencies.listAdminRawDocuments))
	mux.Handle("GET "+Namespace+"/admin/events", dependencies.authorize(ScopeAdminRead, dependencies.listAdminEvents))
	mux.Handle("GET "+Namespace+"/admin/source-catalogs", dependencies.authorize(ScopeAdminRead, dependencies.listAdminSources))
	mux.Handle("GET "+Namespace+"/agent-run/source-metadata", dependencies.authorize(ScopeSourceMetadataRead, dependencies.listSourceMetadata))
	return mux
}

func (d Dependencies) authorize(scope string, next operation) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := strings.TrimSpace(request.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 128 {
			requestID = d.NewRequestID()
		}
		response.Header().Set("X-Request-ID", requestID)
		principal, ok := d.Authenticator.Authenticate(request.Header.Get("Authorization"))
		if !ok {
			writeError(response, requestID, http.StatusUnauthorized, "UNAUTHENTICATED", "valid service identity is required")
			return
		}
		if !principal.HasScope(scope) {
			writeError(response, requestID, http.StatusForbidden, "FORBIDDEN", "service identity lacks the required scope")
			return
		}
		defer func() {
			if recover() != nil {
				writeError(response, requestID, http.StatusInternalServerError, "INTERNAL_ERROR", "internal data service error")
			}
		}()
		next(response, request, principal, requestID)
	})
}

func (d Dependencies) importRawDocuments(response http.ResponseWriter, request *http.Request, principal Principal, requestID string) {
	if d.RawImports == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "raw import service is unavailable")
		return
	}
	var input struct {
		IdempotencyKey string                `json:"idempotency_key"`
		Items          []rawimport.Candidate `json:"items"`
	}
	if err := decodeStrictLimited(response, request, &input); err != nil {
		writeDecodeError(response, requestID, err)
		return
	}
	result, err := d.RawImports.Import(request.Context(), principal.Identity, input.IdempotencyKey, rawimport.Batch{Items: input.Items})
	if err != nil {
		writeRawImportError(response, requestID, err)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	writeEnvelope(response, status, requestID, result)
}

func (d Dependencies) rawImportStatus(response http.ResponseWriter, request *http.Request, principal Principal, requestID string) {
	if d.RawImports == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "raw import service is unavailable")
		return
	}
	key := strings.TrimSpace(request.PathValue("idempotency_key"))
	if key == "" || utf8.RuneCountInString(key) > 200 {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "idempotency_key must contain 1..200 characters")
		return
	}
	status, err := d.RawImports.Status(request.Context(), principal.Identity, key)
	if err != nil {
		writeRawImportError(response, requestID, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"request_id": requestID, "status": status.State, "result": status.Result})
}

func (d Dependencies) importReviewedEvent(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	if d.ReviewedEvents == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "reviewed event import service is unavailable")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBodyBytes)
	pkg, err := domainimport.DecodeStrict(request.Body)
	if err != nil {
		writeDecodeError(response, requestID, err)
		return
	}
	if _, err := pkg.Validate(); err != nil {
		writeError(response, requestID, http.StatusUnprocessableEntity, "REVIEWED_EVENT_IMPORT_REJECTED", "reviewed event package failed validation")
		return
	}
	result, err := d.ReviewedEvents.Import(request.Context(), pkg)
	if err != nil {
		if errors.Is(err, eventapp.ErrIdempotencyConflict) {
			writeError(response, requestID, http.StatusConflict, "EVENT_IMPORT_IDEMPOTENCY_CONFLICT", "idempotency key conflicts with reviewed event payload")
			return
		}
		writeError(response, requestID, http.StatusInternalServerError, "REVIEWED_EVENT_IMPORT_FAILED", "reviewed event import failed")
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	writeEnvelope(response, status, requestID, reviewedResult(result))
}

func (d Dependencies) listResearchThemes(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	window, limit, ok := researchListQuery(response, request, requestID)
	if !ok {
		return
	}
	if d.Research == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
		return
	}
	result, err := d.Research.ListThemes(request.Context(), miniappapi.ResearchListRequest{WindowHours: window, Limit: limit, Cursor: request.URL.Query().Get("cursor")})
	writeResearchResult(response, requestID, result, err)
}

func (d Dependencies) getResearchTheme(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	window, ok := optionalInt(response, requestID, request.URL.Query().Get("window_hours"), miniappapi.DefaultResearchWindowHours, miniappapi.MinResearchWindowHours, miniappapi.MaxResearchWindowHours, "window_hours")
	if !ok {
		return
	}
	if d.Research == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
		return
	}
	result, err := d.Research.GetTheme(request.Context(), request.PathValue("theme_id"), miniappapi.ResearchDetailRequest{WindowHours: window})
	if err != nil {
		writeResearchError(response, requestID, err)
		return
	}
	writeEnvelope(response, http.StatusOK, requestID, map[string]any{"theme": result.ResearchThemeItem, "events": result.Events})
}

func (d Dependencies) listResearchAnchors(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	window, limit, ok := researchListQuery(response, request, requestID)
	if !ok {
		return
	}
	if d.Research == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
		return
	}
	result, err := d.Research.ListAnchors(request.Context(), miniappapi.ResearchListRequest{WindowHours: window, Limit: limit, Cursor: request.URL.Query().Get("cursor")})
	writeResearchResult(response, requestID, result, err)
}

func (d Dependencies) getResearchAnchor(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	window, ok := optionalInt(response, requestID, request.URL.Query().Get("window_hours"), miniappapi.DefaultResearchWindowHours, miniappapi.MinResearchWindowHours, miniappapi.MaxResearchWindowHours, "window_hours")
	if !ok {
		return
	}
	if d.Research == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
		return
	}
	result, err := d.Research.GetAnchor(request.Context(), request.PathValue("anchor_id"), miniappapi.ResearchDetailRequest{WindowHours: window})
	if err != nil {
		writeResearchError(response, requestID, err)
		return
	}
	writeEnvelope(response, http.StatusOK, requestID, map[string]any{"anchor": result.ResearchAnchorItem, "events": result.Events})
}

func (d Dependencies) listAdminRawDocuments(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	page, pageSize, ok := pageQuery(response, request, requestID)
	if !ok {
		return
	}
	filter := repositories.RawDocumentListFilter{Title: strings.TrimSpace(request.URL.Query().Get("title")), SourceID: strings.TrimSpace(request.URL.Query().Get("source_id")), IngestStatus: domain.IngestStatus(request.URL.Query().Get("ingest_status")), Page: page, PageSize: pageSize}
	if filter.SourceID != "" && !repositories.IsUUID(filter.SourceID) {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "source_id must be a UUID")
		return
	}
	if filter.IngestStatus != "" && !oneOf(string(filter.IngestStatus), "collected", "duplicate", "failed", "pending_extract") {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "unsupported ingest_status")
		return
	}
	if d.Admin == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "admin aggregate store is unavailable")
		return
	}
	pageResult, err := d.Admin.ListRawDocuments(request.Context(), filter)
	if err != nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "admin raw-document aggregate failed")
		return
	}
	items := make([]adminRawDocument, 0, len(pageResult.Items))
	for _, document := range pageResult.Items {
		items = append(items, rawDocumentDTO(document))
	}
	writeEnvelope(response, http.StatusOK, requestID, map[string]any{"items": items, "total": pageResult.Total, "page": pageResult.Page, "page_size": pageResult.PageSize})
}

func (d Dependencies) listAdminEvents(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	page, pageSize, ok := pageQuery(response, request, requestID)
	if !ok {
		return
	}
	filter := repositories.EventListFilter{Title: strings.TrimSpace(request.URL.Query().Get("title")), EventStatus: domain.EventStatus(request.URL.Query().Get("event_status")), FactStatus: domain.FactStatus(request.URL.Query().Get("fact_status")), Page: page, PageSize: pageSize}
	if filter.EventStatus != "" && !oneOf(string(filter.EventStatus), "candidate", "confirmed", "rejected") || filter.FactStatus != "" && !oneOf(string(filter.FactStatus), "unverified", "verified", "disputed") {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "unsupported event or fact status")
		return
	}
	var err error
	for value, target := range map[string]**time.Time{
		"event_time_from": &filter.EventTimeFrom, "event_time_to": &filter.EventTimeTo,
		"first_seen_from": &filter.FirstSeenFrom, "first_seen_to": &filter.FirstSeenTo,
	} {
		*target, err = optionalUTC(request.URL.Query().Get(value))
		if err != nil {
			writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", value+" must be a UTC RFC3339 timestamp")
			return
		}
	}
	if d.Admin == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "admin aggregate store is unavailable")
		return
	}
	pageResult, err := d.Admin.ListEvents(request.Context(), filter)
	if err != nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "admin event aggregate failed")
		return
	}
	items := make([]adminEvent, 0, len(pageResult.Items))
	for _, event := range pageResult.Items {
		items = append(items, eventDTO(event))
	}
	writeEnvelope(response, http.StatusOK, requestID, map[string]any{"items": items, "total": pageResult.Total, "page": pageResult.Page, "page_size": pageResult.PageSize})
}

func (d Dependencies) listAdminSources(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	status := domain.SourceCatalogStatus(request.URL.Query().Get("status"))
	if status != "" && !oneOf(string(status), "active", "inactive", "disabled") {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "unsupported source status")
		return
	}
	if d.Admin == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "admin aggregate store is unavailable")
		return
	}
	sources, err := d.Admin.ListSourceCatalogs(request.Context(), repositories.SourceCatalogListFilter{Status: status})
	if err != nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "admin source aggregate failed")
		return
	}
	items := make([]adminSource, 0, len(sources))
	for _, source := range sources {
		items = append(items, adminSourceDTO(source))
	}
	writeEnvelope(response, http.StatusOK, requestID, map[string]any{"items": items})
}

func (d Dependencies) listSourceMetadata(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	limit, ok := optionalInt(response, requestID, request.URL.Query().Get("limit"), 20, 1, 100, "limit")
	if !ok {
		return
	}
	status := domain.SourceCatalogStatus(request.URL.Query().Get("status"))
	if status != "" && !oneOf(string(status), "active", "inactive", "disabled") {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "unsupported source status")
		return
	}
	if d.SourceMetadata == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "source metadata store is unavailable")
		return
	}
	sources, err := d.SourceMetadata.ListSourceCatalogs(request.Context(), repositories.SourceCatalogListFilter{Status: status})
	if err != nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "source metadata aggregate failed")
		return
	}
	cursor := request.URL.Query().Get("cursor")
	start := 0
	if cursor != "" {
		found := false
		for index, source := range sources {
			if source.ID == cursor {
				start = index + 1
				found = true
				break
			}
		}
		if !found {
			writeError(response, requestID, http.StatusBadRequest, "INVALID_CURSOR", "source metadata cursor is invalid")
			return
		}
	}
	end := start + limit
	if end > len(sources) {
		end = len(sources)
	}
	items := make([]sourceMetadata, 0, end-start)
	for _, source := range sources[start:end] {
		items = append(items, sourceMetadataDTO(source))
	}
	var nextCursor *string
	if end < len(sources) && len(items) > 0 {
		value := sources[end-1].ID
		nextCursor = &value
	}
	writeEnvelope(response, http.StatusOK, requestID, map[string]any{"items": items, "next_cursor": nextCursor})
}

func decodeStrictLimited(response http.ResponseWriter, request *http.Request, target any) error {
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func writeDecodeError(response http.ResponseWriter, requestID string, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) || strings.Contains(err.Error(), "request body too large") {
		writeError(response, requestID, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1048576 bytes")
		return
	}
	writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract")
}

func writeRawImportError(response http.ResponseWriter, requestID string, err error) {
	switch rawimport.ErrorCode(err) {
	case rawimport.CodeIdempotencyConflict, rawimport.CodeIdentityConflict, rawimport.CodeBatchCollision:
		writeError(response, requestID, http.StatusConflict, rawimport.ErrorCode(err), err.Error())
	case rawimport.CodeInvalidRequest:
		writeError(response, requestID, http.StatusUnprocessableEntity, rawimport.ErrorCode(err), err.Error())
	default:
		writeError(response, requestID, http.StatusInternalServerError, "RAW_DOCUMENT_IMPORT_FAILED", "raw document import failed")
	}
}

func researchListQuery(response http.ResponseWriter, request *http.Request, requestID string) (int, int, bool) {
	window, ok := optionalInt(response, requestID, request.URL.Query().Get("window_hours"), miniappapi.DefaultResearchWindowHours, miniappapi.MinResearchWindowHours, miniappapi.MaxResearchWindowHours, "window_hours")
	if !ok {
		return 0, 0, false
	}
	limit, ok := optionalInt(response, requestID, request.URL.Query().Get("limit"), miniappapi.DefaultResearchLimit, 1, miniappapi.MaxResearchLimit, "limit")
	return window, limit, ok
}

func pageQuery(response http.ResponseWriter, request *http.Request, requestID string) (int, int, bool) {
	page, ok := optionalInt(response, requestID, request.URL.Query().Get("page"), 1, 1, 1_000_000, "page")
	if !ok {
		return 0, 0, false
	}
	pageSize, ok := optionalInt(response, requestID, request.URL.Query().Get("page_size"), 50, 1, 100, "page_size")
	return page, pageSize, ok
}

func optionalInt(response http.ResponseWriter, requestID, raw string, fallback, minimum, maximum int, name string) (int, bool) {
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("%s must be between %d and %d", name, minimum, maximum))
		return 0, false
	}
	return value, true
}

func optionalUTC(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	_, offset := value.Zone()
	if offset != 0 {
		return nil, fmt.Errorf("timestamp is not UTC")
	}
	value = value.UTC()
	return &value, nil
}

func writeResearchResult(response http.ResponseWriter, requestID string, result any, err error) {
	if err != nil {
		writeResearchError(response, requestID, err)
		return
	}
	writeEnvelope(response, http.StatusOK, requestID, result)
}

func writeResearchError(response http.ResponseWriter, requestID string, err error) {
	switch {
	case errors.Is(err, miniappapi.ErrInvalidResearchRequest):
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, repositories.ErrResearchNotFound):
		writeError(response, requestID, http.StatusNotFound, "NOT_FOUND", "research aggregate was not found")
	default:
		writeError(response, requestID, http.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "research aggregate failed")
	}
}

type adminRawDocument struct {
	ID               string  `json:"id"`
	SourceID         string  `json:"source_id"`
	IngestChannel    string  `json:"ingest_channel"`
	SourceType       string  `json:"source_type"`
	SourceName       string  `json:"source_name"`
	SourceURL        string  `json:"source_url"`
	SourceExternalID string  `json:"source_external_id,omitempty"`
	Title            string  `json:"title"`
	ContentText      string  `json:"content_text"`
	ContentLevel     string  `json:"content_level"`
	RawObjectURI     string  `json:"raw_object_uri"`
	RawMIMEType      string  `json:"raw_mime_type"`
	Language         string  `json:"language"`
	PublishedAt      *string `json:"published_at"`
	CollectedAt      string  `json:"collected_at"`
	IngestStatus     string  `json:"ingest_status"`
}

func rawDocumentDTO(document domain.RawDocument) adminRawDocument {
	return adminRawDocument{ID: document.ID, SourceID: document.SourceID, IngestChannel: document.IngestChannel, SourceType: document.SourceType, SourceName: document.SourceName, SourceURL: document.SourceURL, SourceExternalID: document.SourceExternalID, Title: document.Title, ContentText: document.ContentText, ContentLevel: document.ContentLevel, RawObjectURI: document.RawObjectURI, RawMIMEType: document.RawMIMEType, Language: document.Language, PublishedAt: formatOptionalTime(document.PublishedAt), CollectedAt: document.CollectedAt.UTC().Format(time.RFC3339Nano), IngestStatus: string(document.IngestStatus)}
}

type adminEvent struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Summary         string  `json:"summary"`
	EventTime       *string `json:"event_time"`
	FirstSeenAt     string  `json:"first_seen_at"`
	KnowableAt      *string `json:"knowable_at"`
	EventStatus     string  `json:"event_status"`
	FactStatus      string  `json:"fact_status"`
	DedupeKey       string  `json:"dedupe_key"`
	PrimarySourceID *string `json:"primary_source_id"`
}

func eventDTO(event domain.Event) adminEvent {
	var primary *string
	if event.PrimarySourceID != "" {
		value := event.PrimarySourceID
		primary = &value
	}
	return adminEvent{ID: event.ID, Title: event.Title, Summary: event.Summary, EventTime: formatOptionalTime(event.EventTime), FirstSeenAt: event.FirstSeenAt.UTC().Format(time.RFC3339Nano), KnowableAt: formatOptionalTime(event.KnowableAt), EventStatus: string(event.EventStatus), FactStatus: string(event.FactStatus), DedupeKey: event.DedupeKey, PrimarySourceID: primary}
}

type adminSource struct {
	ID            string `json:"id"`
	IngestChannel string `json:"ingest_channel"`
	ProviderKey   string `json:"provider_key"`
	ConnectorKey  string `json:"connector_key"`
	ParserKey     string `json:"parser_key"`
	SourceType    string `json:"source_type"`
	SourceName    string `json:"source_name"`
	SourceURL     string `json:"source_url"`
	SourceLevel   string `json:"source_level"`
	TopicHint     string `json:"topic_hint"`
	UsagePolicy   string `json:"usage_policy"`
	Status        string `json:"status"`
}

func adminSourceDTO(source domain.SourceCatalog) adminSource {
	return adminSource{ID: source.ID, IngestChannel: source.IngestChannel, ProviderKey: source.ProviderKey, ConnectorKey: source.ConnectorKey, ParserKey: source.ParserKey, SourceType: source.SourceType, SourceName: source.SourceName, SourceURL: source.SourceURL, SourceLevel: source.SourceLevel, TopicHint: source.TopicHint, UsagePolicy: source.UsagePolicy, Status: string(source.Status)}
}

type sourceMetadata struct {
	ID             string         `json:"id"`
	IngestChannel  string         `json:"ingest_channel"`
	ProviderKey    string         `json:"provider_key"`
	ConnectorKey   string         `json:"connector_key"`
	ParserKey      string         `json:"parser_key"`
	SourceType     string         `json:"source_type"`
	SourceName     string         `json:"source_name"`
	SourceURL      string         `json:"source_url"`
	AuthType       string         `json:"auth_type"`
	CredentialRef  *string        `json:"credential_ref"`
	ApprovedConfig map[string]any `json:"approved_config"`
	RateLimitHint  map[string]int `json:"rate_limit_hint"`
	UsagePolicy    string         `json:"usage_policy"`
	Status         string         `json:"status"`
}

func sourceMetadataDTO(source domain.SourceCatalog) sourceMetadata {
	approved := map[string]any{}
	for _, key := range []string{"collection_mode", "route_template", "prompt_ref", "prompt_version", "language", "result_limit", "timeout_seconds"} {
		if value, ok := source.SourceConfig[key]; ok {
			approved[key] = value
		}
	}
	if source.RouteTemplate != "" {
		approved["route_template"] = source.RouteTemplate
	}
	requests := positiveInt(source.RateLimitPolicy["requests"])
	window := positiveInt(source.RateLimitPolicy["window_seconds"])
	if requests == 0 {
		requests = positiveInt(source.RateLimitPolicy["requests_per_minute"])
		if requests > 0 {
			window = 60
		}
	}
	if requests == 0 {
		requests = 1
	}
	if window == 0 {
		window = 60
	}
	var credentialRef *string
	if source.CredentialRef != "" {
		value := source.CredentialRef
		credentialRef = &value
	}
	return sourceMetadata{ID: source.ID, IngestChannel: source.IngestChannel, ProviderKey: source.ProviderKey, ConnectorKey: source.ConnectorKey, ParserKey: source.ParserKey, SourceType: source.SourceType, SourceName: source.SourceName, SourceURL: source.SourceURL, AuthType: source.AuthType, CredentialRef: credentialRef, ApprovedConfig: approved, RateLimitHint: map[string]int{"requests": requests, "window_seconds": window}, UsagePolicy: source.UsagePolicy, Status: string(source.Status)}
}

func reviewedResult(result eventapp.Result) map[string]any {
	return map[string]any{
		"package_id": result.PackageID, "receipt_id": result.ReceiptID, "event_id": result.EventID,
		"raw_document_ids": result.RawDocumentIDs, "event_source_ids": result.EventSourceIDs,
		"event_tag_map_ids": result.EventTagMapIDs, "payload_hash": result.PayloadHash,
		"counts": map[string]int{"raw_documents": len(result.RawDocumentIDs), "events": 1, "event_sources": len(result.EventSourceIDs), "event_tags": len(result.EventTagMapIDs), "receipts": 1},
	}
}

func positiveInt(value any) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	}
	return 0
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func writeEnvelope(response http.ResponseWriter, status int, requestID string, result any) {
	writeJSON(response, status, map[string]any{"request_id": requestID, "result": result})
}

func writeError(response http.ResponseWriter, requestID string, status int, code, message string) {
	writeJSON(response, status, map[string]any{"request_id": requestID, "error": map[string]any{"code": code, "message": message, "details": map[string]any{}}})
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}
