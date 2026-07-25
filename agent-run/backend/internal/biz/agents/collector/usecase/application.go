package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/planning"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

const (
	collectorAgentVersion = "collector.v1"
	plannerTimeout        = 30 * time.Second
	connectorTimeout      = 30 * time.Second
	executionTimeout      = 5 * time.Minute
	maxParallel           = 3
	candidateLimit        = 10
)

var collectorConnectorKeys = collector.ConnectorKeys()

var ErrNotReady = errors.New("AgentRun is not ready")
var ErrPublicationPending = errors.New("Artifact publication is prepared for reconciliation")
var errStatePersistence = errors.New("AgentRun state persistence failed")
var errArtifactMaterialization = errors.New("Artifact materialization failed")

type Application struct {
	store          Repository
	runtimeBuilder RuntimeBuilder
	artifacts      ArtifactStore
	environment    string
	executionTTL   time.Duration
	now            func() time.Time
	lifecycleCtx   context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
	closing        bool
	active         sync.WaitGroup
	events         EventLogger
}

type Option func(*Application)

type EventLogger interface {
	Error(eventCode string, executionID string)
}

type discardEventLogger struct{}

func (discardEventLogger) Error(string, string) {}

func WithEventLogger(logger EventLogger) Option {
	return func(application *Application) {
		if logger != nil {
			application.events = logger
		}
	}
}

func WithEnvironment(environment string) Option {
	return func(application *Application) {
		application.environment = environment
	}
}

func WithExecutionTimeout(timeout time.Duration) Option {
	return func(application *Application) {
		application.executionTTL = timeout
	}
}

func New(
	store Repository,
	modelFactory ModelFactory,
	connectorFactory ConnectorFactory,
	artifactStore ArtifactStore,
	options ...Option,
) (*Application, error) {
	if store == nil {
		return nil, errors.New("AgentRun Store is required")
	}
	if modelFactory == nil {
		return nil, errors.New("Model Factory is required")
	}
	if connectorFactory == nil {
		return nil, errors.New("Connector Factory is required")
	}
	if artifactStore == nil {
		return nil, errors.New("Artifact Store is required")
	}
	application := &Application{
		store:        store,
		artifacts:    artifactStore,
		environment:  "dev",
		executionTTL: executionTimeout,
		now:          time.Now,
		events:       discardEventLogger{},
	}
	application.lifecycleCtx, application.cancel = context.WithCancel(context.Background())
	application.runtimeBuilder = runtimeFactory{
		store:            store,
		modelFactory:     modelFactory,
		connectorFactory: connectorFactory,
		artifacts:        artifactStore,
		now:              application.now,
	}
	for _, option := range options {
		option(application)
	}
	if application.executionTTL <= 0 {
		return nil, errors.New("Execution timeout must be positive")
	}
	return application, nil
}

func (a *Application) Ready(ctx context.Context) error {
	_, err := a.loadReadyRuntimeConfiguration(ctx)
	return err
}

func (a *Application) loadReadyRuntimeConfiguration(ctx context.Context) (collector.RuntimeConfiguration, error) {
	if a.environment != "dev" && a.environment != "uat" {
		return collector.RuntimeConfiguration{}, ErrNotReady
	}
	if !a.store.SchemaReady(ctx) {
		return collector.RuntimeConfiguration{}, ErrNotReady
	}
	runtimeConfiguration, err := a.loadRuntimeConfiguration(ctx)
	if err != nil {
		return collector.RuntimeConfiguration{}, ErrNotReady
	}
	if err := a.artifacts.Ready(ctx); err != nil {
		return collector.RuntimeConfiguration{}, ErrNotReady
	}
	return runtimeConfiguration, nil
}

func (a *Application) CreateCollectorRun(ctx context.Context, idempotencyKey, prompt string) (agentrun.Execution, agentrun.CreateDisposition, error) {
	createdAt := a.now().UTC()
	return a.createCollectorRun(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: idempotencyKey,
		Prompt:         prompt,
		CreatedAt:      createdAt,
		TriggeredAt:    createdAt,
		TriggerSource:  agentrun.TriggerAPI,
		AgentVersion:   collectorAgentVersion,
		InvocationKeys: collectorConnectorKeys,
	})
}

