-- +goose Up
DROP TABLE chain_node_physical_constraints;
DROP TABLE chain_node_relations;

DROP TABLE industry_relationship_import_receipts;
DROP FUNCTION prevent_industry_relationship_import_receipt_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000070 is forward-only; restore the pre-migration PostgreSQL snapshot';
END;
$$;
-- +goose StatementEnd
