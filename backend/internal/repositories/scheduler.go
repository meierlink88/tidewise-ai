package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type SchedulerRepository interface {
	LoadSchedulerConfig(context.Context) (domain.SchedulerConfig, error)
	SaveSchedulerConfig(context.Context, domain.SchedulerConfig) (domain.SchedulerConfig, error)
	CreateIngestionRun(context.Context, domain.IngestionRun) (domain.IngestionRun, error)
	RecordIngestionRunSource(context.Context, domain.IngestionRunSource) error
	CompleteIngestionRun(context.Context, domain.IngestionRun) error
	RecentIngestionRuns(context.Context, int) ([]domain.IngestionRun, error)
}

func (r *InMemoryRepository) LoadSchedulerConfig(_ context.Context) (domain.SchedulerConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return cloneSchedulerConfig(r.schedulerConfig), nil
}

func (r *InMemoryRepository) SaveSchedulerConfig(_ context.Context, config domain.SchedulerConfig) (domain.SchedulerConfig, error) {
	config = normalizeSchedulerConfig(config)
	if err := config.Validate(); err != nil {
		return domain.SchedulerConfig{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if config.ConfigVersion <= 0 {
		config.ConfigVersion = 1
	}
	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now
	r.schedulerConfig = cloneSchedulerConfig(config)
	return cloneSchedulerConfig(r.schedulerConfig), nil
}

func defaultSchedulerConfig() domain.SchedulerConfig {
	return domain.SchedulerConfig{
		ID:              "default",
		Enabled:         false,
		Mode:            domain.SchedulerModeInterval,
		IntervalMinutes: 60,
		Concurrency:     1,
		BatchSize:       10,
		TimeoutSeconds:  180,
		SourceFilter:    domain.SchedulerSourceFilter{},
		Timezone:        "Asia/Shanghai",
		ConfigVersion:   1,
	}
}

func normalizeSchedulerConfig(config domain.SchedulerConfig) domain.SchedulerConfig {
	if config.ID == "" {
		config.ID = "default"
	}
	if config.Mode == "" {
		config.Mode = domain.SchedulerModeInterval
	}
	if config.Mode == domain.SchedulerModeInterval && config.IntervalMinutes == 0 {
		config.IntervalMinutes = 60
	}
	if config.Concurrency == 0 {
		config.Concurrency = 1
	}
	if config.BatchSize == 0 {
		config.BatchSize = 10
	}
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 180
	}
	if config.Timezone == "" {
		config.Timezone = "Asia/Shanghai"
	}
	if config.ConfigVersion == 0 {
		config.ConfigVersion = 1
	}
	return config
}

func cloneSchedulerConfig(config domain.SchedulerConfig) domain.SchedulerConfig {
	config.FixedTimes = append([]string(nil), config.FixedTimes...)
	return config
}

func (r PostgresRepository) LoadSchedulerConfig(ctx context.Context) (domain.SchedulerConfig, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, enabled, mode, interval_minutes, fixed_times, concurrency,
       batch_size, timeout_seconds, source_filter, timezone, config_version,
       last_run_id, last_run_at, created_at, updated_at
FROM ingestion_scheduler_configs
WHERE id = 'default'
`)
	config, err := scanSchedulerConfig(row)
	if err == sql.ErrNoRows {
		return defaultSchedulerConfig(), nil
	}
	if err != nil {
		return domain.SchedulerConfig{}, fmt.Errorf("load scheduler config: %w", err)
	}
	return config, nil
}

func (r PostgresRepository) SaveSchedulerConfig(ctx context.Context, config domain.SchedulerConfig) (domain.SchedulerConfig, error) {
	config = normalizeSchedulerConfig(config)
	if err := config.Validate(); err != nil {
		return domain.SchedulerConfig{}, err
	}
	fixedTimes, err := json.Marshal(config.FixedTimes)
	if err != nil {
		return domain.SchedulerConfig{}, fmt.Errorf("marshal fixed times: %w", err)
	}
	sourceFilter, err := json.Marshal(schedulerSourceFilterMap(config.SourceFilter))
	if err != nil {
		return domain.SchedulerConfig{}, fmt.Errorf("marshal scheduler source filter: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
INSERT INTO ingestion_scheduler_configs (
    id, enabled, mode, interval_minutes, fixed_times, concurrency,
    batch_size, timeout_seconds, source_filter, timezone, config_version
) VALUES (
    $1, $2, $3, $4, $5::jsonb, $6,
    $7, $8, $9::jsonb, $10, 1
) ON CONFLICT (id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    mode = EXCLUDED.mode,
    interval_minutes = EXCLUDED.interval_minutes,
    fixed_times = EXCLUDED.fixed_times,
    concurrency = EXCLUDED.concurrency,
    batch_size = EXCLUDED.batch_size,
    timeout_seconds = EXCLUDED.timeout_seconds,
    source_filter = EXCLUDED.source_filter,
    timezone = EXCLUDED.timezone,
    config_version = ingestion_scheduler_configs.config_version + 1,
    updated_at = now()
RETURNING id, enabled, mode, interval_minutes, fixed_times, concurrency,
          batch_size, timeout_seconds, source_filter, timezone, config_version,
          last_run_id, last_run_at, created_at, updated_at
`, config.ID, config.Enabled, config.Mode, nullablePositiveInt(config.IntervalMinutes), string(fixedTimes), config.Concurrency,
		config.BatchSize, config.TimeoutSeconds, string(sourceFilter), config.Timezone)

	saved, err := scanSchedulerConfig(row)
	if err != nil {
		return domain.SchedulerConfig{}, fmt.Errorf("save scheduler config: %w", err)
	}
	return saved, nil
}

