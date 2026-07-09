package scheduler

import (
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestTriggerPlannerIntervalMode(t *testing.T) {
	planner := TriggerPlanner{}
	config := domain.SchedulerConfig{
		ID:              "default",
		Enabled:         true,
		Mode:            domain.SchedulerModeInterval,
		IntervalMinutes: 30,
		Concurrency:     1,
		BatchSize:       10,
		TimeoutSeconds:  180,
		Timezone:        "Asia/Shanghai",
	}
	now := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
	lastRun := now.Add(-31 * time.Minute)

	if !planner.ShouldRun(now, config, &lastRun) {
		t.Fatal("ShouldRun() = false, want true after interval elapsed")
	}
	next := planner.NextTrigger(now, config, &lastRun)
	if !next.Equal(lastRun.Add(30 * time.Minute)) {
		t.Fatalf("NextTrigger() = %s, want %s", next, lastRun.Add(30*time.Minute))
	}

	lastRun = now.Add(-10 * time.Minute)
	if planner.ShouldRun(now, config, &lastRun) {
		t.Fatal("ShouldRun() = true, want false before interval elapsed")
	}
}

func TestTriggerPlannerDisabledConfigDoesNotRun(t *testing.T) {
	planner := TriggerPlanner{}
	config := domain.SchedulerConfig{
		ID:              "default",
		Enabled:         false,
		Mode:            domain.SchedulerModeInterval,
		IntervalMinutes: 30,
		Concurrency:     1,
		BatchSize:       10,
		TimeoutSeconds:  180,
		Timezone:        "Asia/Shanghai",
	}

	if planner.ShouldRun(time.Now(), config, nil) {
		t.Fatal("ShouldRun() = true, want false when scheduler is disabled")
	}
	if next := planner.NextTrigger(time.Now(), config, nil); !next.IsZero() {
		t.Fatalf("NextTrigger() = %s, want zero time when scheduler is disabled", next)
	}
}

func TestTriggerPlannerFixedTimes(t *testing.T) {
	location := mustLoadLocation(t, "Asia/Shanghai")
	planner := TriggerPlanner{}
	config := domain.SchedulerConfig{
		ID:             "default",
		Enabled:        true,
		Mode:           domain.SchedulerModeFixedTimes,
		FixedTimes:     []string{"09:00", "12:00", "15:00", "18:00", "21:00"},
		Concurrency:    1,
		BatchSize:      10,
		TimeoutSeconds: 180,
		Timezone:       "Asia/Shanghai",
	}
	now := time.Date(2026, 7, 9, 12, 0, 30, 0, location)
	lastRun := time.Date(2026, 7, 9, 8, 59, 0, 0, location)

	if !planner.ShouldRun(now, config, &lastRun) {
		t.Fatal("ShouldRun() = false, want true at fixed trigger time")
	}

	afterAll := time.Date(2026, 7, 9, 22, 0, 0, 0, location)
	next := planner.NextTrigger(afterAll, config, &lastRun)
	want := time.Date(2026, 7, 10, 9, 0, 0, 0, location)
	if !next.Equal(want) {
		t.Fatalf("NextTrigger() = %s, want %s", next, want)
	}
}

func TestTriggerPlannerFirstRunIsDue(t *testing.T) {
	planner := TriggerPlanner{}
	config := domain.SchedulerConfig{
		ID:              "default",
		Enabled:         true,
		Mode:            domain.SchedulerModeInterval,
		IntervalMinutes: 60,
		Concurrency:     1,
		BatchSize:       10,
		TimeoutSeconds:  180,
		Timezone:        "Asia/Shanghai",
	}

	if !planner.ShouldRun(time.Now(), config, nil) {
		t.Fatal("ShouldRun() = false, want first enabled run to be due")
	}
}

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()

	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return location
}
