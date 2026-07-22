package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/testsupport"
)

var testConnectorKeys = []string{
	"parallel_search", "tavily", "bocha", "cls_telegraph",
	"eastmoney_fastnews", "eastmoney_stock_news", "stcn_quicknews",
}

func TestMigrateSeedsCollectorV1(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_seed_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer database.Close()

	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	store := postgres.New(database)
	if !store.SchemaReady(ctx) {
		t.Fatal("migrated schema is not ready")
	}
	version, err := store.GetAgentVersion(ctx, "collector.v1")
	if err != nil {
		t.Fatalf("get seeded agent version: %v", err)
	}
	if version.AgentKey != "collector" {
		t.Fatalf("agent key = %q, want collector", version.AgentKey)
	}
	if version.Version != "collector.v1" {
		t.Fatalf("version = %q, want collector.v1", version.Version)
	}
	if _, err := database.Exec(ctx, `ALTER TABLE provider_configs DROP COLUMN updated_at`); err != nil {
		t.Fatal(err)
	}
	if store.SchemaReady(ctx) {
		t.Fatal("schema with a missing required column reported ready")
	}
}

func TestOpenRejectsTidewiseDataDatabase(t *testing.T) {
	_, err := postgres.Open(context.Background(), "postgres://user:password@localhost:5432/tidewise_local?sslmode=disable")
	if err == nil {
		t.Fatal("expected Tidewise Data database to be rejected before connection")
	}
}

func TestCompleteExecutionRejectsPendingCandidatesBeforeDatabaseWrite(t *testing.T) {
	t.Parallel()

	err := (&postgres.Store{}).CompleteExecution(context.Background(), agentrun.ExecutionCompletion{
		ExecutionID:     "execution",
		Status:          agentrun.StatusSucceeded,
		CandidateCounts: map[string]int{"results_pending": 1},
		CompletedAt:     time.Now(),
	})
	if err == nil {
		t.Fatal("successful completion accepted pending Candidates")
	}
}

func TestCreateExecutionEnforcesIdempotencyAndSingleActiveRun(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_execution_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer database.Close()
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	store := postgres.New(database)
	if err := store.FailStaleExecutions(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("clear stale execution: %v", err)
	}

	key := "test-create-" + time.Now().UTC().Format("20060102150405.000000000")
	first, disposition, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: key,
		Prompt:         "采集最近一周中国半导体产业链资讯\n并保留直接来源。",
		CreatedAt:      time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC),
		AgentVersion:   "collector.v1",
		InvocationKeys: testConnectorKeys,
	})
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	if disposition != agentrun.ExecutionCreated {
		t.Fatalf("disposition = %q, want %q", disposition, agentrun.ExecutionCreated)
	}
	if first.ID == "" || first.Status != agentrun.StatusQueued {
		t.Fatalf("created execution = %#v", first)
	}
	if len(first.Invocations) != 7 {
		t.Fatalf("invocation count = %d, want 7", len(first.Invocations))
	}

	replayed, disposition, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: key,
		Prompt:         "采集最近一周中国半导体产业链资讯\n并保留直接来源。",
		CreatedAt:      time.Now().UTC(),
		AgentVersion:   "collector.v1",
		InvocationKeys: testConnectorKeys,
	})
	if err != nil {
		t.Fatalf("replay execution: %v", err)
	}
	if disposition != agentrun.ExecutionReplayed || replayed.ID != first.ID {
		t.Fatalf("replay = %#v, %q; want original %q", replayed, disposition, first.ID)
	}

	_, _, err = store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: key,
		Prompt:         "不同的采集意图",
		CreatedAt:      time.Now().UTC(),
		AgentVersion:   "collector.v1",
		InvocationKeys: testConnectorKeys,
	})
	if !errors.Is(err, agentrun.ErrIdempotencyConflict) {
		t.Fatalf("different prompt error = %v, want idempotency conflict", err)
	}

	_, _, err = store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: key + "-other",
		Prompt:         "另一轮采集",
		CreatedAt:      time.Now().UTC(),
		AgentVersion:   "collector.v1",
		InvocationKeys: testConnectorKeys,
	})
	var active *agentrun.ActiveExecutionError
	if !errors.As(err, &active) || active.ExecutionID != first.ID {
		t.Fatalf("second active error = %v, want active execution %q", err, first.ID)
	}
}

