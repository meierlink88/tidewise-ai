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

type EventLogger interface {
	Error(string)
}

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
		logger: discardEventLogger{},
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
		a.logger.Error("event_fact_tick_failed")
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-a.notify:
		}
		if err := a.Tick(ctx); err != nil {
			a.logger.Error("event_fact_tick_failed")
		}
	}
}

type discardEventLogger struct{}

func (discardEventLogger) Error(string) {}

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
		unitAdvanced, err := a.extract(ctx, attempt, runtime.Run, runtime.ExtractFacts)
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
) (bool, error) {
	catalog, err := a.data.ActiveEventTags(ctx)
	if err != nil {
		partial := decodePersistedResult(attempt)
		if len(partial.Candidates) == 0 && len(partial.NoEventReason) == 0 {
			if extractFacts == nil {
				return false, errors.New("Event Fact-only extraction runtime is invalid")
			}
			extracted, extractionErr := extractFacts(ctx, &attempt)
			if extractionErr != nil {
				callCount := 1
				if errors.Is(extractionErr, eventworkflow.ErrExtractionModel) {
					callCount = 1
				}
				return false, a.repository.RetryExtraction(
					ctx, attempt,
					eventfact.Result{
						ExecutionID: attempt.ID, ExtractionModelCalls: callCount,
					},
					"Event Fact model is unavailable", a.now().UTC(),
				)
			}
			partial = *extracted
		}
		if status, terminal := preCatalogTerminalStatus(partial); terminal {
			err := a.repository.CompleteWithoutPublication(
				ctx, attempt, partial, status, a.now().UTC(),
			)
			return err == nil, err
		}
		return false, a.repository.SetAwaitingTagCatalog(
			ctx, attempt, partial, "Data Event Tag Catalog is unavailable", a.now().UTC(),
		)
	}
	if err := a.repository.SetExecutionCatalog(
		ctx, attempt.ID, catalog.Revision, catalog.Hash, a.now().UTC(),
	); err != nil {
		return false, err
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
		if errors.Is(err, eventworkflow.ErrExtractionModel) ||
			errors.Is(err, eventworkflow.ErrReviewModel) ||
			errors.Is(err, context.DeadlineExceeded) {
			return false, a.repository.RetryExtraction(
				ctx, attempt,
				eventfact.Result{ExecutionID: attempt.ID, ExtractionModelCalls: 1},
				"Event Fact model is unavailable", a.now().UTC(),
			)
		}
		a.logger.Error(eventworkflow.FailureCode(err))
		completeErr := a.repository.CompleteWithoutPublication(
			ctx, attempt,
			eventfact.Result{ExecutionID: attempt.ID, ExtractionModelCalls: 1},
			eventfact.WorkRejected, a.now().UTC(),
		)
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
		return completeErr == nil, completeErr
	}
	if len(journals) > 0 {
		return false, a.repository.CompleteExtraction(ctx, attempt, *result, journals, a.now().UTC())
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
	return completeErr == nil, completeErr
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
