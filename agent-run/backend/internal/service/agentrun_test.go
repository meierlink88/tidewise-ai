package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/agent-run/backend/api/agentrun/v1"
	collectorusecase "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/usecase"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

type eventSemanticUseCaseStub struct {
	request  eventsemantic.ReanalysisRequest
	replayed bool
	err      error
}

func (s *eventSemanticUseCaseStub) RequestReanalysis(
	_ context.Context,
	request eventsemantic.ReanalysisRequest,
) (eventsemantic.WorkItem, bool, error) {
	s.request = request
	return eventsemantic.WorkItem{
		ID:      "11111111-1111-4111-8111-111111111111",
		EventID: request.EventID, SupersedesSubmissionID: request.SupersedesSubmissionID,
		Status: "pending", CreatedAt: time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC),
	}, s.replayed, s.err
}

func TestCollectorErrorMapsBizFailuresWithoutInfrastructureDetails(t *testing.T) {
	for name, test := range map[string]struct {
		input  error
		status int
		code   string
	}{
		"not ready": {
			input: collectorusecase.ErrNotReady, status: http.StatusServiceUnavailable,
			code: "CONFIGURATION_NOT_READY",
		},
		"idempotency": {
			input: agentrun.ErrIdempotencyConflict, status: http.StatusConflict,
			code: "IDEMPOTENCY_CONFLICT",
		},
		"unknown": {
			input: errors.New("database secret detail"), status: http.StatusInternalServerError,
			code: "INTERNAL_ERROR",
		},
	} {
		t.Run(name, func(t *testing.T) {
			public, ok := collectorError(test.input).(*v1.PublicError)
			if !ok || public.Status != test.status || public.Code != test.code {
				t.Fatalf("public error = %#v", public)
			}
			if public.Message == "database secret detail" {
				t.Fatal("infrastructure error leaked through Service")
			}
		})
	}
}

func TestCreateEventSemanticReanalysisDelegatesToAgentRunWorkItemUseCase(t *testing.T) {
	semantic := &eventSemanticUseCaseStub{replayed: true}
	service := &AgentRunService{eventSemantic: semantic}
	result, err := service.CreateEventSemanticReanalysis(
		context.Background(),
		&v1.CreateEventSemanticReanalysisRequest{
			IdempotencyKey:         "semantic-reanalysis-1",
			EventID:                "22222222-2222-4222-8222-222222222222",
			SupersedesSubmissionID: "33333333-3333-4333-8333-333333333333",
			Reason:                 " ontology_upgrade ",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkItemID == "" || !result.Replayed || result.Status != "pending" {
		t.Fatalf("result = %#v", result)
	}
	if semantic.request.Reason != "ontology_upgrade" ||
		semantic.request.IdempotencyKey != "semantic-reanalysis-1" {
		t.Fatalf("request = %#v", semantic.request)
	}
}

func TestCollectorSubmissionResultConvertsBizSnapshotWithoutSharingMaps(t *testing.T) {
	now := time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC)
	input := agentrun.Execution{
		ID: "11111111-1111-4111-8111-111111111111", AgentVersion: "collector.v1",
		Status: agentrun.StatusSucceeded, PromptSHA256: "hash", PromptBytes: 10,
		CandidateCounts: map[string]int{"results_pending": 0},
		Artifacts:       map[string]string{"manifest": "/tmp/manifest.json"},
		CreatedAt:       now,
		Invocations: []agentrun.ConnectorInvocation{{
			ConnectorKey: "tavily", Status: agentrun.InvocationCompleted, ResultCount: 1,
		}},
	}
	result := collectorSubmissionResult(input)
	if result.ExecutionID != input.ID || result.StatusURL != "/api/v1/collector/runs/"+input.ID ||
		len(result.Invocations) != 1 || result.Invocations[0].ConnectorKey != "tavily" {
		t.Fatalf("Collector result = %#v", result)
	}
	result.CandidateCounts["results_pending"] = 1
	result.Artifacts["manifest"] = "changed"
	if input.CandidateCounts["results_pending"] != 0 || input.Artifacts["manifest"] != "/tmp/manifest.json" {
		t.Fatal("Service response shares mutable Biz maps")
	}
}
