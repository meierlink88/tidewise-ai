-- +goose Up
-- Event Semantic retirement is destructive and forward-only. Operators must stop
-- Data writes and take a reviewed database snapshot before applying this migration.

DROP TRIGGER trg_research_reasoning_tree_node_signals_immutable ON research_reasoning_tree_node_signals;
DROP TRIGGER trg_research_reasoning_tree_nodes_immutable ON research_reasoning_tree_nodes;
DROP TRIGGER trg_research_reasoning_tree_events_immutable ON research_reasoning_tree_events;
DROP TRIGGER trg_research_reasoning_trees_immutable ON research_reasoning_trees;
DROP TRIGGER trg_research_reasoning_tree_receipts_immutable ON research_reasoning_tree_import_receipts;
DROP TRIGGER trg_research_theme_events_immutable ON research_theme_events;
DROP TRIGGER trg_research_theme_impacts_immutable ON research_theme_impacts;
DROP TRIGGER trg_research_themes_immutable ON research_themes;
DROP TRIGGER trg_research_theme_receipts_immutable ON research_theme_import_receipts;

CREATE TEMPORARY TABLE retired_research_theme_ids (id VARCHAR(40) PRIMARY KEY) ON COMMIT DROP;
INSERT INTO retired_research_theme_ids (id)
SELECT theme.id
FROM research_themes AS theme
JOIN research_theme_import_receipts AS receipt ON receipt.id = theme.import_receipt_id
WHERE receipt.publication_contract_version <> 3
   OR receipt.publication_mode <> 'analyst_snapshot';

CREATE TEMPORARY TABLE retired_research_tree_ids (id VARCHAR(40) PRIMARY KEY) ON COMMIT DROP;
INSERT INTO retired_research_tree_ids (id)
SELECT tree.id
FROM research_reasoning_trees AS tree
JOIN research_reasoning_tree_import_receipts AS receipt ON receipt.id = tree.import_receipt_id
WHERE receipt.publication_contract_version <> 3
   OR receipt.publication_mode <> 'analyst_snapshot'
   OR tree.theme_id IN (SELECT id FROM retired_research_theme_ids);

DELETE FROM research_reasoning_tree_node_signals
WHERE reasoning_tree_node_id IN (
    SELECT node.id FROM research_reasoning_tree_nodes AS node
    WHERE node.reasoning_tree_id IN (SELECT id FROM retired_research_tree_ids)
);
DELETE FROM research_reasoning_tree_nodes
WHERE reasoning_tree_id IN (SELECT id FROM retired_research_tree_ids);
DELETE FROM research_reasoning_tree_events
WHERE reasoning_tree_id IN (SELECT id FROM retired_research_tree_ids);
DELETE FROM research_reasoning_trees
WHERE id IN (SELECT id FROM retired_research_tree_ids);
DELETE FROM research_reasoning_tree_import_receipts
WHERE publication_contract_version <> 3
   OR publication_mode <> 'analyst_snapshot'
   OR theme_id IN (SELECT id FROM retired_research_theme_ids);

DELETE FROM research_theme_events WHERE theme_id IN (SELECT id FROM retired_research_theme_ids);
DELETE FROM research_theme_impacts WHERE theme_id IN (SELECT id FROM retired_research_theme_ids);
DELETE FROM research_themes WHERE id IN (SELECT id FROM retired_research_theme_ids);
DELETE FROM research_theme_import_receipts
WHERE publication_contract_version <> 3 OR publication_mode <> 'analyst_snapshot';

ALTER TABLE research_theme_import_receipts
    DROP COLUMN theme_ids_by_key CASCADE,
    DROP COLUMN write_counts CASCADE,
    DROP COLUMN reasoning_tree_ids_by_industry_chain_id CASCADE,
    ALTER COLUMN publication_contract_version DROP DEFAULT,
    ALTER COLUMN publication_mode DROP DEFAULT,
    ALTER COLUMN aggregate_theme_id SET NOT NULL,
    ADD CONSTRAINT chk_research_theme_receipts_snapshot_contract CHECK (
        publication_contract_version = 3
        AND publication_mode = 'analyst_snapshot'
        AND jsonb_typeof(reasoning_tree_ids_by_tree_key) = 'object'
        AND jsonb_array_length(jsonb_path_query_array(
            reasoning_tree_ids_by_tree_key, '$.keyvalue()')) >= 1
    );

ALTER TABLE research_reasoning_tree_import_receipts
    DROP COLUMN reasoning_tree_ids_by_industry_chain_id CASCADE,
    ALTER COLUMN publication_contract_version DROP DEFAULT,
    ALTER COLUMN publication_mode DROP DEFAULT,
    ADD CONSTRAINT chk_research_reasoning_tree_receipts_snapshot_contract CHECK (
        publication_contract_version = 3
        AND publication_mode = 'analyst_snapshot'
        AND jsonb_typeof(reasoning_tree_ids_by_tree_key) = 'object'
        AND jsonb_array_length(jsonb_path_query_array(
            reasoning_tree_ids_by_tree_key, '$.keyvalue()')) >= 1
    );

