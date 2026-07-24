package agentrunclient

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const AdminAPIPrefix = "/api/admin/v1"

type Client interface {
	GetAgentSchedule(context.Context, string) (AgentSchedule, error)
	PutAgentSchedule(context.Context, string, PutAgentScheduleInput) (AgentSchedule, error)
	PatchAgentSchedule(context.Context, string, PatchAgentScheduleInput) (AgentSchedule, error)
	ListAgentExecutions(context.Context, AgentExecutionQuery) (AgentExecutionPage, error)
	ListModelProviders(context.Context) ([]ModelProviderConfiguration, error)
	GetModelProvider(context.Context, string) (ModelProviderConfiguration, error)
	PatchModelProvider(context.Context, string, ModelProviderPatch) (ModelProviderConfiguration, error)
	ListConnectors(context.Context) ([]ConnectorConfiguration, error)
	GetConnector(context.Context, string) (ConnectorConfiguration, error)
	PatchConnector(context.Context, string, ConnectorPatch) (ConnectorConfiguration, error)
}

type ScheduleType string

const (
	ScheduleTypeCron  ScheduleType = "cron"
	ScheduleTypeDaily ScheduleType = "daily"
)

type AgentSchedule struct {
	ID             string          `json:"schedule_id"`
	AgentKey       string          `json:"agent_key"`
	AgentVersion   string          `json:"agent_version"`
	ScheduleType   ScheduleType    `json:"schedule_type"`
	CronExpression string          `json:"cron_expression,omitempty"`
	DailyTimes     []string        `json:"daily_times,omitempty"`
	Input          json.RawMessage `json:"input"`
	Enabled        bool            `json:"enabled"`
	LastTriggered  *time.Time      `json:"last_triggered_at,omitempty"`
	NextRun        *time.Time      `json:"next_run_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type PutAgentScheduleInput struct {
	AgentVersion   string          `json:"agent_version"`
	ScheduleType   ScheduleType    `json:"schedule_type"`
	CronExpression string          `json:"cron_expression,omitempty"`
	DailyTimes     []string        `json:"daily_times,omitempty"`
	Input          json.RawMessage `json:"input"`
	Enabled        bool            `json:"enabled"`
}

type PatchAgentScheduleInput struct {
	AgentVersion   *string          `json:"agent_version,omitempty"`
	ScheduleType   *ScheduleType    `json:"schedule_type,omitempty"`
	CronExpression *string          `json:"cron_expression,omitempty"`
	DailyTimes     *[]string        `json:"daily_times,omitempty"`
	Input          *json.RawMessage `json:"input,omitempty"`
	Enabled        *bool            `json:"enabled,omitempty"`
}

type AgentExecutionQuery struct {
	AgentKey string
	Page     int
	PageSize int
}

type AgentExecutionPage struct {
	Items      []AgentExecution `json:"items"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalItems int              `json:"total_items"`
	TotalPages int              `json:"total_pages"`
}

type AgentExecution struct {
	ID                   string     `json:"execution_id"`
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

type ModelProviderConfiguration struct {
	ProviderKey   string     `json:"provider_key"`
	BaseURL       string     `json:"base_url"`
	Model         string     `json:"model"`
	Configured    bool       `json:"configured"`
	KeyConfigured bool       `json:"key_configured"`
	MaskedKey     string     `json:"masked_key,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type ModelProviderPatch struct {
	BaseURL *string `json:"base_url,omitempty"`
	Model   *string `json:"model,omitempty"`
	APIKey  *string `json:"api_key,omitempty"`
}

type ConnectorConfiguration struct {
	ConnectorKey  string     `json:"connector_key"`
	BaseURL       string     `json:"base_url"`
	Configured    bool       `json:"configured"`
	KeyConfigured bool       `json:"key_configured"`
	MaskedKey     string     `json:"masked_key,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type ConnectorPatch struct {
	BaseURL *string `json:"base_url,omitempty"`
	APIKey  *string `json:"api_key,omitempty"`
}

type Error struct {
	StatusCode int
	Code       string
}

func (e *Error) Error() string {
	if e == nil {
		return "AgentRun request failed"
	}
	return "AgentRun request failed"
}

func IsNotFound(err error) bool {
	var clientError *Error
	return errors.As(err, &clientError) && clientError.StatusCode == 404
}
