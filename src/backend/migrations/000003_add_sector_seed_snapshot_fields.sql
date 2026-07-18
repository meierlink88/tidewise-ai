-- +goose Up
ALTER TABLE sector_profiles
    ADD COLUMN IF NOT EXISTS rank_snapshot INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS snapshot_date DATE;

-- +goose Down
SELECT 'sector seed snapshot fields rollback requires a reviewed forward migration or restored backup';
