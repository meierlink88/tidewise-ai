-- +goose Up
-- Coordinated stop-write cutover. The two formal IndustryChain mapping sets move
-- out of generic Entity Relations. The known simulated wafer fixture is retired.

LOCK TABLE entity_edges, industry_chain, industry, concept, chain_node,
    industry_chain_node_memberships, industry_chain_graph_edges, storylines,
    chain_node_relations, chain_node_physical_constraints,
    entity_external_identifiers, entity_redirects IN SHARE ROW EXCLUSIVE MODE;

-- +goose StatementBegin
DO $$
DECLARE
    simulated_chain_id CONSTANT TEXT := 'ICH23000000-0000-4000-8000-000000000001';
    simulated_node_ids CONSTANT TEXT[] := ARRAY[
        'CND22000000-0000-4000-8000-000000000001',
        'CND22000000-0000-4000-8000-000000000002'
    ];
    fixture_present BOOLEAN;
BEGIN
    SELECT EXISTS (SELECT 1 FROM industry_chain WHERE id = simulated_chain_id)
    INTO fixture_present;

    IF fixture_present THEN
        IF NOT EXISTS (
            SELECT 1 FROM industry_chain
            WHERE id = simulated_chain_id AND name = '模拟晶圆产业链'
        ) THEN
            RAISE EXCEPTION 'The simulated IndustryChain identity belongs to non-fixture data';
        END IF;
        IF (SELECT count(*) FROM chain_node WHERE id = ANY(simulated_node_ids)) <> 2
           OR EXISTS (
               SELECT 1 FROM chain_node
               WHERE id = simulated_node_ids[1] AND name <> '模拟晶圆生产'
               UNION ALL
               SELECT 1 FROM chain_node
               WHERE id = simulated_node_ids[2] AND name <> '模拟8英寸晶圆供给'
           ) THEN
            RAISE EXCEPTION 'The simulated IndustryChain nodes are incomplete or no longer synthetic';
        END IF;
        IF (SELECT count(*) FROM industry_chain_node_memberships
            WHERE industry_chain_id = simulated_chain_id) <> 2
           OR EXISTS (
               SELECT 1 FROM industry_chain_node_memberships
               WHERE industry_chain_id = simulated_chain_id
                 AND chain_node_id <> ALL(simulated_node_ids)
           )
           OR EXISTS (
               SELECT 1 FROM industry_chain_node_memberships
               WHERE chain_node_id = ANY(simulated_node_ids)
                 AND industry_chain_id <> simulated_chain_id
           ) THEN
            RAISE EXCEPTION 'The simulated IndustryChain nodes are not exclusive to the fixture';
        END IF;
        IF EXISTS (
            SELECT 1 FROM industry_chain_graph_edges WHERE industry_chain_id = simulated_chain_id
            UNION ALL
            SELECT 1 FROM storylines WHERE industry_chain_id = simulated_chain_id
            UNION ALL
            SELECT 1 FROM chain_node_relations
            WHERE from_chain_node_id = ANY(simulated_node_ids)
               OR to_chain_node_id = ANY(simulated_node_ids)
            UNION ALL
            SELECT 1 FROM chain_node_physical_constraints
            WHERE chain_node_id = ANY(simulated_node_ids)
        ) THEN
            RAISE EXCEPTION 'The simulated IndustryChain fixture owns non-disposable dependent facts';
        END IF;
        IF (SELECT count(*) FROM entity_edges
            WHERE from_entity_id = simulated_chain_id
               OR to_entity_id = simulated_chain_id
               OR from_entity_id = ANY(simulated_node_ids)
               OR to_entity_id = ANY(simulated_node_ids)) <> 2
           OR NOT EXISTS (
               SELECT 1 FROM entity_edges
               WHERE id = 'ERL26400000-0000-4000-8000-000000000004'
                 AND from_entity_id = simulated_chain_id
                 AND relation_type = 'mapped_to_industry'
           )
           OR NOT EXISTS (
               SELECT 1 FROM entity_edges
               WHERE id = 'ERL22000000-0000-4000-8000-000000000003'
                 AND from_entity_id = simulated_node_ids[1]
                 AND to_entity_id = simulated_node_ids[2]
                 AND relation_type = 'produces'
           ) THEN
            RAISE EXCEPTION 'The simulated IndustryChain fixture has unexpected Entity Relations';
        END IF;
        IF EXISTS (
            SELECT 1 FROM entity_external_identifiers WHERE entity_id = simulated_chain_id OR entity_id = ANY(simulated_node_ids)
            UNION ALL
            SELECT 1 FROM entity_redirects
            WHERE source_entity_id = simulated_chain_id OR target_entity_id = simulated_chain_id
               OR source_entity_id = ANY(simulated_node_ids) OR target_entity_id = ANY(simulated_node_ids)
        ) THEN
            RAISE EXCEPTION 'The simulated IndustryChain fixture is referenced by retained Data facts';
        END IF;

        DELETE FROM entity_edges
        WHERE id IN (
            'ERL26400000-0000-4000-8000-000000000004',
            'ERL22000000-0000-4000-8000-000000000003'
        );
        DELETE FROM industry_chain_node_memberships
        WHERE industry_chain_id = simulated_chain_id;
        DELETE FROM chain_node WHERE id = ANY(simulated_node_ids);
        DELETE FROM industry_chain WHERE id = simulated_chain_id;
    ELSIF EXISTS (
        SELECT 1 FROM chain_node WHERE id = ANY(simulated_node_ids)
        UNION ALL
        SELECT 1 FROM industry_chain_node_memberships
        WHERE industry_chain_id = simulated_chain_id OR chain_node_id = ANY(simulated_node_ids)
        UNION ALL
        SELECT 1 FROM entity_edges
        WHERE id IN (
            'ERL26400000-0000-4000-8000-000000000004',
            'ERL22000000-0000-4000-8000-000000000003'
        )
    ) THEN
        RAISE EXCEPTION 'The simulated IndustryChain fixture is partially present';
    END IF;
