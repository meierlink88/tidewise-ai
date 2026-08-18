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
