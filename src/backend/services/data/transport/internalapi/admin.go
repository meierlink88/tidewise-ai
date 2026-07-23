package internalapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/adminquery"
)

func (d Dependencies) listAdminRawDocuments(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	page, pageSize, ok := pageQuery(response, request, requestID)
	if !ok {
		return
	}
	filter := adminquery.RawDocumentListRequest{Title: strings.TrimSpace(request.URL.Query().Get("title")), SourceRef: strings.TrimSpace(request.URL.Query().Get("source_ref")), IngestStatus: domain.IngestStatus(request.URL.Query().Get("ingest_status")), Page: page, PageSize: pageSize}
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
	filter := adminquery.EventListRequest{Title: strings.TrimSpace(request.URL.Query().Get("title")), EventStatus: domain.EventStatus(request.URL.Query().Get("event_status")), FactStatus: domain.FactStatus(request.URL.Query().Get("fact_status")), Page: page, PageSize: pageSize}
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

type adminRawDocument struct {
	ID               string  `json:"id"`
	ContractVersion  int     `json:"contract_version"`
	ArtifactID       string  `json:"artifact_id,omitempty"`
	SourceRef        string  `json:"source_ref,omitempty"`
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
	ContentSHA256    string  `json:"content_sha256"`
}

func rawDocumentDTO(document domain.RawDocument) adminRawDocument {
	return adminRawDocument{ID: document.ID, ContractVersion: document.ContractVersion, ArtifactID: document.ArtifactID, SourceRef: document.SourceRef, IngestChannel: document.IngestChannel, SourceType: document.SourceType, SourceName: document.SourceName, SourceURL: document.SourceURL, SourceExternalID: document.SourceExternalID, Title: document.Title, ContentText: document.ContentText, ContentLevel: document.ContentLevel, RawObjectURI: document.RawObjectURI, RawMIMEType: document.RawMIMEType, Language: document.Language, PublishedAt: formatOptionalTime(document.PublishedAt), CollectedAt: document.CollectedAt.UTC().Format(time.RFC3339Nano), IngestStatus: string(document.IngestStatus), ContentSHA256: document.ContentHash}
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
