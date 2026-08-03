-- +goose Up
ALTER TABLE research_theme_import_receipts
    DROP CONSTRAINT chk_research_theme_receipts_contract_version,
    DROP CONSTRAINT chk_research_theme_receipts_aggregate_v2,
    ADD COLUMN publication_mode TEXT NOT NULL DEFAULT 'formal',
    ADD COLUMN reasoning_tree_ids_by_tree_key JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT chk_research_theme_receipts_contract_version
        CHECK (publication_contract_version IN (1, 2, 3)),
    ADD CONSTRAINT chk_research_theme_receipts_publication_mode
        CHECK (publication_mode IN ('formal', 'analyst_snapshot')),
    ADD CONSTRAINT chk_research_theme_receipts_aggregate CHECK (
        publication_contract_version = 1
        OR (publication_contract_version = 2
            AND publication_mode = 'formal'
            AND aggregate_theme_id IS NOT NULL
            AND jsonb_typeof(reasoning_tree_ids_by_industry_chain_entity_id) = 'object'
            AND jsonb_array_length(jsonb_path_query_array(
                reasoning_tree_ids_by_industry_chain_entity_id, '$.keyvalue()')) >= 1)
        OR (publication_contract_version = 3
            AND publication_mode = 'analyst_snapshot'
            AND aggregate_theme_id IS NOT NULL
            AND reasoning_tree_ids_by_industry_chain_entity_id = '{}'::jsonb
            AND jsonb_typeof(reasoning_tree_ids_by_tree_key) = 'object'
            AND jsonb_array_length(jsonb_path_query_array(
                reasoning_tree_ids_by_tree_key, '$.keyvalue()')) >= 1)
    );

ALTER TABLE research_reasoning_tree_import_receipts
    DROP CONSTRAINT chk_research_reasoning_tree_receipts_tree_ids,
    ADD COLUMN publication_contract_version INTEGER NOT NULL DEFAULT 2,
    ADD COLUMN publication_mode TEXT NOT NULL DEFAULT 'formal',
    ADD COLUMN reasoning_tree_ids_by_tree_key JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT chk_research_reasoning_tree_receipts_contract
        CHECK ((publication_contract_version = 2 AND publication_mode = 'formal'
                AND jsonb_typeof(reasoning_tree_ids_by_industry_chain_entity_id) = 'object'
                AND jsonb_array_length(jsonb_path_query_array(
                    reasoning_tree_ids_by_industry_chain_entity_id, '$.keyvalue()')) >= 1)
            OR (publication_contract_version = 3 AND publication_mode = 'analyst_snapshot'
                AND reasoning_tree_ids_by_industry_chain_entity_id = '{}'::jsonb
                AND jsonb_typeof(reasoning_tree_ids_by_tree_key) = 'object'
                AND jsonb_array_length(jsonb_path_query_array(
                    reasoning_tree_ids_by_tree_key, '$.keyvalue()')) >= 1));

ALTER TABLE research_theme_impacts
    DROP CONSTRAINT research_theme_impacts_pkey,
    ALTER COLUMN chain_node_entity_id DROP NOT NULL,
    ADD COLUMN node_key TEXT,
    ADD COLUMN display_name TEXT,
    ADD CONSTRAINT chk_research_theme_impacts_identity CHECK (
        (chain_node_entity_id IS NOT NULL)::integer + (node_key IS NOT NULL)::integer = 1),
    ADD CONSTRAINT chk_research_theme_impacts_snapshot CHECK (
        node_key IS NULL OR (node_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'
            AND display_name IS NOT NULL AND btrim(display_name) <> ''));
CREATE UNIQUE INDEX uq_research_theme_impacts_formal
    ON research_theme_impacts(theme_id, chain_node_entity_id) WHERE chain_node_entity_id IS NOT NULL;
CREATE UNIQUE INDEX uq_research_theme_impacts_snapshot
    ON research_theme_impacts(theme_id, node_key) WHERE node_key IS NOT NULL;

ALTER TABLE research_theme_events ADD COLUMN evidence_ids UUID[] NOT NULL DEFAULT '{}';
ALTER TABLE research_reasoning_tree_events ADD COLUMN evidence_ids UUID[] NOT NULL DEFAULT '{}';

ALTER TABLE research_reasoning_trees
    ALTER COLUMN industry_chain_entity_id DROP NOT NULL,
    ADD COLUMN tree_key TEXT,
    ADD COLUMN display_name TEXT,
    ADD CONSTRAINT chk_research_reasoning_trees_identity CHECK (
        (industry_chain_entity_id IS NOT NULL)::integer + (tree_key IS NOT NULL)::integer = 1),
    ADD CONSTRAINT chk_research_reasoning_trees_snapshot CHECK (
        tree_key IS NULL OR (tree_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'
            AND display_name IS NOT NULL AND btrim(display_name) <> ''));
