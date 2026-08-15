-- +goose Up
-- Coordinated stop-write migration. Take an approved snapshot before apply.
-- Existing Entity and Entity Relation UUID suffixes are preserved. Code-owned
-- catalogs and Evidence identities are deterministically remapped so every
-- domain object uses PREFIX immediately followed by a lowercase UUID.

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE TEMP TABLE migration_000050_country_codes (
    alpha3 CHAR(3) PRIMARY KEY,
    alpha2 CHAR(2) NOT NULL UNIQUE
) ON COMMIT DROP;

INSERT INTO migration_000050_country_codes (alpha3, alpha2) VALUES
    ('ABW', 'AW'),
    ('AFG', 'AF'),
    ('AGO', 'AO'),
    ('AIA', 'AI'),
    ('ALA', 'AX'),
    ('ALB', 'AL'),
    ('AND', 'AD'),
    ('ARE', 'AE'),
    ('ARG', 'AR'),
    ('ARM', 'AM'),
    ('ASM', 'AS'),
    ('ATA', 'AQ'),
    ('ATF', 'TF'),
    ('ATG', 'AG'),
    ('AUS', 'AU'),
    ('AUT', 'AT'),
    ('AZE', 'AZ'),
    ('BDI', 'BI'),
    ('BEL', 'BE'),
    ('BEN', 'BJ'),
    ('BES', 'BQ'),
    ('BFA', 'BF'),
    ('BGD', 'BD'),
    ('BGR', 'BG'),
    ('BHR', 'BH'),
    ('BHS', 'BS'),
    ('BIH', 'BA'),
    ('BLM', 'BL'),
    ('BLR', 'BY'),
    ('BLZ', 'BZ'),
    ('BMU', 'BM'),
    ('BOL', 'BO'),
    ('BRA', 'BR'),
    ('BRB', 'BB'),
    ('BRN', 'BN'),
    ('BTN', 'BT'),
    ('BVT', 'BV'),
    ('BWA', 'BW'),
    ('CAF', 'CF'),
    ('CAN', 'CA'),
    ('CCK', 'CC'),
    ('CHE', 'CH'),
    ('CHL', 'CL'),
    ('CHN', 'CN'),
    ('CIV', 'CI'),
    ('CMR', 'CM'),
    ('COD', 'CD'),
    ('COG', 'CG'),
    ('COK', 'CK'),
    ('COL', 'CO'),
    ('COM', 'KM'),
    ('CPV', 'CV'),
    ('CRI', 'CR'),
    ('CUB', 'CU'),
    ('CUW', 'CW'),
    ('CXR', 'CX'),
    ('CYM', 'KY'),
    ('CYP', 'CY'),
    ('CZE', 'CZ'),
    ('DEU', 'DE'),
    ('DJI', 'DJ'),
    ('DMA', 'DM'),
    ('DNK', 'DK'),
    ('DOM', 'DO'),
    ('DZA', 'DZ'),
    ('ECU', 'EC'),
    ('EGY', 'EG'),
    ('ERI', 'ER'),
    ('ESH', 'EH'),
    ('ESP', 'ES'),
    ('EST', 'EE'),
    ('ETH', 'ET'),
    ('FIN', 'FI'),
    ('FJI', 'FJ'),
    ('FLK', 'FK'),
    ('FRA', 'FR'),
    ('FRO', 'FO'),
    ('FSM', 'FM'),
    ('GAB', 'GA'),
    ('GBR', 'GB'),
    ('GEO', 'GE'),
    ('GGY', 'GG'),
    ('GHA', 'GH'),
    ('GIB', 'GI'),
    ('GIN', 'GN'),
    ('GLP', 'GP'),
    ('GMB', 'GM'),
    ('GNB', 'GW'),
    ('GNQ', 'GQ'),
    ('GRC', 'GR'),
    ('GRD', 'GD'),
    ('GRL', 'GL'),
    ('GTM', 'GT'),
    ('GUF', 'GF'),
    ('GUM', 'GU'),
    ('GUY', 'GY'),
    ('HKG', 'HK'),
    ('HMD', 'HM'),
    ('HND', 'HN'),
    ('HRV', 'HR'),
    ('HTI', 'HT'),
    ('HUN', 'HU'),
    ('IDN', 'ID'),
    ('IMN', 'IM'),
    ('IND', 'IN'),
    ('IOT', 'IO'),
    ('IRL', 'IE'),
    ('IRN', 'IR'),
    ('IRQ', 'IQ'),
    ('ISL', 'IS'),
    ('ISR', 'IL'),
    ('ITA', 'IT'),
    ('JAM', 'JM'),
    ('JEY', 'JE'),
    ('JOR', 'JO'),
    ('JPN', 'JP'),
    ('KAZ', 'KZ'),
    ('KEN', 'KE'),
    ('KGZ', 'KG'),
    ('KHM', 'KH'),
    ('KIR', 'KI'),
    ('KNA', 'KN'),
    ('KOR', 'KR'),
    ('KWT', 'KW'),
    ('LAO', 'LA'),
    ('LBN', 'LB'),
    ('LBR', 'LR'),
    ('LBY', 'LY'),
    ('LCA', 'LC'),
    ('LIE', 'LI'),
    ('LKA', 'LK'),
    ('LSO', 'LS'),
    ('LTU', 'LT'),
    ('LUX', 'LU'),
    ('LVA', 'LV'),
    ('MAC', 'MO'),
    ('MAF', 'MF'),
    ('MAR', 'MA'),
    ('MCO', 'MC'),
    ('MDA', 'MD'),
    ('MDG', 'MG'),
    ('MDV', 'MV'),
    ('MEX', 'MX'),
    ('MHL', 'MH'),
    ('MKD', 'MK'),
    ('MLI', 'ML'),
    ('MLT', 'MT'),
    ('MMR', 'MM'),
    ('MNE', 'ME'),
    ('MNG', 'MN'),
    ('MNP', 'MP'),
    ('MOZ', 'MZ'),
    ('MRT', 'MR'),
    ('MSR', 'MS'),
    ('MTQ', 'MQ'),
    ('MUS', 'MU'),
    ('MWI', 'MW'),
    ('MYS', 'MY'),
    ('MYT', 'YT'),
    ('NAM', 'NA'),
    ('NCL', 'NC'),
    ('NER', 'NE'),
    ('NFK', 'NF'),
    ('NGA', 'NG'),
    ('NIC', 'NI'),
    ('NIU', 'NU'),
    ('NLD', 'NL'),
    ('NOR', 'NO'),
    ('NPL', 'NP'),
    ('NRU', 'NR'),
    ('NZL', 'NZ'),
    ('OMN', 'OM'),
    ('PAK', 'PK'),
    ('PAN', 'PA'),
    ('PCN', 'PN'),
    ('PER', 'PE'),
    ('PHL', 'PH'),
    ('PLW', 'PW'),
    ('PNG', 'PG'),
    ('POL', 'PL'),
    ('PRI', 'PR'),
    ('PRK', 'KP'),
    ('PRT', 'PT'),
    ('PRY', 'PY'),
    ('PSE', 'PS'),
    ('PYF', 'PF'),
    ('QAT', 'QA'),
    ('REU', 'RE'),
    ('ROU', 'RO'),
    ('RUS', 'RU'),
    ('RWA', 'RW'),
    ('SAU', 'SA'),
    ('SDN', 'SD'),
    ('SEN', 'SN'),
    ('SGP', 'SG'),
    ('SGS', 'GS'),
    ('SHN', 'SH'),
    ('SJM', 'SJ'),
    ('SLB', 'SB'),
    ('SLE', 'SL'),
    ('SLV', 'SV'),
    ('SMR', 'SM'),
    ('SOM', 'SO'),
    ('SPM', 'PM'),
    ('SRB', 'RS'),
    ('SSD', 'SS'),
    ('STP', 'ST'),
    ('SUR', 'SR'),
    ('SVK', 'SK'),
    ('SVN', 'SI'),
    ('SWE', 'SE'),
    ('SWZ', 'SZ'),
    ('SXM', 'SX'),
    ('SYC', 'SC'),
    ('SYR', 'SY'),
    ('TCA', 'TC'),
    ('TCD', 'TD'),
    ('TGO', 'TG'),
    ('THA', 'TH'),
    ('TJK', 'TJ'),
    ('TKL', 'TK'),
    ('TKM', 'TM'),
    ('TLS', 'TL'),
    ('TON', 'TO'),
    ('TTO', 'TT'),
    ('TUN', 'TN'),
    ('TUR', 'TR'),
    ('TUV', 'TV'),
    ('TWN', 'TW'),
    ('TZA', 'TZ'),
    ('UGA', 'UG'),
    ('UKR', 'UA'),
    ('UMI', 'UM'),
    ('URY', 'UY'),
    ('USA', 'US'),
    ('UZB', 'UZ'),
    ('VAT', 'VA'),
    ('VCT', 'VC'),
    ('VEN', 'VE'),
    ('VGB', 'VG'),
    ('VIR', 'VI'),
    ('VNM', 'VN'),
    ('VUT', 'VU'),
    ('WLF', 'WF'),
    ('WSM', 'WS'),
    ('YEM', 'YE'),
    ('ZAF', 'ZA'),
    ('ZMB', 'ZM'),
    ('ZWE', 'ZW')
