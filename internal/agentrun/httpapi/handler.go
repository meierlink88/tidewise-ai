package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/admin"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/scheduling"
)

const maxAdminRequestBytes = 128 * 1024

type application interface {
	ListModelProviders(context.Context) ([]agentrun.ModelProviderConfigView, error)
	GetModelProvider(context.Context, string) (agentrun.ModelProviderConfigView, error)
	PatchModelProvider(context.Context, string, admin.ModelProviderPatch) (agentrun.ModelProviderConfigView, error)
	ListConnectors(context.Context) ([]agentrun.ConnectorConfigView, error)
	GetConnector(context.Context, string) (agentrun.ConnectorConfigView, error)
	PatchConnector(context.Context, string, admin.ConnectorPatch) (agentrun.ConnectorConfigView, error)
	ListAgentSchedules(context.Context) ([]agentrun.AgentSchedule, error)
	GetAgentSchedule(context.Context, string) (agentrun.AgentSchedule, error)
	PutAgentSchedule(context.Context, agentrun.PutAgentScheduleInput) (agentrun.AgentSchedule, error)
	PatchAgentSchedule(context.Context, string, agentrun.PatchAgentScheduleInput) (agentrun.AgentSchedule, error)
	ListAgentExecutions(context.Context, agentrun.ExecutionListQuery) (agentrun.ExecutionPage, error)
}

type handler struct {
	application application
	adminToken  string
}

func NewAdminHandler(application application, adminToken string) http.Handler {
	h := &handler{application: application, adminToken: adminToken}
	mux := http.NewServeMux()
	mux.Handle("GET /api/admin/v1/model-providers", h.authenticate(http.HandlerFunc(h.listModelProviders)))
	mux.Handle("GET /api/admin/v1/model-providers/{provider_key}", h.authenticate(http.HandlerFunc(h.getModelProvider)))
	mux.Handle("PATCH /api/admin/v1/model-providers/{provider_key}", h.authenticate(http.HandlerFunc(h.patchModelProvider)))
	mux.Handle("GET /api/admin/v1/connectors", h.authenticate(http.HandlerFunc(h.listConnectors)))
	mux.Handle("GET /api/admin/v1/connectors/{connector_key}", h.authenticate(http.HandlerFunc(h.getConnector)))
	mux.Handle("PATCH /api/admin/v1/connectors/{connector_key}", h.authenticate(http.HandlerFunc(h.patchConnector)))
	mux.Handle("GET /api/admin/v1/agent-schedules", h.authenticate(http.HandlerFunc(h.listAgentSchedules)))
	mux.Handle("GET /api/admin/v1/agent-schedules/{agent_key}", h.authenticate(http.HandlerFunc(h.getAgentSchedule)))
	mux.Handle("PUT /api/admin/v1/agent-schedules/{agent_key}", h.authenticate(http.HandlerFunc(h.putAgentSchedule)))
	mux.Handle("PATCH /api/admin/v1/agent-schedules/{agent_key}", h.authenticate(http.HandlerFunc(h.patchAgentSchedule)))
	mux.Handle("GET /api/admin/v1/agent-executions", h.authenticate(http.HandlerFunc(h.listAgentExecutions)))
	return mux
}

func (h *handler) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if h.adminToken == "" || request.Header.Get("Authorization") != "Bearer "+h.adminToken {
			writeError(writer, http.StatusUnauthorized, "unauthorized", "Authentication is required")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (h *handler) listModelProviders(writer http.ResponseWriter, request *http.Request) {
	views, err := h.application.ListModelProviders(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal_error", "Could not list Model Provider Configurations")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": views})
}

