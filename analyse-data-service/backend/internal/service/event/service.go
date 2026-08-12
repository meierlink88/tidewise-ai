package event

import (
	"context"
	"errors"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	eventapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/event"
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/event"
)

type UseCase interface {
	Import(context.Context, string, eventbiz.PublicationBatch) (eventbiz.Result, error)
	ActiveTags(context.Context) (eventbiz.EventTagCatalog, error)
	ListEvents(context.Context, eventbiz.EventListRequest) (eventbiz.EventPage, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Event use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) PublishReviewedEvents(ctx context.Context, request *eventapi.PublicationRequest) (*v1.Response[eventapi.PublicationResult], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "Event Publication request is required")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, eventapi.ErrorDataServiceNotReady, "Event Publication service is unavailable")
	}
	result, err := s.useCase.Import(ctx, principalIdentity(ctx), publicationInput(request))
	if err == nil {
		return &v1.Response[eventapi.PublicationResult]{Status: v1.StatusCreated, Result: publicationResult(result)}, nil
	}
	var validation *eventbiz.ValidationError
	if errors.As(err, &validation) {
		return nil, publicErrorWithDetails(v1.StatusUnprocessableEntity, eventapi.ErrorEventPublicationInvalid, "Event Publication failed validation", map[string]any{"issues": validation.Issues})
	}
	var conflict *eventbiz.ConflictError
	if errors.As(err, &conflict) {
		return nil, publicErrorWithDetails(v1.StatusConflict, eventapi.ErrorEventPublicationConflict, "Event Publication conflicts with stored data", map[string]any{"issues": conflict.Issues})
	}
	return nil, publicError(v1.StatusInternalServerError, eventapi.ErrorEventPublicationFailed, "Event Publication failed")
}

func (s *Service) ListActiveEventTags(ctx context.Context, request *eventapi.TagCatalogRequest) (*v1.Response[eventapi.TagCatalog], error) {
	if request == nil || !request.Active {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "active must be exactly true")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, eventapi.ErrorDataServiceNotReady, "Event Tag Catalog service is unavailable")
	}
	catalog, err := s.useCase.ActiveTags(ctx)
	if err != nil {
		return nil, publicError(v1.StatusInternalServerError, eventapi.ErrorEventTagCatalogFailed, "Event Tag Catalog is unavailable")
	}
	tags := make([]eventapi.TagCatalogItem, len(catalog.Tags))
	for position, tag := range catalog.Tags {
		tags[position] = eventapi.TagCatalogItem{
			ID: tag.ID, TagKind: tag.Kind, Code: tag.Code, Name: tag.Name, IsActive: tag.Active,
		}
	}
	return &v1.Response[eventapi.TagCatalog]{Status: v1.StatusOK, Result: eventapi.TagCatalog{
		Tags: tags,
	}}, nil
}

func (s *Service) ListEvents(ctx context.Context, request *eventapi.ListRequest) (*v1.Response[eventapi.Page], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "Event query is required")
	}
	page, err := v1.ParseBoundedInt(request.Page, 1, 1, 1_000_000, "page")
	if err != nil {
		return nil, err
	}
	pageSize, err := v1.ParseBoundedInt(request.PageSize, 50, 1, 100, "page_size")
	if err != nil {
		return nil, err
	}
	filter := eventbiz.EventListRequest{
		Title: strings.TrimSpace(request.Title), EventStatus: eventbiz.EventStatus(request.EventStatus),
		FactStatus: eventbiz.FactStatus(request.FactStatus), Page: page, PageSize: pageSize,
	}
	if filter.EventStatus != "" && !oneOf(string(filter.EventStatus), string(eventbiz.EventStatusCandidate), string(eventbiz.EventStatusConfirmed), string(eventbiz.EventStatusRejected)) ||
		filter.FactStatus != "" && !oneOf(string(filter.FactStatus), string(eventbiz.FactStatusUnverified), string(eventbiz.FactStatusVerified), string(eventbiz.FactStatusDisputed)) {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "unsupported event or fact status")
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
			return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, value.name+" must be a UTC RFC3339 timestamp")
		}
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, eventapi.ErrorDataServiceNotReady, "Event service is unavailable")
	}
	result, err := s.useCase.ListEvents(ctx, filter)
	if err != nil {
		return nil, publicError(v1.StatusInternalServerError, eventapi.ErrorDataRepositoryFailure, "admin event aggregate failed")
	}
	items := make([]eventapi.Item, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, eventapi.Item{
			ID: item.ID, Title: item.Title, Summary: item.Summary, EventTime: formatOptionalTime(item.EventTime),
			FirstSeenAt: item.FirstSeenAt.UTC().Format(time.RFC3339Nano), KnowableAt: formatOptionalTime(item.KnowableAt),
			EventStatus: string(item.EventStatus), FactStatus: string(item.FactStatus), DedupeKey: item.DedupeKey,
		})
	}
	return &v1.Response[eventapi.Page]{Status: v1.StatusOK, Result: eventapi.Page{
		Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize,
	}}, nil
}