;

-- Mirrors internal/core/id.Derive. uuid.NameSpaceOID is hashed with the
-- UTF-8 seed PREFIX<NUL>NAMESPACE<NUL>PART and RFC 4122 v5 bits.
-- +goose StatementBegin
CREATE FUNCTION migration_000050_derived_id(prefix TEXT, namespace TEXT, part TEXT)
RETURNS TEXT
LANGUAGE plpgsql
IMMUTABLE
STRICT
AS $$
DECLARE
    value BYTEA;
    encoded TEXT;
BEGIN
    IF prefix !~ '^[A-Z]{2,8}$' OR btrim(namespace) = '' OR btrim(part) = '' THEN
        RAISE EXCEPTION 'invalid deterministic ID input';
    END IF;
    value := substring(
        public.digest(
            uuid_send('6ba7b812-9dad-11d1-80b4-00c04fd430c8'::uuid)
            || convert_to(prefix, 'UTF8') || '\x00'::bytea
            || convert_to(btrim(namespace), 'UTF8') || '\x00'::bytea
            || convert_to(btrim(part), 'UTF8'),
            'sha1'
        )
        FROM 1 FOR 16
    );
    value := set_byte(value, 6, (get_byte(value, 6) & 15) | 80);
    value := set_byte(value, 8, (get_byte(value, 8) & 63) | 128);
    encoded := encode(value, 'hex');
    RETURN prefix
        || substring(encoded FROM 1 FOR 8) || '-'
        || substring(encoded FROM 9 FOR 4) || '-'
        || substring(encoded FROM 13 FOR 4) || '-'
        || substring(encoded FROM 17 FOR 4) || '-'
        || substring(encoded FROM 21 FOR 12);
