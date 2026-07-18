-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    invalid_stages TEXT;
BEGIN
    SELECT string_agg(DISTINCT transmission_stage, ', ' ORDER BY transmission_stage)
    INTO invalid_stages
    FROM research_themes
    WHERE transmission_stage NOT IN ('identification', 'validation', 'diffusion', 'dampening');

    IF invalid_stages IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'research_themes contains transmission_stage values that require reviewed migration: ' || invalid_stages;
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE research_themes
    DROP CONSTRAINT chk_research_themes_stage;

ALTER TABLE research_themes
    ADD CONSTRAINT chk_research_themes_stage
    CHECK (transmission_stage IN ('identification', 'validation', 'diffusion', 'dampening'));

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000023 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
