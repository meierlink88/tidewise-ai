-- +goose Up
DROP INDEX IF EXISTS ux_event_sources_v2_primary;
DROP INDEX IF EXISTS ux_event_sources_v2_event_document;

ALTER TABLE event_sources
    DROP CONSTRAINT chk_event_sources_v2_contract,
    DROP CONSTRAINT chk_event_sources_contract_version;

ALTER TABLE events
    DROP CONSTRAINT fk_events_primary_source,
    DROP COLUMN primary_source_id;

ALTER TABLE event_sources
    RENAME COLUMN evidence_excerpt TO evidence_statement;

UPDATE event_sources
SET contract_version = 3
WHERE contract_version = 2;

ALTER TABLE event_sources
    DROP COLUMN is_primary,
    ADD CONSTRAINT chk_event_sources_contract_version
        CHECK (contract_version IN (1, 3)),
    ADD CONSTRAINT chk_event_sources_v3_contract
        CHECK (
            contract_version = 1
            OR (
                source_level IN ('primary', 'secondary')
                AND btrim(evidence_statement) <> ''
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

CREATE UNIQUE INDEX ux_event_sources_v3_event_document
    ON event_sources (event_id, raw_document_id)
    WHERE contract_version = 3;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000038 is forward-only; primary Event Evidence semantics are intentionally removed';
END;
$$;
-- +goose StatementEnd
