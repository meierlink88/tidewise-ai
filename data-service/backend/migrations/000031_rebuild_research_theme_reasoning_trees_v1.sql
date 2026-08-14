-- +goose Up
DROP TABLE IF EXISTS research_anchor_events;
DROP TABLE IF EXISTS research_anchor_chain_nodes;
DROP TABLE IF EXISTS research_anchors;
DROP TABLE IF EXISTS research_anchor_import_receipts;
DROP TABLE IF EXISTS research_theme_indices;
DROP TABLE IF EXISTS research_theme_events;
DROP TABLE IF EXISTS research_theme_chain_nodes;
DROP TABLE IF EXISTS research_themes;
DROP TABLE IF EXISTS research_theme_import_receipts;

DROP FUNCTION IF EXISTS prevent_research_anchor_import_receipt_mutation();
DROP FUNCTION IF EXISTS prevent_research_theme_import_receipt_mutation();

CREATE TABLE research_theme_import_receipts (
    id UUID PRIMARY KEY,
    analysis_batch_id TEXT NOT NULL UNIQUE,
    publisher_subject TEXT NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    theme_ids_by_key JSONB NOT NULL,
    write_counts JSONB NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_research_theme_import_receipts_id_batch UNIQUE (id, analysis_batch_id),
    CONSTRAINT chk_research_theme_import_receipts_batch
        CHECK (char_length(analysis_batch_id) BETWEEN 1 AND 200 AND btrim(analysis_batch_id) <> ''),
    CONSTRAINT chk_research_theme_import_receipts_publisher
        CHECK (char_length(publisher_subject) BETWEEN 1 AND 200 AND btrim(publisher_subject) <> ''),
    CONSTRAINT chk_research_theme_import_receipts_payload_hash
        CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_research_theme_import_receipts_theme_ids
        CHECK (jsonb_typeof(theme_ids_by_key) = 'object'
            AND jsonb_array_length(jsonb_path_query_array(theme_ids_by_key, '$.keyvalue()')) >= 1),
    CONSTRAINT chk_research_theme_import_receipts_counts
        CHECK (jsonb_typeof(write_counts) = 'object'
            AND write_counts ?& ARRAY['themes','impacts','event_associations','receipts']::text[]
            AND jsonb_array_length(jsonb_path_query_array(write_counts, '$.keyvalue()')) = 4
            AND (write_counts ->> 'themes')::integer >= 1
            AND (write_counts ->> 'impacts')::integer >= (write_counts ->> 'themes')::integer
            AND (write_counts ->> 'event_associations')::integer >= 0
            AND (write_counts ->> 'receipts')::integer = 1),
    CONSTRAINT chk_research_theme_import_receipts_times CHECK (imported_at >= published_at)
);

