package service

import (
	"context"
	"net/http"
	"strings"
	"time"

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

func (s *AdminService) ListRawDocuments(
	ctx context.Context,
	request *v1.ListRawDocumentsRequest,
) (*v1.RawDocumentListResponse, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	page, err := s.admin.ListRawDocuments(ctx, biz.RawDocumentListQuery{
		Title: request.Title, SourceRef: request.SourceRef, Page: request.Page, PageSize: request.PageSize,
	})
	if err != nil {
		return nil, v1.NewHTTPError(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
	items := make([]v1.RawDocument, 0, len(page.Items))
	for _, document := range page.Items {
		items = append(items, rawDocument(document))
	}
	return &v1.RawDocumentListResponse{
		Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize,
	}, nil
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

func eventQuery(request *v1.ListEventsRequest) (biz.EventListQuery, error) {
	eventTimeFrom, err := parseOptionalTime(request.EventTimeFrom)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	eventTimeTo, err := parseOptionalTime(request.EventTimeTo)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	firstSeenFrom, err := parseOptionalTime(request.FirstSeenFrom)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	firstSeenTo, err := parseOptionalTime(request.FirstSeenTo)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	eventStatus := biz.EventStatus(request.EventStatus)
	if eventStatus != "" && eventStatus != biz.EventStatusCandidate &&
		eventStatus != biz.EventStatusConfirmed && eventStatus != biz.EventStatusRejected {
		return biz.EventListQuery{}, invalidRequest("unsupported event status")
	}
	factStatus := biz.FactStatus(request.FactStatus)
	if factStatus != "" && factStatus != biz.FactStatusUnverified &&
		factStatus != biz.FactStatusVerified && factStatus != biz.FactStatusDisputed {
		return biz.EventListQuery{}, invalidRequest("unsupported fact status")
	}
	return biz.EventListQuery{
		Title: request.Title, EventStatus: eventStatus, FactStatus: factStatus,
		EventTimeFrom: eventTimeFrom, EventTimeTo: eventTimeTo,
		FirstSeenFrom: firstSeenFrom, FirstSeenTo: firstSeenTo,
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

func rawDocument(value biz.RawDocument) v1.RawDocument {
	response := v1.RawDocument{
		ID: value.ID, ContractVersion: value.ContractVersion, ArtifactID: value.ArtifactID,
		SourceRef: value.SourceRef, IngestChannel: value.IngestChannel, SourceType: value.SourceType,
		SourceName: value.SourceName, SourceURL: value.SourceURL, SourceExternalID: value.SourceExternalID,
		Title: value.Title, ContentText: value.ContentText, RawObjectURI: value.RawObjectURI,
		RawMIMEType: value.RawMIMEType, Language: value.Language,
		CollectedAt: value.CollectedAt.Format(time.RFC3339), IngestStatus: string(value.IngestStatus),
		ContentSHA256: value.ContentSHA256,
	}
	if value.PublishedAt != nil {
		response.PublishedAt = value.PublishedAt.Format(time.RFC3339)
	}
	return response
}

func event(value biz.Event) v1.Event {
	response := v1.Event{
		ID: value.ID, Title: value.Title, Summary: value.Summary,
		FirstSeenAt: value.FirstSeenAt.Format(time.RFC3339),
		EventStatus: string(value.EventStatus), FactStatus: string(value.FactStatus),
		DedupeKey: value.DedupeKey,
	}
	if value.EventTime != nil {
		response.EventTime = value.EventTime.Format(time.RFC3339)
	}
	if value.KnowableAt != nil {
		response.KnowableAt = value.KnowableAt.Format(time.RFC3339)
	}
	return response
}
