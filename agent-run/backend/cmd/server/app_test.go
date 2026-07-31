package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

func TestCloseWithinHonorsShutdownDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	blocked := make(chan struct{})
	if err := closeWithin(ctx, func() { <-blocked }); err == nil {
		t.Fatal("closeWithin accepted a cleanup that exceeded its deadline")
	}
	close(blocked)
}

func TestCloseWithinWaitsForResourceCleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	closed := false
	if err := closeWithin(ctx, func() { closed = true }); err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatal("resource cleanup did not run")
	}
}

func TestShutdownWithinEachStartsWorkersWithIndependentBudgets(t *testing.T) {
	secondStarted := make(chan struct{})
	first := func(ctx context.Context) error {
		select {
		case <-secondStarted:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	second := func(context.Context) error {
		close(secondStarted)
		return nil
	}

	if err := shutdownWithinEach(100*time.Millisecond, first, second); err != nil {
		t.Fatalf("shutdown workers: %v", err)
	}
}

func TestAgentLifecycleLoggerEmitsStableSafeJSONFields(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	adapter := slogAgentLifecycleLogger{logger: logger, environment: "uat"}
	adapter.Info(agentrun.AgentLifecycleEvent{
		Code: "agent_execution_completed", AgentKey: "event-semantic-enricher",
		AgentVersion: "event-semantic-enricher.v1", RuntimeMode: "worker",
		ExecutionID: "execution-1", WorkItemID: "work-item-1",
		Status: "succeeded", Outcome: "processed", Duration: 25 * time.Millisecond,
		Counts: map[string]int{
			"accepted_candidates": 2,
			"prompt_secret":       1,
			"candidate_events":    -1,
		},
	})

	var event map[string]any
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	for key, expected := range map[string]string{
		"service": "agentrun", "environment": "uat",
		"event_code":   "agent_execution_completed",
		"agent_key":    "event-semantic-enricher",
		"execution_id": "execution-1", "work_item_id": "work-item-1",
	} {
		if event[key] != expected {
			t.Fatalf("%s = %#v, want %q", key, event[key], expected)
		}
	}
	if strings.Contains(output.String(), "prompt") ||
		strings.Contains(output.String(), "evidence") ||
		strings.Contains(output.String(), "authorization") {
		t.Fatalf("unsafe payload field appeared in lifecycle log: %s", output.String())
	}
	counts, ok := event["counts"].(map[string]any)
	if !ok || counts["accepted_candidates"] != float64(2) ||
		len(counts) != 1 {
		t.Fatalf("safe counts = %#v", event["counts"])
	}
}
