package scheduler

import (
	"context"
	"strings"
	"time"

	ingestionruntime "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/runtime"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type Repository interface {
	repositories.SchedulerRepository
}

type Runner interface {
	Run(context.Context, repositories.SourceCatalogFilter, RunOptions) ingestionruntime.IngestionReport
}

type ServiceOptions struct {
	Clock func() time.Time
}

type RunOptions struct {
	Concurrency    int
	TimeoutSeconds int
}

type Service struct {
	repository Repository
	runner     Runner
	clock      func() time.Time
	planner    TriggerPlanner
}

type RunReport struct {
	RunID     string
	Status    domain.SchedulerRunStatus
	Total     int
	Succeeded int
	Failed    int
	Skipped   int
	Errors    []string
}

func NewService(repository Repository, runner Runner, options ServiceOptions) Service {
	clock := options.Clock
	if clock == nil {
		clock = time.Now
	}
	return Service{
		repository: repository,
		runner:     runner,
		clock:      clock,
		planner:    TriggerPlanner{},
	}
}

func (s Service) RunOnce(ctx context.Context, trigger domain.SchedulerTriggerType) (RunReport, error) {
	config, err := s.repository.LoadSchedulerConfig(ctx)
	if err != nil {
		return RunReport{}, err
	}
	if !config.Enabled {
		return RunReport{Status: domain.SchedulerRunStatusSkipped}, nil
	}
	if err := config.Validate(); err != nil {
		return RunReport{}, err
	}
	return s.runWithConfig(ctx, trigger, config)
}

func (s Service) RunDue(ctx context.Context) (RunReport, error) {
	config, err := s.repository.LoadSchedulerConfig(ctx)
	if err != nil {
		return RunReport{}, err
	}
	now := s.clock()
	if !s.planner.ShouldRun(now, config, config.LastRunAt) {
		return RunReport{Status: domain.SchedulerRunStatusSkipped}, nil
	}
	trigger := domain.SchedulerTriggerInterval
	if config.Mode == domain.SchedulerModeFixedTimes {
		trigger = domain.SchedulerTriggerFixedTime
	}
	return s.runWithConfig(ctx, trigger, config)
}

func (s Service) runWithConfig(ctx context.Context, trigger domain.SchedulerTriggerType, config domain.SchedulerConfig) (RunReport, error) {
	startedAt := s.clock()
	run, err := s.repository.CreateIngestionRun(ctx, domain.IngestionRun{
		ID:              runID(startedAt),
		TriggerType:     trigger,
		Status:          domain.SchedulerRunStatusRunning,
		StartedAt:       startedAt,
		SchedulerConfig: schedulerConfigSnapshot(config),
	})
	if err != nil {
		return RunReport{}, err
	}

	runtimeReport := s.runner.Run(ctx, sourceCatalogFilter(config), RunOptions{
		Concurrency:    config.Concurrency,
		TimeoutSeconds: config.TimeoutSeconds,
	})
	for _, result := range runtimeReport.SourceResults {
		if err := s.repository.RecordIngestionRunSource(ctx, sourceRunResult(run.ID, result)); err != nil {
			return RunReport{}, err
		}
	}

	finishedAt := s.clock()
	completed := run
	completed.Status = runStatus(runtimeReport)
	completed.FinishedAt = &finishedAt
	completed.TotalSources = runtimeReport.Total
	completed.SucceededSources = runtimeReport.Succeeded
	completed.FailedSources = runtimeReport.Failed
	completed.SkippedSources = 0
	completed.ErrorSummary = strings.Join(runtimeReport.Errors, "; ")
	if err := s.repository.CompleteIngestionRun(ctx, completed); err != nil {
		return RunReport{}, err
	}

	return RunReport{
		RunID:     run.ID,
		Status:    completed.Status,
		Total:     runtimeReport.Total,
		Succeeded: runtimeReport.Succeeded,
		Failed:    runtimeReport.Failed,
		Skipped:   0,
		Errors:    append([]string(nil), runtimeReport.Errors...),
	}, nil
}

func sourceCatalogFilter(config domain.SchedulerConfig) repositories.SourceCatalogFilter {
	return repositories.SourceCatalogFilter{
		ProviderKey:   config.SourceFilter.ProviderKey,
		IngestChannel: config.SourceFilter.IngestChannel,
		SourceType:    config.SourceFilter.SourceType,
		Limit:         config.BatchSize,
	}
}

func sourceRunResult(runID string, result ingestionruntime.SourceIngestionResult) domain.IngestionRunSource {
	status := domain.SchedulerSourceRunStatusSucceeded
	if result.Status == ingestionruntime.SourceIngestionStatusFailed {
		status = domain.SchedulerSourceRunStatusFailed
	}
	finishedAt := result.FinishedAt
	return domain.IngestionRunSource{
		ID:                 runID + "-" + result.SourceID,
		RunID:              runID,
		SourceID:           result.SourceID,
		Status:             status,
		DocumentsWritten:   result.DocumentsWritten,
		DocumentsDuplicate: result.DocumentsDuplicate,
		ErrorMessage:       result.Error,
		StartedAt:          result.StartedAt,
		FinishedAt:         &finishedAt,
		DurationMillis:     result.DurationMillis,
	}
}

func runStatus(report ingestionruntime.IngestionReport) domain.SchedulerRunStatus {
	if report.Total == 0 {
		return domain.SchedulerRunStatusSkipped
	}
	if report.Failed == 0 {
		return domain.SchedulerRunStatusSucceeded
	}
	if report.Succeeded == 0 {
		return domain.SchedulerRunStatusFailed
	}
	return domain.SchedulerRunStatusPartial
}

func schedulerConfigSnapshot(config domain.SchedulerConfig) map[string]any {
	return map[string]any{
		"enabled":          config.Enabled,
		"mode":             string(config.Mode),
		"interval_minutes": config.IntervalMinutes,
		"fixed_times":      append([]string(nil), config.FixedTimes...),
		"concurrency":      config.Concurrency,
		"batch_size":       config.BatchSize,
		"timeout_seconds":  config.TimeoutSeconds,
		"source_filter": map[string]string{
			"provider_key":   config.SourceFilter.ProviderKey,
			"ingest_channel": config.SourceFilter.IngestChannel,
			"source_type":    config.SourceFilter.SourceType,
		},
		"timezone": config.Timezone,
	}
}

func runID(value time.Time) string {
	return "ingestion-run-" + value.UTC().Format("20060102150405.000000000")
}
