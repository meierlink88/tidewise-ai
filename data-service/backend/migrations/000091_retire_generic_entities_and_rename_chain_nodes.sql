-- +goose Up
-- Coordinated cutover: stop Data traffic and back up the database before applying.
ALTER TABLE chain_node RENAME TO industry_chain_node;
ALTER TABLE industry_chain_graph_edges RENAME TO industry_chain_node_graph;

DROP TABLE policy_body_profiles, person_profiles, instrument_profiles, index_profiles,
    security_profiles, theme_profiles, commodity_profiles, market_profiles;
DROP TABLE entity_edges;
DROP TABLE entity_nodes;
DROP FUNCTION assert_entity_profile_type();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION assert_data_object_identity_unique()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
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

    IF (TG_TABLE_NAME <> 'industry' AND EXISTS (SELECT 1 FROM industry WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'concept' AND EXISTS (SELECT 1 FROM concept WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'industry_chain_node' AND EXISTS (SELECT 1 FROM industry_chain_node WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'industry_chain' AND EXISTS (SELECT 1 FROM industry_chain WHERE id = NEW.id))
       OR (TG_TABLE_NAME <> 'company' AND EXISTS (SELECT 1 FROM company WHERE id = NEW.id)) THEN
        RAISE EXCEPTION 'Data object identity % already belongs to another object', NEW.id;
    END IF;
    RETURN NEW;
END;
$function$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION assert_entity_relation_identity_unique()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
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

    IF (TG_TABLE_NAME <> 'industry_chain_industry_links' AND EXISTS (
           SELECT 1 FROM industry_chain_industry_links WHERE id = NEW.id
       ))
       OR (TG_TABLE_NAME <> 'industry_chain_concept_links' AND EXISTS (
           SELECT 1 FROM industry_chain_concept_links WHERE id = NEW.id
       )) THEN
        RAISE EXCEPTION 'Entity Relation identity % already belongs to another relation store', NEW.id;
    END IF;
    RETURN NEW;
END;
$function$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION data_object_type(reference_id text)
 RETURNS text
 LANGUAGE sql
 STABLE
AS $function$
    SELECT CASE WHEN count(*) = 1 THEN min(object_type) END
    FROM (
        SELECT 'industry'::text object_type FROM industry WHERE id = reference_id
        UNION ALL SELECT 'concept' FROM concept WHERE id = reference_id
        UNION ALL SELECT 'chain_node' FROM industry_chain_node WHERE id = reference_id
        UNION ALL SELECT 'industry_chain' FROM industry_chain WHERE id = reference_id
    ) value
$function$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION protect_data_object_references()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended(OLD.id, 0));
    IF EXISTS (SELECT 1 FROM industry_chain_industry_links WHERE industry_chain_id = OLD.id OR industry_id = OLD.id)
       OR EXISTS (SELECT 1 FROM industry_chain_concept_links WHERE industry_chain_id = OLD.id OR concept_id = OLD.id) THEN
        RAISE EXCEPTION 'Data object % is still referenced and cannot change identity or be deleted', OLD.id;
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$function$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION protect_data_object_truncate()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    has_references BOOLEAN;
BEGIN
    EXECUTE format($query$
        SELECT EXISTS (
            WITH references_to_objects(id) AS (
                SELECT industry_chain_id FROM industry_chain_industry_links
                UNION ALL SELECT industry_id FROM industry_chain_industry_links
                UNION ALL SELECT industry_chain_id FROM industry_chain_concept_links
                UNION ALL SELECT concept_id FROM industry_chain_concept_links
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
$function$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION reject_industry_chain_graph_cycle()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    PERFORM 1
    FROM industry_chain_node_memberships membership
    WHERE membership.industry_chain_id = NEW.industry_chain_id
      AND membership.chain_node_id IN (NEW.from_chain_node_id, NEW.to_chain_node_id)
    ORDER BY membership.chain_node_id
    FOR SHARE;

    PERFORM pg_advisory_xact_lock(hashtext('industry_chain_topology:' || NEW.industry_chain_id));

    IF EXISTS (
        WITH RECURSIVE reachable(node_id) AS (
            SELECT NEW.to_chain_node_id
            UNION
            SELECT edge.to_chain_node_id
            FROM industry_chain_node_graph edge
            JOIN reachable current_path ON edge.from_chain_node_id = current_path.node_id
            WHERE edge.industry_chain_id = NEW.industry_chain_id
              AND edge.id <> NEW.id
        )
        SELECT 1 FROM reachable WHERE node_id = NEW.from_chain_node_id
    ) THEN
        RAISE EXCEPTION 'industry chain topology must remain acyclic';
    END IF;
    RETURN NEW;
END;
$function$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 RAISE EXCEPTION 'migration 000091 is forward-only; restore the pre-migration database backup and matching application';
END $$;
-- +goose StatementEnd
