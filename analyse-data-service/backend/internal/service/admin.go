package service

import (
	"context"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/adminquery"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

func (s *DataService) ListRawDocuments(ctx context.Context, request *v1.RawDocumentListRequest) (*v1.Response[v1.AdminRawDocumentPage], error) {
	page, err := v1.ParseBoundedInt(request.Page, 1, 1, 1_000_000, "page")
	if err != nil {
		return nil, err
	}
	pageSize, err := v1.ParseBoundedInt(request.PageSize, 50, 1, 100, "page_size")
	if err != nil {
		return nil, err
	}
	filter := adminquery.RawDocumentListRequest{
		Title: strings.TrimSpace(request.Title), SourceRef: strings.TrimSpace(request.SourceRef),
		IngestStatus: model.IngestStatus(request.IngestStatus), Page: page, PageSize: pageSize,
	}
	if filter.IngestStatus != "" && !oneOf(string(filter.IngestStatus), "collected", "duplicate", "failed", "pending_extract") {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "unsupported ingest_status")
	}
	if s == nil || s.dependencies.Admin == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "admin aggregate store is unavailable")
	}
	result, err := s.dependencies.Admin.ListRawDocuments(ctx, filter)
	if err != nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "admin raw-document aggregate failed")
	}
	items := make([]v1.AdminRawDocument, 0, len(result.Items))
	for _, document := range result.Items {
		items = append(items, rawDocumentDTO(document))
	}
	return &v1.Response[v1.AdminRawDocumentPage]{Status: v1.StatusOK, Result: v1.AdminRawDocumentPage{
		Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize,
	}}, nil
}

func (s *DataService) ListEvents(ctx context.Context, request *v1.EventListRequest) (*v1.Response[v1.AdminEventPage], error) {
	page, err := v1.ParseBoundedInt(request.Page, 1, 1, 1_000_000, "page")
	if err != nil {
		return nil, err
	}
	pageSize, err := v1.ParseBoundedInt(request.PageSize, 50, 1, 100, "page_size")
	if err != nil {
		return nil, err
	}
	filter := adminquery.EventListRequest{
		Title: strings.TrimSpace(request.Title), EventStatus: model.EventStatus(request.EventStatus),
		FactStatus: model.FactStatus(request.FactStatus), Page: page, PageSize: pageSize,
	}
	if filter.EventStatus != "" && !oneOf(string(filter.EventStatus), "candidate", "confirmed", "rejected") ||
		filter.FactStatus != "" && !oneOf(string(filter.FactStatus), "unverified", "verified", "disputed") {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "unsupported event or fact status")
	}
	for _, value := range []struct {
		name   string
		raw    string
		target **time.Time
	}{
		{name: "event_time_from", raw: request.EventTimeFrom, target: &filter.EventTimeFrom},
		{name: "event_time_to", raw: request.EventTimeTo, target: &filter.EventTimeTo},
		{name: "first_seen_from", raw: request.FirstSeenFrom, target: &filter.FirstSeenFrom},
		{name: "first_seen_to", raw: request.FirstSeenTo, target: &filter.FirstSeenTo},
	} {
		*value.target, err = optionalUTC(value.raw)
		if err != nil {
			return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", value.name+" must be a UTC RFC3339 timestamp")
		}
	}
	if s == nil || s.dependencies.Admin == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "admin aggregate store is unavailable")
	}
	result, err := s.dependencies.Admin.ListEvents(ctx, filter)
	if err != nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "admin event aggregate failed")
	}
	items := make([]v1.AdminEvent, 0, len(result.Items))
	for _, event := range result.Items {
		items = append(items, eventDTO(event))
	}
	return &v1.Response[v1.AdminEventPage]{Status: v1.StatusOK, Result: v1.AdminEventPage{
		Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize,
	}}, nil
}

func rawDocumentDTO(document model.RawDocument) v1.AdminRawDocument {
	return v1.AdminRawDocument{
		ID: document.ID, ContractVersion: document.ContractVersion, ArtifactID: document.ArtifactID,
		SourceRef: document.SourceRef, IngestChannel: document.IngestChannel, SourceType: document.SourceType,
		SourceName: document.SourceName, SourceURL: document.SourceURL, SourceExternalID: document.SourceExternalID,
		Title: document.Title, ContentText: document.ContentText, ContentLevel: document.ContentLevel,
		RawObjectURI: document.RawObjectURI, RawMIMEType: document.RawMIMEType, Language: document.Language,
		PublishedAt: formatOptionalTime(document.PublishedAt), CollectedAt: document.CollectedAt.UTC().Format(time.RFC3339Nano),
		IngestStatus: string(document.IngestStatus), ContentSHA256: document.ContentHash,
	}
}

func eventDTO(event model.Event) v1.AdminEvent {
	var primary *string
	if event.PrimarySourceID != "" {
		value := event.PrimarySourceID
		primary = &value
	}
	return v1.AdminEvent{
		ID: event.ID, Title: event.Title, Summary: event.Summary, EventTime: formatOptionalTime(event.EventTime),
		FirstSeenAt: event.FirstSeenAt.UTC().Format(time.RFC3339Nano), KnowableAt: formatOptionalTime(event.KnowableAt),
		EventStatus: string(event.EventStatus), FactStatus: string(event.FactStatus), DedupeKey: event.DedupeKey,
		PrimarySourceID: primary,
	}
}
