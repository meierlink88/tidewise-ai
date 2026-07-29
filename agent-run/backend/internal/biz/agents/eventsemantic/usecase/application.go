package usecase

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	semanticworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic/workflow"
)

type Runtime struct {
	GeneratorModel string
	ReviewerModel  string
	Run            compose.Runnable[*semanticworkflow.Input, *eventsemantic.Result]
}

type RuntimeProvider func(context.Context) (Runtime, error)

type EventLogger interface {
	Error(string)
}

type Application struct {
	repository eventsemantic.Repository
	data       eventsemantic.DataClient
	runtime    RuntimeProvider
	interval   time.Duration
	now        func() time.Time
	workerID   string
	notify     chan struct{}
	cancel     context.CancelFunc
	done       chan struct{}
	startOnce  sync.Once
	stopOnce   sync.Once
	logger     EventLogger
}

type Option func(*Application)

func WithEventLogger(logger EventLogger) Option {
	return func(application *Application) {
		if logger != nil {
			application.logger = logger
		}
	}
}

func New(
	repository eventsemantic.Repository,
	data eventsemantic.DataClient,
	runtime RuntimeProvider,
	interval time.Duration,
	options ...Option,
) (*Application, error) {
	if repository == nil || data == nil || runtime == nil || interval <= 0 {
		return nil, errors.New("Event Semantic Enricher dependencies are required")
	}
	application := &Application{
		repository: repository, data: data, runtime: runtime, interval: interval,
		now: time.Now, workerID: "event-semantic-enricher", notify: make(chan struct{}, 1),
		done: make(chan struct{}), logger: discardLogger{},
	}
	for _, option := range options {
		if option != nil {
			option(application)
		}
	}
	return application, nil
}

func (a *Application) Start(parent context.Context) {
	a.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		a.cancel = cancel
		go a.loop(ctx)
	})
}

func (a *Application) Notify() {
	select {
	case a.notify <- struct{}{}:
	default:
	}
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.stopOnce.Do(func() {
		if a.cancel != nil {
			a.cancel()
		}
	})
	select {
	case <-a.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *Application) loop(ctx context.Context) {
	defer close(a.done)
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	if err := a.Tick(ctx); err != nil {
		a.logger.Error("event_semantic_tick_failed")
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-a.notify:
		}
		if err := a.Tick(ctx); err != nil {
			a.logger.Error("event_semantic_tick_failed")
		}
	}
}

func (a *Application) Tick(ctx context.Context) error {
	eligible, err := a.data.ListEligibleEvents(ctx, 20)
	if err != nil {
		return err
	}
	now := a.now().UTC()
	if err := a.repository.EnsureInitialWorkItems(ctx, eligible, now); err != nil {
		return err
	}
	attempt, found, err := a.repository.StartNextExecution(
		ctx, a.workerID, semanticworkflow.WorkflowHash(), now,
	)
	if err != nil || !found {
		return err
	}
	fail := func(code, summary string) error {
		failure := errors.New(code)
		completionErr := a.repository.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
			ExecutionID: attempt.ID, Status: "failed", ErrorCode: code,
			ErrorSummary: summary, CompletedAt: a.now().UTC(),
		})
		return errors.Join(failure, completionErr)
	}
	existing, err := a.data.GetEventSemantics(ctx, attempt.WorkItem.EventID)
	if err != nil {
		return fail(
			"event_semantic_reconciliation_unavailable",
			"Data Event Semantic reconciliation is unavailable",
		)
	}
	var resumableSubmission *eventsemantic.SubmissionResult
	for index := range existing.Submissions {
		submission := existing.Submissions[index]
		if submission.AgentExecutionID == attempt.ID &&
			isTerminalSubmissionStatus(submission.Status) {
			return a.repository.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
				ExecutionID: attempt.ID, Status: "succeeded", CompletedAt: a.now().UTC(),
			})
		}
		if submission.AgentExecutionID == attempt.ID &&
			(submission.Status == "pending_review" || submission.Status == "needs_reanalysis") &&
			submission.ReviewerWorkPackage != nil {
			copy := submission
			resumableSubmission = &copy
		}
	}
	contextLease, err := a.data.CreateContextLease(ctx, eventsemantic.ContextLeaseRequest{
		EventID:                attempt.WorkItem.EventID,
		SupersedesSubmissionID: attempt.WorkItem.SupersedesSubmissionID,
		AgentExecutionID:       attempt.ID,
		WorkerID:               a.workerID, LeaseSeconds: 15 * 60,
	})
	if err != nil {
		return fail("event_semantic_context_lease_unavailable", "Data Event Semantic Context Lease is unavailable")
	}
	attempt.ContextLease = contextLease
	contextSnapshot, err := a.data.Context(ctx, contextLease.ContextLeaseID)
	if err != nil {
		return fail("event_semantic_context_unavailable", "Data Event Semantic Context is unavailable")
	}
	attempt.Context = contextSnapshot
	runtime, err := a.runtime(ctx)
	if err != nil || runtime.Run == nil || runtime.GeneratorModel == "" || runtime.ReviewerModel == "" {
		return fail("event_semantic_runtime_unavailable", "Event Semantic runtime is unavailable")
	}
	result, err := runtime.Run.Invoke(ctx, &semanticworkflow.Input{
		Attempt: attempt, Context: contextSnapshot, ExistingSubmission: resumableSubmission,
		GeneratorModel: runtime.GeneratorModel, ReviewerModel: runtime.ReviewerModel,
	})
	if err != nil {
		code := "event_semantic_workflow_failed"
		if errors.Is(err, eventsemantic.ErrModelUnavailable) {
			code = "event_semantic_model_unavailable"
		}
		return fail(code, "Event Semantic workflow did not complete")
	}
	if result == nil || result.SubmissionID == "" {
		return fail("event_semantic_result_invalid", "Event Semantic workflow returned an invalid result")
	}
	return a.repository.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
		ExecutionID: attempt.ID, Status: "succeeded", CompletedAt: a.now().UTC(),
	})
}

func isTerminalSubmissionStatus(status string) bool {
	switch status {
	case "accepted", "rejected", "quarantined", "superseded":
		return true
	default:
		return false
	}
}

func (a *Application) RequestReanalysis(
	ctx context.Context,
	request eventsemantic.ReanalysisRequest,
) (eventsemantic.WorkItem, bool, error) {
	item, replayed, err := a.repository.EnqueueReanalysis(ctx, request, a.now().UTC())
	if err == nil {
		a.Notify()
	}
	return item, replayed, err
}

type discardLogger struct{}

func (discardLogger) Error(string) {}
