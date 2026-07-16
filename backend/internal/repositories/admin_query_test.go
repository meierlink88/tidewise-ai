package repositories

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestInMemoryRepositoryListsRawDocumentsWithPaginationAndTitleSearch(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	base := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
	for _, doc := range []domain.RawDocument{
		validRawDocumentWithTitle("raw-1", "央行公布金融数据", base.Add(2*time.Minute)),
		validRawDocumentWithTitle("raw-2", "美联储维持利率不变", base.Add(time.Minute)),
		validRawDocumentWithTitle("raw-3", "央行开展逆回购操作", base),
	} {
		if _, err := repo.UpsertRawDocument(context.Background(), doc); err != nil {
			t.Fatalf("UpsertRawDocument(%s) error = %v", doc.ID, err)
		}
	}

	page, err := repo.ListRawDocuments(context.Background(), RawDocumentListFilter{Title: "央行", Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("ListRawDocuments() error = %v", err)
	}

	if page.Total != 2 || page.Page != 1 || page.PageSize != 1 {
		t.Fatalf("page = %+v, want total/page/page_size 2/1/1", page)
	}
	if got := rawDocumentIDs(page.Items); !reflect.DeepEqual(got, []string{"raw-1"}) {
		t.Fatalf("page ids = %v, want newest matching raw-1", got)
	}
}

func TestInMemoryRepositoryListsEventsWithFilters(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	eventTime := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID:          "event-1",
		Title:       "美联储维持利率不变",
		Summary:     "FOMC 会议维持联邦基金利率目标区间不变。",
		EventTime:   &eventTime,
		FirstSeenAt: eventTime.Add(10 * time.Minute),
		EventStatus: domain.EventStatusConfirmed,
		FactStatus:  domain.FactStatusVerified,
		DedupeKey:   "fed-rate-hold",
		FactPayload: domain.FactPayload{},
	}); err != nil {
		t.Fatalf("SeedEvent(event-1) error = %v", err)
	}
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID:          "event-2",
		Title:       "欧洲央行政策表态",
		Summary:     "欧洲央行官员表态。",
		EventTime:   ptrTimeForRepositoryTest(eventTime.Add(-24 * time.Hour)),
		FirstSeenAt: eventTime.Add(-23 * time.Hour),
		EventStatus: domain.EventStatusCandidate,
		FactStatus:  domain.FactStatusUnverified,
		DedupeKey:   "ecb-policy",
		FactPayload: domain.FactPayload{},
	}); err != nil {
		t.Fatalf("SeedEvent(event-2) error = %v", err)
	}

	page, err := repo.ListEvents(context.Background(), EventListFilter{
		Title:         "美联储",
		EventStatus:   domain.EventStatusConfirmed,
		FactStatus:    domain.FactStatusVerified,
		EventTimeFrom: ptrTimeForRepositoryTest(eventTime.Add(-time.Hour)),
		EventTimeTo:   ptrTimeForRepositoryTest(eventTime.Add(time.Hour)),
		FirstSeenFrom: ptrTimeForRepositoryTest(eventTime),
		FirstSeenTo:   ptrTimeForRepositoryTest(eventTime.Add(time.Hour)),
		Page:          1,
		PageSize:      50,
	})
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}

	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != "event-1" {
		t.Fatalf("event page = %+v, want only event-1", page)
	}
}

func TestInMemoryRepositoryListsSourceCatalogsByStatus(t *testing.T) {
	repo := NewInMemoryRepository([]domain.SourceCatalog{
		{ID: "source-1", ProviderKey: "rss", IngestChannel: "rss_feed", SourceType: "news", SourceName: "RSS 1", Status: domain.SourceCatalogStatusActive},
		{ID: "source-2", ProviderKey: "rss", IngestChannel: "rss_feed", SourceType: "news", SourceName: "RSS 2", Status: domain.SourceCatalogStatusInactive},
		{ID: "source-3", ProviderKey: "eastmoney", IngestChannel: "http_api", SourceType: "market", SourceName: "东方财富", Status: domain.SourceCatalogStatusDisabled},
	})

	sources, err := repo.ListSourceCatalogs(context.Background(), SourceCatalogListFilter{Status: domain.SourceCatalogStatusInactive})
	if err != nil {
		t.Fatalf("ListSourceCatalogs() error = %v", err)
	}

	if got := sourceIDs(sources); !reflect.DeepEqual(got, []string{"source-2"}) {
		t.Fatalf("source ids = %v, want inactive source-2", got)
	}
}

func validRawDocumentWithTitle(id string, title string, collectedAt time.Time) domain.RawDocument {
	doc := validRawDocument(id)
	doc.Title = title
	doc.ContentHash = id + "-hash"
	doc.CollectedAt = collectedAt
	return doc
}

func rawDocumentIDs(items []domain.RawDocument) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func ptrTimeForRepositoryTest(value time.Time) *time.Time {
	return &value
}
