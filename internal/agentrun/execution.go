package agentrun

import (
	"errors"
	"time"
)

type AgentVersion struct {
	AgentKey string
	Version  string
}

type ExecutionStatus string

const (
	StatusQueued             ExecutionStatus = "queued"
	StatusPlanning           ExecutionStatus = "planning"
	StatusCollecting         ExecutionStatus = "collecting"
	StatusMaterializing      ExecutionStatus = "materializing"
	StatusSucceeded          ExecutionStatus = "succeeded"
	StatusSucceededNoChange  ExecutionStatus = "succeeded_no_change"
	StatusPartiallySucceeded ExecutionStatus = "partially_succeeded"
	StatusFailed             ExecutionStatus = "failed"
	StatusSkipped            ExecutionStatus = "skipped"
)

type InvocationStatus string

const (
	InvocationPending    InvocationStatus = "pending"
	InvocationRunning    InvocationStatus = "running"
	InvocationCompleted  InvocationStatus = "completed"
	InvocationFailed     InvocationStatus = "failed"
	InvocationNotInvoked InvocationStatus = "not_invoked"
)

type ConnectorInvocation struct {
	ConnectorKey string           `json:"connector_key"`
	Status       InvocationStatus `json:"status"`
	ResultCount  int              `json:"result_count"`
	ErrorCode    string           `json:"error_code,omitempty"`
	ErrorSummary string           `json:"error_summary,omitempty"`
	StartedAt    *time.Time       `json:"started_at,omitempty"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
}

type Execution struct {
	ID                   string                `json:"execution_id"`
	AgentVersion         string                `json:"agent_version"`
	IdempotencyKey       string                `json:"-"`
	Prompt               string                `json:"-"`
	PromptSHA256         string                `json:"prompt_sha256"`
	PromptBytes          int                   `json:"prompt_bytes"`
	Status               ExecutionStatus       `json:"status"`
	ErrorCode            string                `json:"error_code,omitempty"`
	ErrorSummary         string                `json:"error_summary,omitempty"`
	StopReason           string                `json:"stop_reason,omitempty"`
	BlockedByExecutionID string                `json:"blocked_by_execution_id,omitempty"`
	CandidateCounts      map[string]int        `json:"candidate_counts"`
	Artifacts            map[string]string     `json:"artifacts"`
	CreatedAt            time.Time             `json:"created_at"`
	StartedAt            *time.Time            `json:"started_at,omitempty"`
	CompletedAt          *time.Time            `json:"completed_at,omitempty"`
	Invocations          []ConnectorInvocation `json:"invocations"`
}

type CreateExecutionInput struct {
	IdempotencyKey string
	Prompt         string
	CreatedAt      time.Time
	AgentVersion   string
	InvocationKeys []string
}

type CreateDisposition string

const (
	ExecutionCreated  CreateDisposition = "created"
	ExecutionReplayed CreateDisposition = "replayed"
	ExecutionSkipped  CreateDisposition = "skipped"
)

var ErrIdempotencyConflict = errors.New("idempotency key already belongs to a different prompt")

type ActiveExecutionError struct {
	ActiveExecutionID  string
	SkippedExecutionID string
}

type InvocationCompletion struct {
	ExecutionID  string
	ConnectorKey string
	Status       InvocationStatus
	ResultCount  int
	ErrorCode    string
	ErrorSummary string
	CompletedAt  time.Time
}

type ExecutionFailure struct {
	ExecutionID       string
	ErrorCode         string
	ErrorSummary      string
	StopReason        string
	NotInvokedSummary string
	Artifacts         map[string]string
	CompletedAt       time.Time
}

type ExecutionCompletion struct {
	ExecutionID     string
	Status          ExecutionStatus
	StopReason      string
	ErrorCode       string
	ErrorSummary    string
	CandidateCounts map[string]int
	Artifacts       map[string]string
	CompletedAt     time.Time
}

type PublicationReference struct {
	ExecutionID string
	PlanPath    string
	PlanSHA256  string
	PreparedAt  time.Time
}

func (e *ActiveExecutionError) Error() string {
	return "another Agent Execution is active"
}
