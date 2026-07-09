package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestPostgresRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run PostgreSQL repository integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewPostgresRepository(db)
	runID := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	source := domain.SourceCatalog{
		ID:            runID + "-source",
		IngestChannel: "rss_feed",
		ProviderKey:   "rss",
		ConnectorKey:  "rss_feed",
		ParserKey:     "rss_item",
		SourceType:    "news",
		SourceName:    "集成测试来源",
		SourceURL:     "https://example.com/feed.xml",
		SourceLevel:   "secondary",
		AuthType:      "none",
		SourceConfig: map[string]any{
			"kind": "rss_feed",
		},
		UsagePolicy: "integration-test",
		Status:      domain.SourceCatalogStatusActive,
	}

	if err := repo.SeedSource(ctx, source); err != nil {
		t.Fatalf("SeedSource() error = %v", err)
	}

	sources, err := repo.ActiveSources(ctx, SourceCatalogFilter{ProviderKey: "rss", IngestChannel: "rss_feed"})
	if err != nil {
		t.Fatalf("ActiveSources() error = %v", err)
	}
	if len(sources) == 0 {
		t.Fatal("ActiveSources() returned no rows")
	}
	if got := sources[0].SourceConfig["kind"]; got != "rss_feed" {
		t.Fatalf("SourceConfig[kind] = %v, want rss_feed", got)
	}

	doc := domain.RawDocument{
		ID:               runID + "-doc-a",
		SourceID:         source.ID,
		IngestChannel:    "rss_feed",
		SourceType:       "news",
		SourceName:       "集成测试来源",
		SourceURL:        "https://example.com/item-a",
		SourceExternalID: runID + "-item-a",
		Title:            "集成测试标题",
		ContentText:      "集成测试正文",
		ContentHash:      runID + "-hash-a",
		CollectedAt:      time.Now(),
		IngestStatus:     domain.IngestStatusCollected,
	}

	first, err := repo.UpsertRawDocument(ctx, doc)
	if err != nil {
		t.Fatalf("UpsertRawDocument() error = %v", err)
	}
	if !first.Created {
		t.Fatal("first UpsertRawDocument() should create a row")
	}

	duplicate := doc
	duplicate.ID = runID + "-doc-b"
	duplicate.ContentHash = runID + "-hash-b"
	second, err := repo.UpsertRawDocument(ctx, duplicate)
	if err != nil {
		t.Fatalf("UpsertRawDocument() duplicate error = %v", err)
	}
	if second.Created {
		t.Fatal("duplicate external id should not create a row")
	}
	if second.DuplicateOf == "" {
		t.Fatal("duplicate result should include DuplicateOf")
	}
}
