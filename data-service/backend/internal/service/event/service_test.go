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
	observed := time.Date(2026, 8, 19, 2, 0, 0, 0, time.UTC)
	who := "Example Corp"
	useCase := &eventUseCaseStub{page: eventbiz.EventPage{
		Items: []eventbiz.Event{{
			ID: "EVT11111111-1111-4111-8111-111111111111", Title: "Event", Summary: "Summary",
			Semantic: eventbiz.Semantic{Actors: []string{who}, Action: "acts", Objects: []string{"object"},
				Stage: eventbiz.EventStageOccurred, Modality: eventbiz.ModalityFact, Jurisdictions: []string{},
				Time: eventbiz.EventTime{OccurredAt: &occurred, Precision: eventbiz.TimePrecisionDay}, Metrics: []eventbiz.Metric{}},
			Status: eventbiz.LifecycleStatusActive,
		}, {
			ID: "EVT22222222-2222-4222-8222-222222222222", Title: "Observed Event", Summary: "Observed summary",
			Semantic: eventbiz.Semantic{Actors: []string{who}, Action: "warns", Objects: []string{"object"},
				Stage: eventbiz.EventStageExpected, Modality: eventbiz.ModalitySpec, Jurisdictions: []string{},
				Time: eventbiz.EventTime{ObservedAt: &observed, Precision: eventbiz.TimePrecisionInstant}, Metrics: []eventbiz.Metric{}},
			Status: eventbiz.LifecycleStatusActive,
		}},
		Total: 2, Page: 2, PageSize: 10,
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
	if response.Status != v1.StatusOK || len(response.Result.Items) != 2 {
		t.Fatalf("response = %#v", response)
	}
	item := response.Result.Items[0]
	if item.Semantic.Modality != "FACT" || item.Status != "ACTIVE" || item.Semantic.Time.OccurredAt == nil || len(item.Semantic.Actors) != 1 || item.Semantic.Actors[0] != who {
		t.Fatalf("item = %#v", item)
	}
	observedItem := response.Result.Items[1]
	if observedItem.Semantic.Time.ObservedAt == nil || *observedItem.Semantic.Time.ObservedAt != observed.Format(time.RFC3339) ||
		observedItem.Semantic.Time.OccurredAt != nil || observedItem.Semantic.Time.AnnouncedAt != nil || observedItem.Semantic.Time.EffectiveAt != nil {
		t.Fatalf("observed item = %#v", observedItem)
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
	request    eventbiz.EventListRequest
	page       eventbiz.EventPage
	err        error
	publishErr error
}

func (s *eventUseCaseStub) ListEvents(_ context.Context, request eventbiz.EventListRequest) (eventbiz.EventPage, error) {
	s.request = request
	return s.page, s.err
}

func (s *eventUseCaseStub) Publish(context.Context, string, string, eventbiz.CreateInput) (eventbiz.PublicationResult, error) {
	return eventbiz.PublicationResult{}, s.publishErr
}

func TestPublishEventMapsUnknownEvidenceToStableUnprocessableEntity(t *testing.T) {
	service, err := NewService(&eventUseCaseStub{publishErr: &eventbiz.ReferenceError{
		Field: "evidence_ids", Message: "contains an unavailable Evidence",
	}})
	if err != nil {
		t.Fatal(err)
	}
	effective := "2026-08-25T00:00:00Z"
	_, err = service.PublishEvent(context.Background(), &eventapi.PublicationRequest{
		PublicationKey: "submission-1:create",
		Event: eventapi.PublicationEvent{
			Title: "Event", Summary: "Summary",
			Semantic: eventapi.Semantic{
				Actors: []string{"actor"}, Action: "acts", Objects: []string{"object"},
				Stage: "OCCURRED", Modality: "FACT", Jurisdictions: []string{},
				Time: eventapi.Time{EffectiveAt: &effective, Precision: "DAY"}, Metrics: []eventapi.Metric{},
			},
		},
		EvidenceIDs: []string{"EVD11111111-1111-4111-8111-111111111111"},
	})
	var public *v1.PublicError
	if !errors.As(err, &public) || public.Status != v1.StatusUnprocessableEntity ||
		public.Code != eventapi.ErrorEventEvidenceReferenceInvalid {
		t.Fatalf("error = %#v", err)
	}
}
