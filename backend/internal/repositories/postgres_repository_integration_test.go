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

func TestPostgresRepositorySchedulerIntegration(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run PostgreSQL scheduler repository integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewPostgresRepository(db)
	runID := fmt.Sprintf("scheduler-integration-%d", time.Now().UnixNano())
	source := domain.SourceCatalog{
		ID:            runID + "-source",
		IngestChannel: "ai_web_research",
		ProviderKey:   "llm_web_research",
		ConnectorKey:  "llm_web_research",
		ParserKey:     "llm_research_items",
		SourceType:    "news",
		SourceName:    "调度集成测试来源",
		SourceURL:     "https://example.com/scheduler",
		SourceLevel:   "secondary",
		AuthType:      "none",
		UsagePolicy:   "integration-test",
		Status:        domain.SourceCatalogStatusActive,
	}
	if err := repo.SeedSource(ctx, source); err != nil {
		t.Fatalf("SeedSource() error = %v", err)
	}

	config, err := repo.SaveSchedulerConfig(ctx, domain.SchedulerConfig{
		ID:              "default",
		Enabled:         true,
		Mode:            domain.SchedulerModeInterval,
		IntervalMinutes: 60,
		Concurrency:     2,
		BatchSize:       10,
		TimeoutSeconds:  180,
		Timezone:        "Asia/Shanghai",
		SourceFilter: domain.SchedulerSourceFilter{
			ProviderKey:   "llm_web_research",
			IngestChannel: "ai_web_research",
			SourceType:    "news",
		},
	})
	if err != nil {
		t.Fatalf("SaveSchedulerConfig() error = %v", err)
	}
	if config.SourceFilter.ProviderKey != "llm_web_research" {
		t.Fatalf("saved SourceFilter.ProviderKey = %q", config.SourceFilter.ProviderKey)
	}

	started := time.Now()
	finished := started.Add(time.Second)
	run, err := repo.CreateIngestionRun(ctx, domain.IngestionRun{
		ID:              runID + "-run",
		TriggerType:     domain.SchedulerTriggerManualOnce,
		Status:          domain.SchedulerRunStatusRunning,
		StartedAt:       started,
		SchedulerConfig: map[string]any{"mode": string(domain.SchedulerModeInterval)},
	})
	if err != nil {
		t.Fatalf("CreateIngestionRun() error = %v", err)
	}
	if err := repo.RecordIngestionRunSource(ctx, domain.IngestionRunSource{
		ID:                 runID + "-run-source",
		RunID:              run.ID,
		SourceID:           source.ID,
		Status:             domain.SchedulerSourceRunStatusSucceeded,
		DocumentsWritten:   1,
		DocumentsDuplicate: 0,
		StartedAt:          started,
		FinishedAt:         &finished,
		DurationMillis:     1000,
	}); err != nil {
		t.Fatalf("RecordIngestionRunSource() error = %v", err)
	}
	run.Status = domain.SchedulerRunStatusSucceeded
	run.FinishedAt = &finished
	run.TotalSources = 1
	run.SucceededSources = 1
	if err := repo.CompleteIngestionRun(ctx, run); err != nil {
		t.Fatalf("CompleteIngestionRun() error = %v", err)
	}

	runs, err := repo.RecentIngestionRuns(ctx, 5)
	if err != nil {
		t.Fatalf("RecentIngestionRuns() error = %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("RecentIngestionRuns() returned no rows")
	}
}

func TestPostgresRepositoryGraphProjectionIntegration(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run PostgreSQL graph projection repository integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewPostgresRepository(db)
	runID := fmt.Sprintf("graph-integration-%d", time.Now().UnixNano())
	firstEntityID := NormalizeUUID(runID, "entity-a")
	secondEntityID := NormalizeUUID(runID, "entity-b")
	edgeID := NormalizeUUID(runID, "edge")

	if _, err := db.ExecContext(ctx, `
INSERT INTO entity_nodes (
    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
    ($1, $2, 'economy', 'economy', '中国', '中国', '{}'::text[], 'active'),
    ($3, $4, 'alliance_org', 'alliance', 'G20', '二十国集团', '{}'::text[], 'active')
ON CONFLICT (id) DO UPDATE SET
    entity_key = EXCLUDED.entity_key,
    updated_at = now()
`, firstEntityID, runID+"-economy:cn", secondEntityID, runID+"-alliance:g20"); err != nil {
		t.Fatalf("insert graph entities: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO entity_edges (
    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
) VALUES (
    $1, $2, $3, 'member_of', 'integration test', 'active'
) ON CONFLICT (id) DO UPDATE SET
    relation_type = EXCLUDED.relation_type,
    evidence_note = EXCLUDED.evidence_note,
    updated_at = now()
`, edgeID, firstEntityID, secondEntityID); err != nil {
		t.Fatalf("insert graph edge: %v", err)
	}

	nodes, err := repo.ListGraphEntityNodes(ctx)
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("ListGraphEntityNodes() returned no rows")
	}
	edges, err := repo.ListGraphEntityEdges(ctx)
	if err != nil {
		t.Fatalf("ListGraphEntityEdges() error = %v", err)
	}
	if len(edges) == 0 {
		t.Fatal("ListGraphEntityEdges() returned no rows")
	}

	started := time.Now()
	finished := started.Add(time.Second)
	run, err := repo.CreateGraphProjectionRun(ctx, GraphProjectionRun{
		ID:             runID + "-run",
		ProjectionType: GraphProjectionTypeEntityGraph,
		Mode:           GraphProjectionModeProjectEntities,
		Status:         GraphProjectionRunStatusRunning,
		StartedAt:      started,
		ConfigSummary:  map[string]any{"namespace": "tidewise"},
	})
	if err != nil {
		t.Fatalf("CreateGraphProjectionRun() error = %v", err)
	}
	if err := repo.RecordGraphProjectionRunItem(ctx, GraphProjectionRunItem{
		ID:       runID + "-item",
		RunID:    run.ID,
		ItemType: GraphProjectionRunItemTypeEntity,
		ItemKey:  firstEntityID,
		Status:   GraphProjectionRunItemStatusProjected,
	}); err != nil {
		t.Fatalf("RecordGraphProjectionRunItem() error = %v", err)
	}
	run.Status = GraphProjectionRunStatusSucceeded
	run.FinishedAt = &finished
	run.SourceRowCount = 2
	run.ProjectedCount = 2
	if err := repo.CompleteGraphProjectionRun(ctx, run); err != nil {
		t.Fatalf("CompleteGraphProjectionRun() error = %v", err)
	}
	runs, err := repo.RecentGraphProjectionRuns(ctx, 5)
	if err != nil {
		t.Fatalf("RecentGraphProjectionRuns() error = %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("RecentGraphProjectionRuns() returned no rows")
	}
}
