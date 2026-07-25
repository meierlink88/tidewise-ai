package data

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

func TestAgentRunScheduleWireValidatesAndMaps(t *testing.T) {
	wire := agentScheduleWire{
		ID: "schedule-1", AgentKey: "collector", AgentVersion: "collector.v1",
		ScheduleType: "daily", DailyTimes: []string{"08:30"},
		Input:     json.RawMessage(`{"prompt":"  preserve me  "}`),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	got, err := wire.toBiz()
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentKey != "collector" || string(got.Input) != `{"prompt":"  preserve me  "}` {
		t.Fatalf("mapped schedule = %#v", got)
	}
}

func TestAgentRunScheduleWireRejectsMissingIdentity(t *testing.T) {
	_, err := (agentScheduleWire{
		ID: "schedule-1", AgentVersion: "collector.v1",
		ScheduleType: "daily", DailyTimes: []string{"08:30"}, Input: json.RawMessage(`{}`),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).toBiz()
	if !errors.Is(err, biz.ErrAgentRunUnavailable) {
		t.Fatalf("error = %v, want AgentRun unavailable", err)
	}
}

func TestAgentRunScheduleWireRejectsNullInputAndMissingTimestamps(t *testing.T) {
	tests := []agentScheduleWire{
		{
			ID: "schedule-1", AgentKey: "collector", AgentVersion: "collector.v1",
			ScheduleType: "daily", Input: json.RawMessage(`null`),
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: "schedule-1", AgentKey: "collector", AgentVersion: "collector.v1",
			ScheduleType: "daily", Input: json.RawMessage(`{}`),
		},
	}
	for _, wire := range tests {
		if _, err := wire.toBiz(); !errors.Is(err, biz.ErrAgentRunUnavailable) {
			t.Fatalf("error = %v, want AgentRun unavailable", err)
		}
	}
}

func TestAgentRunExecutionPageWireRejectsInvalidPagination(t *testing.T) {
	_, err := (agentExecutionPageWire{Page: 0, PageSize: 20}).toBiz()
	if !errors.Is(err, biz.ErrAgentRunUnavailable) {
		t.Fatalf("error = %v, want AgentRun unavailable", err)
	}
}

func TestAgentRunWireRejectsMissingExecutionTimesAndInvalidConfigurationURLs(t *testing.T) {
	if _, err := (agentExecutionWire{
		ID: "execution-1", AgentKey: "collector", AgentVersion: "collector.v1",
		TriggerSource: "schedule", Status: "running",
	}).toBiz(); !errors.Is(err, biz.ErrAgentRunUnavailable) {
		t.Fatalf("execution error = %v", err)
	}
	if _, err := (modelProviderWire{ProviderKey: "deepseek", BaseURL: "not-a-url"}).toBiz(); !errors.Is(err, biz.ErrAgentRunUnavailable) {
		t.Fatalf("provider error = %v", err)
	}
	if _, err := (connectorWire{ConnectorKey: "tavily", BaseURL: ""}).toBiz(); !errors.Is(err, biz.ErrAgentRunUnavailable) {
		t.Fatalf("connector error = %v", err)
	}
}

func TestAgentRunHTTPStatusMapsToBusinessErrors(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{status: 400, want: biz.ErrAgentRunInvalidRequest},
		{status: 401, want: biz.ErrAgentRunAuthentication},
		{status: 404, want: biz.ErrAgentRunNotFound},
		{status: 409, want: biz.ErrAgentRunConflict},
		{status: 503, want: biz.ErrAgentRunUnavailable},
	}
	for _, test := range tests {
		if got := agentRunBusinessError(test.status); !errors.Is(got, test.want) {
			t.Errorf("status %d mapped to %v, want %v", test.status, got, test.want)
		}
	}
}
