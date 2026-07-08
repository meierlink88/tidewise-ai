package repositories

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestInMemoryRepositoryListsActiveSources(t *testing.T) {
	repo := NewInMemoryRepository([]domain.SourceCatalog{
		{
			ID:            "source-1",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			Status:        domain.SourceCatalogStatusActive,
		},
		{
			ID:            "source-2",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			Status:        domain.SourceCatalogStatusDisabled,
		},
		{
			ID:            "source-3",
			IngestChannel: "web_fetch",
			ProviderKey:   "web",
			Status:        domain.SourceCatalogStatusActive,
		},
	})

	sources, err := repo.ActiveSources(context.Background(), SourceCatalogFilter{
		ProviderKey:   "rss",
		IngestChannel: "rss_feed",
	})
	if err != nil {
		t.Fatalf("ActiveSources() error = %v", err)
	}

	if got, want := sourceIDs(sources), []string{"source-1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("source ids = %v, want %v", got, want)
	}
}

func TestInMemoryRepositoryUpsertsRawDocumentByExternalID(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	doc := validRawDocument("raw-1")
	doc.SourceExternalID = "external-1"

	first, err := repo.UpsertRawDocument(context.Background(), doc)
	if err != nil {
		t.Fatalf("UpsertRawDocument() error = %v", err)
	}
	if !first.Created {
		t.Fatal("first write should create document")
	}

	duplicate := validRawDocument("raw-2")
	duplicate.SourceExternalID = "external-1"
	duplicate.ContentHash = "hash-2"

	second, err := repo.UpsertRawDocument(context.Background(), duplicate)
	if err != nil {
		t.Fatalf("UpsertRawDocument() duplicate error = %v", err)
	}
	if second.Created {
		t.Fatal("duplicate external id should not create document")
	}
	if second.Document.ID != "raw-1" {
		t.Fatalf("duplicate document id = %q, want raw-1", second.Document.ID)
	}
}

func TestInMemoryRepositoryUpsertsRawDocumentByContentHash(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	doc := validRawDocument("raw-1")
	doc.SourceExternalID = ""

	if _, err := repo.UpsertRawDocument(context.Background(), doc); err != nil {
		t.Fatalf("UpsertRawDocument() error = %v", err)
	}

	duplicate := validRawDocument("raw-2")
	duplicate.SourceExternalID = ""

	result, err := repo.UpsertRawDocument(context.Background(), duplicate)
	if err != nil {
		t.Fatalf("UpsertRawDocument() duplicate error = %v", err)
	}
	if result.Created {
		t.Fatal("duplicate content hash should not create document")
	}
	if result.DuplicateOf != "raw-1" {
		t.Fatalf("DuplicateOf = %q, want raw-1", result.DuplicateOf)
	}
}

func TestInMemoryRepositoryUpdatesRawDocumentStatus(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	doc := validRawDocument("raw-1")
	if _, err := repo.UpsertRawDocument(context.Background(), doc); err != nil {
		t.Fatalf("UpsertRawDocument() error = %v", err)
	}

	if err := repo.UpdateRawDocumentStatus(context.Background(), "raw-1", domain.IngestStatusFailed); err != nil {
		t.Fatalf("UpdateRawDocumentStatus() error = %v", err)
	}

	stored, ok := repo.RawDocument("raw-1")
	if !ok {
		t.Fatal("raw document not found")
	}
	if stored.IngestStatus != domain.IngestStatusFailed {
		t.Fatalf("IngestStatus = %q, want %q", stored.IngestStatus, domain.IngestStatusFailed)
	}
}

func validRawDocument(id string) domain.RawDocument {
	return domain.RawDocument{
		ID:            id,
		SourceID:      "source-1",
		IngestChannel: "rss_feed",
		SourceType:    "news",
		SourceName:    "示例来源",
		Title:         "示例标题",
		ContentHash:   "hash-1",
		CollectedAt:   time.Now(),
		IngestStatus:  domain.IngestStatusCollected,
	}
}

func sourceIDs(sources []domain.SourceCatalog) []string {
	ids := make([]string, 0, len(sources))
	for _, source := range sources {
		ids = append(ids, source.ID)
	}
	return ids
}
