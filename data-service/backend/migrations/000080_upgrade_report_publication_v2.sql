-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM reports LIMIT 1) OR
       EXISTS (SELECT 1 FROM report_evidence_links LIMIT 1) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'report-publication.v2 requires an empty v1 Report store; restore the reviewed pre-publication state before applying migration 000080';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE reports
    DROP CONSTRAINT chk_reports_source_report_id,
    DROP CONSTRAINT chk_reports_contract_version;

ALTER TABLE reports
    RENAME COLUMN source_report_id TO publisher_report_id;

ALTER TABLE reports
    ADD CONSTRAINT chk_reports_publisher_report_id CHECK (
        publisher_report_id = btrim(publisher_report_id) AND publisher_report_id <> ''
    ),
    ADD CONSTRAINT chk_reports_contract_version CHECK (
        contract_version = 'report-publication.v2'
    );

COMMENT ON COLUMN reports.publisher_report_id IS
    'AgentOS 为当前发布报告生成的全局唯一、重试稳定外部身份；不是其来源报告身份。';
COMMENT ON COLUMN reports.content IS
    '报告完整 typed JSONB 快照；产业链必选，上层 Section 可选，summary/detail 同源且不持久化首页卡片。';

ALTER TABLE report_evidence_links
    DROP CONSTRAINT chk_report_evidence_links_scope_type,
    DROP CONSTRAINT chk_report_evidence_links_scope_key,
    DROP CONSTRAINT chk_report_evidence_links_role;

ALTER TABLE report_evidence_links
    ADD CONSTRAINT chk_report_evidence_links_scope_type CHECK (
        scope_type IN (
            'section_summary',
            'anchor',
            'reasoning_step',
            'transmission',
            'industry_chain_summary',
            'industry_chain_node'
        )
    ),
    ADD CONSTRAINT chk_report_evidence_links_scope_key CHECK (
        scope_key ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$'
    ),
    ADD CONSTRAINT chk_report_evidence_links_role CHECK (
        role IN ('direct_target', 'supports_claim', 'supports_reasoning', 'supports_transmission')
    );

COMMENT ON COLUMN report_evidence_links.scope_type IS
    'Evidence 引用的 v2 Report 分析对象类型；不包含页面卡片或空层占位。';

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000080 is a forward-only Report v2 contract cutover; restore the reviewed pre-cutover snapshot with the previous application releases';
END;
$$;
-- +goose StatementEnd
