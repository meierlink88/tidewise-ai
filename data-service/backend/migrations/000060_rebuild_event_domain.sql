-- +goose Up
-- Issue #277 authorizes a destructive, zero-compatibility Event reset. Existing
-- Event and dependent Research facts are intentionally deleted, not converted.

DROP TRIGGER IF EXISTS trg_research_theme_receipts_immutable ON research_theme_import_receipts;
DROP TRIGGER IF EXISTS trg_research_themes_immutable ON research_themes;
DROP TRIGGER IF EXISTS trg_research_theme_impacts_immutable ON research_theme_impacts;
DROP TRIGGER IF EXISTS trg_research_theme_events_immutable ON research_theme_events;
DROP TRIGGER IF EXISTS trg_research_reasoning_tree_receipts_immutable ON research_reasoning_tree_import_receipts;
DROP TRIGGER IF EXISTS trg_research_reasoning_trees_immutable ON research_reasoning_trees;
DROP TRIGGER IF EXISTS trg_research_reasoning_tree_events_immutable ON research_reasoning_tree_events;
DROP TRIGGER IF EXISTS trg_research_reasoning_tree_nodes_immutable ON research_reasoning_tree_nodes;
DROP TRIGGER IF EXISTS trg_research_reasoning_tree_node_signals_immutable ON research_reasoning_tree_node_signals;

TRUNCATE TABLE
    research_theme_import_receipts,
    research_themes,
    research_theme_impacts,
    research_theme_events,
    research_reasoning_tree_import_receipts,
    research_reasoning_trees,
    research_reasoning_tree_events,
    research_reasoning_tree_nodes,
    research_reasoning_tree_node_signals
CASCADE;

DROP TABLE IF EXISTS event_publication_receipts CASCADE;
DROP TABLE IF EXISTS event_tag_maps CASCADE;
DROP TABLE IF EXISTS event_tag_defs CASCADE;
DROP TABLE IF EXISTS event_sources CASCADE;
DROP TABLE IF EXISTS raw_documents CASCADE;
DROP TABLE events CASCADE;

