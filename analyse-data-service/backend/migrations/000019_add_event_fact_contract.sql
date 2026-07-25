-- +goose Up
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS fact_payload JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE event_sources
    ADD COLUMN IF NOT EXISTS evidence_relation VARCHAR(32),
    ADD COLUMN IF NOT EXISTS supports_fields TEXT[] NOT NULL DEFAULT '{}'::text[];

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_event_sources_evidence_relation'
          AND conrelid = 'event_sources'::regclass
    ) THEN
        ALTER TABLE event_sources
            ADD CONSTRAINT chk_event_sources_evidence_relation
            CHECK (evidence_relation IS NULL OR evidence_relation IN ('supports', 'contradicts', 'context'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_event_sources_supports_fields'
          AND conrelid = 'event_sources'::regclass
    ) THEN
        ALTER TABLE event_sources
            ADD CONSTRAINT chk_event_sources_supports_fields
            CHECK (
                evidence_relation IS NULL
                OR evidence_relation = 'context'
                OR cardinality(supports_fields) > 0
            );
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE event_tag_maps
    ADD COLUMN IF NOT EXISTS confidence NUMERIC(5,4),
    ADD COLUMN IF NOT EXISTS assignment_reason TEXT NOT NULL DEFAULT '';

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_event_tag_maps_confidence'
          AND conrelid = 'event_tag_maps'::regclass
    ) THEN
        ALTER TABLE event_tag_maps
            ADD CONSTRAINT chk_event_tag_maps_confidence
            CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1));
    END IF;
END;
$$;
-- +goose StatementEnd

CREATE UNIQUE INDEX IF NOT EXISTS ux_event_sources_event_document_evidence
    ON event_sources (event_id, raw_document_id, evidence_hash);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000019 is irreversible; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
