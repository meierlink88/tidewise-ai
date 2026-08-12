package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact/publication"
	eventworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact/workflow"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

type RunWorkflow func(context.Context, *eventworkflow.Input) (*eventfact.Result, error)
type ExtractFacts func(context.Context, *eventfact.ExecutionAttempt) (*eventfact.Result, error)

type Runtime struct {
	Snapshot      eventfact.ExtractionSnapshot
	Run           RunWorkflow
	ExtractFacts  ExtractFacts
	ReadArtifacts func(context.Context, []string) ([]eventfact.Artifact, error)
}

type RuntimeProvider func(context.Context) (Runtime, error)

type EventLogger = agentrun.AgentLifecycleLogger

type Option func(*Application)

type Application struct {
	repository eventfact.Repository
	data       eventfact.DataClient
	runtime    RuntimeProvider
	interval   time.Duration
	now        func() time.Time
	notify     chan struct{}
	cancel     context.CancelFunc
	done       chan struct{}
	startOnce  sync.Once
	stopOnce   sync.Once
	logger     EventLogger
}

func New(
	repository eventfact.Repository,
	data eventfact.DataClient,
	runtime RuntimeProvider,
	interval time.Duration,
	options ...Option,
) (*Application, error) {
	if repository == nil || data == nil || runtime == nil {
		return nil, errors.New("Event Fact Extractor dependencies are required")
	}
	if interval <= 0 {
		return nil, errors.New("Event Fact Extractor runtime configuration is invalid")
	}
	application := &Application{
		repository: repository, data: data, runtime: runtime,
		interval: interval, now: time.Now, notify: make(chan struct{}, 1), done: make(chan struct{}),
		logger: agentrun.DiscardAgentLifecycleLogger{},
	}
	for _, option := range options {
		if option != nil {
			option(application)
		}
	}
	return application, nil
}

func WithEventLogger(logger EventLogger) Option {
	return func(application *Application) {
		if logger != nil {
			application.logger = logger
		}
	}
}

