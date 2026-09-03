-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM reports LIMIT 1) OR
       EXISTS (SELECT 1 FROM report_evidence_links LIMIT 1) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'final Report publication contract requires an empty Report store before applying migration 000081';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE reports
    DROP CONSTRAINT chk_reports_contract_version,
    DROP COLUMN contract_version;

ALTER TABLE reports
    RENAME COLUMN content TO report;

COMMENT ON COLUMN reports.content_hash IS
    '服务端对 immutable report JSON 计算的 lowercase SHA-256；不对 API 暴露。';
COMMENT ON COLUMN reports.report IS
    'AgentOS 发布的不可变、扁平完整 Report JSONB 快照；精确形状由定稿发布 fixture 与 OpenAPI 约束，不从 Event、Signal 或图主数据重建。';

ALTER TABLE report_evidence_links
    DROP CONSTRAINT chk_report_evidence_links_scope_type,
    DROP CONSTRAINT chk_report_evidence_links_scope_key,
    DROP CONSTRAINT chk_report_evidence_links_role,
    DROP CONSTRAINT chk_report_evidence_links_display_order,
    DROP CONSTRAINT uq_report_evidence_links_scope_evidence,
    DROP CONSTRAINT uq_report_evidence_links_scope_order,
    DROP COLUMN role;

ALTER TABLE report_evidence_links
    RENAME COLUMN scope_key TO scope_path;

ALTER TABLE report_evidence_links
    ALTER COLUMN scope_path TYPE VARCHAR(1000);

ALTER TABLE report_evidence_links
    RENAME COLUMN display_order TO position;

ALTER TABLE report_evidence_links
    ADD CONSTRAINT chk_report_evidence_links_scope_type CHECK (
        scope_type IN (
            'section_summary',
            'anchor',
            'reasoning_step',
            'industry_chain_summary',
            'industry_chain_node'
        )
    ),
    ADD CONSTRAINT chk_report_evidence_links_scope_path CHECK (
        scope_path = btrim(scope_path) AND scope_path <> ''
    ),
    ADD CONSTRAINT chk_report_evidence_links_position CHECK (position >= 1),
    ADD CONSTRAINT uq_report_evidence_links_scope_evidence
        UNIQUE (report_id, scope_path, evidence_id),
    ADD CONSTRAINT uq_report_evidence_links_scope_position
        UNIQUE (report_id, scope_path, position);

CREATE INDEX idx_report_evidence_links_report_scope_position
    ON report_evidence_links (report_id, scope_path, position);

COMMENT ON COLUMN report_evidence_links.scope_type IS
    '发布时显式引用 Evidence 的 Report 作用域类型。';
COMMENT ON COLUMN report_evidence_links.scope_path IS
    'Data 从 immutable Report JSON 位置与 report-local key 生成的作用域路径；不对 Miniapp 暴露。';
COMMENT ON COLUMN report_evidence_links.position IS
    '同一 Report Evidence 作用域中从 1 开始的发布顺序。';

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000081 is a forward-only Report contract cutover; restore the reviewed pre-cutover snapshot with matching application releases';
END;
$$;
-- +goose StatementEnd
