package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/testsupport"
	"github.com/jackc/pgx/v5/pgconn"
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
	if _, err := database.Exec(ctx, `ALTER TABLE connector_configs DROP COLUMN updated_at`); err != nil {
		t.Fatal(err)
	}
	if store.SchemaReady(ctx) {
		t.Fatal("schema with a missing required column reported ready")
	}
}

func TestMigrationSplitsModelAndConnectorConfigurations(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_config_split_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	applyHistoricalMigrations(t, ctx, database, []string{
		"001_agent_registry.sql",
		"002_executions.sql",
		"003_provider_configs.sql",
		"004_collector_audit_publication.sql",
	})
	configs := [][]string{
		{"deepseek", "https://deepseek.test/v1", "deepseek-chat", "deepseek-secret"},
		{"parallel_search", "https://parallel.test/search", "", "parallel-secret"},
		{"tavily", "https://tavily.test/search", "", "tavily-secret"},
		{"bocha", "https://bocha.test/search", "", "bocha-secret"},
		{"cls_telegraph", "https://cls.test/roll", "", ""},
		{"eastmoney_fastnews", "https://eastmoney.test/fast", "", ""},
		{"eastmoney_stock_news", "https://eastmoney.test/search", "", ""},
		{"stcn_quicknews", "https://stcn.test/list", "", ""},
	}
	for _, config := range configs {
		if _, err := database.Exec(ctx, `
			INSERT INTO provider_configs (provider_key, base_url, model, api_key)
			VALUES ($1, $2, $3, $4)
		`, config[0], config[1], config[2], config[3]); err != nil {
			t.Fatal(err)
		}
	}

	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate historical Provider configurations: %v", err)
	}
	var oldTable, modelTable, connectorTable *string
	if err := database.QueryRow(ctx, `
		SELECT to_regclass('provider_configs')::text,
		       to_regclass('model_provider_configs')::text,
		       to_regclass('connector_configs')::text
	`).Scan(&oldTable, &modelTable, &connectorTable); err != nil {
		t.Fatal(err)
	}
	if oldTable != nil || modelTable == nil || connectorTable == nil {
		t.Fatalf("configuration tables old=%v model=%v connector=%v", oldTable, modelTable, connectorTable)
	}
	var providerKey, baseURL, model, apiKey string
	if err := database.QueryRow(ctx, `
		SELECT provider_key, base_url, model, api_key
		FROM model_provider_configs
	`).Scan(&providerKey, &baseURL, &model, &apiKey); err != nil {
		t.Fatal(err)
	}
	if providerKey != "deepseek" || baseURL != "https://deepseek.test/v1" ||
		model != "deepseek-chat" || apiKey != "deepseek-secret" {
		t.Fatalf("migrated Model Provider Configuration = %q %q %q %q", providerKey, baseURL, model, apiKey)
	}
	var connectorCount int
	if err := database.QueryRow(ctx, `SELECT count(*) FROM connector_configs`).Scan(&connectorCount); err != nil {
		t.Fatal(err)
	}
	if connectorCount != 7 {
		t.Fatalf("migrated Connector Configurations = %d, want 7", connectorCount)
	}
	var tavilyURL, tavilyKey string
	if err := database.QueryRow(ctx, `
		SELECT base_url, api_key FROM connector_configs WHERE connector_key = 'tavily'
	`).Scan(&tavilyURL, &tavilyKey); err != nil {
		t.Fatal(err)
	}
	if tavilyURL != "https://tavily.test/search" || tavilyKey != "tavily-secret" {
		t.Fatalf("migrated Tavily configuration = %q %q", tavilyURL, tavilyKey)
	}
	if !postgres.New(database).SchemaReady(ctx) {
		t.Fatal("split configuration schema is not ready")
	}
}

