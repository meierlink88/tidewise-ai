package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/scheduling"
)

const (
	AgentKey           = "collector"
	AgentVersion       = "collector.v1"
	maxCollectorPrompt = 64 * 1024
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

type ScheduleRunner struct {
	collector scheduledCollector
}

type collectorScheduleInput struct {
	Prompt string `json:"prompt"`
}

func NewScheduleRunner(collector scheduledCollector) ScheduleRunner {
	return ScheduleRunner{collector: collector}
}

func (r ScheduleRunner) ValidateInput(_ context.Context, version string, payload json.RawMessage) error {
	if version != AgentVersion {
		return errors.New("Collector Agent Version is unsupported")
	}
	_, err := decodeCollectorScheduleInput(payload)
	return err
}

func (r ScheduleRunner) ConfigurationReady(ctx context.Context, version string) error {
	if version != AgentVersion {
		return errors.New("Collector Agent Version is unsupported")
	}
	return r.collector.Ready(ctx)
}

func (r ScheduleRunner) Trigger(ctx context.Context, trigger scheduling.Trigger) error {
	if trigger.AgentKey != AgentKey || trigger.AgentVersion != AgentVersion {
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
