package scheduling

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

type Trigger struct {
	ScheduleID   string
	AgentKey     string
	AgentVersion string
	InputPayload json.RawMessage
	TriggeredAt  time.Time
}

type AgentRunner interface {
	ValidateInput(context.Context, string, json.RawMessage) error
	ConfigurationReady(context.Context, string) error
	Trigger(context.Context, Trigger) error
}

type Store interface {
	GetAgentVersion(context.Context, string) (agentrun.AgentVersion, error)
	PutAgentSchedule(context.Context, agentrun.PutAgentScheduleInput) (agentrun.AgentSchedule, error)
	GetAgentSchedule(context.Context, string) (agentrun.AgentSchedule, error)
	ListAgentSchedules(context.Context) ([]agentrun.AgentSchedule, error)
	ListEnabledAgentSchedules(context.Context) ([]agentrun.AgentSchedule, error)
}

type Runtime interface {
	ValidateCronExpression(string) error
	Start(context.Context, []agentrun.AgentSchedule) error
	Sync(context.Context, agentrun.AgentSchedule) error
	Shutdown() error
}

type Service struct {
	store   Store
	runners map[string]AgentRunner
	runtime Runtime
	now     func() time.Time
	mu      sync.RWMutex
	started bool
}

type Option func(*Service)

func WithNow(now func() time.Time) Option {
	return func(service *Service) {
		service.now = now
	}
}

func New(store Store, runners map[string]AgentRunner, runtime Runtime, options ...Option) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("Agent Schedule Store is required")
	}
	if len(runners) == 0 {
		return nil, fmt.Errorf("at least one Agent Runner is required")
	}
	if runtime == nil {
		return nil, fmt.Errorf("Agent Schedule Runtime is required")
	}
	service := &Service{store: store, runners: runners, runtime: runtime, now: time.Now}
	for _, option := range options {
		option(service)
	}
	if service.now == nil {
		return nil, fmt.Errorf("Agent Schedule Clock is required")
	}
	return service, nil
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.RLock()
	started := s.started
	s.mu.RUnlock()
	if started {
		return nil
	}
	schedules, err := s.store.ListEnabledAgentSchedules(ctx)
	if err != nil {
		return fmt.Errorf("load enabled Agent Schedules: %w", err)
	}
	for _, schedule := range schedules {
		if err := s.validate(ctx, schedule, true); err != nil {
			return err
		}
	}
	if err := s.runtime.Start(ctx, schedules); err != nil {
		return err
	}
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
	return nil
}

func (s *Service) List(ctx context.Context) ([]agentrun.AgentSchedule, error) {
	return s.store.ListAgentSchedules(ctx)
}

func (s *Service) Get(ctx context.Context, agentKey string) (agentrun.AgentSchedule, error) {
	if _, exists := s.runners[agentKey]; !exists {
		return agentrun.AgentSchedule{}, fmt.Errorf("%w: %s", agentrun.ErrAgentNotRegistered, agentKey)
	}
	return s.store.GetAgentSchedule(ctx, agentKey)
}

func (s *Service) Put(ctx context.Context, input agentrun.PutAgentScheduleInput) (agentrun.AgentSchedule, error) {
	normalized, err := s.normalize(ctx, input)
	if err != nil {
		return agentrun.AgentSchedule{}, err
	}
	saved, err := s.store.PutAgentSchedule(ctx, normalized)
	if err != nil {
		return agentrun.AgentSchedule{}, err
	}
	s.mu.RLock()
	started := s.started
	s.mu.RUnlock()
	if !started {
		return saved, nil
	}
	if err := s.runtime.Sync(ctx, saved); err != nil {
		return saved, fmt.Errorf("%w: %v", agentrun.ErrScheduleRuntimeSync, err)
	}
	return s.store.GetAgentSchedule(ctx, saved.AgentKey)
}

func (s *Service) Patch(ctx context.Context, agentKey string, patch agentrun.PatchAgentScheduleInput) (agentrun.AgentSchedule, error) {
	current, err := s.Get(ctx, agentKey)
	if err != nil {
		return agentrun.AgentSchedule{}, err
	}
	input := agentrun.PutAgentScheduleInput{
		AgentKey: current.AgentKey, AgentVersion: current.AgentVersion,
		Type: current.Type, CronExpression: current.CronExpression,
		DailyTimes:   append([]string(nil), current.DailyTimes...),
		InputPayload: append(json.RawMessage(nil), current.InputPayload...),
		Enabled:      current.Enabled, UpdatedAt: patch.UpdatedAt,
	}
	if patch.AgentVersion != nil {
		input.AgentVersion = *patch.AgentVersion
	}
	if patch.Type != nil {
		if *patch.Type != input.Type {
			input.CronExpression = ""
			input.DailyTimes = nil
		}
		input.Type = *patch.Type
	}
	if patch.CronExpression != nil {
		input.CronExpression = *patch.CronExpression
	}
	if patch.DailyTimes != nil {
		input.DailyTimes = append([]string(nil), (*patch.DailyTimes)...)
	}
	if patch.InputPayload != nil {
		input.InputPayload = append(json.RawMessage(nil), (*patch.InputPayload)...)
	}
	if patch.Enabled != nil {
		input.Enabled = *patch.Enabled
	}
	return s.Put(ctx, input)
}

