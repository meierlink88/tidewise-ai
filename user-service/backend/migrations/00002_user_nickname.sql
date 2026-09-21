-- +goose Up
ALTER TABLE users ADD COLUMN nickname TEXT NOT NULL DEFAULT ''
    CHECK (char_length(nickname) <= 32 AND nickname = btrim(nickname));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'User migrations are forward-only'; END $$;
-- +goose StatementEnd
