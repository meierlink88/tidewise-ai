package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/admin-portal/backend/api/admin/v1"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

var _ v1.AdminHTTPServer = (*AdminService)(nil)

type AdminService struct {
	admin *biz.Service
}

func NewAdminService(admin *biz.Service) *AdminService {
	return &AdminService{admin: admin}
}

func (s *AdminService) ListRawDocuments(
	ctx context.Context,
	request *v1.ListRawDocumentsRequest,
) (*v1.RawDocumentListResponse, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	page, err := s.admin.ListRawDocuments(ctx, biz.RawDocumentListQuery{
		Title: request.Title, SourceRef: request.SourceRef, Page: request.Page, PageSize: request.PageSize,
	})
	if err != nil {
		return nil, v1.NewHTTPError(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
	items := make([]v1.RawDocument, 0, len(page.Items))
	for _, document := range page.Items {
		items = append(items, rawDocument(document))
	}
	return &v1.RawDocumentListResponse{
		Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize,
	}, nil
}

func (s *AdminService) ListEvents(
	ctx context.Context,
	request *v1.ListEventsRequest,
) (*v1.EventListResponse, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	query, err := eventQuery(request)
	if err != nil {
		return nil, err
	}
	page, err := s.admin.ListEvents(ctx, query)
	if err != nil {
		return nil, v1.NewHTTPError(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
	items := make([]v1.Event, 0, len(page.Items))
	for _, value := range page.Items {
		items = append(items, event(value))
	}
	return &v1.EventListResponse{
		Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize,
	}, nil
}

func (s *AdminService) GetAgentSchedule(
	ctx context.Context,
	request *v1.AgentKeyRequest,
) (*v1.AgentSchedule, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.GetAgentSchedule(ctx, request.AgentKey)
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	response := agentSchedule(result)
	return &response, nil
}

func (s *AdminService) SaveAgentSchedule(
	ctx context.Context,
	request *v1.SaveAgentScheduleRequest,
) (*v1.AgentSchedule, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.SaveAgentSchedule(ctx, request.AgentKey, biz.SaveAgentScheduleInput{
		AgentVersion: request.AgentVersion, ScheduleType: biz.ScheduleType(request.ScheduleType),
		CronExpression: request.CronExpression, DailyTimes: request.DailyTimes, Input: request.Input,
	})
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	response := agentSchedule(result)
	return &response, nil
}

func (s *AdminService) SetAgentScheduleEnabled(
	ctx context.Context,
	request *v1.SetAgentScheduleEnabledRequest,
) (*v1.AgentSchedule, error) {
	if s == nil || s.admin == nil || request == nil || request.Enabled == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.SetAgentScheduleEnabled(ctx, request.AgentKey, *request.Enabled)
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	response := agentSchedule(result)
	return &response, nil
}

func (s *AdminService) ListAgentExecutions(
	ctx context.Context,
	request *v1.ListAgentExecutionsRequest,
) (*v1.AgentExecutionPage, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.ListCollectorExecutions(ctx, request.Page)
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	items := make([]v1.AgentExecution, 0, len(result.Items))
	for _, value := range result.Items {
		items = append(items, agentExecution(value))
	}
	return &v1.AgentExecutionPage{
		Items: items, Page: result.Page, PageSize: result.PageSize,
		TotalItems: result.TotalItems, TotalPages: result.TotalPages,
	}, nil
}

func (s *AdminService) ListAgentStatuses(
	ctx context.Context,
	_ *v1.EmptyRequest,
) (*v1.AgentStatusListResponse, error) {
	if s == nil || s.admin == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.ListAgentStatuses(ctx)
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	items := make([]v1.AgentStatus, 0, len(result))
	for _, value := range result {
		items = append(items, v1.AgentStatus{
			AgentKey: value.AgentKey, DisplayName: value.DisplayName,
			CurrentVersion: value.CurrentVersion, IsWorking: value.IsWorking,
			CurrentExecutionStatus: value.CurrentExecutionStatus, UpdatedAt: value.UpdatedAt,
		})
	}
	return &v1.AgentStatusListResponse{Items: items}, nil
}

func (s *AdminService) ListModelProviders(
	ctx context.Context,
	_ *v1.EmptyRequest,
) (*v1.ModelProviderListResponse, error) {
	if s == nil || s.admin == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.ListModelProviders(ctx)
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	items := make([]v1.ModelProviderConfiguration, 0, len(result))
	for _, value := range result {
		items = append(items, modelProvider(value))
	}
	return &v1.ModelProviderListResponse{Items: items}, nil
}

func (s *AdminService) GetModelProvider(
	ctx context.Context,
	request *v1.ProviderKeyRequest,
) (*v1.ModelProviderConfiguration, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.GetModelProvider(ctx, request.ProviderKey)
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	response := modelProvider(result)
	return &response, nil
}

func (s *AdminService) PatchModelProvider(
	ctx context.Context,
	request *v1.PatchModelProviderRequest,
) (*v1.ModelProviderConfiguration, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.PatchModelProvider(ctx, request.ProviderKey, biz.ModelProviderPatch{
		BaseURL: request.BaseURL, Model: request.Model, APIKey: request.APIKey,
	})
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	response := modelProvider(result)
	return &response, nil
}

func (s *AdminService) ListConnectors(
	ctx context.Context,
	_ *v1.EmptyRequest,
) (*v1.ConnectorListResponse, error) {
	if s == nil || s.admin == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.ListConnectors(ctx)
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	items := make([]v1.ConnectorConfiguration, 0, len(result))
	for _, value := range result {
		items = append(items, connector(value))
	}
	return &v1.ConnectorListResponse{Items: items}, nil
}

func (s *AdminService) GetConnector(
	ctx context.Context,
	request *v1.ConnectorKeyRequest,
) (*v1.ConnectorConfiguration, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.GetConnector(ctx, request.ConnectorKey)
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	response := connector(result)
	return &response, nil
}

func (s *AdminService) PatchConnector(
	ctx context.Context,
	request *v1.PatchConnectorRequest,
) (*v1.ConnectorConfiguration, error) {
	if s == nil || s.admin == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.admin.PatchConnector(ctx, request.ConnectorKey, biz.ConnectorPatch{
		BaseURL: request.BaseURL, APIKey: request.APIKey,
	})
	if err != nil {
		return nil, mapAgentRunError(err)
	}
	response := connector(result)
	return &response, nil
}

func eventQuery(request *v1.ListEventsRequest) (biz.EventListQuery, error) {
	eventTimeFrom, err := parseOptionalTime(request.EventTimeFrom)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	eventTimeTo, err := parseOptionalTime(request.EventTimeTo)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	firstSeenFrom, err := parseOptionalTime(request.FirstSeenFrom)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	firstSeenTo, err := parseOptionalTime(request.FirstSeenTo)
	if err != nil {
		return biz.EventListQuery{}, err
	}
	eventStatus := biz.EventStatus(request.EventStatus)
	if eventStatus != "" && eventStatus != biz.EventStatusCandidate &&
		eventStatus != biz.EventStatusConfirmed && eventStatus != biz.EventStatusRejected {
		return biz.EventListQuery{}, invalidRequest("unsupported event status")
	}
	factStatus := biz.FactStatus(request.FactStatus)
	if factStatus != "" && factStatus != biz.FactStatusUnverified &&
		factStatus != biz.FactStatusVerified && factStatus != biz.FactStatusDisputed {
		return biz.EventListQuery{}, invalidRequest("unsupported fact status")
	}
	return biz.EventListQuery{
		Title: request.Title, EventStatus: eventStatus, FactStatus: factStatus,
		EventTimeFrom: eventTimeFrom, EventTimeTo: eventTimeTo,
		FirstSeenFrom: firstSeenFrom, FirstSeenTo: firstSeenTo,
		Page: request.Page, PageSize: request.PageSize,
	}, nil
}

func parseOptionalTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, invalidRequest("time query must use RFC3339")
	}
	return &parsed, nil
}

func invalidRequest(message string) error {
	return v1.NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", message)
}

func mapAgentRunError(err error) error {
	switch {
	case errors.Is(err, biz.ErrAgentRunInvalidRequest):
		return v1.NewHTTPError(http.StatusBadRequest, "AGENTRUN_INVALID_REQUEST", "AgentRun rejected the request")
	case errors.Is(err, biz.ErrAgentRunAuthentication):
		return v1.NewHTTPError(http.StatusServiceUnavailable, "AGENTRUN_AUTHENTICATION_FAILED", "AgentRun authentication is unavailable")
	case errors.Is(err, biz.ErrAgentRunNotFound):
		return v1.NewHTTPError(http.StatusNotFound, "AGENTRUN_NOT_FOUND", "AgentRun resource was not found")
	case errors.Is(err, biz.ErrAgentRunConflict):
		return v1.NewHTTPError(http.StatusConflict, "AGENTRUN_CONFLICT", "AgentRun rejected the conflicting request")
	default:
		return v1.NewHTTPError(http.StatusServiceUnavailable, "AGENTRUN_UNAVAILABLE", "AgentRun is unavailable")
	}
}

func rawDocument(value biz.RawDocument) v1.RawDocument {
	response := v1.RawDocument{
		ID: value.ID, ContractVersion: value.ContractVersion, ArtifactID: value.ArtifactID,
		SourceRef: value.SourceRef, IngestChannel: value.IngestChannel, SourceType: value.SourceType,
		SourceName: value.SourceName, SourceURL: value.SourceURL, SourceExternalID: value.SourceExternalID,
		Title: value.Title, ContentText: value.ContentText, RawObjectURI: value.RawObjectURI,
		RawMIMEType: value.RawMIMEType, Language: value.Language,
		CollectedAt: value.CollectedAt.Format(time.RFC3339), IngestStatus: string(value.IngestStatus),
		ContentSHA256: value.ContentSHA256,
	}
	if value.PublishedAt != nil {
		response.PublishedAt = value.PublishedAt.Format(time.RFC3339)
	}
	return response
}

func event(value biz.Event) v1.Event {
	response := v1.Event{
		ID: value.ID, Title: value.Title, Summary: value.Summary,
		FirstSeenAt: value.FirstSeenAt.Format(time.RFC3339),
		EventStatus: string(value.EventStatus), FactStatus: string(value.FactStatus),
		DedupeKey: value.DedupeKey,
	}
	if value.EventTime != nil {
		response.EventTime = value.EventTime.Format(time.RFC3339)
	}
	if value.KnowableAt != nil {
		response.KnowableAt = value.KnowableAt.Format(time.RFC3339)
	}
	if value.PrimarySourceID != nil {
		response.PrimarySourceID = *value.PrimarySourceID
	}
	return response
}

func agentSchedule(value biz.AgentSchedule) v1.AgentSchedule {
	return v1.AgentSchedule{
		ID: value.ID, AgentKey: value.AgentKey, AgentVersion: value.AgentVersion,
		ScheduleType: string(value.ScheduleType), CronExpression: value.CronExpression,
		DailyTimes: value.DailyTimes, Input: value.Input, Enabled: value.Enabled,
		LastTriggered: value.LastTriggered, NextRun: value.NextRun,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func agentExecution(value biz.AgentExecution) v1.AgentExecution {
	return v1.AgentExecution{
		ID: value.ID, AgentKey: value.AgentKey, AgentVersion: value.AgentVersion,
		TriggerSource: value.TriggerSource, ScheduleID: value.ScheduleID, Status: value.Status,
		ErrorCode: value.ErrorCode, ErrorSummary: value.ErrorSummary, StopReason: value.StopReason,
		BlockedByExecutionID: value.BlockedByExecutionID, CreatedAt: value.CreatedAt,
		TriggeredAt: value.TriggeredAt, StartedAt: value.StartedAt, CompletedAt: value.CompletedAt,
	}
}

func modelProvider(value biz.ModelProviderConfiguration) v1.ModelProviderConfiguration {
	return v1.ModelProviderConfiguration{
		ProviderKey: value.ProviderKey, BaseURL: value.BaseURL, Model: value.Model,
		Configured: value.Configured, KeyConfigured: value.KeyConfigured,
		MaskedKey: value.MaskedKey, UpdatedAt: value.UpdatedAt,
	}
}

func connector(value biz.ConnectorConfiguration) v1.ConnectorConfiguration {
	return v1.ConnectorConfiguration{
		ConnectorKey: value.ConnectorKey, BaseURL: value.BaseURL,
		Configured: value.Configured, KeyConfigured: value.KeyConfigured,
		MaskedKey: value.MaskedKey, UpdatedAt: value.UpdatedAt,
	}
}
