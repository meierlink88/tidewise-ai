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
	Snapshot     eventfact.ExtractionSnapshot
	Run          RunWorkflow
	ExtractFacts ExtractFacts
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
	if err := a.deliver(ctx); err != nil {
		return err
	}
	runtime, err := a.runtime(ctx)
	if err != nil {
		return err
	}
	if runtime.Run == nil || len(runtime.Snapshot.PromptSHA256) != 64 ||
		len(runtime.Snapshot.SchemaSHA256) != 64 ||
		runtime.Snapshot.ProviderKey == "" || runtime.Snapshot.Model == "" {
		return errors.New("Event Fact Extractor runtime is invalid")
	}
	attempt, exists, err := a.repository.ClaimNextWork(ctx, runtime.Snapshot, now)
	if err != nil {
		return err
	}
	if exists {
		if err := a.extract(ctx, attempt, runtime.Run, runtime.ExtractFacts); err != nil {
			return err
		}
	}
	return a.deliver(ctx)
}

func (a *Application) extract(
	ctx context.Context,
	attempt eventfact.ExecutionAttempt,
	run RunWorkflow,
	extractFacts ExtractFacts,
) error {
	catalog, err := a.data.ActiveEventTags(ctx)
	if err != nil {
		partial := decodePersistedResult(attempt)
		if len(partial.Candidates) == 0 && len(partial.NoEventReason) == 0 {
			if extractFacts == nil {
				return errors.New("Event Fact-only extraction runtime is invalid")
			}
			extracted, extractionErr := extractFacts(ctx, &attempt)
			if extractionErr != nil {
				callCount := 1
				if errors.Is(extractionErr, eventworkflow.ErrExtractionModel) {
					callCount = 1
				}
				return a.repository.RetryExtraction(
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
			return a.repository.CompleteWithoutPublication(
				ctx, attempt, partial, status, a.now().UTC(),
			)
		}
		return a.repository.SetAwaitingTagCatalog(
			ctx, attempt, partial, "Data Event Tag Catalog is unavailable", a.now().UTC(),
		)
	}
	if err := a.repository.SetExecutionCatalog(
		ctx, attempt.ID, catalog.Revision, catalog.Hash, a.now().UTC(),
	); err != nil {
		return err
	}
	var resume *eventfact.Result
	if attempt.WorkItem.Status == eventfact.WorkAwaitingTagCatalog {
		persisted := decodePersistedResult(attempt)
		if len(persisted.Candidates) > 0 || len(persisted.NoEventReason) > 0 {
			resume = &persisted
		}
	}
	result, err := run(ctx, &eventworkflow.Input{
		Attempt: attempt, Catalog: catalog, ResumeResult: resume,
	})
	if err != nil {
		if errors.Is(err, eventworkflow.ErrExtractionModel) ||
			errors.Is(err, eventworkflow.ErrReviewModel) ||
			errors.Is(err, context.DeadlineExceeded) {
			return a.repository.RetryExtraction(
				ctx, attempt,
				eventfact.Result{ExecutionID: attempt.ID, ExtractionModelCalls: 1},
				"Event Fact model is unavailable", a.now().UTC(),
			)
		}
		return a.repository.CompleteWithoutPublication(
			ctx, attempt,
			eventfact.Result{ExecutionID: attempt.ID, ExtractionModelCalls: 1},
			eventfact.WorkRejected, a.now().UTC(),
		)
	}
	for _, candidate := range result.Candidates {
		if candidate.ReviewState == eventfact.ReviewManual {
			return a.repository.CompleteWithoutPublication(
				ctx, attempt, *result, eventfact.WorkAwaitingReview, a.now().UTC(),
			)
		}
	}
	journals, err := publication.Build(attempt.WorkItem.Key, attempt.WorkItem.ExtractorAgentVersion, *result)
	if err != nil {
		return a.repository.CompleteWithoutPublication(
			ctx, attempt, *result, eventfact.WorkRejected, a.now().UTC(),
		)
	}
	if len(journals) > 0 {
		return a.repository.CompleteExtraction(ctx, attempt, *result, journals, a.now().UTC())
	}
	status := eventfact.WorkNoEvents
	for _, candidate := range result.Candidates {
		if candidate.ReviewState == eventfact.ReviewRejected {
			status = eventfact.WorkRejected
		}
	}
	return a.repository.CompleteWithoutPublication(ctx, attempt, *result, status, a.now().UTC())
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

func (a *Application) deliver(ctx context.Context) error {
	now := a.now().UTC()
	journals, err := a.repository.ListDeliverableJournals(ctx, now)
	if err != nil {
		return err
	}
	for _, journal := range journals {
		claimed, err := a.repository.MarkJournalSending(ctx, journal, now)
		if err != nil {
			return err
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
					return err
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
				return err
			}
			continue
		}
		canonical, err := publication.CanonicalEvents(journal.Payload)
		if err != nil {
			return a.repository.MarkJournalBlocked(
				ctx, journal, "stored_payload_invalid",
				"Stored Event publication payload is invalid", a.now().UTC(),
			)
		}
		if err := a.repository.AcknowledgeJournal(
			ctx, journal, receiptID, canonical, a.now().UTC(),
		); err != nil {
			return err
		}
	}
	return nil
}

func decodePersistedResult(attempt eventfact.ExecutionAttempt) eventfact.Result {
	var result eventfact.Result
	if len(attempt.WorkItem.ExtractionResult) > 0 {
		_ = json.Unmarshal(attempt.WorkItem.ExtractionResult, &result)
	}
	result.ExecutionID = attempt.ID
	return result
}
