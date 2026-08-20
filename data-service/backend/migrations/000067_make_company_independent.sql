-- +goose Up
-- Coordinated stop-write migration. Take an approved snapshot before apply.
-- Existing Company UUID suffixes and all representable Company facts are preserved.
-- Controller, Storyline anchor, and Security issuer relations are intentionally retired.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM entity_nodes entity
        WHERE entity.entity_type = 'company'
          AND NOT EXISTS (
              SELECT 1 FROM company_profiles profile WHERE profile.entity_id = entity.id
          )
    ) THEN
        RAISE EXCEPTION 'A legacy Company Entity has no Company Profile and cannot be migrated';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM company_profiles profile
        JOIN entity_nodes entity ON entity.id = profile.entity_id
        WHERE entity.entity_type <> 'company'
           OR entity.entity_key !~ '^company:.{1,30}$'
           OR btrim(substring(entity.entity_key FROM 9)) = ''
           OR btrim(entity.name) = ''
           OR char_length(entity.name) > 200
           OR array_position(entity.aliases, NULL) IS NOT NULL
           OR EXISTS (
               SELECT 1 FROM unnest(entity.aliases) alias
               WHERE btrim(alias) = '' OR char_length(alias) > 200
           )
           OR cardinality(entity.aliases) <> (
               SELECT count(DISTINCT alias) FROM unnest(entity.aliases) alias
           )
           OR entity.updated_at < entity.created_at
    ) THEN
        RAISE EXCEPTION 'Legacy Company facts cannot be represented by the independent Company contract';
    END IF;

    IF EXISTS (
        SELECT substring(entity.entity_key FROM 9)
        FROM company_profiles profile
        JOIN entity_nodes entity ON entity.id = profile.entity_id
        GROUP BY substring(entity.entity_key FROM 9)
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'Legacy Company codes are not unique';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM company_profiles profile
        WHERE btrim(profile.industry_name) <> ''
          AND (SELECT count(*) FROM industry value WHERE value.name = btrim(profile.industry_name)) <> 1
    ) THEN
        RAISE EXCEPTION 'Every nonblank legacy Company industry label must match exactly one Industry name';
    END IF;

    IF EXISTS (
        WITH company_ids AS (SELECT entity_id id FROM company_profiles)
        SELECT 1 FROM entity_edges value JOIN company_ids target
          ON target.id IN (value.from_entity_id, value.to_entity_id)
        UNION ALL
        SELECT 1 FROM entity_external_identifiers value JOIN company_ids target
          ON target.id = value.entity_id
        UNION ALL
        SELECT 1 FROM entity_redirects value JOIN company_ids target
          ON target.id IN (value.source_entity_id, value.target_entity_id)
        UNION ALL
        SELECT 1 FROM index_profiles value JOIN company_ids target
          ON target.id = value.market_entity_id
        UNION ALL
        SELECT 1 FROM instrument_profiles value JOIN company_ids target
          ON target.id = value.underlying_entity_id
        UNION ALL
        SELECT 1 FROM person_profiles value JOIN company_ids target
          ON target.id = value.organization_entity_id
    ) THEN
        RAISE EXCEPTION 'A legacy Company is still referenced through an unsupported Entity relation';
    END IF;
END;
$$;
-- +goose StatementEnd

CREATE TEMP TABLE migration_000067_company_map ON COMMIT DROP AS
SELECT profile.entity_id AS old_id,
       'COM' || substring(profile.entity_id FROM 4) AS new_id,
       substring(entity.entity_key FROM 9) AS code,
       entity.name,
       entity.aliases,
       entity.status,
       entity.created_at,
       entity.updated_at,
       NULLIF(btrim(profile.area), '') AS operating_area,
       profile.registration_country_id,
       matched_industry.id AS industry_id
FROM company_profiles profile
JOIN entity_nodes entity ON entity.id = profile.entity_id
LEFT JOIN industry matched_industry ON matched_industry.name = btrim(profile.industry_name)
  AND btrim(profile.industry_name) <> '';

