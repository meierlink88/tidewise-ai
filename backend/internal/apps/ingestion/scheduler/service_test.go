package scheduler

import (
	"context"
	"testing"
	"time"

	ingestionruntime "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/runtime"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestServiceRunOnceSkipsWhenDisabled(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	runner := &fakeRunner{}
	service := NewService(repo, runner, ServiceOptions{
		Clock: fixedClock(time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)),
	})

	report, err := service.RunOnce(context.Background(), domain.SchedulerTriggerManualOnce)
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if report.Status != domain.SchedulerRunStatusSkipped {
		t.Fatalf("Status = %q, want skipped", report.Status)
	}
	if runner.called {
		t.Fatal("runner must not be called when scheduler is disabled")
	}
}

func TestServiceRunOnceCreatesRunAndRecordsSourceResults(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	config := domain.SchedulerConfig{
		ID:              "default",
		Enabled:         true,
		Mode:            domain.SchedulerModeInterval,
		IntervalMinutes: 60,
		Concurrency:     2,
		BatchSize:       20,
		TimeoutSeconds:  180,
		Timezone:        "Asia/Shanghai",
		SourceFilter: domain.SchedulerSourceFilter{
			ProviderKey:   "llm_web_research",
			IngestChannel: "ai_web_research",
			SourceType:    "news",
		},
	}
	if _, err := repo.SaveSchedulerConfig(context.Background(), config); err != nil {
		t.Fatalf("SaveSchedulerConfig() error = %v", err)
	}
	started := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
	runner := &fakeRunner{report: ingestionruntime.IngestionReport{
		Total:     2,
		Succeeded: 1,
		Failed:    1,
		Errors:    []string{"source-2 failed"},
		SourceResults: []ingestionruntime.SourceIngestionResult{
			{
				SourceID:          "source-1",
				Status:            ingestionruntime.SourceIngestionStatusSucceeded,
				DocumentsWritten:  3,
				StartedAt:         started,
				FinishedAt:        started.Add(time.Second),
				DurationMillis:    1000,
			},
			{
				SourceID:       "source-2",
				Status:         ingestionruntime.SourceIngestionStatusFailed,
				Error:          "source-2 failed",
				StartedAt:      started,
				FinishedAt:     started.Add(time.Second),
				DurationMillis: 1000,
			},
		},
	}}
	service := NewService(repo, runner, ServiceOptions{Clock: fixedClock(started)})

	report, err := service.RunOnce(context.Background(), domain.SchedulerTriggerManualOnce)
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if report.Status != domain.SchedulerRunStatusPartial {
		t.Fatalf("Status = %q, want partial", report.Status)
	}
	if runner.filter.SourceID != "" {
		t.Fatalf("runner filter SourceID = %q, want empty", runner.filter.SourceID)
	}
	if runner.filter.ProviderKey != "llm_web_research" || runner.filter.IngestChannel != "ai_web_research" || runner.filter.SourceType != "news" {
		t.Fatalf("runner filter = %+v, want configured global filter", runner.filter)
	}
	if len(repo.IngestionRunSources(report.RunID)) != 2 {
		t.Fatalf("run source results = %d, want 2", len(repo.IngestionRunSources(report.RunID)))
	}
}

type fakeRunner struct {
	called bool
	filter repositories.SourceCatalogFilter
	report ingestionruntime.IngestionReport
}

func (r *fakeRunner) Run(_ context.Context, filter repositories.SourceCatalogFilter) ingestionruntime.IngestionReport {
	r.called = true
	r.filter = filter
	return r.report
}

func fixedClock(value time.Time) func() time.Time {
	return func() time.Time {
		return value
	}
}
