-- +goose Up
CREATE TABLE IF NOT EXISTS ingestion_scheduler_configs (
    id TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    mode TEXT NOT NULL,
    interval_minutes INTEGER,
    fixed_times JSONB NOT NULL DEFAULT '[]'::jsonb,
    concurrency INTEGER NOT NULL DEFAULT 1,
    batch_size INTEGER NOT NULL DEFAULT 10,
    timeout_seconds INTEGER NOT NULL DEFAULT 180,
    source_filter JSONB NOT NULL DEFAULT '{}'::jsonb,
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    config_version INTEGER NOT NULL DEFAULT 1,
    last_run_id UUID,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ingestion_scheduler_configs_mode_check
        CHECK (mode IN ('interval', 'fixed_times')),
    CONSTRAINT ingestion_scheduler_configs_interval_check
        CHECK (interval_minutes IS NULL OR interval_minutes > 0),
    CONSTRAINT ingestion_scheduler_configs_concurrency_check
        CHECK (concurrency > 0),
    CONSTRAINT ingestion_scheduler_configs_batch_size_check
        CHECK (batch_size > 0),
    CONSTRAINT ingestion_scheduler_configs_timeout_seconds_check
        CHECK (timeout_seconds > 0)
);

INSERT INTO ingestion_scheduler_configs (
    id, enabled, mode, interval_minutes, fixed_times, concurrency,
    batch_size, timeout_seconds, source_filter, timezone, config_version
) VALUES (
    'default', FALSE, 'interval', 60, '[]'::jsonb, 1,
    10, 180, '{}'::jsonb, 'Asia/Shanghai', 1
) ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS ingestion_runs (
    id UUID PRIMARY KEY,
    trigger_type TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    total_sources INTEGER NOT NULL DEFAULT 0,
    succeeded_sources INTEGER NOT NULL DEFAULT 0,
    failed_sources INTEGER NOT NULL DEFAULT 0,
    skipped_sources INTEGER NOT NULL DEFAULT 0,
    scheduler_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_summary TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ingestion_runs_status_check
        CHECK (status IN ('running', 'succeeded', 'failed', 'partial', 'skipped')),
    CONSTRAINT ingestion_runs_trigger_type_check
        CHECK (trigger_type IN ('manual_once', 'interval', 'fixed_time'))
);

CREATE TABLE IF NOT EXISTS ingestion_run_sources (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES ingestion_runs(id),
    source_id UUID NOT NULL REFERENCES source_catalogs(id),
    status TEXT NOT NULL,
    documents_written INTEGER NOT NULL DEFAULT 0,
    documents_duplicate INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    duration_millis INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ingestion_run_sources_status_check
        CHECK (status IN ('succeeded', 'failed', 'skipped')),
    CONSTRAINT ingestion_run_sources_documents_written_check
        CHECK (documents_written >= 0),
    CONSTRAINT ingestion_run_sources_documents_duplicate_check
        CHECK (documents_duplicate >= 0),
    CONSTRAINT ingestion_run_sources_duration_millis_check
        CHECK (duration_millis >= 0)
);

CREATE INDEX IF NOT EXISTS idx_ingestion_runs_started_at
    ON ingestion_runs (started_at DESC);

CREATE INDEX IF NOT EXISTS idx_ingestion_runs_status
    ON ingestion_runs (status);

CREATE INDEX IF NOT EXISTS idx_ingestion_run_sources_run_id
    ON ingestion_run_sources (run_id);

CREATE INDEX IF NOT EXISTS idx_ingestion_run_sources_source_id
    ON ingestion_run_sources (source_id);

-- +goose Down
SELECT 'ingestion scheduler rollback requires a reviewed forward migration or restored backup';
