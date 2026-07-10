-- +goose Up
ALTER TABLE entity_edges
    ADD COLUMN IF NOT EXISTS source_name TEXT NOT NULL DEFAULT '';

ALTER TABLE entity_edges
    ADD COLUMN IF NOT EXISTS source_url TEXT NOT NULL DEFAULT '';

ALTER TABLE entity_edges
    ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ;

-- +goose Down
-- Entity edge provenance rollback requires a reviewed forward migration or restored backup.
