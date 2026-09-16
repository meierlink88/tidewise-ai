-- Candidate for Data #513, NOT an active Goose migration.
-- Promote only with the matching Data/AgentOS/report consumers and reviewed recovery point.
-- Execute within one transaction; never run against a serving database.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

DO $$
BEGIN
    IF current_setting('tidewise.document_event_cutover', true) IS DISTINCT FROM 'issue-513-reviewed' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'document Event cutover requires explicit reviewed authorization';
    END IF;
    IF (SELECT max(version_id) FROM goose_db_version WHERE is_applied) IS DISTINCT FROM 93::bigint THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'candidate requires reviewed baseline ledger 93';
    END IF;
END $$;

-- Lock before checking: a concurrent legacy writer cannot insert after preflight.
LOCK TABLE events, event_evidence_links, event_actor_links, event_asset_links,
    event_publication_receipts, evidences,
    report_evidence_links, report_archive, report_publications, report_summary, report_detail
    IN ACCESS EXCLUSIVE MODE;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM events)
       OR EXISTS (SELECT 1 FROM event_evidence_links)
       OR EXISTS (SELECT 1 FROM event_actor_links)
       OR EXISTS (SELECT 1 FROM event_asset_links)
       OR EXISTS (SELECT 1 FROM event_publication_receipts)
       OR EXISTS (SELECT 1 FROM report_evidence_links)
       OR EXISTS (SELECT 1 FROM report_archive)
       OR EXISTS (SELECT 1 FROM report_publications)
       OR EXISTS (SELECT 1 FROM report_summary)
       OR EXISTS (SELECT 1 FROM report_detail) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'legacy Event/Report data requires an explicit archival or rebuild plan; Evidence-only discard is authorized';
    END IF;
END $$;

DROP TRIGGER trg_events_require_evidence ON events;
DROP TRIGGER trg_event_evidence_links_require_evidence ON event_evidence_links;
DROP FUNCTION enforce_event_evidence_links();
DROP TABLE event_actor_links;
DROP TABLE event_asset_links;
DROP TABLE event_evidence_links;

ALTER TABLE events
    DROP CONSTRAINT chk_events_semantic,
    DROP CONSTRAINT chk_events_modality,
    DROP COLUMN semantic,
    DROP COLUMN modality,
    DROP COLUMN occurred_at,
    DROP COLUMN announced_at,
    ALTER COLUMN title TYPE TEXT,
    ADD COLUMN keywords TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN collected_at TIMESTAMPTZ NOT NULL,
    ADD COLUMN published_at TIMESTAMPTZ;
ALTER TABLE events ADD CONSTRAINT chk_events_keywords CHECK (
    COALESCE(array_ndims(keywords), 1) = 1
    AND cardinality(keywords) <= 5
    AND array_position(keywords, NULL::TEXT) IS NULL
);
CREATE INDEX idx_events_collected_at_id ON events(collected_at DESC, id);
CREATE INDEX idx_events_published_at_id ON events(published_at DESC, id) WHERE published_at IS NOT NULL;
COMMENT ON COLUMN events.collected_at IS 'Frozen Event extraction completion time, supplied by AgentOS; not Raw collection or database insertion time.';
COMMENT ON COLUMN events.published_at IS 'Original news publication time copied from Raw Evidence; unknown stays NULL.';
COMMENT ON COLUMN events.keywords IS 'Zero to five explicit quantitative facts from the article; not topic labels.';

