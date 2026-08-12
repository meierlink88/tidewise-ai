-- +goose Up
DROP TABLE raw_evidence_publication_receipts;
DROP TABLE evidence_publication_receipts;
DROP FUNCTION prevent_evidence_publication_receipt_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000044 is forward-only; use a reviewed forward repair';
END;
$$;
-- +goose StatementEnd