END;
$$;
-- +goose StatementEnd

-- Snapshot every formal source row so post-delete verification remains exact.
CREATE TEMP TABLE migration_000069_industry_links ON COMMIT DROP AS
SELECT id, from_entity_id AS industry_chain_id, to_entity_id AS industry_id, created_at
FROM entity_edges
WHERE relation_type = 'mapped_to_industry';

CREATE TEMP TABLE migration_000069_concept_links ON COMMIT DROP AS
SELECT id, from_entity_id AS industry_chain_id, to_entity_id AS concept_id, created_at
FROM entity_edges
WHERE relation_type = 'mapped_to_concept';

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM entity_edges relation
        WHERE relation.relation_type IN ('mapped_to_industry', 'mapped_to_concept')
          AND (
              relation.status <> 'active'
              OR relation.updated_at <> relation.created_at
              OR relation.id !~ '^ERL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
              OR NOT EXISTS (SELECT 1 FROM industry_chain value WHERE value.id = relation.from_entity_id)
              OR (
                  relation.relation_type = 'mapped_to_industry'
                  AND NOT EXISTS (SELECT 1 FROM industry value WHERE value.id = relation.to_entity_id)
              )
              OR (
                  relation.relation_type = 'mapped_to_concept'
                  AND NOT EXISTS (SELECT 1 FROM concept value WHERE value.id = relation.to_entity_id)
              )
          )
    ) THEN
        RAISE EXCEPTION 'A legacy IndustryChain mapping cannot be represented by a typed Link';
    END IF;
    IF EXISTS (
        SELECT industry_chain_id, industry_id
        FROM migration_000069_industry_links
        GROUP BY industry_chain_id, industry_id HAVING count(*) > 1
    ) OR EXISTS (
        SELECT industry_chain_id, concept_id
        FROM migration_000069_concept_links
        GROUP BY industry_chain_id, concept_id HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'Legacy IndustryChain mappings contain duplicate endpoint pairs';
    END IF;
    IF EXISTS (
        SELECT 1 FROM industry_chain value
        WHERE NOT EXISTS (
            SELECT 1 FROM migration_000069_industry_links link
            WHERE link.industry_chain_id = value.id
        )
    ) THEN
        RAISE EXCEPTION 'Every remaining IndustryChain must have at least one Industry mapping';
    END IF;
END;
$$;
-- +goose StatementEnd

CREATE TABLE industry_chain_industry_links (
    id VARCHAR(39) PRIMARY KEY,
    industry_chain_id VARCHAR(39) NOT NULL REFERENCES industry_chain(id) ON DELETE RESTRICT,
    industry_id VARCHAR(39) NOT NULL REFERENCES industry(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_industry_chain_industry_link UNIQUE (industry_chain_id, industry_id),
    CONSTRAINT chk_industry_chain_industry_link_identity CHECK (
        id ~ '^ERL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    )
);

CREATE INDEX idx_industry_chain_industry_link_industry
    ON industry_chain_industry_links (industry_id, industry_chain_id);

CREATE TABLE industry_chain_concept_links (
    id VARCHAR(39) PRIMARY KEY,
    industry_chain_id VARCHAR(39) NOT NULL REFERENCES industry_chain(id) ON DELETE RESTRICT,
    concept_id VARCHAR(39) NOT NULL REFERENCES concept(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_industry_chain_concept_link UNIQUE (industry_chain_id, concept_id),
    CONSTRAINT chk_industry_chain_concept_link_identity CHECK (
        id ~ '^ERL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    )
);

CREATE INDEX idx_industry_chain_concept_link_concept
    ON industry_chain_concept_links (concept_id, industry_chain_id);

INSERT INTO industry_chain_industry_links (id, industry_chain_id, industry_id, created_at)
SELECT id, industry_chain_id, industry_id, created_at
FROM migration_000069_industry_links;

INSERT INTO industry_chain_concept_links (id, industry_chain_id, concept_id, created_at)
SELECT id, industry_chain_id, concept_id, created_at
FROM migration_000069_concept_links;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT id, industry_chain_id, industry_id, created_at
        FROM migration_000069_industry_links
        EXCEPT
        SELECT id, industry_chain_id, industry_id, created_at
        FROM industry_chain_industry_links
    ) OR EXISTS (
        SELECT id, industry_chain_id, industry_id, created_at
        FROM industry_chain_industry_links
        EXCEPT
        SELECT id, industry_chain_id, industry_id, created_at
        FROM migration_000069_industry_links
    ) OR EXISTS (
        SELECT id, industry_chain_id, concept_id, created_at
        FROM migration_000069_concept_links
        EXCEPT
        SELECT id, industry_chain_id, concept_id, created_at
        FROM industry_chain_concept_links
    ) OR EXISTS (
        SELECT id, industry_chain_id, concept_id, created_at
        FROM industry_chain_concept_links
        EXCEPT
        SELECT id, industry_chain_id, concept_id, created_at
        FROM migration_000069_concept_links
    ) THEN
        RAISE EXCEPTION 'IndustryChain Link migration did not preserve the complete endpoint sets';
    END IF;
END;
$$;
-- +goose StatementEnd

DELETE FROM entity_edges
WHERE relation_type IN ('mapped_to_industry', 'mapped_to_concept');

ALTER TABLE entity_edges ADD CONSTRAINT chk_entity_edges_reserved_mapping_types CHECK (
    relation_type NOT IN ('mapped_to_industry', 'mapped_to_concept')
);

-- ERL remains the public Research Graph identity family, but each identity may
-- belong to exactly one of the three physical relation stores.
-- +goose StatementBegin
CREATE FUNCTION assert_entity_relation_identity_unique()
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
        PERFORM pg_advisory_xact_lock(hashtextextended('entity-relation:' || lock_id, 0));
    END LOOP;

    IF (TG_TABLE_NAME <> 'entity_edges' AND EXISTS (SELECT 1 FROM entity_edges WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'industry_chain_industry_links' AND EXISTS (
           SELECT 1 FROM industry_chain_industry_links WHERE id = NEW.id
       ))
       OR (TG_TABLE_NAME <> 'industry_chain_concept_links' AND EXISTS (
           SELECT 1 FROM industry_chain_concept_links WHERE id = NEW.id
       )) THEN
        RAISE EXCEPTION 'Entity Relation identity % already belongs to another relation store', NEW.id;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_entity_edge_relation_identity_unique
BEFORE INSERT OR UPDATE OF id ON entity_edges
FOR EACH ROW EXECUTE FUNCTION assert_entity_relation_identity_unique();
CREATE TRIGGER trg_industry_chain_industry_link_identity_unique
BEFORE INSERT OR UPDATE OF id ON industry_chain_industry_links
FOR EACH ROW EXECUTE FUNCTION assert_entity_relation_identity_unique();
CREATE TRIGGER trg_industry_chain_concept_link_identity_unique
BEFORE INSERT OR UPDATE OF id ON industry_chain_concept_links
FOR EACH ROW EXECUTE FUNCTION assert_entity_relation_identity_unique();

-- Preserve the established friendly owner-delete/truncate contract in addition
-- to the new restrictive foreign keys.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION protect_data_object_references()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended(OLD.id, 0));
    IF EXISTS (SELECT 1 FROM entity_edges WHERE from_entity_id = OLD.id OR to_entity_id = OLD.id)
       OR EXISTS (SELECT 1 FROM industry_chain_industry_links WHERE industry_chain_id = OLD.id OR industry_id = OLD.id)
       OR EXISTS (SELECT 1 FROM industry_chain_concept_links WHERE industry_chain_id = OLD.id OR concept_id = OLD.id)
       OR EXISTS (SELECT 1 FROM entity_external_identifiers WHERE entity_id = OLD.id)
       OR EXISTS (
           SELECT 1 FROM entity_redirects
           WHERE source_entity_id = OLD.id OR target_entity_id = OLD.id
       ) THEN
        RAISE EXCEPTION 'Data object % is still referenced and cannot change identity or be deleted', OLD.id;
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION protect_data_object_truncate()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    has_references BOOLEAN;
BEGIN
    EXECUTE format($query$
        SELECT EXISTS (
            WITH references_to_objects(id) AS (
                SELECT from_entity_id FROM entity_edges
                UNION ALL SELECT to_entity_id FROM entity_edges
                UNION ALL SELECT industry_chain_id FROM industry_chain_industry_links
                UNION ALL SELECT industry_id FROM industry_chain_industry_links
                UNION ALL SELECT industry_chain_id FROM industry_chain_concept_links
                UNION ALL SELECT concept_id FROM industry_chain_concept_links
                UNION ALL SELECT entity_id FROM entity_external_identifiers
                UNION ALL SELECT source_entity_id FROM entity_redirects
                UNION ALL SELECT target_entity_id FROM entity_redirects
            )
            SELECT 1
            FROM references_to_objects reference
            JOIN %I owner ON owner.id = reference.id
        )
    $query$, TG_TABLE_NAME) INTO has_references;
    IF has_references THEN
        RAISE EXCEPTION 'Data object table % still owns referenced facts and cannot be truncated', TG_TABLE_NAME;
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM entity_edges
        WHERE relation_type IN ('mapped_to_industry', 'mapped_to_concept')
    ) OR EXISTS (
        SELECT 1 FROM industry_chain value
        WHERE NOT EXISTS (
            SELECT 1 FROM industry_chain_industry_links link
            WHERE link.industry_chain_id = value.id
        )
    ) OR EXISTS (
        SELECT id FROM entity_edges
        INTERSECT SELECT id FROM industry_chain_industry_links
    ) OR EXISTS (
        SELECT id FROM entity_edges
        INTERSECT SELECT id FROM industry_chain_concept_links
    ) OR EXISTS (
        SELECT id FROM industry_chain_industry_links
        INTERSECT SELECT id FROM industry_chain_concept_links
    ) THEN
        RAISE EXCEPTION 'IndustryChain typed Link cutover invariants are not satisfied';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
SELECT 'migration 000069 is forward-only; restore the pre-cutover PostgreSQL snapshot' AS message;
