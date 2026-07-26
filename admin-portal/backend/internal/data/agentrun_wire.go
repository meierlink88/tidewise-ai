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
