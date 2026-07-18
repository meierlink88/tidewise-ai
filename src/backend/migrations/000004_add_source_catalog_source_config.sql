-- +goose Up
ALTER TABLE source_catalogs
    ADD COLUMN IF NOT EXISTS source_config JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
SELECT 'source catalog source_config rollback requires a reviewed forward migration or restored backup';
