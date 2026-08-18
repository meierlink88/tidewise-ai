package event

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	eventapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/event"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
)

func TestListEventsMapsNewEventContract(t *testing.T) {
	occurred := time.Date(2026, 8, 18, 1, 0, 0, 0, time.UTC)
	who := "Example Corp"
	useCase := &eventUseCaseStub{page: eventbiz.EventPage{
		Items: []eventbiz.Event{{
			ID: "EVT11111111-1111-4111-8111-111111111111", Title: "Event", Summary: "Summary",
			Semantic: eventbiz.Semantic{Who: &who}, Modality: eventbiz.ModalityFact,
			OccurredAt: &occurred, Status: eventbiz.LifecycleStatusActive,
		}},
		Total: 1, Page: 2, PageSize: 10,
	}}
	service, err := NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.ListEvents(context.Background(), &eventapi.ListRequest{
		Title: " Event ", Modality: "FACT", Status: "ACTIVE",
		OccurredFrom: "2026-08-18T00:00:00Z", Page: "2", PageSize: "10",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != v1.StatusOK || len(response.Result.Items) != 1 {
		t.Fatalf("response = %#v", response)
	}
	item := response.Result.Items[0]
	if item.Modality != "FACT" || item.Status != "ACTIVE" || item.OccurredAt == nil || item.Semantic.Who == nil || *item.Semantic.Who != who {
		t.Fatalf("item = %#v", item)
	}
	if useCase.request.Title != "Event" || useCase.request.OccurredFrom == nil || useCase.request.Page != 2 {
		t.Fatalf("Biz request = %#v", useCase.request)
	}
}

func TestListEventsRejectsRetiredAndInvalidFilters(t *testing.T) {
	service, err := NewService(new(eventUseCaseStub))
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []*eventapi.ListRequest{
		{Modality: "UNKNOWN"},
		{Status: "confirmed"},
		{AnnouncedFrom: "2026-08-18T08:00:00+08:00"},
	} {
		_, err := service.ListEvents(context.Background(), request)
		var public *v1.PublicError
		if !errors.As(err, &public) || public.Status != v1.StatusBadRequest || public.Code != eventapi.ErrorInvalidRequest {
			t.Fatalf("error = %#v", err)
		}
	}
}

type eventUseCaseStub struct {
	request eventbiz.EventListRequest
	page    eventbiz.EventPage
	err     error
}

func (s *eventUseCaseStub) ListEvents(_ context.Context, request eventbiz.EventListRequest) (eventbiz.EventPage, error) {
	s.request = request
	return s.page, s.err
}