ALTER TABLE research_theme_impacts
    DROP COLUMN chain_node_id CASCADE,
    ALTER COLUMN node_key SET NOT NULL,
    ALTER COLUMN display_name SET NOT NULL,
    ADD CONSTRAINT research_theme_impacts_pkey PRIMARY KEY (theme_id, node_key),
    ADD CONSTRAINT chk_research_theme_impacts_snapshot_key
        CHECK (node_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'),
    ADD CONSTRAINT chk_research_theme_impacts_snapshot_name
        CHECK (btrim(display_name) <> '');

ALTER TABLE research_reasoning_trees
    DROP COLUMN industry_chain_id CASCADE,
    ALTER COLUMN tree_key SET NOT NULL,
    ALTER COLUMN display_name SET NOT NULL,
    ADD CONSTRAINT chk_research_reasoning_trees_snapshot_key
        CHECK (tree_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'),
    ADD CONSTRAINT chk_research_reasoning_trees_snapshot_name
        CHECK (btrim(display_name) <> '');

ALTER TABLE research_reasoning_tree_nodes
    DROP COLUMN chain_node_id CASCADE,
    DROP COLUMN incoming_industry_chain_graph_edge_id CASCADE,
    DROP COLUMN incoming_source_kind CASCADE,
    DROP COLUMN direct_impact_assertion_id CASCADE,
    DROP COLUMN direct_impact_semantic_submission_id CASCADE,
    DROP COLUMN direct_impact_evidence_id CASCADE,
    DROP COLUMN direct_impact_evidence_hash CASCADE,
    DROP COLUMN direct_impact_affected_variable_key CASCADE,
    DROP COLUMN direct_impact_affected_direction CASCADE,
    DROP COLUMN inference_upstream_variable_signal_id CASCADE,
    DROP COLUMN inference_upstream_direct_impact_assertion_id CASCADE,
    DROP COLUMN inference_entity_relation_id CASCADE,
    ALTER COLUMN node_key SET NOT NULL,
    ALTER COLUMN display_name SET NOT NULL,
    ADD CONSTRAINT chk_research_reasoning_tree_nodes_snapshot_key
        CHECK (node_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'),
    ADD CONSTRAINT chk_research_reasoning_tree_nodes_snapshot_name
        CHECK (btrim(display_name) <> ''),
    ADD CONSTRAINT chk_research_reasoning_tree_nodes_snapshot_incoming CHECK (
        (position = 1
            AND incoming_transmission_title IS NULL
            AND incoming_transmission_mechanism IS NULL
            AND incoming_condition_summary IS NULL)
        OR (position > 1
            AND incoming_transmission_mechanism IS NOT NULL
            AND btrim(incoming_transmission_mechanism) <> '')
    );

ALTER TABLE research_reasoning_tree_node_signals
    DROP COLUMN variable_signal_key CASCADE,
    DROP COLUMN source_kind CASCADE,
    DROP COLUMN variable_signal_id CASCADE,
    DROP COLUMN semantic_submission_id CASCADE,
    DROP COLUMN evidence_id CASCADE,
    DROP COLUMN evidence_hash CASCADE,
    DROP COLUMN upstream_variable_signal_id CASCADE,
    DROP COLUMN upstream_direct_impact_assertion_id CASCADE,
    DROP COLUMN entity_relation_id CASCADE,
    DROP COLUMN industry_chain_graph_edge_id CASCADE,
    ALTER COLUMN signal_key SET NOT NULL,
    ADD CONSTRAINT research_reasoning_tree_node_signals_pkey
        PRIMARY KEY (reasoning_tree_node_id, signal_key),
    ADD CONSTRAINT chk_research_reasoning_tree_node_signals_snapshot_key
        CHECK (signal_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$');

DROP TABLE
    event_semantic_review_snapshots,
    event_semantic_candidate_snapshots,
    event_semantic_resolution_bindings,
    variable_signal_measurements,
    direct_impact_assertions,
    variable_signals,
    event_entity_links,
    event_semantic_submissions,
    event_semantic_context_leases,
    event_semantic_acceptance_policies,
    direct_transmission_rules,
    variable_definition_entity_types,
    variable_definitions,
    product_profiles;

-- Independent Data objects still participate in polymorphic EntityRelation,
-- external-identifier, and redirect references. Keep their delete/update and
-- truncate guards, but remove the retired Event Semantic reference sources.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION protect_data_object_references()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended(OLD.id, 0));
    IF EXISTS (SELECT 1 FROM entity_edges WHERE from_entity_id = OLD.id OR to_entity_id = OLD.id)
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

DROP FUNCTION event_semantic_measurement_evidence_ids_compat();

CREATE TRIGGER trg_research_theme_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_theme_import_receipts
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_themes_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_themes
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_theme_impacts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_theme_impacts
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_theme_events_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_theme_events
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_reasoning_tree_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_reasoning_tree_import_receipts
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_reasoning_trees_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_reasoning_trees
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_reasoning_tree_events_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_reasoning_tree_events
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_reasoning_tree_nodes_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_reasoning_tree_nodes
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();
CREATE TRIGGER trg_research_reasoning_tree_node_signals_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_reasoning_tree_node_signals
FOR EACH STATEMENT EXECUTE FUNCTION prevent_research_publication_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000059 is destructive and forward-only; restore the reviewed pre-migration snapshot';
END;
$$;
-- +goose StatementEnd
