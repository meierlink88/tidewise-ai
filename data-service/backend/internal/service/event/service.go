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
		items = append(items, eventapi.Item{
			ID: item.ID, Title: item.Title, Summary: item.Summary,
			Semantic: eventapi.Semantic{
				Who: item.Semantic.Who, What: item.Semantic.What, When: item.Semantic.When,
				Where: item.Semantic.Where, Why: item.Semantic.Why, How: item.Semantic.How,
			},
			Modality: string(item.Modality), OccurredAt: formatOptionalTime(item.OccurredAt),
			AnnouncedAt: formatOptionalTime(item.AnnouncedAt), Status: string(item.Status),
		})
	}
	return &v1.Response[eventapi.Page]{Status: v1.StatusOK, Result: eventapi.Page{
		Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize,
	}}, nil
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
