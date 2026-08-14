-- +goose Up
CREATE TABLE evidence_categories (
    id VARCHAR(32) PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_evidence_categories_id CHECK (id ~ '^EVC_[0-9]{3}$'),
    CONSTRAINT chk_evidence_categories_code CHECK (code ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT chk_evidence_categories_name CHECK (btrim(name) <> ''),
    CONSTRAINT chk_evidence_categories_description CHECK (btrim(description) <> '')
);

CREATE TABLE raw_evidence_category_links (
    raw_evidence_id VARCHAR(32) NOT NULL
        REFERENCES raw_evidences(raw_evidence_id) ON DELETE RESTRICT,
    category_id VARCHAR(32) NOT NULL
        REFERENCES evidence_categories(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (raw_evidence_id, category_id)
);

CREATE INDEX idx_raw_evidence_category_links_category_id
    ON raw_evidence_category_links (category_id);

COMMENT ON TABLE evidence_categories IS
    'Raw Evidence的受控内容分类目录；描述完整材料的内容形态或编辑目的。';
COMMENT ON COLUMN evidence_categories.id IS '稳定分类ID，格式EVC_三位数字。';
COMMENT ON COLUMN evidence_categories.code IS '稳定机器分类编码。';
COMMENT ON COLUMN evidence_categories.name IS '分类中文名称。';
COMMENT ON COLUMN evidence_categories.description IS '分类含义和判断边界。';
COMMENT ON COLUMN evidence_categories.created_at IS '数据库生成的分类创建时间。';

COMMENT ON TABLE raw_evidence_category_links IS
    'Raw Evidence与受控Evidence Category之间的多对多关系。';
COMMENT ON COLUMN raw_evidence_category_links.raw_evidence_id IS '被分类的Raw Evidence ID。';
COMMENT ON COLUMN raw_evidence_category_links.category_id IS '关联的Evidence Category ID。';
COMMENT ON COLUMN raw_evidence_category_links.created_at IS '数据库生成的关系创建时间。';

INSERT INTO evidence_categories (id, code, name, description) VALUES
    ('EVC_001', 'EVENT_BRIEF', '事件快讯', '简短报告已经发生或正在发生的事件，核心目的是说明发生了什么。'),
    ('EVC_002', 'FINANCIAL_REPORT_DATA_SUMMARY', '财报数据摘要', '提炼财务报告中的核心指标、同比环比变化或业绩表现。'),
    ('EVC_003', 'MARKET_MOVEMENT_BRIEF', '行情异动简讯', '简短报告市场价格、收益率、成交量等指标的显著变化，通常只附简要原因。'),
    ('EVC_004', 'MARKET_MOVEMENT_ANALYSIS', '市场异动分析', '分析市场价格或指标变化的原因、传导关系及潜在影响。'),
    ('EVC_005', 'FORECAST_PLAN_OUTLOOK', '预测/计划/展望', '表达对未来的预测、指引、目标、计划或行动展望。'),
    ('EVC_006', 'INDUSTRY_THEME_ANALYSIS', '行业/主题分析', '围绕产业链、行业或投资主题展开具有逻辑链条的分析。'),
    ('EVC_007', 'IN_DEPTH_REPORT', '专题/深度报道', '围绕一个专题进行长篇、多角度调查、梳理或深度报道。'),
    ('EVC_008', 'POLICY_DOCUMENT_SUMMARY', '政策文件摘要', '对正式政策、法规、规划或政府文件进行解读和摘要。'),
    ('EVC_009', 'INTERVIEW_OR_STATEMENT', '人物访谈/表态', '记录特定人物的采访内容、讲话、声明或公开表态。'),
    ('EVC_010', 'SOCIAL_MEDIA_BRIEF', '社交媒体快讯', '来自社交媒体平台的简短发布、评论或即时消息。'),
    ('EVC_011', 'COMMENTARY_EDITORIAL_OPINION', '评论/社论/观点', '以作者或机构主观判断、立场和评价为核心的评论性内容。');

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000047 is forward-only; use a reviewed forward repair';
END;
$$;
-- +goose StatementEnd
