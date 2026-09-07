-- +goose Up
ALTER TABLE report_evidence_links DROP CONSTRAINT chk_report_evidence_links_scope_type;
ALTER TABLE report_evidence_links ADD CONSTRAINT chk_report_evidence_links_scope_type CHECK (
    scope_type IN ('section_summary', 'anchor', 'reasoning_step', 'industry_chain_summary',
                   'industry_chain_node', 'story_summary', 'concept_summary', 'chain_reasoning_step')
);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'Report analysis scopes are forward-compatible; retain this schema when rolling back the application';
END;
$$;
-- +goose StatementEnd
