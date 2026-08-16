-- +goose Up
-- This forward-only contract change intentionally does not backfill legacy rows.
-- Environments with published facts must be recreated or restored from an
-- explicitly reviewed snapshot before applying the new application contract.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM organization_categories)
        OR EXISTS (SELECT 1 FROM organization_domain_tags)
        OR EXISTS (SELECT 1 FROM organization_domain_tag_links)
        OR EXISTS (SELECT 1 FROM raw_evidence_category_links)
    THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migration 000053 requires empty Organization category/tag and Raw Evidence category link tables; no legacy identity backfill is supported';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE organizations
    DROP CONSTRAINT organizations_category_code_fkey;

ALTER TABLE organization_categories
    ADD COLUMN id VARCHAR(39) NOT NULL,
    DROP CONSTRAINT organization_categories_pkey,
    ADD CONSTRAINT organization_categories_pkey PRIMARY KEY (id),
    ADD CONSTRAINT uq_organization_categories_code UNIQUE (code),
    ADD CONSTRAINT chk_organization_categories_identity CHECK (
        id ~ '^OCA[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    );

ALTER TABLE organizations
    ADD CONSTRAINT organizations_category_code_fkey
        FOREIGN KEY (category_code) REFERENCES organization_categories(code) ON DELETE RESTRICT;

ALTER TABLE organization_domain_tags
    ADD COLUMN id VARCHAR(39) NOT NULL,
    DROP CONSTRAINT organization_domain_tags_pkey,
    ADD CONSTRAINT organization_domain_tags_pkey PRIMARY KEY (id),
    ADD CONSTRAINT uq_organization_domain_tags_code UNIQUE (code),
    ADD CONSTRAINT chk_organization_domain_tags_identity CHECK (
        id ~ '^ODT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    );

ALTER TABLE organization_domain_tag_links
    ADD COLUMN id VARCHAR(39) NOT NULL,
    DROP CONSTRAINT organization_domain_tag_links_pkey,
    ADD CONSTRAINT organization_domain_tag_links_pkey PRIMARY KEY (id),
    ADD CONSTRAINT uq_organization_domain_tag_links_endpoints UNIQUE (organization_id, domain_tag_code),
    ADD CONSTRAINT chk_organization_domain_tag_links_identity CHECK (
        id ~ '^ODL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    );

ALTER TABLE raw_evidences RENAME COLUMN raw_evidence_id TO id;
ALTER TABLE evidences RENAME COLUMN evidence_id TO id;

ALTER TABLE raw_evidence_category_links
    ADD COLUMN id VARCHAR(39) NOT NULL,
    DROP CONSTRAINT raw_evidence_category_links_pkey,
    ADD CONSTRAINT raw_evidence_category_links_pkey PRIMARY KEY (id),
    ADD CONSTRAINT uq_raw_evidence_category_links_endpoints UNIQUE (raw_evidence_id, category_id),
    ADD CONSTRAINT chk_raw_evidence_category_links_identity CHECK (
        id ~ '^RCL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    );

COMMENT ON COLUMN organization_categories.id IS 'Data Service生成的Organization Category正式ID。';
COMMENT ON COLUMN organization_domain_tags.id IS 'Data Service生成的Organization Domain Tag正式ID。';
COMMENT ON COLUMN organization_domain_tag_links.id IS 'Data Service生成的Organization Domain Tag Link正式ID。';
COMMENT ON COLUMN raw_evidences.id IS 'Data Service生成的Raw Evidence正式ID。';
COMMENT ON COLUMN evidences.id IS 'Data Service生成的Atomic Evidence正式ID。';
COMMENT ON COLUMN raw_evidence_category_links.id IS 'Data Service生成的Raw Evidence Category Link正式ID。';

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000053 is forward-only; restore a reviewed snapshot with the previous application';
END;
$$;
-- +goose StatementEnd
