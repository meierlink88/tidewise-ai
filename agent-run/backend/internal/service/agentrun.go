package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	v1 "github.com/meierlink88/tidewise-ai/agent-run/backend/api/agentrun/v1"
	collectorusecase "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/usecase"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/admin"
)

type CollectorUseCase interface {
	Ready(context.Context) error
	CreateCollectorRun(context.Context, string, string) (agentrun.Execution, agentrun.CreateDisposition, error)
	GetCollectorRun(context.Context, string) (agentrun.Execution, error)
}

type AdminUseCase interface {
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
	ListAgentStatuses(context.Context) ([]agentrun.AgentStatus, error)
}

type EventSemanticUseCase interface {
	RequestReanalysis(
		context.Context,
		eventsemantic.ReanalysisRequest,
	) (eventsemantic.WorkItem, bool, error)
}

type AgentRunService struct {
	collector         CollectorUseCase
	admin             AdminUseCase
	eventSemantic     EventSemanticUseCase
	scheduleReadiness Readiness
}

type Readiness interface {
	Ready(context.Context) error
}

func NewAgentRunService(
	collector CollectorUseCase,
	adminUseCase AdminUseCase,
	eventSemanticUseCase EventSemanticUseCase,
	scheduleReadiness Readiness,
) (*AgentRunService, error) {
	if collector == nil {
		return nil, errors.New("Collector Use Case is required")
	}
	if adminUseCase == nil {
		return nil, errors.New("Admin Use Case is required")
	}
	if eventSemanticUseCase == nil {
		return nil, errors.New("Event Semantic Use Case is required")
	}
	if scheduleReadiness == nil {
		return nil, errors.New("Agent Schedule Readiness is required")
	}
	return &AgentRunService{
		collector: collector, admin: adminUseCase, eventSemantic: eventSemanticUseCase,
		scheduleReadiness: scheduleReadiness,
	}, nil
}

func (s *AgentRunService) Ready(ctx context.Context) error {
	if err := s.collector.Ready(ctx); err != nil {
		return err
	}
	return s.scheduleReadiness.Ready(ctx)
}

func (s *AgentRunService) CreateCollectorRun(
	ctx context.Context,
	request *v1.CreateCollectorSubmissionRequest,
) (*v1.CollectorSubmissionResult, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" {
		return nil, v1.InvalidRequest("PROMPT_REQUIRED", "Prompt must not be blank")
	}
	if len([]byte(request.Prompt)) > v1.MaxCollectorPrompt {
		return nil, v1.NewPublicError(http.StatusRequestEntityTooLarge, "PROMPT_TOO_LARGE", "Prompt exceeds 64 KiB", nil)
	}
	execution, _, err := s.collector.CreateCollectorRun(ctx, request.IdempotencyKey, request.Prompt)
	if err != nil {
		return nil, collectorError(err)
	}
	result := collectorSubmissionResult(execution)
	return &result, nil
}

func (s *AgentRunService) GetCollectorRun(
	ctx context.Context,
	request *v1.GetCollectorSubmissionRequest,
) (*v1.CollectorSubmissionResult, error) {
	if request == nil {
		return nil, executionNotFound()
	}
	if _, err := uuid.Parse(request.ExecutionID); err != nil {
		return nil, executionNotFound()
	}
	execution, err := s.collector.GetCollectorRun(ctx, request.ExecutionID)
	if err != nil {
		if errors.Is(err, agentrun.ErrExecutionNotFound) {
			return nil, executionNotFound()
		}
		return nil, internalError("Could not read Agent Execution")
	}
	result := collectorSubmissionResult(execution)
	return &result, nil
}

func (s *AgentRunService) CreateEventSemanticReanalysis(
	ctx context.Context,
	request *v1.CreateEventSemanticReanalysisRequest,
) (*v1.EventSemanticWorkItem, error) {
	if request == nil ||
		strings.TrimSpace(request.IdempotencyKey) == "" ||
		strings.TrimSpace(request.Reason) == "" {
		return nil, v1.InvalidRequest("INVALID_REQUEST", "Event Semantic reanalysis request is invalid")
	}
	if _, err := uuid.Parse(request.EventID); err != nil {
		return nil, v1.InvalidRequest("INVALID_EVENT_ID", "Event ID is invalid")
	}
	if _, err := uuid.Parse(request.SupersedesSubmissionID); err != nil {
		return nil, v1.InvalidRequest(
			"INVALID_SUBMISSION_ID",
			"Superseded Submission ID is invalid",
		)
	}
	item, replayed, err := s.eventSemantic.RequestReanalysis(ctx, eventsemantic.ReanalysisRequest{
		EventID:                request.EventID,
		SupersedesSubmissionID: request.SupersedesSubmissionID,
		Reason:                 strings.TrimSpace(request.Reason),
		IdempotencyKey:         request.IdempotencyKey,
	})
	if err != nil {
		if errors.Is(err, eventsemantic.ErrReanalysisIdempotencyConflict) {
			return nil, v1.NewPublicError(
				http.StatusConflict,
				"IDEMPOTENCY_CONFLICT",
				"Idempotency-Key belongs to another Event Semantic reanalysis request",
				nil,
			)
		}
		return nil, internalError("Could not enqueue Event Semantic reanalysis")
	}
	return &v1.EventSemanticWorkItem{
		WorkItemID: item.ID, EventID: item.EventID,
		SupersedesSubmissionID: item.SupersedesSubmissionID,
		Status:                 item.Status, Replayed: replayed, CreatedAt: item.CreatedAt,
	}, nil
}

