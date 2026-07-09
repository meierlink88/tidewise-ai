package scheduler

import (
	"sort"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type TriggerPlanner struct{}

func (TriggerPlanner) ShouldRun(now time.Time, config domain.SchedulerConfig, lastRunAt *time.Time) bool {
	if !config.Enabled {
		return false
	}
	if err := config.Validate(); err != nil {
		return false
	}
	if lastRunAt == nil {
		return true
	}

	switch config.Mode {
	case domain.SchedulerModeInterval:
		return !now.Before(lastRunAt.Add(time.Duration(config.IntervalMinutes) * time.Minute))
	case domain.SchedulerModeFixedTimes:
		trigger := nextFixedTriggerAfter(*lastRunAt, config)
		return !trigger.IsZero() && !trigger.After(now)
	default:
		return false
	}
}

func (TriggerPlanner) NextTrigger(now time.Time, config domain.SchedulerConfig, lastRunAt *time.Time) time.Time {
	if !config.Enabled {
		return time.Time{}
	}
	if err := config.Validate(); err != nil {
		return time.Time{}
	}

	switch config.Mode {
	case domain.SchedulerModeInterval:
		if lastRunAt == nil {
			return now
		}
		return lastRunAt.Add(time.Duration(config.IntervalMinutes) * time.Minute)
	case domain.SchedulerModeFixedTimes:
		if lastRunAt != nil {
			trigger := nextFixedTriggerAfter(*lastRunAt, config)
			if !trigger.IsZero() && !trigger.Before(now) {
				return trigger
			}
		}
		return nextFixedTriggerAtOrAfter(now, config)
	default:
		return time.Time{}
	}
}

func nextFixedTriggerAfter(value time.Time, config domain.SchedulerConfig) time.Time {
	location := schedulerLocation(config)
	reference := value.In(location)
	for dayOffset := 0; dayOffset <= 1; dayOffset++ {
		day := reference.AddDate(0, 0, dayOffset)
		for _, fixed := range sortedFixedTimes(config.FixedTimes) {
			trigger := fixedTimeOnDay(day, fixed, location)
			if trigger.After(reference) {
				return trigger
			}
		}
	}
	return time.Time{}
}

func nextFixedTriggerAtOrAfter(value time.Time, config domain.SchedulerConfig) time.Time {
	location := schedulerLocation(config)
	reference := value.In(location)
	for dayOffset := 0; dayOffset <= 1; dayOffset++ {
		day := reference.AddDate(0, 0, dayOffset)
		for _, fixed := range sortedFixedTimes(config.FixedTimes) {
			trigger := fixedTimeOnDay(day, fixed, location)
			if !trigger.Before(reference) {
				return trigger
			}
		}
	}
	return time.Time{}
}

func fixedTimeOnDay(day time.Time, value string, location *time.Location) time.Time {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return time.Time{}
	}
	return time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0, location)
}

func sortedFixedTimes(values []string) []string {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}

func schedulerLocation(config domain.SchedulerConfig) *time.Location {
	location, err := time.LoadLocation(config.Timezone)
	if err != nil {
		return time.Local
	}
	return location
}
