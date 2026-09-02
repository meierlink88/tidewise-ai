-- +goose Up
CREATE TABLE reports (
    id VARCHAR(39) PRIMARY KEY,
    source_report_id VARCHAR(200) NOT NULL UNIQUE,
    contract_version VARCHAR(64) NOT NULL,
    content_hash CHAR(64) NOT NULL,
    content JSONB NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_reports_id CHECK (
        id ~ '^RPT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_reports_source_report_id CHECK (
        source_report_id = btrim(source_report_id) AND source_report_id <> ''
    ),
    CONSTRAINT chk_reports_contract_version CHECK (contract_version = 'report-publication.v1'),
    CONSTRAINT chk_reports_content_hash CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_reports_content_object CHECK (jsonb_typeof(content) = 'object')
);

CREATE INDEX idx_reports_published_at_id
    ON reports (published_at DESC, id ASC);

COMMENT ON TABLE reports IS '外部 AgentOS 发布的不可变完整 Report 快照。';
COMMENT ON COLUMN reports.id IS 'Data 生成的 Report 正式身份，格式为 RPT 加 canonical lowercase UUID。';
COMMENT ON COLUMN reports.source_report_id IS 'AgentOS 为一次推理结果生成的全局唯一、重试稳定身份。';
COMMENT ON COLUMN reports.contract_version IS '固定报告发布包合同版本。';
COMMENT ON COLUMN reports.content_hash IS '对 contract_version 与 canonical content 计算的 lowercase SHA-256。';
COMMENT ON COLUMN reports.content IS '报告完整 typed JSONB 快照，包含持久化卡片、详情、图边和 Evidence 引用。';
COMMENT ON COLUMN reports.published_at IS 'Data 首次成功发布时生成的唯一发布时间事实。';

CREATE TABLE report_evidence_links (
    id VARCHAR(39) PRIMARY KEY,
    report_id VARCHAR(39) NOT NULL REFERENCES reports(id) ON DELETE RESTRICT,
    evidence_id VARCHAR(39) NOT NULL REFERENCES evidences(id) ON DELETE RESTRICT,
    scope_type VARCHAR(32) NOT NULL,
    scope_key VARCHAR(128) NOT NULL,
    role VARCHAR(200) NOT NULL,
    display_order INTEGER NOT NULL,
    CONSTRAINT chk_report_evidence_links_id CHECK (
        id ~ '^RPE[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_report_evidence_links_report_id CHECK (
        report_id ~ '^RPT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_report_evidence_links_evidence_id CHECK (
        evidence_id ~ '^EVD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_report_evidence_links_scope_type CHECK (
        scope_type IN (
            'report_card',
            'layer',
            'anchor',
            'reasoning_step',
            'transmission_path',
            'candidate_mechanism',
            'industry_chain',
            'industry_chain_node'
        )
    ),
    CONSTRAINT chk_report_evidence_links_scope_key CHECK (
        scope_key ~ '^[a-z0-9][a-z0-9._-]{0,127}$'
    ),
    CONSTRAINT chk_report_evidence_links_role CHECK (
        role = btrim(role) AND role <> '' AND char_length(role) <= 200
    ),
    CONSTRAINT chk_report_evidence_links_display_order CHECK (display_order >= 1),
    CONSTRAINT uq_report_evidence_links_scope_evidence
        UNIQUE (report_id, scope_type, scope_key, evidence_id),
    CONSTRAINT uq_report_evidence_links_scope_order
        UNIQUE (report_id, scope_type, scope_key, display_order)
);

CREATE INDEX idx_report_evidence_links_evidence_id
    ON report_evidence_links (evidence_id);

COMMENT ON TABLE report_evidence_links IS 'Report 中某一显式作用域对已有 Atomic Evidence 的直接不可变引用。';
COMMENT ON COLUMN report_evidence_links.id IS 'Data 生成的 Report Evidence Link 正式身份。';
COMMENT ON COLUMN report_evidence_links.report_id IS '所属不可变 Report。';
COMMENT ON COLUMN report_evidence_links.evidence_id IS '直接引用的既有 Atomic Evidence；不经 Event 查询。';
COMMENT ON COLUMN report_evidence_links.scope_type IS 'Evidence 引用的 Report 子对象类型。';
COMMENT ON COLUMN report_evidence_links.scope_key IS 'Evidence 引用所属的 Report-local key。';
COMMENT ON COLUMN report_evidence_links.role IS '报告发布方显式声明的 Evidence 作用。';
COMMENT ON COLUMN report_evidence_links.display_order IS '同一 Report 作用域内从 1 开始连续的发布顺序。';

-- +goose StatementBegin
CREATE FUNCTION prevent_report_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'Report publications are immutable';
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_reports_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON reports
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_report_mutation();

CREATE TRIGGER trg_report_evidence_links_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON report_evidence_links
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_report_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000079 is a forward-only Report publication schema; restore the reviewed pre-cutover snapshot with the previous application releases';
END;
$$;
-- +goose StatementEnd
