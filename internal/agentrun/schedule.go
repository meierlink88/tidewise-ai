package agentrun

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrAgentScheduleNotFound = errors.New("Agent Schedule was not found")

type ScheduleType string

const (
	ScheduleCron  ScheduleType = "cron"
	ScheduleDaily ScheduleType = "daily"
)

type AgentSchedule struct {
	ID             string          `json:"schedule_id"`
	AgentKey       string          `json:"agent_key"`
	AgentVersion   string          `json:"agent_version"`
	Type           ScheduleType    `json:"schedule_type"`
	CronExpression string          `json:"cron_expression,omitempty"`
	DailyTimes     []string        `json:"daily_times,omitempty"`
	InputPayload   json.RawMessage `json:"input"`
	Enabled        bool            `json:"enabled"`
	LastTriggered  *time.Time      `json:"last_triggered_at,omitempty"`
	NextRun        *time.Time      `json:"next_run_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type PutAgentScheduleInput struct {
	AgentKey       string
	AgentVersion   string
	Type           ScheduleType
	CronExpression string
	DailyTimes     []string
	InputPayload   json.RawMessage
	Enabled        bool
	UpdatedAt      time.Time
}

type PatchAgentScheduleInput struct {
	AgentVersion   *string
	Type           *ScheduleType
	CronExpression *string
	DailyTimes     *[]string
	InputPayload   *json.RawMessage
	Enabled        *bool
	UpdatedAt      time.Time
}