func (s *Service) Shutdown() error {
	err := s.runtime.Shutdown()
	s.mu.Lock()
	s.started = false
	s.mu.Unlock()
	return err
}

func (s *Service) Ready(context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.started {
		return fmt.Errorf("Agent Schedule Runtime is not started")
	}
	return nil
}

func (s *Service) normalize(ctx context.Context, input agentrun.PutAgentScheduleInput) (agentrun.PutAgentScheduleInput, error) {
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = s.now().UTC()
	} else {
		input.UpdatedAt = input.UpdatedAt.UTC()
	}
	input.AgentKey = strings.TrimSpace(input.AgentKey)
	input.AgentVersion = strings.TrimSpace(input.AgentVersion)
	input.CronExpression = strings.TrimSpace(input.CronExpression)
	if len(input.InputPayload) == 0 || len(input.InputPayload) > 64*1024 {
		return agentrun.PutAgentScheduleInput{}, fmt.Errorf("%w: Agent Schedule Input must be at most 64 KiB", agentrun.ErrInvalidSchedule)
	}
	var inputObject map[string]json.RawMessage
	if err := json.Unmarshal(input.InputPayload, &inputObject); err != nil || inputObject == nil {
		return agentrun.PutAgentScheduleInput{}, fmt.Errorf("%w: Agent Schedule Input must be a JSON object", agentrun.ErrInvalidSchedule)
	}
	if input.Type == agentrun.ScheduleDaily {
		seen := make(map[string]struct{}, len(input.DailyTimes))
		times := make([]string, 0, len(input.DailyTimes))
		for _, value := range input.DailyTimes {
			value = strings.TrimSpace(value)
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			times = append(times, value)
		}
		sort.Strings(times)
		input.DailyTimes = times
		input.CronExpression = ""
	} else if input.Type == agentrun.ScheduleCron {
		input.DailyTimes = nil
	}
	schedule := agentrun.AgentSchedule{
		AgentKey: input.AgentKey, AgentVersion: input.AgentVersion, Type: input.Type,
		CronExpression: input.CronExpression, DailyTimes: input.DailyTimes,
		InputPayload: input.InputPayload, Enabled: input.Enabled,
	}
	if err := s.validate(ctx, schedule, input.Enabled); err != nil {
		return agentrun.PutAgentScheduleInput{}, err
	}
	return input, nil
}

func (s *Service) validate(ctx context.Context, schedule agentrun.AgentSchedule, requireConfiguration bool) error {
	runner, exists := s.runners[schedule.AgentKey]
	if !exists {
		return fmt.Errorf("%w: %s", agentrun.ErrAgentNotRegistered, schedule.AgentKey)
	}
	version, err := s.store.GetAgentVersion(ctx, schedule.AgentVersion)
	if err != nil || version.AgentKey != schedule.AgentKey {
		return fmt.Errorf("%w: Agent Version does not belong to Agent", agentrun.ErrInvalidSchedule)
	}
	if err := runner.ValidateInput(ctx, schedule.AgentVersion, schedule.InputPayload); err != nil {
		return fmt.Errorf("%w: validate Agent Schedule Input", agentrun.ErrInvalidSchedule)
	}
	if err := s.validateDefinition(schedule); err != nil {
		return fmt.Errorf("%w: %v", agentrun.ErrInvalidSchedule, err)
	}
	if requireConfiguration {
		if err := runner.ConfigurationReady(ctx, schedule.AgentVersion); err != nil {
			return fmt.Errorf("%w: Agent Schedule configuration is not ready", agentrun.ErrInvalidSchedule)
		}
	}
	return nil
}

func (s *Service) validateDefinition(schedule agentrun.AgentSchedule) error {
	switch schedule.Type {
	case agentrun.ScheduleCron:
		if len(strings.Fields(schedule.CronExpression)) != 5 {
			return fmt.Errorf("Cron expression must contain exactly five fields")
		}
		return s.runtime.ValidateCronExpression(schedule.CronExpression)
	case agentrun.ScheduleDaily:
		if len(schedule.DailyTimes) == 0 {
			return fmt.Errorf("Daily Schedule requires at least one time")
		}
		for _, value := range schedule.DailyTimes {
			if len(value) != len("00:00") || value[2] != ':' {
				return fmt.Errorf("Daily time must use HH:MM")
			}
			hour, hourErr := strconv.Atoi(value[:2])
			minute, minuteErr := strconv.Atoi(value[3:])
			if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				return fmt.Errorf("Daily time is invalid")
			}
		}
		return nil
	default:
		return fmt.Errorf("Agent Schedule type is invalid")
	}
}
