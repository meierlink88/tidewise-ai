-- +goose Up
-- Coordinated stop-write cutover. Preserve all profiled ChainNode and IndustryChain
-- identities and supported references, then remove only their shadow Entity rows.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM chain_node_profiles profile
        LEFT JOIN entity_nodes entity ON entity.id = profile.entity_id
        WHERE entity.id IS NULL
           OR entity.entity_type <> 'chain_node'
           OR entity.status <> 'active'
           OR entity.canonical_name <> entity.name
           OR btrim(entity.name) = ''
           OR entity.aliases IS NULL
           OR array_position(entity.aliases, NULL) IS NOT NULL
           OR EXISTS (
               SELECT 1 FROM unnest(entity.aliases) alias WHERE btrim(alias) = ''
           )
           OR cardinality(entity.aliases) <> (
               SELECT count(DISTINCT alias) FROM unnest(entity.aliases) alias
           )
           OR entity.updated_at < entity.created_at
           OR profile.review_status IS NULL
    ) THEN
        RAISE EXCEPTION 'ChainNode profiles cannot be represented without losing Entity facts';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM industry_chain_definitions definition
        LEFT JOIN entity_nodes entity ON entity.id = definition.entity_id
        WHERE entity.id IS NULL
           OR entity.entity_type <> 'industry_chain'
           OR entity.status <> 'active'
           OR entity.canonical_name <> entity.name
           OR btrim(entity.name) = ''
           OR entity.aliases IS NULL
           OR array_position(entity.aliases, NULL) IS NOT NULL
           OR EXISTS (
               SELECT 1 FROM unnest(entity.aliases) alias WHERE btrim(alias) = ''
           )
           OR cardinality(entity.aliases) <> (
               SELECT count(DISTINCT alias) FROM unnest(entity.aliases) alias
           )
           OR EXISTS (
               SELECT 1
               FROM unnest(definition.observable_variables) variable
               WHERE btrim(variable) = ''
           )
           OR cardinality(definition.observable_variables) <> (
               SELECT count(DISTINCT variable)
               FROM unnest(definition.observable_variables) variable
           )
           OR definition.updated_at < definition.created_at
    ) THEN
        RAISE EXCEPTION 'IndustryChain definitions cannot be represented without losing Entity facts';
    END IF;

    IF EXISTS (
        WITH independent_ids AS (
            SELECT entity_id id FROM chain_node_profiles
            UNION ALL
            SELECT entity_id FROM industry_chain_definitions
        )
        SELECT 1 FROM index_profiles value JOIN independent_ids target ON target.id = value.market_entity_id
        UNION ALL
        SELECT 1 FROM security_profiles value JOIN independent_ids target ON target.id = value.issuer_company_entity_id
        UNION ALL
        SELECT 1 FROM instrument_profiles value JOIN independent_ids target ON target.id = value.underlying_entity_id
        UNION ALL
        SELECT 1 FROM person_profiles value JOIN independent_ids target ON target.id = value.organization_entity_id
    ) THEN
        RAISE EXCEPTION 'A profiled ChainNode or IndustryChain is referenced through a typed legacy Entity foreign key';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION valid_independent_object_text_set(values_to_check TEXT[], allow_empty BOOLEAN)
RETURNS BOOLEAN
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT (allow_empty OR cardinality(values_to_check) > 0)
       AND array_position(values_to_check, NULL) IS NULL
       AND NOT EXISTS (
           SELECT 1 FROM unnest(values_to_check) value WHERE btrim(value) = ''
       )
       AND cardinality(values_to_check) = (
           SELECT count(DISTINCT value) FROM unnest(values_to_check) value
       )
$$;
-- +goose StatementEnd

DROP TRIGGER trg_chain_node_profile_entity_type ON chain_node_profiles;
DROP TRIGGER trg_chain_node_profile_review_status_entity_type ON chain_node_profiles;
DROP TRIGGER trg_industry_chain_definition_entity_type ON industry_chain_definitions;
DROP TRIGGER trg_protect_active_industry_chain_membership ON industry_chain_node_memberships;
DROP TRIGGER trg_reject_industry_chain_graph_cycle ON industry_chain_graph_edges;
DROP FUNCTION protect_active_industry_chain_membership();
DROP FUNCTION reject_industry_chain_graph_cycle();

ALTER TABLE chain_node_profiles RENAME TO chain_node;
ALTER TABLE chain_node RENAME COLUMN entity_id TO id;
ALTER TABLE chain_node DROP CONSTRAINT chain_node_profiles_entity_id_fkey;
ALTER TABLE chain_node RENAME CONSTRAINT chain_node_profiles_pkey TO chain_node_pkey;
ALTER TABLE chain_node
    RENAME CONSTRAINT chk_chain_node_profile_review_status TO chk_chain_node_review_status;
