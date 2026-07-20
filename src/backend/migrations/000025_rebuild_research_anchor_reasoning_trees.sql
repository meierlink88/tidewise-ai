-- +goose Up
LOCK TABLE
    research_anchors,
    research_anchor_chain_nodes,
    research_anchor_events,
    research_anchor_indices
IN ACCESS EXCLUSIVE MODE;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM research_anchors LIMIT 1)
        OR EXISTS (SELECT 1 FROM research_anchor_chain_nodes LIMIT 1)
        OR EXISTS (SELECT 1 FROM research_anchor_events LIMIT 1)
        OR EXISTS (SELECT 1 FROM research_anchor_indices LIMIT 1)
    THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migration 000025 requires all legacy research anchor tables to be empty';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP TABLE research_anchor_indices;
DROP TABLE research_anchor_events;
DROP TABLE research_anchor_chain_nodes;
DROP TABLE research_anchors;

CREATE TABLE research_anchor_import_receipts (
    id UUID PRIMARY KEY,
    theme_id UUID NOT NULL UNIQUE REFERENCES research_themes(id),
    publisher_subject TEXT NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    anchor_ids_by_center_chain_node_id JSONB NOT NULL,
    write_counts JSONB NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_research_anchor_import_receipts_id_theme UNIQUE (id, theme_id),
    CONSTRAINT chk_research_anchor_import_receipts_publisher CHECK (
        char_length(publisher_subject) BETWEEN 1 AND 200
        AND btrim(publisher_subject) <> ''
    ),
    CONSTRAINT chk_research_anchor_import_receipts_payload_hash CHECK (
        payload_hash ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT chk_research_anchor_import_receipts_anchor_ids CHECK (
        jsonb_typeof(anchor_ids_by_center_chain_node_id) = 'object'
        AND jsonb_array_length(
            jsonb_path_query_array(anchor_ids_by_center_chain_node_id, '$.keyvalue()')
        ) >= 1
        AND jsonb_array_length(
            jsonb_path_query_array(
                anchor_ids_by_center_chain_node_id,
                '$.keyvalue() ? (@.key like_regex "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" && @.value like_regex "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")'
            )
        ) = jsonb_array_length(
            jsonb_path_query_array(anchor_ids_by_center_chain_node_id, '$.keyvalue()')
        )
    ),
    CONSTRAINT chk_research_anchor_import_receipts_counts CHECK (
        jsonb_typeof(write_counts) = 'object'
        AND write_counts ?& ARRAY[
            'anchors',
            'event_associations',
            'path_nodes',
            'receipts'
        ]::text[]
        AND jsonb_array_length(jsonb_path_query_array(write_counts, '$.keyvalue()')) = 4
        AND jsonb_typeof(write_counts -> 'anchors') = 'number'
        AND jsonb_typeof(write_counts -> 'event_associations') = 'number'
        AND jsonb_typeof(write_counts -> 'path_nodes') = 'number'
        AND jsonb_typeof(write_counts -> 'receipts') = 'number'
        AND (write_counts ->> 'anchors') ~ '^[0-9]+$'
        AND (write_counts ->> 'event_associations') ~ '^[0-9]+$'
        AND (write_counts ->> 'path_nodes') ~ '^[0-9]+$'
        AND (write_counts ->> 'receipts') ~ '^[0-9]+$'
        AND (write_counts ->> 'anchors')::integer >= 1
        AND (write_counts ->> 'event_associations')::integer >= (write_counts ->> 'anchors')::integer
        AND (write_counts ->> 'path_nodes')::integer >= 2 * (write_counts ->> 'anchors')::integer
        AND (write_counts ->> 'receipts')::integer = 1
        AND jsonb_array_length(
            jsonb_path_query_array(anchor_ids_by_center_chain_node_id, '$.keyvalue()')
        ) = (write_counts ->> 'anchors')::integer
    ),
    CONSTRAINT chk_research_anchor_import_receipts_times CHECK (imported_at >= published_at)
);

CREATE TABLE research_anchors (
    id UUID PRIMARY KEY,
    theme_id UUID NOT NULL REFERENCES research_themes(id) ON DELETE CASCADE,
    center_chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
    import_receipt_id UUID NOT NULL,
    one_line_conclusion TEXT NOT NULL,
    fact_summary TEXT NOT NULL,
    net_direction_summary TEXT NOT NULL,
    trading_direction TEXT NOT NULL,
    next_checkpoint TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_research_anchors_theme_center UNIQUE (theme_id, center_chain_node_entity_id),
    CONSTRAINT fk_research_anchors_receipt_theme FOREIGN KEY (import_receipt_id, theme_id)
        REFERENCES research_anchor_import_receipts(id, theme_id),
    CONSTRAINT chk_research_anchors_conclusion_nonblank CHECK (btrim(one_line_conclusion) <> ''),
    CONSTRAINT chk_research_anchors_fact_summary_nonblank CHECK (btrim(fact_summary) <> ''),
    CONSTRAINT chk_research_anchors_net_direction_nonblank CHECK (btrim(net_direction_summary) <> ''),
    CONSTRAINT chk_research_anchors_trading_direction_nonblank CHECK (btrim(trading_direction) <> ''),
    CONSTRAINT chk_research_anchors_checkpoint_nonblank CHECK (btrim(next_checkpoint) <> '')
);

CREATE TABLE research_anchor_chain_nodes (
    anchor_id UUID NOT NULL REFERENCES research_anchors(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
    change_direction TEXT NOT NULL,
    change_summary TEXT NOT NULL,
    impact_summary TEXT NOT NULL,
    incoming_transmission_mechanism TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (anchor_id, position),
    UNIQUE (anchor_id, chain_node_entity_id),
    CONSTRAINT chk_research_anchor_path_position CHECK (position >= 1),
    CONSTRAINT chk_research_anchor_path_direction CHECK (
        change_direction IN ('increase', 'decrease', 'mixed', 'unchanged', 'uncertain')
    ),
    CONSTRAINT chk_research_anchor_path_change_summary_nonblank CHECK (btrim(change_summary) <> ''),
    CONSTRAINT chk_research_anchor_path_impact_summary_nonblank CHECK (btrim(impact_summary) <> ''),
    CONSTRAINT chk_research_anchor_path_incoming_mechanism CHECK (
        (position = 1 AND incoming_transmission_mechanism IS NULL)
        OR (
            position > 1
            AND btrim(incoming_transmission_mechanism) <> ''
            AND incoming_transmission_mechanism IS NOT NULL
        )
    )
);

CREATE TABLE research_anchor_events (
    anchor_id UUID NOT NULL REFERENCES research_anchors(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id),
    evidence_role TEXT NOT NULL,
    evidence_summary TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (anchor_id, event_id),
    CONSTRAINT chk_research_anchor_event_role CHECK (
        evidence_role IN ('driver', 'supporting', 'contradicting', 'context')
    ),
    CONSTRAINT chk_research_anchor_event_summary_nonblank CHECK (btrim(evidence_summary) <> '')
);

CREATE INDEX idx_research_anchor_import_receipts_published_at
    ON research_anchor_import_receipts (published_at DESC, id);
CREATE INDEX idx_research_anchors_center_chain_node
    ON research_anchors (center_chain_node_entity_id);
CREATE INDEX idx_research_anchors_import_receipt_id
    ON research_anchors (import_receipt_id);
CREATE INDEX idx_research_anchor_chain_nodes_node
    ON research_anchor_chain_nodes (chain_node_entity_id);
CREATE INDEX idx_research_anchor_events_event
    ON research_anchor_events (event_id);

-- +goose StatementBegin
CREATE FUNCTION prevent_research_anchor_import_receipt_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'research anchor import receipts are immutable';
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_research_anchor_import_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_anchor_import_receipts
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_research_anchor_import_receipt_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000025 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
