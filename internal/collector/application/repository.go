package application

import (
	"context"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
)

// Repository is the persistence seam required by the Collector application.
// PostgreSQL is the current adapter; tests may replace it at this seam.
type Repository interface {
	SchemaReady(context.Context) bool
	LoadProviderConfigs(context.Context) (map[string]agentrun.ProviderConfig, error)
	FindExecutionByIdempotencyKey(context.Context, string, string) (agentrun.Execution, bool, error)
	CreateExecution(context.Context, agentrun.CreateExecutionInput) (agentrun.Execution, agentrun.CreateDisposition, error)
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
