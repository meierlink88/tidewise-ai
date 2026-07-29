-- +goose Up
ALTER TABLE research_theme_import_receipts
    ADD COLUMN publication_contract_version INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN aggregate_theme_id UUID,
    ADD COLUMN reasoning_tree_ids_by_industry_chain_entity_id JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN aggregate_write_counts JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT chk_research_theme_receipts_contract_version
        CHECK (publication_contract_version IN (1, 2)),
    ADD CONSTRAINT chk_research_theme_receipts_aggregate_v2 CHECK (
        publication_contract_version = 1
        OR (
            aggregate_theme_id IS NOT NULL
            AND jsonb_typeof(reasoning_tree_ids_by_industry_chain_entity_id) = 'object'
            AND jsonb_array_length(jsonb_path_query_array(
                reasoning_tree_ids_by_industry_chain_entity_id, '$.keyvalue()')) >= 1
            AND jsonb_typeof(aggregate_write_counts) = 'object'
            AND aggregate_write_counts ?& ARRAY[
                'themes','impacts','theme_event_associations','reasoning_trees',
                'nodes','tree_event_associations','signal_associations','receipts'
            ]::text[]
            AND (aggregate_write_counts ->> 'themes')::integer = 1
            AND (aggregate_write_counts ->> 'reasoning_trees')::integer >= 1
            AND (aggregate_write_counts ->> 'receipts')::integer = 2
        )
    );

ALTER TABLE research_reasoning_tree_nodes
    ADD COLUMN incoming_source_kind VARCHAR(32),
    ADD COLUMN direct_impact_assertion_id UUID REFERENCES direct_impact_assertions(id) ON DELETE RESTRICT,
    ADD COLUMN direct_impact_semantic_submission_id UUID REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT,
    ADD COLUMN direct_impact_evidence_id UUID REFERENCES event_sources(id) ON DELETE RESTRICT,
    ADD COLUMN direct_impact_evidence_hash CHAR(64),
    ADD COLUMN direct_impact_affected_variable_key VARCHAR(128),
    ADD COLUMN direct_impact_affected_direction VARCHAR(32),
    ADD COLUMN inference_upstream_variable_signal_id UUID REFERENCES variable_signals(id) ON DELETE RESTRICT,
    ADD COLUMN inference_upstream_direct_impact_assertion_id UUID REFERENCES direct_impact_assertions(id) ON DELETE RESTRICT,
    ADD COLUMN inference_entity_relation_id UUID REFERENCES entity_edges(id) ON DELETE RESTRICT,
    ADD CONSTRAINT chk_research_reasoning_tree_node_incoming_source
        CHECK (incoming_source_kind IS NULL OR incoming_source_kind IN (
            'formal_direct_impact', 'analyst_inference'
        )),
    ADD CONSTRAINT chk_research_reasoning_tree_node_formal_impact CHECK (
        incoming_source_kind IS DISTINCT FROM 'formal_direct_impact'
        OR (
            direct_impact_assertion_id IS NOT NULL
            AND direct_impact_semantic_submission_id IS NOT NULL
            AND direct_impact_evidence_id IS NOT NULL
            AND direct_impact_evidence_hash ~ '^[0-9a-f]{64}$'
            AND btrim(direct_impact_affected_variable_key) <> ''
            AND direct_impact_affected_direction IN ('increase','decrease','unchanged','mixed','uncertain')
            AND inference_upstream_variable_signal_id IS NULL
            AND inference_upstream_direct_impact_assertion_id IS NULL
            AND inference_entity_relation_id IS NULL
        )
    ),
    ADD CONSTRAINT chk_research_reasoning_tree_node_inference CHECK (
        incoming_source_kind IS DISTINCT FROM 'analyst_inference'
        OR (
            direct_impact_assertion_id IS NULL
            AND direct_impact_semantic_submission_id IS NULL
            AND direct_impact_evidence_id IS NULL
            AND direct_impact_evidence_hash IS NULL
            AND direct_impact_affected_variable_key IS NULL
            AND direct_impact_affected_direction IS NULL
            AND (
                (inference_upstream_variable_signal_id IS NOT NULL)::integer
                + (inference_upstream_direct_impact_assertion_id IS NOT NULL)::integer
            ) = 1
            AND (
                inference_entity_relation_id IS NOT NULL
                OR incoming_industry_chain_graph_edge_id IS NOT NULL
            )
        )
    );

ALTER TABLE research_reasoning_tree_node_signals
    ADD COLUMN source_kind VARCHAR(32) NOT NULL DEFAULT 'legacy_snapshot',
    ADD COLUMN variable_signal_id UUID REFERENCES variable_signals(id) ON DELETE RESTRICT,
    ADD COLUMN semantic_submission_id UUID REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT,
    ADD COLUMN evidence_id UUID REFERENCES event_sources(id) ON DELETE RESTRICT,
    ADD COLUMN evidence_hash CHAR(64),
    ADD COLUMN upstream_variable_signal_id UUID REFERENCES variable_signals(id) ON DELETE RESTRICT,
    ADD COLUMN upstream_direct_impact_assertion_id UUID REFERENCES direct_impact_assertions(id) ON DELETE RESTRICT,
    ADD COLUMN entity_relation_id UUID REFERENCES entity_edges(id) ON DELETE RESTRICT,
    ADD COLUMN industry_chain_graph_edge_id UUID REFERENCES industry_chain_graph_edges(id) ON DELETE RESTRICT,
    ADD CONSTRAINT chk_research_reasoning_tree_signal_source
        CHECK (source_kind IN ('legacy_snapshot','formal_signal','analyst_inference')),
    ADD CONSTRAINT chk_research_reasoning_tree_signal_formal CHECK (
        source_kind <> 'formal_signal'
        OR (
            variable_signal_id IS NOT NULL
            AND semantic_submission_id IS NOT NULL
            AND evidence_id IS NOT NULL
            AND evidence_hash ~ '^[0-9a-f]{64}$'
            AND upstream_variable_signal_id IS NULL
            AND upstream_direct_impact_assertion_id IS NULL
            AND entity_relation_id IS NULL
            AND industry_chain_graph_edge_id IS NULL
        )
    ),
    ADD CONSTRAINT chk_research_reasoning_tree_signal_inference CHECK (
        source_kind <> 'analyst_inference'
        OR (
            variable_signal_id IS NULL
            AND semantic_submission_id IS NULL
            AND evidence_id IS NULL
            AND evidence_hash IS NULL
            AND (
                (upstream_variable_signal_id IS NOT NULL)::integer
                + (upstream_direct_impact_assertion_id IS NOT NULL)::integer
            ) = 1
            AND (
                (entity_relation_id IS NOT NULL)::integer
                + (industry_chain_graph_edge_id IS NOT NULL)::integer
            ) = 1
        )
    );

CREATE INDEX idx_research_reasoning_tree_node_signals_formal_signal
    ON research_reasoning_tree_node_signals(variable_signal_id)
    WHERE variable_signal_id IS NOT NULL;
CREATE INDEX idx_research_reasoning_tree_nodes_direct_impact
    ON research_reasoning_tree_nodes(direct_impact_assertion_id)
    WHERE direct_impact_assertion_id IS NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000033 is forward-only; restore the prior revision with a fresh database or reviewed backup';
END;
$$;
-- +goose StatementEnd
