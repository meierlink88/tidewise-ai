package data

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

type agentScheduleWire struct {
	ID             string           `json:"schedule_id"`
	AgentKey       string           `json:"agent_key"`
	AgentVersion   string           `json:"agent_version"`
	ScheduleType   biz.ScheduleType `json:"schedule_type"`
	CronExpression string           `json:"cron_expression,omitempty"`
	DailyTimes     []string         `json:"daily_times,omitempty"`
	Input          json.RawMessage  `json:"input"`
	Enabled        bool             `json:"enabled"`
	LastTriggered  *time.Time       `json:"last_triggered_at,omitempty"`
	NextRun        *time.Time       `json:"next_run_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

func (w agentScheduleWire) toBiz() (biz.AgentSchedule, error) {
	var inputObject map[string]json.RawMessage
	if strings.TrimSpace(w.ID) == "" || strings.TrimSpace(w.AgentKey) == "" ||
		strings.TrimSpace(w.AgentVersion) == "" || json.Unmarshal(w.Input, &inputObject) != nil ||
		inputObject == nil || w.CreatedAt.IsZero() || w.UpdatedAt.IsZero() ||
		(w.ScheduleType != biz.ScheduleTypeDaily && w.ScheduleType != biz.ScheduleTypeCron) {
		return biz.AgentSchedule{}, biz.ErrAgentRunUnavailable
	}
	return biz.AgentSchedule{
		ID: w.ID, AgentKey: w.AgentKey, AgentVersion: w.AgentVersion,
		ScheduleType: w.ScheduleType, CronExpression: w.CronExpression,
		DailyTimes: append([]string(nil), w.DailyTimes...), Input: append(json.RawMessage(nil), w.Input...),
		Enabled: w.Enabled, LastTriggered: w.LastTriggered, NextRun: w.NextRun,
		CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}, nil
}

type putAgentScheduleWire struct {
	AgentVersion   string           `json:"agent_version"`
	ScheduleType   biz.ScheduleType `json:"schedule_type"`
	CronExpression string           `json:"cron_expression,omitempty"`
	DailyTimes     []string         `json:"daily_times,omitempty"`
	Input          json.RawMessage  `json:"input"`
	Enabled        bool             `json:"enabled"`
}

func newPutAgentScheduleWire(input biz.PutAgentScheduleInput) putAgentScheduleWire {
	return putAgentScheduleWire{
		AgentVersion: input.AgentVersion, ScheduleType: input.ScheduleType,
		CronExpression: input.CronExpression, DailyTimes: input.DailyTimes,
		Input: input.Input, Enabled: input.Enabled,
	}
}

type patchAgentScheduleWire struct {
	AgentVersion   *string           `json:"agent_version,omitempty"`
	ScheduleType   *biz.ScheduleType `json:"schedule_type,omitempty"`
	CronExpression *string           `json:"cron_expression,omitempty"`
	DailyTimes     *[]string         `json:"daily_times,omitempty"`
	Input          *json.RawMessage  `json:"input,omitempty"`
	Enabled        *bool             `json:"enabled,omitempty"`
}

func newPatchAgentScheduleWire(input biz.PatchAgentScheduleInput) patchAgentScheduleWire {
	return patchAgentScheduleWire(input)
}

type agentExecutionPageWire struct {
	Items      []agentExecutionWire `json:"items"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalItems int                  `json:"total_items"`
	TotalPages int                  `json:"total_pages"`
}

func (w agentExecutionPageWire) toBiz() (biz.AgentExecutionPage, error) {
	if w.Page < 1 || w.PageSize != 20 || w.TotalItems < 0 || w.TotalPages < 0 {
		return biz.AgentExecutionPage{}, biz.ErrAgentRunUnavailable
	}
	items := make([]biz.AgentExecution, 0, len(w.Items))
	for _, item := range w.Items {
		mapped, err := item.toBiz()
		if err != nil {
			return biz.AgentExecutionPage{}, err
		}
		items = append(items, mapped)
	}
	return biz.AgentExecutionPage{
		Items: items, Page: w.Page, PageSize: w.PageSize,
		TotalItems: w.TotalItems, TotalPages: w.TotalPages,
	}, nil
}

