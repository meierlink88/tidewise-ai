package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func (r *InMemoryRepository) CreateIngestionRun(_ context.Context, run domain.IngestionRun) (domain.IngestionRun, error) {
	if err := run.Validate(); err != nil {
		return domain.IngestionRun{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	run.UpdatedAt = now
	run.SchedulerConfig = cloneMap(run.SchedulerConfig)
	r.ingestionRuns[run.ID] = run
	return cloneIngestionRun(run), nil
}

func (r *InMemoryRepository) RecordIngestionRunSource(_ context.Context, result domain.IngestionRunSource) error {
	if err := result.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.ingestionRuns[result.RunID]; !ok {
		return fmt.Errorf("ingestion run %q not found", result.RunID)
	}
	now := time.Now()
	if result.CreatedAt.IsZero() {
		result.CreatedAt = now
	}
	result.UpdatedAt = now
	r.runSources[result.RunID] = append(r.runSources[result.RunID], result)
	return nil
}

func (r *InMemoryRepository) CompleteIngestionRun(_ context.Context, run domain.IngestionRun) error {
	if err := run.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.ingestionRuns[run.ID]; !ok {
		return fmt.Errorf("ingestion run %q not found", run.ID)
	}
	run.UpdatedAt = time.Now()
	run.SchedulerConfig = cloneMap(run.SchedulerConfig)
	r.ingestionRuns[run.ID] = run

	config := r.schedulerConfig
	config.LastRunID = run.ID
	config.LastRunAt = &run.StartedAt
	config.UpdatedAt = run.UpdatedAt
	r.schedulerConfig = config
	return nil
}

func (r *InMemoryRepository) RecentIngestionRuns(_ context.Context, limit int) ([]domain.IngestionRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limit <= 0 {
		limit = 20
	}
	runs := make([]domain.IngestionRun, 0, len(r.ingestionRuns))
	for _, run := range r.ingestionRuns {
		runs = append(runs, cloneIngestionRun(run))
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})
	if len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func (r *InMemoryRepository) IngestionRunSources(runID string) []domain.IngestionRunSource {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := r.runSources[runID]
	copied := make([]domain.IngestionRunSource, len(items))
	copy(copied, items)
	return copied
}

func cloneIngestionRun(run domain.IngestionRun) domain.IngestionRun {
	run.SchedulerConfig = cloneMap(run.SchedulerConfig)
	return run
}

func (r PostgresRepository) CreateIngestionRun(ctx context.Context, run domain.IngestionRun) (domain.IngestionRun, error) {
	run.ID = NormalizeUUID(run.ID)
	if err := run.Validate(); err != nil {
		return domain.IngestionRun{}, err
	}
	schedulerConfig, err := json.Marshal(cloneMap(run.SchedulerConfig))
	if err != nil {
		return domain.IngestionRun{}, fmt.Errorf("marshal run scheduler config: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
INSERT INTO ingestion_runs (
    id, trigger_type, status, started_at, finished_at, total_sources,
    succeeded_sources, failed_sources, skipped_sources, scheduler_config, error_summary
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10::jsonb, $11
) RETURNING id, trigger_type, status, started_at, finished_at, total_sources,
            succeeded_sources, failed_sources, skipped_sources, scheduler_config,
            error_summary, created_at, updated_at
`, run.ID, run.TriggerType, run.Status, run.StartedAt, nullTime(run.FinishedAt),
		run.TotalSources, run.SucceededSources, run.FailedSources, run.SkippedSources, string(schedulerConfig), run.ErrorSummary)

	created, err := scanIngestionRun(row)
	if err != nil {
		return domain.IngestionRun{}, fmt.Errorf("create ingestion run: %w", err)
	}
	return created, nil
}

func (r PostgresRepository) RecordIngestionRunSource(ctx context.Context, result domain.IngestionRunSource) error {
	result.ID = NormalizeUUID(result.ID)
	result.RunID = NormalizeUUID(result.RunID)
	result.SourceID = NormalizeUUID(result.SourceID)
	if err := result.Validate(); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx, `
INSERT INTO ingestion_run_sources (
    id, run_id, source_id, status, documents_written, documents_duplicate,
    error_message, started_at, finished_at, duration_millis
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10
) ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    documents_written = EXCLUDED.documents_written,
    documents_duplicate = EXCLUDED.documents_duplicate,
    error_message = EXCLUDED.error_message,
    started_at = EXCLUDED.started_at,
    finished_at = EXCLUDED.finished_at,
    duration_millis = EXCLUDED.duration_millis,
    updated_at = now()
`, result.ID, result.RunID, result.SourceID, result.Status, result.DocumentsWritten, result.DocumentsDuplicate,
		result.ErrorMessage, result.StartedAt, nullTime(result.FinishedAt), result.DurationMillis)
	if err != nil {
		return fmt.Errorf("record ingestion run source: %w", err)
	}
	return nil
}

func (r PostgresRepository) CompleteIngestionRun(ctx context.Context, run domain.IngestionRun) error {
	run.ID = NormalizeUUID(run.ID)
	if err := run.Validate(); err != nil {
		return err
	}
	schedulerConfig, err := json.Marshal(cloneMap(run.SchedulerConfig))
	if err != nil {
		return fmt.Errorf("marshal completed run scheduler config: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE ingestion_runs
SET status = $2,
    finished_at = $3,
    total_sources = $4,
    succeeded_sources = $5,
    failed_sources = $6,
    skipped_sources = $7,
    scheduler_config = $8::jsonb,
    error_summary = $9,
    updated_at = now()
WHERE id = $1
`, run.ID, run.Status, nullTime(run.FinishedAt), run.TotalSources, run.SucceededSources,
		run.FailedSources, run.SkippedSources, string(schedulerConfig), run.ErrorSummary)
	if err != nil {
		return fmt.Errorf("complete ingestion run: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read completed run affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("ingestion run %q not found", run.ID)
	}

	_, err = r.db.ExecContext(ctx, `
UPDATE ingestion_scheduler_configs
SET last_run_id = $1,
    last_run_at = $2,
    updated_at = now()
WHERE id = 'default'
`, run.ID, run.StartedAt)
	if err != nil {
		return fmt.Errorf("update scheduler last run: %w", err)
	}
	return nil
}

func (r PostgresRepository) RecentIngestionRuns(ctx context.Context, limit int) ([]domain.IngestionRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, trigger_type, status, started_at, finished_at, total_sources,
       succeeded_sources, failed_sources, skipped_sources, scheduler_config,
       error_summary, created_at, updated_at
FROM ingestion_runs
ORDER BY started_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent ingestion runs: %w", err)
	}
	defer rows.Close()

	runs := make([]domain.IngestionRun, 0)
	for rows.Next() {
		run, err := scanIngestionRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent ingestion runs: %w", err)
	}
	return runs, nil
}

func scanIngestionRun(scanner rawDocumentScanner) (domain.IngestionRun, error) {
	var run domain.IngestionRun
	var finishedAt sql.NullTime
	var schedulerConfigBytes []byte
	if err := scanner.Scan(
		&run.ID,
		&run.TriggerType,
		&run.Status,
		&run.StartedAt,
		&finishedAt,
		&run.TotalSources,
		&run.SucceededSources,
		&run.FailedSources,
		&run.SkippedSources,
		&schedulerConfigBytes,
		&run.ErrorSummary,
		&run.CreatedAt,
		&run.UpdatedAt,
	); err != nil {
		return domain.IngestionRun{}, err
	}
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	if len(schedulerConfigBytes) > 0 {
		if err := json.Unmarshal(schedulerConfigBytes, &run.SchedulerConfig); err != nil {
			return domain.IngestionRun{}, fmt.Errorf("parse run scheduler config: %w", err)
		}
	}
	if run.SchedulerConfig == nil {
		run.SchedulerConfig = map[string]any{}
	}
	return run, nil
}
