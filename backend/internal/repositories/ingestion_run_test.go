package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestInMemoryRepositoryRecordsIngestionRuns(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	started := time.Now().Add(-time.Minute)
	finished := time.Now()
	run := domain.IngestionRun{
		ID:          "run-1",
		TriggerType: domain.SchedulerTriggerManualOnce,
		Status:      domain.SchedulerRunStatusRunning,
		StartedAt:   started,
	}

	if _, err := repo.CreateIngestionRun(context.Background(), run); err != nil {
		t.Fatalf("CreateIngestionRun() error = %v", err)
	}

	sourceResult := domain.IngestionRunSource{
		ID:                 "run-source-1",
		RunID:              "run-1",
		SourceID:           "source-1",
		Status:             domain.SchedulerSourceRunStatusSucceeded,
		DocumentsWritten:   5,
		DocumentsDuplicate: 2,
		StartedAt:          started,
		FinishedAt:         &finished,
		DurationMillis:     120,
	}
	if err := repo.RecordIngestionRunSource(context.Background(), sourceResult); err != nil {
		t.Fatalf("RecordIngestionRunSource() error = %v", err)
	}

	run.Status = domain.SchedulerRunStatusSucceeded
	run.FinishedAt = &finished
	run.TotalSources = 1
	run.SucceededSources = 1
	if err := repo.CompleteIngestionRun(context.Background(), run); err != nil {
		t.Fatalf("CompleteIngestionRun() error = %v", err)
	}

	runs, err := repo.RecentIngestionRuns(context.Background(), 5)
	if err != nil {
		t.Fatalf("RecentIngestionRuns() error = %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("recent runs length = %d, want 1", len(runs))
	}
	if runs[0].Status != domain.SchedulerRunStatusSucceeded {
		t.Fatalf("run status = %q, want succeeded", runs[0].Status)
	}
	if got := repo.IngestionRunSources("run-1"); len(got) != 1 {
		t.Fatalf("run source results length = %d, want 1", len(got))
	}
}
