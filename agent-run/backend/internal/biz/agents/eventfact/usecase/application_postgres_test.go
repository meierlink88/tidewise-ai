package usecase

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
	eventworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact/workflow"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/testsupport"
)

func TestPostgresRestartReplaysUnknownPublicationWithoutExtractionClaim(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(
		ctx, databaseURL, "event_fact_restart_test",
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
	collector, _, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "event-fact-restart-collector",
		InputPayload:   json.RawMessage(`{"prompt":"collect"}`),
		Prompt:         "collect",
		CreatedAt:      now,
		TriggeredAt:    now,
		TriggerSource:  agentrun.TriggerAPI,
		AgentVersion:   "collector.v1",
		InvocationKeys: []string{"one"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []agentrun.ExecutionStatus{
		agentrun.StatusPlanning, agentrun.StatusCollecting, agentrun.StatusMaterializing,
	} {
		if err := store.SetExecutionStatus(ctx, collector.ID, status, now); err != nil {
			t.Fatal(err)
		}
	}
	reference := agentrun.PublicationReference{
		ExecutionID: collector.ID,
		PlanPath:    "/tmp/event-fact-restart-plan.json",
		PlanSHA256:  strings.Repeat("a", 64),
		PreparedAt:  now,
	}
	if err := store.PreparePublication(ctx, reference); err != nil {
		t.Fatal(err)
	}
	if err := store.CommitPreparedPublication(ctx, reference, agentrun.ExecutionCompletion{
		ExecutionID: collector.ID,
		Status:      agentrun.StatusSucceeded,
		StopReason:  "connectors_completed",
		CandidateCounts: map[string]int{
			"accepted": 1, "results_pending": 0,
		},
		Artifacts:   map[string]string{"manifest": "runs/" + collector.ID + "/manifest.json"},
		CompletedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}

	data := &lossThenSuccessData{}
	modelCalls := 0
	runtime := func(context.Context) (Runtime, error) {
		return Runtime{
			Snapshot: eventFactTestSnapshot(),
			ReadArtifacts: func(context.Context, []string) ([]eventfact.Artifact, error) {
				result := approvedResult()
				result.PublicationArtifacts[0].CollectorExecutionID = collector.ID
				return result.PublicationArtifacts, nil
			},
			Run: func(_ context.Context, input *eventworkflow.Input) (*eventfact.Result, error) {
				modelCalls++
				result := approvedResult()
				result.ExecutionID = input.Attempt.ID
				result.PublicationArtifacts[0].CollectorExecutionID = collector.ID
				result.Artifacts = []eventfact.ArtifactSummary{{
					ArtifactID:           result.PublicationArtifacts[0].ArtifactID,
					CollectorExecutionID: collector.ID,
					ContentSHA256:        result.PublicationArtifacts[0].ContentSHA256,
				}}
				return result, nil
			},
		}, nil
	}
	first, err := New(store, data, runtime, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if modelCalls != 1 || len(data.published) != 1 {
		t.Fatalf("first process modelCalls=%d publications=%d", modelCalls, len(data.published))
	}

	restarted, err := New(postgres.New(database), data, runtime, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if modelCalls != 1 || len(data.published) != 2 {
		t.Fatalf("restarted process modelCalls=%d publications=%d", modelCalls, len(data.published))
	}
	if string(data.published[0]) != string(data.published[1]) {
		t.Fatal("PostgreSQL replay changed immutable publication bytes")
	}
}

func eventFactTestSnapshot() eventfact.ExtractionSnapshot {
	return eventfact.ExtractionSnapshot{
		PromptSHA256: strings.Repeat("c", 64),
		SchemaSHA256: strings.Repeat("d", 64),
		ProviderKey:  "deepseek",
		Model:        "deepseek-chat",
	}
}
