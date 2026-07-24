// Package usecase contains Admin Portal business orchestration.
package usecase

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/agentrunclient"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/dataclient"
)

var ErrDataServiceUnavailable = errors.New("data service unavailable")
var ErrAgentRunUnavailable = errors.New("AgentRun unavailable")

type Service struct {
	dataClient     dataclient.DataServiceClient
	agentRunClient agentrunclient.Client
}

func NewService(dataClient dataclient.DataServiceClient, agentRunClient agentrunclient.Client) *Service {
	return &Service{dataClient: dataClient, agentRunClient: agentRunClient}
}

func (s *Service) ListRawDocuments(ctx context.Context, query dataclient.RawDocumentListQuery) (dataclient.RawDocumentPage, error) {
	if s == nil || s.dataClient == nil {
		return dataclient.RawDocumentPage{}, ErrDataServiceUnavailable
	}
	return s.dataClient.ListRawDocuments(ctx, query)
}

func (s *Service) ListEvents(ctx context.Context, query dataclient.EventListQuery) (dataclient.EventPage, error) {
	if s == nil || s.dataClient == nil {
		return dataclient.EventPage{}, ErrDataServiceUnavailable
	}
	return s.dataClient.ListEvents(ctx, query)
}

func (s *Service) GetAgentSchedule(ctx context.Context, agentKey string) (agentrunclient.AgentSchedule, error) {
	if s == nil || s.agentRunClient == nil {
		return agentrunclient.AgentSchedule{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.GetAgentSchedule(ctx, agentKey)
}

type SaveAgentScheduleInput struct {
	AgentVersion   string
	ScheduleType   agentrunclient.ScheduleType
	CronExpression string
	DailyTimes     []string
	Input          json.RawMessage
}

func (s *Service) SaveAgentSchedule(
	ctx context.Context,
	agentKey string,
	input SaveAgentScheduleInput,
) (agentrunclient.AgentSchedule, error) {
	if s == nil || s.agentRunClient == nil {
		return agentrunclient.AgentSchedule{}, ErrAgentRunUnavailable
	}
	_, err := s.agentRunClient.GetAgentSchedule(ctx, agentKey)
	if err != nil {
		if !agentrunclient.IsNotFound(err) {
			return agentrunclient.AgentSchedule{}, err
		}
		return s.agentRunClient.PutAgentSchedule(ctx, agentKey, agentrunclient.PutAgentScheduleInput{
			AgentVersion: input.AgentVersion, ScheduleType: input.ScheduleType,
			CronExpression: input.CronExpression, DailyTimes: input.DailyTimes,
			Input: input.Input, Enabled: false,
		})
	}
	agentVersion := input.AgentVersion
	scheduleType := input.ScheduleType
	cronExpression := input.CronExpression
	dailyTimes := input.DailyTimes
	rawInput := input.Input
	return s.agentRunClient.PatchAgentSchedule(ctx, agentKey, agentrunclient.PatchAgentScheduleInput{
		AgentVersion: &agentVersion, ScheduleType: &scheduleType,
		CronExpression: &cronExpression, DailyTimes: &dailyTimes, Input: &rawInput,
	})
}

func (s *Service) SetAgentScheduleEnabled(
	ctx context.Context,
	agentKey string,
	enabled bool,
) (agentrunclient.AgentSchedule, error) {
	if s == nil || s.agentRunClient == nil {
		return agentrunclient.AgentSchedule{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.PatchAgentSchedule(ctx, agentKey, agentrunclient.PatchAgentScheduleInput{Enabled: &enabled})
}

func (s *Service) ListCollectorExecutions(ctx context.Context, page int) (agentrunclient.AgentExecutionPage, error) {
	if s == nil || s.agentRunClient == nil {
		return agentrunclient.AgentExecutionPage{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.ListAgentExecutions(ctx, agentrunclient.AgentExecutionQuery{
		AgentKey: "collector", Page: page, PageSize: 20,
	})
}

func (s *Service) ListModelProviders(ctx context.Context) ([]agentrunclient.ModelProviderConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return nil, ErrAgentRunUnavailable
	}
	return s.agentRunClient.ListModelProviders(ctx)
}

func (s *Service) GetModelProvider(ctx context.Context, providerKey string) (agentrunclient.ModelProviderConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return agentrunclient.ModelProviderConfiguration{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.GetModelProvider(ctx, providerKey)
}

func (s *Service) PatchModelProvider(
	ctx context.Context,
	providerKey string,
	patch agentrunclient.ModelProviderPatch,
) (agentrunclient.ModelProviderConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return agentrunclient.ModelProviderConfiguration{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.PatchModelProvider(ctx, providerKey, patch)
}

func (s *Service) ListConnectors(ctx context.Context) ([]agentrunclient.ConnectorConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return nil, ErrAgentRunUnavailable
	}
	return s.agentRunClient.ListConnectors(ctx)
}

func (s *Service) GetConnector(ctx context.Context, connectorKey string) (agentrunclient.ConnectorConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return agentrunclient.ConnectorConfiguration{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.GetConnector(ctx, connectorKey)
}

func (s *Service) PatchConnector(
	ctx context.Context,
	connectorKey string,
	patch agentrunclient.ConnectorPatch,
) (agentrunclient.ConnectorConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return agentrunclient.ConnectorConfiguration{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.PatchConnector(ctx, connectorKey, patch)
}
