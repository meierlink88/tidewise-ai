package service

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	v1 "github.com/meierlink88/tidewise-ai/admin-portal/backend/api/admin/v1"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

var _ v1.AdminHTTPServer = (*AdminService)(nil)

type AdminService struct {
	admin *biz.Service
}

func NewAdminService(admin *biz.Service) *AdminService {
	return &AdminService{admin: admin}
}

func (s *AdminService) ListEvents(
	ctx context.Context,
	request *v1.ListEventsRequest,
) (*v1.EventListResponse, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	query, err := eventQuery(request)
	if err != nil {
		return nil, err
	}
	page, err := s.admin.ListEvents(ctx, query)
	if err != nil {
		return nil, v1.NewHTTPError(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
	items := make([]v1.Event, 0, len(page.Items))
	for _, value := range page.Items {
		items = append(items, event(value))
	}
	return &v1.EventListResponse{
		Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize,
	}, nil
}

func (s *AdminService) ListEvidences(ctx context.Context, request *v1.ListEvidencesRequest) (*v1.EvidenceListResponse, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	query, err := evidenceQuery(request)
	if err != nil {
		return nil, err
	}
	page, err := s.admin.ListEvidences(ctx, query)
	if err != nil {
		return nil, adminReadError(err)
	}
	items := make([]v1.Evidence, 0, len(page.Items))
	for _, value := range page.Items {
		items = append(items, evidence(value))
	}
	return &v1.EvidenceListResponse{Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}

func (s *AdminService) GetCollectionDocument(ctx context.Context, request *v1.GetCollectionDocumentRequest) (*v1.CollectionDocumentResponse, error) {
	if s == nil || s.admin == nil || request == nil || !rawEvidenceIDPattern.MatchString(request.RawEvidenceID) {
		return nil, v1.ErrInvalidRequest
	}
	document, err := s.admin.GetCollectionDocument(ctx, request.RawEvidenceID)
	if err != nil {
		if errors.Is(err, biz.ErrRawEvidenceNotFound) {
			return nil, v1.NewHTTPError(http.StatusNotFound, "RAW_EVIDENCE_NOT_FOUND", "raw evidence was not found")
		}
		return nil, adminReadError(err)
	}
	response := &v1.CollectionDocumentResponse{Available: document.Available}
	if document.Available {
		response.URL = &document.URL
	}
	return response, nil
}

func (s *AdminService) ListEvidenceCategories(ctx context.Context, request *v1.EmptyRequest) (*v1.EvidenceCategoryListResponse, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	values, err := s.admin.ListEvidenceCategories(ctx)
	if err != nil {
		return nil, adminReadError(err)
	}
	categories := make([]v1.EvidenceCategory, 0, len(values))
	for _, value := range values {
		categories = append(categories, evidenceCategory(value))
	}
	return &v1.EvidenceCategoryListResponse{Categories: categories}, nil
}

func (s *AdminService) ListSources(ctx context.Context, request *v1.ListSourcesRequest) (*v1.SourceListResponse, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	query, err := sourceQuery(request)
	if err != nil {
		return nil, err
	}
	page, err := s.admin.ListSources(ctx, query)
	if err != nil {
		return nil, adminReadError(err)
	}
	items := make([]v1.Source, 0, len(page.Items))
	for _, value := range page.Items {
		items = append(items, v1.Source{ID: value.ID, Code: value.Code, Name: value.Name, OwnershipType: value.OwnershipType,
			ChannelType: value.ChannelType, Enabled: value.Enabled, Priority: value.Priority,
			DefaultSourceLevel: value.DefaultSourceLevel, UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339Nano)})
	}
	return &v1.SourceListResponse{Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}

func evidenceQuery(request *v1.ListEvidencesRequest) (biz.EvidenceListQuery, error) {
	if utf8.RuneCountInString(strings.TrimSpace(request.Title)) > 500 || utf8.RuneCountInString(strings.TrimSpace(request.Summary)) > 200 || utf8.RuneCountInString(strings.TrimSpace(request.SourceID)) > 32 || utf8.RuneCountInString(strings.TrimSpace(request.SourceName)) > 100 {
		return biz.EvidenceListQuery{}, invalidRequest("Evidence text filter is too long")
	}
	if categoryID := strings.TrimSpace(request.CategoryID); categoryID != "" && !evidenceCategoryIDPattern.MatchString(categoryID) {
		return biz.EvidenceListQuery{}, invalidRequest("category_id is invalid")
	}
	publishedFrom, err := parseOptionalUTCTime(request.PublishedFrom)
	if err != nil {
		return biz.EvidenceListQuery{}, err
	}
	publishedTo, err := parseOptionalUTCTime(request.PublishedTo)
	if err != nil {
		return biz.EvidenceListQuery{}, err
	}
	collectedFrom, err := parseOptionalUTCTime(request.CollectedFrom)
	if err != nil {
		return biz.EvidenceListQuery{}, err
	}
	collectedTo, err := parseOptionalUTCTime(request.CollectedTo)
	if err != nil {
		return biz.EvidenceListQuery{}, err
	}
	if invalidRange(publishedFrom, publishedTo) || invalidRange(collectedFrom, collectedTo) {
		return biz.EvidenceListQuery{}, invalidRequest("time range start must not follow end")
	}
	isSplit, err := parseOptionalBool(request.IsSplit)
	if err != nil {
		return biz.EvidenceListQuery{}, err
	}
	if request.SourceLevel != "" && !validSourceLevel(request.SourceLevel) {
		return biz.EvidenceListQuery{}, invalidRequest("unsupported source level")
	}
	return biz.EvidenceListQuery{Title: strings.TrimSpace(request.Title), Summary: strings.TrimSpace(request.Summary), CategoryID: strings.TrimSpace(request.CategoryID),
		SourceID: strings.TrimSpace(request.SourceID), SourceName: strings.TrimSpace(request.SourceName), SourceLevel: strings.TrimSpace(request.SourceLevel), IsSplit: isSplit,
		PublishedFrom: publishedFrom, PublishedTo: publishedTo, CollectedFrom: collectedFrom, CollectedTo: collectedTo,
		Page: request.Page, PageSize: request.PageSize}, nil
}

func sourceQuery(request *v1.ListSourcesRequest) (biz.SourceListQuery, error) {
	if utf8.RuneCountInString(strings.TrimSpace(request.Query)) > 100 {
		return biz.SourceListQuery{}, invalidRequest("Source text filter is too long")
	}
	updatedFrom, err := parseOptionalUTCTime(request.UpdatedFrom)
	if err != nil {
		return biz.SourceListQuery{}, err
	}
	updatedTo, err := parseOptionalUTCTime(request.UpdatedTo)
	if err != nil {
		return biz.SourceListQuery{}, err
	}
	if invalidRange(updatedFrom, updatedTo) {
		return biz.SourceListQuery{}, invalidRequest("time range start must not follow end")
	}
	enabled, err := parseOptionalBool(request.Enabled)
	if err != nil {
		return biz.SourceListQuery{}, err
	}
	priority, err := parseOptionalPriority(request.Priority)
	if err != nil {
		return biz.SourceListQuery{}, err
	}
	if request.OwnershipType != "" && request.OwnershipType != "fixed" && request.OwnershipType != "dynamic" {
		return biz.SourceListQuery{}, invalidRequest("unsupported ownership type")
	}
	if request.ChannelType != "" && request.ChannelType != "web_search" && request.ChannelType != "api" && request.ChannelType != "rss" {
		return biz.SourceListQuery{}, invalidRequest("unsupported channel type")
	}
	if request.DefaultSourceLevel != "" && !validSourceLevel(request.DefaultSourceLevel) {
		return biz.SourceListQuery{}, invalidRequest("unsupported source level")
	}
	return biz.SourceListQuery{Text: strings.TrimSpace(request.Query), OwnershipType: request.OwnershipType, ChannelType: request.ChannelType,
		Enabled: enabled, Priority: priority, DefaultSourceLevel: request.DefaultSourceLevel,
		UpdatedFrom: updatedFrom, UpdatedTo: updatedTo, Page: request.Page, PageSize: request.PageSize}, nil
}

func parseOptionalUTCTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || parsed.Location() != time.UTC {
		return nil, invalidRequest("time query must use UTC RFC3339")
	}
	return &parsed, nil
}

func parseOptionalBool(value string) (*bool, error) {
	switch strings.TrimSpace(value) {
	case "":
		return nil, nil
	case "true":
		result := true
		return &result, nil
	case "false":
		result := false
		return &result, nil
	default:
		return nil, invalidRequest("boolean query must be true or false")
	}
}

func parseOptionalPriority(value string) (*int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if len(value) != 1 || value[0] < '1' || value[0] > '5' {
		return nil, invalidRequest("priority must be between 1 and 5")
	}
	result := int(value[0] - '0')
	return &result, nil
}

func invalidRange(from, to *time.Time) bool { return from != nil && to != nil && from.After(*to) }
func validSourceLevel(value string) bool {
	return value == "L1_OFFICIAL" || value == "L2_WIRE" || value == "L3_MEDIA" || value == "L4_SOCIAL"
}
func internalError() error {
	return v1.NewHTTPError(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func adminReadError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, biz.ErrDataServiceUnavailable) {
		return v1.NewHTTPError(http.StatusServiceUnavailable, "DATA_SERVICE_UNAVAILABLE", "data service is temporarily unavailable")
	}
	return internalError()
}

func evidence(value biz.Evidence) v1.Evidence {
	categories := make([]v1.EvidenceCategory, 0, len(value.Categories))
	for _, category := range value.Categories {
		categories = append(categories, evidenceCategory(category))
	}
	response := v1.Evidence{ID: value.ID, RawEvidenceID: value.RawEvidenceID, Title: value.Title, Summary: value.Summary,
		Semantic:   v1.EvidenceSemantic{Who: value.Semantic.Who, What: value.Semantic.What, When: value.Semantic.When, Where: value.Semantic.Where, Why: value.Semantic.Why, How: value.Semantic.How},
		Categories: categories, SourceID: value.SourceID, SourceName: value.SourceName, SourceLevel: value.SourceLevel, SourceURL: value.SourceURL,
		IsOriginal: value.IsOriginal, QuotedSourceName: value.QuotedSourceName, Keywords: append([]string{}, value.Keywords...), IsSplit: value.IsSplit,
		CollectedAt: value.CollectedAt.UTC().Format(time.RFC3339Nano)}
	if value.PublishedAt != nil {
		formatted := value.PublishedAt.UTC().Format(time.RFC3339Nano)
		response.PublishedAt = &formatted
	}
	return response
}

func evidenceCategory(value biz.EvidenceCategory) v1.EvidenceCategory {
	return v1.EvidenceCategory{ID: value.ID, Code: value.Code, Name: value.Name, Description: value.Description}
}

var evidenceCategoryIDPattern = regexp.MustCompile(`^EVC[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var rawEvidenceIDPattern = regexp.MustCompile(`^RAW[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func eventQuery(request *v1.ListEventsRequest) (biz.EventListQuery, error) {
	occurredFrom, err := parseOptionalTime(request.OccurredFrom)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	occurredTo, err := parseOptionalTime(request.OccurredTo)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	announcedFrom, err := parseOptionalTime(request.AnnouncedFrom)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	announcedTo, err := parseOptionalTime(request.AnnouncedTo)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	modality := biz.EventModality(request.Modality)
	if modality != "" && modality != biz.EventModalityFact && modality != biz.EventModalityPlan && modality != biz.EventModalitySpec {
		return biz.EventListQuery{}, invalidRequest("unsupported event modality")
	}
	status := biz.EventLifecycleStatus(request.Status)
	if status != "" && status != biz.EventLifecycleActive &&
		status != biz.EventLifecycleDeprecated && status != biz.EventLifecycleArchived {
		return biz.EventListQuery{}, invalidRequest("unsupported event status")
	}
	return biz.EventListQuery{
		Title: request.Title, Modality: modality, Status: status,
		OccurredFrom: occurredFrom, OccurredTo: occurredTo,
		AnnouncedFrom: announcedFrom, AnnouncedTo: announcedTo,
		Page: request.Page, PageSize: request.PageSize,
	}, nil
}

func parseOptionalTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, invalidRequest("time query must use RFC3339")
	}
	return &parsed, nil
}

func invalidRequest(message string) error {
	return v1.NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", message)
}

func event(value biz.Event) v1.Event {
	response := v1.Event{
		ID: value.ID, Title: value.Title, Summary: value.Summary,
		Semantic: v1.EventSemantic{
			Who: value.Semantic.Who, What: value.Semantic.What, When: value.Semantic.When,
			Where: value.Semantic.Where, Why: value.Semantic.Why, How: value.Semantic.How,
		},
		Modality: string(value.Modality), Status: string(value.Status),
	}
	if value.OccurredAt != nil {
		formatted := value.OccurredAt.Format(time.RFC3339)
		response.OccurredAt = &formatted
	}
	if value.AnnouncedAt != nil {
		formatted := value.AnnouncedAt.Format(time.RFC3339)
		response.AnnouncedAt = &formatted
	}
	return response
}
