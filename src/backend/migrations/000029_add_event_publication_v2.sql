-- +goose Up
ALTER TABLE raw_documents
    ADD COLUMN contract_version SMALLINT NOT NULL DEFAULT 1,
    ADD COLUMN artifact_id TEXT,
    ADD COLUMN source_ref TEXT;

ALTER TABLE raw_documents
    ALTER COLUMN ingest_channel SET DEFAULT '';

ALTER TABLE raw_documents
    ADD CONSTRAINT chk_raw_documents_contract_version
        CHECK (contract_version IN (1, 2)),
    ADD CONSTRAINT chk_raw_documents_v2_contract
        CHECK (
            contract_version = 1
            OR (
                artifact_id IS NOT NULL
                AND char_length(artifact_id) BETWEEN 1 AND 256
                AND btrim(artifact_id) <> ''
                AND source_ref IS NOT NULL
                AND char_length(source_ref) BETWEEN 1 AND 256
                AND btrim(source_ref) <> ''
                AND btrim(source_type) <> ''
                AND btrim(source_name) <> ''
                AND btrim(title) <> ''
                AND content_hash ~ '^[0-9a-f]{64}$'
                AND content_text = ''
                AND raw_object_uri = ''
                AND ingest_channel = ''
                AND source_external_id IS NULL
                AND content_level = ''
                AND (source_url = '' OR source_url ~ '^https?://')
            )
        );

CREATE UNIQUE INDEX ux_raw_documents_artifact_id
    ON raw_documents (artifact_id)
    WHERE contract_version = 2;

CREATE INDEX idx_raw_documents_source_ref
    ON raw_documents (source_ref)
    WHERE contract_version = 2;

ALTER TABLE raw_documents
    DROP COLUMN source_id;

ALTER TABLE event_sources
    ADD COLUMN contract_version SMALLINT NOT NULL DEFAULT 1,
    ADD COLUMN is_primary BOOLEAN;

ALTER TABLE event_sources
    ADD CONSTRAINT chk_event_sources_contract_version
        CHECK (contract_version IN (1, 2)),
    ADD CONSTRAINT chk_event_sources_v2_contract
        CHECK (
            contract_version = 1
            OR (
                is_primary IS NOT NULL
                AND source_level IN ('primary', 'secondary')
                AND btrim(evidence_excerpt) <> ''
                AND evidence_hash ~ '^[0-9a-f]{64}$'
                AND evidence_relation IN ('supports', 'contradicts', 'context')
                AND supports_fields <@ ARRAY['title', 'factual_summary', 'occurred_at', 'fact_payload']::text[]
                AND array_position(supports_fields, NULL::text) IS NULL
                AND (
                    evidence_relation = 'context'
                    OR cardinality(supports_fields) > 0
                )
            )
        );

CREATE UNIQUE INDEX ux_event_sources_v2_event_document
    ON event_sources (event_id, raw_document_id)
    WHERE contract_version = 2;

CREATE UNIQUE INDEX ux_event_sources_v2_primary
    ON event_sources (event_id)
    WHERE contract_version = 2 AND is_primary;

CREATE TABLE event_publication_receipts (
    id UUID PRIMARY KEY,
    contract_version SMALLINT NOT NULL DEFAULT 2,
    package_id TEXT NOT NULL,
    caller_subject TEXT NOT NULL,
    extractor_execution_id TEXT NOT NULL,
    extractor_agent_version TEXT NOT NULL,
    collector_executions JSONB NOT NULL,
    event_ids UUID[] NOT NULL,
    raw_document_ids UUID[] NOT NULL,
    event_source_ids UUID[] NOT NULL,
    event_tag_map_ids UUID[] NOT NULL,
    review_metadata JSONB NOT NULL,
    write_counts JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_event_publication_receipts_version
        CHECK (contract_version = 2),
    CONSTRAINT chk_event_publication_receipts_package
        CHECK (char_length(package_id) BETWEEN 1 AND 256 AND btrim(package_id) <> ''),
    CONSTRAINT chk_event_publication_receipts_caller
        CHECK (char_length(caller_subject) BETWEEN 1 AND 200 AND btrim(caller_subject) <> ''),
    CONSTRAINT chk_event_publication_receipts_extractor_execution
        CHECK (char_length(extractor_execution_id) BETWEEN 1 AND 256 AND btrim(extractor_execution_id) <> ''),
    CONSTRAINT chk_event_publication_receipts_extractor_version
        CHECK (char_length(extractor_agent_version) BETWEEN 1 AND 256 AND btrim(extractor_agent_version) <> ''),
    CONSTRAINT chk_event_publication_receipts_collector_executions
        CHECK (
            jsonb_typeof(collector_executions) = 'array'
            AND jsonb_array_length(collector_executions) >= 1
        ),
    CONSTRAINT chk_event_publication_receipts_event_ids
        CHECK (
            array_ndims(event_ids) = 1
            AND cardinality(event_ids) BETWEEN 1 AND 10
            AND array_position(event_ids, NULL::uuid) IS NULL
        ),
    CONSTRAINT chk_event_publication_receipts_raw_document_ids
        CHECK (
            array_ndims(raw_document_ids) = 1
            AND cardinality(raw_document_ids) >= 1
            AND array_position(raw_document_ids, NULL::uuid) IS NULL
        ),
    CONSTRAINT chk_event_publication_receipts_event_source_ids
        CHECK (
            array_ndims(event_source_ids) = 1
            AND cardinality(event_source_ids) >= 1
            AND array_position(event_source_ids, NULL::uuid) IS NULL
        ),
    CONSTRAINT chk_event_publication_receipts_event_tag_map_ids
        CHECK (
            array_ndims(event_tag_map_ids) = 1
            AND cardinality(event_tag_map_ids) >= 1
            AND array_position(event_tag_map_ids, NULL::uuid) IS NULL
        ),
    CONSTRAINT chk_event_publication_receipts_review_metadata
        CHECK (
            jsonb_typeof(review_metadata) = 'array'
            AND jsonb_array_length(review_metadata) = cardinality(event_ids)
        ),
    CONSTRAINT chk_event_publication_receipts_write_counts
        CHECK (
            jsonb_typeof(write_counts) = 'object'
            AND write_counts ?& ARRAY[
                'events_created',
                'events_reused',
                'raw_documents_created',
                'raw_documents_reused',
                'event_sources_created',
                'event_sources_reused',
                'event_tags_created',
                'event_tags_reused'
            ]::text[]
        )
);

CREATE INDEX idx_event_publication_receipts_package
    ON event_publication_receipts (package_id);

CREATE INDEX idx_event_publication_receipts_caller_imported
    ON event_publication_receipts (caller_subject, imported_at DESC);

-- +goose StatementBegin
CREATE FUNCTION prevent_event_publication_receipt_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'event publication receipts are immutable';
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_event_publication_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON event_publication_receipts
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_event_publication_receipt_mutation();

DROP TABLE raw_document_import_receipts;
DROP FUNCTION prevent_raw_document_import_receipt_mutation();
DROP TABLE event_import_receipts;

DROP TABLE ingestion_run_sources;
DROP TABLE ingestion_runs;
DROP TABLE ingestion_scheduler_configs;
DROP TABLE source_catalogs;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000029 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
