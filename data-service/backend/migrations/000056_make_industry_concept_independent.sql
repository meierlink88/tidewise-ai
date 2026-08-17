-- +goose Up
-- This migration is a coordinated stop-write cutover. It preserves every profiled
-- Industry/Concept identity and every supported reference before removing shadow
-- Entity rows. Unprofiled legacy Entity rows remain untouched.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM industry_profiles profile
        LEFT JOIN entity_nodes entity ON entity.id = profile.entity_id
        WHERE entity.id IS NULL
           OR entity.entity_type <> 'industry'
           OR entity.status <> 'active'
           OR entity.canonical_name <> entity.name
           OR btrim(entity.name) = ''
           OR entity.aliases IS NULL
    ) THEN
        RAISE EXCEPTION 'Industry profiles cannot be represented without losing Entity facts';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM concept_profiles profile
        LEFT JOIN entity_nodes entity ON entity.id = profile.entity_id
        WHERE entity.id IS NULL
           OR entity.entity_type <> 'concept'
           OR entity.status <> 'active'
           OR entity.canonical_name <> entity.name
           OR btrim(entity.name) = ''
           OR entity.aliases IS NULL
    ) THEN
        RAISE EXCEPTION 'Concept profiles cannot be represented without losing Entity facts';
    END IF;

    IF EXISTS (
        SELECT classification_system, industry_code
        FROM industry_profiles
        GROUP BY classification_system, industry_code
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'Industry classification versions cannot be removed without an identity collision';
    END IF;

    IF EXISTS (
        WITH independent_ids AS (
            SELECT entity_id id FROM industry_profiles
            UNION ALL
            SELECT entity_id FROM concept_profiles
        )
        SELECT 1 FROM index_profiles value JOIN independent_ids target ON target.id = value.market_entity_id
        UNION ALL
        SELECT 1 FROM security_profiles value JOIN independent_ids target ON target.id = value.issuer_company_entity_id
        UNION ALL
        SELECT 1 FROM instrument_profiles value JOIN independent_ids target ON target.id = value.underlying_entity_id
        UNION ALL
        SELECT 1 FROM person_profiles value JOIN independent_ids target ON target.id = value.organization_entity_id
    ) THEN
        RAISE EXCEPTION 'A profiled Industry or Concept is referenced through a typed legacy Entity foreign key';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER trg_industry_profile_entity_type ON industry_profiles;
DROP TRIGGER trg_concept_profile_entity_type ON concept_profiles;
DROP TRIGGER trg_validate_industry_profile_hierarchy ON industry_profiles;
DROP FUNCTION validate_industry_profile_hierarchy();

ALTER TABLE industry_profiles RENAME TO industry;
ALTER TABLE concept_profiles RENAME TO concept;

ALTER TABLE industry
    DROP CONSTRAINT industry_profiles_parent_industry_entity_id_fkey;

ALTER TABLE industry
    DROP CONSTRAINT industry_profiles_pkey,
    DROP CONSTRAINT industry_profiles_entity_id_fkey,
    DROP CONSTRAINT uq_industry_profile_classification_identity,
    DROP CONSTRAINT chk_industry_profile_version_nonblank,
    DROP CONSTRAINT chk_industry_profile_level,
    DROP CONSTRAINT chk_industry_profile_parent_presence,
    DROP CONSTRAINT chk_industry_profile_path_length,
    DROP CONSTRAINT chk_industry_profile_boundary_nonblank;

ALTER TABLE industry RENAME COLUMN entity_id TO id;
ALTER TABLE industry RENAME COLUMN parent_industry_entity_id TO parent_industry_id;

DROP INDEX idx_industry_profiles_parent;
DROP INDEX idx_industry_profiles_classification_level;
DROP INDEX idx_industry_profiles_review_status;

ALTER TABLE industry
    ADD COLUMN name TEXT,
    ADD COLUMN aliases TEXT[];

UPDATE industry value
SET name = entity.name,
    aliases = entity.aliases
FROM entity_nodes entity
WHERE entity.id = value.id;

ALTER TABLE industry
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN aliases SET NOT NULL,
    ALTER COLUMN aliases SET DEFAULT '{}'::TEXT[],
    DROP COLUMN classification_version,
    DROP COLUMN classification_level,
    DROP COLUMN boundary_note,
    ADD CONSTRAINT industry_pkey PRIMARY KEY (id),
    ADD CONSTRAINT fk_industry_parent
        FOREIGN KEY (parent_industry_id) REFERENCES industry(id) ON DELETE RESTRICT,
    ADD CONSTRAINT uq_industry_classification_identity
        UNIQUE (classification_system, industry_code),
    ADD CONSTRAINT chk_industry_identity CHECK (
        id ~ '^ENT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    ADD CONSTRAINT chk_industry_name_nonblank CHECK (btrim(name) <> ''),
    ADD CONSTRAINT chk_industry_aliases CHECK (
        array_position(aliases, NULL) IS NULL
    ),
    ADD CONSTRAINT chk_industry_root_path CHECK (
        (parent_industry_id IS NULL AND cardinality(hierarchy_path_codes) = 1)
        OR (parent_industry_id IS NOT NULL AND cardinality(hierarchy_path_codes) >= 2)
    );

CREATE INDEX idx_industry_parent ON industry (parent_industry_id);
CREATE INDEX idx_industry_review_classification
    ON industry (review_status, classification_system, industry_code);
CREATE INDEX idx_industry_name ON industry (name, id);
CREATE INDEX idx_industry_aliases_gin ON industry USING GIN (aliases);

ALTER TABLE concept
    DROP CONSTRAINT concept_profiles_pkey,
    DROP CONSTRAINT concept_profiles_entity_id_fkey,
    DROP CONSTRAINT chk_concept_profile_boundary_nonblank;

ALTER TABLE concept RENAME COLUMN entity_id TO id;

DROP INDEX idx_concept_profiles_type_review;

ALTER TABLE concept
    ADD COLUMN name TEXT,
    ADD COLUMN aliases TEXT[];

UPDATE concept value
SET name = entity.name,
    aliases = entity.aliases
FROM entity_nodes entity
WHERE entity.id = value.id;

ALTER TABLE concept
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN aliases SET NOT NULL,
    ALTER COLUMN aliases SET DEFAULT '{}'::TEXT[],
    DROP COLUMN boundary_note,
    ADD CONSTRAINT concept_pkey PRIMARY KEY (id),
    ADD CONSTRAINT chk_concept_identity CHECK (
        id ~ '^ENT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    ADD CONSTRAINT chk_concept_name_nonblank CHECK (btrim(name) <> ''),
    ADD CONSTRAINT chk_concept_aliases CHECK (
        array_position(aliases, NULL) IS NULL
    );

CREATE INDEX idx_concept_type_review ON concept (concept_type, review_status, name, id);
CREATE INDEX idx_concept_aliases_gin ON concept USING GIN (aliases);

-- +goose StatementBegin
CREATE FUNCTION validate_industry_hierarchy()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    parent_system TEXT;
    parent_path TEXT[];
BEGIN
    IF NEW.parent_industry_id IS NULL THEN
        IF NEW.hierarchy_path_codes <> ARRAY[NEW.industry_code]::TEXT[] THEN
            RAISE EXCEPTION 'root Industry path must contain only its own code';
        END IF;
    ELSE
        IF NEW.parent_industry_id = NEW.id THEN
            RAISE EXCEPTION 'Industry cannot be its own parent';
        END IF;

        SELECT classification_system, hierarchy_path_codes
        INTO parent_system, parent_path
        FROM industry
        WHERE id = NEW.parent_industry_id
        FOR SHARE;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'Industry parent % does not exist', NEW.parent_industry_id;
        END IF;
        IF parent_system <> NEW.classification_system THEN
            RAISE EXCEPTION 'Industry parent must use the same classification system';
        END IF;
        IF NEW.hierarchy_path_codes <> (parent_path || NEW.industry_code) THEN
            RAISE EXCEPTION 'Industry hierarchy path must extend the direct parent path';
        END IF;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        PERFORM 1
        FROM industry child
        WHERE child.parent_industry_id = OLD.id
        FOR SHARE;

        IF EXISTS (
            SELECT 1
            FROM industry child
            WHERE child.parent_industry_id = OLD.id
              AND (
                  child.classification_system <> NEW.classification_system
                  OR child.hierarchy_path_codes <> (NEW.hierarchy_path_codes || child.industry_code)
              )
        ) THEN
            RAISE EXCEPTION 'Industry update would invalidate an existing child hierarchy';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_validate_industry_hierarchy
BEFORE INSERT OR UPDATE ON industry
FOR EACH ROW EXECUTE FUNCTION validate_industry_hierarchy();

ALTER TABLE entity_edges
    DROP CONSTRAINT entity_edges_from_entity_id_fkey,
    DROP CONSTRAINT entity_edges_to_entity_id_fkey;
ALTER TABLE entity_external_identifiers
    DROP CONSTRAINT entity_external_identifiers_entity_id_fkey;
ALTER TABLE event_entity_links
    DROP CONSTRAINT event_entity_links_entity_id_fkey;
ALTER TABLE entity_redirects
    DROP CONSTRAINT entity_redirects_source_entity_id_fkey,
    DROP CONSTRAINT entity_redirects_target_entity_id_fkey;
ALTER TABLE direct_impact_assertions
    DROP CONSTRAINT direct_impact_assertions_target_entity_id_fkey;
ALTER TABLE event_semantic_resolution_bindings
    DROP CONSTRAINT event_semantic_resolution_bindings_anchor_entity_id_fkey,
    DROP CONSTRAINT event_semantic_resolution_bindings_target_entity_id_fkey;

-- +goose StatementBegin
CREATE FUNCTION data_object_type(reference_id TEXT)
RETURNS TEXT
LANGUAGE SQL
STABLE
AS $$
    SELECT object_type
    FROM (
        SELECT entity_type::TEXT object_type FROM entity_nodes WHERE id = reference_id
        UNION ALL
        SELECT 'industry' FROM industry WHERE id = reference_id
        UNION ALL
        SELECT 'concept' FROM concept WHERE id = reference_id
    ) value
    LIMIT 1
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION assert_data_object_references()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    column_name TEXT;
    reference_id TEXT;
BEGIN
    FOREACH column_name IN ARRAY TG_ARGV LOOP
        reference_id := to_jsonb(NEW) ->> column_name;
        IF reference_id IS NOT NULL AND data_object_type(reference_id) IS NULL THEN
            RAISE EXCEPTION '% references unknown Data object % through %', TG_TABLE_NAME, reference_id, column_name;
        END IF;
    END LOOP;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_entity_edges_object_references
BEFORE INSERT OR UPDATE OF from_entity_id, to_entity_id ON entity_edges
FOR EACH ROW EXECUTE FUNCTION assert_data_object_references('from_entity_id', 'to_entity_id');
CREATE TRIGGER trg_external_identifier_object_reference
BEFORE INSERT OR UPDATE OF entity_id ON entity_external_identifiers
FOR EACH ROW EXECUTE FUNCTION assert_data_object_references('entity_id');
CREATE TRIGGER trg_event_entity_link_object_reference
BEFORE INSERT OR UPDATE OF entity_id ON event_entity_links
FOR EACH ROW WHEN (NEW.entity_id IS NOT NULL)
EXECUTE FUNCTION assert_data_object_references('entity_id');
CREATE TRIGGER trg_entity_redirect_object_references
BEFORE INSERT OR UPDATE OF source_entity_id, target_entity_id ON entity_redirects
FOR EACH ROW EXECUTE FUNCTION assert_data_object_references('source_entity_id', 'target_entity_id');
CREATE TRIGGER trg_direct_impact_object_reference
BEFORE INSERT OR UPDATE OF target_entity_id ON direct_impact_assertions
FOR EACH ROW EXECUTE FUNCTION assert_data_object_references('target_entity_id');
CREATE TRIGGER trg_resolution_binding_object_references
BEFORE INSERT OR UPDATE OF anchor_entity_id, target_entity_id ON event_semantic_resolution_bindings
FOR EACH ROW EXECUTE FUNCTION assert_data_object_references('anchor_entity_id', 'target_entity_id');

-- The former Entity foreign keys also provided ON DELETE protection. Keep that
-- contract after the polymorphic references start resolving against three object
-- tables instead of entity_nodes alone.
-- +goose StatementBegin
CREATE FUNCTION protect_independent_data_object_references()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM entity_edges WHERE from_entity_id = OLD.id OR to_entity_id = OLD.id)
       OR EXISTS (SELECT 1 FROM entity_external_identifiers WHERE entity_id = OLD.id)
       OR EXISTS (SELECT 1 FROM event_entity_links WHERE entity_id = OLD.id)
       OR EXISTS (
           SELECT 1 FROM entity_redirects
           WHERE source_entity_id = OLD.id OR target_entity_id = OLD.id
       )
       OR EXISTS (SELECT 1 FROM direct_impact_assertions WHERE target_entity_id = OLD.id)
       OR EXISTS (
           SELECT 1 FROM event_semantic_resolution_bindings
           WHERE anchor_entity_id = OLD.id OR target_entity_id = OLD.id
       ) THEN
        RAISE EXCEPTION 'Data object % is still referenced and cannot be deleted', OLD.id;
    END IF;
    RETURN OLD;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_industry_protect_object_references
BEFORE DELETE ON industry
FOR EACH ROW EXECUTE FUNCTION protect_independent_data_object_references();
CREATE TRIGGER trg_concept_protect_object_references
BEFORE DELETE ON concept
FOR EACH ROW EXECUTE FUNCTION protect_independent_data_object_references();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_entity_redirect()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    source_type TEXT;
    target_type TEXT;
BEGIN
    source_type := data_object_type(NEW.source_entity_id);
    target_type := data_object_type(NEW.target_entity_id);

    PERFORM pg_advisory_xact_lock(hashtext('entity_redirects'));

    IF source_type IS NULL OR target_type IS NULL THEN
        RAISE EXCEPTION 'redirect source and target identities must exist';
    END IF;
    IF NEW.redirect_kind = 'merge' AND source_type <> target_type THEN
        RAISE EXCEPTION 'merge redirect requires source and target of the same type';
    END IF;
    IF NEW.redirect_kind = 'reclassification' AND source_type = target_type THEN
        RAISE EXCEPTION 'reclassification redirect requires source and target of different types';
    END IF;
    IF EXISTS (
        WITH RECURSIVE reachable(object_id) AS (
            SELECT NEW.target_entity_id
            UNION
            SELECT redirect.target_entity_id
            FROM entity_redirects redirect
            JOIN reachable current_path ON redirect.source_entity_id = current_path.object_id
            WHERE redirect.source_entity_id <> NEW.source_entity_id
        )
        SELECT 1 FROM reachable WHERE object_id = NEW.source_entity_id
    ) THEN
        RAISE EXCEPTION 'Data object redirect graph must remain acyclic';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION protect_profiled_entity_identity()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM chain_node_profiles WHERE entity_id = OLD.id)
       AND (NEW.entity_type <> 'chain_node' OR btrim(NEW.entity_key) = '') THEN
        RAISE EXCEPTION 'chain node profile identity cannot change type or use a blank key';
    END IF;
    IF EXISTS (SELECT 1 FROM industry_chain_definitions WHERE entity_id = OLD.id)
       AND (NEW.entity_type <> 'industry_chain' OR btrim(NEW.entity_key) = '') THEN
        RAISE EXCEPTION 'industry chain profile identity cannot change type or use a blank key';
    END IF;

    IF NEW.entity_type IS DISTINCT FROM OLD.entity_type AND EXISTS (
        SELECT 1 FROM entity_redirects
        WHERE source_entity_id = OLD.id OR target_entity_id = OLD.id
    ) THEN
        RAISE EXCEPTION 'Entity type change would invalidate an existing Data object redirect';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- Supported references now resolve against the independent tables. Removing the
-- profiled shadows is the proof that runtime behavior cannot fall back to them.
DELETE FROM entity_nodes entity
WHERE entity.id IN (
    SELECT id FROM industry
    UNION ALL
    SELECT id FROM concept
);

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM industry value JOIN entity_nodes entity ON entity.id = value.id
        UNION ALL
        SELECT 1 FROM concept value JOIN entity_nodes entity ON entity.id = value.id
    ) THEN
        RAISE EXCEPTION 'Independent Industry or Concept still has a shadow Entity row';
    END IF;

    IF EXISTS (
        SELECT from_entity_id object_id FROM entity_edges
        UNION ALL SELECT to_entity_id FROM entity_edges
        UNION ALL SELECT entity_id FROM entity_external_identifiers
        UNION ALL SELECT entity_id FROM event_entity_links WHERE entity_id IS NOT NULL
        UNION ALL SELECT source_entity_id FROM entity_redirects
        UNION ALL SELECT target_entity_id FROM entity_redirects
        UNION ALL SELECT target_entity_id FROM direct_impact_assertions
        UNION ALL SELECT anchor_entity_id FROM event_semantic_resolution_bindings
        UNION ALL SELECT target_entity_id FROM event_semantic_resolution_bindings
        EXCEPT
        SELECT id FROM entity_nodes
        EXCEPT
        SELECT id FROM industry
        EXCEPT
        SELECT id FROM concept
    ) THEN
        RAISE EXCEPTION 'A migrated Data object reference is orphaned';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000056 is forward-only; restore the pre-cutover PostgreSQL snapshot and previous Data Service';
END;
$$;
-- +goose StatementEnd
