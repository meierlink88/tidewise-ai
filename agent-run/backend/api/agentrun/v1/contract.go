package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const (
	OperationCreateCollectorRun  = "/agentrun.v1.AgentRun/CreateCollectorRun"
	OperationGetCollectorRun     = "/agentrun.v1.AgentRun/GetCollectorRun"
	OperationListModelProviders  = "/agentrun.v1.AgentRun/ListModelProviders"
	OperationGetModelProvider    = "/agentrun.v1.AgentRun/GetModelProvider"
	OperationPatchModelProvider  = "/agentrun.v1.AgentRun/PatchModelProvider"
	OperationListConnectors      = "/agentrun.v1.AgentRun/ListConnectors"
	OperationGetConnector        = "/agentrun.v1.AgentRun/GetConnector"
	OperationPatchConnector      = "/agentrun.v1.AgentRun/PatchConnector"
	OperationListAgentSchedules  = "/agentrun.v1.AgentRun/ListAgentSchedules"
	OperationGetAgentSchedule    = "/agentrun.v1.AgentRun/GetAgentSchedule"
	OperationPutAgentSchedule    = "/agentrun.v1.AgentRun/PutAgentSchedule"
	OperationPatchAgentSchedule  = "/agentrun.v1.AgentRun/PatchAgentSchedule"
	OperationListAgentExecutions = "/agentrun.v1.AgentRun/ListAgentExecutions"
)

const (
	CollectorRunsPath   = "/api/v1/collector/runs"
	AdminAPIPrefix      = "/api/admin/v1"
	MaxCollectorPrompt  = 64 * 1024
	MaxAgentInput       = 64 * 1024
	MaxRequestBody      = MaxCollectorPrompt*6 + 4096
	MaxAdminRequestBody = 128 * 1024
)

var ErrInvalidRequest = errors.New("invalid AgentRun API request")

type AgentRunHTTPServer interface {
	CreateCollectorRun(context.Context, *CreateCollectorRunRequest) (*CollectorRunResult, error)
	GetCollectorRun(context.Context, *GetCollectorRunRequest) (*CollectorRunResult, error)
	ListModelProviders(context.Context, *ListModelProvidersRequest) (*ModelProviderList, error)
	GetModelProvider(context.Context, *GetModelProviderRequest) (*ModelProviderConfiguration, error)
	PatchModelProvider(context.Context, *PatchModelProviderRequest) (*ModelProviderConfiguration, error)
	ListConnectors(context.Context, *ListConnectorsRequest) (*ConnectorList, error)
	GetConnector(context.Context, *GetConnectorRequest) (*ConnectorConfiguration, error)
	PatchConnector(context.Context, *PatchConnectorRequest) (*ConnectorConfiguration, error)
	ListAgentSchedules(context.Context, *ListAgentSchedulesRequest) (*AgentScheduleList, error)
	GetAgentSchedule(context.Context, *GetAgentScheduleRequest) (*AgentSchedule, error)
	PutAgentSchedule(context.Context, *PutAgentScheduleRequest) (*AgentSchedule, error)
	PatchAgentSchedule(context.Context, *PatchAgentScheduleRequest) (*AgentSchedule, error)
	ListAgentExecutions(context.Context, *ListAgentExecutionsRequest) (*AgentExecutionPage, error)
}

type PublicError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func (e *PublicError) Error() string {
	return e.Code
}

func NewPublicError(status int, code, message string, details any) error {
	if details == nil {
		details = map[string]any{}
	}
	return &PublicError{Status: status, Code: code, Message: message, Details: details}
}

func InvalidRequest(code, message string) error {
	return NewPublicError(http.StatusBadRequest, code, message, nil)
}

type CreateCollectorRunRequest struct {
	IdempotencyKey string `json:"-"`
	Prompt         string `json:"prompt"`
}

type GetCollectorRunRequest struct {
	ExecutionID string
}

type ListModelProvidersRequest struct{}

type GetModelProviderRequest struct {
	ProviderKey string
}

type PatchModelProviderRequest struct {
	ProviderKey string
	BaseURL     OptionalString `json:"base_url"`
	Model       OptionalString `json:"model"`
	APIKey      OptionalString `json:"api_key"`
}

type ListConnectorsRequest struct{}

type GetConnectorRequest struct {
	ConnectorKey string
}

type PatchConnectorRequest struct {
	ConnectorKey string
	BaseURL      OptionalString `json:"base_url"`
	APIKey       OptionalString `json:"api_key"`
}

type ListAgentSchedulesRequest struct{}

type GetAgentScheduleRequest struct {
	AgentKey string
}

type PutAgentScheduleRequest struct {
	AgentKey       string          `json:"-"`
	AgentVersion   string          `json:"agent_version"`
	ScheduleType   string          `json:"schedule_type"`
	CronExpression string          `json:"cron_expression"`
	DailyTimes     []string        `json:"daily_times"`
	Input          json.RawMessage `json:"input"`
	Enabled        *bool           `json:"enabled"`
}

