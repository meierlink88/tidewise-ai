package usecase

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	semanticworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic/workflow"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

const (
	eligibleEventPageSize = 20
	maxEligibleEventPages = 100
)

type Runtime struct {
	GeneratorModel string
	ReviewerModel  string
	Run            compose.Runnable[*semanticworkflow.Input, *eventsemantic.Result]
}

type RuntimeProvider func(context.Context) (Runtime, error)

type EventLogger = agentrun.AgentLifecycleLogger

type Application struct {
	repository      eventsemantic.Repository
	data            eventsemantic.DataClient
	runtime         RuntimeProvider
	interval        time.Duration
	now             func() time.Time
	workerID        string
	notify          chan struct{}
	cancel          context.CancelFunc
	done            chan struct{}
	startOnce       sync.Once
	stopOnce        sync.Once
	discoveryMu     sync.Mutex
	discoveryCursor string
	logger          EventLogger
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
		done: make(chan struct{}), logger: agentrun.DiscardAgentLifecycleLogger{},
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
		a.logger.Info(agentrun.AgentLifecycleEvent{
			Code: "agent_runtime_started", AgentKey: eventsemantic.AgentKey,
			AgentVersion: eventsemantic.AgentVersion, RuntimeMode: "worker",
			Status: "running",
		})
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
		a.logger.Info(agentrun.AgentLifecycleEvent{
			Code: "agent_runtime_stopped", AgentKey: eventsemantic.AgentKey,
			AgentVersion: eventsemantic.AgentVersion, RuntimeMode: "worker",
			Status: "stopped",
		})
		return nil
	case <-ctx.Done():
		a.logger.Error(agentrun.AgentLifecycleEvent{
			Code: "agent_runtime_failed", AgentKey: eventsemantic.AgentKey,
			AgentVersion: eventsemantic.AgentVersion, RuntimeMode: "worker",
			Status: "failed", Stage: "shutdown", ErrorCode: "shutdown_deadline",
		})
		return ctx.Err()
	}
}

func (a *Application) loop(ctx context.Context) {
	defer close(a.done)
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	if err := a.Tick(ctx); err != nil {
		a.logCycleFailure()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-a.notify:
		}
		if err := a.Tick(ctx); err != nil {
			a.logCycleFailure()
		}
	}
}

func (a *Application) Tick(ctx context.Context) error {
	if permit, ok := a.repository.(eventsemantic.ProcessingPermit); ok {
		return permit.WithEventSemanticProcessingPermit(ctx, func() error {
			return a.tick(ctx)
		})
	}
	return a.tick(ctx)
}

