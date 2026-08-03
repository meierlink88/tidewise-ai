package postgres_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/testsupport"
)

var testConnectorKeys = []string{
	"parallel_search", "tavily", "bocha", "cls_telegraph",
	"eastmoney_fastnews", "eastmoney_stock_news", "stcn_quicknews",
}

func TestMigrationReportIsReadOnlyAndTracksPendingMigrations(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "migration_report_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	report, err := postgres.InspectMigrations(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	if report.CurrentVersion != "" || len(report.Applied) != 0 || len(report.Pending) != 14 {
		t.Fatalf("empty database migration report = %#v", report)
	}
	var ledger *string
	if err := database.QueryRow(ctx, `SELECT to_regclass('schema_migrations')::text`).Scan(&ledger); err != nil {
		t.Fatal(err)
	}
	if ledger != nil {
		t.Fatal("read-only migration report created schema_migrations")
	}

	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	report, err = postgres.InspectMigrations(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	if report.CurrentVersion != "014" ||
		len(report.Applied) != 14 || len(report.Pending) != 0 {
		t.Fatalf("migrated database report = %#v", report)
	}
}

func TestMigrateSeedsCurrentAgentVersions(t *testing.T) {
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
	extractorVersion, err := store.GetAgentVersion(ctx, "event-fact-extractor.v1")
	if err != nil {
		t.Fatalf("get seeded Event Fact Extractor version: %v", err)
	}
	if extractorVersion.AgentKey != "event-fact-extractor" {
		t.Fatalf("Event Fact Extractor agent key = %q", extractorVersion.AgentKey)
	}
	extractorV2, err := store.GetAgentVersion(ctx, eventfact.AgentVersion)
	if err != nil {
		t.Fatalf("get seeded Event Fact Extractor V2 version: %v", err)
	}
	if extractorV2.AgentKey != eventfact.AgentKey {
		t.Fatalf("Event Fact Extractor V2 agent key = %q", extractorV2.AgentKey)
	}
	semanticVersion, err := store.GetAgentVersion(ctx, "event-semantic-enricher.v1")
	if err != nil {
		t.Fatalf("get seeded Event Semantic Enricher version: %v", err)
	}
	if semanticVersion.AgentKey != "event-semantic-enricher" {
		t.Fatalf("Event Semantic Enricher agent key = %q", semanticVersion.AgentKey)
	}
	semanticV3, err := store.GetAgentVersion(ctx, eventsemantic.AgentVersion)
	if err != nil {
		t.Fatalf("get seeded Event Semantic V3 version: %v", err)
	}
	if semanticV3.AgentKey != eventsemantic.AgentKey {
		t.Fatalf("Event Semantic V3 agent key = %q", semanticV3.AgentKey)
	}
	execution, disposition, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "event-semantic-v3-registry-test",
		InputPayload:   json.RawMessage(`{"work_item_id":"11111111-1111-4111-8111-111111111111"}`),
		CreatedAt:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		AgentVersion:   eventsemantic.AgentVersion,
	})
	if err != nil || disposition != agentrun.ExecutionCreated || execution.AgentKey != eventsemantic.AgentKey {
		t.Fatalf("create Event Semantic V3 execution = %#v, %q, %v", execution, disposition, err)
	}
	if _, err := database.Exec(ctx, `ALTER TABLE connector_configs DROP COLUMN updated_at`); err != nil {
		t.Fatal(err)
	}
	if store.SchemaReady(ctx) {
		t.Fatal("schema with a missing required column reported ready")
	}
}

