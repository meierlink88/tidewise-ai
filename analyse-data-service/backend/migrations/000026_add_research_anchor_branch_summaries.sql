-- +goose Up
LOCK TABLE
    research_anchor_import_receipts,
    research_anchors,
    research_anchor_events,
    research_anchor_chain_nodes
IN ACCESS EXCLUSIVE MODE;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM research_anchor_import_receipts LIMIT 1)
       OR EXISTS (SELECT 1 FROM research_anchors LIMIT 1)
       OR EXISTS (SELECT 1 FROM research_anchor_events LIMIT 1)
       OR EXISTS (SELECT 1 FROM research_anchor_chain_nodes LIMIT 1) THEN
        RAISE EXCEPTION 'migration 000026 requires empty Research Anchor publication tables; reset local development data or publish through a reviewed forward path';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE research_anchors
    ADD COLUMN support_summary TEXT,
    ADD COLUMN counter_summary TEXT;

ALTER TABLE research_anchors
    ADD CONSTRAINT chk_research_anchors_support_summary_nonblank
        CHECK (btrim(support_summary) <> ''),
    ADD CONSTRAINT chk_research_anchors_counter_summary_nonblank
        CHECK (counter_summary IS NULL OR btrim(counter_summary) <> '');

ALTER TABLE research_anchors
    ALTER COLUMN support_summary SET NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000026 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END $$;
-- +goose StatementEnd
