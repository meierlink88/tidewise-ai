-- +goose Up
ALTER TABLE users ADD COLUMN phone_number TEXT,
 ADD COLUMN phone_verified_at TIMESTAMPTZ,
 ADD CONSTRAINT users_phone_format CHECK (phone_number IS NULL OR phone_number ~ '^\+[0-9]{7,15}$'),
 ADD CONSTRAINT users_phone_verified CHECK ((phone_number IS NULL) = (phone_verified_at IS NULL));
ALTER TABLE user_sessions ADD COLUMN privacy_version TEXT,
 ADD COLUMN privacy_accepted_at TIMESTAMPTZ,
 ADD CONSTRAINT sessions_privacy_consent CHECK ((privacy_version IS NULL) = (privacy_accepted_at IS NULL));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'User migrations are forward-only'; END $$;
-- +goose StatementEnd
