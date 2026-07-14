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
