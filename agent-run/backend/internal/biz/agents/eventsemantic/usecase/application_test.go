package usecase

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	semanticworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic/workflow"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

type applicationRepositoryStub struct {
	completions []eventsemantic.ExecutionCompletion
	noWork      bool
	reanalysis  eventsemantic.ReanalysisRequest
	ensured     [][]eventsemantic.EligibleEvent
	insertLater bool
	completeErr error
}

type permittedApplicationRepositoryStub struct {
	*applicationRepositoryStub
	permitCalls int
}

func (s *permittedApplicationRepositoryStub) WithEventSemanticProcessingPermit(
	_ context.Context,
	operation func() error,
) error {
	s.permitCalls++
	return operation()
}

func (s *applicationRepositoryStub) EnsureInitialWorkItems(
	_ context.Context,
	events []eventsemantic.EligibleEvent,
	_ time.Time,
) (int, error) {
	s.ensured = append(s.ensured, append([]eventsemantic.EligibleEvent(nil), events...))
	if s.insertLater && len(events) == 1 && events[0].EventID == "event-later" {
		return 1, nil
	}
	return 0, nil
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
	eventID := "event-1"
	if s.insertLater {
		eventID = "event-later"
	}
	return eventsemantic.ExecutionAttempt{
		ID: "execution-1",
		WorkItem: eventsemantic.WorkItem{
			ID: "work-item-1", EventID: eventID, Status: "running",
		},
	}, true, nil
}

func (s *applicationRepositoryStub) CompleteExecution(
	_ context.Context,
	completion eventsemantic.ExecutionCompletion,
) error {
	s.completions = append(s.completions, completion)
	return s.completeErr
}

type lifecycleLoggerStub struct {
	info  []agentrun.AgentLifecycleEvent
	warn  []agentrun.AgentLifecycleEvent
	error []agentrun.AgentLifecycleEvent
}

func (l *lifecycleLoggerStub) Info(event agentrun.AgentLifecycleEvent) {
	l.info = append(l.info, event)
}
func (l *lifecycleLoggerStub) Warn(event agentrun.AgentLifecycleEvent) {
	l.warn = append(l.warn, event)
}
func (l *lifecycleLoggerStub) Error(event agentrun.AgentLifecycleEvent) {
	l.error = append(l.error, event)
}

type applicationDataStub struct {
	contextErr        error
	noWork            bool
	semantics         eventsemantic.EventSemantics
	leaseCalls        int
	leaseErr          error
	pages             map[string]eventsemantic.EligibleEventPage
	cursors           []string
	contextSnapshot   eventsemantic.Context
	submissionResult  eventsemantic.SubmissionResult
	submissionRequest eventsemantic.SubmissionRequest
}

func (s *applicationDataStub) ListEligibleEvents(
	_ context.Context,
	_ int,
	cursor string,
) (eventsemantic.EligibleEventPage, error) {
	s.cursors = append(s.cursors, cursor)
	if s.pages != nil {
		return s.pages[cursor], nil
	}
	if s.noWork {
		return eventsemantic.EligibleEventPage{}, nil
	}
	return eventsemantic.EligibleEventPage{
		Events: []eventsemantic.EligibleEvent{{EventID: "event-1"}},
	}, nil
}
func (s *applicationDataStub) CreateContextLease(
	_ context.Context,
	request eventsemantic.ContextLeaseRequest,
) (eventsemantic.ContextLease, error) {
	s.leaseCalls++
	if s.leaseErr != nil {
		return eventsemantic.ContextLease{}, s.leaseErr
	}
	return eventsemantic.ContextLease{
		ContextLeaseID: "lease-1", EventID: request.EventID, Status: "active",
		LeaseExpiresAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}, nil
}
func (s *applicationDataStub) Context(context.Context, string) (eventsemantic.Context, error) {
	return s.contextSnapshot, s.contextErr
}
func (s *applicationDataStub) CreateSubmission(
	_ context.Context,
	request eventsemantic.SubmissionRequest,
) (eventsemantic.SubmissionResult, error) {
	s.submissionRequest = request
	return s.submissionResult, nil
}

type queuedSemanticModel struct {
	responses []string
}

type applicationRetrieverStub struct{}

func (applicationRetrieverStub) ExactEntities(_ context.Context, lookups []eventsemantic.EntityLookup) ([]eventsemantic.EntityCandidateSet, error) {
	result := make([]eventsemantic.EntityCandidateSet, 0, len(lookups))
	for _, lookup := range lookups {
		result = append(result, eventsemantic.EntityCandidateSet{CandidateKey: lookup.CandidateKey})
	}
	return result, nil
}