END;
$$;
-- +goose StatementEnd

CREATE TEMP TABLE migration_000050_id_map (
    identity_kind TEXT NOT NULL,
    old_id TEXT NOT NULL,
    new_id VARCHAR(39) NOT NULL,
    PRIMARY KEY (identity_kind, old_id),
    UNIQUE (identity_kind, new_id)
) ON COMMIT DROP;

INSERT INTO migration_000050_id_map
SELECT 'entity', id::text, 'ENT' || id::text FROM entity_nodes;

INSERT INTO migration_000050_id_map
SELECT 'entity_relation', id::text, 'ERL' || id::text FROM entity_edges;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM countries country
        LEFT JOIN migration_000050_country_codes code ON code.alpha3 = btrim(country.code)
        WHERE code.alpha3 IS NULL
    ) THEN
        RAISE EXCEPTION 'Country migration found a code outside ISO 3166-1 alpha-3';
    END IF;
END;
$$;
-- +goose StatementEnd

INSERT INTO migration_000050_id_map
SELECT 'country', country.id, migration_000050_derived_id('COU', 'country', code.alpha2)
FROM countries country
JOIN migration_000050_country_codes code ON code.alpha3 = btrim(country.code);

INSERT INTO migration_000050_id_map
SELECT 'region', id, migration_000050_derived_id('REG', 'region', code)
FROM regions;