CREATE TABLE events (
    id VARCHAR(39) PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    summary TEXT NOT NULL,
    semantic JSONB NOT NULL,
    modality VARCHAR(4) NOT NULL,
    occurred_at TIMESTAMPTZ,
    announced_at TIMESTAMPTZ,
    status VARCHAR(10) NOT NULL DEFAULT 'ACTIVE',
    CONSTRAINT chk_events_identity CHECK (
        id ~ '^EVT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_events_title CHECK (btrim(title) <> ''),
    CONSTRAINT chk_events_summary CHECK (btrim(summary) <> ''),
    CONSTRAINT chk_events_semantic CHECK (
        jsonb_typeof(semantic) = 'object'
        AND jsonb_array_length(jsonb_path_query_array(semantic, '$.keyvalue()')) = 6
        AND semantic ?& ARRAY['who', 'what', 'when', 'where', 'why', 'how']
        AND jsonb_typeof(semantic -> 'who') IN ('string', 'null')
        AND jsonb_typeof(semantic -> 'what') IN ('string', 'null')
        AND jsonb_typeof(semantic -> 'when') IN ('string', 'null')
        AND jsonb_typeof(semantic -> 'where') IN ('string', 'null')
        AND jsonb_typeof(semantic -> 'why') IN ('string', 'null')
        AND jsonb_typeof(semantic -> 'how') IN ('string', 'null')
    ),
    CONSTRAINT chk_events_modality CHECK (modality IN ('FACT', 'PLAN', 'SPEC')),
    CONSTRAINT chk_events_status CHECK (status IN ('ACTIVE', 'DEPRECATED', 'ARCHIVED'))
);

CREATE TABLE event_evidence_links (
    id VARCHAR(39) PRIMARY KEY,
    event_id VARCHAR(39) NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    evidence_id VARCHAR(39) NOT NULL REFERENCES evidences(id) ON DELETE RESTRICT,
    contribution_weight NUMERIC(3,2) NOT NULL,
    CONSTRAINT chk_event_evidence_links_identity CHECK (
        id ~ '^EEL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_event_evidence_links_weight CHECK (contribution_weight BETWEEN 0.00 AND 1.00),
    CONSTRAINT uq_event_evidence_links_endpoints UNIQUE (event_id, evidence_id)
);

CREATE INDEX idx_event_evidence_links_evidence_id ON event_evidence_links(evidence_id);

CREATE TABLE event_actor_links (
    id VARCHAR(39) PRIMARY KEY,
    event_id VARCHAR(39) NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    actor_id VARCHAR(64) NOT NULL,
    actor_type VARCHAR(30),
    actor_name VARCHAR(200),
    relation_type VARCHAR(30) NOT NULL,
    relation_strength NUMERIC(3,2),
    confidence NUMERIC(3,2) NOT NULL DEFAULT 0.70,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_event_actor_links_identity CHECK (
        id ~ '^EAC[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_event_actor_links_actor_id CHECK (btrim(actor_id) <> ''),
    CONSTRAINT chk_event_actor_links_actor_type CHECK (
        actor_type IS NULL OR actor_type IN ('COUNTRY', 'PERSON', 'ORGANIZATION', 'COMPANY')
    ),
    CONSTRAINT chk_event_actor_links_relation_type CHECK (
        relation_type IN ('MENTIONS', 'AFFECTS', 'ORIGINATES_FROM', 'TARGETS')
    ),
    CONSTRAINT chk_event_actor_links_strength CHECK (
        relation_strength IS NULL OR relation_strength BETWEEN 0.00 AND 1.00
    ),
    CONSTRAINT chk_event_actor_links_confidence CHECK (confidence BETWEEN 0.00 AND 0.99),
    CONSTRAINT uq_event_actor_links_relation UNIQUE (event_id, actor_id, relation_type)
);

CREATE INDEX idx_event_actor_links_actor_id ON event_actor_links(actor_id);

CREATE TABLE event_asset_links (
    id VARCHAR(39) PRIMARY KEY,
    event_id VARCHAR(39) NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    asset_id VARCHAR(64) NOT NULL,
    asset_type VARCHAR(30),
    asset_name VARCHAR(200),
    impact_direction VARCHAR(10) NOT NULL,
    impact_magnitude NUMERIC(3,2),
    CONSTRAINT chk_event_asset_links_identity CHECK (
        id ~ '^EAS[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_event_asset_links_asset_id CHECK (btrim(asset_id) <> ''),
    CONSTRAINT chk_event_asset_links_asset_type CHECK (
        asset_type IS NULL OR asset_type IN ('SECURITY', 'COMMODITY', 'INDEX', 'RATE', 'FOREX', 'DERIVATIVE')
    ),
    CONSTRAINT chk_event_asset_links_direction CHECK (
        impact_direction IN ('POSITIVE', 'NEGATIVE', 'NEUTRAL')
    ),
    CONSTRAINT chk_event_asset_links_magnitude CHECK (
        impact_magnitude IS NULL OR impact_magnitude BETWEEN 0.00 AND 1.00
    ),
    CONSTRAINT uq_event_asset_links_endpoints UNIQUE (event_id, asset_id)
);

CREATE INDEX idx_event_asset_links_asset_id ON event_asset_links(asset_id);

-- The non-empty Evidence set is a transaction-level aggregate invariant. The
-- deferred checks allow Event and links to be inserted in one transaction.
-- +goose StatementBegin
CREATE FUNCTION enforce_event_evidence_links()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    affected_event_id TEXT;
BEGIN
    IF TG_TABLE_NAME = 'event_evidence_links'
       AND TG_OP = 'UPDATE'
       AND OLD.event_id IS DISTINCT FROM NEW.event_id
       AND EXISTS (SELECT 1 FROM events WHERE id = OLD.event_id)
       AND NOT EXISTS (SELECT 1 FROM event_evidence_links WHERE event_id = OLD.event_id) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'Event requires at least one Event Evidence Link';
    END IF;
    IF TG_TABLE_NAME = 'events' THEN
        affected_event_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.id ELSE NEW.id END;
    ELSE
        affected_event_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.event_id ELSE NEW.event_id END;
    END IF;
    IF EXISTS (SELECT 1 FROM events WHERE id = affected_event_id)
       AND NOT EXISTS (SELECT 1 FROM event_evidence_links WHERE event_id = affected_event_id) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'Event requires at least one Event Evidence Link';
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER trg_events_require_evidence
AFTER INSERT OR UPDATE ON events
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_event_evidence_links();

CREATE CONSTRAINT TRIGGER trg_event_evidence_links_require_evidence
AFTER INSERT OR UPDATE OR DELETE ON event_evidence_links
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_event_evidence_links();

ALTER TABLE research_theme_events
    ADD CONSTRAINT research_theme_events_event_id_fkey
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE RESTRICT;
ALTER TABLE research_reasoning_tree_events
    ADD CONSTRAINT research_reasoning_tree_events_event_id_fkey
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE RESTRICT;

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
        MESSAGE = 'migration 000060 is destructive and forward-only; restore the reviewed pre-migration snapshot';
END;
$$;
-- +goose StatementEnd
