-- +goose Up
CREATE TABLE raw_evidences (
    raw_evidence_id VARCHAR(32) PRIMARY KEY,
    source_id VARCHAR(32) NOT NULL,
    source_name VARCHAR(100) NOT NULL,
    source_level VARCHAR(20) NOT NULL,
    source_url TEXT NOT NULL,
    is_original BOOLEAN NOT NULL,
    quoted_source_id VARCHAR(32),
    quoted_source_name VARCHAR(100),
    title VARCHAR(500),
    raw_text TEXT NOT NULL,
    published_at TIMESTAMPTZ,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    content_hash VARCHAR(64) GENERATED ALWAYS AS (
        encode(sha256(raw_text::bytea), 'hex')
    ) STORED,
    keywords TEXT[] NOT NULL DEFAULT '{}',
    CONSTRAINT chk_raw_evidences_id CHECK (btrim(raw_evidence_id) <> ''),
    CONSTRAINT chk_raw_evidences_source_id CHECK (btrim(source_id) <> ''),
    CONSTRAINT chk_raw_evidences_source_name CHECK (btrim(source_name) <> ''),
    CONSTRAINT chk_raw_evidences_source_level CHECK (
        source_level IN ('L1_OFFICIAL', 'L2_WIRE', 'L3_MEDIA', 'L4_SOCIAL')
    ),
    CONSTRAINT chk_raw_evidences_source_url CHECK (btrim(source_url) <> ''),
    CONSTRAINT chk_raw_evidences_title CHECK (title IS NULL OR btrim(title) <> ''),
    CONSTRAINT chk_raw_evidences_raw_text CHECK (btrim(raw_text) <> ''),
    CONSTRAINT chk_raw_evidences_origin CHECK (
        (is_original AND quoted_source_id IS NULL AND quoted_source_name IS NULL)
        OR (NOT is_original AND quoted_source_name IS NOT NULL AND btrim(quoted_source_name) <> '')
    ),
    CONSTRAINT chk_raw_evidences_keywords_one_dimension CHECK (
        COALESCE(array_ndims(keywords), 1) = 1
    )
);

COMMENT ON TABLE raw_evidences IS '完整原始文章或原始证据；由采集发布方提交，Data Service 权威保存。';
COMMENT ON COLUMN raw_evidences.raw_evidence_id IS '发布方提供的 Raw Evidence 稳定自然身份。';
COMMENT ON COLUMN raw_evidences.source_id IS '原始材料的信源 ID。';
COMMENT ON COLUMN raw_evidences.source_name IS '原始材料的信源名称。';
COMMENT ON COLUMN raw_evidences.source_level IS '信源等级：L1_OFFICIAL/L2_WIRE/L3_MEDIA/L4_SOCIAL。';
COMMENT ON COLUMN raw_evidences.source_url IS '原始材料链接。';
COMMENT ON COLUMN raw_evidences.is_original IS '当前信源是否为原创来源。';
COMMENT ON COLUMN raw_evidences.quoted_source_id IS '转载材料的上游信源 ID。';
COMMENT ON COLUMN raw_evidences.quoted_source_name IS '转载材料的上游信源名称。';
COMMENT ON COLUMN raw_evidences.title IS '原始材料标题。';
COMMENT ON COLUMN raw_evidences.raw_text IS '完整原文正文。';
COMMENT ON COLUMN raw_evidences.published_at IS '文章发布时间；不得替代 Evidence 事实发生时间。';
COMMENT ON COLUMN raw_evidences.collected_at IS '采集完成时间。';
COMMENT ON COLUMN raw_evidences.content_hash IS 'Data 从 raw_text 生成的 SHA-256。';
COMMENT ON COLUMN raw_evidences.keywords IS '发布方生成的阅读辅助关键词；Data 原样有序保存。';

CREATE TABLE evidences (
    evidence_id VARCHAR(32) PRIMARY KEY,
    raw_evidence_id VARCHAR(32) NOT NULL REFERENCES raw_evidences(raw_evidence_id),
    split_order INTEGER NOT NULL,
    is_split BOOLEAN NOT NULL,
    layer_type VARCHAR(10) NOT NULL,
    source_who TEXT,
    source_what TEXT NOT NULL,
    source_when TIMESTAMPTZ,
    source_when_raw TEXT,
    source_where TEXT,
    source_why TEXT,
    source_how TEXT,
    source_who_core TEXT,
    source_what_core TEXT,
    source_when_core TIMESTAMPTZ,
    source_when_raw_core TEXT,
    source_where_core TEXT,
    source_why_core TEXT,
    source_how_core TEXT,
    expression_fingerprint VARCHAR(200) NOT NULL,
    expression_key VARCHAR(64) NOT NULL,
    fingerprint_version VARCHAR(64) NOT NULL,
    CONSTRAINT uq_evidences_raw_split_order UNIQUE (raw_evidence_id, split_order),
    CONSTRAINT chk_evidences_id CHECK (btrim(evidence_id) <> ''),
    CONSTRAINT chk_evidences_split_order CHECK (split_order >= 0),
    CONSTRAINT chk_evidences_layer_type CHECK (layer_type IN ('SINGLE', 'DOUBLE')),
    CONSTRAINT chk_evidences_source_what CHECK (btrim(source_what) <> ''),
    CONSTRAINT chk_evidences_expression_fingerprint CHECK (btrim(expression_fingerprint) <> ''),
    CONSTRAINT chk_evidences_expression_key CHECK (btrim(expression_key) <> ''),
    CONSTRAINT chk_evidences_fingerprint_version CHECK (btrim(fingerprint_version) <> ''),
    CONSTRAINT chk_evidences_layer_fields CHECK (
        (layer_type = 'SINGLE'
            AND source_who_core IS NULL
            AND source_what_core IS NULL
            AND source_when_core IS NULL
            AND source_when_raw_core IS NULL
            AND source_where_core IS NULL
            AND source_why_core IS NULL
            AND source_how_core IS NULL)
        OR (layer_type = 'DOUBLE'
            AND source_what_core IS NOT NULL
            AND btrim(source_what_core) <> '')
    )
);

