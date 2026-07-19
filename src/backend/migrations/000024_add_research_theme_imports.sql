-- +goose Up
ALTER TABLE research_themes
    RENAME COLUMN index_impact_summary TO market_confirmation_summary;

ALTER TABLE research_themes
    RENAME CONSTRAINT chk_research_themes_index_summary_nonblank
    TO chk_research_themes_market_confirmation_summary_nonblank;

CREATE TABLE research_theme_import_receipts (
    id UUID PRIMARY KEY,
    analysis_batch_id TEXT NOT NULL UNIQUE,
    publisher_subject TEXT NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    theme_ids_by_key JSONB NOT NULL,
    write_counts JSONB NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_research_theme_import_receipts_batch CHECK (
        char_length(analysis_batch_id) BETWEEN 1 AND 200
        AND btrim(analysis_batch_id) <> ''
    ),
    CONSTRAINT chk_research_theme_import_receipts_publisher CHECK (
        char_length(publisher_subject) BETWEEN 1 AND 200
        AND btrim(publisher_subject) <> ''
    ),
    CONSTRAINT chk_research_theme_import_receipts_payload_hash CHECK (
        payload_hash ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT chk_research_theme_import_receipts_theme_ids CHECK (
        jsonb_typeof(theme_ids_by_key) = 'object'
        AND jsonb_array_length(jsonb_path_query_array(theme_ids_by_key, '$.keyvalue()')) >= 1
    ),
    CONSTRAINT chk_research_theme_import_receipts_counts CHECK (
        jsonb_typeof(write_counts) = 'object'
        AND write_counts ?& ARRAY[
            'themes',
            'chain_node_associations',
            'event_associations',
            'receipts'
        ]::text[]
        AND jsonb_typeof(write_counts -> 'themes') = 'number'
        AND jsonb_typeof(write_counts -> 'chain_node_associations') = 'number'
        AND jsonb_typeof(write_counts -> 'event_associations') = 'number'
        AND jsonb_typeof(write_counts -> 'receipts') = 'number'
        AND (write_counts ->> 'themes')::integer >= 1
        AND (write_counts ->> 'chain_node_associations')::integer >= (write_counts ->> 'themes')::integer
        AND (write_counts ->> 'event_associations')::integer >= (write_counts ->> 'themes')::integer
        AND (write_counts ->> 'receipts')::integer = 1
        AND jsonb_array_length(jsonb_path_query_array(theme_ids_by_key, '$.keyvalue()')) = (write_counts ->> 'themes')::integer
    ),
    CONSTRAINT chk_research_theme_import_receipts_times CHECK (imported_at >= published_at)
);

ALTER TABLE research_themes
    ADD COLUMN theme_key TEXT,
    ADD COLUMN import_receipt_id UUID REFERENCES research_theme_import_receipts(id);

UPDATE research_themes
SET theme_key = 'legacy:' || lower(id::text)
WHERE theme_key IS NULL;

ALTER TABLE research_themes
    ALTER COLUMN theme_key SET NOT NULL,
    ADD CONSTRAINT uq_research_themes_batch_theme_key UNIQUE (analysis_batch_id, theme_key),
    ADD CONSTRAINT chk_research_themes_import_identity CHECK (
        theme_key ~ '^[a-z0-9][a-z0-9._:-]{0,127}$'
        AND (
            import_receipt_id IS NOT NULL
            OR theme_key = 'legacy:' || lower(id::text)
        )
    );

ALTER TABLE research_themes
    DROP CONSTRAINT chk_research_themes_window,
    ADD CONSTRAINT chk_research_themes_window CHECK (
        (window_start IS NULL AND window_end IS NULL)
        OR (
            window_start IS NOT NULL
            AND window_end IS NOT NULL
            AND window_end > window_start
        )
    ) NOT VALID;

CREATE INDEX idx_research_theme_import_receipts_published_at
    ON research_theme_import_receipts (published_at DESC, id);
CREATE INDEX idx_research_themes_analysis_batch_id
    ON research_themes (analysis_batch_id);
CREATE INDEX idx_research_themes_import_receipt_id
    ON research_themes (import_receipt_id);

-- +goose StatementBegin
CREATE FUNCTION prevent_research_theme_import_receipt_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'research theme import receipts are immutable';
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_research_theme_import_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON research_theme_import_receipts
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_research_theme_import_receipt_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000024 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
