package biz

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSaveAgentSchedulePreservesExistingEnabledState(t *testing.T) {
	var patch PatchAgentScheduleInput
	repo := &fakeAgentRunRepo{
		getSchedule: func(context.Context, string) (AgentSchedule, error) {
			return AgentSchedule{Enabled: true}, nil
		},
		patchSchedule: func(_ context.Context, _ string, input PatchAgentScheduleInput) (AgentSchedule, error) {
			patch = input
			return AgentSchedule{Enabled: true}, nil
		},
	}

	result, err := NewService(nil, repo).SaveAgentSchedule(context.Background(), "collector", SaveAgentScheduleInput{
		AgentVersion: "collector.v1", ScheduleType: ScheduleTypeDaily,
		DailyTimes: []string{"08:00"}, Input: json.RawMessage(`{"collection_prompt":"collect"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if patch.Enabled != nil || !result.Enabled {
		t.Fatalf("patch/result enabled = %#v/%v, want omitted/true", patch.Enabled, result.Enabled)
	}
}

func TestFirstAgentScheduleSaveCreatesDisabledSchedule(t *testing.T) {
	var created PutAgentScheduleInput
	repo := &fakeAgentRunRepo{
		getSchedule: func(context.Context, string) (AgentSchedule, error) {
			return AgentSchedule{}, ErrAgentRunNotFound
		},
		putSchedule: func(_ context.Context, _ string, input PutAgentScheduleInput) (AgentSchedule, error) {
			created = input
			return AgentSchedule{Enabled: input.Enabled}, nil
		},
	}

	result, err := NewService(nil, repo).SaveAgentSchedule(context.Background(), "collector", SaveAgentScheduleInput{
		AgentVersion: "collector.v1", ScheduleType: ScheduleTypeCron, CronExpression: "0 * * * *",
		Input: json.RawMessage(`{"collection_prompt":"collect"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Enabled || result.Enabled {
		t.Fatalf("created/result enabled = %v/%v, want false/false", created.Enabled, result.Enabled)
	}
}

func TestCollectorExecutionsUseFixedAgentAndPageSize(t *testing.T) {
	var query AgentExecutionQuery
	repo := &fakeAgentRunRepo{
		listExecutions: func(_ context.Context, input AgentExecutionQuery) (AgentExecutionPage, error) {
			query = input
			return AgentExecutionPage{Page: input.Page, PageSize: input.PageSize}, nil
		},
	}

	result, err := NewService(nil, repo).ListCollectorExecutions(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if query.AgentKey != "collector" || query.Page != 3 || query.PageSize != 20 ||
		result.Page != 3 || result.PageSize != 20 {
		t.Fatalf("query/result = %#v/%#v", query, result)
	}
}

func TestListAgentStatusesDelegatesWithoutExpandingExecutionDetails(t *testing.T) {
	now := time.Date(2026, 7, 29, 8, 30, 0, 0, time.UTC)
	repo := &fakeAgentRunRepo{
		listStatuses: func(context.Context) ([]AgentStatus, error) {
			return []AgentStatus{{
				AgentKey: "event-semantic-enricher", DisplayName: "Event Semantic Enricher",
				CurrentVersion: "event-semantic-enricher.v1", IsWorking: true,
				CurrentExecutionStatus: "running", UpdatedAt: now,
			}}, nil
		},
	}

	result, err := NewService(nil, repo).ListAgentStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].AgentKey != "event-semantic-enricher" ||
		!result[0].IsWorking || result[0].CurrentExecutionStatus != "running" {
		t.Fatalf("statuses = %#v", result)
	}
}

type fakeAgentRunRepo struct {
	getSchedule    func(context.Context, string) (AgentSchedule, error)
	putSchedule    func(context.Context, string, PutAgentScheduleInput) (AgentSchedule, error)
	patchSchedule  func(context.Context, string, PatchAgentScheduleInput) (AgentSchedule, error)
	listExecutions func(context.Context, AgentExecutionQuery) (AgentExecutionPage, error)
	listStatuses   func(context.Context) ([]AgentStatus, error)
}

func (f *fakeAgentRunRepo) GetAgentSchedule(ctx context.Context, key string) (AgentSchedule, error) {
	return f.getSchedule(ctx, key)
}
func (f *fakeAgentRunRepo) PutAgentSchedule(ctx context.Context, key string, input PutAgentScheduleInput) (AgentSchedule, error) {
	return f.putSchedule(ctx, key, input)
}
func (f *fakeAgentRunRepo) PatchAgentSchedule(ctx context.Context, key string, input PatchAgentScheduleInput) (AgentSchedule, error) {
	return f.patchSchedule(ctx, key, input)
}
func (f *fakeAgentRunRepo) ListAgentExecutions(ctx context.Context, query AgentExecutionQuery) (AgentExecutionPage, error) {
	return f.listExecutions(ctx, query)
}
func (f *fakeAgentRunRepo) ListAgentStatuses(ctx context.Context) ([]AgentStatus, error) {
	return f.listStatuses(ctx)
}
func (*fakeAgentRunRepo) ListModelProviders(context.Context) ([]ModelProviderConfiguration, error) {
	return nil, nil
}
func (*fakeAgentRunRepo) GetModelProvider(context.Context, string) (ModelProviderConfiguration, error) {
	return ModelProviderConfiguration{}, nil
}
func (*fakeAgentRunRepo) PatchModelProvider(context.Context, string, ModelProviderPatch) (ModelProviderConfiguration, error) {
	return ModelProviderConfiguration{}, nil
}
func (*fakeAgentRunRepo) ListConnectors(context.Context) ([]ConnectorConfiguration, error) {
	return nil, nil
}
func (*fakeAgentRunRepo) GetConnector(context.Context, string) (ConnectorConfiguration, error) {
	return ConnectorConfiguration{}, nil
}
func (*fakeAgentRunRepo) PatchConnector(context.Context, string, ConnectorPatch) (ConnectorConfiguration, error) {
	return ConnectorConfiguration{}, nil
}