func TestSaveEventSemanticStageAuditIsIdempotent(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "event_semantic_stage_audit_test")
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
	eventID := "22222222-2222-4222-8222-222222222222"
	workID := "11111111-1111-4111-8111-111111111111"
	executionID := "33333333-3333-4333-8333-333333333333"
	if _, err := database.Exec(ctx, `
		INSERT INTO agent_executions(
			execution_id,agent_key,agent_version,idempotency_key,input_payload,
			trigger_source,triggered_at,status,created_at,updated_at
		) VALUES ($1,'event-semantic-enricher','event-semantic-enricher.v3',
			'stage-audit-execution','{}'::jsonb,'dependent',now(),'running',now(),now())
	`, executionID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, `
		INSERT INTO event_semantic_work_items(
			work_item_id,event_id,trigger_source,reason,idempotency_key,status,
			attempt_count,max_attempts,lease_expires_at,current_execution_id,created_at,updated_at
		) VALUES ($1,$2,'eligible_event','','stage-audit-test','running',1,2,now()+interval '5 minutes',$3,now(),now())
	`, workID, eventID, executionID); err != nil {
		t.Fatal(err)
	}
	store := postgres.New(database)
	audit := eventsemantic.StageAudit{
		ContractVersion: "event-semantic-stage-audit.v1", EventID: eventID,
		Mentions:   []eventsemantic.MentionAudit{{CandidateKey: "company", Mention: "某公司", EvidenceIDs: []string{"evidence"}}},
		Violations: []eventsemantic.StageViolationAudit{{Stage: "mention_extraction", Attempt: "initial", Codes: []string{"mention_key_invalid"}}},
		Isolations: []eventsemantic.CandidateIsolationAudit{{
			Stage: "mention_extraction", CandidateKey: "bad-mention", ReasonCode: "mention_key_invalid", Owner: "model",
		}},
	}
	if err := store.SaveStageAudit(ctx, executionID, audit); err != nil {
		t.Fatal(err)
	}
	resumeAudit := eventsemantic.StageAudit{
		ContractVersion: audit.ContractVersion, EventID: eventID,
		Violations: []eventsemantic.StageViolationAudit{{Stage: "entity_selection", Attempt: "repair", Codes: []string{"selection_missing"}}},
		Isolations: []eventsemantic.CandidateIsolationAudit{{
			Stage: "entity_selection", CandidateKey: "company", ReasonCode: "selection_missing", Owner: "model",
		}},
	}
	if err := store.SaveStageAudit(ctx, executionID, resumeAudit); err != nil {
		t.Fatal(err)
	}
	var contract string
	var mentionCount, violationCount, isolationCount int
	if err := database.QueryRow(ctx, `
		SELECT contract_version, jsonb_array_length(summary->'mentions'),
		       jsonb_array_length(summary->'violations'), jsonb_array_length(summary->'isolations')
		FROM event_semantic_stage_audits WHERE execution_id=$1
	`, executionID).Scan(&contract, &mentionCount, &violationCount, &isolationCount); err != nil {
		t.Fatal(err)
	}
	if contract != audit.ContractVersion || mentionCount != 1 || violationCount != 2 || isolationCount != 2 {
		t.Fatalf("contract=%q mentions=%d violations=%d isolations=%d", contract, mentionCount, violationCount, isolationCount)
	}
}

func TestSchemaReadyRequiresEventSemanticStageAuditShape(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "event_semantic_stage_audit_readiness_test")
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
	if !store.SchemaReady(ctx) {
		t.Fatal("fresh AgentRun schema is not ready")
	}
	if _, err := database.Exec(ctx, `ALTER TABLE event_semantic_stage_audits DROP COLUMN summary`); err != nil {
		t.Fatal(err)
	}
	if store.SchemaReady(ctx) {
		t.Fatal("schema without Event Semantic stage audit summary reported ready")
	}
}

