-- +goose Up
ALTER TABLE reports ADD COLUMN evidence_counts JSONB;
ALTER TABLE reports ADD CONSTRAINT chk_reports_evidence_counts CHECK (
    evidence_counts IS NULL OR jsonb_typeof(evidence_counts) = 'object'
);
COMMENT ON COLUMN reports.evidence_counts IS 'Server-computed distinct Evidence counts by scope path at publication; NULL uses historical snapshot fallback.';

-- +goose Down
ALTER TABLE reports DROP COLUMN evidence_counts;
