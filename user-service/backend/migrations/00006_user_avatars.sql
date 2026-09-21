-- +goose Up
CREATE TABLE user_avatars (
 user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 image_data BYTEA NOT NULL CHECK(octet_length(image_data) BETWEEN 1 AND 131072),
 content_type TEXT NOT NULL CHECK(content_type = 'image/jpeg'),
 updated_at TIMESTAMPTZ NOT NULL,
 privacy_version TEXT NOT NULL
);
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'User migrations are forward-only'; END $$;
-- +goose StatementEnd
