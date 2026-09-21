-- +goose Up
ALTER TABLE user_configurations RENAME TO configuration;
ALTER TABLE configuration RENAME CONSTRAINT user_configurations_pkey TO configuration_pkey;
ALTER TABLE configuration RENAME CONSTRAINT user_configurations_code_key TO configuration_code_key;
ALTER TABLE configuration RENAME CONSTRAINT user_configurations_code_check TO configuration_code_check;
ALTER TABLE configuration RENAME CONSTRAINT user_configurations_value_check TO configuration_value_check;

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'User migrations are forward-only'; END $$;
-- +goose StatementEnd
