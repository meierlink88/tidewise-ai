package usecase

import (
	"context"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

// Repository is the persistence seam required by the Collector application.
// PostgreSQL is the current adapter; tests may replace it at this seam.
type Repository interface {
	SchemaReady(context.Context) bool
	LoadModelProviderConfigs(context.Context) (map[string]agentrun.ModelProviderConfig, error)
	LoadConnectorConfigs(context.Context) (map[string]agentrun.ConnectorConfig, error)
	FindExecutionByIdempotencyKey(context.Context, string, string) (agentrun.Execution, bool, error)
	CreateExecution(context.Context, agentrun.CreateExecutionInput) (agentrun.Execution, agentrun.CreateDisposition, error)
	CreateExecutionIfActive(context.Context, agentrun.CreateExecutionInput) (agentrun.Execution, agentrun.CreateDisposition, error)
	GetExecution(context.Context, string) (agentrun.Execution, error)
	SetExecutionStatus(context.Context, string, agentrun.ExecutionStatus, time.Time) error
	StartInvocation(context.Context, string, string, time.Time) error
	FinishInvocation(context.Context, agentrun.InvocationCompletion) error
	FailExecutionAndIncompleteInvocations(context.Context, agentrun.ExecutionFailure) error
	PreparePublication(context.Context, agentrun.PublicationReference) error
	ListPreparedPublications(context.Context) ([]agentrun.PublicationReference, error)
	CommitPreparedPublication(context.Context, agentrun.PublicationReference, agentrun.ExecutionCompletion) error
	AttachTerminalArtifacts(context.Context, string, map[string]string, time.Time) error
	ListTerminalExecutionsWithoutArtifacts(context.Context) ([]agentrun.Execution, error)
	FailStaleExecutions(context.Context, time.Time) error
}

// ModelFactory constructs the Eino core model used by one immutable Execution snapshot.
// Provider SDK details remain in the Data adapter.
type ModelFactory interface {
	New(context.Context, agentrun.ModelProviderConfig) (model.BaseChatModel, error)
}

// ConnectorFactory constructs the fixed Connector set for one immutable Execution snapshot.
type ConnectorFactory interface {
	New(collector.RuntimeConfiguration) ([]collector.Connector, error)
}

// ArtifactStore owns filesystem readiness, publication reconciliation and terminal audit I/O.
type ArtifactStore interface {
	Ready(context.Context) error
	Materializer(int) collector.Materializer
	ReconcilePreparedPublications(context.Context) error
	WriteTerminalAudit(agentrun.Execution) (map[string]string, error)
}

// RuntimeBuilder compiles the concrete Collector Eino Workflow for one Execution.
type RuntimeBuilder interface {
	Build(
		context.Context,
		string,
		collector.RuntimeConfiguration,
	) (compose.Runnable[*collector.Request, *collector.Result], error)
}