func (applicationRetrieverStub) SearchEntities(_ context.Context, lookups []eventsemantic.EntityLookup, _ int) ([]eventsemantic.EntityCandidateSet, error) {
	result := make([]eventsemantic.EntityCandidateSet, 0, len(lookups))
	for _, lookup := range lookups {
		result = append(result, eventsemantic.EntityCandidateSet{CandidateKey: lookup.CandidateKey})
	}
	return result, nil
}

func (m *queuedSemanticModel) Generate(
	context.Context,
	[]*schema.Message,
	...model.Option,
) (*schema.Message, error) {
	if len(m.responses) == 0 {
		return nil, errors.New("unexpected semantic model call")
	}
	response := m.responses[0]
	m.responses = m.responses[1:]
	return schema.AssistantMessage(response, nil), nil
}

func (*queuedSemanticModel) Stream(
	context.Context,
	[]*schema.Message,
	...model.Option,
) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("stream is not used")
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
		!completion.Retryable ||
		completion.ErrorCode != "event_semantic_context_unavailable" ||
		completion.ErrorSummary != "Data Event Semantic Context is unavailable" {
		t.Fatalf("completion = %#v", completion)
	}
}

func TestTickRejectsContextOwnedByAnotherWorkerBeforeLoadingRuntime(t *testing.T) {
	repository := &applicationRepositoryStub{}
	data := &applicationDataStub{contextSnapshot: eventsemantic.Context{
		ContextLeaseID: "lease-1", AgentExecutionID: "execution-1", WorkerID: "other-worker",
		LeaseExpiresAt: "2026-08-01T00:00:00Z", Event: eventsemantic.Event{ID: "event-1"},
	}}
	application, err := New(repository, data, func(context.Context) (Runtime, error) {
		t.Fatal("runtime must not be loaded for a mismatched Context identity")
		return Runtime{}, nil
	}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.Tick(context.Background()); err == nil {
		t.Fatal("expected mismatched Context identity to fail the execution")
	}
	if len(repository.completions) != 1 ||
		repository.completions[0].ErrorCode != "event_semantic_context_identity_mismatch" ||
		!repository.completions[0].Retryable {
		t.Fatalf("completions = %#v", repository.completions)
	}
}

func TestValidateContextIdentityRejectsEveryMismatchedIdentity(t *testing.T) {
	expiresAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	attempt := eventsemantic.ExecutionAttempt{ID: "execution-1", WorkItem: eventsemantic.WorkItem{
		EventID: "event-1", SupersedesSubmissionID: "submission-1",
	}}
	lease := eventsemantic.ContextLease{
		ContextLeaseID: "lease-1", EventID: "event-1", SupersedesSubmissionID: "submission-1",
		Status: "active", LeaseExpiresAt: expiresAt,
	}
	contextValue := eventsemantic.Context{
		ContextLeaseID: "lease-1", AgentExecutionID: "execution-1", WorkerID: "worker-1",
		LeaseExpiresAt: expiresAt.Format(time.RFC3339Nano), Event: eventsemantic.Event{ID: "event-1"},
	}
	tests := []struct {
		name   string
		mutate func(*eventsemantic.ExecutionAttempt, *eventsemantic.ContextLease, *eventsemantic.Context)
	}{
		{name: "lease status", mutate: func(_ *eventsemantic.ExecutionAttempt, lease *eventsemantic.ContextLease, _ *eventsemantic.Context) {
			lease.Status = "expired"
		}},
		{name: "lease id", mutate: func(_ *eventsemantic.ExecutionAttempt, _ *eventsemantic.ContextLease, value *eventsemantic.Context) {
			value.ContextLeaseID = "lease-2"
		}},
		{name: "agent execution", mutate: func(_ *eventsemantic.ExecutionAttempt, _ *eventsemantic.ContextLease, value *eventsemantic.Context) {
			value.AgentExecutionID = "execution-2"
		}},
		{name: "worker", mutate: func(_ *eventsemantic.ExecutionAttempt, _ *eventsemantic.ContextLease, value *eventsemantic.Context) {
			value.WorkerID = "worker-2"
		}},
		{name: "lease event", mutate: func(_ *eventsemantic.ExecutionAttempt, lease *eventsemantic.ContextLease, _ *eventsemantic.Context) {
			lease.EventID = "event-2"
		}},
		{name: "context event", mutate: func(_ *eventsemantic.ExecutionAttempt, _ *eventsemantic.ContextLease, value *eventsemantic.Context) {
			value.Event.ID = "event-2"
		}},
		{name: "supersedes", mutate: func(_ *eventsemantic.ExecutionAttempt, lease *eventsemantic.ContextLease, _ *eventsemantic.Context) {
			lease.SupersedesSubmissionID = "submission-2"
		}},
		{name: "expiry", mutate: func(_ *eventsemantic.ExecutionAttempt, _ *eventsemantic.ContextLease, value *eventsemantic.Context) {
			value.LeaseExpiresAt = expiresAt.Add(time.Second).Format(time.RFC3339Nano)
		}},
		{name: "malformed expiry", mutate: func(_ *eventsemantic.ExecutionAttempt, _ *eventsemantic.ContextLease, value *eventsemantic.Context) {
			value.LeaseExpiresAt = "invalid"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotAttempt, gotLease, gotContext := attempt, lease, contextValue
			test.mutate(&gotAttempt, &gotLease, &gotContext)
			if err := validateContextIdentity(gotAttempt, gotLease, gotContext, "worker-1"); err == nil {
				t.Fatal("expected identity mismatch")
			}
		})
	}
	if err := validateContextIdentity(attempt, lease, contextValue, "worker-1"); err != nil {
		t.Fatalf("valid identity rejected: %v", err)
	}
}

func TestTickCompletesWithoutRetryWhenEventNoLongerRequiresSemantics(t *testing.T) {
	repository := &applicationRepositoryStub{}
	data := &applicationDataStub{leaseErr: &eventsemantic.RemoteError{
		Status: 409, Code: "EVENT_SEMANTICS_NOT_REQUIRED",
		Summary: "Event no longer requires semantics",
	}}
	application, err := New(
		repository,
		data,
		func(context.Context) (Runtime, error) {
			t.Fatal("runtime must not load when the Event no longer requires semantics")
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
	if len(repository.completions) != 1 ||
		repository.completions[0].Status != "succeeded" {
		t.Fatalf("completions = %#v", repository.completions)
	}
}

func TestTickDoesNotRetryNewSemanticInputInvariantFailure(t *testing.T) {
	repository := &applicationRepositoryStub{}
	data := &applicationDataStub{leaseErr: &eventsemantic.RemoteError{
		Status: 422, Code: "EVENT_SEMANTICS_INPUT_INVALID",
		Summary: "Event Semantic input is invalid",
	}}
	application, err := New(
		repository,
		data,
		func(context.Context) (Runtime, error) { return Runtime{}, nil },
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := application.Tick(context.Background()); err == nil {
		t.Fatal("input invariant failure was hidden")
	}
	if len(repository.completions) != 1 ||
		repository.completions[0].Status != "failed" ||
		repository.completions[0].Retryable {
		t.Fatalf("completions = %#v", repository.completions)
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

func TestTickScansPastKnownFirstPageAndCompletesLaterEvent(t *testing.T) {
	repository := &applicationRepositoryStub{insertLater: true}
	logger := &lifecycleLoggerStub{}
	known := make([]eventsemantic.EligibleEvent, eligibleEventPageSize)
	for index := range known {
		known[index].EventID = "event-known-" + strconv.Itoa(index)
	}
	data := &applicationDataStub{
		contextSnapshot: eventsemantic.Context{
			ContextLeaseID: "lease-1", AgentExecutionID: "execution-1",
			WorkerID: "event-semantic-enricher", LeaseExpiresAt: "2026-08-01T00:00:00Z",
			ManifestContractVersion: "event-semantic-context-manifest.v2",
			Event:                   eventsemantic.Event{ID: "event-later"},
			EntityTypeDefinitions:   []eventsemantic.EntityTypeDefinition{{TypeKey: "company", Status: "active"}},
			VariableDefinitions:     []eventsemantic.VariableDefinition{{Key: "revenue", Version: 1, Status: "active"}},
			MeasurementContract:     eventsemantic.MeasurementContract{Representation: "evidence_grounded_narrative"},
		},
		submissionResult: eventsemantic.SubmissionResult{
			SubmissionID: "submission-later",
			EventID:      "event-later",
			Status:       "rejected",
		},
		pages: map[string]eventsemantic.EligibleEventPage{
			"": {
				Events:     known,
				NextCursor: "next-page",
			},
			"next-page": {
				Events: []eventsemantic.EligibleEvent{{EventID: "event-later"}},
			},
		},
	}
	generator := &queuedSemanticModel{responses: []string{`{"mentions":[],"variable_signals":[]}`}}
	reviewer := &queuedSemanticModel{}
	run, err := semanticworkflow.New(context.Background(), data, applicationRetrieverStub{}, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	application, err := New(
		repository,
		data,
		func(context.Context) (Runtime, error) {
			return Runtime{
				GeneratorModel: "generator", ReviewerModel: "reviewer", Run: run,
			}, nil
		},
		time.Hour,
		WithEventLogger(logger),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := application.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(data.cursors) != 2 || data.cursors[0] != "" || data.cursors[1] != "next-page" {
		t.Fatalf("cursors = %#v", data.cursors)
	}
	if len(repository.ensured) != 2 ||
		repository.ensured[1][0].EventID != "event-later" {
		t.Fatalf("ensured = %#v", repository.ensured)
	}
	if len(repository.completions) != 1 ||
		repository.completions[0].Status != "succeeded" {
		t.Fatalf("completions = %#v", repository.completions)
	}
	if data.submissionRequest.EventID != "event-later" ||
		len(data.submissionRequest.EntityLinks) != 0 ||
		len(generator.responses) != 0 {
		t.Fatalf(
			"submission=%#v remainingGeneratorResponses=%d",
			data.submissionRequest, len(generator.responses),
		)
	}
	if len(logger.info) != 2 ||
		logger.info[1].Code != "agent_execution_completed" ||
		logger.info[1].Counts["events"] != 1 ||
		logger.info[1].Counts["submissions"] != 1 {
		t.Fatalf("lifecycle events = %#v", logger.info)
	}
}

func TestTickResumesBoundedDiscoveryCursorAcrossCycles(t *testing.T) {
	repository := &applicationRepositoryStub{
		insertLater: true,
		noWork:      true,
	}
	data := &applicationDataStub{
		pages: make(map[string]eventsemantic.EligibleEventPage),
	}
	cursor := ""
	for page := 0; page < maxEligibleEventPages; page++ {
		next := "page-" + strconv.Itoa(page+1)
		data.pages[cursor] = eventsemantic.EligibleEventPage{
			Events: []eventsemantic.EligibleEvent{{
				EventID: "event-known-" + strconv.Itoa(page),
			}},
			NextCursor: next,
		}
		cursor = next
	}
	data.pages[cursor] = eventsemantic.EligibleEventPage{
		Events: []eventsemantic.EligibleEvent{{EventID: "event-later"}},
	}
	application, err := New(
		repository,
		data,
		func(context.Context) (Runtime, error) {
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
	if len(data.cursors) != maxEligibleEventPages ||
		data.cursors[0] != "" ||
		data.cursors[len(data.cursors)-1] != "page-99" {
		t.Fatalf("first bounded scan cursors = %#v", data.cursors)
	}
	if err := application.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if data.cursors[len(data.cursors)-1] != "page-100" ||
		len(repository.ensured) != maxEligibleEventPages+1 ||
		repository.ensured[len(repository.ensured)-1][0].EventID != "event-later" {
		t.Fatalf(
			"resumed cursors=%#v ensured=%#v",
			data.cursors, repository.ensured,
		)
	}
}

func TestTickUsesRepositoryProcessingPermit(t *testing.T) {
	repository := &permittedApplicationRepositoryStub{
		applicationRepositoryStub: &applicationRepositoryStub{noWork: true},
	}
	application, err := New(
		repository,
		&applicationDataStub{noWork: true},
		func(context.Context) (Runtime, error) { return Runtime{}, nil },
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.permitCalls != 1 {
		t.Fatalf("processing permit calls = %d", repository.permitCalls)
	}
}

func TestTickDoesNotLogCompletionBeforeDurableStateTransition(t *testing.T) {
	repository := &applicationRepositoryStub{
		completeErr: errors.New("database unavailable"),
	}
	data := &applicationDataStub{
		semantics: eventsemantic.EventSemantics{
			EventID: "event-1",
			Submissions: []eventsemantic.SubmissionResult{{
				SubmissionID: "submission-1", AgentExecutionID: "execution-1",
				Status: "accepted",
			}},
		},
	}
	logger := &lifecycleLoggerStub{}
	application, err := New(
		repository,
		data,
		func(context.Context) (Runtime, error) {
			t.Fatal("runtime must not load after terminal reconciliation")
			return Runtime{}, nil
		},
		time.Hour,
		WithEventLogger(logger),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := application.Tick(context.Background()); err == nil {
		t.Fatal("durable completion failure was hidden")
	}

	if len(logger.info) != 1 ||
		logger.info[0].Code != "agent_execution_started" {
		t.Fatalf("info events = %#v", logger.info)
	}
	if len(logger.warn) != 0 || len(logger.error) != 0 {
		t.Fatalf("warn=%#v error=%#v", logger.warn, logger.error)
	}
}

var _ eventsemantic.Repository = (*applicationRepositoryStub)(nil)
var _ eventsemantic.ProcessingPermit = (*permittedApplicationRepositoryStub)(nil)
var _ eventsemantic.DataClient = (*applicationDataStub)(nil)