func (s *AgentRunService) ListModelProviders(
	ctx context.Context,
	_ *v1.ListModelProvidersRequest,
) (*v1.ModelProviderList, error) {
	items, err := s.admin.ListModelProviders(ctx)
	if err != nil {
		return nil, internalError("Could not list Model Provider Configurations")
	}
	result := &v1.ModelProviderList{Items: make([]v1.ModelProviderConfiguration, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, modelProviderConfiguration(item))
	}
	return result, nil
}

func (s *AgentRunService) GetModelProvider(
	ctx context.Context,
	request *v1.GetModelProviderRequest,
) (*v1.ModelProviderConfiguration, error) {
	view, err := s.admin.GetModelProvider(ctx, request.ProviderKey)
	if err != nil {
		return nil, adminError(err)
	}
	result := modelProviderConfiguration(view)
	return &result, nil
}

func (s *AgentRunService) PatchModelProvider(
	ctx context.Context,
	request *v1.PatchModelProviderRequest,
) (*v1.ModelProviderConfiguration, error) {
	view, err := s.admin.PatchModelProvider(ctx, request.ProviderKey, admin.ModelProviderPatch{
		BaseURL: request.BaseURL.Pointer(),
		Model:   request.Model.Pointer(),
		APIKey:  request.APIKey.Pointer(),
	})
	if err != nil {
		return nil, adminError(err)
	}
	result := modelProviderConfiguration(view)
	return &result, nil
}

func (s *AgentRunService) ListConnectors(
	ctx context.Context,
	_ *v1.ListConnectorsRequest,
) (*v1.ConnectorList, error) {
	items, err := s.admin.ListConnectors(ctx)
	if err != nil {
		return nil, internalError("Could not list Connector Configurations")
	}
	result := &v1.ConnectorList{Items: make([]v1.ConnectorConfiguration, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, connectorConfiguration(item))
	}
	return result, nil
}

func (s *AgentRunService) GetConnector(
	ctx context.Context,
	request *v1.GetConnectorRequest,
) (*v1.ConnectorConfiguration, error) {
	view, err := s.admin.GetConnector(ctx, request.ConnectorKey)
	if err != nil {
		return nil, adminError(err)
	}
	result := connectorConfiguration(view)
	return &result, nil
}

func (s *AgentRunService) PatchConnector(
	ctx context.Context,
	request *v1.PatchConnectorRequest,
) (*v1.ConnectorConfiguration, error) {
	view, err := s.admin.PatchConnector(ctx, request.ConnectorKey, admin.ConnectorPatch{
		BaseURL: request.BaseURL.Pointer(),
		APIKey:  request.APIKey.Pointer(),
	})
	if err != nil {
		return nil, adminError(err)
	}
	result := connectorConfiguration(view)
	return &result, nil
}

func (s *AgentRunService) ListAgentSchedules(
	ctx context.Context,
	_ *v1.ListAgentSchedulesRequest,
) (*v1.AgentScheduleList, error) {
	items, err := s.admin.ListAgentSchedules(ctx)
	if err != nil {
		return nil, internalError("Could not list Agent Schedules")
	}
	result := &v1.AgentScheduleList{Items: make([]v1.AgentSchedule, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, agentSchedule(item))
	}
	return result, nil
}

func (s *AgentRunService) GetAgentSchedule(
	ctx context.Context,
	request *v1.GetAgentScheduleRequest,
) (*v1.AgentSchedule, error) {
	schedule, err := s.admin.GetAgentSchedule(ctx, request.AgentKey)
	if err != nil {
		return nil, scheduleError(err)
	}
	result := agentSchedule(schedule)
	return &result, nil
}

