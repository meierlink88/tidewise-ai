-- +goose Up
CREATE INDEX idx_events_event_semantic_eligible_scan
    ON events(first_seen_at, id)
    WHERE event_status = 'confirmed'
      AND fact_status = 'verified'
      AND event_time IS NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000034 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
