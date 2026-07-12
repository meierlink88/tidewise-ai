package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
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
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})

	ctx := context.Background()
	repo := NewPostgresRepository(db)
	runID := fmt.Sprintf("graph-integration-%d", time.Now().UnixNano())
	firstEntityID := NormalizeUUID(runID, "entity-a")
	secondEntityID := NormalizeUUID(runID, "entity-b")
	edgeID := NormalizeUUID(runID, "edge")
	projectionRunID := NormalizeUUID(runID + "-run")
	t.Cleanup(func() {
		for _, cleanup := range []struct {
			query string
			args  []any
		}{
			{`DELETE FROM graph_projection_run_items WHERE run_id = $1`, []any{projectionRunID}},
			{`DELETE FROM graph_projection_runs WHERE id = $1`, []any{projectionRunID}},
			{`DELETE FROM entity_edges WHERE id = $1`, []any{edgeID}},
			{`DELETE FROM entity_nodes WHERE id IN ($1, $2)`, []any{firstEntityID, secondEntityID}},
		} {
			if _, err := db.ExecContext(ctx, cleanup.query, cleanup.args...); err != nil {
				t.Errorf("clean graph projection integration data: %v", err)
			}
		}
	})

	if _, err := db.ExecContext(ctx, `
INSERT INTO entity_nodes (
    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
    ($1, $2, 'economy', 'economy', '中国', '中国', ARRAY['China', 'PRC']::text[], 'active'),
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
	var projectedEntity *GraphEntityNode
	for i := range nodes {
		if nodes[i].ID == firstEntityID {
			projectedEntity = &nodes[i]
			break
		}
	}
	if projectedEntity == nil {
		t.Fatalf("ListGraphEntityNodes() missing inserted entity %s", firstEntityID)
	}
	if !reflect.DeepEqual(projectedEntity.Aliases, []string{"China", "PRC"}) {
		t.Fatalf("projected aliases = %v, want [China PRC]", projectedEntity.Aliases)
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

func TestPostgresGraphProjectionSourcesOnlyContainActiveRows(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run PostgreSQL graph projection source integration test")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := NewPostgresRepository(db)
	nodes, err := repo.ListGraphEntityNodes(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	activeIDs := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if node.Status != domain.StatusActive {
			t.Fatalf("projection source contains inactive node %s with status %s", node.ID, node.Status)
		}
		if node.EntityType == domain.EntityTypeSector {
			if node.ClassificationCode != domain.SectorClassificationIndustry && node.ClassificationCode != domain.SectorClassificationTheme && node.ClassificationCode != domain.SectorClassificationMarket && node.ClassificationCode != domain.SectorClassificationStyle && node.ClassificationCode != domain.SectorClassificationRegion {
				t.Fatalf("projection sector %s has invalid classification %q", node.ID, node.ClassificationCode)
			}
		} else if node.ClassificationCode != "" {
			t.Fatalf("non-sector projection node %s has classification %q", node.ID, node.ClassificationCode)
		}
		activeIDs[node.ID] = struct{}{}
	}
	edges, err := repo.ListGraphEntityEdges(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityEdges() error = %v", err)
	}
	for _, edge := range edges {
		if edge.Status != domain.StatusActive {
			t.Fatalf("projection source contains inactive edge %s", edge.ID)
		}
		if _, ok := activeIDs[edge.FromEntityID]; !ok {
			t.Fatalf("projection edge %s has inactive or absent from endpoint %s", edge.ID, edge.FromEntityID)
		}
		if _, ok := activeIDs[edge.ToEntityID]; !ok {
			t.Fatalf("projection edge %s has inactive or absent to endpoint %s", edge.ID, edge.ToEntityID)
		}
	}
}

func TestPostgresRepositoryBenchmarkObservationIntegration(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run PostgreSQL benchmark observation repository integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})

	ctx := context.Background()
	repo := NewPostgresRepository(db)
	runID := fmt.Sprintf("benchmark-observation-integration-%d", time.Now().UnixNano())
	benchmarkID := NormalizeUUID(runID, "benchmark")
	otherBenchmarkID := NormalizeUUID(runID, "other-benchmark")
	indexID := NormalizeUUID(runID, "index")
	entityIDs := []string{benchmarkID, otherBenchmarkID, indexID}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DELETE FROM benchmark_observations WHERE benchmark_entity_id = ANY($1::uuid[])`, entityIDs); err != nil {
			t.Errorf("cleanup benchmark observations: %v", err)
		}
		if _, err := db.ExecContext(context.Background(), `DELETE FROM entity_nodes WHERE id = ANY($1::uuid[])`, entityIDs); err != nil {
			t.Errorf("cleanup benchmark entities: %v", err)
		}
	})

	if _, err := db.ExecContext(ctx, `
INSERT INTO entity_nodes (
    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
    ($1, $2, 'benchmark', 'market', 'Benchmark A', 'Benchmark A', '{}'::text[], 'active'),
    ($3, $4, 'benchmark', 'market', 'Benchmark B', 'Benchmark B', '{}'::text[], 'active'),
    ($5, $6, 'index', 'market', 'Index A', 'Index A', '{}'::text[], 'active')
`, benchmarkID, runID+":benchmark-a", otherBenchmarkID, runID+":benchmark-b", indexID, runID+":index-a"); err != nil {
		t.Fatalf("insert benchmark observation entities: %v", err)
	}

	observedAt := time.Date(2026, 7, 12, 9, 30, 0, 0, time.UTC)
	first, err := repo.UpsertBenchmarkObservation(ctx, domain.BenchmarkObservation{
		ID:                 runID + "-observation-first",
		BenchmarkEntityID:  benchmarkID,
		ObservedAt:         observedAt,
		Value:              "4.25",
		Unit:               "percent",
		SourceName:         runID + "-source-a",
		SourceURL:          "https://example.com/source-a",
		ExternalSeriesCode: "SERIES-A",
		QualityStatus:      domain.BenchmarkObservationQualityRaw,
	})
	if err != nil {
		t.Fatalf("UpsertBenchmarkObservation(first) error = %v", err)
	}
	if !first.Created {
		t.Fatal("first UpsertBenchmarkObservation() should create a row")
	}

	updated, err := repo.UpsertBenchmarkObservation(ctx, domain.BenchmarkObservation{
		ID:                 runID + "-observation-retry",
		BenchmarkEntityID:  benchmarkID,
		ObservedAt:         observedAt,
		Value:              "4.30",
		Unit:               "percent",
		SourceName:         runID + "-source-a",
		SourceURL:          "https://example.com/source-a-updated",
		ExternalSeriesCode: "SERIES-A",
		QualityStatus:      domain.BenchmarkObservationQualityValidated,
	})
	if err != nil {
		t.Fatalf("UpsertBenchmarkObservation(conflict) error = %v", err)
	}
	if updated.Created {
		t.Fatal("same benchmark/time/source should update the existing row")
	}
	if updated.Observation.ID != first.Observation.ID {
		t.Fatalf("updated observation ID = %q, want original ID %q", updated.Observation.ID, first.Observation.ID)
	}
	if updated.Observation.Value != "4.30" || updated.Observation.QualityStatus != domain.BenchmarkObservationQualityValidated {
		t.Fatalf("updated observation = %+v, want updated value and quality status", updated.Observation)
	}

	for _, observation := range []domain.BenchmarkObservation{
		{
			ID:                runID + "-observation-source-b",
			BenchmarkEntityID: benchmarkID,
			ObservedAt:        observedAt,
			Value:             "4.31",
			Unit:              "percent",
			SourceName:        runID + "-source-b",
			SourceURL:         "https://example.com/source-b",
			QualityStatus:     domain.BenchmarkObservationQualityRaw,
		},
		{
			ID:                runID + "-observation-latest",
			BenchmarkEntityID: benchmarkID,
			ObservedAt:        observedAt.Add(time.Hour),
			Value:             "4.32",
			Unit:              "percent",
			SourceName:        runID + "-source-a",
			SourceURL:         "https://example.com/source-a",
			QualityStatus:     domain.BenchmarkObservationQualityRaw,
		},
		{
			ID:                runID + "-observation-other-benchmark",
			BenchmarkEntityID: otherBenchmarkID,
			ObservedAt:        observedAt.Add(2 * time.Hour),
			Value:             "3.80",
			Unit:              "percent",
			SourceName:        runID + "-source-a",
			SourceURL:         "https://example.com/source-a",
			QualityStatus:     domain.BenchmarkObservationQualityRaw,
		},
	} {
		result, err := repo.UpsertBenchmarkObservation(ctx, observation)
		if err != nil {
			t.Fatalf("UpsertBenchmarkObservation(%s) error = %v", observation.ID, err)
		}
		if !result.Created {
			t.Fatalf("UpsertBenchmarkObservation(%s) should create a distinct row", observation.ID)
		}
	}

	filtered, err := repo.ListBenchmarkObservations(ctx, BenchmarkObservationFilter{BenchmarkEntityID: benchmarkID})
	if err != nil {
		t.Fatalf("ListBenchmarkObservations(filtered) error = %v", err)
	}
	if got, want := len(filtered), 3; got != want {
		t.Fatalf("filtered observations length = %d, want %d", got, want)
	}
	if !filtered[0].ObservedAt.Equal(observedAt.Add(time.Hour)) {
		t.Fatalf("first filtered observed_at = %s, want latest %s", filtered[0].ObservedAt, observedAt.Add(time.Hour))
	}
	if filtered[1].SourceName == filtered[2].SourceName {
		t.Fatalf("same-time observations should preserve different sources: %+v", filtered)
	}

	all, err := repo.ListBenchmarkObservations(ctx, BenchmarkObservationFilter{})
	if err != nil {
		t.Fatalf("ListBenchmarkObservations(empty filter) error = %v", err)
	}
	positions := map[string]int{}
	for index, observation := range all {
		if observation.BenchmarkEntityID == benchmarkID || observation.BenchmarkEntityID == otherBenchmarkID {
			positions[observation.ID] = index
		}
	}
	if got, want := len(positions), 4; got != want {
		t.Fatalf("empty filter returned %d integration observations, want %d", got, want)
	}
	if positions[NormalizeUUID(runID+"-observation-other-benchmark")] >= positions[NormalizeUUID(runID+"-observation-latest")] {
		t.Fatalf("empty-filter observations are not ordered by observed_at descending: %+v", all)
	}

	_, err = repo.UpsertBenchmarkObservation(ctx, domain.BenchmarkObservation{
		ID:                runID + "-observation-index",
		BenchmarkEntityID: indexID,
		ObservedAt:        observedAt,
		Value:             "20",
		Unit:              "points",
		SourceName:        runID + "-source-a",
		QualityStatus:     domain.BenchmarkObservationQualityRaw,
	})
	if err == nil {
		t.Fatal("UpsertBenchmarkObservation(index) error = nil, want non-benchmark entity rejection")
	}
}