func TestMigrationRejectsUnknownProviderWithoutPartialSplit(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_config_unknown_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	applyHistoricalMigrations(t, ctx, database, []string{
		"001_agent_registry.sql",
		"002_executions.sql",
		"003_provider_configs.sql",
		"004_collector_audit_publication.sql",
	})
	if _, err := database.Exec(ctx, `
		INSERT INTO provider_configs (provider_key, base_url, model, api_key)
		VALUES ('unexpected_provider', 'https://unexpected.test', '', '')
	`); err != nil {
		t.Fatal(err)
	}

	if err := postgres.Migrate(ctx, database); err == nil {
		t.Fatal("migration accepted an unknown Provider configuration")
	}
	var oldTable, modelTable, connectorTable *string
	if err := database.QueryRow(ctx, `
		SELECT to_regclass('provider_configs')::text,
		       to_regclass('model_provider_configs')::text,
		       to_regclass('connector_configs')::text
	`).Scan(&oldTable, &modelTable, &connectorTable); err != nil {
		t.Fatal(err)
	}
	if oldTable == nil || modelTable != nil || connectorTable != nil {
		t.Fatalf("failed migration left partial tables old=%v model=%v connector=%v", oldTable, modelTable, connectorTable)
	}
	var recorded bool
	if err := database.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM schema_migrations
			WHERE version = 'migrations/005_split_provider_configs.sql'
		)
	`).Scan(&recorded); err != nil {
		t.Fatal(err)
	}
	if recorded {
		t.Fatal("failed migration was recorded")
	}
}

func TestMigrationAddsAgentSchedulesAndBackfillsExecutionTriggerMetadata(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_schedule_upgrade_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	applyHistoricalMigrations(t, ctx, database, []string{
		"001_agent_registry.sql",
		"002_executions.sql",
		"003_provider_configs.sql",
		"004_collector_audit_publication.sql",
		"005_split_provider_configs.sql",
	})
	executionID := "00000000-0000-0000-0000-000000000023"
	if _, err := database.Exec(ctx, `
		INSERT INTO agent_executions (
			execution_id, agent_version, idempotency_key, prompt, prompt_sha256,
			prompt_bytes, status, stop_reason, created_at, completed_at, updated_at
		) VALUES ($1, 'collector.v1', 'historical-schedule-upgrade', 'historical prompt',
		          $2, 17, 'succeeded', 'connectors_completed', now(), now(), now())
	`, executionID, strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}

	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	var scheduleTable *string
	if err := database.QueryRow(ctx, `SELECT to_regclass('agent_schedules')::text`).Scan(&scheduleTable); err != nil {
		t.Fatal(err)
	}
	if scheduleTable == nil || *scheduleTable != "agent_schedules" {
		t.Fatalf("agent_schedules table = %v", scheduleTable)
	}
	var agentKey, triggerSource string
	var inputPayload []byte
	if err := database.QueryRow(ctx, `
		SELECT agent_key, trigger_source, input_payload
		FROM agent_executions
		WHERE execution_id = $1
	`, executionID).Scan(&agentKey, &triggerSource, &inputPayload); err != nil {
		t.Fatal(err)
	}
	var input map[string]string
	if err := json.Unmarshal(inputPayload, &input); err != nil {
		t.Fatal(err)
	}
	if agentKey != "collector" || triggerSource != "api" || input["prompt"] != "historical prompt" {
		t.Fatalf("backfilled execution agent=%q trigger=%q input=%#v", agentKey, triggerSource, input)
	}
	var promptNullable string
	if err := database.QueryRow(ctx, `
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'agent_executions'
		  AND column_name = 'prompt'
	`).Scan(&promptNullable); err != nil {
		t.Fatal(err)
	}
	if promptNullable != "YES" {
		t.Fatalf("Collector prompt compatibility projection nullable = %q, want YES", promptNullable)
	}
	var activeIndex string
	if err := database.QueryRow(ctx, `
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = current_schema()
		  AND indexname = 'agent_executions_one_active'
	`).Scan(&activeIndex); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(activeIndex, "(agent_key)") {
		t.Fatalf("active execution index = %q, want per-agent uniqueness", activeIndex)
	}
	if !postgres.New(database).SchemaReady(ctx) {
		t.Fatal("schedule migration schema is not ready")
	}
}

func applyHistoricalMigrations(
	t *testing.T,
	ctx context.Context,
	database interface {
		Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	},
	names []string,
) {
	t.Helper()
	if _, err := database.Exec(ctx, `
		CREATE TABLE schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
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
	if first.AgentKey != "collector" || first.TriggerSource != agentrun.TriggerAPI ||
		!first.TriggeredAt.Equal(first.CreatedAt) {
		t.Fatalf("execution trigger metadata = %#v", first)
	}
	var firstInput map[string]string
	if err := json.Unmarshal(first.InputPayload, &firstInput); err != nil {
		t.Fatal(err)
	}
	if firstInput["prompt"] != "采集最近一周中国半导体产业链资讯\n并保留直接来源。" {
		t.Fatalf("execution input = %#v", firstInput)
	}
	if len(first.Invocations) != 7 {
		t.Fatalf("invocation count = %d, want 7", len(first.Invocations))
	}
	if _, err := database.Exec(ctx, `
		INSERT INTO agent_definitions (agent_key, display_name)
		VALUES ('analyst', 'Analyst Agent');
		INSERT INTO agent_versions (version, agent_key)
		VALUES ('analyst.v1', 'analyst')
	`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "analyst-missing-input",
		CreatedAt:      time.Now().UTC(),
		AgentVersion:   "analyst.v1",
	}); err == nil {
		t.Fatal("generic Agent Execution accepted a missing Agent Input")
	}
	analyst, analystDisposition, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "analyst-active",
		InputPayload:   json.RawMessage(`{"thesis":"分析产业链"}`),
		CreatedAt:      time.Now().UTC(),
		AgentVersion:   "analyst.v1",
	})
	if err != nil || analystDisposition != agentrun.ExecutionCreated ||
		analyst.AgentKey != "analyst" || len(analyst.Invocations) != 0 {
		t.Fatalf("different-Agent execution = %#v, %q, %v", analyst, analystDisposition, err)
	}
	blockedAnalyst, blockedDisposition, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "analyst-overlap",
		InputPayload:   json.RawMessage(`{"thesis":"再次分析产业链"}`),
		CreatedAt:      time.Now().UTC(),
		AgentVersion:   "analyst.v1",
	})
	if err != nil || blockedDisposition != agentrun.ExecutionSkipped ||
		blockedAnalyst.BlockedByExecutionID != analyst.ID {
		t.Fatalf("same-Agent overlap = %#v, %q, %v", blockedAnalyst, blockedDisposition, err)
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

func TestAgentScheduleRepositoryPreservesIdentityAcrossReplacement(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "storage_schedule_test")
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
	now := time.Date(2026, 7, 24, 2, 0, 0, 0, time.UTC)
	created, err := store.PutAgentSchedule(ctx, agentrun.PutAgentScheduleInput{
		AgentKey:       "collector",
		AgentVersion:   "collector.v1",
		Type:           agentrun.ScheduleCron,
		CronExpression: "0 */2 * * *",
		InputPayload:   json.RawMessage(`{"prompt":"采集最近两小时资讯"}`),
		Enabled:        true,
		UpdatedAt:      now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.AgentKey != "collector" || created.Type != agentrun.ScheduleCron ||
		created.CronExpression != "0 */2 * * *" || !created.Enabled {
		t.Fatalf("created Schedule = %#v", created)
	}

	replaced, err := store.PutAgentSchedule(ctx, agentrun.PutAgentScheduleInput{
		AgentKey:     "collector",
		AgentVersion: "collector.v1",
		Type:         agentrun.ScheduleDaily,
		DailyTimes:   []string{"09:00", "13:30"},
		InputPayload: json.RawMessage(`{"prompt":"采集固定时点资讯"}`),
		Enabled:      false,
		UpdatedAt:    now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if replaced.ID != created.ID || replaced.Type != agentrun.ScheduleDaily ||
		len(replaced.DailyTimes) != 2 || replaced.DailyTimes[0] != "09:00" ||
		replaced.CronExpression != "" || replaced.Enabled {
		t.Fatalf("replaced Schedule = %#v", replaced)
	}
	loaded, err := store.GetAgentSchedule(ctx, "collector")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != created.ID || string(loaded.InputPayload) != `{"prompt": "采集固定时点资讯"}` {
		t.Fatalf("loaded Schedule = %#v", loaded)
	}
	triggeredAt := now.Add(2 * time.Minute)
	execution, disposition, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "scheduled-collector-execution",
		InputPayload:   json.RawMessage(`{"prompt":"采集固定时点资讯"}`),
		Prompt:         "采集固定时点资讯",
		CreatedAt:      triggeredAt,
		TriggeredAt:    triggeredAt,
		TriggerSource:  agentrun.TriggerSchedule,
		ScheduleID:     loaded.ID,
		AgentVersion:   "collector.v1",
		InvocationKeys: testConnectorKeys,
	})
	if err != nil || disposition != agentrun.ExecutionCreated {
		t.Fatalf("scheduled Execution = %#v, disposition=%q, err=%v", execution, disposition, err)
	}
	if execution.TriggerSource != agentrun.TriggerSchedule ||
		execution.ScheduleID != loaded.ID ||
		!execution.TriggeredAt.Equal(triggeredAt) ||
		string(execution.InputPayload) != `{"prompt": "采集固定时点资讯"}` {
		t.Fatalf("scheduled Execution metadata = %#v", execution)
	}
	listed, err := store.ListAgentSchedules(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("listed Schedules = %#v, err=%v", listed, err)
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

func TestModelAndConnectorConfigurationsUseSeparateCurrentRowsAndRedactedViews(t *testing.T) {
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

	modelConfig := agentrun.ModelProviderConfig{
		ProviderKey: "deepseek",
		BaseURL:     "https://deepseek.test/v1",
		Model:       "deepseek-chat",
		APIKey:      "deepseek-secret-1234",
	}
	if err := store.UpsertModelProviderConfig(ctx, modelConfig); err != nil {
		t.Fatalf("upsert DeepSeek: %v", err)
	}
	connectorConfigs := []agentrun.ConnectorConfig{
		{ConnectorKey: "parallel_search", BaseURL: "https://parallel.test/search", APIKey: "parallel-secret-1234"},
		{ConnectorKey: "tavily", BaseURL: "https://tavily.test/search", APIKey: "tavily-secret-1234"},
		{ConnectorKey: "bocha", BaseURL: "https://bocha.test/search", APIKey: "bocha-secret-1234"},
		{ConnectorKey: "cls_telegraph", BaseURL: "https://cls.test/roll"},
		{ConnectorKey: "eastmoney_fastnews", BaseURL: "https://eastmoney.test/fast"},
		{ConnectorKey: "eastmoney_stock_news", BaseURL: "https://eastmoney.test/search"},
		{ConnectorKey: "stcn_quicknews", BaseURL: "https://stcn.test/list"},
	}
	for _, config := range connectorConfigs {
		if err := store.UpsertConnectorConfig(ctx, config); err != nil {
			t.Fatalf("upsert %s: %v", config.ConnectorKey, err)
		}
	}
	modelConfig.Model = "deepseek-reasoner"
	if err := store.UpsertModelProviderConfig(ctx, modelConfig); err != nil {
		t.Fatalf("replace DeepSeek config: %v", err)
	}

	models, err := store.LoadModelProviderConfigs(ctx)
	if err != nil {
		t.Fatalf("load Model Provider Configurations: %v", err)
	}
	if models["deepseek"].Model != "deepseek-reasoner" || models["deepseek"].APIKey != "deepseek-secret-1234" {
		t.Fatalf("DeepSeek runtime config = %#v", models["deepseek"])
	}
	if len(models) != 1 {
		t.Fatalf("Model Provider Configuration count = %d, want 1", len(models))
	}
	connectors, err := store.LoadConnectorConfigs(ctx)
	if err != nil {
		t.Fatalf("load Connector Configurations: %v", err)
	}
	if len(connectors) != 7 || connectors["tavily"].APIKey != "tavily-secret-1234" {
		t.Fatalf("Connector Configurations = %#v", connectors)
	}
	modelViews, err := store.ListModelProviderConfigViews(ctx)
	if err != nil {
		t.Fatalf("list Model Provider Configuration views: %v", err)
	}
	if len(modelViews) != 1 || modelViews[0].ProviderKey != "deepseek" ||
		!modelViews[0].KeyConfigured || modelViews[0].MaskedKey != "***1234" ||
		modelViews[0].Model != "deepseek-reasoner" {
		t.Fatalf("DeepSeek redacted view = %#v", modelViews)
	}
	connectorViews, err := store.ListConnectorConfigViews(ctx)
	if err != nil {
		t.Fatalf("list Connector Configuration views: %v", err)
	}
	if len(connectorViews) != 7 {
		t.Fatalf("Connector Configuration views = %d, want 7", len(connectorViews))
	}
	encoded, err := json.Marshal(struct {
		Models     []agentrun.ModelProviderConfigView `json:"models"`
		Connectors []agentrun.ConnectorConfigView     `json:"connectors"`
	}{Models: modelViews, Connectors: connectorViews})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "deepseek-secret-1234") ||
		strings.Contains(string(encoded), "tavily-secret-1234") {
		t.Fatalf("configuration view leaked API key: %s", encoded)
	}
}