func publicationInput(request *eventapi.PublicationRequest) eventbiz.PublicationBatch {
	collectors := make([]eventbiz.CollectorExecution, 0, len(request.Provenance.CollectorExecutions))
	for _, collector := range request.Provenance.CollectorExecutions {
		collectors = append(collectors, eventbiz.CollectorExecution{ArtifactID: collector.ArtifactID, CollectorExecutionID: collector.CollectorExecutionID})
	}
	documents := make([]eventbiz.EventEvidenceRecord, 0, len(request.RawDocuments))
	for _, document := range request.RawDocuments {
		documents = append(documents, eventbiz.EventEvidenceRecord{
			ArtifactID: document.ArtifactID, ContentSHA256: document.ContentSHA256,
			SourceRef: document.SourceRef, SourceName: document.SourceName, SourceType: document.SourceType,
			SourceURL: document.SourceURL, Title: document.Title, PublishedAt: document.PublishedAt,
			CollectedAt: document.CollectedAt, Language: document.Language, MIMEType: document.MIMEType,
		})
	}
	events := make([]eventbiz.PublicationEvent, 0, len(request.Events))
	for _, input := range request.Events {
		evidence := make([]eventbiz.EventEvidenceLinkInput, 0, len(input.Evidence))
		for _, item := range input.Evidence {
			evidence = append(evidence, eventbiz.EventEvidenceLinkInput{
				ArtifactID: item.ArtifactID, EvidenceRelation: item.EvidenceRelation,
				EvidenceStatement: item.EvidenceStatement, SupportsFields: append([]string(nil), item.SupportsFields...),
				SourceLevel: item.SourceLevel,
			})
		}
		tags := make([]eventbiz.EventTagInput, 0, len(input.Tags))
		for _, tag := range input.Tags {
			tags = append(tags, eventbiz.EventTagInput{
				TagID: tag.TagID, TagKind: tag.TagKind, TagCode: tag.TagCode, Confidence: tag.Confidence,
				AssignmentReason: tag.AssignmentReason, AssignSource: tag.AssignSource,
			})
		}
		events = append(events, eventbiz.PublicationEvent{
			DedupeKey: input.DedupeKey, Title: input.Title, FactualSummary: input.FactualSummary,
			OccurredAt: input.OccurredAt, FactPayload: input.FactPayload, Evidence: evidence, Tags: tags,
			Review: eventbiz.Review{ReviewID: input.Review.ReviewID, EvidenceGrade: input.Review.EvidenceGrade, Reasons: append([]string(nil), input.Review.Reasons...)},
		})
	}
	return eventbiz.PublicationBatch{
		PackageID: request.PackageID,
		Provenance: eventbiz.Provenance{
			ExtractorExecutionID: request.Provenance.ExtractorExecutionID, ExtractorAgentVersion: request.Provenance.ExtractorAgentVersion,
			CollectorExecutions: collectors,
		},
		RawDocuments: documents, Events: events,
	}
}

func publicationResult(result eventbiz.Result) eventapi.PublicationResult {
	events := make([]eventapi.PublicationEventResult, 0, len(result.Events))
	for _, item := range result.Events {
		events = append(events, eventapi.PublicationEventResult{DedupeKey: item.DedupeKey, EventID: item.EventID, Disposition: item.Disposition})
	}
	documents := make([]eventapi.PublicationRawDocumentResult, 0, len(result.RawDocuments))
	for _, item := range result.RawDocuments {
		documents = append(documents, eventapi.PublicationRawDocumentResult{ArtifactID: item.ArtifactID, RawDocumentID: item.RawDocumentID, Disposition: item.Disposition})
	}
	return eventapi.PublicationResult{
		ReceiptID: result.ReceiptID, PackageID: result.PackageID, ImportedAt: result.ImportedAt,
		Events: events, RawDocuments: documents,
		Counts: eventapi.PublicationCounts{
			EventsCreated: result.Counts.EventsCreated, EventsReused: result.Counts.EventsReused,
			RawDocumentsCreated: result.Counts.RawDocumentsCreated, RawDocumentsReused: result.Counts.RawDocumentsReused,
			EventSourcesCreated: result.Counts.EventSourcesCreated, EventSourcesReused: result.Counts.EventSourcesReused,
			EventTagsCreated: result.Counts.EventTagsCreated, EventTagsReused: result.Counts.EventTagsReused,
		},
	}
}

func principalIdentity(ctx context.Context) string {
	if principal, ok := v1.PrincipalFromContext(ctx); ok {
		return principal.Identity
	}
	return ""
}

func optionalUTC(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil || value.Location() != time.UTC {
		return nil, errors.New("timestamp must use UTC")
	}
	return &value, nil
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}

func publicErrorWithDetails(status int, code, message string, details any) error {
	return v1.NewPublicError(status, code, message, details)
}

var _ eventapi.Service = (*Service)(nil)