INSERT INTO migration_000050_id_map
SELECT 'organization', id, migration_000050_derived_id('ORG', 'organization', code)
FROM organizations;

INSERT INTO migration_000050_id_map
SELECT 'raw_evidence', raw_evidence_id,
       migration_000050_derived_id('RAW', 'raw-evidence-legacy', raw_evidence_id)
FROM raw_evidences;

INSERT INTO migration_000050_id_map
SELECT 'evidence', evidence_id,
       migration_000050_derived_id('EVD', 'evidence-legacy', evidence_id)
FROM evidences;

INSERT INTO migration_000050_id_map
SELECT 'evidence_category', id,
       migration_000050_derived_id('EVC', 'evidence-category', id)
FROM evidence_categories;

CREATE TEMP TABLE migration_000050_identity_columns (
    table_oid OID NOT NULL,
    column_attnum SMALLINT NOT NULL,
    table_identity TEXT NOT NULL,
    column_name TEXT NOT NULL,
    depth INTEGER NOT NULL,
    PRIMARY KEY (table_oid, column_attnum)
) ON COMMIT DROP;

CREATE TEMP TABLE migration_000050_fk_definitions (
    table_oid OID NOT NULL,
    table_identity TEXT NOT NULL,
    constraint_name TEXT NOT NULL,
    constraint_definition TEXT NOT NULL,
    PRIMARY KEY (table_oid, constraint_name)
) ON COMMIT DROP;

CREATE TEMP TABLE migration_000050_trigger_definitions (
    table_oid OID NOT NULL,
    table_identity TEXT NOT NULL,
    trigger_name TEXT NOT NULL,
    trigger_definition TEXT NOT NULL,
    PRIMARY KEY (table_oid, trigger_name)
) ON COMMIT DROP;

-- Rewrites a root identity and every transitive FK column that carries the
-- same logical identity. Composite FKs are dropped and restored as a whole.
-- +goose StatementBegin
CREATE FUNCTION migration_000050_rewrite_identity(
    root_table REGCLASS,
    root_column TEXT,
    selected_kind TEXT
)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    inserted_count INTEGER;
    item RECORD;
    missing_count BIGINT;
