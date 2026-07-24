package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/scheduling"
)

const (
	collectorAgentKey     = "collector"
	collectorAgentVersion = "collector.v1"
	maxCollectorPrompt    = 64 * 1024
)

type scheduledCollector interface {
	Ready(context.Context) error
	CreateScheduledCollectorRun(
		context.Context,
		string,
		string,
		string,
		json.RawMessage,
		time.Time,
	) (agentrun.Execution, agentrun.CreateDisposition, error)
}

type collectorScheduleRunner struct {
	collector scheduledCollector
}

type collectorScheduleInput struct {
	Prompt string `json:"prompt"`
}

func (r collectorScheduleRunner) ValidateInput(_ context.Context, version string, payload json.RawMessage) error {
	if version != collectorAgentVersion {
		return errors.New("Collector Agent Version is unsupported")
	}
	_, err := decodeCollectorScheduleInput(payload)
	return err
}

func (r collectorScheduleRunner) ConfigurationReady(ctx context.Context, version string) error {
	if version != collectorAgentVersion {
		return errors.New("Collector Agent Version is unsupported")
	}
	return r.collector.Ready(ctx)
}

func (r collectorScheduleRunner) Trigger(ctx context.Context, trigger scheduling.Trigger) error {
	if trigger.AgentKey != collectorAgentKey || trigger.AgentVersion != collectorAgentVersion {
		return errors.New("Collector trigger identity is invalid")
	}
	input, err := decodeCollectorScheduleInput(trigger.InputPayload)
	if err != nil {
		return err
	}
	idempotencyKey := fmt.Sprintf(
		"schedule:%s:%s",
		trigger.ScheduleID,
		trigger.TriggeredAt.UTC().Format(time.RFC3339Nano),
	)
	_, _, err = r.collector.CreateScheduledCollectorRun(
		ctx,
		idempotencyKey,
		trigger.ScheduleID,
		input.Prompt,
		trigger.InputPayload,
		trigger.TriggeredAt,
	)
	var active *agentrun.ActiveExecutionError
	if errors.As(err, &active) {
		return nil
	}
	return err
}

func decodeCollectorScheduleInput(payload json.RawMessage) (collectorScheduleInput, error) {
	var input collectorScheduleInput
	if len(payload) == 0 || len(payload) > maxCollectorPrompt*6+4096 {
		return input, errors.New("Collector Agent Input is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return input, errors.New("Collector Agent Input is invalid")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return input, errors.New("Collector Agent Input is invalid")
	}
	if strings.TrimSpace(input.Prompt) == "" {
		return input, errors.New("Collector Prompt is required")
	}
	if len([]byte(input.Prompt)) > maxCollectorPrompt {
		return input, errors.New("Collector Prompt exceeds 64 KiB")
	}
	return input, nil
}