CREATE INDEX idx_evidences_expression_key ON evidences (expression_key);
CREATE INDEX idx_evidences_raw_evidence_id ON evidences (raw_evidence_id);

COMMENT ON TABLE evidences IS '从 Raw Evidence 清洗得到的、可直接消费的原子 Evidence。';
COMMENT ON COLUMN evidences.evidence_id IS '发布方提供的 Evidence 稳定身份。';
COMMENT ON COLUMN evidences.raw_evidence_id IS '父 Raw Evidence 身份。';
COMMENT ON COLUMN evidences.split_order IS '同一 Raw Evidence 内从 0 连续的拆分顺序。';
COMMENT ON COLUMN evidences.is_split IS 'Data 根据完整发布集合基数派生；一对一为 false，一对多为 true。';
COMMENT ON COLUMN evidences.layer_type IS '5W1H 层级：SINGLE 或 DOUBLE。';
COMMENT ON COLUMN evidences.source_who IS '第一层 5W1H：谁。';
COMMENT ON COLUMN evidences.source_what IS '第一层 5W1H：发生了什么。';
COMMENT ON COLUMN evidences.source_when IS '第一层结构化事实发生时间。';
COMMENT ON COLUMN evidences.source_when_raw IS '第一层原始时间表达。';
COMMENT ON COLUMN evidences.source_where IS '第一层 5W1H：何地。';
COMMENT ON COLUMN evidences.source_why IS '第一层 5W1H：为何。';
COMMENT ON COLUMN evidences.source_how IS '第一层 5W1H：如何、程度或数量。';
COMMENT ON COLUMN evidences.source_who_core IS '第二层核心 5W1H：谁。';
COMMENT ON COLUMN evidences.source_what_core IS '第二层核心 5W1H：发生了什么。';
COMMENT ON COLUMN evidences.source_when_core IS '第二层核心结构化事实发生时间。';
COMMENT ON COLUMN evidences.source_when_raw_core IS '第二层核心原始时间表达。';
COMMENT ON COLUMN evidences.source_where_core IS '第二层核心 5W1H：何地。';
COMMENT ON COLUMN evidences.source_why_core IS '第二层核心 5W1H：为何。';
COMMENT ON COLUMN evidences.source_how_core IS '第二层核心 5W1H：如何、程度或数量。';
COMMENT ON COLUMN evidences.expression_fingerprint IS '确定性规范化后的可读原子事实表达。';
COMMENT ON COLUMN evidences.expression_key IS '发布方提供的稳定机器去重键；允许多条 Evidence 共享。';
COMMENT ON COLUMN evidences.fingerprint_version IS '表达规范化或哈希算法版本。';

CREATE TABLE raw_evidence_publication_receipts (
    id UUID PRIMARY KEY,
    contract_version SMALLINT NOT NULL DEFAULT 1,
    caller_subject TEXT NOT NULL,
    raw_evidence_id VARCHAR(32) NOT NULL REFERENCES raw_evidences(raw_evidence_id),
    disposition TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_raw_evidence_receipts_version CHECK (contract_version = 1),
    CONSTRAINT chk_raw_evidence_receipts_caller CHECK (btrim(caller_subject) <> ''),
    CONSTRAINT chk_raw_evidence_receipts_disposition CHECK (disposition IN ('created', 'reused'))
);

CREATE INDEX idx_raw_evidence_receipts_identity
    ON raw_evidence_publication_receipts (raw_evidence_id, imported_at DESC);

CREATE TABLE evidence_publication_receipts (
    id UUID PRIMARY KEY,
    contract_version SMALLINT NOT NULL DEFAULT 1,
    caller_subject TEXT NOT NULL,
    raw_evidence_id VARCHAR(32) NOT NULL REFERENCES raw_evidences(raw_evidence_id),
    evidence_ids VARCHAR(32)[] NOT NULL,
    created_count INTEGER NOT NULL,
    reused_count INTEGER NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_evidence_receipts_version CHECK (contract_version = 1),
    CONSTRAINT chk_evidence_receipts_caller CHECK (btrim(caller_subject) <> ''),
    CONSTRAINT chk_evidence_receipts_ids CHECK (
        COALESCE(array_ndims(evidence_ids), 1) = 1
        AND cardinality(evidence_ids) >= 1
        AND array_position(evidence_ids, NULL::VARCHAR) IS NULL
    ),
    CONSTRAINT chk_evidence_receipts_counts CHECK (
        created_count >= 0
        AND reused_count >= 0
        AND created_count + reused_count = cardinality(evidence_ids)
        AND (created_count = 0 OR reused_count = 0)
    )
);

CREATE INDEX idx_evidence_receipts_identity
    ON evidence_publication_receipts (raw_evidence_id, imported_at DESC);

-- +goose StatementBegin
CREATE FUNCTION prevent_evidence_publication_receipt_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'Evidence Publication receipts are immutable';
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_raw_evidence_publication_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON raw_evidence_publication_receipts
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_evidence_publication_receipt_mutation();

CREATE TRIGGER trg_evidence_publication_receipts_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON evidence_publication_receipts
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_evidence_publication_receipt_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000042 is forward-only; restore the prior revision with a reviewed backup';
END;
$$;
-- +goose StatementEnd
