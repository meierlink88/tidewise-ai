package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const (
	OperationCreateCollectorRun            = "/agentrun.v1.AgentRun/CreateCollectorRun"
	OperationGetCollectorRun               = "/agentrun.v1.AgentRun/GetCollectorRun"
	OperationCreateEventSemanticReanalysis = "/agentrun.v1.AgentRun/CreateEventSemanticReanalysis"
	OperationListModelProviders            = "/agentrun.v1.AgentRun/ListModelProviders"
	OperationGetModelProvider              = "/agentrun.v1.AgentRun/GetModelProvider"
	OperationPatchModelProvider            = "/agentrun.v1.AgentRun/PatchModelProvider"
	OperationListConnectors                = "/agentrun.v1.AgentRun/ListConnectors"
	OperationGetConnector                  = "/agentrun.v1.AgentRun/GetConnector"
	OperationPatchConnector                = "/agentrun.v1.AgentRun/PatchConnector"
	OperationListAgentSchedules            = "/agentrun.v1.AgentRun/ListAgentSchedules"
	OperationGetAgentSchedule              = "/agentrun.v1.AgentRun/GetAgentSchedule"
	OperationPutAgentSchedule              = "/agentrun.v1.AgentRun/PutAgentSchedule"
	OperationPatchAgentSchedule            = "/agentrun.v1.AgentRun/PatchAgentSchedule"
	OperationListAgentExecutions           = "/agentrun.v1.AgentRun/ListAgentExecutions"
	OperationListAgentStatuses             = "/agentrun.v1.AgentRun/ListAgentStatuses"
	OperationGetMonitoringSummary          = "/agentrun.v1.AgentRun/GetMonitoringSummary"
	OperationListCollectorMonitoring       = "/agentrun.v1.AgentRun/ListCollectorMonitoring"
	OperationListArtifactMonitoring        = "/agentrun.v1.AgentRun/ListArtifactMonitoring"
	OperationListSemanticMonitoring        = "/agentrun.v1.AgentRun/ListSemanticMonitoring"
)

const (
	CollectorRunsPath           = "/api/v1/collector/runs"
	EventSemanticReanalysisPath = "/api/agentrun/v1/event-semantic-reanalysis"
	AdminAPIPrefix              = "/api/admin/v1"
	MaxCollectorPrompt          = 64 * 1024
	MaxAgentInput               = 64 * 1024
	MaxRequestBody              = MaxCollectorPrompt*6 + 4096
	MaxAdminRequestBody         = 128 * 1024
)

var ErrInvalidRequest = errors.New("invalid AgentRun API request")