func scanSchedulerConfig(scanner rawDocumentScanner) (domain.SchedulerConfig, error) {
	var config domain.SchedulerConfig
	var interval sql.NullInt64
	var fixedTimesBytes []byte
	var sourceFilterBytes []byte
	var lastRunID sql.NullString
	var lastRunAt sql.NullTime
	if err := scanner.Scan(
		&config.ID,
		&config.Enabled,
		&config.Mode,
		&interval,
		&fixedTimesBytes,
		&config.Concurrency,
		&config.BatchSize,
		&config.TimeoutSeconds,
		&sourceFilterBytes,
		&config.Timezone,
		&config.ConfigVersion,
		&lastRunID,
		&lastRunAt,
		&config.CreatedAt,
		&config.UpdatedAt,
	); err != nil {
		return domain.SchedulerConfig{}, err
	}
	if interval.Valid {
		config.IntervalMinutes = int(interval.Int64)
	}
	if len(fixedTimesBytes) > 0 {
		if err := json.Unmarshal(fixedTimesBytes, &config.FixedTimes); err != nil {
			return domain.SchedulerConfig{}, fmt.Errorf("parse scheduler fixed times: %w", err)
		}
	}
	if len(sourceFilterBytes) > 0 {
		var sourceFilter map[string]string
		if err := json.Unmarshal(sourceFilterBytes, &sourceFilter); err != nil {
			return domain.SchedulerConfig{}, fmt.Errorf("parse scheduler source filter: %w", err)
		}
		config.SourceFilter = domain.SchedulerSourceFilter{
			ProviderKey:   sourceFilter["provider_key"],
			IngestChannel: sourceFilter["ingest_channel"],
			SourceType:    sourceFilter["source_type"],
		}
	}
	if lastRunID.Valid {
		config.LastRunID = lastRunID.String
	}
	if lastRunAt.Valid {
		config.LastRunAt = &lastRunAt.Time
	}
	return config, nil
}

func schedulerSourceFilterMap(filter domain.SchedulerSourceFilter) map[string]string {
	result := map[string]string{}
	if filter.ProviderKey != "" {
		result["provider_key"] = filter.ProviderKey
	}
	if filter.IngestChannel != "" {
		result["ingest_channel"] = filter.IngestChannel
	}
	if filter.SourceType != "" {
		result["source_type"] = filter.SourceType
	}
	return result
}
