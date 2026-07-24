package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/scheduling"
)

type recordingScheduledCollector struct {
	readyErr       error
	createErr      error
	idempotencyKey string
	scheduleID     string
	prompt         string
	inputPayload   json.RawMessage
	triggeredAt    time.Time
}

func (c *recordingScheduledCollector) Ready(context.Context) error {
	return c.readyErr
}

func (c *recordingScheduledCollector) CreateScheduledCollectorRun(
	_ context.Context,
	idempotencyKey string,
	scheduleID string,
	prompt string,
	inputPayload json.RawMessage,
	triggeredAt time.Time,
) (agentrun.Execution, agentrun.CreateDisposition, error) {
	c.idempotencyKey = idempotencyKey
	c.scheduleID = scheduleID
	c.prompt = prompt
	c.inputPayload = append(c.inputPayload[:0], inputPayload...)
	c.triggeredAt = triggeredAt
	return agentrun.Execution{}, agentrun.ExecutionCreated, c.createErr
}

func TestCollectorScheduleRunnerValidatesAndForwardsTrigger(t *testing.T) {
	collector := &recordingScheduledCollector{}
	runner := collectorScheduleRunner{collector: collector}
	payload := json.RawMessage(`{"prompt":"采集最近两小时资讯"}`)
	triggeredAt := time.Date(2026, 7, 24, 2, 0, 0, 0, time.UTC)

	if err := runner.ValidateInput(context.Background(), "collector.v1", payload); err != nil {
		t.Fatal(err)
	}
	if err := runner.Trigger(context.Background(), scheduling.Trigger{
		ScheduleID: "schedule-1", AgentKey: "collector", AgentVersion: "collector.v1",
		InputPayload: payload, TriggeredAt: triggeredAt,
	}); err != nil {
		t.Fatal(err)
	}
	if collector.idempotencyKey != "schedule:schedule-1:2026-07-24T02:00:00Z" ||
		collector.scheduleID != "schedule-1" ||
		collector.prompt != "采集最近两小时资讯" ||
		string(collector.inputPayload) != string(payload) ||
		!collector.triggeredAt.Equal(triggeredAt) {
		t.Fatalf("forwarded scheduled Collector call = %#v", collector)
	}

	for _, invalid := range []struct {
		version string
		payload json.RawMessage
	}{
		{version: "collector.v2", payload: payload},
		{version: "collector.v1", payload: json.RawMessage(`{"prompt":"采集","connector":"tavily"}`)},
		{version: "collector.v1", payload: json.RawMessage(`{"prompt":"   "}`)},
		{version: "collector.v1", payload: json.RawMessage(`{"prompt":"` + strings.Repeat("x", maxCollectorPrompt+1) + `"}`)},
	} {
		if err := runner.ValidateInput(context.Background(), invalid.version, invalid.payload); err == nil {
			t.Fatalf("ValidateInput accepted version=%q payload length=%d", invalid.version, len(invalid.payload))
		}
	}
}

func TestCollectorScheduleRunnerTreatsActiveConflictAsAuditedTrigger(t *testing.T) {
	collector := &recordingScheduledCollector{createErr: &agentrun.ActiveExecutionError{
		ActiveExecutionID: "active", SkippedExecutionID: "skipped",
	}}
	runner := collectorScheduleRunner{collector: collector}
	err := runner.Trigger(context.Background(), scheduling.Trigger{
		ScheduleID: "schedule-1", AgentKey: "collector", AgentVersion: "collector.v1",
		InputPayload: json.RawMessage(`{"prompt":"采集资讯"}`), TriggeredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("audited active conflict returned error: %v", err)
	}

	collector.createErr = errors.New("create failed")
	if err := runner.Trigger(context.Background(), scheduling.Trigger{
		ScheduleID: "schedule-2", AgentKey: "collector", AgentVersion: "collector.v1",
		InputPayload: json.RawMessage(`{"prompt":"采集资讯"}`), TriggeredAt: time.Now().UTC(),
	}); err == nil {
		t.Fatal("Collector Schedule runner hid a non-conflict creation failure")
	}
}