BEGIN
    TRUNCATE migration_000050_identity_columns;
    TRUNCATE migration_000050_fk_definitions;
    TRUNCATE migration_000050_trigger_definitions;

    INSERT INTO migration_000050_identity_columns
    SELECT relation.oid, attribute.attnum,
           format('%I.%I', namespace.nspname, relation.relname),
           attribute.attname, 0
    FROM pg_class relation
    JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
    JOIN pg_attribute attribute ON attribute.attrelid = relation.oid
    WHERE relation.oid = root_table AND attribute.attname = root_column;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'identity root %.% is unavailable', root_table, root_column;
    END IF;

    LOOP
        INSERT INTO migration_000050_identity_columns
        SELECT constraint_row.conrelid, child_key.attnum,
               format('%I.%I', namespace.nspname, relation.relname),
               attribute.attname, parent.depth + 1
        FROM migration_000050_identity_columns parent
        JOIN pg_constraint constraint_row
          ON constraint_row.contype = 'f'
         AND constraint_row.confrelid = parent.table_oid
        JOIN LATERAL unnest(constraint_row.confkey) WITH ORDINALITY parent_key(attnum, position)
          ON parent_key.attnum = parent.column_attnum
        JOIN LATERAL unnest(constraint_row.conkey) WITH ORDINALITY child_key(attnum, position)
          ON child_key.position = parent_key.position
        JOIN pg_class relation ON relation.oid = constraint_row.conrelid
        JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
        JOIN pg_attribute attribute
          ON attribute.attrelid = constraint_row.conrelid
         AND attribute.attnum = child_key.attnum
        ON CONFLICT (table_oid, column_attnum) DO NOTHING;
        GET DIAGNOSTICS inserted_count = ROW_COUNT;
        EXIT WHEN inserted_count = 0;
    END LOOP;

    INSERT INTO migration_000050_fk_definitions
    SELECT DISTINCT constraint_row.conrelid,
           format('%I.%I', namespace.nspname, relation.relname),
           constraint_row.conname,
           pg_get_constraintdef(constraint_row.oid)
    FROM migration_000050_identity_columns parent
    JOIN pg_constraint constraint_row
      ON constraint_row.contype = 'f'
     AND constraint_row.confrelid = parent.table_oid
    JOIN LATERAL unnest(constraint_row.confkey) parent_key(attnum)
      ON parent_key.attnum = parent.column_attnum
    JOIN pg_class relation ON relation.oid = constraint_row.conrelid
    JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace;

    -- Updating an identity-bearing column re-evaluates every CHECK on that
    -- row. Temporarily preserve all checks on touched tables so historical
    -- rows allowed by creation-time cutover checks remain migratable.
    INSERT INTO migration_000050_fk_definitions
    SELECT DISTINCT constraint_row.conrelid,
           format('%I.%I', namespace.nspname, relation.relname),
           constraint_row.conname,
           pg_get_constraintdef(constraint_row.oid)
    FROM migration_000050_identity_columns identity_column
    JOIN pg_constraint constraint_row
      ON constraint_row.contype = 'c'
     AND constraint_row.conrelid = identity_column.table_oid
    JOIN pg_class relation ON relation.oid = constraint_row.conrelid
    JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
    ON CONFLICT (table_oid, constraint_name) DO NOTHING;

    INSERT INTO migration_000050_trigger_definitions
    SELECT DISTINCT trigger_row.tgrelid,
           identity_column.table_identity,
           trigger_row.tgname,
           pg_get_triggerdef(trigger_row.oid)
    FROM migration_000050_identity_columns identity_column
    JOIN pg_trigger trigger_row
      ON trigger_row.tgrelid = identity_column.table_oid
     AND NOT trigger_row.tgisinternal;

    FOR item IN
        SELECT * FROM migration_000050_fk_definitions
        ORDER BY table_identity, constraint_name
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', item.table_identity, item.constraint_name);
    END LOOP;

    FOR item IN
        SELECT * FROM migration_000050_trigger_definitions
        ORDER BY table_identity, trigger_name
    LOOP
        EXECUTE format('DROP TRIGGER %I ON %s', item.trigger_name, item.table_identity);
    END LOOP;

    FOR item IN
        SELECT * FROM migration_000050_identity_columns
        ORDER BY depth DESC, table_identity, column_name
    LOOP
        EXECUTE format(
            'ALTER TABLE %s ALTER COLUMN %I TYPE VARCHAR(39) USING %I::text',
            item.table_identity, item.column_name, item.column_name
        );
        EXECUTE format(
            'UPDATE %s target SET %I = mapping.new_id '
            'FROM migration_000050_id_map mapping '
            'WHERE mapping.identity_kind = %L AND target.%I::text = mapping.old_id',
            item.table_identity, item.column_name, selected_kind, item.column_name
        );
        EXECUTE format(
            'SELECT count(*) FROM %s target '
            'WHERE target.%I IS NOT NULL AND NOT EXISTS ('
            'SELECT 1 FROM migration_000050_id_map mapping '
            'WHERE mapping.identity_kind = %L AND mapping.new_id = target.%I::text)',
            item.table_identity, item.column_name, selected_kind, item.column_name
        ) INTO missing_count;
        IF missing_count <> 0 THEN
            RAISE EXCEPTION '% has % unmapped % identities', item.table_identity, missing_count, selected_kind;
        END IF;
    END LOOP;

    FOR item IN
        SELECT * FROM migration_000050_fk_definitions
        ORDER BY table_identity, constraint_name
    LOOP
        EXECUTE format(
            'ALTER TABLE %s ADD CONSTRAINT %I %s',
            item.table_identity, item.constraint_name, item.constraint_definition
        );
    END LOOP;

    FOR item IN
        SELECT * FROM migration_000050_trigger_definitions
        ORDER BY table_identity, trigger_name
    LOOP
        EXECUTE item.trigger_definition;
    END LOOP;
