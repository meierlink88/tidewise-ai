package event

import (
	"context"
	"errors"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	eventapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/event"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
)

type UseCase interface {
	ListEvents(context.Context, eventbiz.EventListRequest) (eventbiz.EventPage, error)
	Publish(context.Context, string, string, eventbiz.CreateInput) (eventbiz.PublicationResult, error)
}

func (s *Service) PublishEvent(ctx context.Context, request *eventapi.PublicationRequest) (*v1.Response[eventapi.PublicationResult], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "Event publication is required")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, eventapi.ErrorDataServiceNotReady, "Event service is unavailable")
	}
	effectiveAt, err := optionalWireUTC(request.Event.Semantic.Time.EffectiveAt)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "semantic.time.effective_at must be UTC RFC3339")
	}
	occurredAt, err := optionalWireUTC(request.Event.Semantic.Time.OccurredAt)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "semantic.time.occurred_at must be UTC RFC3339")
	}
	announcedAt, err := optionalWireUTC(request.Event.Semantic.Time.AnnouncedAt)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "semantic.time.announced_at must be UTC RFC3339")
	}
	evidence := make([]eventbiz.EvidenceLinkInput, 0, len(request.EvidenceIDs))
	for _, evidenceID := range request.EvidenceIDs {
		evidence = append(evidence, eventbiz.EvidenceLinkInput{EvidenceID: evidenceID, ContributionWeight: 1})
	}
	input := eventbiz.CreateInput{Title: request.Event.Title, Summary: request.Event.Summary,
		Semantic: eventbiz.Semantic{Actors: request.Event.Semantic.Actors, Action: request.Event.Semantic.Action,
			Objects: request.Event.Semantic.Objects, Stage: eventbiz.EventStage(request.Event.Semantic.Stage),
			Modality: eventbiz.Modality(request.Event.Semantic.Modality),
			Time: eventbiz.EventTime{OccurredAt: occurredAt, AnnouncedAt: announcedAt, EffectiveAt: effectiveAt,
				Precision: eventbiz.TimePrecision(request.Event.Semantic.Time.Precision)},
			Jurisdictions: request.Event.Semantic.Jurisdictions, Reason: request.Event.Semantic.Reason,
			Method: request.Event.Semantic.Method, Metrics: eventMetrics(request.Event.Semantic.Metrics)},
		Status: eventbiz.LifecycleStatusActive, Evidence: evidence}
	principal, _ := v1.PrincipalFromContext(ctx)
	result, err := s.useCase.Publish(ctx, principal.Identity, request.PublicationKey, input)
	if err != nil {
		if errors.Is(err, eventbiz.ErrPublicationPayloadConflict) {
			return nil, publicError(v1.StatusConflict, eventapi.ErrorEventPublishConflict, "publication_key conflicts with another Event payload")
		}
		var validation *eventbiz.ValidationError
		if errors.As(err, &validation) {
			return nil, publicError(v1.StatusUnprocessableEntity, eventapi.ErrorInvalidRequest, validation.Error())
		}
		var reference *eventbiz.ReferenceError
		if errors.As(err, &reference) {
			return nil, publicError(v1.StatusUnprocessableEntity, eventapi.ErrorEventEvidenceReferenceInvalid,
				"Event publication references unavailable Evidence")
		}
		return nil, publicError(v1.StatusInternalServerError, eventapi.ErrorDataRepositoryFailure, "Event publication failed")
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	evidenceLinkIDs := make([]string, 0, len(result.Evidence))
	for _, link := range result.Evidence {
		evidenceLinkIDs = append(evidenceLinkIDs, link.ID)
	}
	return &v1.Response[eventapi.PublicationResult]{Status: status, Result: eventapi.PublicationResult{
		Event: eventItem(result.Event), EvidenceLinkIDs: evidenceLinkIDs,
		ReceiptID: result.ReceiptID, PayloadHash: result.PayloadHash, Replayed: result.Replayed,
	}}, nil
}

func optionalWireUTC(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	return optionalUTC(*raw)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Event use case is required")
	}
	return &Service{useCase: useCase}, nil
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
		Title: strings.TrimSpace(request.Title), Modality: eventbiz.Modality(request.Modality),
		Status: eventbiz.LifecycleStatus(request.Status), Page: page, PageSize: pageSize,
	}
	if filter.Modality != "" && !oneOf(string(filter.Modality), string(eventbiz.ModalityFact), string(eventbiz.ModalityPlan), string(eventbiz.ModalitySpec)) {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "unsupported modality")
	}
	if filter.Status != "" && !oneOf(string(filter.Status), string(eventbiz.LifecycleStatusActive), string(eventbiz.LifecycleStatusDeprecated), string(eventbiz.LifecycleStatusArchived)) {
		return nil, publicError(v1.StatusBadRequest, eventapi.ErrorInvalidRequest, "unsupported status")
	}
	for _, value := range []struct {
		name   string
		raw    string
		target **time.Time
	}{
		{name: "occurred_from", raw: request.OccurredFrom, target: &filter.OccurredFrom},
		{name: "occurred_to", raw: request.OccurredTo, target: &filter.OccurredTo},
		{name: "announced_from", raw: request.AnnouncedFrom, target: &filter.AnnouncedFrom},
		{name: "announced_to", raw: request.AnnouncedTo, target: &filter.AnnouncedTo},
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
		return nil, publicError(v1.StatusInternalServerError, eventapi.ErrorDataRepositoryFailure, "admin Event aggregate failed")
	}
	items := make([]eventapi.Item, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, eventItem(item))
	}
	return &v1.Response[eventapi.Page]{Status: v1.StatusOK, Result: eventapi.Page{
		Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize,
	}}, nil
}

func eventItem(item eventbiz.Event) eventapi.Item {
	return eventapi.Item{ID: item.ID, Title: item.Title, Summary: item.Summary,
		Semantic: eventapi.Semantic{Actors: item.Semantic.Actors, Action: item.Semantic.Action,
			Objects: item.Semantic.Objects, Stage: string(item.Semantic.Stage),
			Modality: string(item.Semantic.Modality), Jurisdictions: item.Semantic.Jurisdictions,
			Time: eventapi.Time{OccurredAt: formatOptionalTime(item.Semantic.Time.OccurredAt),
				AnnouncedAt: formatOptionalTime(item.Semantic.Time.AnnouncedAt),
				EffectiveAt: formatOptionalTime(item.Semantic.Time.EffectiveAt),
				Precision:   string(item.Semantic.Time.Precision)},
			Reason: item.Semantic.Reason, Method: item.Semantic.Method, Metrics: apiEventMetrics(item.Semantic.Metrics)},
		Status: string(item.Status)}
}

func eventMetrics(values []eventapi.Metric) []eventbiz.Metric {
	result := make([]eventbiz.Metric, len(values))
	for index, value := range values {
		result[index] = eventbiz.Metric{Name: value.Name, Value: value.Value, Unit: value.Unit,
			Change: value.Change, Period: value.Period}
	}
	return result
}

func apiEventMetrics(values []eventbiz.Metric) []eventapi.Metric {
	result := make([]eventapi.Metric, len(values))
	for index, value := range values {
		result[index] = eventapi.Metric{Name: value.Name, Value: value.Value, Unit: value.Unit,
			Change: value.Change, Period: value.Period}
	}
	return result
}

func optionalUTC(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil || parsed.Location() != time.UTC {
		return nil, errors.New("timestamp must be UTC RFC3339")
	}
	parsed = parsed.UTC()
	return &parsed, nil
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

func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}
