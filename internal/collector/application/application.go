package application

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/artifacts"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/planning"
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
var errStatePersistence = errors.New("AgentRun state persistence failed")
var errArtifactMaterialization = errors.New("Artifact materialization failed")

type Application struct {
	store                      Repository
	artifactRoot               string
	environment                string
	executionTTL               time.Duration
	now                        func() time.Time
	runtimeConfiguration       collector.RuntimeConfiguration
	runtimeConfigurationLoaded bool
}

type Option func(*Application)

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

func New(store Repository, artifactRoot string, options ...Option) (*Application, error) {
	if store == nil {
		return nil, errors.New("AgentRun Store is required")
	}
	if artifactRoot == "" {
		return nil, errors.New("Artifact root is required")
	}
	application := &Application{store: store, artifactRoot: artifactRoot, environment: "dev", executionTTL: executionTimeout, now: time.Now}
	for _, option := range options {
		option(application)
	}
	if application.executionTTL <= 0 {
		return nil, errors.New("Execution timeout must be positive")
	}
	if runtimeConfiguration, err := application.loadRuntimeConfiguration(context.Background()); err == nil {
		application.runtimeConfiguration = runtimeConfiguration
		application.runtimeConfigurationLoaded = true
	}
	return application, nil
}

func (a *Application) Ready(ctx context.Context) error {
	if a.environment != "dev" && a.environment != "uat" {
		return ErrNotReady
	}
	if !a.store.SchemaReady(ctx) {
		return ErrNotReady
	}
	if !a.runtimeConfigurationLoaded {
		return ErrNotReady
	}
	if err := os.MkdirAll(a.artifactRoot, 0o755); err != nil {
		return ErrNotReady
	}
	probe, err := os.CreateTemp(a.artifactRoot, ".ready-*")
	if err != nil {
		return ErrNotReady
	}
	name := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(name)
		return ErrNotReady
	}
	if err := os.Remove(name); err != nil {
		return ErrNotReady
	}
	return nil
}

func (a *Application) CreateCollectorRun(ctx context.Context, idempotencyKey, prompt string) (agentrun.Execution, agentrun.CreateDisposition, error) {
	if execution, found, err := a.store.FindExecutionByIdempotencyKey(ctx, idempotencyKey, prompt); err != nil {
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
	if err := a.Ready(ctx); err != nil {
		return agentrun.Execution{}, "", ErrNotReady
	}
	if !a.runtimeConfigurationLoaded {
		return agentrun.Execution{}, "", ErrNotReady
	}
	createdAt := a.now().UTC()
	execution, disposition, err := a.store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: idempotencyKey,
		Prompt:         prompt,
		CreatedAt:      createdAt,
		AgentVersion:   collectorAgentVersion,
		InvocationKeys: collectorConnectorKeys,
	})
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
		go a.run(execution, a.runtimeConfiguration)
	}
	return execution, disposition, nil
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
	return artifacts.ReconcilePreparedPublications(ctx, a.artifactRoot, a.store)
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
		paths, err := artifacts.WriteTerminalAudit(a.artifactRoot, execution)
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
	ctx, cancel := context.WithTimeout(context.Background(), a.executionTTL)
	defer cancel()
	if err := a.store.SetExecutionStatus(ctx, execution.ID, agentrun.StatusPlanning, a.now()); err != nil {
		a.fail(execution.ID, "state_transition_failed", "Could not start Agent Execution")
		return
	}

	workflow, err := (runtimeFactory{store: a.store, artifactRoot: a.artifactRoot, now: a.now}).Build(ctx, execution.ID, runtimeConfig)
	if err != nil {
		a.fail(execution.ID, "workflow_initialization_failed", "Could not initialize Collector Workflow")
		return
	}
	result, err := workflow.Invoke(ctx, &collector.Request{
		RunID: execution.ID, Prompt: execution.Prompt, CandidateLimit: candidateLimit,
		CollectedAt: execution.CreatedAt,
	})
	if err != nil {
		if errors.Is(err, artifacts.ErrPublicationPending) {
			log.Printf("AgentRun left a prepared Artifact publication for reconciliation execution_id=%s", execution.ID)
			return
		}
		code := "execution_failed"
		summary := "Collector execution failed"
		if errors.Is(err, context.DeadlineExceeded) {
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
		log.Printf("AgentRun could not persist terminal failure execution_id=%s", executionID)
		return
	}
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	execution, err := a.store.GetExecution(readCtx, executionID)
	readCancel()
	if err != nil {
		log.Printf("AgentRun could not read terminal failure execution_id=%s", executionID)
		return
	}
	paths, err := artifacts.WriteTerminalAudit(a.artifactRoot, execution)
	if err != nil {
		log.Printf("AgentRun could not publish terminal Artifact audit execution_id=%s error=%v", execution.ID, err)
		return
	}
	if err := retryStateWrite(func(ctx context.Context) error {
		return a.store.AttachTerminalArtifacts(ctx, execution.ID, paths, a.now())
	}); err != nil {
		log.Printf("AgentRun could not attach terminal Artifact audit execution_id=%s", execution.ID)
	}
}

func (a *Application) attachTerminalAudit(execution agentrun.Execution) {
	paths, err := artifacts.WriteTerminalAudit(a.artifactRoot, execution)
	if err != nil {
		log.Printf("AgentRun could not publish terminal Artifact audit execution_id=%s error=%v", execution.ID, err)
		return
	}
	if err := retryStateWrite(func(ctx context.Context) error {
		return a.store.AttachTerminalArtifacts(ctx, execution.ID, paths, a.now())
	}); err != nil {
		log.Printf("AgentRun could not attach terminal Artifact audit execution_id=%s", execution.ID)
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
		if finishErr := c.store.FinishInvocation(finishCtx, agentrun.InvocationCompletion{
			ExecutionID: c.executionID, ConnectorKey: c.Name(), Status: agentrun.InvocationFailed,
			ErrorCode: "connector_failed", ErrorSummary: "Connector request failed", CompletedAt: c.now(),
		}); finishErr != nil {
			return nil, fmt.Errorf("%w: fail Connector", errStatePersistence)
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
		if errors.Is(err, artifacts.ErrPublicationPending) {
			return nil, err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, errArtifactMaterialization
	}
	return result, nil
}