type AgentRunHTTPServer interface {
	CreateCollectorRun(context.Context, *CreateCollectorSubmissionRequest) (*CollectorSubmissionResult, error)
	GetCollectorRun(context.Context, *GetCollectorSubmissionRequest) (*CollectorSubmissionResult, error)
	CreateEventSemanticReanalysis(context.Context, *CreateEventSemanticReanalysisRequest) (*EventSemanticWorkItem, error)
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
	ListAgentStatuses(context.Context, *ListAgentStatusesRequest) (*AgentStatusList, error)
	GetMonitoringSummary(context.Context, *MonitoringSummaryRequest) (*MonitoringSummary, error)
	ListCollectorMonitoring(context.Context, *MonitoringListRequest) (*CollectorMonitoringPage, error)
	ListArtifactMonitoring(context.Context, *MonitoringListRequest) (*ArtifactMonitoringPage, error)
	ListSemanticMonitoring(context.Context, *MonitoringListRequest) (*SemanticMonitoringPage, error)
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

type CreateCollectorSubmissionRequest struct {
	IdempotencyKey string `json:"-"`
	Prompt         string `json:"prompt"`
}

type GetCollectorSubmissionRequest struct {
	ExecutionID string
}

type CreateEventSemanticReanalysisRequest struct {
	IdempotencyKey         string `json:"-"`
	EventID                string `json:"event_id"`
	SupersedesSubmissionID string `json:"supersedes_submission_id"`
	Reason                 string `json:"reason"`
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

type ListAgentStatusesRequest struct{}

type MonitoringSummaryRequest struct{ Window string }

type MonitoringListRequest struct {
	Window   string
	State    string
	Page     int
	PageSize int
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

type AgentStatusList struct {
	Items []AgentStatus `json:"items"`
}

type AgentStatus struct {
	AgentKey               string    `json:"agent_key"`
	DisplayName            string    `json:"display_name"`
	CurrentVersion         string    `json:"current_version"`
	IsWorking              bool      `json:"is_working"`
	CurrentExecutionStatus string    `json:"current_execution_status"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type MonitoringCounts struct {
	Success int `json:"success"`
	Running int `json:"running"`
	Failure int `json:"failure"`
}

type MonitoringSummary struct {
	Window                     string           `json:"window"`
	GeneratedAt                time.Time        `json:"generated_at"`
	Collector                  MonitoringCounts `json:"collector"`
	ArtifactExtraction         MonitoringCounts `json:"artifact_extraction"`
	Semantic                   MonitoringCounts `json:"semantic"`
	CollectorRawResults        int              `json:"collector_raw_results"`
	CollectorMergedResults     int              `json:"collector_merged_results"`
	CollectorAcceptedArtifacts int              `json:"collector_accepted_artifacts"`
	ArtifactPublished          int              `json:"artifact_published"`
	ArtifactNoEvents           int              `json:"artifact_no_events"`
	ArtifactFormalEvents       int              `json:"artifact_formal_events"`
	SemanticSubmissions        int              `json:"semantic_submissions"`
	SemanticAcceptedCandidates int              `json:"semantic_accepted_candidates"`
	SemanticRejectedCandidates int              `json:"semantic_rejected_candidates"`
}

type MonitoringPage struct {
	Window      string    `json:"window"`
	GeneratedAt time.Time `json:"generated_at"`
	Page        int       `json:"page"`
	PageSize    int       `json:"page_size"`
	TotalItems  int       `json:"total_items"`
	TotalPages  int       `json:"total_pages"`
}

type CollectorMonitoringPage struct {
	Items []CollectorMonitoringItem `json:"items"`
	MonitoringPage
}

type CollectorMonitoringItem struct {
	ExecutionID       string     `json:"execution_id"`
	State             string     `json:"state"`
	RawStatus         string     `json:"raw_status"`
	TriggerSource     string     `json:"trigger_source"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	DurationMs        *int64     `json:"duration_ms,omitempty"`
	RawResults        int        `json:"raw_results"`
	MergedResults     int        `json:"merged_results"`
	AcceptedArtifacts int        `json:"accepted_artifacts"`
	ErrorCode         string     `json:"error_code,omitempty"`
}

type ArtifactMonitoringPage struct {
	Items []ArtifactMonitoringItem `json:"items"`
	MonitoringPage
}

type ArtifactMonitoringItem struct {
	ExtractionKey        string     `json:"extraction_key"`
	ArtifactID           string     `json:"artifact_id"`
	CollectorExecutionID string     `json:"collector_execution_id"`
	State                string     `json:"state"`
	RawStatus            string     `json:"raw_status"`
	UpdatedAt            time.Time  `json:"updated_at"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	DurationMs           *int64     `json:"duration_ms,omitempty"`
	EventCandidates      int        `json:"event_candidates"`
	AcknowledgedJournals int        `json:"acknowledged_journals"`
	TotalJournals        int        `json:"total_journals"`
	ErrorCode            string     `json:"error_code,omitempty"`
}

type SemanticMonitoringPage struct {
	Items []SemanticMonitoringItem `json:"items"`
	MonitoringPage
}

type SemanticMonitoringItem struct {
	WorkItemID         string     `json:"work_item_id"`
	EventID            string     `json:"event_id"`
	TriggerSource      string     `json:"trigger_source"`
	State              string     `json:"state"`
	RawStatus          string     `json:"raw_status"`
	UpdatedAt          time.Time  `json:"updated_at"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	DurationMs         *int64     `json:"duration_ms,omitempty"`
	AttemptCount       int        `json:"attempt_count"`
	MaxAttempts        int        `json:"max_attempts"`
	AcceptedCandidates int        `json:"accepted_candidates"`
	RejectedCandidates int        `json:"rejected_candidates"`
	ErrorCode          string     `json:"error_code,omitempty"`
}

type EventSemanticWorkItem struct {
	WorkItemID             string    `json:"work_item_id"`
	EventID                string    `json:"event_id"`
	SupersedesSubmissionID string    `json:"supersedes_submission_id"`
	Status                 string    `json:"status"`
	Replayed               bool      `json:"replayed"`
	CreatedAt              time.Time `json:"created_at"`
}

type CollectorSubmissionResult struct {
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