func (a *Application) tick(ctx context.Context) error {
	now := a.now().UTC()
	attempt, found, err := a.repository.StartNextExecution(
		ctx, a.workerID, semanticworkflow.WorkflowHash(), now,
	)
	if err != nil {
		return err
	}
	if !found {
		if err := a.discoverInitialWork(ctx, now); err != nil {
			return err
		}
		attempt, found, err = a.repository.StartNextExecution(
			ctx, a.workerID, semanticworkflow.WorkflowHash(), now,
		)
		if err != nil || !found {
			return err
		}
	}
	startedAt := a.now().UTC()
	a.logger.Info(a.lifecycleEvent(
		"agent_execution_started", attempt, startedAt, "running", "", "", "",
	))
	fail := func(code, summary string, retryable bool) error {
		failure := errors.New(code)
		completionErr := a.repository.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
			ExecutionID: attempt.ID, Status: "failed", ErrorCode: code,
			ErrorSummary: summary, Retryable: retryable, CompletedAt: a.now().UTC(),
		})
		if completionErr == nil {
			eventCode := "agent_execution_failed"
			if retryable && attempt.WorkItem.AttemptCount < attempt.WorkItem.MaxAttempts {
				eventCode = "agent_execution_retry_scheduled"
				a.logger.Warn(a.lifecycleEvent(
					eventCode, attempt, startedAt, "failed", "retry_scheduled",
					code, "execution",
				))
			} else {
				a.logger.Error(a.lifecycleEvent(
					eventCode, attempt, startedAt, "failed", "terminal_failure",
					code, "execution",
				))
			}
			a.Notify()
		}
		return errors.Join(failure, completionErr)
	}
	existing, err := a.data.GetEventSemantics(ctx, attempt.WorkItem.EventID)
	if err != nil {
		return fail(
			"event_semantic_reconciliation_unavailable",
			"Data Event Semantic reconciliation is unavailable",
			retryableSemanticFailure(err, true),
		)
	}
	var resumableSubmission *eventsemantic.SubmissionResult
	for index := range existing.Submissions {
		submission := existing.Submissions[index]
		if submission.AgentExecutionID == attempt.ID &&
			isTerminalSubmissionStatus(submission.Status) {
			return a.completeSuccess(
				ctx, attempt, startedAt, "submission_reconciled", submission.Status,
				semanticSubmissionCounts(submission),
			)
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
		if semanticRemoteCode(err) == "EVENT_SEMANTICS_NOT_REQUIRED" {
			return a.completeSuccess(
				ctx, attempt, startedAt, "not_required", "succeeded",
				map[string]int{
					"events": 1, "submissions": 0,
					"accepted_candidates": 0, "rejected_candidates": 0,
				},
			)
		}
		return fail(
			"event_semantic_context_lease_unavailable",
			"Data Event Semantic Context Lease is unavailable",
			retryableSemanticFailure(err, true),
		)
	}
	attempt.ContextLease = contextLease
	contextSnapshot, err := a.data.Context(ctx, contextLease.ContextLeaseID)
	if err != nil {
		return fail(
			"event_semantic_context_unavailable",
			"Data Event Semantic Context is unavailable",
			retryableSemanticFailure(err, true),
		)
	}
	if err := validateContextIdentity(attempt, contextLease, contextSnapshot, a.workerID); err != nil {
		return fail(
			"event_semantic_context_identity_mismatch",
			"Data Event Semantic Context identity does not match the current Attempt, Worker and Lease",
			true,
		)
	}
	attempt.Context = contextSnapshot
	runtime, err := a.runtime(ctx)
	if err != nil || runtime.Run == nil || runtime.GeneratorModel == "" || runtime.ReviewerModel == "" {
		return fail(
			"event_semantic_runtime_unavailable",
			"Event Semantic runtime is unavailable",
			true,
		)
	}
	audit := eventsemantic.StageAudit{
		ContractVersion: "event-semantic-stage-audit.v1",
		EventID:         contextSnapshot.Event.ID,
	}
	result, err := runtime.Run.Invoke(ctx, &semanticworkflow.Input{
		Attempt: attempt, Context: contextSnapshot, ExistingSubmission: resumableSubmission,
		GeneratorModel: runtime.GeneratorModel, ReviewerModel: runtime.ReviewerModel, Audit: &audit,
	})
	if err != nil {
		audit.ExecutionFailure = executionFailureAudit(err)
	}
	if auditErr := a.repository.SaveStageAudit(ctx, attempt.ID, audit); auditErr != nil {
		return fail(
			"event_semantic_audit_persistence_failed",
			"Event Semantic stage audit could not be persisted",
			true,
		)
	}
	if err != nil {
		code := "event_semantic_workflow_failed"
		if errors.Is(err, eventsemantic.ErrModelUnavailable) {
			code = "event_semantic_model_unavailable"
		} else if remoteCode := semanticRemoteCode(err); remoteCode != "" {
			code = remoteCode
		}
		return fail(
			code,
			"Event Semantic workflow did not complete",
			retryableSemanticFailure(err, true),
		)
	}
	if result == nil || result.SubmissionID == "" {
		return fail(
			"event_semantic_result_invalid",
			"Event Semantic workflow returned an invalid result",
			false,
		)
	}
	return a.completeSuccess(
		ctx, attempt, startedAt, "processed", result.Status,
		map[string]int{
			"events": 1, "submissions": 1,
			"accepted_candidates": result.AcceptedCandidates,
			"rejected_candidates": result.RejectedCandidates,
		},
	)
}

func executionFailureAudit(err error) *eventsemantic.ExecutionFailureAudit {
	if err == nil {
		return nil
	}
	code := semanticRemoteCode(err)
	owner := "transport"
	if errors.Is(err, eventsemantic.ErrModelUnavailable) || code == "event_semantic_model_contract_invalid" {
		owner = "model"
	} else if code == "qdrant_response_invalid" {
		owner = "retrieval_contract"
	} else if code != "" {
		owner = "data_contract"
	}
	if code == "" {
		code = "event_semantic_workflow_failed"
	}
	return &eventsemantic.ExecutionFailureAudit{ReasonCode: code, Owner: owner}
}

