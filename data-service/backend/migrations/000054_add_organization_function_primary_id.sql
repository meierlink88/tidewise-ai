-- +goose Up
-- This forward-only contract change intentionally does not backfill legacy rows.
-- Environments must have no Organization facts or tag links and must clear the
-- reproducible Function/Domain Tag catalog in a reviewed stop-write window.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM organizations)
        OR EXISTS (SELECT 1 FROM organization_domain_tag_links)
        OR EXISTS (SELECT 1 FROM organization_domain_tags)
        OR EXISTS (SELECT 1 FROM organization_functions)
    THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migration 000054 requires empty Organization facts, tag links, Domain Tags, and Functions; no legacy identity backfill is supported';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE organizations
    DROP CONSTRAINT organizations_function_code_fkey;

ALTER TABLE organization_domain_tags
    DROP CONSTRAINT organization_domain_tags_function_code_fkey;

ALTER TABLE organization_functions
    ADD COLUMN id VARCHAR(39) NOT NULL,
    DROP CONSTRAINT organization_functions_pkey,
    ADD CONSTRAINT organization_functions_pkey PRIMARY KEY (id),
    ADD CONSTRAINT uq_organization_functions_code UNIQUE (code),
    ADD CONSTRAINT chk_organization_functions_identity CHECK (
        id ~ '^OFN[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    );

ALTER TABLE organizations
    ADD CONSTRAINT organizations_function_code_fkey
        FOREIGN KEY (function_code) REFERENCES organization_functions(code) ON DELETE RESTRICT;

ALTER TABLE organization_domain_tags
    ADD CONSTRAINT organization_domain_tags_function_code_fkey
        FOREIGN KEY (function_code) REFERENCES organization_functions(code) ON DELETE RESTRICT;

COMMENT ON COLUMN organization_functions.id IS 'Data Service生成的Organization Function正式ID。';

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000054 is forward-only; restore a reviewed snapshot with the previous application';
END;
$$;
-- +goose StatementEnd
