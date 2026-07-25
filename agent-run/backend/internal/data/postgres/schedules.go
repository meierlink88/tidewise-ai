package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

const scheduleColumns = `
	schedule_id::text, agent_key, agent_version, schedule_type,
	COALESCE(cron_expression, ''), daily_times, input_payload, enabled,
	last_triggered_at, next_run_at, created_at, updated_at
`

func (s *Store) PutAgentSchedule(ctx context.Context, input agentrun.PutAgentScheduleInput) (agentrun.AgentSchedule, error) {
	if len(input.InputPayload) == 0 || len(input.InputPayload) > 64*1024 {
		return agentrun.AgentSchedule{}, fmt.Errorf("Agent Schedule Input must be at most 64 KiB")
	}
	var cronExpression any
	var dailyTimes any
	switch input.Type {
	case agentrun.ScheduleCron:
		cronExpression = input.CronExpression
	case agentrun.ScheduleDaily:
		encoded, err := json.Marshal(input.DailyTimes)
		if err != nil {
			return agentrun.AgentSchedule{}, fmt.Errorf("encode Agent Schedule daily times: %w", err)
		}
		dailyTimes = encoded
	default:
		return agentrun.AgentSchedule{}, fmt.Errorf("unsupported Agent Schedule type")
	}
	updatedAt := input.UpdatedAt.UTC()
	id := uuid.NewString()
	var savedID string
	err := s.database.QueryRow(ctx, `
		INSERT INTO agent_schedules (
			schedule_id, agent_key, agent_version, schedule_type,
			cron_expression, daily_times, input_payload, enabled,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		ON CONFLICT (agent_key) DO UPDATE
		SET agent_version = EXCLUDED.agent_version,
		    schedule_type = EXCLUDED.schedule_type,
		    cron_expression = EXCLUDED.cron_expression,
		    daily_times = EXCLUDED.daily_times,
		    input_payload = EXCLUDED.input_payload,
		    enabled = EXCLUDED.enabled,
		    updated_at = EXCLUDED.updated_at
		RETURNING schedule_id::text
	`, id, input.AgentKey, input.AgentVersion, input.Type, cronExpression,
		dailyTimes, input.InputPayload, input.Enabled, updatedAt).Scan(&savedID)
	if err != nil {
		return agentrun.AgentSchedule{}, fmt.Errorf("put Agent Schedule: %w", err)
	}
	return s.GetAgentSchedule(ctx, input.AgentKey)
}

func (s *Store) GetAgentSchedule(ctx context.Context, agentKey string) (agentrun.AgentSchedule, error) {
	schedule, err := scanSchedule(s.database.QueryRow(ctx, `
		SELECT `+scheduleColumns+`
		FROM agent_schedules
		WHERE agent_key = $1
	`, agentKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return agentrun.AgentSchedule{}, agentrun.ErrAgentScheduleNotFound
	}
	return schedule, err
}

func (s *Store) ListAgentSchedules(ctx context.Context) ([]agentrun.AgentSchedule, error) {
	rows, err := s.database.Query(ctx, `
		SELECT `+scheduleColumns+`
		FROM agent_schedules
		ORDER BY agent_key
	`)
	if err != nil {
		return nil, fmt.Errorf("list Agent Schedules: %w", err)
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func (s *Store) ListEnabledAgentSchedules(ctx context.Context) ([]agentrun.AgentSchedule, error) {
	rows, err := s.database.Query(ctx, `
		SELECT `+scheduleColumns+`
		FROM agent_schedules
		WHERE enabled
		ORDER BY agent_key
	`)
	if err != nil {
		return nil, fmt.Errorf("list enabled Agent Schedules: %w", err)
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func (s *Store) UpdateAgentScheduleRuntime(
	ctx context.Context,
	scheduleID string,
	lastTriggeredAt *time.Time,
	nextRunAt *time.Time,
) error {
	_, err := s.database.Exec(ctx, `
		UPDATE agent_schedules
		SET last_triggered_at = COALESCE($2, last_triggered_at),
		    next_run_at = $3
		WHERE schedule_id = $1
	`, scheduleID, lastTriggeredAt, nextRunAt)
	if err != nil {
		return fmt.Errorf("update Agent Schedule runtime state: %w", err)
	}
	return nil
}

type scheduleRow interface {
	Scan(...any) error
}

func scanSchedule(row scheduleRow) (agentrun.AgentSchedule, error) {
	var schedule agentrun.AgentSchedule
	var dailyTimes []byte
	var inputPayload []byte
	err := row.Scan(
		&schedule.ID, &schedule.AgentKey, &schedule.AgentVersion, &schedule.Type,
		&schedule.CronExpression, &dailyTimes, &inputPayload, &schedule.Enabled,
		&schedule.LastTriggered, &schedule.NextRun, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		return agentrun.AgentSchedule{}, err
	}
	if len(dailyTimes) > 0 {
		if err := json.Unmarshal(dailyTimes, &schedule.DailyTimes); err != nil {
			return agentrun.AgentSchedule{}, fmt.Errorf("decode Agent Schedule daily times: %w", err)
		}
	}
	schedule.InputPayload = append(schedule.InputPayload[:0], inputPayload...)
	return schedule, nil
}

func scanSchedules(rows pgx.Rows) ([]agentrun.AgentSchedule, error) {
	schedules := make([]agentrun.AgentSchedule, 0)
	for rows.Next() {
		schedule, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan Agent Schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list Agent Schedules: %w", err)
	}
	return schedules, nil
}