func (a *Application) CreateScheduledCollectorRun(
	ctx context.Context,
	idempotencyKey string,
	scheduleID string,
	prompt string,
	inputPayload json.RawMessage,
	triggeredAt time.Time,
) (agentrun.Execution, agentrun.CreateDisposition, error) {
	triggeredAt = triggeredAt.UTC()
	return a.createCollectorRun(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: idempotencyKey,
		InputPayload:   append(json.RawMessage(nil), inputPayload...),
		Prompt:         prompt,
		CreatedAt:      triggeredAt,
		TriggeredAt:    triggeredAt,
		TriggerSource:  agentrun.TriggerSchedule,
		ScheduleID:     scheduleID,
		AgentVersion:   collectorAgentVersion,
		InvocationKeys: collectorConnectorKeys,
	})
}

func (a *Application) createCollectorRun(
	ctx context.Context,
	input agentrun.CreateExecutionInput,
) (agentrun.Execution, agentrun.CreateDisposition, error) {
	if execution, found, err := a.store.FindExecutionByIdempotencyKey(ctx, input.IdempotencyKey, input.Prompt); err != nil {
		return agentrun.Execution{}, "", err
	} else if found {
		if execution.Status == agentrun.StatusSkipped {
			a.attachTerminalAudit(execution)
			return execution, agentrun.ExecutionReplayed, &agentrun.ActiveExecutionError{
				ActiveExecutionID: execution.BlockedByExecutionID, SkippedExecutionID: execution.ID,
			}
		}
		return execution, agentrun.ExecutionReplayed, nil
	}
	runtimeConfiguration, err := a.loadReadyRuntimeConfiguration(ctx)
	if err != nil {
		execution, disposition, createErr := a.store.CreateExecutionIfActive(ctx, input)
		if errors.Is(createErr, agentrun.ErrNoActiveExecution) {
			return agentrun.Execution{}, "", ErrNotReady
		}
		if createErr != nil {
			return agentrun.Execution{}, "", createErr
		}
		if disposition == agentrun.ExecutionSkipped || execution.Status == agentrun.StatusSkipped {
			a.attachTerminalAudit(execution)
			return execution, disposition, &agentrun.ActiveExecutionError{
				ActiveExecutionID: execution.BlockedByExecutionID, SkippedExecutionID: execution.ID,
			}
		}
		return execution, disposition, nil
	}
	execution, disposition, err := a.store.CreateExecution(ctx, input)
	if err != nil {
		return agentrun.Execution{}, "", err
	}
	if disposition == agentrun.ExecutionSkipped {
		a.attachTerminalAudit(execution)
		return execution, disposition, &agentrun.ActiveExecutionError{
			ActiveExecutionID: execution.BlockedByExecutionID, SkippedExecutionID: execution.ID,
		}
	}
	if disposition == agentrun.ExecutionCreated {
		if !a.start(execution, runtimeConfiguration) {
			a.fail(execution.ID, "service_stopping", "AgentRun is stopping")
		}
	}
	return execution, disposition, nil
}

func (a *Application) start(execution agentrun.Execution, runtimeConfiguration collector.RuntimeConfiguration) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closing {
		return false
	}
	a.active.Add(1)
	go func() {
		defer a.active.Done()
		a.run(execution, runtimeConfiguration)
	}()
	return true
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.BeginShutdown()
	return a.Wait(ctx)
}

func (a *Application) BeginShutdown() {
	a.mu.Lock()
	if !a.closing {
		a.closing = true
		a.cancel()
	}
	a.mu.Unlock()
}

func (a *Application) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		a.active.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *Application) loadRuntimeConfiguration(ctx context.Context) (collector.RuntimeConfiguration, error) {
	models, err := a.store.LoadModelProviderConfigs(ctx)
	if err != nil {
		return collector.RuntimeConfiguration{}, err
	}
	connectors, err := a.store.LoadConnectorConfigs(ctx)
	if err != nil {
		return collector.RuntimeConfiguration{}, err
	}
	return collector.BuildRuntimeConfigurationForEnvironment(models, connectors, a.environment)
}

func (a *Application) GetCollectorRun(ctx context.Context, executionID string) (agentrun.Execution, error) {
	return a.store.GetExecution(ctx, executionID)
}

func (a *Application) ReconcilePreparedPublications(ctx context.Context) error {
	return a.artifacts.ReconcilePreparedPublications(ctx)
}

