package adminquery

import (
	"context"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

func TestServiceMapsTransportRequestsToRepositoryQueries(t *testing.T) {
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		rawPage: repositories.RawDocumentPage{
			Items: []domain.RawDocument{{ID: "raw-1", Title: "Raw", CollectedAt: now}},
			Total: 1, Page: 2, PageSize: 10,
		},
		eventPage: repositories.EventPage{
			Items: []domain.Event{{ID: "event-1", Title: "Event", FirstSeenAt: now}},
			Total: 1, Page: 3, PageSize: 20,
		},
	}
	service := NewService(repository)

	rawPage, err := service.ListRawDocuments(context.Background(), RawDocumentListRequest{
		Title: "raw", SourceRef: "source:reuters:world", IngestStatus: domain.IngestStatusCollected, Page: 2, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.rawFilter.Title != "raw" || repository.rawFilter.SourceRef != "source:reuters:world" || rawPage.Total != 1 || rawPage.Items[0].ID != "raw-1" {
		t.Fatalf("raw query was not mapped: filter=%#v page=%#v", repository.rawFilter, rawPage)
	}

	eventPage, err := service.ListEvents(context.Background(), EventListRequest{
		Title: "event", EventStatus: domain.EventStatusConfirmed, FactStatus: domain.FactStatusVerified, Page: 3, PageSize: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.eventFilter.EventStatus != domain.EventStatusConfirmed || repository.eventFilter.FactStatus != domain.FactStatusVerified || eventPage.Items[0].ID != "event-1" {
		t.Fatalf("event query was not mapped: filter=%#v page=%#v", repository.eventFilter, eventPage)
	}

}

type fakeRepository struct {
	rawPage     repositories.RawDocumentPage
	eventPage   repositories.EventPage
	rawFilter   repositories.RawDocumentListFilter
	eventFilter repositories.EventListFilter
}

func (f *fakeRepository) ListRawDocuments(_ context.Context, filter repositories.RawDocumentListFilter) (repositories.RawDocumentPage, error) {
	f.rawFilter = filter
	return f.rawPage, nil
}

func (f *fakeRepository) ListEvents(_ context.Context, filter repositories.EventListFilter) (repositories.EventPage, error) {
	f.eventFilter = filter
	return f.eventPage, nil
}
