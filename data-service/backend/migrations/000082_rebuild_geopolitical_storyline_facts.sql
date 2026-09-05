-- +goose Up
-- Issue #413 authorizes this zero-compatibility geopolitical storyline cutover.
-- Stop legacy Storyline and GeopoliticRivalry writers and preserve a reviewed
-- PostgreSQL recovery point before applying it.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM storyline_event_links LIMIT 1)
       OR EXISTS (SELECT 1 FROM storylines LIMIT 1)
       OR EXISTS (SELECT 1 FROM storyline_domain_tactics LIMIT 1)
       OR EXISTS (SELECT 1 FROM storyline_domains LIMIT 1)
       OR EXISTS (SELECT 1 FROM geopolitic_rivalries LIMIT 1) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migration 000082 requires empty legacy Storyline and GeopoliticRivalry tables; preserve the recovery point and clear the coordinated dataset before retrying';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP TABLE storyline_event_links;
DROP TABLE storylines;
DROP TABLE storyline_domain_tactics;
DROP TABLE storyline_domains;

DROP TYPE storyline_data_alignment_status;
DROP TYPE storyline_status;
DROP TYPE storyline_type;
DROP TYPE storyline_domain_category;

DROP TABLE geopolitic_rivalries;
DROP TYPE geopolitic_rivalry_status;
DROP TYPE geopolitic_rivalry_type;

-- +goose StatementBegin
CREATE FUNCTION validate_geopolitic_domain_tactics(input JSONB)
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

CREATE TABLE geopolitic_domains (
    id VARCHAR(39) PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    tactics JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_geopolitic_domains_identity CHECK (
        id ~ '^GPD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_geopolitic_domains_code CHECK (
        code ~ '^[A-Z][A-Z0-9_]{0,49}$'
    ),
    CONSTRAINT chk_geopolitic_domains_required_text CHECK (
        btrim(name) <> '' AND btrim(description) <> ''
    ),
    CONSTRAINT chk_geopolitic_domains_tactics CHECK (
        validate_geopolitic_domain_tactics(tactics)
    ),
    CONSTRAINT chk_geopolitic_domains_timestamp_order CHECK (
        updated_at >= created_at
    )
);

CREATE INDEX idx_geopolitic_domains_list
    ON geopolitic_domains (code, id);

-- +goose StatementBegin
CREATE FUNCTION prevent_geopolitic_domain_code_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.code IS DISTINCT FROM OLD.code THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'GeopoliticDomain code is immutable';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_geopolitic_domains_code_immutable
BEFORE UPDATE OF code ON geopolitic_domains
FOR EACH ROW EXECUTE FUNCTION prevent_geopolitic_domain_code_mutation();

CREATE TABLE geopolitic_rivalries (
    id VARCHAR(39) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    category VARCHAR(100) NOT NULL,
    geopolitic_domain_id VARCHAR(39) NOT NULL
        REFERENCES geopolitic_domains(id) ON DELETE RESTRICT,
    core_proposition TEXT NOT NULL,
    core_actors TEXT NOT NULL,
    main_transmission TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_geopolitic_rivalries_identity CHECK (
        id ~ '^GPR[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_geopolitic_rivalries_required_text CHECK (
        btrim(name) <> '' AND btrim(category) <> '' AND
        btrim(core_proposition) <> '' AND btrim(core_actors) <> '' AND
        btrim(main_transmission) <> ''
    ),
    CONSTRAINT chk_geopolitic_rivalries_timestamp_order CHECK (
        updated_at >= created_at
    )
);

CREATE INDEX idx_geopolitic_rivalries_list
    ON geopolitic_rivalries (name, id);
CREATE INDEX idx_geopolitic_rivalries_domain_list
    ON geopolitic_rivalries (geopolitic_domain_id, name, id);
CREATE INDEX idx_geopolitic_rivalries_category_list
    ON geopolitic_rivalries (category, name, id);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000082 is a forward-only geopolitical storyline cutover; restore the reviewed pre-cutover snapshot with matching application releases';
END;
$$;
-- +goose StatementEnd