ALTER TABLE storylines DROP CONSTRAINT chk_storylines_anchor;
ALTER TABLE storylines DROP COLUMN company_entity_id;
ALTER TABLE storylines ADD CONSTRAINT chk_storylines_anchor CHECK (
    (storyline_type = 'GEOPOLITICAL' AND rivalry_id IS NOT NULL AND
        macro_economic_id IS NULL AND industry_chain_id IS NULL)
 OR (storyline_type = 'MACRO' AND rivalry_id IS NULL AND
        macro_economic_id IS NOT NULL AND industry_chain_id IS NULL)
 OR (storyline_type = 'INDUSTRY' AND rivalry_id IS NULL AND
        macro_economic_id IS NULL AND industry_chain_id IS NOT NULL)
 OR (storyline_type = 'CORPORATE' AND rivalry_id IS NULL AND
        macro_economic_id IS NULL AND industry_chain_id IS NULL)
);

ALTER TABLE security_profiles DROP COLUMN issuer_company_entity_id;

CREATE TYPE company_ownership_type AS ENUM (
    'STATE_CONTROLLED',
    'FAMILY_CONTROLLED',
    'FOUNDER_CONTROLLED',
    'INSTITUTION_CONTROLLED',
    'DISPERSED',
    'OTHER'
);

-- +goose StatementBegin
CREATE FUNCTION valid_company_aliases(values_to_check TEXT[])
RETURNS BOOLEAN
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT valid_independent_object_text_set(values_to_check, TRUE)
       AND NOT EXISTS (
           SELECT 1 FROM unnest(values_to_check) value WHERE char_length(value) > 200
       )
$$;
-- +goose StatementEnd

ALTER TABLE company_profiles RENAME TO company;
ALTER TABLE company RENAME COLUMN entity_id TO id;
ALTER TABLE company DROP CONSTRAINT company_profiles_entity_id_fkey;
ALTER TABLE company RENAME CONSTRAINT company_profiles_pkey TO company_pkey;
ALTER TABLE company RENAME COLUMN area TO operating_area;

ALTER TABLE company
    ADD COLUMN code VARCHAR(30),
    ADD COLUMN name VARCHAR(200),
    ADD COLUMN name_en VARCHAR(200),
    ADD COLUMN legal_name VARCHAR(300),
    ADD COLUMN aliases TEXT[],
    ADD COLUMN headquarters_city VARCHAR(100),
    ADD COLUMN founding_date DATE,
    ADD COLUMN ipo_date DATE,
    ADD COLUMN legal_form VARCHAR(64),
    ADD COLUMN ownership_type company_ownership_type,
    ADD COLUMN strategic_positioning TEXT,
    ADD COLUMN description TEXT,
    ADD COLUMN status VARCHAR(32),
    ADD COLUMN created_at TIMESTAMPTZ,
    ADD COLUMN updated_at TIMESTAMPTZ;

UPDATE company value
SET id = mapping.new_id,
    code = mapping.code,
    name = mapping.name,
    aliases = mapping.aliases,
    operating_area = mapping.operating_area,
    status = mapping.status,
    created_at = mapping.created_at,
    updated_at = mapping.updated_at
FROM migration_000067_company_map mapping
WHERE value.id = mapping.old_id;

