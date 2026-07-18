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
		sources: []domain.SourceCatalog{{ID: "source-1", Status: domain.SourceCatalogStatusActive}},
	}
	service := NewService(repository)

	rawPage, err := service.ListRawDocuments(context.Background(), RawDocumentListRequest{
		Title: "raw", SourceID: "22222222-2222-5222-8222-222222222222", IngestStatus: domain.IngestStatusCollected, Page: 2, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.rawFilter.Title != "raw" || repository.rawFilter.SourceID != "22222222-2222-5222-8222-222222222222" || rawPage.Total != 1 || rawPage.Items[0].ID != "raw-1" {
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

	sources, err := service.ListSourceCatalogs(context.Background(), SourceCatalogListRequest{Status: domain.SourceCatalogStatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if repository.sourceFilter.Status != domain.SourceCatalogStatusActive || len(sources) != 1 || sources[0].ID != "source-1" {
		t.Fatalf("source query was not mapped: filter=%#v sources=%#v", repository.sourceFilter, sources)
	}
}

type fakeRepository struct {
	rawPage      repositories.RawDocumentPage
	eventPage    repositories.EventPage
	sources      []domain.SourceCatalog
	rawFilter    repositories.RawDocumentListFilter
	eventFilter  repositories.EventListFilter
	sourceFilter repositories.SourceCatalogListFilter
}

func (f *fakeRepository) ListRawDocuments(_ context.Context, filter repositories.RawDocumentListFilter) (repositories.RawDocumentPage, error) {
	f.rawFilter = filter
	return f.rawPage, nil
}

func (f *fakeRepository) ListEvents(_ context.Context, filter repositories.EventListFilter) (repositories.EventPage, error) {
	f.eventFilter = filter
	return f.eventPage, nil
}

func (f *fakeRepository) ListSourceCatalogs(_ context.Context, filter repositories.SourceCatalogListFilter) ([]domain.SourceCatalog, error) {
	f.sourceFilter = filter
	return f.sources, nil
}