type PatchAgentScheduleRequest struct {
	AgentKey       string           `json:"-"`
	AgentVersion   *string          `json:"agent_version"`
	ScheduleType   *string          `json:"schedule_type"`
	CronExpression *string          `json:"cron_expression"`
	DailyTimes     *[]string        `json:"daily_times"`
	Input          *json.RawMessage `json:"input"`
	Enabled        *bool            `json:"enabled"`
}

type ListAgentExecutionsRequest struct {
	AgentKey  string
	Page      int
	PageSize  int
	SortOrder string
}

type OptionalString struct {
	Set   bool
	Value string
}

func (o *OptionalString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return ErrInvalidRequest
	}
	if err := json.Unmarshal(data, &o.Value); err != nil {
		return err
	}
	o.Set = true
	return nil
}

func (o OptionalString) Pointer() *string {
	if !o.Set {
		return nil
	}
	value := o.Value
	return &value
}

type ModelProviderList struct {
	Items []ModelProviderConfiguration `json:"items"`
}

type ModelProviderConfiguration struct {
	ProviderKey   string     `json:"provider_key"`
	BaseURL       string     `json:"base_url"`
	Model         string     `json:"model"`
	Configured    bool       `json:"configured"`
	KeyConfigured bool       `json:"key_configured"`
	MaskedKey     string     `json:"masked_key,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type ConnectorList struct {
	Items []ConnectorConfiguration `json:"items"`
}

type ConnectorConfiguration struct {
	ConnectorKey  string     `json:"connector_key"`
	BaseURL       string     `json:"base_url"`
	Configured    bool       `json:"configured"`
	KeyConfigured bool       `json:"key_configured"`
	MaskedKey     string     `json:"masked_key,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type AgentScheduleList struct {
	Items []AgentSchedule `json:"items"`
}

type AgentSchedule struct {
	ScheduleID     string          `json:"schedule_id"`
	AgentKey       string          `json:"agent_key"`
	AgentVersion   string          `json:"agent_version"`
	ScheduleType   string          `json:"schedule_type"`
	CronExpression string          `json:"cron_expression,omitempty"`
	DailyTimes     []string        `json:"daily_times,omitempty"`
	Input          json.RawMessage `json:"input"`
	Enabled        bool            `json:"enabled"`
	LastTriggered  *time.Time      `json:"last_triggered_at,omitempty"`
	NextRun        *time.Time      `json:"next_run_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type AgentExecutionPage struct {
	Items      []AgentExecutionListItem `json:"items"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalItems int                      `json:"total_items"`
	TotalPages int                      `json:"total_pages"`
}

type AgentExecutionListItem struct {
	ExecutionID          string     `json:"execution_id"`
	AgentKey             string     `json:"agent_key"`
	AgentVersion         string     `json:"agent_version"`
	TriggerSource        string     `json:"trigger_source"`
	ScheduleID           string     `json:"schedule_id,omitempty"`
	Status               string     `json:"status"`
	ErrorCode            string     `json:"error_code,omitempty"`
	ErrorSummary         string     `json:"error_summary,omitempty"`
	StopReason           string     `json:"stop_reason,omitempty"`
	BlockedByExecutionID string     `json:"blocked_by_execution_id,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	TriggeredAt          time.Time  `json:"triggered_at"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
}

type CollectorRunResult struct {
	Schema               string                `json:"schema"`
	AgentKey             string                `json:"agent_key"`
	AgentVersion         string                `json:"agent_version"`
	ExecutionID          string                `json:"execution_id"`
	Status               string                `json:"status"`
	StatusURL            string                `json:"status_url"`
	PromptSHA256         string                `json:"prompt_sha256"`
	PromptBytes          int                   `json:"prompt_bytes"`
	Invocations          []ConnectorInvocation `json:"invocations"`
	CandidateCounts      map[string]int        `json:"candidate_counts"`
	Artifacts            map[string]string     `json:"artifacts"`
	CreatedAt            time.Time             `json:"created_at"`
	StartedAt            *time.Time            `json:"started_at"`
	CompletedAt          *time.Time            `json:"completed_at"`
	ErrorCode            string                `json:"error_code"`
	ErrorSummary         string                `json:"error_summary"`
	StopReason           string                `json:"stop_reason"`
	BlockedByExecutionID string                `json:"blocked_by_execution_id"`
}

type ConnectorInvocation struct {
	ConnectorKey string     `json:"connector_key"`
	Status       string     `json:"status"`
	ResultCount  int        `json:"result_count"`
	ErrorCode    string     `json:"error_code,omitempty"`
	ErrorSummary string     `json:"error_summary,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}