func TestExecutionRepositoryEnforcesStateGraphAndRollsBackTerminalFailure(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_state_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	store := postgres.New(database)
	now := time.Now().UTC()
	execution, _, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "state-graph", Prompt: "prompt", CreatedAt: now,
		AgentVersion: "collector.v1", InvocationKeys: []string{"one"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, `
		CREATE FUNCTION reject_invocation_update() RETURNS trigger AS $$
		BEGIN RAISE EXCEPTION 'injected invocation update failure'; END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER reject_invocation_update BEFORE UPDATE ON connector_invocations
		FOR EACH ROW EXECUTE FUNCTION reject_invocation_update();
	`); err != nil {
		t.Fatal(err)
	}
	if err := store.FailExecutionAndIncompleteInvocations(ctx, agentrun.ExecutionFailure{
		ExecutionID:       execution.ID,
		ErrorCode:         "failed",
		ErrorSummary:      "safe",
		NotInvokedSummary: "not invoked",
		CompletedAt:       now,
	}); err == nil {
		t.Fatal("expected terminal transaction failure")
	}
	rolledBack, err := store.GetExecution(ctx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Status != agentrun.StatusQueued || rolledBack.Invocations[0].Status != agentrun.InvocationPending {
		t.Fatalf("terminal transaction was not rolled back: %#v", rolledBack)
	}
	if _, err := database.Exec(ctx, `DROP TRIGGER reject_invocation_update ON connector_invocations; DROP FUNCTION reject_invocation_update()`); err != nil {
		t.Fatal(err)
	}
	if err := store.SetExecutionStatus(ctx, execution.ID, agentrun.StatusCollecting, now); err == nil {
		t.Fatal("queued execution advanced directly to collecting")
	}
	if err := store.SetExecutionStatus(ctx, execution.ID, agentrun.StatusPlanning, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SetExecutionStatus(ctx, execution.ID, agentrun.StatusPlanning, now); err == nil {
		t.Fatal("planning execution re-entered planning")
	}
	if err := store.CompleteExecution(ctx, agentrun.ExecutionCompletion{
		ExecutionID: execution.ID,
		Status:      agentrun.StatusPlanning,
		CompletedAt: now,
	}); err == nil {
		t.Fatal("active status accepted as terminal completion")
	}
	if err := store.SetExecutionStatus(ctx, execution.ID, agentrun.StatusCollecting, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SetExecutionStatus(ctx, execution.ID, agentrun.StatusMaterializing, now); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteExecution(ctx, agentrun.ExecutionCompletion{
		ExecutionID:     execution.ID,
		Status:          agentrun.StatusSucceededNoChange,
		CandidateCounts: map[string]int{"results_pending": 0},
		Artifacts:       map[string]string{},
		CompletedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestProviderConfigurationUsesCurrentRowsAndRedactedViews(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_provider_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer database.Close()
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	store := postgres.New(database)

	configs := []agentrun.ProviderConfig{
		{Key: "deepseek", BaseURL: "https://deepseek.test/v1", Model: "deepseek-chat", APIKey: "deepseek-secret-1234"},
		{Key: "parallel_search", BaseURL: "https://parallel.test/search", APIKey: "parallel-secret-1234"},
		{Key: "tavily", BaseURL: "https://tavily.test/search", APIKey: "tavily-secret-1234"},
		{Key: "bocha", BaseURL: "https://bocha.test/search", APIKey: "bocha-secret-1234"},
		{Key: "cls_telegraph", BaseURL: "https://cls.test/roll"},
		{Key: "eastmoney_fastnews", BaseURL: "https://eastmoney.test/fast"},
		{Key: "eastmoney_stock_news", BaseURL: "https://eastmoney.test/search"},
		{Key: "stcn_quicknews", BaseURL: "https://stcn.test/list"},
	}
	for _, config := range configs {
		if err := store.UpsertProviderConfig(ctx, config); err != nil {
			t.Fatalf("upsert %s: %v", config.Key, err)
		}
	}
	configs[0].Model = "deepseek-reasoner"
	if err := store.UpsertProviderConfig(ctx, configs[0]); err != nil {
		t.Fatalf("replace DeepSeek config: %v", err)
	}

	loaded, err := store.LoadProviderConfigs(ctx)
	if err != nil {
		t.Fatalf("load Provider configs: %v", err)
	}
	if loaded["deepseek"].Model != "deepseek-reasoner" || loaded["deepseek"].APIKey != "deepseek-secret-1234" {
		t.Fatalf("DeepSeek runtime config = %#v", loaded["deepseek"])
	}
	if len(loaded) != 8 {
		t.Fatalf("Provider config count = %d, want 8", len(loaded))
	}

	views, err := store.ListProviderConfigViews(ctx)
	if err != nil {
		t.Fatalf("list Provider config views: %v", err)
	}
	for _, view := range views {
		if view.Key == "deepseek" {
			if !view.KeyConfigured || view.MaskedKey != "***1234" {
				t.Fatalf("DeepSeek redacted view = %#v", view)
			}
			if view.Model != "deepseek-reasoner" {
				t.Fatalf("DeepSeek model = %q", view.Model)
			}
		}
	}
	encoded, err := json.Marshal(views)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "deepseek-secret-1234") {
		t.Fatalf("view leaked API key: %s", encoded)
	}
}
