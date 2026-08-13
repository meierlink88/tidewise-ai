package event

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	eventapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/event"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
)

type fakeUseCase struct {
	catalog eventbiz.EventTagCatalog
	err     error
}

func (f fakeUseCase) Import(context.Context, string, eventbiz.PublicationBatch) (eventbiz.Result, error) {
	return eventbiz.Result{}, f.err
}

func (f fakeUseCase) ActiveTags(context.Context) (eventbiz.EventTagCatalog, error) {
	return f.catalog, f.err
}

func (f fakeUseCase) ListEvents(context.Context, eventbiz.EventListRequest) (eventbiz.EventPage, error) {
	return eventbiz.EventPage{}, f.err
}

func TestCancellationPreservesLegacyPublicErrorMapping(t *testing.T) {
	service, err := NewService(fakeUseCase{err: context.Canceled})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, code string
		call       func() error
	}{
		{name: "publication", code: eventapi.ErrorEventPublicationFailed, call: func() error {
			_, callErr := service.PublishReviewedEvents(v1.WithPrincipal(context.Background(), v1.Principal{Identity: "caller"}), &eventapi.PublicationRequest{})
			return callErr
		}},
		{name: "catalog", code: eventapi.ErrorEventTagCatalogFailed, call: func() error {
			_, callErr := service.ListActiveEventTags(context.Background(), &eventapi.TagCatalogRequest{Active: true})
			return callErr
		}},
		{name: "list", code: eventapi.ErrorDataRepositoryFailure, call: func() error {
			_, callErr := service.ListEvents(context.Background(), &eventapi.ListRequest{})
			return callErr
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var public *v1.PublicError
			if err := test.call(); !errors.As(err, &public) || public.Code != test.code {
				t.Fatalf("error = %#v, want public code %s", err, test.code)
			}
		})
	}
}

func TestListActiveEventTagsMapsCatalogSnapshot(t *testing.T) {
	service, err := NewService(fakeUseCase{catalog: eventbiz.EventTagCatalog{
		Tags: []eventbiz.EventTag{{
			ID: "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "news_category",
			Code: "technology_industry", Name: "科技产业", Active: true,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.ListActiveEventTags(context.Background(), &eventapi.TagCatalogRequest{Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Tags) != 1 || !response.Result.Tags[0].IsActive {
		t.Fatalf("catalog = %#v", response.Result)
	}
}
