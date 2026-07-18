-- +goose Up
CREATE TABLE raw_document_import_receipts (
    id UUID NOT NULL,
    caller_identity TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    raw_document_ids UUID[] NOT NULL,
    result_payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT raw_document_import_receipts_pkey PRIMARY KEY (id),
    CONSTRAINT uq_raw_document_import_receipts_caller_key UNIQUE (caller_identity, idempotency_key),
    CONSTRAINT chk_raw_document_import_receipts_caller CHECK (char_length(caller_identity) BETWEEN 1 AND 200 AND btrim(caller_identity) <> ''),
    CONSTRAINT chk_raw_document_import_receipts_key CHECK (char_length(idempotency_key) BETWEEN 1 AND 200 AND btrim(idempotency_key) <> ''),
    CONSTRAINT chk_raw_document_import_receipts_payload_hash CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_raw_document_import_receipts_raw_ids CHECK (array_ndims(raw_document_ids) = 1 AND cardinality(raw_document_ids) >= 1 AND array_position(raw_document_ids, NULL::uuid) IS NULL),
    CONSTRAINT chk_raw_document_import_receipts_result CHECK (
        jsonb_typeof(result_payload) = 'object'
        AND result_payload ?& ARRAY['receipt_id', 'payload_hash', 'raw_document_ids', 'items', 'imported_at']::text[]
        AND jsonb_typeof(result_payload -> 'receipt_id') = 'string'
        AND result_payload -> 'receipt_id' = to_jsonb(id::text)
        AND jsonb_typeof(result_payload -> 'payload_hash') = 'string'
        AND result_payload -> 'payload_hash' = to_jsonb(payload_hash::text)
        AND jsonb_typeof(result_payload -> 'raw_document_ids') = 'array'
        AND result_payload -> 'raw_document_ids' = to_jsonb(raw_document_ids)
        AND CASE
            WHEN jsonb_typeof(result_payload -> 'items') = 'array' THEN
                jsonb_array_length(result_payload -> 'items') = cardinality(raw_document_ids)
                AND jsonb_path_query_array(result_payload, '$.items[*].raw_document_id') = to_jsonb(raw_document_ids)
                AND jsonb_array_length(jsonb_path_query_array(result_payload, '$.items[*].disposition')) = cardinality(raw_document_ids)
                AND jsonb_path_query_array(result_payload, '$.items[*].disposition') <@ '["created", "reused"]'::jsonb
            ELSE false
        END
        AND jsonb_typeof(result_payload -> 'imported_at') = 'string'
        AND (result_payload ->> 'imported_at')::timestamptz = imported_at
    )
);

CREATE INDEX idx_raw_document_import_receipts_imported_at
    ON raw_document_import_receipts (imported_at);

-- +goose StatementBegin
CREATE FUNCTION prevent_raw_document_import_receipt_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'raw document import receipts are immutable';
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_raw_document_import_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON raw_document_import_receipts
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_raw_document_import_receipt_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000022 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