END;
$$;
-- +goose StatementEnd

ALTER TABLE countries
    DROP CONSTRAINT chk_countries_identity,
    DROP CONSTRAINT chk_countries_code;
ALTER TABLE regions DROP CONSTRAINT chk_regions_identity;
ALTER TABLE organizations DROP CONSTRAINT chk_organizations_identity;
ALTER TABLE evidence_categories DROP CONSTRAINT chk_evidence_categories_id;

SELECT migration_000050_rewrite_identity('entity_nodes', 'id', 'entity');
SELECT migration_000050_rewrite_identity('entity_edges', 'id', 'entity_relation');
SELECT migration_000050_rewrite_identity('countries', 'id', 'country');
SELECT migration_000050_rewrite_identity('regions', 'id', 'region');
SELECT migration_000050_rewrite_identity('organizations', 'id', 'organization');
SELECT migration_000050_rewrite_identity('raw_evidences', 'raw_evidence_id', 'raw_evidence');
SELECT migration_000050_rewrite_identity('evidences', 'evidence_id', 'evidence');
SELECT migration_000050_rewrite_identity('evidence_categories', 'id', 'evidence_category');

UPDATE countries country
SET code = code.alpha2
FROM migration_000050_country_codes code
JOIN migration_000050_id_map mapping
  ON mapping.identity_kind = 'country'
WHERE country.id = mapping.new_id
  AND code.alpha3 = btrim(country.code);

ALTER TABLE countries
    ALTER COLUMN code TYPE CHAR(2) USING btrim(code)::char(2),
    ADD CONSTRAINT chk_countries_code CHECK (code ~ '^[A-Z]{2}$'),
    ADD CONSTRAINT chk_countries_identity CHECK (
        id ~ '^COU[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    );

ALTER TABLE entity_nodes ADD CONSTRAINT chk_entity_nodes_identity CHECK (
    id ~ '^ENT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
);
ALTER TABLE entity_edges ADD CONSTRAINT chk_entity_edges_identity CHECK (
    id ~ '^ERL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
);
ALTER TABLE regions ADD CONSTRAINT chk_regions_identity CHECK (
    id ~ '^REG[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
);
ALTER TABLE organizations ADD CONSTRAINT chk_organizations_identity CHECK (
    id ~ '^ORG[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
);
ALTER TABLE raw_evidences ADD CONSTRAINT chk_raw_evidences_domain_identity CHECK (
    raw_evidence_id ~ '^RAW[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
);
ALTER TABLE evidences ADD CONSTRAINT chk_evidences_domain_identity CHECK (
    evidence_id ~ '^EVD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
);
ALTER TABLE evidence_categories ADD CONSTRAINT chk_evidence_categories_id CHECK (
    id ~ '^EVC[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
);

DROP FUNCTION migration_000050_rewrite_identity(REGCLASS, TEXT, TEXT);
DROP FUNCTION migration_000050_derived_id(TEXT, TEXT, TEXT);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000050 is forward-only; restore the reviewed pre-migration snapshot';
END;
$$;
-- +goose StatementEnd
