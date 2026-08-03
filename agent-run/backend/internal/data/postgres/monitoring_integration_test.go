package postgres_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/testsupport"
)

func TestEventSemanticCompletionFeedsMonitoringProjection(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(
		ctx, databaseURL, "semantic_monitoring_completion_test",
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
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := store.EnsureInitialWorkItems(ctx, []eventsemantic.EligibleEvent{{
		EventID: "11111111-1111-4111-8111-111111111111",
	}}, now); err != nil {
		t.Fatal(err)
	}
	attempt, found, err := store.StartNextExecution(
		ctx, "semantic-worker", strings.Repeat("a", 64), now,
	)
	if err != nil || !found {
		t.Fatalf("start execution found=%v err=%v", found, err)
	}
	if err := store.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
		ExecutionID: attempt.ID, Status: "succeeded",
		CandidateCounts: map[string]int{
			"events": 1, "submissions": 1,
			"accepted_candidates": 2, "rejected_candidates": 1,
		},
		CompletedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	totals, err := store.GetMonitoringBusinessTotals(ctx, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if totals.SemanticSubmissions != 1 || totals.SemanticAcceptedCandidates != 2 ||
		totals.SemanticRejectedCandidates != 1 {
		t.Fatalf("semantic monitoring totals = %+v", totals)
	}
	page, err := store.ListSemanticMonitoring(ctx, agentrun.MonitoringListQuery{
		Since: now.Add(-time.Hour), Statuses: []string{"succeeded"}, Page: 1, PageSize: 20,
	})
	if err != nil || len(page.Items) != 1 || page.Items[0].AcceptedCandidates != 2 ||
		page.Items[0].RejectedCandidates != 1 {
		t.Fatalf("semantic monitoring page = %+v err=%v", page, err)
	}
}

func TestMonitoringProjectionUsesExistingAgentRunFacts(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "monitoring_projection_test")
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

	now := time.Now().UTC().Truncate(time.Second)
	collectorID := "11111111-1111-4111-8111-111111111111"
	extractorID := "22222222-2222-4222-8222-222222222222"
	semanticID := "33333333-3333-4333-8333-333333333333"
	for _, execution := range []struct {
		id, version, agentKey, idempotency, status, counts string
	}{
		{collectorID, "collector.v1", "collector", "monitoring-collector", "succeeded", `{"raw_results":9,"merged_results":7,"accepted":5}`},
		{extractorID, "event-fact-extractor.v2", "event-fact-extractor", "monitoring-extractor", "succeeded", `{}`},
		{semanticID, "event-semantic-enricher.v3", "event-semantic-enricher", "monitoring-semantic", "succeeded", `{"submissions":1,"accepted_candidates":4,"rejected_candidates":1}`},
	} {
		prompt, promptHash, promptBytes := any(nil), any(nil), any(nil)
		if execution.agentKey == "collector" {
			prompt, promptHash, promptBytes = "monitoring", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 10
		}
		_, err = database.Exec(ctx, `
			INSERT INTO agent_executions (
				execution_id, agent_version, idempotency_key, prompt, prompt_sha256, prompt_bytes,
				status, candidate_counts, artifacts, created_at, started_at, completed_at, updated_at,
				agent_key, input_payload, trigger_source, triggered_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,'{}'::jsonb,$9,$9,$10,$10,$11,'{}'::jsonb,$12,$9)`,
			execution.id, execution.version, execution.idempotency, prompt, promptHash, promptBytes,
			execution.status, execution.counts, now.Add(-2*time.Minute), now.Add(-time.Minute),
			execution.agentKey, map[bool]string{true: "api", false: "dependent"}[execution.agentKey == "collector"],
		)
		if err != nil {
			t.Fatalf("insert %s execution: %v", execution.agentKey, err)
		}
	}

	workKey := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	unitKey := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	_, err = database.Exec(ctx, `
		INSERT INTO event_extraction_work_items (
			work_item_key, collector_execution_ids, extractor_agent_version, status,
			current_execution_id, created_at, updated_at
		) VALUES ($1, ARRAY[$2::uuid], 'event-fact-extractor.v2', 'published', $3, $4, $5)`,
		workKey, collectorID, extractorID, now.Add(-2*time.Hour), now.Add(-time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `
		INSERT INTO event_artifact_extraction_units (
			unit_key, work_item_key, artifact_ordinal, artifact_id, collector_execution_id,
			content_sha256, status, current_execution_id, extraction_result, created_at, updated_at
		) VALUES ($1,$2,1,'artifact-1',$3,$4,'published',$5,
			'{"candidates":[{"review_state":"auto_approved"}]}'::jsonb,$6,$7)`,
		unitKey, workKey, collectorID,
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		extractorID, now.Add(-2*time.Hour), now.Add(-time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `
		INSERT INTO event_semantic_work_items (
			work_item_id, event_id, trigger_source, idempotency_key, status, attempt_count,
			max_attempts, current_execution_id, created_at, updated_at
		) VALUES ('44444444-4444-4444-8444-444444444444','55555555-5555-4555-8555-555555555555',
			'eligible_event','monitoring-work-item','succeeded',1,2,$1,$2,$3)`,
		semanticID, now.Add(-2*time.Hour), now.Add(-time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}

	store := postgres.New(database)
	counts, err := store.ListMonitoringStatusCounts(ctx, now.Add(-time.Hour))
	if err != nil || len(counts) != 3 {
		t.Fatalf("status counts = %#v, err = %v", counts, err)
	}
	totals, err := store.GetMonitoringBusinessTotals(ctx, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if totals.CollectorRawResults != 9 || totals.CollectorAcceptedArtifacts != 5 ||
		totals.ArtifactPublished != 1 || totals.ArtifactFormalEvents != 1 ||
		totals.SemanticSubmissions != 1 || totals.SemanticAcceptedCandidates != 4 {
		t.Fatalf("business totals = %+v", totals)
	}

	collector, err := store.ListCollectorMonitoring(ctx, agentrun.MonitoringListQuery{
		Since: now.Add(-time.Hour), Statuses: []string{"succeeded"}, Page: 1, PageSize: 20,
	})
	if err != nil || len(collector.Items) != 1 || collector.Items[0].AcceptedArtifacts != 5 {
		t.Fatalf("collector page = %+v, err = %v", collector, err)
	}
	artifacts, err := store.ListArtifactExtractionMonitoring(ctx, agentrun.MonitoringListQuery{
		Since: now.Add(-time.Hour), Statuses: []string{"published"}, Page: 1, PageSize: 20,
	})
	if err != nil || len(artifacts.Items) != 1 || artifacts.Items[0].EventCandidates != 1 {
		t.Fatalf("artifact page = %+v, err = %v", artifacts, err)
	}
	semantic, err := store.ListSemanticMonitoring(ctx, agentrun.MonitoringListQuery{
		Since: now.Add(-time.Hour), Statuses: []string{"succeeded"}, Page: 1, PageSize: 20,
	})
	if err != nil || len(semantic.Items) != 1 || semantic.Items[0].AcceptedCandidates != 4 {
		t.Fatalf("semantic page = %+v, err = %v", semantic, err)
	}

	secondCollectorID := "66666666-6666-4666-8666-666666666666"
	_, err = database.Exec(ctx, `
		INSERT INTO agent_executions (
			execution_id, agent_version, idempotency_key, prompt, prompt_sha256, prompt_bytes,
			status, candidate_counts, artifacts, created_at, started_at, completed_at, updated_at,
			agent_key, input_payload, trigger_source, triggered_at
		) VALUES ($1,'collector.v1','monitoring-collector-2','monitoring',$2,10,
			'succeeded','{}'::jsonb,'{}'::jsonb,$3,$3,$4,$4,
			'collector','{}'::jsonb,'api',$3)`,
		secondCollectorID, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		now.Add(-2*time.Minute), now.Add(-time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	firstPage, err := store.ListCollectorMonitoring(ctx, agentrun.MonitoringListQuery{
		Since: now.Add(-time.Hour), Statuses: []string{"succeeded"}, Page: 1, PageSize: 1,
	})
	if err != nil || firstPage.TotalItems != 2 || firstPage.TotalPages != 2 ||
		len(firstPage.Items) != 1 || firstPage.Items[0].ExecutionID != secondCollectorID {
		t.Fatalf("stable first page = %+v, err = %v", firstPage, err)
	}
	secondPage, err := store.ListCollectorMonitoring(ctx, agentrun.MonitoringListQuery{
		Since: now.Add(-time.Hour), Statuses: []string{"succeeded"}, Page: 2, PageSize: 1,
	})
	if err != nil || len(secondPage.Items) != 1 || secondPage.Items[0].ExecutionID != collectorID {
		t.Fatalf("stable second page = %+v, err = %v", secondPage, err)
	}
}
