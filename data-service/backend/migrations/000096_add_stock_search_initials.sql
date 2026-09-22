-- +goose Up
ALTER TABLE stock ADD COLUMN name_initials TEXT;
-- Initials are derived by the explicit stock-initialize -reindex operation.
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'stock migrations are forward-only'; END $$;
-- +goose StatementEnd
