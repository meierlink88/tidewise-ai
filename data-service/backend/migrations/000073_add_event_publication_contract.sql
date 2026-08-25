-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM events) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migration 000073 requires an empty events table; preserve the pre-migration recovery point and curate Events through the new contract';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE events DROP CONSTRAINT chk_events_semantic;
ALTER TABLE events ADD CONSTRAINT chk_events_semantic CHECK (
    jsonb_typeof(semantic) = 'object'
    AND jsonb_array_length(jsonb_path_query_array(semantic, '$.keyvalue()')) = 7
    AND semantic ?& ARRAY['actors', 'action', 'objects', 'stage', 'jurisdictions', 'effective_at', 'time_precision']
    AND jsonb_typeof(semantic -> 'actors') = 'array'
    AND jsonb_array_length(semantic -> 'actors') > 0
    AND NOT jsonb_path_exists(semantic, '$.actors[*] ? (@.type() != "string" || @ == "")')
    AND jsonb_typeof(semantic -> 'action') = 'string'
    AND btrim(semantic ->> 'action') <> ''
    AND jsonb_typeof(semantic -> 'objects') = 'array'
    AND jsonb_array_length(semantic -> 'objects') > 0
    AND NOT jsonb_path_exists(semantic, '$.objects[*] ? (@.type() != "string" || @ == "")')
    AND semantic ->> 'stage' IN ('OCCURRED','ANNOUNCED','EFFECTIVE','IMPLEMENTED','UPDATED','SUSPENDED','TERMINATED','EXPECTED')
    AND jsonb_typeof(semantic -> 'jurisdictions') = 'array'
    AND NOT jsonb_path_exists(semantic, '$.jurisdictions[*] ? (@.type() != "string" || @ == "")')
    AND jsonb_typeof(semantic -> 'effective_at') IN ('string','null')
    AND (jsonb_typeof(semantic -> 'effective_at') = 'null'
         OR semantic ->> 'effective_at' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T.*Z$')
    AND semantic ->> 'time_precision' IN ('INSTANT','DAY','MONTH','QUARTER','YEAR','UNKNOWN')
);

CREATE TABLE event_publication_receipts (
    id VARCHAR(39) PRIMARY KEY,
    publisher_subject VARCHAR(200) NOT NULL,
    publication_key VARCHAR(200) NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    event_id VARCHAR(39) NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    published_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_event_publication_receipts_identity CHECK (
        id ~ '^EPR[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_event_publication_receipts_subject CHECK (btrim(publisher_subject) <> ''),
    CONSTRAINT chk_event_publication_receipts_key CHECK (btrim(publication_key) <> ''),
    CONSTRAINT chk_event_publication_receipts_hash CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT uq_event_publication_receipts_subject_key UNIQUE (publisher_subject, publication_key),
    CONSTRAINT uq_event_publication_receipts_event UNIQUE (event_id)
);

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000073 is a forward-only Event contract cutover; restore the pre-migration recovery point with the previous application release';
END
$$;
-- +goose StatementEnd
