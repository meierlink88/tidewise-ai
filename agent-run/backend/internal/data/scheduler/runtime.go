package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	bizschedule "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/scheduling"
)

type RuntimeStore interface {
	UpdateAgentScheduleRuntime(context.Context, string, *time.Time, *time.Time) error
}

type Runtime struct {
	store     RuntimeStore
	runners   map[string]bizschedule.AgentRunner
	location  *time.Location
	clock     clockwork.Clock
	scheduler gocron.Scheduler
	mu        sync.RWMutex
	jobs      map[string]gocron.Job
	started   bool
	lifecycle context.Context
	cancel    context.CancelFunc
	events    EventLogger
}

type Option func(*options)

type options struct {
	clock  clockwork.Clock
	events EventLogger
}

type EventLogger interface {
	Error(eventCode string, scheduleID string)
}

type discardEventLogger struct{}

func (discardEventLogger) Error(string, string) {}

func WithClock(clock clockwork.Clock) Option {
	return func(options *options) {
		options.clock = clock
	}
}

func WithEventLogger(logger EventLogger) Option {
	return func(options *options) {
		if logger != nil {
			options.events = logger
		}
	}
}

func NewRuntime(
	store RuntimeStore,
	location *time.Location,
	runners map[string]bizschedule.AgentRunner,
	runtimeOptions ...Option,
) (*Runtime, error) {
	if store == nil {
		return nil, errors.New("Agent Schedule Runtime Store is required")
	}
	if location == nil {
		return nil, errors.New("Agent Schedule Location is required")
	}
	if len(runners) == 0 {
		return nil, errors.New("at least one Agent Runner is required")
	}
	configured := options{clock: clockwork.NewRealClock(), events: discardEventLogger{}}
	for _, option := range runtimeOptions {
		option(&configured)
	}
	if configured.clock == nil {
		return nil, errors.New("Agent Schedule Clock is required")
	}
	engine, err := gocron.NewScheduler(gocron.WithLocation(location), gocron.WithClock(configured.clock))
	if err != nil {
		return nil, fmt.Errorf("create Agent Scheduler: %w", err)
	}
	lifecycle, cancel := context.WithCancel(context.Background())
	return &Runtime{
		store: store, runners: runners, location: location, clock: configured.clock,
		scheduler: engine, jobs: make(map[string]gocron.Job),
		lifecycle: lifecycle, cancel: cancel, events: configured.events,
	}, nil
}

func (r *Runtime) ValidateCronExpression(expression string) error {
	validator := gocron.NewDefaultCron(false)
	if err := validator.IsValid(expression, r.location, r.clock.Now()); err != nil {
		return errors.New("Cron expression is invalid")
	}
	return nil
}

func (r *Runtime) Start(ctx context.Context, schedules []agentrun.AgentSchedule) error {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()
	for _, schedule := range schedules {
		if err := r.register(schedule); err != nil {
			return err
		}
	}
	r.scheduler.Start()
	r.mu.Lock()
	r.started = true
	jobs := make(map[string]gocron.Job, len(r.jobs))
	for id, job := range r.jobs {
		jobs[id] = job
	}
	r.mu.Unlock()
	for id, job := range jobs {
		if err := r.persistNextRun(ctx, id, job); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) Sync(ctx context.Context, schedule agentrun.AgentSchedule) error {
	scheduleID, err := uuid.Parse(schedule.ID)
	if err != nil {
		return errors.New("Agent Schedule ID is invalid")
	}
	r.mu.Lock()
	_, exists := r.jobs[schedule.ID]
	r.mu.Unlock()
	if exists {
		if err := r.scheduler.RemoveJob(scheduleID); err != nil {
			return fmt.Errorf("remove previous Agent Schedule Job: %w", err)
		}
		r.mu.Lock()
		delete(r.jobs, schedule.ID)
		r.mu.Unlock()
	}
	if !schedule.Enabled {
		return r.store.UpdateAgentScheduleRuntime(ctx, schedule.ID, nil, nil)
	}
	if err := r.register(schedule); err != nil {
		return err
	}
	r.mu.RLock()
	job := r.jobs[schedule.ID]
	r.mu.RUnlock()
	return r.persistNextRun(ctx, schedule.ID, job)
}

func (r *Runtime) Shutdown() error {
	r.cancel()
	return r.scheduler.Shutdown()
}

func (r *Runtime) register(schedule agentrun.AgentSchedule) error {
	definition, err := r.jobDefinition(schedule)
	if err != nil {
		return err
	}
	scheduleID, err := uuid.Parse(schedule.ID)
	if err != nil {
		return fmt.Errorf("Agent Schedule ID is invalid")
	}
	job, err := r.scheduler.NewJob(
		definition,
		gocron.NewTask(func() { r.fire(schedule) }),
		gocron.WithIdentifier(scheduleID),
		gocron.WithName("agent-schedule:"+schedule.AgentKey),
	)
	if err != nil {
		return fmt.Errorf("register Agent Schedule: %w", err)
	}
	r.mu.Lock()
	r.jobs[schedule.ID] = job
	r.mu.Unlock()
	return nil
}

func (r *Runtime) fire(schedule agentrun.AgentSchedule) {
	triggeredAt := r.clock.Now().UTC()
	runner := r.runners[schedule.AgentKey]
	if err := runner.Trigger(r.lifecycle, bizschedule.Trigger{
		ScheduleID: schedule.ID, AgentKey: schedule.AgentKey, AgentVersion: schedule.AgentVersion,
		InputPayload: append([]byte(nil), schedule.InputPayload...), TriggeredAt: triggeredAt,
	}); err != nil {
		r.events.Error("schedule_trigger_failed", schedule.ID)
	}
	r.mu.RLock()
	job := r.jobs[schedule.ID]
	r.mu.RUnlock()
	var nextRun *time.Time
	if job != nil {
		if next, err := job.NextRun(); err == nil {
			next = next.UTC()
			nextRun = &next
		}
	}
	if err := r.store.UpdateAgentScheduleRuntime(r.lifecycle, schedule.ID, &triggeredAt, nextRun); err != nil {
		r.events.Error("schedule_runtime_update_failed", schedule.ID)
	}
}

func (r *Runtime) persistNextRun(ctx context.Context, scheduleID string, job gocron.Job) error {
	next, err := job.NextRun()
	if err != nil {
		return fmt.Errorf("read Agent Schedule next run: %w", err)
	}
	next = next.UTC()
	return r.store.UpdateAgentScheduleRuntime(ctx, scheduleID, nil, &next)
}

func (r *Runtime) jobDefinition(schedule agentrun.AgentSchedule) (gocron.JobDefinition, error) {
	switch schedule.Type {
	case agentrun.ScheduleCron:
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