func (s *AgentRunService) PutAgentSchedule(
	ctx context.Context,
	request *v1.PutAgentScheduleRequest,
) (*v1.AgentSchedule, error) {
	schedule, err := s.admin.PutAgentSchedule(ctx, agentrun.PutAgentScheduleInput{
		AgentKey: request.AgentKey, AgentVersion: request.AgentVersion,
		Type: agentrun.ScheduleType(request.ScheduleType), CronExpression: request.CronExpression,
		DailyTimes: request.DailyTimes, InputPayload: append(json.RawMessage(nil), request.Input...),
		Enabled: *request.Enabled,
	})
	if err != nil {
		return nil, scheduleError(err)
	}
	result := agentSchedule(schedule)
	return &result, nil
}

func (s *AgentRunService) PatchAgentSchedule(
	ctx context.Context,
	request *v1.PatchAgentScheduleRequest,
) (*v1.AgentSchedule, error) {
	var scheduleType *agentrun.ScheduleType
	if request.ScheduleType != nil {
		value := agentrun.ScheduleType(*request.ScheduleType)
		scheduleType = &value
	}
	schedule, err := s.admin.PatchAgentSchedule(ctx, request.AgentKey, agentrun.PatchAgentScheduleInput{
		AgentVersion: request.AgentVersion, Type: scheduleType,
		CronExpression: request.CronExpression, DailyTimes: request.DailyTimes,
		InputPayload: request.Input, Enabled: request.Enabled,
	})
	if err != nil {
		return nil, scheduleError(err)
	}
	result := agentSchedule(schedule)
	return &result, nil
}

func (s *AgentRunService) ListAgentExecutions(
	ctx context.Context,
	request *v1.ListAgentExecutionsRequest,
) (*v1.AgentExecutionPage, error) {
	page, err := s.admin.ListAgentExecutions(ctx, agentrun.ExecutionListQuery{
		AgentKey: request.AgentKey, Page: request.Page, PageSize: request.PageSize,
		Ascending: request.SortOrder == "asc",
	})
	if err != nil {
		return nil, internalError("Could not list Agent Executions")
	}
	result := &v1.AgentExecutionPage{
		Items: make([]v1.AgentExecutionListItem, 0, len(page.Items)),
		Page:  page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	}
	for _, item := range page.Items {
		result.Items = append(result.Items, agentExecutionListItem(item))
	}
	return result, nil
}

func (s *AgentRunService) ListAgentStatuses(
	ctx context.Context,
	_ *v1.ListAgentStatusesRequest,
) (*v1.AgentStatusList, error) {
	items, err := s.admin.ListAgentStatuses(ctx)
	if err != nil {
		return nil, internalError("Could not list Agent statuses")
	}
	result := &v1.AgentStatusList{Items: make([]v1.AgentStatus, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, v1.AgentStatus{
			AgentKey: item.AgentKey, DisplayName: item.DisplayName,
			CurrentVersion: item.CurrentVersion, IsWorking: item.IsWorking,
			CurrentExecutionStatus: item.CurrentExecutionStatus, UpdatedAt: item.UpdatedAt,
		})
	}
	return result, nil
}

