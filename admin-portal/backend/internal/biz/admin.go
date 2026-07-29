// Package biz contains Admin Portal business orchestration and domain ports.
package biz

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrDataServiceUnavailable = errors.New("data service unavailable")

type Service struct {
	dataClient     DataServiceRepo
	agentRunClient AgentRunRepo
}

func NewService(dataClient DataServiceRepo, agentRunClient AgentRunRepo) *Service {
	return &Service{dataClient: dataClient, agentRunClient: agentRunClient}
}

func (s *Service) ListRawDocuments(ctx context.Context, query RawDocumentListQuery) (RawDocumentPage, error) {
	if s == nil || s.dataClient == nil {
		return RawDocumentPage{}, ErrDataServiceUnavailable
	}
	return s.dataClient.ListRawDocuments(ctx, query)
}

func (s *Service) ListEvents(ctx context.Context, query EventListQuery) (EventPage, error) {
	if s == nil || s.dataClient == nil {
		return EventPage{}, ErrDataServiceUnavailable
	}
	return s.dataClient.ListEvents(ctx, query)
}

func (s *Service) GetAgentSchedule(ctx context.Context, agentKey string) (AgentSchedule, error) {
	if s == nil || s.agentRunClient == nil {
		return AgentSchedule{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.GetAgentSchedule(ctx, agentKey)
}

type SaveAgentScheduleInput struct {
	AgentVersion   string
	ScheduleType   ScheduleType
	CronExpression string
	DailyTimes     []string
	Input          json.RawMessage
}

func (s *Service) SaveAgentSchedule(
	ctx context.Context,
	agentKey string,
	input SaveAgentScheduleInput,
) (AgentSchedule, error) {
	if s == nil || s.agentRunClient == nil {
		return AgentSchedule{}, ErrAgentRunUnavailable
	}
	_, err := s.agentRunClient.GetAgentSchedule(ctx, agentKey)
	if err != nil {
		if !IsNotFound(err) {
			return AgentSchedule{}, err
		}
		return s.agentRunClient.PutAgentSchedule(ctx, agentKey, PutAgentScheduleInput{
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
	return s.agentRunClient.PatchAgentSchedule(ctx, agentKey, PatchAgentScheduleInput{
		AgentVersion: &agentVersion, ScheduleType: &scheduleType,
		CronExpression: &cronExpression, DailyTimes: &dailyTimes, Input: &rawInput,
	})
}

func (s *Service) SetAgentScheduleEnabled(
	ctx context.Context,
	agentKey string,
	enabled bool,
) (AgentSchedule, error) {
	if s == nil || s.agentRunClient == nil {
		return AgentSchedule{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.PatchAgentSchedule(ctx, agentKey, PatchAgentScheduleInput{Enabled: &enabled})
}

func (s *Service) ListCollectorExecutions(ctx context.Context, page int) (AgentExecutionPage, error) {
	if s == nil || s.agentRunClient == nil {
		return AgentExecutionPage{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.ListAgentExecutions(ctx, AgentExecutionQuery{
		AgentKey: "collector", Page: page, PageSize: 20,
	})
}

func (s *Service) ListAgentStatuses(ctx context.Context) ([]AgentStatus, error) {
	if s == nil || s.agentRunClient == nil {
		return nil, ErrAgentRunUnavailable
	}
	return s.agentRunClient.ListAgentStatuses(ctx)
}

func (s *Service) ListModelProviders(ctx context.Context) ([]ModelProviderConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return nil, ErrAgentRunUnavailable
	}
	return s.agentRunClient.ListModelProviders(ctx)
}

func (s *Service) GetModelProvider(ctx context.Context, providerKey string) (ModelProviderConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return ModelProviderConfiguration{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.GetModelProvider(ctx, providerKey)
}

func (s *Service) PatchModelProvider(
	ctx context.Context,
	providerKey string,
	patch ModelProviderPatch,
) (ModelProviderConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return ModelProviderConfiguration{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.PatchModelProvider(ctx, providerKey, patch)
}

func (s *Service) ListConnectors(ctx context.Context) ([]ConnectorConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return nil, ErrAgentRunUnavailable
	}
	return s.agentRunClient.ListConnectors(ctx)
}

func (s *Service) GetConnector(ctx context.Context, connectorKey string) (ConnectorConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return ConnectorConfiguration{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.GetConnector(ctx, connectorKey)
}

func (s *Service) PatchConnector(
	ctx context.Context,
	connectorKey string,
	patch ConnectorPatch,
) (ConnectorConfiguration, error) {
	if s == nil || s.agentRunClient == nil {
		return ConnectorConfiguration{}, ErrAgentRunUnavailable
	}
	return s.agentRunClient.PatchConnector(ctx, connectorKey, patch)
}
