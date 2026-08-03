package biz

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type AgentRunRepo interface {
	GetAgentSchedule(context.Context, string) (AgentSchedule, error)
	PutAgentSchedule(context.Context, string, PutAgentScheduleInput) (AgentSchedule, error)
	PatchAgentSchedule(context.Context, string, PatchAgentScheduleInput) (AgentSchedule, error)
	ListAgentExecutions(context.Context, AgentExecutionQuery) (AgentExecutionPage, error)
	ListAgentStatuses(context.Context) ([]AgentStatus, error)
	GetMonitoringSummary(context.Context, string) (MonitoringSummary, error)
	ListCollectorMonitoring(context.Context, MonitoringQuery) (CollectorMonitoringPage, error)
	ListArtifactMonitoring(context.Context, MonitoringQuery) (ArtifactMonitoringPage, error)
	ListSemanticMonitoring(context.Context, MonitoringQuery) (SemanticMonitoringPage, error)
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
	ID             string
	AgentKey       string
	AgentVersion   string
	ScheduleType   ScheduleType
	CronExpression string
	DailyTimes     []string
	Input          json.RawMessage
	Enabled        bool
	LastTriggered  *time.Time
	NextRun        *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PutAgentScheduleInput struct {
	AgentVersion   string
	ScheduleType   ScheduleType
	CronExpression string
	DailyTimes     []string
	Input          json.RawMessage
	Enabled        bool
}

type PatchAgentScheduleInput struct {
	AgentVersion   *string
	ScheduleType   *ScheduleType
	CronExpression *string
	DailyTimes     *[]string
	Input          *json.RawMessage
	Enabled        *bool
}

type AgentExecutionQuery struct {
	AgentKey string
	Page     int
	PageSize int
}

type AgentExecutionPage struct {
	Items      []AgentExecution
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
}

type AgentExecution struct {
	ID                   string
	AgentKey             string
	AgentVersion         string
	TriggerSource        string
	ScheduleID           string
	Status               string
	ErrorCode            string
	ErrorSummary         string
	StopReason           string
	BlockedByExecutionID string
	CreatedAt            time.Time
	TriggeredAt          time.Time
	StartedAt            *time.Time
	CompletedAt          *time.Time
}

type AgentStatus struct {
	AgentKey               string
	DisplayName            string
	CurrentVersion         string
	IsWorking              bool
	CurrentExecutionStatus string
	UpdatedAt              time.Time
}

type MonitoringQuery struct {
	Window, State  string
	Page, PageSize int
}
type MonitoringCounts struct{ Success, Running, Failure int }
type MonitoringSummary struct {
	Window                                                                      string
	GeneratedAt                                                                 time.Time
	Collector, ArtifactExtraction, Semantic                                     MonitoringCounts
	CollectorRawResults, CollectorMergedResults, CollectorAcceptedArtifacts     int
	ArtifactPublished, ArtifactNoEvents, ArtifactFormalEvents                   int
	SemanticSubmissions, SemanticAcceptedCandidates, SemanticRejectedCandidates int
}
type MonitoringPage struct{ Page, PageSize, TotalItems, TotalPages int }
type CollectorMonitoringPage struct {
	Items []CollectorMonitoringItem
	MonitoringPage
}
type CollectorMonitoringItem struct {
	ExecutionID, State, RawStatus, TriggerSource, ErrorCode string
	StartedAt, CompletedAt                                  *time.Time
	RawResults, MergedResults, AcceptedArtifacts            int
}
type ArtifactMonitoringPage struct {
	Items []ArtifactMonitoringItem
	MonitoringPage
}
type ArtifactMonitoringItem struct {
	ExtractionKey, ArtifactID, CollectorExecutionID, State, RawStatus, ErrorCode string
	UpdatedAt                                                                    time.Time
	StartedAt, CompletedAt                                                       *time.Time
	EventCandidates, AcknowledgedJournals, TotalJournals                         int
}
type SemanticMonitoringPage struct {
	Items []SemanticMonitoringItem
	MonitoringPage
}
type SemanticMonitoringItem struct {
	WorkItemID, EventID, TriggerSource, State, RawStatus, ErrorCode   string
	UpdatedAt                                                         time.Time
	StartedAt, CompletedAt                                            *time.Time
	AttemptCount, MaxAttempts, AcceptedCandidates, RejectedCandidates int
}

type ModelProviderConfiguration struct {
	ProviderKey   string
	BaseURL       string
	Model         string
	Configured    bool
	KeyConfigured bool
	MaskedKey     string
	UpdatedAt     *time.Time
}

type ModelProviderPatch struct {
	BaseURL *string
	Model   *string
	APIKey  *string
}

type ConnectorConfiguration struct {
	ConnectorKey  string
	BaseURL       string
	Configured    bool
	KeyConfigured bool
	MaskedKey     string
	UpdatedAt     *time.Time
}

type ConnectorPatch struct {
	BaseURL *string
	APIKey  *string
}

var (
	ErrAgentRunInvalidRequest = errors.New("AgentRun rejected the request")
	ErrAgentRunAuthentication = errors.New("AgentRun authentication failed")
	ErrAgentRunNotFound       = errors.New("AgentRun resource not found")
	ErrAgentRunConflict       = errors.New("AgentRun conflict")
	ErrAgentRunUnavailable    = errors.New("AgentRun unavailable")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrAgentRunNotFound)
}
