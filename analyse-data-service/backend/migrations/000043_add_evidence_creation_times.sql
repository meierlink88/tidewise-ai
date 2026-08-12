-- +goose Up
ALTER TABLE raw_evidences
    ADD COLUMN created_at TIMESTAMPTZ;

ALTER TABLE raw_evidences
    ALTER COLUMN created_at SET DEFAULT transaction_timestamp();

ALTER TABLE raw_evidences
    ADD CONSTRAINT chk_raw_evidences_created_at_new_rows
    CHECK (created_at IS NOT NULL) NOT VALID;

COMMENT ON COLUMN raw_evidences.created_at IS
    'Data数据库生成的Raw Evidence创建时间；历史行不回填并可为空。';

ALTER TABLE evidences
    ADD COLUMN created_at TIMESTAMPTZ;

ALTER TABLE evidences
    ALTER COLUMN created_at SET DEFAULT transaction_timestamp();

ALTER TABLE evidences
    ADD CONSTRAINT chk_evidences_created_at_new_rows
    CHECK (created_at IS NOT NULL) NOT VALID;

COMMENT ON COLUMN evidences.created_at IS
    'Data数据库生成的原子Evidence创建时间；历史行不回填并可为空。';

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000043 is forward-only; use a reviewed forward repair';
END;
$$;
-- +goose StatementEnd
