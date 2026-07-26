ALTER TABLE agent_executions
    DROP CONSTRAINT agent_executions_status_check,
    DROP CONSTRAINT agent_executions_trigger_source_check,
    DROP CONSTRAINT agent_executions_trigger_schedule_check;

ALTER TABLE agent_executions
    ADD CONSTRAINT agent_executions_status_check
        CHECK (status IN (
            'queued', 'planning', 'collecting', 'materializing', 'running',
            'succeeded', 'succeeded_no_change', 'partially_succeeded', 'failed', 'skipped'
        )),
    ADD CONSTRAINT agent_executions_trigger_source_check
        CHECK (trigger_source IN ('api', 'schedule', 'dependent')),
    ADD CONSTRAINT agent_executions_trigger_schedule_check
        CHECK (
            (trigger_source IN ('api', 'dependent') AND schedule_id IS NULL)
            OR
            (trigger_source = 'schedule' AND schedule_id IS NOT NULL)
        );

DROP INDEX agent_executions_one_active;

CREATE UNIQUE INDEX agent_executions_one_active
    ON agent_executions (agent_key)
    WHERE status IN ('queued', 'planning', 'collecting', 'materializing', 'running');

INSERT INTO agent_definitions (agent_key, display_name)
VALUES ('event-fact-extractor', 'Event Fact Extractor Agent')
ON CONFLICT (agent_key) DO NOTHING;

INSERT INTO agent_versions (version, agent_key)
VALUES ('event-fact-extractor.v1', 'event-fact-extractor')
ON CONFLICT (version) DO NOTHING;

