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
	CompleteExecution(context.Context, agentrun.ExecutionCompletion) error
}