func (a *Application) ReconcileStartup(ctx context.Context, staleAt time.Time) error {
	if err := a.ReconcilePreparedPublications(ctx); err != nil {
		return fmt.Errorf("reconcile prepared Artifact publications: %w", err)
	}
	if err := a.store.FailStaleExecutions(ctx, staleAt.UTC()); err != nil {
		return fmt.Errorf("reconcile stale Agent Executions: %w", err)
	}
	if err := a.PublishMissingTerminalAudits(ctx); err != nil {
		return fmt.Errorf("publish terminal Artifact audits: %w", err)
	}
	return nil
}

func (a *Application) PublishMissingTerminalAudits(ctx context.Context) error {
	executions, err := a.store.ListTerminalExecutionsWithoutArtifacts(ctx)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		paths, err := a.artifacts.WriteTerminalAudit(execution)
		if err != nil {
			return err
		}
		if err := a.store.AttachTerminalArtifacts(ctx, execution.ID, paths, a.now()); err != nil {
			return err
		}
	}
	return nil
}

func (a *Application) run(execution agentrun.Execution, runtimeConfig collector.RuntimeConfiguration) {
	ctx, cancel := context.WithTimeout(a.lifecycleCtx, a.executionTTL)
	defer cancel()
	if err := a.store.SetExecutionStatus(ctx, execution.ID, agentrun.StatusPlanning, a.now()); err != nil {
		a.fail(execution.ID, "state_transition_failed", "Could not start Agent Execution")
		return
	}

	workflow, err := a.runtimeBuilder.Build(ctx, execution.ID, runtimeConfig)
	if err != nil {
		a.fail(execution.ID, "workflow_initialization_failed", "Could not initialize Collector Workflow")
		return
	}
	result, err := workflow.Invoke(ctx, &collector.Request{
		RunID: execution.ID, Prompt: execution.Prompt, CandidateLimit: candidateLimit,
		CollectedAt: execution.CreatedAt,
	})
	if err != nil {
		if errors.Is(err, ErrPublicationPending) {
			a.events.Error("artifact_publication_pending", execution.ID)
			return
		}
		code := "execution_failed"
		summary := "Collector execution failed"
		if errors.Is(err, context.Canceled) {
			code = "service_stopping"
			summary = "AgentRun stopped the active Collector Execution"
		} else if errors.Is(err, context.DeadlineExceeded) {
			code = "execution_timeout"
			summary = "Collector execution timed out"
		} else if errors.Is(err, planning.ErrQueryPlanningModel) || errors.Is(err, planning.ErrQueryPlanningSchema) {
			code = "planning_failed"
			summary = "Query planning failed"
		} else if errors.Is(err, errStatePersistence) {
			code = "state_transition_failed"
			summary = "Could not persist Agent Execution state"
		} else if errors.Is(err, errArtifactMaterialization) {
			code = "artifact_failed"
			summary = "Could not publish Collector Artifacts"
		}
		a.fail(execution.ID, code, summary)
		return
	}
	if result == nil {
		a.fail(execution.ID, "execution_failed", "Collector execution returned no result")
		return
	}
}

func (a *Application) fail(executionID, code, summary string) {
	notInvokedSummary := "Connector was not invoked because Agent Execution stopped"
	if code == "planning_failed" || code == "planner_initialization_failed" {
		notInvokedSummary = "Connector was not invoked because query planning failed"
	}
	completedAt := a.now().UTC()
	if err := retryStateWrite(func(ctx context.Context) error {
		return a.store.FailExecutionAndIncompleteInvocations(ctx, agentrun.ExecutionFailure{
			ExecutionID: executionID, ErrorCode: code, ErrorSummary: summary,
			StopReason:        "agent_or_tool_limit",
			NotInvokedSummary: notInvokedSummary, CompletedAt: completedAt,
		})
	}); err != nil {
		a.events.Error("terminal_failure_persistence_failed", executionID)
		return
	}
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	execution, err := a.store.GetExecution(readCtx, executionID)
	readCancel()
	if err != nil {
		a.events.Error("terminal_failure_read_failed", executionID)
		return
	}
	paths, err := a.artifacts.WriteTerminalAudit(execution)
	if err != nil {
		a.events.Error("terminal_audit_publication_failed", execution.ID)
		return
	}
	if err := retryStateWrite(func(ctx context.Context) error {
		return a.store.AttachTerminalArtifacts(ctx, execution.ID, paths, a.now())
	}); err != nil {
		a.events.Error("terminal_audit_attachment_failed", execution.ID)
	}
}