ALTER TABLE chain_node
    ADD COLUMN name TEXT,
    ADD COLUMN aliases TEXT[],
    ADD COLUMN created_at TIMESTAMPTZ,
    ADD COLUMN updated_at TIMESTAMPTZ;

UPDATE chain_node value
SET name = entity.name,
    aliases = entity.aliases,
    created_at = entity.created_at,
    updated_at = entity.updated_at
FROM entity_nodes entity
WHERE entity.id = value.id;

ALTER TABLE chain_node
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN aliases SET NOT NULL,
    ALTER COLUMN aliases SET DEFAULT '{}'::TEXT[],
    ALTER COLUMN review_status SET NOT NULL,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN created_at SET DEFAULT now(),
    ALTER COLUMN updated_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT now(),
    DROP COLUMN boundary_note,
    ADD CONSTRAINT chk_chain_node_identity CHECK (
        id ~ '^ENT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    ADD CONSTRAINT chk_chain_node_name_nonblank CHECK (btrim(name) <> ''),
    ADD CONSTRAINT chk_chain_node_aliases CHECK (
        valid_independent_object_text_set(aliases, TRUE)
    ),
    ADD CONSTRAINT chk_chain_node_timestamp_order CHECK (updated_at >= created_at);

CREATE INDEX idx_chain_node_review_name ON chain_node (review_status, name, id);
CREATE INDEX idx_chain_node_aliases_gin ON chain_node USING GIN (aliases);

ALTER TABLE industry_chain_definitions RENAME TO industry_chain;
ALTER TABLE industry_chain RENAME COLUMN entity_id TO id;
ALTER TABLE industry_chain DROP CONSTRAINT industry_chain_definitions_entity_id_fkey;
ALTER TABLE industry_chain
    RENAME CONSTRAINT industry_chain_definitions_pkey TO industry_chain_pkey;
ALTER TABLE industry_chain
    RENAME CONSTRAINT chk_industry_chain_definition_scope_nonblank TO chk_industry_chain_scope_nonblank;
ALTER TABLE industry_chain
    RENAME CONSTRAINT chk_industry_chain_definition_target_output_nonblank TO chk_industry_chain_target_output_nonblank;
ALTER TABLE industry_chain
    RENAME CONSTRAINT chk_industry_chain_definition_end_use_nonblank TO chk_industry_chain_end_use_nonblank;
ALTER TABLE industry_chain
    RENAME CONSTRAINT chk_industry_chain_definition_geography_nonblank TO chk_industry_chain_geography_nonblank;
ALTER TABLE industry_chain
    RENAME CONSTRAINT chk_industry_chain_definition_review_status TO chk_industry_chain_review_status;
ALTER TABLE industry_chain
    RENAME CONSTRAINT chk_industry_chain_definition_review_note_nonblank TO chk_industry_chain_review_note_nonblank;
ALTER TABLE industry_chain
    RENAME CONSTRAINT chk_industry_chain_definition_route_nonblank TO chk_industry_chain_route_nonblank;
ALTER TABLE industry_chain
    RENAME CONSTRAINT chk_industry_chain_definition_observable_variables TO chk_industry_chain_observable_variables;
ALTER TABLE industry_chain DROP CONSTRAINT chk_industry_chain_observable_variables;
ALTER INDEX idx_industry_chain_definitions_review_date RENAME TO idx_industry_chain_review_date;
ALTER INDEX idx_industry_chain_definitions_primary_country RENAME TO idx_industry_chain_primary_country;
ALTER TABLE industry_chain
    ADD COLUMN name TEXT,
    ADD COLUMN aliases TEXT[];

UPDATE industry_chain value
SET name = entity.name,
    aliases = entity.aliases
FROM entity_nodes entity
WHERE entity.id = value.id;

ALTER TABLE industry_chain
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN aliases SET NOT NULL,
    ALTER COLUMN aliases SET DEFAULT '{}'::TEXT[],
    ADD CONSTRAINT chk_industry_chain_identity CHECK (
        id ~ '^ENT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    ADD CONSTRAINT chk_industry_chain_name_nonblank CHECK (btrim(name) <> ''),
    ADD CONSTRAINT chk_industry_chain_aliases CHECK (
        valid_independent_object_text_set(aliases, TRUE)
    ),
    ADD CONSTRAINT chk_industry_chain_observable_variables CHECK (
        valid_independent_object_text_set(observable_variables, FALSE)
    ),
    ADD CONSTRAINT chk_industry_chain_timestamp_order CHECK (updated_at >= created_at);

CREATE INDEX idx_industry_chain_review_date_name
    ON industry_chain (review_status, as_of_date DESC, name, id);
CREATE INDEX idx_industry_chain_aliases_gin ON industry_chain USING GIN (aliases);

ALTER TABLE industry_chain_node_memberships
    RENAME COLUMN industry_chain_entity_id TO industry_chain_id;
ALTER TABLE industry_chain_node_memberships
    RENAME COLUMN chain_node_entity_id TO chain_node_id;

ALTER TABLE industry_chain_graph_edges
    RENAME COLUMN industry_chain_entity_id TO industry_chain_id;
ALTER TABLE industry_chain_graph_edges
    RENAME COLUMN from_chain_node_entity_id TO from_chain_node_id;
ALTER TABLE industry_chain_graph_edges
    RENAME COLUMN to_chain_node_entity_id TO to_chain_node_id;

ALTER TABLE chain_node_relations
    RENAME COLUMN from_chain_node_entity_id TO from_chain_node_id;
ALTER TABLE chain_node_relations
    RENAME COLUMN to_chain_node_entity_id TO to_chain_node_id;
ALTER TABLE chain_node_physical_constraints
    RENAME COLUMN chain_node_entity_id TO chain_node_id;

ALTER TABLE research_theme_impacts
    RENAME COLUMN chain_node_entity_id TO chain_node_id;
ALTER TABLE research_reasoning_trees
    RENAME COLUMN industry_chain_entity_id TO industry_chain_id;
ALTER TABLE research_reasoning_tree_nodes
    RENAME COLUMN chain_node_entity_id TO chain_node_id;
ALTER TABLE research_theme_import_receipts
    RENAME COLUMN reasoning_tree_ids_by_industry_chain_entity_id
    TO reasoning_tree_ids_by_industry_chain_id;
ALTER TABLE research_reasoning_tree_import_receipts
    RENAME COLUMN reasoning_tree_ids_by_industry_chain_entity_id
    TO reasoning_tree_ids_by_industry_chain_id;

-- +goose StatementBegin
CREATE FUNCTION protect_active_industry_chain_membership()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.status = 'active' AND NEW.status = 'inactive' THEN
        PERFORM pg_advisory_xact_lock(hashtext('industry_chain_topology:' || NEW.industry_chain_id));

        IF EXISTS (
            SELECT 1
            FROM industry_chain_graph_edges edge
            WHERE edge.industry_chain_id = NEW.industry_chain_id
              AND edge.status = 'active'
              AND NEW.chain_node_id IN (edge.from_chain_node_id, edge.to_chain_node_id)
        ) THEN
            RAISE EXCEPTION 'active industry chain graph edges require active memberships';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_protect_active_industry_chain_membership
BEFORE UPDATE OF status ON industry_chain_node_memberships
FOR EACH ROW EXECUTE FUNCTION protect_active_industry_chain_membership();

-- +goose StatementBegin
CREATE FUNCTION reject_industry_chain_graph_cycle()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.status = 'active' THEN
        PERFORM 1
        FROM industry_chain_node_memberships membership
        WHERE membership.industry_chain_id = NEW.industry_chain_id
          AND membership.chain_node_id IN (NEW.from_chain_node_id, NEW.to_chain_node_id)
        ORDER BY membership.chain_node_id
        FOR SHARE;

        IF EXISTS (
            SELECT 1
            FROM industry_chain_node_memberships membership
            WHERE membership.industry_chain_id = NEW.industry_chain_id
              AND membership.chain_node_id IN (NEW.from_chain_node_id, NEW.to_chain_node_id)
              AND membership.status <> 'active'
        ) THEN
            RAISE EXCEPTION 'active industry chain graph edges require active memberships';
        END IF;

        PERFORM pg_advisory_xact_lock(hashtext('industry_chain_topology:' || NEW.industry_chain_id));
    END IF;

    IF NEW.status = 'active' AND EXISTS (
        WITH RECURSIVE reachable(node_id) AS (
            SELECT NEW.to_chain_node_id
            UNION
            SELECT edge.to_chain_node_id
            FROM industry_chain_graph_edges edge
            JOIN reachable current_path ON edge.from_chain_node_id = current_path.node_id
            WHERE edge.industry_chain_id = NEW.industry_chain_id
              AND edge.status = 'active'
              AND edge.id <> NEW.id
        )
        SELECT 1 FROM reachable WHERE node_id = NEW.from_chain_node_id
    ) THEN
        RAISE EXCEPTION 'industry chain topology must remain acyclic';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_reject_industry_chain_graph_cycle
BEFORE INSERT OR UPDATE ON industry_chain_graph_edges
FOR EACH ROW EXECUTE FUNCTION reject_industry_chain_graph_cycle();

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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION protect_profiled_entity_identity()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
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

-- The old owner-delete guard sees the supported references before the independent
-- owner resolver takes over. Remove it only for the atomic shadow deletion.
DROP TRIGGER trg_entity_node_protect_object_delete ON entity_nodes;

DELETE FROM entity_nodes entity
WHERE entity.id IN (
    SELECT id FROM chain_node
    UNION ALL
    SELECT id FROM industry_chain
);

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
        IF NEW.entity_type IN ('industry', 'concept', 'chain_node', 'industry_chain')
           AND (TG_OP = 'INSERT' OR OLD.entity_type IS DISTINCT FROM NEW.entity_type) THEN
            RAISE EXCEPTION 'new independent facts must use their object tables';
        END IF;
        IF EXISTS (SELECT 1 FROM industry WHERE id = NEW.id)
           OR EXISTS (SELECT 1 FROM concept WHERE id = NEW.id)
           OR EXISTS (SELECT 1 FROM chain_node WHERE id = NEW.id)
           OR EXISTS (SELECT 1 FROM industry_chain WHERE id = NEW.id) THEN
            RAISE EXCEPTION 'Data object identity % already belongs to an independent object', NEW.id;
        END IF;
    ELSIF EXISTS (SELECT 1 FROM entity_nodes WHERE id = NEW.id)
       OR (TG_TABLE_NAME <> 'industry' AND EXISTS (SELECT 1 FROM industry WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'concept' AND EXISTS (SELECT 1 FROM concept WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'chain_node' AND EXISTS (SELECT 1 FROM chain_node WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'industry_chain' AND EXISTS (SELECT 1 FROM industry_chain WHERE id = NEW.id)) THEN
        RAISE EXCEPTION 'Data object identity % already belongs to another object', NEW.id;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_chain_node_object_identity_unique
BEFORE INSERT OR UPDATE OF id ON chain_node
FOR EACH ROW EXECUTE FUNCTION assert_data_object_identity_unique();
CREATE TRIGGER trg_industry_chain_object_identity_unique
BEFORE INSERT OR UPDATE OF id ON industry_chain
FOR EACH ROW EXECUTE FUNCTION assert_data_object_identity_unique();

CREATE TRIGGER trg_entity_node_protect_object_delete
BEFORE DELETE ON entity_nodes
FOR EACH ROW EXECUTE FUNCTION protect_data_object_references();
CREATE TRIGGER trg_chain_node_protect_object_delete
BEFORE DELETE ON chain_node
FOR EACH ROW EXECUTE FUNCTION protect_data_object_references();
CREATE TRIGGER trg_chain_node_protect_object_id_update
BEFORE UPDATE OF id ON chain_node
FOR EACH ROW EXECUTE FUNCTION protect_data_object_references();
CREATE TRIGGER trg_industry_chain_protect_object_delete
BEFORE DELETE ON industry_chain
FOR EACH ROW EXECUTE FUNCTION protect_data_object_references();
CREATE TRIGGER trg_industry_chain_protect_object_id_update
BEFORE UPDATE OF id ON industry_chain
FOR EACH ROW EXECUTE FUNCTION protect_data_object_references();
CREATE TRIGGER trg_chain_node_protect_object_truncate
BEFORE TRUNCATE ON chain_node
FOR EACH STATEMENT EXECUTE FUNCTION protect_data_object_truncate();
CREATE TRIGGER trg_industry_chain_protect_object_truncate
BEFORE TRUNCATE ON industry_chain
FOR EACH STATEMENT EXECUTE FUNCTION protect_data_object_truncate();

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM chain_node value JOIN entity_nodes entity ON entity.id = value.id
        UNION ALL
        SELECT 1 FROM industry_chain value JOIN entity_nodes entity ON entity.id = value.id
    ) THEN
        RAISE EXCEPTION 'Independent ChainNode or IndustryChain still has a shadow Entity row';
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
        EXCEPT SELECT id FROM entity_nodes
        EXCEPT SELECT id FROM industry
        EXCEPT SELECT id FROM concept
        EXCEPT SELECT id FROM chain_node
        EXCEPT SELECT id FROM industry_chain
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
        MESSAGE = 'migration 000057 is forward-only; restore the pre-cutover PostgreSQL snapshot and previous Data Service';
END;
$$;
-- +goose StatementEnd