func (h *handler) getModelProvider(writer http.ResponseWriter, request *http.Request) {
	view, err := h.application.GetModelProvider(request.Context(), request.PathValue("provider_key"))
	if err != nil {
		h.writeAdminError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (h *handler) patchModelProvider(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		BaseURL optionalString `json:"base_url"`
		Model   optionalString `json:"model"`
		APIKey  optionalString `json:"api_key"`
	}
	if err := decodeStrict(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	if !input.BaseURL.set && !input.Model.set && !input.APIKey.set {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	view, err := h.application.PatchModelProvider(request.Context(), request.PathValue("provider_key"), admin.ModelProviderPatch{
		BaseURL: input.BaseURL.pointer(),
		Model:   input.Model.pointer(),
		APIKey:  input.APIKey.pointer(),
	})
	if err != nil {
		h.writeAdminError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (h *handler) listConnectors(writer http.ResponseWriter, request *http.Request) {
	views, err := h.application.ListConnectors(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal_error", "Could not list Connector Configurations")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": views})
}

func (h *handler) getConnector(writer http.ResponseWriter, request *http.Request) {
	view, err := h.application.GetConnector(request.Context(), request.PathValue("connector_key"))
	if err != nil {
		h.writeAdminError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (h *handler) patchConnector(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		BaseURL optionalString `json:"base_url"`
		APIKey  optionalString `json:"api_key"`
	}
	if err := decodeStrict(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	if !input.BaseURL.set && !input.APIKey.set {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	view, err := h.application.PatchConnector(request.Context(), request.PathValue("connector_key"), admin.ConnectorPatch{
		BaseURL: input.BaseURL.pointer(),
		APIKey:  input.APIKey.pointer(),
	})
	if err != nil {
		h.writeAdminError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (h *handler) listAgentSchedules(writer http.ResponseWriter, request *http.Request) {
	schedules, err := h.application.ListAgentSchedules(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal_error", "Could not list Agent Schedules")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": schedules})
}

func (h *handler) getAgentSchedule(writer http.ResponseWriter, request *http.Request) {
	schedule, err := h.application.GetAgentSchedule(request.Context(), request.PathValue("agent_key"))
	if err != nil {
		h.writeScheduleError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, schedule)
}

func (h *handler) putAgentSchedule(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		AgentVersion   string                `json:"agent_version"`
		ScheduleType   agentrun.ScheduleType `json:"schedule_type"`
		CronExpression string                `json:"cron_expression"`
		DailyTimes     []string              `json:"daily_times"`
		Input          json.RawMessage       `json:"input"`
		Enabled        *bool                 `json:"enabled"`
	}
	if err := decodeStrict(request, &input); err != nil || input.Enabled == nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	schedule, err := h.application.PutAgentSchedule(request.Context(), agentrun.PutAgentScheduleInput{
		AgentKey: request.PathValue("agent_key"), AgentVersion: input.AgentVersion,
		Type: input.ScheduleType, CronExpression: input.CronExpression,
		DailyTimes: input.DailyTimes, InputPayload: input.Input, Enabled: *input.Enabled,
	})
	if err != nil {
		h.writeScheduleError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, schedule)
}

func (h *handler) patchAgentSchedule(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		AgentVersion   *string                `json:"agent_version"`
		ScheduleType   *agentrun.ScheduleType `json:"schedule_type"`
		CronExpression *string                `json:"cron_expression"`
		DailyTimes     *[]string              `json:"daily_times"`
		Input          json.RawMessage        `json:"input"`
		Enabled        *bool                  `json:"enabled"`
	}
	if err := decodeStrict(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	if input.AgentVersion == nil && input.ScheduleType == nil && input.CronExpression == nil &&
		input.DailyTimes == nil && len(input.Input) == 0 && input.Enabled == nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	var inputPayload *json.RawMessage
	if len(input.Input) > 0 {
		copied := append(json.RawMessage(nil), input.Input...)
		inputPayload = &copied
	}
	schedule, err := h.application.PatchAgentSchedule(
		request.Context(),
		request.PathValue("agent_key"),
		agentrun.PatchAgentScheduleInput{
			AgentVersion: input.AgentVersion, Type: input.ScheduleType,
			CronExpression: input.CronExpression, DailyTimes: input.DailyTimes,
			InputPayload: inputPayload, Enabled: input.Enabled,
		},
	)
	if err != nil {
		h.writeScheduleError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, schedule)
}

func (h *handler) writeScheduleError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, scheduling.ErrAgentNotRegistered), errors.Is(err, agentrun.ErrAgentScheduleNotFound):
		writeError(writer, http.StatusNotFound, "agent_schedule_not_found", "Agent Schedule was not found")
	case errors.Is(err, scheduling.ErrInvalidSchedule):
		writeError(writer, http.StatusBadRequest, "invalid_agent_schedule", "Agent Schedule is invalid")
	case errors.Is(err, scheduling.ErrRuntimeSync):
		writeError(writer, http.StatusServiceUnavailable, "schedule_runtime_sync_failed", "Agent Schedule was saved but runtime synchronization failed")
	default:
		writeError(writer, http.StatusInternalServerError, "internal_error", "Could not manage Agent Schedule")
	}
}

func (h *handler) listAgentExecutions(writer http.ResponseWriter, request *http.Request) {
	page, err := parsePositiveInt(request.URL.Query().Get("page"), 1)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_pagination", "Execution pagination is invalid")
		return
	}
	pageSize, err := parsePositiveInt(request.URL.Query().Get("page_size"), 20)
	if err != nil || pageSize > 100 {
		writeError(writer, http.StatusBadRequest, "invalid_pagination", "Execution pagination is invalid")
		return
	}
	sortOrder := strings.TrimSpace(request.URL.Query().Get("sort_order"))
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		writeError(writer, http.StatusBadRequest, "invalid_sort_order", "Execution sort order is invalid")
		return
	}
	result, err := h.application.ListAgentExecutions(request.Context(), agentrun.ExecutionListQuery{
		AgentKey: strings.TrimSpace(request.URL.Query().Get("agent_key")),
		Page:     page, PageSize: pageSize, Ascending: sortOrder == "asc",
	})
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal_error", "Could not list Agent Executions")
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func parsePositiveInt(value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, errors.New("value must be positive")
	}
	return parsed, nil
}

func (h *handler) writeAdminError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, admin.ErrUnknownTarget):
		writeError(writer, http.StatusNotFound, "configuration_not_found", "Configuration target was not found")
	case errors.Is(err, admin.ErrInvalidConfig):
		writeError(writer, http.StatusBadRequest, "invalid_configuration", "Configuration is invalid")
	default:
		writeError(writer, http.StatusInternalServerError, "internal_error", "Could not update Configuration")
	}
}

type optionalString struct {
	set   bool
	value string
}

func (o *optionalString) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null is not allowed")
	}
	if err := json.Unmarshal(data, &o.value); err != nil {
		return err
	}
	o.set = true
	return nil
}

func (o optionalString) pointer() *string {
	if !o.set {
		return nil
	}
	value := o.value
	return &value
}

func decodeStrict(request *http.Request, target any) error {
	body, err := io.ReadAll(io.LimitReader(request.Body, maxAdminRequestBytes+1))
	if err != nil || len(body) == 0 || len(body) > maxAdminRequestBytes {
		return errors.New("invalid request body")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing request content")
	}
	return nil
}

func writeError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, map[string]any{"error_code": code, "message": message})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
