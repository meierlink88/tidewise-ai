-- +goose Up
DROP TRIGGER trg_protect_profiled_entity_identity ON entity_nodes;
DROP FUNCTION protect_profiled_entity_identity();

DROP TABLE entity_external_identifiers;
DROP TABLE entity_redirects;
DROP FUNCTION validate_entity_redirect();

-- Keep the established friendly owner-delete contract for the remaining
-- generic Entity Relations and typed IndustryChain Links.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION protect_data_object_references()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended(OLD.id, 0));
    IF EXISTS (SELECT 1 FROM entity_edges WHERE from_entity_id = OLD.id OR to_entity_id = OLD.id)
       OR EXISTS (SELECT 1 FROM industry_chain_industry_links WHERE industry_chain_id = OLD.id OR industry_id = OLD.id)
       OR EXISTS (SELECT 1 FROM industry_chain_concept_links WHERE industry_chain_id = OLD.id OR concept_id = OLD.id) THEN
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

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000071 is forward-only; restore the pre-migration PostgreSQL snapshot';
END;
$$;
-- +goose StatementEnd