func (a *Application) Start(parent context.Context) {
	a.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		a.cancel = cancel
		a.logger.Info(agentrun.AgentLifecycleEvent{
			Code: "agent_runtime_started", AgentKey: eventfact.AgentKey,
			AgentVersion: eventfact.AgentVersion, RuntimeMode: "worker",
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

func (a *Application) Enqueue(
	ctx context.Context,
	collectorExecutionIDs []string,
) (eventfact.WorkItem, bool, error) {
	work, created, err := a.repository.EnqueueWork(
		ctx, collectorExecutionIDs, eventfact.AgentVersion, a.now().UTC(),
	)
	if err == nil {
		a.Notify()
	}
	return work, created, err
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
			Code: "agent_runtime_stopped", AgentKey: eventfact.AgentKey,
			AgentVersion: eventfact.AgentVersion, RuntimeMode: "worker",
			Status: "stopped",
		})
		return nil
	case <-ctx.Done():
		a.logger.Error(agentrun.AgentLifecycleEvent{
			Code: "agent_runtime_failed", AgentKey: eventfact.AgentKey,
			AgentVersion: eventfact.AgentVersion, RuntimeMode: "worker",
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
		a.logCycleFailure("event_fact_tick_failed", "tick")
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-a.notify:
		}
		if err := a.Tick(ctx); err != nil {
			a.logCycleFailure("event_fact_tick_failed", "tick")
		}
	}
}

func (a *Application) Tick(ctx context.Context) error {
	now := a.now().UTC()
	if _, err := a.repository.DispatchPendingSignals(ctx, eventfact.AgentVersion, now); err != nil {
		return err
	}
	advanced, err := a.deliver(ctx)
	if err != nil {
		return err
	}
	runtime, err := a.runtime(ctx)
	if err != nil {
		return err
	}
	if runtime.Run == nil || len(runtime.Snapshot.PromptSHA256) != 64 ||
		len(runtime.Snapshot.SchemaSHA256) != 64 ||
		runtime.Snapshot.ProviderKey == "" || runtime.Snapshot.Model == "" ||
		runtime.ReadArtifacts == nil {
		return errors.New("Event Fact Extractor runtime is invalid")
	}
	unplanned, exists, err := a.repository.NextUnplannedWork(ctx)
	if err != nil {
		return err
	}
	if exists {
		artifacts, readErr := runtime.ReadArtifacts(ctx, unplanned.CollectorExecutionIDs)
		if readErr != nil {
			return a.repository.RejectUnplannedWork(
				ctx, unplanned, "Collector Artifacts failed integrity validation", a.now().UTC(),
			)
		}
		summaries := make([]eventfact.ArtifactSummary, 0, len(artifacts))
		for _, artifact := range artifacts {
			summaries = append(summaries, eventfact.ArtifactSummary{
				ArtifactID: artifact.ArtifactID, CollectorExecutionID: artifact.CollectorExecutionID,
				ContentSHA256: artifact.ContentSHA256,
			})
		}
		if err := a.repository.InitializeArtifactUnits(ctx, unplanned, summaries, a.now().UTC()); err != nil {
			return err
		}
	}
	attempt, exists, err := a.repository.ClaimNextWork(ctx, runtime.Snapshot, now)
	if err != nil {
		return err
	}
	if exists {
		startedAt := a.now().UTC()
		a.logger.Info(a.lifecycleEvent(
			"agent_execution_started", attempt, startedAt,
			"running", "", "", nil,
		))
		unitAdvanced, err := a.extract(
			ctx, attempt, runtime.Run, runtime.ExtractFacts, startedAt,
		)
		if err != nil {
			return err
		}
		advanced = advanced || unitAdvanced
	}
	delivered, err := a.deliver(ctx)
	if err != nil {
		return err
	}
	if advanced || delivered {
		a.Notify()
	}
	return nil
}

func (a *Application) extract(
	ctx context.Context,
	attempt eventfact.ExecutionAttempt,
	run RunWorkflow,
	extractFacts ExtractFacts,
	startedAt time.Time,
) (bool, error) {
	catalog, err := a.data.ActiveEventTags(ctx)
	if err != nil {
		partial := decodePersistedResult(attempt)
		if len(partial.Candidates) == 0 && len(partial.NoEventReason) == 0 {
			if extractFacts == nil {
				retryErr := a.repository.RetryExtraction(
					ctx, attempt,
					eventfact.Result{ExecutionID: attempt.ID},
					"Event Fact-only extraction runtime is unavailable",
					a.now().UTC(),
				)
				if retryErr == nil {
					a.logRetry(
						attempt, startedAt, "runtime",
						"event_fact_extraction_runtime_unavailable",
					)
				}
				return false, retryErr
			}
			extracted, extractionErr := extractFacts(ctx, &attempt)
			if extractionErr != nil {
				if rejected, ok := modelContractRejection(attempt.ID, extractionErr); ok {
					completeErr := a.repository.CompleteWithoutPublication(
						ctx, attempt, rejected, eventfact.WorkRejected, a.now().UTC(),
					)
					if completeErr == nil {
						a.logEventFactTerminal(
							attempt, rejected, eventfact.WorkRejected, startedAt, rejected.FailureCode,
						)
					}
					return completeErr == nil, completeErr
				}
				callCount := 1
				if errors.Is(extractionErr, eventworkflow.ErrExtractionModel) {
					callCount = 1
				}
				retryErr := a.repository.RetryExtraction(
					ctx, attempt,
					eventfact.Result{
						ExecutionID: attempt.ID, ExtractionModelCalls: callCount,
					},
					"Event Fact model is unavailable", a.now().UTC(),
				)
				if retryErr == nil {
					a.logRetry(attempt, startedAt, "model", "extractor_model_unavailable")
				}
				return false, retryErr
			}
			partial = *extracted
		}
		if status, terminal := preCatalogTerminalStatus(partial); terminal {
			err := a.repository.CompleteWithoutPublication(
				ctx, attempt, partial, status, a.now().UTC(),
			)
			if err == nil {
				a.logEventFactTerminal(attempt, partial, status, startedAt, "")
			}
			return err == nil, err
		}
		waitErr := a.repository.SetAwaitingTagCatalog(
			ctx, attempt, partial, "Data Event Tag Catalog is unavailable", a.now().UTC(),
		)
		if waitErr == nil {
			a.logRetry(attempt, startedAt, "tag_catalog", "tag_catalog_unavailable")
		}
		return false, waitErr
	}
	var resume *eventfact.Result
	if attempt.Unit.Status == eventfact.WorkAwaitingTagCatalog {
		persisted := decodePersistedResult(attempt)
		if len(persisted.Candidates) > 0 || len(persisted.NoEventReason) > 0 {
			resume = &persisted
		}
	}
	result, err := run(ctx, &eventworkflow.Input{
		Attempt: attempt, ArtifactID: attempt.Unit.ArtifactID,
		Catalog: catalog, ResumeResult: resume,
	})
	if err != nil {
		if rejected, ok := modelContractRejection(attempt.ID, err); ok {
			completeErr := a.repository.CompleteWithoutPublication(
				ctx, attempt, rejected, eventfact.WorkRejected, a.now().UTC(),
			)
			if completeErr == nil {
				a.logEventFactTerminal(
					attempt, rejected, eventfact.WorkRejected, startedAt, rejected.FailureCode,
				)
			}
			return completeErr == nil, completeErr
		}
		if errors.Is(err, eventworkflow.ErrExtractionModel) ||
			errors.Is(err, eventworkflow.ErrReviewModel) ||
			errors.Is(err, context.DeadlineExceeded) {
			retryErr := a.repository.RetryExtraction(
				ctx, attempt,
				eventfact.Result{ExecutionID: attempt.ID, ExtractionModelCalls: 1},
				"Event Fact model is unavailable", a.now().UTC(),
			)
			if retryErr == nil {
				a.logRetry(attempt, startedAt, "model", "extractor_model_unavailable")
			}
			return false, retryErr
		}
		completeErr := a.repository.CompleteWithoutPublication(
			ctx, attempt,
			eventfact.Result{ExecutionID: attempt.ID, ExtractionModelCalls: 1},
			eventfact.WorkRejected, a.now().UTC(),
		)
		if completeErr == nil {
			a.logEventFactTerminal(
				attempt,
				eventfact.Result{ExecutionID: attempt.ID, ExtractionModelCalls: 1},
				eventfact.WorkRejected, startedAt, eventworkflow.FailureCode(err),
			)
		}
		return completeErr == nil, completeErr
	}
	for index := range result.Candidates {
		if result.Candidates[index].ReviewState == eventfact.ReviewManual {
			result.Candidates[index].ReviewState = eventfact.ReviewRejected
			result.Candidates[index].Review.SemanticPass = false
			result.Candidates[index].Review.Reasons = append(
				result.Candidates[index].Review.Reasons,
				"当前版本不启用人工审核，未获得 AI 确认的事件不得发布",
			)
		}
	}
	journals, err := publication.BuildArtifactUnit(
		attempt.WorkItem.Key, attempt.Unit.Key, attempt.Unit.ArtifactOrdinal,
		attempt.WorkItem.ExtractorAgentVersion, *result,
	)
	if err != nil {
		completeErr := a.repository.CompleteWithoutPublication(
			ctx, attempt, *result, eventfact.WorkRejected, a.now().UTC(),
		)
		if completeErr == nil {
			a.logEventFactTerminal(
				attempt, *result, eventfact.WorkRejected, startedAt,
				"event_publication_payload_invalid",
			)
		}
		return completeErr == nil, completeErr
	}
	if len(journals) > 0 {
		completeErr := a.repository.CompleteExtraction(
			ctx, attempt, *result, journals, a.now().UTC(),
		)
		if completeErr == nil {
			a.logEventFactTerminal(
				attempt, *result, eventfact.WorkReadyToPublish, startedAt, "",
			)
		}
		return false, completeErr
	}
	status := eventfact.WorkNoEvents
	for _, candidate := range result.Candidates {
		if candidate.ReviewState == eventfact.ReviewRejected {
			status = eventfact.WorkRejected
		}
	}
	completeErr := a.repository.CompleteWithoutPublication(
		ctx, attempt, *result, status, a.now().UTC(),
	)
	if completeErr == nil {
		a.logEventFactTerminal(attempt, *result, status, startedAt, "")
	}
	return completeErr == nil, completeErr
}

func modelContractRejection(executionID string, err error) (eventfact.Result, bool) {
	var failure *eventworkflow.ModelContractFailure
	if !errors.As(err, &failure) {
		return eventfact.Result{}, false
	}
	result := eventfact.Result{
		ExecutionID: executionID, FailureCode: eventworkflow.FailureCode(err),
		FailureStage: failure.Stage, FailureViolation: failure.Violation,
		FunctionCalls: append([]eventfact.FunctionCallObservation(nil), failure.Observations...),
	}
	for _, observation := range result.FunctionCalls {
		switch observation.Stage {
		case "extraction", "tag_assignment":
			result.ExtractionModelCalls += observation.CallCount
		case "duplicate_judgment", "review":
			result.ReviewModelCalls += observation.CallCount
		}
	}
	if len(result.FunctionCalls) == 0 {
		if failure.Stage == "duplicate_judgment" || failure.Stage == "review" {
			result.ReviewModelCalls = 2
		} else {
			result.ExtractionModelCalls = 2
		}
	}
	return result, true
}

func (a *Application) logEventFactTerminal(
	attempt eventfact.ExecutionAttempt,
	result eventfact.Result,
	status eventfact.WorkStatus,
	startedAt time.Time,
	errorCode string,
) {
	code := "agent_execution_completed"
	if status == eventfact.WorkRejected {
		code = "agent_execution_failed"
		if errorCode == "" {
			errorCode = "event_fact_rejected"
		}
		a.logger.Error(a.lifecycleEvent(
			code, attempt, startedAt, "failed", string(status), errorCode,
			map[string]int{
				"artifacts": len(result.Artifacts), "candidate_events": len(result.Candidates),
			},
		))
		return
	}
	a.logger.Info(a.lifecycleEvent(
		code, attempt, startedAt, "succeeded", string(status), "",
		map[string]int{
			"artifacts": len(result.Artifacts), "candidate_events": len(result.Candidates),
		},
	))
}

func (a *Application) logRetry(
	attempt eventfact.ExecutionAttempt,
	startedAt time.Time,
	stage string,
	errorCode string,
) {
	event := a.lifecycleEvent(
		"agent_execution_retry_scheduled", attempt, startedAt,
		"failed", "retry_scheduled", errorCode, nil,
	)
	event.Stage = stage
	a.logger.Warn(event)
}

func (a *Application) lifecycleEvent(
	code string,
	attempt eventfact.ExecutionAttempt,
	startedAt time.Time,
	status string,
	outcome string,
	errorCode string,
	counts map[string]int,
) agentrun.AgentLifecycleEvent {
	duration := a.now().UTC().Sub(startedAt)
	if duration < 0 {
		duration = 0
	}
	return agentrun.AgentLifecycleEvent{
		Code: code, AgentKey: eventfact.AgentKey,
		AgentVersion: eventfact.AgentVersion, RuntimeMode: "worker",
		ExecutionID: attempt.ID, WorkItemID: attempt.WorkItem.Key,
		TriggerSource: "dependent", Status: status, Outcome: outcome,
		ErrorCode: errorCode, Duration: duration, Counts: counts,
	}
}

func (a *Application) logCycleFailure(errorCode, stage string) {
	a.logger.Error(agentrun.AgentLifecycleEvent{
		Code: "agent_cycle_failed", AgentKey: eventfact.AgentKey,
		AgentVersion: eventfact.AgentVersion, RuntimeMode: "worker",
		Status: "failed", Stage: stage, ErrorCode: errorCode,
	})
}

func preCatalogTerminalStatus(result eventfact.Result) (eventfact.WorkStatus, bool) {
	if len(result.Candidates) == 0 {
		return eventfact.WorkNoEvents, true
	}
	for _, candidate := range result.Candidates {
		if candidate.ReviewState != eventfact.ReviewRejected {
			return "", false
		}
	}
	return eventfact.WorkRejected, true
}

func (a *Application) deliver(ctx context.Context) (bool, error) {
	now := a.now().UTC()
	journals, err := a.repository.ListDeliverableJournals(ctx, now)
	if err != nil {
		return false, err
	}
	advanced := false
	for _, journal := range journals {
		claimed, err := a.repository.MarkJournalSending(ctx, journal, now)
		if err != nil {
			return advanced, err
		}
		if !claimed {
			continue
		}
		receiptID, err := a.data.PublishReviewedEvents(ctx, journal.Payload)
		if err != nil {
			var remote *eventfact.RemoteError
			if errors.As(err, &remote) && remote.Retryable {
				if err := a.repository.MarkJournalRetry(
					ctx, journal, remote.Code, remote.Summary, a.now().UTC(),
				); err != nil {
					return advanced, err
				}
				continue
			}
			code, summary := "event_publication_blocked", "Data rejected Event publication"
			if remote != nil {
				code, summary = remote.Code, remote.Summary
			}
			if err := a.repository.MarkJournalBlocked(
				ctx, journal, code, summary, a.now().UTC(),
			); err != nil {
				return advanced, err
			}
			advanced = true
			continue
		}
		canonical, err := publication.CanonicalEvents(journal.Payload)
		if err != nil {
			err := a.repository.MarkJournalBlocked(
				ctx, journal, "stored_payload_invalid",
				"Stored Event publication payload is invalid", a.now().UTC(),
			)
			return advanced, err
		}
		if err := a.repository.AcknowledgeJournal(
			ctx, journal, receiptID, canonical, a.now().UTC(),
		); err != nil {
			return advanced, err
		}
		advanced = true
	}
	return advanced, nil
}

func decodePersistedResult(attempt eventfact.ExecutionAttempt) eventfact.Result {
	var result eventfact.Result
	if len(attempt.Unit.ExtractionResult) > 0 {
		_ = json.Unmarshal(attempt.Unit.ExtractionResult, &result)
	}
	result.ExecutionID = attempt.ID
	return result
}