CREATE TABLE event_semantics (
    id VARCHAR(39) PRIMARY KEY,
    event_id VARCHAR(39) NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    actor TEXT,
    action TEXT NOT NULL,
    target TEXT,
    announced_time TEXT,
    effective_time TEXT,
    planned_execution_time TEXT,
    executed_time TEXT,
    statement_type TEXT NOT NULL CHECK (statement_type IN ('POLICY','GENERAL')),
    action_status TEXT NOT NULL CHECK (action_status IN ('PLANNED','OCCURRED')),
    assertion_status TEXT NOT NULL CHECK (assertion_status IN ('CONFIRMED','UNCONFIRMED')),
    CONSTRAINT chk_event_semantics_id CHECK (
        id ~ '^ESM[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    )
);
CREATE INDEX idx_event_semantics_event_id ON event_semantics(event_id);
COMMENT ON TABLE event_semantics IS 'Source-grounded semantic items within one document Event; no position or atomic Event identity. Extraction eligibility belongs to AgentOS, not SQL text interpretation.';
COMMENT ON COLUMN event_semantics.assertion_status IS 'Confirmation in source reporting, not independent verification by the system.';
COMMENT ON COLUMN event_semantics.executed_time IS 'Source explicitly states actual execution; never inferred from an elapsed planned date.';

CREATE TABLE event_evidence_links (
    id VARCHAR(39) PRIMARY KEY,
    event_id VARCHAR(39) NOT NULL UNIQUE REFERENCES events(id) ON DELETE RESTRICT,
    raw_evidence_id VARCHAR(39) NOT NULL UNIQUE REFERENCES raw_evidences(id) ON DELETE RESTRICT,
    CONSTRAINT chk_event_evidence_links_identity CHECK (
        id ~ '^EEL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    )
);
COMMENT ON TABLE event_evidence_links IS 'Exactly one Raw document version per Event and at most one Event per Raw; table name retained deliberately.';

CREATE FUNCTION prevent_event_raw_link_mutation() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'Event Raw provenance is immutable';
END $$;
CREATE TRIGGER trg_event_raw_links_immutable BEFORE UPDATE OR DELETE ON event_evidence_links
FOR EACH ROW EXECUTE FUNCTION prevent_event_raw_link_mutation();

CREATE FUNCTION enforce_event_raw_link() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE affected_event_id TEXT;
BEGIN
    IF TG_TABLE_NAME = 'events' THEN
        affected_event_id := NEW.id;
    ELSE
        affected_event_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.event_id ELSE NEW.event_id END;
        IF TG_OP = 'UPDATE' AND OLD.event_id IS DISTINCT FROM NEW.event_id THEN
            IF EXISTS (SELECT 1 FROM events WHERE id = OLD.event_id)
               AND NOT EXISTS (SELECT 1 FROM event_evidence_links WHERE event_id = OLD.event_id) THEN
                RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'Event requires exactly one Raw Evidence link';
            END IF;
        END IF;
    END IF;
    IF EXISTS (SELECT 1 FROM events WHERE id = affected_event_id)
       AND NOT EXISTS (SELECT 1 FROM event_evidence_links WHERE event_id = affected_event_id) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'Event requires exactly one Raw Evidence link';
    END IF;
    RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER trg_events_require_raw
AFTER INSERT OR UPDATE ON events DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_event_raw_link();
CREATE CONSTRAINT TRIGGER trg_event_evidence_links_require_raw
AFTER INSERT OR UPDATE OR DELETE ON event_evidence_links DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_event_raw_link();

CREATE FUNCTION prevent_event_raw_link_truncate() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM events) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'cannot truncate source links while Events exist';
    END IF;
    RETURN NULL;
END $$;
CREATE TRIGGER trg_event_raw_links_no_orphans BEFORE TRUNCATE ON event_evidence_links
FOR EACH STATEMENT EXECUTE FUNCTION prevent_event_raw_link_truncate();

ALTER TABLE event_publication_receipts RENAME COLUMN published_at TO recorded_at;
COMMENT ON COLUMN event_publication_receipts.recorded_at IS 'Data publication receipt time, distinct from original news published_at.';

-- Preserve the report scope and immutable-link behavior, change the referenced fact.
ALTER TABLE report_evidence_links DROP CONSTRAINT report_evidence_links_evidence_id_fkey;
ALTER TABLE report_evidence_links DROP CONSTRAINT chk_report_evidence_links_evidence_id;
ALTER TABLE report_evidence_links RENAME TO report_event_links;
ALTER TABLE report_event_links RENAME COLUMN evidence_id TO event_id;
ALTER TABLE report_event_links DROP CONSTRAINT chk_report_evidence_links_scope_type;
ALTER TABLE report_event_links ADD CONSTRAINT chk_report_event_links_scope_type CHECK (
    scope_type IN ('section_summary','anchor','reasoning_step','industry_chain_summary','industry_chain_node',
                  'story_summary','concept_summary','chain_reasoning_step','normalized_report_event')
);
ALTER TABLE report_event_links ADD CONSTRAINT report_event_links_event_id_fkey
    FOREIGN KEY(event_id) REFERENCES events(id) ON DELETE RESTRICT;
ALTER TABLE report_event_links ADD CONSTRAINT chk_report_event_links_event_id CHECK (
    event_id ~ '^EVT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
);
ALTER TABLE report_event_links RENAME CONSTRAINT report_evidence_links_pkey TO report_event_links_pkey;
ALTER TABLE report_event_links RENAME CONSTRAINT report_evidence_links_report_id_fkey TO report_event_links_report_id_fkey;
ALTER TABLE report_event_links RENAME CONSTRAINT chk_report_evidence_links_id TO chk_report_event_links_id;
ALTER TABLE report_event_links RENAME CONSTRAINT chk_report_evidence_links_report_id TO chk_report_event_links_report_id;
ALTER TABLE report_event_links RENAME CONSTRAINT chk_report_evidence_links_scope_path TO chk_report_event_links_scope_path;
ALTER TABLE report_event_links RENAME CONSTRAINT chk_report_evidence_links_position TO chk_report_event_links_position;
ALTER TABLE report_event_links RENAME CONSTRAINT uq_report_evidence_links_scope_evidence TO uq_report_event_links_scope_event;
ALTER TABLE report_event_links RENAME CONSTRAINT uq_report_evidence_links_scope_position TO uq_report_event_links_scope_position;
ALTER INDEX idx_report_evidence_links_evidence_id RENAME TO idx_report_event_links_event_id;
ALTER INDEX idx_report_evidence_links_report_scope_position RENAME TO idx_report_event_links_report_scope_position;
ALTER TRIGGER trg_report_evidence_links_immutable ON report_event_links RENAME TO trg_report_event_links_immutable;
ALTER TABLE report_archive RENAME COLUMN evidence_counts TO event_counts;
ALTER TABLE report_publications RENAME COLUMN evidence_counts TO event_counts;
COMMENT ON TABLE report_event_links IS 'Immutable report scope references to Event; raw document provenance resolves through event_evidence_links.';
COMMENT ON COLUMN report_event_links.event_id IS 'Referenced document Event, never an EVD or RAW identity.';
COMMENT ON COLUMN report_event_links.position IS 'Report citation presentation order; not a position on event_semantics.';
COMMENT ON COLUMN report_archive.event_counts IS 'Distinct Event counts by report scope, computed at publication.';
COMMENT ON COLUMN report_publications.event_counts IS 'Distinct Event counts by report scope, computed at publication.';

-- User explicitly authorized discarding existing Atomic Evidence rows (#513).
-- No CASCADE: unexpected dependants must stop the entire transaction.
DROP TABLE evidences;