type agentExecutionWire struct {
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

type agentStatusListWire struct {
	Items []agentStatusWire `json:"items"`
}

type agentStatusWire struct {
	AgentKey               string    `json:"agent_key"`
	DisplayName            string    `json:"display_name"`
	CurrentVersion         string    `json:"current_version"`
	IsWorking              bool      `json:"is_working"`
	CurrentExecutionStatus string    `json:"current_execution_status"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type monitoringCountsWire struct {
	Success int `json:"success"`
	Running int `json:"running"`
	Failure int `json:"failure"`
}
type monitoringSummaryWire struct {
	Window                     string               `json:"window"`
	GeneratedAt                time.Time            `json:"generated_at"`
	Collector                  monitoringCountsWire `json:"collector"`
	ArtifactExtraction         monitoringCountsWire `json:"artifact_extraction"`
	Semantic                   monitoringCountsWire `json:"semantic"`
	CollectorRawResults        int                  `json:"collector_raw_results"`
	CollectorMergedResults     int                  `json:"collector_merged_results"`
	CollectorAcceptedArtifacts int                  `json:"collector_accepted_artifacts"`
	ArtifactPublished          int                  `json:"artifact_published"`
	ArtifactNoEvents           int                  `json:"artifact_no_events"`
	ArtifactFormalEvents       int                  `json:"artifact_formal_events"`
	SemanticSubmissions        int                  `json:"semantic_submissions"`
	SemanticAcceptedCandidates int                  `json:"semantic_accepted_candidates"`
	SemanticRejectedCandidates int                  `json:"semantic_rejected_candidates"`
}

func (w monitoringSummaryWire) toBiz() (biz.MonitoringSummary, error) {
	if !validMonitoringWindow(w.Window) || w.GeneratedAt.IsZero() ||
		!w.Collector.valid() || !w.ArtifactExtraction.valid() || !w.Semantic.valid() ||
		w.CollectorRawResults < 0 || w.CollectorMergedResults < 0 ||
		w.CollectorAcceptedArtifacts < 0 || w.ArtifactPublished < 0 || w.ArtifactNoEvents < 0 ||
		w.ArtifactFormalEvents < 0 || w.SemanticSubmissions < 0 ||
		w.SemanticAcceptedCandidates < 0 || w.SemanticRejectedCandidates < 0 {
		return biz.MonitoringSummary{}, biz.ErrAgentRunUnavailable
	}
	return biz.MonitoringSummary{
		Window: w.Window, GeneratedAt: w.GeneratedAt,
		Collector: biz.MonitoringCounts(w.Collector), ArtifactExtraction: biz.MonitoringCounts(w.ArtifactExtraction),
		Semantic: biz.MonitoringCounts(w.Semantic), CollectorRawResults: w.CollectorRawResults,
		CollectorMergedResults: w.CollectorMergedResults, CollectorAcceptedArtifacts: w.CollectorAcceptedArtifacts,
		ArtifactPublished: w.ArtifactPublished, ArtifactNoEvents: w.ArtifactNoEvents,
		ArtifactFormalEvents: w.ArtifactFormalEvents, SemanticSubmissions: w.SemanticSubmissions,
		SemanticAcceptedCandidates: w.SemanticAcceptedCandidates,
		SemanticRejectedCandidates: w.SemanticRejectedCandidates,
	}, nil
}

func (w monitoringCountsWire) valid() bool {
	return w.Success >= 0 && w.Running >= 0 && w.Failure >= 0
}

type monitoringPageWire struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

func (w monitoringPageWire) valid() bool {
	return w.Page >= 1 && w.PageSize >= 1 && w.PageSize <= 100 && w.TotalItems >= 0 && w.TotalPages >= 0
}

func validMonitoringWindow(value string) bool {
	return value == "1h" || value == "6h" || value == "12h" || value == "24h"
}

func validMonitoringState(value string) bool {
	return value == "success" || value == "running" || value == "failure"
}

type collectorMonitoringItemWire struct {
	ExecutionID       string     `json:"execution_id"`
	State             string     `json:"state"`
	RawStatus         string     `json:"raw_status"`
	TriggerSource     string     `json:"trigger_source"`
	StartedAt         *time.Time `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	RawResults        int        `json:"raw_results"`
	MergedResults     int        `json:"merged_results"`
	AcceptedArtifacts int        `json:"accepted_artifacts"`
	ErrorCode         string     `json:"error_code"`
}
type collectorMonitoringPageWire struct {
	Items []collectorMonitoringItemWire `json:"items"`
	monitoringPageWire
}
type artifactMonitoringItemWire struct {
	ExtractionKey        string     `json:"extraction_key"`
	ArtifactID           string     `json:"artifact_id"`
	CollectorExecutionID string     `json:"collector_execution_id"`
	State                string     `json:"state"`
	RawStatus            string     `json:"raw_status"`
	UpdatedAt            time.Time  `json:"updated_at"`
	StartedAt            *time.Time `json:"started_at"`
	CompletedAt          *time.Time `json:"completed_at"`
	EventCandidates      int        `json:"event_candidates"`
	AcknowledgedJournals int        `json:"acknowledged_journals"`
	TotalJournals        int        `json:"total_journals"`
	ErrorCode            string     `json:"error_code"`
}
type artifactMonitoringPageWire struct {
	Items []artifactMonitoringItemWire `json:"items"`
	monitoringPageWire
}
type semanticMonitoringItemWire struct {
	WorkItemID         string     `json:"work_item_id"`
	EventID            string     `json:"event_id"`
	TriggerSource      string     `json:"trigger_source"`
	State              string     `json:"state"`
	RawStatus          string     `json:"raw_status"`
	UpdatedAt          time.Time  `json:"updated_at"`
	StartedAt          *time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	AttemptCount       int        `json:"attempt_count"`
	MaxAttempts        int        `json:"max_attempts"`
	AcceptedCandidates int        `json:"accepted_candidates"`
	RejectedCandidates int        `json:"rejected_candidates"`
	ErrorCode          string     `json:"error_code"`
}
type semanticMonitoringPageWire struct {
	Items []semanticMonitoringItemWire `json:"items"`
	monitoringPageWire
}

func (w agentStatusWire) toBiz() (biz.AgentStatus, error) {
	status := strings.TrimSpace(w.CurrentExecutionStatus)
	if strings.TrimSpace(w.AgentKey) == "" || strings.TrimSpace(w.DisplayName) == "" ||
		strings.TrimSpace(w.CurrentVersion) == "" || status == "" || w.UpdatedAt.IsZero() ||
		(w.IsWorking && status == "idle") || (!w.IsWorking && status != "idle") {
		return biz.AgentStatus{}, biz.ErrAgentRunUnavailable
	}
	return biz.AgentStatus{
		AgentKey: w.AgentKey, DisplayName: w.DisplayName, CurrentVersion: w.CurrentVersion,
		IsWorking: w.IsWorking, CurrentExecutionStatus: status, UpdatedAt: w.UpdatedAt,
	}, nil
}

func (w agentExecutionWire) toBiz() (biz.AgentExecution, error) {
	if strings.TrimSpace(w.ID) == "" || strings.TrimSpace(w.AgentKey) == "" ||
		strings.TrimSpace(w.AgentVersion) == "" || strings.TrimSpace(w.TriggerSource) == "" ||
		strings.TrimSpace(w.Status) == "" || w.CreatedAt.IsZero() || w.TriggeredAt.IsZero() {
		return biz.AgentExecution{}, biz.ErrAgentRunUnavailable
	}
	return biz.AgentExecution{
		ID: w.ID, AgentKey: w.AgentKey, AgentVersion: w.AgentVersion,
		TriggerSource: w.TriggerSource, ScheduleID: w.ScheduleID, Status: w.Status,
		ErrorCode: w.ErrorCode, ErrorSummary: w.ErrorSummary, StopReason: w.StopReason,
		BlockedByExecutionID: w.BlockedByExecutionID, CreatedAt: w.CreatedAt,
		TriggeredAt: w.TriggeredAt, StartedAt: w.StartedAt, CompletedAt: w.CompletedAt,
	}, nil
}

type modelProviderWire struct {
	ProviderKey   string     `json:"provider_key"`
	BaseURL       string     `json:"base_url"`
	Model         string     `json:"model"`
	Configured    bool       `json:"configured"`
	KeyConfigured bool       `json:"key_configured"`
	MaskedKey     string     `json:"masked_key,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

func (w modelProviderWire) toBiz() (biz.ModelProviderConfiguration, error) {
	if strings.TrimSpace(w.ProviderKey) == "" || !validConfigurationURL(w.BaseURL, w.Configured) {
		return biz.ModelProviderConfiguration{}, biz.ErrAgentRunUnavailable
	}
	return biz.ModelProviderConfiguration{
		ProviderKey: w.ProviderKey, BaseURL: w.BaseURL, Model: w.Model,
		Configured: w.Configured, KeyConfigured: w.KeyConfigured,
		MaskedKey: w.MaskedKey, UpdatedAt: w.UpdatedAt,
	}, nil
}

type modelProviderListWire struct {
	Items []modelProviderWire `json:"items"`
}

type modelProviderPatchWire struct {
	BaseURL *string `json:"base_url,omitempty"`
	Model   *string `json:"model,omitempty"`
	APIKey  *string `json:"api_key,omitempty"`
}

type connectorWire struct {
	ConnectorKey  string     `json:"connector_key"`
	BaseURL       string     `json:"base_url"`
	Configured    bool       `json:"configured"`
	KeyConfigured bool       `json:"key_configured"`
	MaskedKey     string     `json:"masked_key,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

func (w connectorWire) toBiz() (biz.ConnectorConfiguration, error) {
	if strings.TrimSpace(w.ConnectorKey) == "" || !validConfigurationURL(w.BaseURL, w.Configured) {
		return biz.ConnectorConfiguration{}, biz.ErrAgentRunUnavailable
	}
	return biz.ConnectorConfiguration{
		ConnectorKey: w.ConnectorKey, BaseURL: w.BaseURL,
		Configured: w.Configured, KeyConfigured: w.KeyConfigured,
		MaskedKey: w.MaskedKey, UpdatedAt: w.UpdatedAt,
	}, nil
}

type connectorListWire struct {
	Items []connectorWire `json:"items"`
}

type connectorPatchWire struct {
	BaseURL *string `json:"base_url,omitempty"`
	APIKey  *string `json:"api_key,omitempty"`
}

func agentRunBusinessError(status int) error {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return biz.ErrAgentRunInvalidRequest
	case http.StatusUnauthorized, http.StatusForbidden:
		return biz.ErrAgentRunAuthentication
	case http.StatusNotFound:
		return biz.ErrAgentRunNotFound
	case http.StatusConflict:
		return biz.ErrAgentRunConflict
	default:
		return biz.ErrAgentRunUnavailable
	}
}

func validAbsoluteHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func validConfigurationURL(value string, configured bool) bool {
	if strings.TrimSpace(value) == "" {
		return !configured
	}
	return validAbsoluteHTTPURL(value)
}