func validateContextIdentity(
	attempt eventsemantic.ExecutionAttempt,
	lease eventsemantic.ContextLease,
	contextValue eventsemantic.Context,
	workerID string,
) error {
	expiresAt, err := time.Parse(time.RFC3339Nano, contextValue.LeaseExpiresAt)
	if err != nil {
		return errors.New("Context lease expiry is invalid")
	}
	if lease.Status != "active" || contextValue.ContextLeaseID != lease.ContextLeaseID ||
		contextValue.AgentExecutionID != attempt.ID || contextValue.WorkerID != workerID ||
		lease.EventID != attempt.WorkItem.EventID || contextValue.Event.ID != attempt.WorkItem.EventID ||
		lease.SupersedesSubmissionID != attempt.WorkItem.SupersedesSubmissionID ||
		!expiresAt.Equal(lease.LeaseExpiresAt) {
		return errors.New("Context identity mismatch")
	}
	return nil
}

func (a *Application) completeSuccess(
	ctx context.Context,
	attempt eventsemantic.ExecutionAttempt,
	startedAt time.Time,
	outcome string,
	status string,
	counts map[string]int,
) error {
	err := a.repository.CompleteExecution(ctx, eventsemantic.ExecutionCompletion{
		ExecutionID: attempt.ID, Status: "succeeded", CompletedAt: a.now().UTC(),
	})
	if err == nil {
		a.logger.Info(a.lifecycleEvent(
			"agent_execution_completed", attempt, startedAt, status, outcome, "", "",
			counts,
		))
		a.Notify()
	}
	return err
}

func (a *Application) lifecycleEvent(
	code string,
	attempt eventsemantic.ExecutionAttempt,
	startedAt time.Time,
	status string,
	outcome string,
	errorCode string,
	stage string,
	counts ...map[string]int,
) agentrun.AgentLifecycleEvent {
	duration := a.now().UTC().Sub(startedAt)
	if duration < 0 {
		duration = 0
	}
	event := agentrun.AgentLifecycleEvent{
		Code: code, AgentKey: eventsemantic.AgentKey,
		AgentVersion: eventsemantic.AgentVersion, RuntimeMode: "worker",
		ExecutionID: attempt.ID, WorkItemID: attempt.WorkItem.ID,
		TriggerSource: attempt.WorkItem.TriggerSource,
		Status:        status, Outcome: outcome, Stage: stage, ErrorCode: errorCode,
		Attempt:     attempt.WorkItem.AttemptCount,
		MaxAttempts: attempt.WorkItem.MaxAttempts, Duration: duration,
	}
	if len(counts) > 0 {
		event.Counts = counts[0]
	}
	return event
}

func (a *Application) logCycleFailure() {
	a.logger.Error(agentrun.AgentLifecycleEvent{
		Code: "agent_cycle_failed", AgentKey: eventsemantic.AgentKey,
		AgentVersion: eventsemantic.AgentVersion, RuntimeMode: "worker",
		Status: "failed", Stage: "tick", ErrorCode: "event_semantic_tick_failed",
	})
}

func semanticRemoteCode(err error) string {
	var remote *eventsemantic.RemoteError
	if errors.As(err, &remote) {
		return remote.Code
	}
	return ""
}

func retryableSemanticFailure(err error, defaultValue bool) bool {
	var remote *eventsemantic.RemoteError
	if errors.As(err, &remote) {
		return remote.Retryable
	}
	return defaultValue
}

func (a *Application) discoverInitialWork(ctx context.Context, now time.Time) error {
	a.discoveryMu.Lock()
	defer a.discoveryMu.Unlock()
	cursor := a.discoveryCursor
	seen := make(map[string]struct{}, maxEligibleEventPages)
	for pageNumber := 0; pageNumber < maxEligibleEventPages; pageNumber++ {
		page, err := a.data.ListEligibleEvents(
			ctx, eligibleEventPageSize, cursor,
		)
		if err != nil {
			return err
		}
		created, err := a.repository.EnsureInitialWorkItems(ctx, page.Events, now)
		if err != nil {
			return err
		}
		if created > 0 || page.NextCursor == "" {
			a.discoveryCursor = ""
			return nil
		}
		if page.NextCursor == cursor {
			return errors.New("eligible Event pagination did not advance")
		}
		if _, exists := seen[page.NextCursor]; exists {
			return errors.New("eligible Event pagination repeated a cursor")
		}
		seen[page.NextCursor] = struct{}{}
		cursor = page.NextCursor
		a.discoveryCursor = cursor
	}
	return nil
}

func isTerminalSubmissionStatus(status string) bool {
	switch status {
	case "accepted", "rejected", "quarantined", "superseded":
		return true
	default:
		return false
	}
}

func semanticSubmissionCounts(
	submission eventsemantic.SubmissionResult,
) map[string]int {
	accepted, rejected := submission.CandidateOutcomeCounts()
	return map[string]int{
		"events": 1, "submissions": 1,
		"accepted_candidates": accepted, "rejected_candidates": rejected,
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
