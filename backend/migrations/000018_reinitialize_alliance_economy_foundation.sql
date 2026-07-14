-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF current_setting('tidewise.alliance_economy_schema_write_authorized', true) IS DISTINCT FROM 'reviewed_local_cleanup_verified' THEN
        RAISE EXCEPTION 'alliance profile schema change is not authorized; complete the scoped local cleanup Review and zero-row verification first';
    END IF;

    IF EXISTS (SELECT 1 FROM alliance_org_profiles) THEN
        RAISE EXCEPTION 'alliance_org_profiles must be empty before migration 000018';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP INDEX IF EXISTS idx_alliance_org_profiles_org_type;
DROP INDEX IF EXISTS idx_alliance_org_profiles_primary_domain;

ALTER TABLE alliance_org_profiles
    DROP COLUMN org_code,
    DROP COLUMN org_type,
    DROP COLUMN primary_domain,
    DROP COLUMN scope_region,
    DROP COLUMN official_url,
    ADD COLUMN abbreviation TEXT NOT NULL DEFAULT '',
    ADD COLUMN leadership_summary TEXT NOT NULL,
    ADD COLUMN influence_scope_summary TEXT NOT NULL,
    ADD CONSTRAINT chk_alliance_org_profiles_abbreviation CHECK (
        btrim(abbreviation) <> '—'
        AND char_length(btrim(abbreviation)) <= 32
    ),
    ADD CONSTRAINT chk_alliance_org_profiles_leadership CHECK (
        btrim(leadership_summary) <> ''
        AND char_length(btrim(leadership_summary)) <= 500
    ),
    ADD CONSTRAINT chk_alliance_org_profiles_influence_scope CHECK (
        btrim(influence_scope_summary) <> ''
        AND char_length(btrim(influence_scope_summary)) <= 1000
    );

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000018 is irreversible; this local exploration batch only supports a reviewed forward rebuild';
END;
$$;
-- +goose StatementEnd
