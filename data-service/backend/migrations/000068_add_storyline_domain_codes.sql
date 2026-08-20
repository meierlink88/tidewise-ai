-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM storyline_domains) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'storyline_domains must be empty before adding required catalog codes';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE storyline_domains
    ADD COLUMN code VARCHAR(30) NOT NULL,
    ADD CONSTRAINT uq_storyline_domains_code UNIQUE (code),
    ADD CONSTRAINT chk_storyline_domains_code CHECK (
        code ~ '^[A-Z][A-Z0-9_]{0,29}$'
    );

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000068 is forward-only; retain StorylineDomain codes and published catalog facts during application rollback';
END;
$$;
-- +goose StatementEnd
