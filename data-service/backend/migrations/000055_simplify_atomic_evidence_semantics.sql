-- +goose Up
-- Zero-compatibility cutover authorized by Issue #255. Stop Evidence writers
-- and take an approved PostgreSQL recovery point before apply. Historical
-- Atomic Evidence is intentionally deleted; Raw Evidence remains intact.

TRUNCATE TABLE evidences;

DROP INDEX idx_evidences_expression_key;

ALTER TABLE evidences
    DROP CONSTRAINT uq_evidences_raw_split_order,
    DROP CONSTRAINT chk_evidences_split_order,
    DROP CONSTRAINT chk_evidences_layer_type,
    DROP CONSTRAINT chk_evidences_source_what,
    DROP CONSTRAINT chk_evidences_expression_fingerprint,
    DROP CONSTRAINT chk_evidences_expression_key,
    DROP CONSTRAINT chk_evidences_fingerprint_version,
    DROP CONSTRAINT chk_evidences_layer_fields,
    DROP COLUMN split_order,
    DROP COLUMN layer_type,
    DROP COLUMN source_who,
    DROP COLUMN source_what,
    DROP COLUMN source_when,
    DROP COLUMN source_when_raw,
    DROP COLUMN source_where,
    DROP COLUMN source_why,
    DROP COLUMN source_how,
    DROP COLUMN source_who_core,
    DROP COLUMN source_what_core,
    DROP COLUMN source_when_core,
    DROP COLUMN source_when_raw_core,
    DROP COLUMN source_where_core,
    DROP COLUMN source_why_core,
    DROP COLUMN source_how_core,
    DROP COLUMN expression_key,
    DROP COLUMN fingerprint_version;

ALTER TABLE evidences
    RENAME COLUMN expression_fingerprint TO summary;

ALTER TABLE evidences
    ADD COLUMN semantic JSONB NOT NULL,
    ADD CONSTRAINT chk_evidences_summary CHECK (btrim(summary) <> ''),
    ADD CONSTRAINT chk_evidences_semantic CHECK (
        jsonb_typeof(semantic) = 'object'
        AND semantic = jsonb_build_object(
            'who', semantic -> 'who',
            'what', semantic -> 'what',
            'when', semantic -> 'when',
            'where', semantic -> 'where',
            'why', semantic -> 'why',
            'how', semantic -> 'how'
        )
        AND jsonb_typeof(semantic -> 'what') = 'string'
        AND btrim(semantic ->> 'what') <> ''
        AND (
            jsonb_typeof(semantic -> 'who') = 'null'
            OR (jsonb_typeof(semantic -> 'who') = 'string' AND btrim(semantic ->> 'who') <> '')
        )
        AND (
            jsonb_typeof(semantic -> 'when') = 'null'
            OR (jsonb_typeof(semantic -> 'when') = 'string' AND btrim(semantic ->> 'when') <> '')
        )
        AND (
            jsonb_typeof(semantic -> 'where') = 'null'
            OR (jsonb_typeof(semantic -> 'where') = 'string' AND btrim(semantic ->> 'where') <> '')
        )
        AND (
            jsonb_typeof(semantic -> 'why') = 'null'
            OR (jsonb_typeof(semantic -> 'why') = 'string' AND btrim(semantic ->> 'why') <> '')
        )
        AND (
            jsonb_typeof(semantic -> 'how') = 'null'
            OR (jsonb_typeof(semantic -> 'how') = 'string' AND btrim(semantic ->> 'how') <> '')
        )
    );

COMMENT ON COLUMN evidences.summary IS '原子 Evidence 的简洁事实摘要。';
COMMENT ON COLUMN evidences.semantic IS '严格一层 who/what/when/where/why/how 结构的原子 Evidence 语义。';

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000055 is forward-only; restore the reviewed pre-migration snapshot';
END;
$$;
-- +goose StatementEnd