func (a *Application) attachTerminalAudit(execution agentrun.Execution) {
	paths, err := a.artifacts.WriteTerminalAudit(execution)
	if err != nil {
		a.events.Error("terminal_audit_publication_failed", execution.ID)
		return
	}
	if err := retryStateWrite(func(ctx context.Context) error {
		return a.store.AttachTerminalArtifacts(ctx, execution.ID, paths, a.now())
	}); err != nil {
		a.events.Error("terminal_audit_attachment_failed", execution.ID)
	}
}

func retryStateWrite(write func(context.Context) error) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		last = write(ctx)
		cancel()
		if last == nil {
			return nil
		}
	}
	return last
}

type trackingPlanner struct {
	executionID string
	store       Repository
	delegate    planning.QueryPlanner
	now         func() time.Time
}

func (p *trackingPlanner) Plan(ctx context.Context, request *collector.Request) (*collector.Request, error) {
	plannerCtx, cancel := context.WithTimeout(ctx, plannerTimeout)
	defer cancel()
	planned, err := p.delegate.Plan(plannerCtx, request)
	if err != nil {
		return nil, err
	}
	if err := p.store.SetExecutionStatus(ctx, p.executionID, agentrun.StatusCollecting, p.now()); err != nil {
		return nil, fmt.Errorf("%w: collecting", errStatePersistence)
	}
	return planned, nil
}

type trackingConnector struct {
	executionID string
	store       Repository
	delegate    collector.Connector
	now         func() time.Time
}

func (c *trackingConnector) Name() string { return c.delegate.Name() }

func (c *trackingConnector) Collect(ctx context.Context, request collector.Request) ([]collector.Candidate, error) {
	if err := c.store.StartInvocation(ctx, c.executionID, c.Name(), c.now()); err != nil {
		return nil, fmt.Errorf("%w: start Connector", errStatePersistence)
	}
	connectorCtx, cancel := context.WithTimeout(ctx, connectorTimeout)
	defer cancel()
	results, err := c.delegate.Collect(connectorCtx, request)
	finishCtx, finishCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finishCancel()
	if err != nil {
		errorCode := "connector_failed"
		errorSummary := "Connector request failed"
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			errorCode = "service_stopping"
			errorSummary = "Connector stopped with AgentRun"
		}
		if finishErr := c.store.FinishInvocation(finishCtx, agentrun.InvocationCompletion{
			ExecutionID: c.executionID, ConnectorKey: c.Name(), Status: agentrun.InvocationFailed,
			ErrorCode: errorCode, ErrorSummary: errorSummary, CompletedAt: c.now(),
		}); finishErr != nil {
			return nil, fmt.Errorf("%w: fail Connector", errStatePersistence)
		}
		if errorCode == "service_stopping" {
			return nil, context.Canceled
		}
		return nil, errors.New("Connector request failed")
	}
	if finishErr := c.store.FinishInvocation(finishCtx, agentrun.InvocationCompletion{
		ExecutionID: c.executionID, ConnectorKey: c.Name(), Status: agentrun.InvocationCompleted,
		ResultCount: len(results), CompletedAt: c.now(),
	}); finishErr != nil {
		return nil, fmt.Errorf("%w: complete Connector", errStatePersistence)
	}
	return results, nil
}

type trackingMaterializer struct {
	executionID string
	store       Repository
	delegate    collector.Materializer
	now         func() time.Time
}

func (m *trackingMaterializer) Materialize(ctx context.Context, request collector.Request, runs map[string]collector.ConnectorRun) (*collector.Result, error) {
	for _, run := range runs {
		if errors.Is(run.Err, errStatePersistence) {
			return nil, errStatePersistence
		}
	}
	if err := m.store.SetExecutionStatus(ctx, m.executionID, agentrun.StatusMaterializing, m.now()); err != nil {
		return nil, fmt.Errorf("%w: materializing", errStatePersistence)
	}
	result, err := m.delegate.Materialize(ctx, request, runs)
	if err != nil {
		if errors.Is(err, ErrPublicationPending) {
			return nil, err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, errArtifactMaterialization
	}
	return result, nil
}