CREATE UNIQUE INDEX uq_research_reasoning_trees_snapshot
    ON research_reasoning_trees(theme_id, tree_key) WHERE tree_key IS NOT NULL;

ALTER TABLE research_reasoning_tree_nodes
    DROP CONSTRAINT chk_research_reasoning_tree_nodes_incoming,
    ALTER COLUMN chain_node_entity_id DROP NOT NULL,
    ADD COLUMN node_key TEXT,
    ADD COLUMN display_name TEXT,
    ADD CONSTRAINT chk_research_reasoning_tree_nodes_identity CHECK (
        (chain_node_entity_id IS NOT NULL)::integer + (node_key IS NOT NULL)::integer = 1),
    ADD CONSTRAINT chk_research_reasoning_tree_nodes_snapshot CHECK (
        node_key IS NULL OR (node_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'
            AND display_name IS NOT NULL AND btrim(display_name) <> '')),
    ADD CONSTRAINT chk_research_reasoning_tree_nodes_incoming CHECK (
        (position = 1
            AND incoming_industry_chain_graph_edge_id IS NULL
            AND incoming_transmission_title IS NULL
            AND incoming_transmission_mechanism IS NULL
            AND incoming_condition_summary IS NULL)
        OR (position > 1 AND node_key IS NULL
            AND incoming_transmission_title IS NOT NULL AND btrim(incoming_transmission_title) <> ''
            AND incoming_transmission_mechanism IS NOT NULL AND btrim(incoming_transmission_mechanism) <> ''
            AND incoming_condition_summary IS NOT NULL AND btrim(incoming_condition_summary) <> '')
        OR (position > 1 AND node_key IS NOT NULL
            AND incoming_industry_chain_graph_edge_id IS NULL
            AND incoming_transmission_mechanism IS NOT NULL
            AND btrim(incoming_transmission_mechanism) <> ''));
CREATE UNIQUE INDEX uq_research_reasoning_tree_nodes_snapshot
    ON research_reasoning_tree_nodes(reasoning_tree_id, node_key) WHERE node_key IS NOT NULL;

ALTER TABLE research_reasoning_tree_node_signals
    DROP CONSTRAINT research_reasoning_tree_node_signals_pkey,
    DROP CONSTRAINT chk_research_reasoning_tree_signal_source,
    ALTER COLUMN variable_signal_key DROP NOT NULL,
    ALTER COLUMN signal_direction DROP NOT NULL,
    ADD COLUMN signal_key TEXT,
    ADD COLUMN variable_name TEXT,
    ADD CONSTRAINT chk_research_reasoning_tree_signal_source
        CHECK (source_kind IN ('legacy_snapshot','formal_signal','analyst_inference','analyst_snapshot')),
    ADD CONSTRAINT chk_research_reasoning_tree_signal_identity CHECK (
        (source_kind = 'analyst_snapshot'
            AND variable_signal_key IS NULL
            AND signal_key IS NOT NULL)
        OR (source_kind <> 'analyst_snapshot'
            AND variable_signal_key IS NOT NULL
            AND signal_key IS NULL)),
    ADD CONSTRAINT chk_research_reasoning_tree_signal_formal_direction CHECK (
        source_kind = 'analyst_snapshot' OR signal_direction IS NOT NULL),
    ADD CONSTRAINT chk_research_reasoning_tree_signal_snapshot CHECK (
        source_kind <> 'analyst_snapshot' OR (
            signal_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'
            AND variable_signal_key IS NULL
            AND variable_signal_id IS NULL AND semantic_submission_id IS NULL
            AND evidence_id IS NULL AND evidence_hash IS NULL
            AND upstream_variable_signal_id IS NULL
            AND upstream_direct_impact_assertion_id IS NULL
            AND entity_relation_id IS NULL AND industry_chain_graph_edge_id IS NULL));
CREATE UNIQUE INDEX uq_research_reasoning_tree_signals_formal
    ON research_reasoning_tree_node_signals(reasoning_tree_node_id, variable_signal_key)
    WHERE variable_signal_key IS NOT NULL;
CREATE UNIQUE INDEX uq_research_reasoning_tree_signals_snapshot
    ON research_reasoning_tree_node_signals(reasoning_tree_node_id, signal_key)
    WHERE signal_key IS NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000041 is forward-only; restore the prior revision with a reviewed backup';
END;
$$;
-- +goose StatementEnd
