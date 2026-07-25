-- +goose Up
ALTER TABLE entity_nodes
    ADD COLUMN IF NOT EXISTS entity_key TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_entity_nodes_entity_key
    ON entity_nodes (entity_key);

CREATE TABLE IF NOT EXISTS graph_projection_runs (
    id UUID PRIMARY KEY,
    projection_type TEXT NOT NULL,
    mode TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    source_row_count INTEGER NOT NULL DEFAULT 0,
    projected_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    error_summary TEXT NOT NULL DEFAULT '',
    config_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT graph_projection_runs_projection_type_check
        CHECK (projection_type IN ('entity_graph')),
    CONSTRAINT graph_projection_runs_mode_check
        CHECK (mode IN ('project_entities', 'rebuild_entities')),
    CONSTRAINT graph_projection_runs_status_check
        CHECK (status IN ('running', 'succeeded', 'failed', 'partial')),
    CONSTRAINT graph_projection_runs_source_row_count_check
        CHECK (source_row_count >= 0),
    CONSTRAINT graph_projection_runs_projected_count_check
        CHECK (projected_count >= 0),
    CONSTRAINT graph_projection_runs_skipped_count_check
        CHECK (skipped_count >= 0),
    CONSTRAINT graph_projection_runs_failed_count_check
        CHECK (failed_count >= 0)
);

CREATE TABLE IF NOT EXISTS graph_projection_run_items (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES graph_projection_runs(id),
    item_type TEXT NOT NULL,
    item_key TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT graph_projection_run_items_item_type_check
        CHECK (item_type IN ('entity_node', 'entity_relationship')),
    CONSTRAINT graph_projection_run_items_status_check
        CHECK (status IN ('projected', 'skipped', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_graph_projection_runs_started_at
    ON graph_projection_runs (started_at DESC);

CREATE INDEX IF NOT EXISTS idx_graph_projection_runs_type_status
    ON graph_projection_runs (projection_type, status);

CREATE INDEX IF NOT EXISTS idx_graph_projection_run_items_run_id
    ON graph_projection_run_items (run_id);

-- +goose Down
SELECT 'graph projection rollback requires a reviewed forward migration or restored backup';
