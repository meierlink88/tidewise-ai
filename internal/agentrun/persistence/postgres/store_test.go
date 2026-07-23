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

func TestMigrationBackfillsHistoricalTerminalStopReasons(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_upgrade_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec(ctx, `CREATE TABLE schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"001_agent_registry.sql", "002_executions.sql", "003_provider_configs.sql"} {
		payload, err := os.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(ctx, string(payload)); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, "migrations/"+name); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	cases := []struct {
		key       string
		status    agentrun.ExecutionStatus
		errorCode string
		want      string
	}{
		{"historical-succeeded", agentrun.StatusSucceeded, "", "connectors_completed"},
		{"historical-no-change", agentrun.StatusSucceededNoChange, "", "connectors_completed"},
		{"historical-partial", agentrun.StatusPartiallySucceeded, "", "completed_with_connector_failures"},
		{"historical-failed", agentrun.StatusFailed, "execution_failed", "agent_or_tool_limit"},
		{"historical-all-failed", agentrun.StatusFailed, "all_connectors_failed", "completed_with_connector_failures"},
	}
	for _, testCase := range cases {
		_, err := database.Exec(ctx, `
			INSERT INTO agent_executions (
				execution_id, agent_version, idempotency_key, prompt, prompt_sha256,
				prompt_bytes, status, error_code, created_at, completed_at, updated_at
			) VALUES (gen_random_uuid(), 'collector.v1', $1, 'prompt', $2, 6, $3, NULLIF($4, ''), $5, $5, $5)
		`, testCase.key, strings.Repeat("a", 64), testCase.status, testCase.errorCode, now)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	rows, err := database.Query(ctx, `SELECT idempotency_key, stop_reason FROM agent_executions ORDER BY idempotency_key`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := make(map[string]string, len(cases))
	for _, testCase := range cases {
		want[testCase.key] = testCase.want
	}
	seen := 0
	for rows.Next() {
		var key string
		var stopReason string
		if err := rows.Scan(&key, &stopReason); err != nil {
			t.Fatal(err)
		}
		if stopReason != want[key] {
			t.Fatalf("historical execution %q stop reason = %q, want %q", key, stopReason, want[key])
		}
		seen++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if seen != len(want) {
		t.Fatalf("backfilled rows = %d, want %d", seen, len(want))
	}
}

func TestOpenRejectsTidewiseDataDatabase(t *testing.T) {
	_, err := postgres.Open(context.Background(), "postgres://user:password@localhost:5432/tidewise_local?sslmode=disable")
	if err == nil {
		t.Fatal("expected Tidewise Data database to be rejected before connection")
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

	skipped, disposition, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: key + "-other",
		Prompt:         "另一轮采集",
		CreatedAt:      time.Now().UTC(),
		AgentVersion:   "collector.v1",
		InvocationKeys: testConnectorKeys,
	})
	if err != nil {
		t.Fatal(err)
	}
	if disposition != agentrun.ExecutionSkipped || skipped.Status != agentrun.StatusSkipped ||
		skipped.StopReason != "skipped_previous_run_active" ||
		skipped.BlockedByExecutionID != first.ID {
		t.Fatalf("skipped execution = %#v, disposition=%q", skipped, disposition)
	}
	if len(skipped.Invocations) != len(testConnectorKeys) {
		t.Fatalf("skipped invocations = %#v", skipped.Invocations)
	}
	for _, invocation := range skipped.Invocations {
		if invocation.Status != agentrun.InvocationNotInvoked || invocation.ErrorCode != "not_invoked" {
			t.Fatalf("skipped invocation = %#v", invocation)
		}
	}
	replayedSkipped, replayDisposition, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: key + "-other",
		Prompt:         "另一轮采集",
		CreatedAt:      time.Now().UTC(),
		AgentVersion:   "collector.v1",
		InvocationKeys: testConnectorKeys,
	})
	if err != nil || replayDisposition != agentrun.ExecutionReplayed || replayedSkipped.ID != skipped.ID {
		t.Fatalf("skipped replay = %#v, %q, %v", replayedSkipped, replayDisposition, err)
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
		StopReason:        "agent_or_tool_limit",
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
	if err := store.SetExecutionStatus(ctx, execution.ID, agentrun.StatusCollecting, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SetExecutionStatus(ctx, execution.ID, agentrun.StatusMaterializing, now); err != nil {
		t.Fatal(err)
	}
}

func TestPreparedPublicationProtectsAndIdempotentlyCommitsExecution(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_publication_test")
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
		IdempotencyKey: "publication", Prompt: "prompt", CreatedAt: now,
		AgentVersion: "collector.v1", InvocationKeys: []string{"one"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []agentrun.ExecutionStatus{
		agentrun.StatusPlanning, agentrun.StatusCollecting, agentrun.StatusMaterializing,
	} {
		if err := store.SetExecutionStatus(ctx, execution.ID, status, now); err != nil {
			t.Fatal(err)
		}
	}
	reference := agentrun.PublicationReference{
		ExecutionID: execution.ID, PlanPath: "/tmp/plan.json",
		PlanSHA256: strings.Repeat("a", 64), PreparedAt: now,
	}
	if err := store.PreparePublication(ctx, reference); err != nil {
		t.Fatal(err)
	}
	if err := store.PreparePublication(ctx, reference); err != nil {
		t.Fatalf("idempotent prepare: %v", err)
	}
	conflict := reference
	conflict.PlanSHA256 = strings.Repeat("b", 64)
	if err := store.PreparePublication(ctx, conflict); err == nil {
		t.Fatal("conflicting publication identity was accepted")
	}
	if err := store.FailStaleExecutions(ctx, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	protected, err := store.GetExecution(ctx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if protected.Status != agentrun.StatusMaterializing {
		t.Fatalf("prepared Execution status = %q", protected.Status)
	}
	if err := store.FailExecutionAndIncompleteInvocations(ctx, agentrun.ExecutionFailure{
		ExecutionID: execution.ID, ErrorCode: "injected", ErrorSummary: "safe",
		StopReason: "agent_or_tool_limit", NotInvokedSummary: "not invoked", CompletedAt: now,
	}); err == nil {
		t.Fatal("prepared Execution was downgraded to failed")
	}
	references, err := store.ListPreparedPublications(ctx)
	if err != nil || len(references) != 1 || references[0].PlanSHA256 != reference.PlanSHA256 {
		t.Fatalf("prepared references = %#v, err=%v", references, err)
	}
	completion := agentrun.ExecutionCompletion{
		ExecutionID: execution.ID, Status: agentrun.StatusSucceeded,
		StopReason:      "connectors_completed",
		CandidateCounts: map[string]int{"results_pending": 0, "accepted": 1},
		Artifacts:       map[string]string{"manifest": "/tmp/manifest.json"},
		CompletedAt:     now.Add(time.Minute),
	}
	if err := store.CommitPreparedPublication(ctx, reference, completion); err != nil {
		t.Fatal(err)
	}
	if err := store.CommitPreparedPublication(ctx, reference, completion); err != nil {
		t.Fatalf("idempotent commit: %v", err)
	}
	committed, err := store.GetExecution(ctx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if committed.Status != agentrun.StatusSucceeded ||
		committed.StopReason != "connectors_completed" ||
		committed.Artifacts["manifest"] != "/tmp/manifest.json" {
		t.Fatalf("committed Execution = %#v", committed)
	}
	references, err = store.ListPreparedPublications(ctx)
	if err != nil || len(references) != 0 {
		t.Fatalf("committed references = %#v, err=%v", references, err)
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