func collectorError(err error) error {
	switch {
	case errors.Is(err, collectorusecase.ErrNotReady):
		return v1.NewPublicError(http.StatusServiceUnavailable, "CONFIGURATION_NOT_READY", "Collector configuration is not ready", nil)
	case errors.Is(err, agentrun.ErrIdempotencyConflict):
		return v1.NewPublicError(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency-Key belongs to another Prompt", nil)
	default:
		var active *agentrun.ActiveExecutionError
		if errors.As(err, &active) {
			return v1.NewPublicError(http.StatusConflict, "ACTIVE_EXECUTION_EXISTS", "Another Collector run is active", map[string]any{
				"active_execution_id": active.ActiveExecutionID, "skipped_execution_id": active.SkippedExecutionID,
			})
		}
		return internalError("Could not create Collector run")
	}
}

func adminError(err error) error {
	switch {
	case errors.Is(err, admin.ErrUnknownTarget):
		return v1.NewPublicError(http.StatusNotFound, "CONFIGURATION_NOT_FOUND", "Configuration target was not found", nil)
	case errors.Is(err, admin.ErrInvalidConfig):
		return v1.InvalidRequest("INVALID_CONFIGURATION", "Configuration is invalid")
	default:
		return internalError("Could not update Configuration")
	}
}

func scheduleError(err error) error {
	switch {
	case errors.Is(err, agentrun.ErrAgentNotRegistered), errors.Is(err, agentrun.ErrAgentScheduleNotFound):
		return v1.NewPublicError(http.StatusNotFound, "AGENT_SCHEDULE_NOT_FOUND", "Agent Schedule was not found", nil)
	case errors.Is(err, agentrun.ErrInvalidSchedule):
		return v1.InvalidRequest("INVALID_AGENT_SCHEDULE", "Agent Schedule is invalid")
	case errors.Is(err, agentrun.ErrScheduleRuntimeSync):
		return v1.NewPublicError(http.StatusServiceUnavailable, "SCHEDULE_RUNTIME_SYNC_FAILED", "Agent Schedule was saved but runtime synchronization failed", nil)
	default:
		return internalError("Could not manage Agent Schedule")
	}
}

func executionNotFound() error {
	return v1.NewPublicError(http.StatusNotFound, "EXECUTION_NOT_FOUND", "Agent Execution was not found", nil)
}

func internalError(message string) error {
	return v1.NewPublicError(http.StatusInternalServerError, "INTERNAL_ERROR", message, nil)
}

func modelProviderConfiguration(input agentrun.ModelProviderConfigView) v1.ModelProviderConfiguration {
	return v1.ModelProviderConfiguration{
		ProviderKey: input.ProviderKey, BaseURL: input.BaseURL, Model: input.Model,
		Configured: input.Configured, KeyConfigured: input.KeyConfigured,
		MaskedKey: input.MaskedKey, UpdatedAt: input.UpdatedAt,
	}
}

func connectorConfiguration(input agentrun.ConnectorConfigView) v1.ConnectorConfiguration {
	return v1.ConnectorConfiguration{
		ConnectorKey: input.ConnectorKey, BaseURL: input.BaseURL,
		Configured: input.Configured, KeyConfigured: input.KeyConfigured,
		MaskedKey: input.MaskedKey, UpdatedAt: input.UpdatedAt,
	}
}

func agentSchedule(input agentrun.AgentSchedule) v1.AgentSchedule {
	return v1.AgentSchedule{
		ScheduleID: input.ID, AgentKey: input.AgentKey, AgentVersion: input.AgentVersion,
		ScheduleType: string(input.Type), CronExpression: input.CronExpression,
		DailyTimes: append([]string(nil), input.DailyTimes...), Input: append(json.RawMessage(nil), input.InputPayload...),
		Enabled: input.Enabled, LastTriggered: input.LastTriggered, NextRun: input.NextRun,
		CreatedAt: input.CreatedAt, UpdatedAt: input.UpdatedAt,
	}
}

func agentExecutionListItem(input agentrun.ExecutionListItem) v1.AgentExecutionListItem {
	return v1.AgentExecutionListItem{
		ExecutionID: input.ID, AgentKey: input.AgentKey, AgentVersion: input.AgentVersion,
		TriggerSource: string(input.TriggerSource), ScheduleID: input.ScheduleID, Status: string(input.Status),
		ErrorCode: input.ErrorCode, ErrorSummary: input.ErrorSummary, StopReason: input.StopReason,
		BlockedByExecutionID: input.BlockedByExecutionID, CreatedAt: input.CreatedAt,
		TriggeredAt: input.TriggeredAt, StartedAt: input.StartedAt, CompletedAt: input.CompletedAt,
	}
}

func collectorSubmissionResult(input agentrun.Execution) v1.CollectorSubmissionResult {
	invocations := make([]v1.ConnectorInvocation, 0, len(input.Invocations))
	for _, invocation := range input.Invocations {
		invocations = append(invocations, v1.ConnectorInvocation{
			ConnectorKey: invocation.ConnectorKey, Status: string(invocation.Status),
			ResultCount: invocation.ResultCount, ErrorCode: invocation.ErrorCode,
			ErrorSummary: invocation.ErrorSummary, StartedAt: invocation.StartedAt, CompletedAt: invocation.CompletedAt,
		})
	}
	return v1.CollectorSubmissionResult{
		Schema: "collector_run.v1", AgentKey: "collector", AgentVersion: input.AgentVersion,
		ExecutionID: input.ID, Status: string(input.Status), StatusURL: v1.CollectorRunsPath + "/" + input.ID,
		PromptSHA256: input.PromptSHA256, PromptBytes: input.PromptBytes, Invocations: invocations,
		CandidateCounts: cloneIntMap(input.CandidateCounts), Artifacts: cloneStringMap(input.Artifacts), CreatedAt: input.CreatedAt,
		StartedAt: input.StartedAt, CompletedAt: input.CompletedAt, ErrorCode: input.ErrorCode,
		ErrorSummary: input.ErrorSummary, StopReason: input.StopReason,
		BlockedByExecutionID: input.BlockedByExecutionID,
	}
}

func cloneIntMap(input map[string]int) map[string]int {
	if input == nil {
		return nil
	}
	output := make(map[string]int, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
