-- +goose Up
-- Issue #417: coordinated, empty-table-only replacement. No automatic fact deletion.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM macro_economics LIMIT 1) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'migration 000084 requires an empty macro_economics table; preserve a recovery point and review existing facts before retrying';
    END IF;
END;
$$;
-- +goose StatementEnd
DROP TABLE macro_economics;
DROP TYPE macro_economic_type;
DROP TYPE macro_economic_status;

-- +goose StatementBegin
CREATE FUNCTION validate_macroeconomic_candidate_assets(input JSONB)
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


-- +goose StatementBegin
CREATE FUNCTION validate_macroeconomic_domain_tactics(input JSONB)
RETURNS BOOLEAN
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    tactic JSONB;
    seen_names TEXT[] := ARRAY[]::TEXT[];
BEGIN
    IF input IS NULL OR jsonb_typeof(input) <> 'array' OR jsonb_array_length(input) = 0 THEN
        RETURN FALSE;
    END IF;
    FOR tactic IN SELECT value FROM jsonb_array_elements(input)
    LOOP
        IF jsonb_typeof(tactic) <> 'object' OR
           tactic <> jsonb_build_object(
               'name', tactic -> 'name',
               'description', tactic -> 'description'
           ) OR
           jsonb_typeof(tactic -> 'name') <> 'string' OR
           jsonb_typeof(tactic -> 'description') <> 'string' OR
           btrim(tactic ->> 'name') = '' OR
           char_length(tactic ->> 'name') > 50 OR
           btrim(tactic ->> 'description') = '' OR
           tactic ->> 'name' = ANY(seen_names) THEN
            RETURN FALSE;
        END IF;
        seen_names := array_append(seen_names, tactic ->> 'name');
    END LOOP;
    RETURN TRUE;
END;
$$;
-- +goose StatementEnd

CREATE TABLE macro_economics_domain (
    id VARCHAR(39) PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    tactics JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_macro_economics_domain_identity CHECK (
        id ~ '^MCD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_macro_economics_domain_code CHECK (
        code ~ '^[A-Z][A-Z0-9_]{0,49}$'
    ),
    CONSTRAINT chk_macro_economics_domain_required_text CHECK (
        btrim(name) <> '' AND btrim(description) <> ''
    ),
    CONSTRAINT chk_macro_economics_domain_tactics CHECK (
        validate_macroeconomic_domain_tactics(tactics)
    ),
    CONSTRAINT chk_macro_economics_domain_timestamp_order CHECK (
        updated_at >= created_at
    )
);

CREATE INDEX idx_macro_economics_domain_list
    ON macro_economics_domain (code, id);

-- +goose StatementBegin
CREATE FUNCTION prevent_macroeconomic_domain_code_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.code IS DISTINCT FROM OLD.code THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'MacroEconomicDomain code is immutable';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_macro_economics_domain_code_immutable
BEFORE UPDATE OF code ON macro_economics_domain
FOR EACH ROW EXECUTE FUNCTION prevent_macroeconomic_domain_code_mutation();

CREATE TABLE macro_economics (
    id VARCHAR(39) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    macro_economics_domain_id VARCHAR(39) NOT NULL
        REFERENCES macro_economics_domain(id) ON DELETE RESTRICT,
    core_proposition TEXT NOT NULL,
    candidate_assets JSONB NOT NULL CHECK (validate_macroeconomic_candidate_assets(candidate_assets)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_macro_economics_identity CHECK (
        id ~ '^MEC[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_macro_economics_required_text CHECK (
        btrim(name) <> '' AND btrim(core_proposition) <> ''
    ),
    CONSTRAINT chk_macro_economics_timestamp_order CHECK (
        updated_at >= created_at
    )
);

CREATE INDEX idx_macro_economics_list
    ON macro_economics (name, id);
CREATE INDEX idx_macro_economics_by_domain
    ON macro_economics (macro_economics_domain_id, name, id);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000084 is a forward-only macroeconomic storyline cutover; restore the reviewed pre-cutover snapshot with matching application releases';
END;
$$;
-- +goose StatementEnd