ALTER TABLE company
    ALTER COLUMN code SET NOT NULL,
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN aliases SET NOT NULL,
    ALTER COLUMN aliases SET DEFAULT '{}'::TEXT[],
    ALTER COLUMN operating_area DROP NOT NULL,
    ALTER COLUMN operating_area DROP DEFAULT,
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN status SET DEFAULT 'active',
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN created_at SET DEFAULT now(),
    ALTER COLUMN updated_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT now(),
    DROP COLUMN industry_name,
    DROP COLUMN controller_name,
    DROP COLUMN controller_type,
    ADD CONSTRAINT uq_company_code UNIQUE (code),
    ADD CONSTRAINT chk_company_identity CHECK (
        id ~ '^COM[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    ADD CONSTRAINT chk_company_code_nonblank CHECK (btrim(code) <> ''),
    ADD CONSTRAINT chk_company_name_nonblank CHECK (btrim(name) <> ''),
    ADD CONSTRAINT chk_company_aliases CHECK (
        valid_company_aliases(aliases)
    ),
    ADD CONSTRAINT chk_company_optional_text CHECK (
        (name_en IS NULL OR btrim(name_en) <> '')
        AND (legal_name IS NULL OR btrim(legal_name) <> '')
        AND (operating_area IS NULL OR btrim(operating_area) <> '')
        AND (headquarters_city IS NULL OR btrim(headquarters_city) <> '')
        AND (legal_form IS NULL OR btrim(legal_form) <> '')
        AND (strategic_positioning IS NULL OR btrim(strategic_positioning) <> '')
        AND (description IS NULL OR btrim(description) <> '')
    ),
    ADD CONSTRAINT chk_company_date_order CHECK (
        founding_date IS NULL OR ipo_date IS NULL OR ipo_date >= founding_date
    ),
    ADD CONSTRAINT chk_company_status CHECK (status IN ('active', 'inactive', 'merged')),
    ADD CONSTRAINT chk_company_timestamp_order CHECK (updated_at >= created_at);

CREATE INDEX idx_company_list ON company (code, id);
CREATE INDEX idx_company_registration_country ON company (registration_country_id, code, id);

-- +goose StatementBegin
CREATE FUNCTION protect_company_code()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.code IS DISTINCT FROM OLD.code THEN
        RAISE EXCEPTION 'Company code is immutable';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_protect_company_code
BEFORE UPDATE OF code ON company
FOR EACH ROW EXECUTE FUNCTION protect_company_code();

-- +goose StatementBegin
CREATE FUNCTION migration_000067_company_industry_link_id(company_id TEXT, industry_id TEXT)
RETURNS TEXT
LANGUAGE plpgsql
IMMUTABLE
STRICT
AS $$
DECLARE
    value BYTEA;
    encoded TEXT;
BEGIN
    value := substring(
        public.digest(
            uuid_send('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid)
            || convert_to('CIL', 'UTF8') || '\x00'::bytea
            || convert_to('company-industry-link', 'UTF8') || '\x00'::bytea
            || convert_to(company_id, 'UTF8') || '\x00'::bytea
            || convert_to(industry_id, 'UTF8'),
            'sha1'
        )
        FROM 1 FOR 16
    );
    value := set_byte(value, 6, (get_byte(value, 6) & 15) | 80);
    value := set_byte(value, 8, (get_byte(value, 8) & 63) | 128);
    encoded := encode(value, 'hex');
    RETURN 'CIL'
        || substring(encoded FROM 1 FOR 8) || '-'
        || substring(encoded FROM 9 FOR 4) || '-'
        || substring(encoded FROM 13 FOR 4) || '-'
        || substring(encoded FROM 17 FOR 4) || '-'
        || substring(encoded FROM 21 FOR 12);
END;
$$;
-- +goose StatementEnd

CREATE TABLE company_industry_links (
    id VARCHAR(39) PRIMARY KEY,
    company_id VARCHAR(39) NOT NULL REFERENCES company(id) ON DELETE RESTRICT,
    industry_id VARCHAR(39) NOT NULL REFERENCES industry(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_company_industry_link UNIQUE (company_id, industry_id),
    CONSTRAINT chk_company_industry_link_identity CHECK (
        id ~ '^CIL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    )
);

INSERT INTO company_industry_links (id, company_id, industry_id)
SELECT migration_000067_company_industry_link_id(new_id, industry_id), new_id, industry_id
FROM migration_000067_company_map
WHERE industry_id IS NOT NULL;

DROP FUNCTION migration_000067_company_industry_link_id(TEXT, TEXT);

CREATE INDEX idx_company_industry_link_industry
    ON company_industry_links (industry_id, company_id);

DELETE FROM entity_nodes entity
USING migration_000067_company_map mapping
WHERE entity.id = mapping.old_id;

-- Keep the shared identity guard aware that Company is now independent.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION assert_data_object_identity_unique()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    lock_id TEXT;
    lock_ids TEXT[] := ARRAY[NEW.id]::TEXT[];
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.id IS DISTINCT FROM NEW.id THEN
        lock_ids := array_append(lock_ids, OLD.id);
    END IF;
    FOR lock_id IN SELECT DISTINCT value FROM unnest(lock_ids) value ORDER BY value LOOP
        PERFORM pg_advisory_xact_lock(hashtextextended(lock_id, 0));
    END LOOP;

    IF TG_TABLE_NAME = 'entity_nodes' THEN
        IF NEW.entity_type IN ('industry', 'concept', 'chain_node', 'industry_chain', 'company')
           AND (TG_OP = 'INSERT' OR OLD.entity_type IS DISTINCT FROM NEW.entity_type) THEN
            RAISE EXCEPTION 'new independent facts must use their object tables';
        END IF;
        IF EXISTS (SELECT 1 FROM industry WHERE id = NEW.id)
           OR EXISTS (SELECT 1 FROM concept WHERE id = NEW.id)
           OR EXISTS (SELECT 1 FROM chain_node WHERE id = NEW.id)
           OR EXISTS (SELECT 1 FROM industry_chain WHERE id = NEW.id)
           OR EXISTS (SELECT 1 FROM company WHERE id = NEW.id) THEN
            RAISE EXCEPTION 'Data object identity % already belongs to an independent object', NEW.id;
        END IF;
    ELSIF EXISTS (SELECT 1 FROM entity_nodes WHERE id = NEW.id)
       OR (TG_TABLE_NAME <> 'industry' AND EXISTS (SELECT 1 FROM industry WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'concept' AND EXISTS (SELECT 1 FROM concept WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'chain_node' AND EXISTS (SELECT 1 FROM chain_node WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'industry_chain' AND EXISTS (SELECT 1 FROM industry_chain WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'company' AND EXISTS (SELECT 1 FROM company WHERE id = NEW.id)) THEN
        RAISE EXCEPTION 'Data object identity % already belongs to another object', NEW.id;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION data_object_type(reference_id TEXT)
RETURNS TEXT
LANGUAGE SQL
STABLE
AS $$
    SELECT CASE WHEN count(*) = 1 THEN min(object_type) END
    FROM (
        SELECT entity_type::TEXT object_type FROM entity_nodes WHERE id = reference_id
        UNION ALL SELECT 'industry' FROM industry WHERE id = reference_id
        UNION ALL SELECT 'concept' FROM concept WHERE id = reference_id
        UNION ALL SELECT 'chain_node' FROM chain_node WHERE id = reference_id
        UNION ALL SELECT 'industry_chain' FROM industry_chain WHERE id = reference_id
    ) value
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_company_object_identity_unique
BEFORE INSERT OR UPDATE OF id ON company
FOR EACH ROW EXECUTE FUNCTION assert_data_object_identity_unique();

CREATE TRIGGER trg_company_protect_object_delete
BEFORE DELETE ON company
FOR EACH ROW EXECUTE FUNCTION protect_data_object_references();

CREATE TRIGGER trg_company_protect_object_id_update
BEFORE UPDATE OF id ON company
FOR EACH ROW EXECUTE FUNCTION protect_data_object_references();

CREATE TRIGGER trg_company_protect_object_truncate
BEFORE TRUNCATE ON company
FOR EACH STATEMENT EXECUTE FUNCTION protect_data_object_truncate();
