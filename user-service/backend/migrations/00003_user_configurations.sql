-- +goose Up
CREATE TABLE user_configurations (
    id UUID PRIMARY KEY,
    code TEXT COLLATE "C" NOT NULL UNIQUE CHECK (code <> '' AND code = btrim(code)),
    value JSONB NOT NULL CHECK (jsonb_typeof(value) = 'object'),
    updated_at TIMESTAMPTZ NOT NULL
);
COMMENT ON TABLE user_configurations IS 'Private service configuration; may contain secrets. No public dictionary API.';

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'User migrations are forward-only'; END $$;
-- +goose StatementEnd