CREATE TABLE research_themes (
    id UUID PRIMARY KEY,
    theme_key TEXT NOT NULL,
    analysis_batch_id TEXT NOT NULL,
    import_receipt_id UUID NOT NULL,
    title TEXT NOT NULL,
    one_line_conclusion TEXT NOT NULL,
    conclusion_direction TEXT NOT NULL,
    impact_strength TEXT NOT NULL,
    attention_level TEXT,
    conclusion_status TEXT,
    transmission_stage TEXT NOT NULL,
    investment_guidance_action TEXT NOT NULL,
    investment_guidance_summary TEXT NOT NULL,
    time_horizon_category TEXT NOT NULL,
    time_horizon_summary TEXT,
    transmission_summary TEXT,
    checkpoint_summary TEXT,
    risk_summary TEXT,
    analysis_as_of TIMESTAMPTZ NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_research_themes_batch_theme_key UNIQUE (analysis_batch_id, theme_key),
    CONSTRAINT fk_research_themes_receipt_batch
        FOREIGN KEY (import_receipt_id, analysis_batch_id)
        REFERENCES research_theme_import_receipts(id, analysis_batch_id),
    CONSTRAINT chk_research_themes_key CHECK (theme_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'),
    CONSTRAINT chk_research_themes_title CHECK (btrim(title) <> ''),
    CONSTRAINT chk_research_themes_conclusion CHECK (btrim(one_line_conclusion) <> ''),
    CONSTRAINT chk_research_themes_conclusion_direction
        CHECK (conclusion_direction IN ('positive','negative','mixed','neutral','uncertain')),
    CONSTRAINT chk_research_themes_impact_strength
        CHECK (impact_strength IN ('strong','medium','weak','unknown')),
    CONSTRAINT chk_research_themes_attention_level
        CHECK (attention_level IS NULL OR attention_level IN ('high','medium','low')),
    CONSTRAINT chk_research_themes_conclusion_status
        CHECK (conclusion_status IS NULL OR conclusion_status IN ('supported','partial','conflicted')),
    CONSTRAINT chk_research_themes_stage
        CHECK (transmission_stage IN ('identification','validation','diffusion','dampening')),
    CONSTRAINT chk_research_themes_guidance_action
        CHECK (investment_guidance_action IN ('focus','avoid','observe','differentiate')),
    CONSTRAINT chk_research_themes_guidance_summary CHECK (btrim(investment_guidance_summary) <> ''),
    CONSTRAINT chk_research_themes_time_horizon
        CHECK (time_horizon_category IN ('short_term','medium_term','long_term','custom')),
    CONSTRAINT chk_research_themes_window CHECK (window_end > window_start)
);

CREATE TABLE research_theme_impacts (
    theme_id UUID NOT NULL REFERENCES research_themes(id) ON DELETE CASCADE,
    chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
    relation_role TEXT NOT NULL,
    impact_direction TEXT NOT NULL,
    impact_summary TEXT,
    display_order INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (theme_id, chain_node_entity_id),
    CONSTRAINT uq_research_theme_impacts_order UNIQUE (theme_id, display_order),
    CONSTRAINT chk_research_theme_impacts_order CHECK (display_order >= 1),
    CONSTRAINT chk_research_theme_impacts_role
        CHECK (relation_role IN ('driver','beneficiary','constraint','exposure')),
    CONSTRAINT chk_research_theme_impacts_direction
        CHECK (impact_direction IN ('positive','negative','mixed','neutral','uncertain'))
);

CREATE TABLE research_theme_events (
    theme_id UUID NOT NULL REFERENCES research_themes(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id),
    evidence_role TEXT NOT NULL,
    supported_claim TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (theme_id, event_id),
    CONSTRAINT chk_research_theme_events_role
        CHECK (evidence_role IN ('driver','supporting','contradicting','context'))
);

CREATE TABLE research_reasoning_tree_import_receipts (
    id UUID PRIMARY KEY,
    theme_id UUID NOT NULL UNIQUE REFERENCES research_themes(id),
    publisher_subject TEXT NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    reasoning_tree_ids_by_industry_chain_entity_id JSONB NOT NULL,
    write_counts JSONB NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_research_reasoning_tree_receipts_id_theme UNIQUE (id, theme_id),
    CONSTRAINT chk_research_reasoning_tree_receipts_publisher
        CHECK (char_length(publisher_subject) BETWEEN 1 AND 200 AND btrim(publisher_subject) <> ''),
    CONSTRAINT chk_research_reasoning_tree_receipts_payload_hash
        CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_research_reasoning_tree_receipts_tree_ids
        CHECK (jsonb_typeof(reasoning_tree_ids_by_industry_chain_entity_id) = 'object'
            AND jsonb_array_length(jsonb_path_query_array(
                reasoning_tree_ids_by_industry_chain_entity_id, '$.keyvalue()')) >= 1),
    CONSTRAINT chk_research_reasoning_tree_receipts_counts
        CHECK (jsonb_typeof(write_counts) = 'object'
            AND write_counts ?& ARRAY[
                'reasoning_trees','nodes','event_associations','signal_associations','receipts'
            ]::text[]
            AND jsonb_array_length(jsonb_path_query_array(write_counts, '$.keyvalue()')) = 5
            AND (write_counts ->> 'reasoning_trees')::integer >= 1
            AND (write_counts ->> 'nodes')::integer >= (write_counts ->> 'reasoning_trees')::integer
            AND (write_counts ->> 'event_associations')::integer >= 0
            AND (write_counts ->> 'signal_associations')::integer >= (write_counts ->> 'nodes')::integer
            AND (write_counts ->> 'receipts')::integer = 1),
    CONSTRAINT chk_research_reasoning_tree_receipts_times CHECK (imported_at >= published_at)
);

CREATE TABLE research_reasoning_trees (
    id UUID PRIMARY KEY,
    theme_id UUID NOT NULL REFERENCES research_themes(id) ON DELETE CASCADE,
    import_receipt_id UUID NOT NULL,
    industry_chain_entity_id UUID NOT NULL REFERENCES industry_chain_definitions(entity_id),
    title TEXT NOT NULL,
    display_order INTEGER NOT NULL,
    one_line_conclusion TEXT NOT NULL,
    fact_summary TEXT,
    transmission_summary TEXT,
    impact_direction TEXT NOT NULL,
    impact_strength TEXT NOT NULL,
    impact_summary TEXT,
    conclusion_boundary_summary TEXT,
    support_summary TEXT,
    counter_summary TEXT,
    invalidation_conditions JSONB NOT NULL DEFAULT '[]'::jsonb,
    checkpoints JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_research_reasoning_trees_theme_chain UNIQUE (theme_id, industry_chain_entity_id),
    CONSTRAINT uq_research_reasoning_trees_theme_order UNIQUE (theme_id, display_order),
    CONSTRAINT fk_research_reasoning_trees_receipt_theme
        FOREIGN KEY (import_receipt_id, theme_id)
        REFERENCES research_reasoning_tree_import_receipts(id, theme_id),
    CONSTRAINT chk_research_reasoning_trees_order CHECK (display_order >= 1),
    CONSTRAINT chk_research_reasoning_trees_title CHECK (btrim(title) <> ''),
    CONSTRAINT chk_research_reasoning_trees_conclusion CHECK (btrim(one_line_conclusion) <> ''),
    CONSTRAINT chk_research_reasoning_trees_direction
        CHECK (impact_direction IN ('positive','negative','mixed','neutral','uncertain')),
    CONSTRAINT chk_research_reasoning_trees_strength
        CHECK (impact_strength IN ('strong','medium','weak','unknown')),
    CONSTRAINT chk_research_reasoning_trees_invalidation
        CHECK (jsonb_typeof(invalidation_conditions) = 'array'),
    CONSTRAINT chk_research_reasoning_trees_checkpoints CHECK (jsonb_typeof(checkpoints) = 'array')
);

CREATE TABLE research_reasoning_tree_events (
    reasoning_tree_id UUID NOT NULL REFERENCES research_reasoning_trees(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id),
    evidence_role TEXT NOT NULL,
    display_order INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (reasoning_tree_id, event_id),
    CONSTRAINT uq_research_reasoning_tree_events_order UNIQUE (reasoning_tree_id, display_order),
    CONSTRAINT chk_research_reasoning_tree_events_order CHECK (display_order >= 1),
    CONSTRAINT chk_research_reasoning_tree_events_role
        CHECK (evidence_role IN ('driver','supporting','contradicting','context'))
);

CREATE TABLE research_reasoning_tree_nodes (
    id UUID PRIMARY KEY,
    reasoning_tree_id UUID NOT NULL REFERENCES research_reasoning_trees(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
    state_summary TEXT,
    impact_direction TEXT NOT NULL,
    impact_strength TEXT NOT NULL,
    impact_summary TEXT,
    reasoning_basis_summary TEXT,
    evidence_gap_summary TEXT,
    incoming_industry_chain_graph_edge_id UUID REFERENCES industry_chain_graph_edges(id),
    incoming_transmission_title TEXT,
    incoming_transmission_mechanism TEXT,
    incoming_condition_summary TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_research_reasoning_tree_nodes_position UNIQUE (reasoning_tree_id, position),
    CONSTRAINT uq_research_reasoning_tree_nodes_entity UNIQUE (reasoning_tree_id, chain_node_entity_id),
    CONSTRAINT chk_research_reasoning_tree_nodes_position CHECK (position >= 1),
    CONSTRAINT chk_research_reasoning_tree_nodes_direction
        CHECK (impact_direction IN ('positive','negative','mixed','neutral','uncertain')),
    CONSTRAINT chk_research_reasoning_tree_nodes_strength
        CHECK (impact_strength IN ('strong','medium','weak','unknown')),
    CONSTRAINT chk_research_reasoning_tree_nodes_incoming CHECK (
        (position = 1
            AND incoming_industry_chain_graph_edge_id IS NULL
            AND incoming_transmission_title IS NULL
            AND incoming_transmission_mechanism IS NULL
            AND incoming_condition_summary IS NULL)
        OR
        (position > 1
            AND incoming_transmission_title IS NOT NULL
            AND btrim(incoming_transmission_title) <> ''
            AND incoming_transmission_mechanism IS NOT NULL
            AND btrim(incoming_transmission_mechanism) <> ''
            AND incoming_condition_summary IS NOT NULL
            AND btrim(incoming_condition_summary) <> '')
    )
);

CREATE TABLE research_reasoning_tree_node_signals (
    reasoning_tree_node_id UUID NOT NULL REFERENCES research_reasoning_tree_nodes(id) ON DELETE CASCADE,
    variable_signal_key TEXT NOT NULL,
    signal_role TEXT NOT NULL,
    signal_direction TEXT NOT NULL,
    display_summary TEXT NOT NULL,
    display_order INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (reasoning_tree_node_id, variable_signal_key),
    CONSTRAINT uq_research_reasoning_tree_node_signals_order
        UNIQUE (reasoning_tree_node_id, display_order),
    CONSTRAINT chk_research_reasoning_tree_node_signals_key
        CHECK (variable_signal_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'),
    CONSTRAINT chk_research_reasoning_tree_node_signals_role
        CHECK (signal_role IN ('primary','supporting','contradicting')),
    CONSTRAINT chk_research_reasoning_tree_node_signals_direction
        CHECK (signal_direction IN ('increase','decrease','mixed','unchanged','uncertain')),
    CONSTRAINT chk_research_reasoning_tree_node_signals_summary
        CHECK (char_length(display_summary) BETWEEN 1 AND 200
            AND display_summary = btrim(display_summary)),
    CONSTRAINT chk_research_reasoning_tree_node_signals_order CHECK (display_order BETWEEN 1 AND 5)
);

CREATE INDEX idx_research_theme_import_receipts_published_at
    ON research_theme_import_receipts (published_at DESC, id);
CREATE INDEX idx_research_themes_published_at
    ON research_themes (published_at DESC, id);
CREATE INDEX idx_research_theme_impacts_node
    ON research_theme_impacts (chain_node_entity_id);
CREATE INDEX idx_research_theme_events_event
    ON research_theme_events (event_id);
CREATE INDEX idx_research_reasoning_tree_receipts_published_at
    ON research_reasoning_tree_import_receipts (published_at DESC, id);
CREATE INDEX idx_research_reasoning_trees_theme_order
    ON research_reasoning_trees (theme_id, display_order, id);
CREATE INDEX idx_research_reasoning_tree_nodes_tree_position
    ON research_reasoning_tree_nodes (reasoning_tree_id, position, id);
CREATE INDEX idx_research_reasoning_tree_nodes_entity
    ON research_reasoning_tree_nodes (chain_node_entity_id);
CREATE INDEX idx_research_reasoning_tree_events_event
    ON research_reasoning_tree_events (event_id);

-- +goose StatementBegin
CREATE FUNCTION prevent_research_publication_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'published research Theme and Reason Tree data is immutable';
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

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
        MESSAGE = 'migration 000031 is forward-only; restore the prior revision with a fresh database or reviewed backup';
END;
$$;
-- +goose StatementEnd