CREATE TABLE artifact_ready_signals (
    collector_execution_id uuid PRIMARY KEY
        REFERENCES agent_executions(execution_id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    dispatched_at timestamptz,
    CONSTRAINT artifact_ready_signals_status_check
        CHECK (status IN ('pending', 'dispatched')),
    CONSTRAINT artifact_ready_signals_dispatch_check
        CHECK (
            (status = 'pending' AND dispatched_at IS NULL)
            OR
            (status = 'dispatched' AND dispatched_at IS NOT NULL)
        )
);

CREATE TABLE event_extraction_work_items (
    work_item_key char(64) PRIMARY KEY,
    collector_execution_ids uuid[] NOT NULL,
    extractor_agent_version text NOT NULL
        REFERENCES agent_versions(version),
    status text NOT NULL,
    current_execution_id uuid REFERENCES agent_executions(execution_id),
    extraction_result jsonb NOT NULL DEFAULT '{}'::jsonb,
    tag_catalog_revision text,
    tag_catalog_hash char(64),
    error_code text,
    error_summary text,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT event_extraction_work_items_key_check
        CHECK (work_item_key ~ '^[0-9a-f]{64}$'),
    CONSTRAINT event_extraction_work_items_collectors_check
        CHECK (
            array_ndims(collector_execution_ids) = 1
            AND cardinality(collector_execution_ids) >= 1
            AND array_position(collector_execution_ids, NULL::uuid) IS NULL
        ),
    CONSTRAINT event_extraction_work_items_status_check
        CHECK (status IN (
            'pending', 'running', 'awaiting_tag_catalog', 'awaiting_review',
            'ready_to_publish', 'publishing', 'published', 'retry_wait',
            'blocked', 'rejected', 'no_events'
        )),
    CONSTRAINT event_extraction_work_items_result_check
        CHECK (jsonb_typeof(extraction_result) = 'object'),
    CONSTRAINT event_extraction_work_items_catalog_check
        CHECK (
            (tag_catalog_revision IS NULL AND tag_catalog_hash IS NULL)
            OR
            (
                tag_catalog_revision IS NOT NULL
                AND tag_catalog_hash ~ '^[0-9a-f]{64}$'
            )
        )
);

CREATE INDEX event_extraction_work_items_pending_idx
    ON event_extraction_work_items (updated_at, work_item_key)
    WHERE status IN ('pending', 'awaiting_tag_catalog', 'retry_wait', 'ready_to_publish', 'publishing');

CREATE TABLE event_extractor_executions (
    execution_id uuid PRIMARY KEY
        REFERENCES agent_executions(execution_id) ON DELETE CASCADE,
    work_item_key char(64) NOT NULL
        REFERENCES event_extraction_work_items(work_item_key) ON DELETE CASCADE,
    prompt_sha256 char(64) NOT NULL,
    schema_sha256 char(64) NOT NULL,
    provider_key text NOT NULL,
    model text NOT NULL,
    tag_catalog_revision text,
    tag_catalog_hash char(64),
    extraction_model_calls integer NOT NULL DEFAULT 0,
    review_model_calls integer NOT NULL DEFAULT 0,
    CONSTRAINT event_extractor_executions_prompt_hash_check
        CHECK (prompt_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT event_extractor_executions_schema_hash_check
        CHECK (schema_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT event_extractor_executions_catalog_check
        CHECK (
            (tag_catalog_revision IS NULL AND tag_catalog_hash IS NULL)
            OR
            (
                tag_catalog_revision IS NOT NULL
                AND tag_catalog_hash ~ '^[0-9a-f]{64}$'
            )
        ),
    CONSTRAINT event_extractor_executions_call_count_check
        CHECK (extraction_model_calls >= 0 AND review_model_calls >= 0)
);

CREATE TABLE event_publication_journal (
    work_item_key char(64) NOT NULL
        REFERENCES event_extraction_work_items(work_item_key) ON DELETE CASCADE,
    batch_ordinal smallint NOT NULL,
    package_id text NOT NULL,
    payload_bytes bytea NOT NULL,
    payload_sha256 char(64) NOT NULL,
    status text NOT NULL,
    receipt_id text,
    attempt_count integer NOT NULL DEFAULT 0,
    error_code text,
    error_summary text,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (work_item_key, batch_ordinal),
    UNIQUE (package_id),
    CONSTRAINT event_publication_journal_ordinal_check
        CHECK (batch_ordinal >= 1),
    CONSTRAINT event_publication_journal_payload_check
        CHECK (
            octet_length(payload_bytes) > 0
            AND payload_sha256 ~ '^[0-9a-f]{64}$'
        ),
    CONSTRAINT event_publication_journal_status_check
        CHECK (status IN ('prepared', 'sending', 'acknowledged', 'retry_wait', 'blocked')),
    CONSTRAINT event_publication_journal_attempt_check
        CHECK (attempt_count >= 0),
    CONSTRAINT event_publication_journal_receipt_check
        CHECK (
            (status = 'acknowledged' AND receipt_id IS NOT NULL)
            OR
            (status <> 'acknowledged')
        )
);

CREATE INDEX event_publication_journal_delivery_idx
    ON event_publication_journal (updated_at, work_item_key, batch_ordinal)
    WHERE status IN ('prepared', 'sending', 'retry_wait');

CREATE TABLE event_fact_canonical_events (
    dedupe_key text PRIMARY KEY,
    identity_hash char(64) NOT NULL UNIQUE,
    core_facts jsonb NOT NULL,
    published_at timestamptz NOT NULL,
    CONSTRAINT event_fact_canonical_events_identity_check
        CHECK (identity_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT event_fact_canonical_events_core_check
        CHECK (jsonb_typeof(core_facts) = 'object')
);

CREATE FUNCTION prevent_event_publication_journal_payload_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.work_item_key <> OLD.work_item_key
       OR NEW.batch_ordinal <> OLD.batch_ordinal
       OR NEW.package_id <> OLD.package_id
       OR NEW.payload_bytes <> OLD.payload_bytes
       OR NEW.payload_sha256 <> OLD.payload_sha256
    THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'Event Publication Journal payload is immutable';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER event_publication_journal_payload_immutable
BEFORE UPDATE ON event_publication_journal
FOR EACH ROW
EXECUTE FUNCTION prevent_event_publication_journal_payload_mutation();
