-- +goose Up
ALTER TABLE research_theme_impacts
    ADD COLUMN primary_signal_display_summary TEXT,
    ADD CONSTRAINT chk_research_theme_impacts_primary_signal_summary CHECK (
        primary_signal_display_summary IS NULL
        OR (
            char_length(primary_signal_display_summary) BETWEEN 1 AND 200
            AND primary_signal_display_summary = btrim(primary_signal_display_summary)
        )
    );

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000034 is forward-only; restore the prior revision with a fresh database or reviewed backup';
END;
$$;
-- +goose StatementEnd
