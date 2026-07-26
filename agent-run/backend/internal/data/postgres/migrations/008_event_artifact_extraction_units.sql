ALTER TABLE event_extraction_work_items
    DROP CONSTRAINT event_extraction_work_items_status_check;

ALTER TABLE event_extraction_work_items
    ADD CONSTRAINT event_extraction_work_items_status_check
        CHECK (status IN (
            'pending', 'running', 'awaiting_tag_catalog', 'awaiting_review',
            'ready_to_publish', 'publishing', 'published', 'partially_published',
            'retry_wait', 'blocked', 'rejected', 'no_events'
        ));

CREATE TABLE event_artifact_extraction_units (
    unit_key char(64) PRIMARY KEY,
    work_item_key char(64) NOT NULL
        REFERENCES event_extraction_work_items(work_item_key) ON DELETE CASCADE,
    artifact_ordinal integer NOT NULL,
    artifact_id text NOT NULL,
    collector_execution_id uuid NOT NULL
        REFERENCES agent_executions(execution_id) ON DELETE CASCADE,
    content_sha256 char(64) NOT NULL,
    status text NOT NULL,
    current_execution_id uuid REFERENCES agent_executions(execution_id),
    extraction_result jsonb NOT NULL DEFAULT '{}'::jsonb,
    tag_catalog_revision text,
    tag_catalog_hash char(64),
    error_code text,
    error_summary text,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT event_artifact_units_work_ordinal_key
        UNIQUE (work_item_key, artifact_ordinal),
    CONSTRAINT event_artifact_units_work_artifact_key
        UNIQUE (work_item_key, artifact_id),
    CONSTRAINT event_artifact_extraction_units_key_check
        CHECK (unit_key ~ '^[0-9a-f]{64}$'),
    CONSTRAINT event_artifact_extraction_units_ordinal_check
        CHECK (artifact_ordinal >= 1),
    CONSTRAINT event_artifact_extraction_units_artifact_check
        CHECK (char_length(btrim(artifact_id)) > 0),
    CONSTRAINT event_artifact_extraction_units_content_check
        CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT event_artifact_extraction_units_status_check
        CHECK (status IN (
            'pending', 'running', 'awaiting_tag_catalog', 'ready_to_publish',
            'publishing', 'published', 'retry_wait', 'blocked', 'rejected',
            'no_events'
        )),
    CONSTRAINT event_artifact_extraction_units_result_check
        CHECK (jsonb_typeof(extraction_result) = 'object'),
    CONSTRAINT event_artifact_extraction_units_catalog_check
        CHECK (
            (tag_catalog_revision IS NULL AND tag_catalog_hash IS NULL)
            OR
            (
                tag_catalog_revision IS NOT NULL
                AND tag_catalog_hash ~ '^[0-9a-f]{64}$'
            )
        )
);

CREATE INDEX event_artifact_extraction_units_ready_idx
    ON event_artifact_extraction_units (updated_at, work_item_key, artifact_ordinal)
    WHERE status IN ('pending', 'awaiting_tag_catalog', 'retry_wait');

ALTER TABLE event_extractor_executions
    ADD COLUMN unit_key char(64)
        REFERENCES event_artifact_extraction_units(unit_key) ON DELETE CASCADE;

ALTER TABLE event_publication_journal
    ADD COLUMN unit_key char(64)
        REFERENCES event_artifact_extraction_units(unit_key) ON DELETE CASCADE;

CREATE UNIQUE INDEX event_publication_journal_unit_key_unique
    ON event_publication_journal (unit_key)
    WHERE unit_key IS NOT NULL;
