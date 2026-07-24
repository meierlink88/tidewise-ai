package scheduling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/jonboulle/clockwork"
)

var (
	ErrAgentNotRegistered = errors.New("Agent is not registered")
	ErrInvalidSchedule    = errors.New("Agent Schedule is invalid")
	ErrRuntimeSync        = errors.New("Agent Schedule runtime synchronization failed")
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
	UpdateAgentScheduleRuntime(context.Context, string, *time.Time, *time.Time) error
}

type Service struct {
	store     Store
	location  *time.Location
	runners   map[string]AgentRunner
	clock     clockwork.Clock
	scheduler gocron.Scheduler
	mu        sync.RWMutex
	jobs      map[string]gocron.Job
	started   bool
}

type Option func(*options)

type options struct {
	clock clockwork.Clock
}

func WithClock(clock clockwork.Clock) Option {
	return func(options *options) {
		options.clock = clock
	}
}

func New(store Store, location *time.Location, runners map[string]AgentRunner, serviceOptions ...Option) (*Service, error) {
	if store == nil {
		return nil, errors.New("Agent Schedule Store is required")
	}
	if location == nil {
		return nil, errors.New("Agent Schedule Location is required")
	}
	if len(runners) == 0 {
		return nil, errors.New("at least one Agent Runner is required")
	}
	configured := options{clock: clockwork.NewRealClock()}
	for _, option := range serviceOptions {
		option(&configured)
	}
	if configured.clock == nil {
		return nil, errors.New("Agent Schedule Clock is required")
	}
	scheduler, err := gocron.NewScheduler(
		gocron.WithLocation(location),
		gocron.WithClock(configured.clock),
	)
	if err != nil {
		return nil, fmt.Errorf("create Agent Scheduler: %w", err)
	}
	return &Service{
		store: store, location: location, runners: runners, clock: configured.clock,
		scheduler: scheduler, jobs: make(map[string]gocron.Job),
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	schedules, err := s.store.ListEnabledAgentSchedules(ctx)
	if err != nil {
		return fmt.Errorf("load enabled Agent Schedules: %w", err)
	}
	for _, schedule := range schedules {
		if err := s.validate(ctx, schedule); err != nil {
			return err
		}
		if err := s.register(schedule); err != nil {
			return err
		}
	}
	s.scheduler.Start()
	s.mu.Lock()
	s.started = true
	jobs := make(map[string]gocron.Job, len(s.jobs))
	for id, job := range s.jobs {
		jobs[id] = job
	}
	s.mu.Unlock()
	for id, job := range jobs {
		if err := s.persistNextRun(ctx, id, job); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) List(ctx context.Context) ([]agentrun.AgentSchedule, error) {
	return s.store.ListAgentSchedules(ctx)
}

func (s *Service) Get(ctx context.Context, agentKey string) (agentrun.AgentSchedule, error) {
	if _, exists := s.runners[agentKey]; !exists {
		return agentrun.AgentSchedule{}, fmt.Errorf("%w: %s", ErrAgentNotRegistered, agentKey)
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
	if err := s.sync(ctx, saved); err != nil {
		return saved, fmt.Errorf("%w: %v", ErrRuntimeSync, err)
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
	return s.scheduler.Shutdown()
}

func (s *Service) validate(ctx context.Context, schedule agentrun.AgentSchedule) error {
	runner, exists := s.runners[schedule.AgentKey]
	if !exists {
		return fmt.Errorf("%w: %s", ErrAgentNotRegistered, schedule.AgentKey)
	}
	version, err := s.store.GetAgentVersion(ctx, schedule.AgentVersion)
	if err != nil || version.AgentKey != schedule.AgentKey {
		return fmt.Errorf("%w: Agent Version does not belong to Agent", ErrInvalidSchedule)
	}
	if err := runner.ValidateInput(ctx, schedule.AgentVersion, schedule.InputPayload); err != nil {
		return fmt.Errorf("%w: validate Agent Schedule Input: %v", ErrInvalidSchedule, err)
	}
	if err := runner.ConfigurationReady(ctx, schedule.AgentVersion); err != nil {
		return fmt.Errorf("%w: Agent Schedule configuration is not ready", ErrInvalidSchedule)
	}
	_, err = s.jobDefinition(schedule)
	return err
}

func (s *Service) normalize(ctx context.Context, input agentrun.PutAgentScheduleInput) (agentrun.PutAgentScheduleInput, error) {
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = s.clock.Now().UTC()
	} else {
		input.UpdatedAt = input.UpdatedAt.UTC()
	}
	input.AgentKey = strings.TrimSpace(input.AgentKey)
	input.AgentVersion = strings.TrimSpace(input.AgentVersion)
	input.CronExpression = strings.TrimSpace(input.CronExpression)
	if len(input.InputPayload) == 0 || len(input.InputPayload) > 64*1024 {
		return agentrun.PutAgentScheduleInput{}, fmt.Errorf("%w: Agent Schedule Input must be at most 64 KiB", ErrInvalidSchedule)
	}
	var inputObject map[string]json.RawMessage
	if err := json.Unmarshal(input.InputPayload, &inputObject); err != nil || inputObject == nil {
		return agentrun.PutAgentScheduleInput{}, fmt.Errorf("%w: Agent Schedule Input must be a JSON object", ErrInvalidSchedule)
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
	runner, exists := s.runners[input.AgentKey]
	if !exists {
		return agentrun.PutAgentScheduleInput{}, fmt.Errorf("%w: %s", ErrAgentNotRegistered, input.AgentKey)
	}
	version, err := s.store.GetAgentVersion(ctx, input.AgentVersion)
	if err != nil || version.AgentKey != input.AgentKey {
		return agentrun.PutAgentScheduleInput{}, fmt.Errorf("%w: Agent Version does not belong to Agent", ErrInvalidSchedule)
	}
	if err := runner.ValidateInput(ctx, input.AgentVersion, input.InputPayload); err != nil {
		return agentrun.PutAgentScheduleInput{}, fmt.Errorf("%w: validate Agent Schedule Input", ErrInvalidSchedule)
	}
	if _, err := s.jobDefinition(schedule); err != nil {
		return agentrun.PutAgentScheduleInput{}, fmt.Errorf("%w: %v", ErrInvalidSchedule, err)
	}
	if input.Enabled {
		if err := runner.ConfigurationReady(ctx, input.AgentVersion); err != nil {
			return agentrun.PutAgentScheduleInput{}, fmt.Errorf("%w: Agent Schedule configuration is not ready", ErrInvalidSchedule)
		}
	}
	return input, nil
}

func (s *Service) sync(ctx context.Context, schedule agentrun.AgentSchedule) error {
	scheduleID, err := uuid.Parse(schedule.ID)
	if err != nil {
		return errors.New("Agent Schedule ID is invalid")
	}
	s.mu.Lock()
	_, exists := s.jobs[schedule.ID]
	s.mu.Unlock()
	if exists {
		if err := s.scheduler.RemoveJob(scheduleID); err != nil {
			return fmt.Errorf("remove previous Agent Schedule Job: %w", err)
		}
		s.mu.Lock()
		delete(s.jobs, schedule.ID)
		s.mu.Unlock()
	}
	if !schedule.Enabled {
		return s.store.UpdateAgentScheduleRuntime(ctx, schedule.ID, nil, nil)
	}
	if err := s.register(schedule); err != nil {
		return err
	}
	s.mu.RLock()
	job := s.jobs[schedule.ID]
	s.mu.RUnlock()
	return s.persistNextRun(ctx, schedule.ID, job)
}

func (s *Service) register(schedule agentrun.AgentSchedule) error {
	definition, err := s.jobDefinition(schedule)
	if err != nil {
		return err
	}
	scheduleID, err := uuid.Parse(schedule.ID)
	if err != nil {
		return fmt.Errorf("Agent Schedule ID is invalid")
	}
	job, err := s.scheduler.NewJob(
		definition,
		gocron.NewTask(func() {
			s.fire(schedule)
		}),
		gocron.WithIdentifier(scheduleID),
		gocron.WithName("agent-schedule:"+schedule.AgentKey),
	)
	if err != nil {
		return fmt.Errorf("register Agent Schedule: %w", err)
	}
	s.mu.Lock()
	s.jobs[schedule.ID] = job
	s.mu.Unlock()
	return nil
}

func (s *Service) fire(schedule agentrun.AgentSchedule) {
	triggeredAt := s.clock.Now().UTC()
	runner := s.runners[schedule.AgentKey]
	_ = runner.Trigger(context.Background(), Trigger{
		ScheduleID: schedule.ID, AgentKey: schedule.AgentKey, AgentVersion: schedule.AgentVersion,
		InputPayload: append(json.RawMessage(nil), schedule.InputPayload...), TriggeredAt: triggeredAt,
	})
	s.mu.RLock()
	job := s.jobs[schedule.ID]
	s.mu.RUnlock()
	var nextRun *time.Time
	if job != nil {
		if next, err := job.NextRun(); err == nil {
			next = next.UTC()
			nextRun = &next
		}
	}
	_ = s.store.UpdateAgentScheduleRuntime(context.Background(), schedule.ID, &triggeredAt, nextRun)
}

func (s *Service) persistNextRun(ctx context.Context, scheduleID string, job gocron.Job) error {
	next, err := job.NextRun()
	if err != nil {
		return fmt.Errorf("read Agent Schedule next run: %w", err)
	}
	next = next.UTC()
	if err := s.store.UpdateAgentScheduleRuntime(ctx, scheduleID, nil, &next); err != nil {
		return err
	}
	return nil
}

func (s *Service) jobDefinition(schedule agentrun.AgentSchedule) (gocron.JobDefinition, error) {
	switch schedule.Type {
	case agentrun.ScheduleCron:
		if len(strings.Fields(schedule.CronExpression)) != 5 {
			return nil, errors.New("Cron expression must contain exactly five fields")
		}
		validator := gocron.NewDefaultCron(false)
		if err := validator.IsValid(schedule.CronExpression, s.location, s.clock.Now()); err != nil {
			return nil, fmt.Errorf("Cron expression is invalid")
		}
		return gocron.CronJob(schedule.CronExpression, false), nil
	case agentrun.ScheduleDaily:
		atTimes := make([]gocron.AtTime, 0, len(schedule.DailyTimes))
		for _, value := range schedule.DailyTimes {
			if len(value) != len("00:00") || value[2] != ':' {
				return nil, errors.New("Daily time must use HH:MM")
			}
			parts := strings.Split(value, ":")
			hour, hourErr := strconv.Atoi(parts[0])
			minute, minuteErr := strconv.Atoi(parts[1])
			if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				return nil, errors.New("Daily time is invalid")
			}
			atTimes = append(atTimes, gocron.NewAtTime(uint(hour), uint(minute), 0))
		}
		if len(atTimes) == 0 {
			return nil, errors.New("Daily Schedule requires at least one time")
		}
		return gocron.DailyJob(1, gocron.NewAtTimes(atTimes[0], atTimes[1:]...)), nil
	default:
		return nil, errors.New("Agent Schedule type is invalid")
	}
}
