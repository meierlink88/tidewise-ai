package v1

import (
	"context"
	"encoding/json"
	"time"
)

const APIPrefix = "/api/admin/v1"

const (
	OperationListRawDocuments        = "admin.rawDocuments.list"
	OperationListEvents              = "admin.events.list"
	OperationGetAgentSchedule        = "admin.agentSchedules.get"
	OperationSaveAgentSchedule       = "admin.agentSchedules.save"
	OperationSetScheduleEnabled      = "admin.agentSchedules.setEnabled"
	OperationListAgentExecutions     = "admin.agentExecutions.list"
	OperationListAgentStatuses       = "admin.agentStatuses.list"
	OperationGetMonitoringSummary    = "admin.monitoring.summary"
	OperationListCollectorMonitoring = "admin.monitoring.collector.list"
	OperationListArtifactMonitoring  = "admin.monitoring.artifact.list"
	OperationListSemanticMonitoring  = "admin.monitoring.semantic.list"
	OperationListModelProviders      = "admin.modelProviders.list"
	OperationGetModelProvider        = "admin.modelProviders.get"
	OperationPatchModelProvider      = "admin.modelProviders.patch"
	OperationListConnectors          = "admin.connectors.list"
	OperationGetConnector            = "admin.connectors.get"
	OperationPatchConnector          = "admin.connectors.patch"
)

type AdminHTTPServer interface {
	ListRawDocuments(context.Context, *ListRawDocumentsRequest) (*RawDocumentListResponse, error)
	ListEvents(context.Context, *ListEventsRequest) (*EventListResponse, error)
	GetAgentSchedule(context.Context, *AgentKeyRequest) (*AgentSchedule, error)
	SaveAgentSchedule(context.Context, *SaveAgentScheduleRequest) (*AgentSchedule, error)
	SetAgentScheduleEnabled(context.Context, *SetAgentScheduleEnabledRequest) (*AgentSchedule, error)
	ListAgentExecutions(context.Context, *ListAgentExecutionsRequest) (*AgentExecutionPage, error)
	ListAgentStatuses(context.Context, *EmptyRequest) (*AgentStatusListResponse, error)
	GetMonitoringSummary(context.Context, *MonitoringSummaryRequest) (*MonitoringSummary, error)
	ListCollectorMonitoring(context.Context, *MonitoringListRequest) (*CollectorMonitoringPage, error)
	ListArtifactMonitoring(context.Context, *MonitoringListRequest) (*ArtifactMonitoringPage, error)
	ListSemanticMonitoring(context.Context, *MonitoringListRequest) (*SemanticMonitoringPage, error)
	ListModelProviders(context.Context, *EmptyRequest) (*ModelProviderListResponse, error)
	GetModelProvider(context.Context, *ProviderKeyRequest) (*ModelProviderConfiguration, error)
	PatchModelProvider(context.Context, *PatchModelProviderRequest) (*ModelProviderConfiguration, error)
	ListConnectors(context.Context, *EmptyRequest) (*ConnectorListResponse, error)
	GetConnector(context.Context, *ConnectorKeyRequest) (*ConnectorConfiguration, error)
	PatchConnector(context.Context, *PatchConnectorRequest) (*ConnectorConfiguration, error)
}

type EmptyRequest struct{}

type ListRawDocumentsRequest struct {
	Title     string
	SourceRef string
	Page      int
	PageSize  int
}

type ListEventsRequest struct {
	Title         string
	EventStatus   string
	FactStatus    string
	EventTimeFrom string
	EventTimeTo   string
	FirstSeenFrom string
	FirstSeenTo   string
	Page          int
	PageSize      int
}

type AgentKeyRequest struct {
	AgentKey string `json:"-"`
}

type SaveAgentScheduleRequest struct {
	AgentKey       string          `json:"-"`
	AgentVersion   string          `json:"agent_version"`
	ScheduleType   string          `json:"schedule_type"`
	CronExpression string          `json:"cron_expression"`
	DailyTimes     []string        `json:"daily_times"`
	Input          json.RawMessage `json:"input"`
}

type SetAgentScheduleEnabledRequest struct {
	AgentKey string `json:"-"`
	Enabled  *bool  `json:"enabled"`
}

type ListAgentExecutionsRequest struct {
	Page int
}
type MonitoringSummaryRequest struct{ Window string }
type MonitoringListRequest struct {
	Window, State  string
	Page, PageSize int
}

type ProviderKeyRequest struct {
	ProviderKey string `json:"-"`
}

type PatchModelProviderRequest struct {
	ProviderKey string  `json:"-"`
	BaseURL     *string `json:"base_url"`
	Model       *string `json:"model"`
	APIKey      *string `json:"api_key"`
}

type ConnectorKeyRequest struct {
	ConnectorKey string `json:"-"`
}

type PatchConnectorRequest struct {
	ConnectorKey string  `json:"-"`
	BaseURL      *string `json:"base_url"`
	APIKey       *string `json:"api_key"`
}

type RawDocumentListResponse struct {
	Items    []RawDocument `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

type RawDocument struct {
	ID               string `json:"id"`
	ContractVersion  int    `json:"contract_version"`
	ArtifactID       string `json:"artifact_id,omitempty"`
	SourceRef        string `json:"source_ref,omitempty"`
	IngestChannel    string `json:"ingest_channel"`
	SourceType       string `json:"source_type"`
	SourceName       string `json:"source_name"`
	SourceURL        string `json:"source_url"`
	SourceExternalID string `json:"source_external_id,omitempty"`
	Title            string `json:"title"`
	ContentText      string `json:"content_text"`
	RawObjectURI     string `json:"raw_object_uri"`
	RawMIMEType      string `json:"raw_mime_type"`
	Language         string `json:"language"`
	PublishedAt      string `json:"published_at,omitempty"`
	CollectedAt      string `json:"collected_at"`
	IngestStatus     string `json:"ingest_status"`
	ContentSHA256    string `json:"content_sha256"`
}

type EventListResponse struct {
	Items    []Event `json:"items"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

type Event struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	EventTime   string `json:"event_time,omitempty"`
	FirstSeenAt string `json:"first_seen_at"`
	KnowableAt  string `json:"knowable_at,omitempty"`
	EventStatus string `json:"event_status"`
	FactStatus  string `json:"fact_status"`
	DedupeKey   string `json:"dedupe_key"`
}

type AgentSchedule struct {
	ID             string          `json:"schedule_id"`
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

type AgentStatusListResponse struct {
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
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
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
	AttemptCount       int        `json:"attempt_count"`
	MaxAttempts        int        `json:"max_attempts"`
	AcceptedCandidates int        `json:"accepted_candidates"`
	RejectedCandidates int        `json:"rejected_candidates"`
	ErrorCode          string     `json:"error_code,omitempty"`
}

type ModelProviderListResponse struct {
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

type ConnectorListResponse struct {
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
