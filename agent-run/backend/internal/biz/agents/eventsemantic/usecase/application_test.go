package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

type applicationRepositoryStub struct {
	completions []eventsemantic.ExecutionCompletion
	noWork      bool
	reanalysis  eventsemantic.ReanalysisRequest
}

func (*applicationRepositoryStub) EnsureInitialWorkItems(
	_ context.Context,
	_ []eventsemantic.EligibleEvent,
	_ time.Time,
) error {
	return nil
}

func (s *applicationRepositoryStub) EnqueueReanalysis(
	_ context.Context,
	request eventsemantic.ReanalysisRequest,
	now time.Time,
) (eventsemantic.WorkItem, bool, error) {
	s.reanalysis = request
	return eventsemantic.WorkItem{
		ID: "work-item-1", EventID: request.EventID,
		SupersedesSubmissionID: request.SupersedesSubmissionID,
		Status:                 "pending", CreatedAt: now, UpdatedAt: now,
	}, false, nil
}

func (s *applicationRepositoryStub) StartNextExecution(
	_ context.Context,
	_ string,
	_ string,
	_ time.Time,
) (eventsemantic.ExecutionAttempt, bool, error) {
	if s.noWork {
		return eventsemantic.ExecutionAttempt{}, false, nil
	}
	return eventsemantic.ExecutionAttempt{
		ID: "execution-1",
		WorkItem: eventsemantic.WorkItem{
			ID: "work-item-1", EventID: "event-1", Status: "running",
		},
	}, true, nil
}

func (s *applicationRepositoryStub) CompleteExecution(
	_ context.Context,
	completion eventsemantic.ExecutionCompletion,
) error {
	s.completions = append(s.completions, completion)
	return nil
}

type applicationDataStub struct {
	contextErr error
	noWork     bool
	semantics  eventsemantic.EventSemantics
	leaseCalls int
}

func (s *applicationDataStub) ListEligibleEvents(context.Context, int) ([]eventsemantic.EligibleEvent, error) {
	if s.noWork {
		return nil, nil
	}
	return []eventsemantic.EligibleEvent{{EventID: "event-1"}}, nil
}
func (s *applicationDataStub) CreateContextLease(
	_ context.Context,
	request eventsemantic.ContextLeaseRequest,
) (eventsemantic.ContextLease, error) {
	s.leaseCalls++
	return eventsemantic.ContextLease{
		ContextLeaseID: "lease-1", EventID: request.EventID, Status: "active",
	}, nil
}
func (s *applicationDataStub) Context(context.Context, string) (eventsemantic.Context, error) {
	return eventsemantic.Context{}, s.contextErr
}
func (*applicationDataStub) Resolve(context.Context, string, []eventsemantic.EntityMention) ([]eventsemantic.EntityResolution, error) {
	return nil, nil
}
func (*applicationDataStub) SearchDirectTargets(context.Context, string, string, []string) ([]eventsemantic.DirectTarget, error) {
	return nil, nil
}
func (*applicationDataStub) CreateSubmission(context.Context, eventsemantic.SubmissionRequest) (eventsemantic.SubmissionResult, error) {
	return eventsemantic.SubmissionResult{}, nil
}
func (s *applicationDataStub) GetEventSemantics(context.Context, string) (eventsemantic.EventSemantics, error) {
	return s.semantics, nil
}
func (*applicationDataStub) SubmitReview(context.Context, string, eventsemantic.ReviewRequest) (eventsemantic.SubmissionResult, error) {
	return eventsemantic.SubmissionResult{}, nil
}
func TestTickPersistsFailureAndStillReturnsAnError(t *testing.T) {
	repository := &applicationRepositoryStub{}
	data := &applicationDataStub{contextErr: errors.New("remote context failure")}
	application, err := New(repository, data, func(context.Context) (Runtime, error) {
		t.Fatal("runtime must not be loaded when context acquisition fails")
		return Runtime{}, nil
	}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time {
		return time.Date(2026, 7, 29, 8, 30, 0, 0, time.UTC)
	}

	tickErr := application.Tick(context.Background())

	if tickErr == nil {
		t.Fatal("expected the failed tick to remain observable")
	}
	if len(repository.completions) != 1 {
		t.Fatalf("completions = %#v", repository.completions)
	}
	completion := repository.completions[0]
	if completion.Status != "failed" ||
		completion.ErrorCode != "event_semantic_context_unavailable" ||
		completion.ErrorSummary != "Data Event Semantic Context is unavailable" {
		t.Fatalf("completion = %#v", completion)
	}
}

func TestApplicationStartAndShutdownOwnTheWorkerLifecycle(t *testing.T) {
	repository := &applicationRepositoryStub{noWork: true}
	application, err := New(
		repository,
		&applicationDataStub{noWork: true},
		func(context.Context) (Runtime, error) {
			t.Fatal("runtime must not load without eligible work")
			return Runtime{}, nil
		},
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	application.Start(context.Background())
	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := application.Shutdown(shutdownContext); err != nil {
		t.Fatalf("shutdown worker: %v", err)
	}
}

func TestTickReconcilesTerminalDataSubmissionWithoutRerunningModels(t *testing.T) {
	repository := &applicationRepositoryStub{}
	data := &applicationDataStub{
		semantics: eventsemantic.EventSemantics{
			EventID: "event-1",
			Submissions: []eventsemantic.SubmissionResult{{
				SubmissionID:     "submission-1",
				AgentExecutionID: "execution-1",
				Status:           "accepted",
			}},
		},
	}
	application, err := New(
		repository,
		data,
		func(context.Context) (Runtime, error) {
			t.Fatal("runtime must not load after a terminal Submission is reconciled")
			return Runtime{}, nil
		},
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if data.leaseCalls != 0 || len(repository.completions) != 1 ||
		repository.completions[0].Status != "succeeded" {
		t.Fatalf("leaseCalls=%d completions=%#v", data.leaseCalls, repository.completions)
	}
}

func TestRequestReanalysisCreatesAnAgentRunOwnedWorkItem(t *testing.T) {
	repository := &applicationRepositoryStub{}
	application, err := New(
		repository,
		&applicationDataStub{noWork: true},
		func(context.Context) (Runtime, error) { return Runtime{}, nil },
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	item, replayed, err := application.RequestReanalysis(context.Background(), eventsemantic.ReanalysisRequest{
		EventID:                "88888888-8888-4888-8888-888888888888",
		SupersedesSubmissionID: "66666666-6666-4666-8666-666666666666",
		Reason:                 "ontology_upgrade", IdempotencyKey: "reanalysis-001",
	})
	if err != nil || replayed || item.ID != "work-item-1" {
		t.Fatalf("item=%#v replayed=%v err=%v", item, replayed, err)
	}
	if repository.reanalysis.EventID != item.EventID ||
		repository.reanalysis.SupersedesSubmissionID != item.SupersedesSubmissionID {
		t.Fatalf("request=%#v item=%#v", repository.reanalysis, item)
	}
}

var _ eventsemantic.Repository = (*applicationRepositoryStub)(nil)
var _ eventsemantic.DataClient = (*applicationDataStub)(nil)
