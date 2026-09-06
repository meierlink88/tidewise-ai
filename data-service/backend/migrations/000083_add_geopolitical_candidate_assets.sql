-- +goose Up
-- Issue #415 adds a required, reviewed candidate-asset universe to each
-- geopolitical storyline. Catalog facts remain outside schema migrations.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM geopolitic_rivalries LIMIT 1) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migration 000083 requires an empty geopolitic_rivalries table; preserve the recovery point and clear the replaceable v1 catalog before retrying';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION validate_geopolitic_candidate_assets(input JSONB)
RETURNS BOOLEAN
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    asset JSONB;
    asset_name TEXT;
    seen_assets TEXT[] := ARRAY[]::TEXT[];
BEGIN
    IF input IS NULL OR jsonb_typeof(input) <> 'array' OR jsonb_array_length(input) = 0 THEN
        RETURN FALSE;
    END IF;
    FOR asset IN SELECT value FROM jsonb_array_elements(input)
    LOOP
        IF jsonb_typeof(asset) <> 'string' THEN
            RETURN FALSE;
        END IF;
        asset_name := asset #>> '{}';
        IF asset_name = '' OR asset_name <> btrim(asset_name) OR
           char_length(asset_name) > 100 OR asset_name = ANY(seen_assets) THEN
            RETURN FALSE;
        END IF;
        seen_assets := array_append(seen_assets, asset_name);
    END LOOP;
    RETURN TRUE;
END;
$$;
-- +goose StatementEnd

ALTER TABLE geopolitic_rivalries
    ADD COLUMN candidate_assets JSONB NOT NULL,
    ADD CONSTRAINT chk_geopolitic_rivalries_candidate_assets CHECK (
        validate_geopolitic_candidate_assets(candidate_assets)
    );

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000083 is a forward-only geopolitical candidate-asset cutover; restore the reviewed pre-cutover snapshot with matching application releases';
END;
$$;
-- +goose StatementEnd