func TestPreparePreviousReleaseRollbackRestoresPre011InvariantAndMigrationsReplay(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(
		ctx, databaseURL, "storage_release_rollback_test",
	)
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
	// Reproduce the interrupted legacy rollback observed in UAT: the old helper
	// removed only 012 while 011, 013, and 014 remained in the ledger.
	if _, err := database.Exec(ctx, `
		DELETE FROM schema_migrations
		WHERE version = 'migrations/012_event_semantic_entity_first_v3.sql'
	`); err != nil {
		t.Fatal(err)
	}
	eventID := "22222222-2222-4222-8222-222222222222"
	if _, err := database.Exec(ctx, `
		INSERT INTO event_semantic_work_items (
			work_item_id, event_id, trigger_source, reason, idempotency_key,
			status, attempt_count, max_attempts, created_at, updated_at
		) VALUES (
			'11111111-1111-4111-8111-111111111111', $1::uuid, 'eligible_event', '',
			'event-semantic-initial:' || $1::text, 'skipped', 0, 1, now(), now()
		)
	`, eventID); err != nil {
		t.Fatal(err)
	}
	if err := postgres.PreparePreviousReleaseRollback(ctx, database, "010"); err != nil {
		t.Fatal(err)
	}
	var journalUnitIndexExists bool
	if err := database.QueryRow(ctx, `
		SELECT to_regclass('event_publication_journal_unit_key_unique') IS NOT NULL
	`).Scan(&journalUnitIndexExists); err != nil {
		t.Fatal(err)
	}
	if !journalUnitIndexExists {
		t.Fatal("previous-release journal unit uniqueness was not restored")
	}
	report, err := postgres.InspectMigrations(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	if got := migrationVersions(report.Pending); !reflect.DeepEqual(got, []string{"011", "012", "013", "014"}) {
		t.Fatalf("rollback-compatible migration report = %#v", report)
	}
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	if !postgres.New(database).SchemaReady(ctx) {
		t.Fatal("replayed post-010 migrations schema is not ready")
	}
}

func TestPreparePreviousReleaseRollbackPreservesMigrationsOwnedByPreviousRelease(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(
		ctx, databaseURL, "storage_partial_release_rollback_test",
	)
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
	if err := postgres.PreparePreviousReleaseRollback(ctx, database, "013"); err != nil {
		t.Fatal(err)
	}
	report, err := postgres.InspectMigrations(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	if got := migrationVersions(report.Pending); !reflect.DeepEqual(got, []string{"014"}) {
		t.Fatalf("partial rollback migration report = %#v", report)
	}
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	if !postgres.New(database).SchemaReady(ctx) {
		t.Fatal("replayed migration 014 schema is not ready")
	}
}

func migrationVersions(migrations []postgres.Migration) []string {
	versions := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		versions = append(versions, migration.Version)
	}
	return versions
}

func TestHistoricalEventSemanticMaintenanceBlocksProcessingCycles(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(
		ctx, databaseURL, "storage_semantic_maintenance_lock_test",
	)
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
	maintenanceEntered := make(chan struct{})
	releaseMaintenance := make(chan struct{})
	maintenanceResult := make(chan error, 1)
	go func() {
		maintenanceResult <- store.WithHistoricalEventSemanticMaintenance(
			ctx,
			func() error {
				close(maintenanceEntered)
				select {
				case <-releaseMaintenance:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		)
	}()
	select {
	case <-maintenanceEntered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	processingEntered := make(chan struct{})
	processingResult := make(chan error, 1)
	go func() {
		processingResult <- store.WithEventSemanticProcessingPermit(
			ctx,
			func() error {
				close(processingEntered)
				return nil
			},
		)
	}()
	select {
	case <-processingEntered:
		t.Fatal("processing cycle entered while historical maintenance held the lock")
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseMaintenance)
	for label, result := range map[string]<-chan error{
		"maintenance": maintenanceResult,
		"processing":  processingResult,
	} {
		select {
		case err := <-result:
			if err != nil {
				t.Fatalf("%s lock operation: %v", label, err)
			}
		case <-ctx.Done():
			t.Fatalf("%s lock operation: %v", label, ctx.Err())
		}
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

func TestEventSemanticWorkItemRetriesAreOwnedByAgentRun(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "semantic_restart_test")
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
	now := time.Date(2026, 7, 29, 8, 30, 0, 0, time.UTC)
	eventID := "22222222-2222-4222-8222-222222222222"
	if _, err := store.EnsureInitialWorkItems(ctx, []eventsemantic.EligibleEvent{{EventID: eventID}}, now); err != nil {
		t.Fatal(err)
	}
	attempt, found, err := store.StartNextExecution(ctx, "worker-1", strings.Repeat("a", 64), now)
	if err != nil {
		t.Fatal(err)
	}
	if !found || attempt.WorkItem.EventID != eventID || attempt.WorkItem.AttemptCount != 1 {
		t.Fatalf("first attempt = %#v found=%v", attempt, found)
	}
	if err := store.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
		ExecutionID: attempt.ID, Status: "failed", ErrorCode: "temporary_failure",
		ErrorSummary: "retry", Retryable: true, CompletedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	second, found, err := store.StartNextExecution(
		ctx, "worker-1", strings.Repeat("a", 64), now.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !found || second.ID != attempt.ID ||
		second.WorkItem.ID != attempt.WorkItem.ID || second.WorkItem.AttemptCount != 2 {
		t.Fatalf("second attempt = %#v found=%v", second, found)
	}
	if err := store.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
		ExecutionID: second.ID, Status: "failed", ErrorCode: "permanent_failure",
		ErrorSummary: "exhausted", CompletedAt: now.Add(3 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	_, found, err = store.StartNextExecution(
		ctx, "worker-1", strings.Repeat("a", 64), now.Add(4*time.Minute),
	)
	if err != nil || found {
		t.Fatalf("exhausted Work Item found=%v err=%v", found, err)
	}
}

func TestHistoricalEventDispositionIsIdempotentAndRecoversOnlyValidFailures(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(
		ctx, databaseURL, "semantic_history_test",
	)
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
	now := time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC)
	invalidExisting := "22222222-2222-4222-8222-222222222221"
	invalidMissing := "22222222-2222-4222-8222-222222222222"
	validFailed := "22222222-2222-4222-8222-222222222223"
	for _, eventID := range []string{invalidExisting, validFailed} {
		if _, err := store.EnsureInitialWorkItems(
			ctx, []eventsemantic.EligibleEvent{{EventID: eventID}}, now,
		); err != nil {
			t.Fatal(err)
		}
		attempt, found, err := store.StartNextExecution(
			ctx, "history-worker", strings.Repeat("a", 64), now,
		)
		if err != nil || !found {
			t.Fatalf("start %s: found=%v err=%v", eventID, found, err)
		}
		for attemptNumber := 0; attemptNumber < 2; attemptNumber++ {
			if err := store.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
				ExecutionID: attempt.ID, Status: "failed", ErrorCode: "historical",
				ErrorSummary: "historical", Retryable: true,
				CompletedAt: now.Add(
					time.Duration(attemptNumber*2+1) * time.Minute,
				),
			}); err != nil {
				t.Fatal(err)
			}
			if attemptNumber == 0 {
				attempt, found, err = store.StartNextExecution(
					ctx, "history-worker", strings.Repeat("a", 64), now.Add(2*time.Minute),
				)
				if err != nil || !found {
					t.Fatalf("retry %s: found=%v err=%v", eventID, found, err)
				}
			}
		}
	}
	manifest := eventsemantic.HistoricalManifest{
		Version:     eventsemantic.HistoricalManifestVersion,
		GeneratedAt: now.Add(4 * time.Minute),
		InvalidEventIDs: []string{
			invalidExisting, invalidMissing,
		},
		ValidEventIDs: []string{validFailed},
	}

	plan, err := store.PlanHistoricalEventDisposition(ctx, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if plan.SkippedCreated != 1 || plan.SkippedUpdated != 1 ||
		plan.ValidFailuresRecovered != 1 {
		t.Fatalf("plan = %#v", plan)
	}
	applied, err := store.ApplyHistoricalEventDisposition(
		ctx, manifest, now.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if applied.SkippedCreated != 1 || applied.SkippedUpdated != 1 ||
		applied.ValidFailuresRecovered != 1 {
		t.Fatalf("applied = %#v", applied)
	}
	recovery, found, err := store.StartNextExecution(
		ctx, "history-worker", strings.Repeat("a", 64), now.Add(6*time.Minute),
	)
	if err != nil || !found || recovery.WorkItem.EventID != validFailed {
		t.Fatalf("start controlled recovery: attempt=%#v found=%v err=%v", recovery, found, err)
	}
	if err := store.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
		ExecutionID: recovery.ID, Status: "failed", ErrorCode: "still-failed",
		ErrorSummary: "controlled repair failed", Retryable: true,
		CompletedAt: now.Add(7 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	repeated, err := store.ApplyHistoricalEventDisposition(
		ctx, manifest, now.Add(8*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.AlreadySkipped != 2 ||
		repeated.FailedAfterAuditPreserved != 1 ||
		repeated.SkippedCreated != 0 ||
		repeated.ValidFailuresRecovered != 0 {
		t.Fatalf("repeated = %#v", repeated)
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
	var signalCount int
	if err := database.QueryRow(ctx, `
		SELECT count(*) FROM artifact_ready_signals WHERE collector_execution_id = $1
	`, execution.ID).Scan(&signalCount); err != nil {
		t.Fatal(err)
	}
	if signalCount != 1 {
		t.Fatalf("Artifact ready signal count = %d, want 1", signalCount)
	}
	if err := store.CommitPreparedPublication(ctx, reference, completion); err != nil {
		t.Fatalf("idempotent commit: %v", err)
	}
	if dispatched, err := store.DispatchPendingSignals(
		ctx, eventfact.AgentVersion, now.Add(2*time.Minute),
	); err != nil || dispatched != 1 {
		t.Fatalf("dispatch Artifact signal = %d, err=%v", dispatched, err)
	}
	var workCount int
	if err := database.QueryRow(ctx, `
		SELECT count(*)
		FROM event_extraction_work_items
		WHERE collector_execution_ids = ARRAY[$1::uuid]
		  AND extractor_agent_version = $2
	`, execution.ID, eventfact.AgentVersion).Scan(&workCount); err != nil {
		t.Fatal(err)
	}
	if workCount != 1 {
		t.Fatalf("Event extraction Work Item count = %d, want 1", workCount)
	}
	unplanned, unplannedExists, err := store.NextUnplannedWork(ctx)
	if err != nil || !unplannedExists {
		t.Fatalf("unplanned Event extraction Work Item: exists=%v err=%v", unplannedExists, err)
	}
	if err := store.InitializeArtifactUnits(
		ctx,
		unplanned,
		[]eventfact.ArtifactSummary{{
			ArtifactID:           "sha256:artifact",
			CollectorExecutionID: execution.ID,
			ContentSHA256:        strings.Repeat("b", 64),
		}, {
			ArtifactID:           "sha256:artifact-2",
			CollectorExecutionID: execution.ID,
			ContentSHA256:        strings.Repeat("c", 64),
		}},
		now.Add(2*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	attempt, claimed, err := store.ClaimNextWork(ctx, eventfact.ExtractionSnapshot{
		PromptSHA256: strings.Repeat("c", 64),
		SchemaSHA256: strings.Repeat("d", 64),
		ProviderKey:  "deepseek",
		Model:        "deepseek-chat",
	}, now.Add(3*time.Minute))
	if err != nil || !claimed {
		t.Fatalf("claim Event extraction Work Item: claimed=%v err=%v", claimed, err)
	}
	if attempt.WorkItem.ExtractorAgentVersion != eventfact.AgentVersion ||
		len(attempt.WorkItem.CollectorExecutionIDs) != 1 ||
		attempt.WorkItem.CollectorExecutionIDs[0] != execution.ID ||
		attempt.Unit.ArtifactID != "sha256:artifact" {
		t.Fatalf("Extractor attempt = %#v", attempt)
	}
	extractorExecution, err := store.GetExecution(ctx, attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if extractorExecution.AgentKey != eventfact.AgentKey ||
		extractorExecution.TriggerSource != agentrun.TriggerDependent ||
		extractorExecution.Status != agentrun.StatusRunning {
		t.Fatalf("Extractor Agent Execution = %#v", extractorExecution)
	}
	if err := store.SetExecutionCatalog(
		ctx, attempt.ID, "event-tags:catalog", strings.Repeat("e", 64), now.Add(4*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	journalPayload := []byte(`{"package_id":"immutable-package"}`)
	journalSum := sha256.Sum256(journalPayload)
	journal := eventfact.JournalEntry{
		WorkItemKey: attempt.WorkItem.Key, UnitKey: attempt.Unit.Key, BatchOrdinal: 1,
		PackageID: "immutable-package", Payload: journalPayload,
		PayloadHash: hex.EncodeToString(journalSum[:]),
	}
	secondJournalPayload := []byte(`{"package_id":"immutable-package-2"}`)
	secondJournalSum := sha256.Sum256(secondJournalPayload)
	secondJournal := eventfact.JournalEntry{
		WorkItemKey: attempt.WorkItem.Key, UnitKey: attempt.Unit.Key, BatchOrdinal: 2,
		PackageID: "immutable-package-2", Payload: secondJournalPayload,
		PayloadHash: hex.EncodeToString(secondJournalSum[:]),
	}
	if err := store.CompleteExtraction(
		ctx, attempt,
		eventfact.Result{ExecutionID: attempt.ID, ExtractionModelCalls: 1},
		[]eventfact.JournalEntry{journal, secondJournal}, now.Add(5*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, `
		UPDATE event_publication_journal
		SET payload_bytes = '{"package_id":"mutated"}'
		WHERE work_item_key = $1 AND batch_ordinal = 1
	`, attempt.WorkItem.Key); err == nil {
		t.Fatal("Publication Journal allowed payload mutation")
	}
	deliverable, err := store.ListDeliverableJournals(ctx, now.Add(6*time.Minute))
	if err != nil || len(deliverable) != 2 ||
		string(deliverable[0].Payload) != string(journalPayload) {
		t.Fatalf("deliverable Publication Journal = %#v, err=%v", deliverable, err)
	}
	if claimed, err := store.MarkJournalSending(
		ctx, deliverable[0], now.Add(6*time.Minute),
	); err != nil || !claimed {
		t.Fatal(err)
	}
	if claimed, err := store.MarkJournalSending(
		ctx, deliverable[0], now.Add(6*time.Minute),
	); err != nil || claimed {
		t.Fatalf("duplicate Publication claim: claimed=%v err=%v", claimed, err)
	}
	if err := store.MarkJournalRetry(
		ctx, deliverable[0], "transport_unknown", "response was lost", now.Add(7*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if _, claimed, err := store.ClaimNextWork(ctx, eventfact.ExtractionSnapshot{
		PromptSHA256: strings.Repeat("c", 64),
		SchemaSHA256: strings.Repeat("d", 64),
		ProviderKey:  "deepseek",
		Model:        "deepseek-chat",
	}, now.Add(8*time.Minute)); err != nil || claimed {
		t.Fatalf("Publication retry was reclaimed for extraction: claimed=%v err=%v", claimed, err)
	}
	replayed, err := store.ListDeliverableJournals(ctx, now.Add(8*time.Minute))
	if err != nil || len(replayed) != 2 {
		t.Fatalf("replayed Publication Journal = %#v, err=%v", replayed, err)
	}
	var replayedFirst, replayedSecond eventfact.JournalEntry
	for _, entry := range replayed {
		switch entry.PackageID {
		case journal.PackageID:
			replayedFirst = entry
		case secondJournal.PackageID:
			replayedSecond = entry
		}
	}
	if replayedFirst.PackageID == "" || replayedSecond.PackageID == "" {
		t.Fatalf("replayed journals did not preserve package identity: %#v", replayed)
	}
	if claimed, err := store.MarkJournalSending(
		ctx, replayedFirst, now.Add(8*time.Minute),
	); err != nil || !claimed {
		t.Fatalf("claim replayed Publication Journal: claimed=%v err=%v", claimed, err)
	}
	canonical := eventfact.CanonicalEvent{
		DedupeKey:    "event-fact:" + strings.Repeat("f", 64),
		IdentityHash: strings.Repeat("f", 64),
		CoreFacts:    json.RawMessage(`{"title":"事件","factual_summary":"事实摘要","occurred_at":null,"fact_payload":{"action":"test"}}`),
	}
	if err := store.AcknowledgeJournal(
		ctx, replayedFirst, "receipt-1", []eventfact.CanonicalEvent{canonical},
		now.Add(9*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	var unitStatus string
	if err := database.QueryRow(ctx, `
		SELECT status FROM event_artifact_extraction_units WHERE unit_key = $1
	`, attempt.Unit.Key).Scan(&unitStatus); err != nil {
		t.Fatal(err)
	}
	if unitStatus != "publishing" {
		t.Fatalf("unit status after first of two acknowledgements = %q", unitStatus)
	}
	if claimed, err := store.MarkJournalSending(
		ctx, replayedSecond, now.Add(9*time.Minute),
	); err != nil || !claimed {
		t.Fatalf("claim second Publication Journal: claimed=%v err=%v", claimed, err)
	}
	if err := store.AcknowledgeJournal(
		ctx, replayedSecond, "receipt-2", nil, now.Add(10*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkJournalRetry(
		ctx, deliverable[0], "late_transport_error", "stale worker result", now.Add(11*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	var publicationStatus, workStatus string
	if err := database.QueryRow(ctx, `
		SELECT j.status, w.status
		FROM event_publication_journal j
		JOIN event_extraction_work_items w USING (work_item_key)
		WHERE j.work_item_key = $1 AND j.batch_ordinal = 1
	`, attempt.WorkItem.Key).Scan(&publicationStatus, &workStatus); err != nil {
		t.Fatal(err)
	}
	if publicationStatus != "acknowledged" || workStatus != "pending" {
		t.Fatalf("stale delivery result regressed journal=%q work=%q", publicationStatus, workStatus)
	}
	recalled, err := store.FindCanonicalEvents(ctx, []string{canonical.IdentityHash})
	if err != nil || len(recalled) != 1 ||
		recalled[0].DedupeKey != canonical.DedupeKey {
		t.Fatalf("recalled canonical Event facts = %#v, err=%v", recalled, err)
	}
	secondAttempt, claimed, err := store.ClaimNextWork(ctx, eventfact.ExtractionSnapshot{
		PromptSHA256: strings.Repeat("c", 64),
		SchemaSHA256: strings.Repeat("d", 64),
		ProviderKey:  "deepseek",
		Model:        "deepseek-chat",
	}, now.Add(11*time.Minute))
	if err != nil || !claimed || secondAttempt.Unit.ArtifactID != "sha256:artifact-2" ||
		secondAttempt.Unit.ArtifactOrdinal != 2 {
		t.Fatalf("claim second Artifact Unit: attempt=%#v claimed=%v err=%v", secondAttempt, claimed, err)
	}
	if err := store.CompleteWithoutPublication(
		ctx,
		secondAttempt,
		eventfact.Result{
			ExecutionID: secondAttempt.ID, ExtractionModelCalls: 1, ReviewModelCalls: 2,
			FailureCode:  "event_fact_review_contract_missing_tool_call",
			FailureStage: "review", FailureViolation: "missing_tool_call",
			FunctionCalls: []eventfact.FunctionCallObservation{
				{Stage: "extraction", CallCount: 1, FinishReason: "tool_calls", ArgumentBytes: 128},
				{Stage: "review", CallCount: 2, FinishReason: "stop", Violation: "missing_tool_call"},
			},
		},
		eventfact.WorkRejected,
		now.Add(12*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	var finalWorkStatus, persistedStage, persistedFinishReason string
	var persistedArgumentBytes int
	if err := database.QueryRow(ctx, `
		SELECT w.status,
		       u.extraction_result->>'failure_stage',
		       u.extraction_result->'function_calls'->0->>'finish_reason',
		       (u.extraction_result->'function_calls'->0->>'argument_bytes')::int
		FROM event_extraction_work_items w
		JOIN event_artifact_extraction_units u USING (work_item_key)
		WHERE w.work_item_key = $1 AND u.unit_key = $2
	`, attempt.WorkItem.Key, secondAttempt.Unit.Key).Scan(
		&finalWorkStatus, &persistedStage, &persistedFinishReason, &persistedArgumentBytes,
	); err != nil {
		t.Fatal(err)
	}
	if finalWorkStatus != "partially_published" {
		t.Fatalf("mixed Artifact Unit outcomes produced Work status %q", finalWorkStatus)
	}
	if persistedStage != "review" || persistedFinishReason != "tool_calls" ||
		persistedArgumentBytes != 128 {
		t.Fatalf(
			"persisted Function observation stage=%q finish=%q argument_bytes=%d",
			persistedStage, persistedFinishReason, persistedArgumentBytes,
		)
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

func TestFailPublicationReconciliationAtomicallyReleasesPreparedExecution(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(
		ctx, databaseURL, "publication_failure_test",
	)
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
		IdempotencyKey: "publication-failure", Prompt: "prompt", CreatedAt: now,
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
		ExecutionID: execution.ID, PlanPath: "/tmp/failed-plan.json",
		PlanSHA256: strings.Repeat("c", 64), PreparedAt: now,
	}
	if err := store.PreparePublication(ctx, reference); err != nil {
		t.Fatal(err)
	}
	if err := store.FailPublicationReconciliation(ctx, agentrun.ExecutionFailure{
		ExecutionID:       execution.ID,
		ErrorCode:         "artifact_publication_reconciliation_exhausted",
		ErrorSummary:      "Artifact publication reconciliation exhausted its retry budget",
		StopReason:        "agent_or_tool_limit",
		NotInvokedSummary: "Connector did not complete",
		CompletedAt:       now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	failed, err := store.GetExecution(ctx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != agentrun.StatusFailed ||
		failed.ErrorCode != "artifact_publication_reconciliation_exhausted" ||
		failed.Artifacts["failed_publication_plan"] != reference.PlanPath ||
		failed.Artifacts["failed_publication_sha256"] != reference.PlanSHA256 {
		t.Fatalf("failed execution = %#v", failed)
	}
	references, err := store.ListPreparedPublications(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(references) != 0 {
		t.Fatalf("prepared publications after failure = %#v", references)
	}
	withoutAudit, err := store.ListTerminalExecutionsWithoutArtifacts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(withoutAudit) != 1 || withoutAudit[0].ID != execution.ID {
		t.Fatalf("terminal Executions awaiting audit = %#v", withoutAudit)
	}
	if err := store.AttachTerminalArtifacts(
		ctx,
		execution.ID,
		map[string]string{"summary": "/tmp/terminal-audit.json"},
		now.Add(2*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	withAudit, err := store.GetExecution(ctx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if withAudit.Artifacts["failed_publication_plan"] != reference.PlanPath ||
		withAudit.Artifacts["failed_publication_sha256"] != reference.PlanSHA256 ||
		withAudit.Artifacts["summary"] != "/tmp/terminal-audit.json" {
		t.Fatalf("merged terminal Artifacts = %#v", withAudit.Artifacts)
	}
	if err := store.AttachTerminalArtifacts(
		ctx,
		execution.ID,
		map[string]string{"summary": "/tmp/terminal-audit.json"},
		now.Add(3*time.Minute),
	); err != nil {
		t.Fatalf("idempotent terminal Artifact merge: %v", err)
	}

	uncertain, _, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "publication-uncertain", Prompt: "prompt", CreatedAt: now,
		AgentVersion: "collector.v1", InvocationKeys: []string{"one"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []agentrun.ExecutionStatus{
		agentrun.StatusPlanning, agentrun.StatusCollecting, agentrun.StatusMaterializing,
	} {
		if err := store.SetExecutionStatus(ctx, uncertain.ID, status, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.FailPublicationReconciliation(ctx, agentrun.ExecutionFailure{
		ExecutionID:       uncertain.ID,
		ErrorCode:         "artifact_publication_reconciliation_exhausted",
		ErrorSummary:      "Artifact publication remained non-terminal after reconciliation",
		StopReason:        "agent_or_tool_limit",
		NotInvokedSummary: "Connector did not complete",
		CompletedAt:       now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("uncertain prepare failure: %v", err)
	}
	failedWithoutPublication, err := store.GetExecution(ctx, uncertain.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failedWithoutPublication.Status != agentrun.StatusFailed {
		t.Fatalf("uncertain prepare status = %q", failedWithoutPublication.Status)
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
