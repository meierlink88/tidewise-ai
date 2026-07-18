-- +goose Up
ALTER TABLE raw_documents
    ADD COLUMN IF NOT EXISTS content_level VARCHAR(32) NOT NULL DEFAULT '';

ALTER TABLE event_tag_defs
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS display_order INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE IF NOT EXISTS event_import_receipts (
    id UUID PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    package_id TEXT NOT NULL,
    review_id TEXT NOT NULL,
    review_decision VARCHAR(32) NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    event_id UUID NOT NULL REFERENCES events(id),
    raw_document_ids UUID[] NOT NULL,
    event_source_ids UUID[] NOT NULL,
    event_tag_map_ids UUID[] NOT NULL,
    review_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_event_import_receipts_decision CHECK (review_decision IN ('auto_approved', 'pending_evidence', 'manual_review')),
    CONSTRAINT chk_event_import_receipts_payload_hash CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_event_import_receipts_raw_ids CHECK (cardinality(raw_document_ids) >= 1),
    CONSTRAINT chk_event_import_receipts_source_ids CHECK (cardinality(event_source_ids) >= 1),
    CONSTRAINT chk_event_import_receipts_tag_ids CHECK (cardinality(event_tag_map_ids) BETWEEN 1 AND 5),
    CONSTRAINT chk_event_import_receipts_metadata CHECK (jsonb_typeof(review_metadata) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_event_import_receipts_event_id ON event_import_receipts (event_id);
CREATE INDEX IF NOT EXISTS idx_event_import_receipts_package_id ON event_import_receipts (package_id);
CREATE INDEX IF NOT EXISTS idx_event_import_receipts_imported_at ON event_import_receipts (imported_at);

-- The importer resolves this row by UUID. The manifest identity is retained in source_config
-- because source_catalogs has no dedicated manifest identity column.
-- +goose StatementBegin
DO $$
DECLARE
    fixed_source_id UUID := 'cd209afe-2ea9-54b8-bdd7-db64eebf0d71';
    manifest_identity TEXT := 'tidewise:agent:event-reviewed-outbox';
    existing_id UUID;
    existing_manifest TEXT;
BEGIN
    SELECT id, source_config->>'manifest_identity'
      INTO existing_id, existing_manifest
      FROM source_catalogs
     WHERE source_config->>'manifest_identity' = manifest_identity
     LIMIT 1;
    IF existing_id IS NOT NULL AND existing_id <> fixed_source_id THEN
        RAISE EXCEPTION 'event import source manifest identity % is assigned to % instead of %', manifest_identity, existing_id, fixed_source_id;
    END IF;

    SELECT id INTO existing_id FROM source_catalogs WHERE id = fixed_source_id;
    IF existing_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM source_catalogs
         WHERE id = fixed_source_id AND source_config->>'manifest_identity' = manifest_identity
    ) THEN
        RAISE EXCEPTION 'fixed event import source UUID % is occupied by another source', fixed_source_id;
    END IF;
END;
$$;
-- +goose StatementEnd

INSERT INTO source_catalogs (
    id, ingest_channel, provider_key, connector_key, parser_key, source_type,
    source_name, source_url, source_level, topic_hint, auth_required, auth_type,
    credential_ref, usage_policy, source_config, status
) VALUES (
    'cd209afe-2ea9-54b8-bdd7-db64eebf0d71', 'agent_reviewed_outbox', 'tidewise',
    'local_file', 'event_reviewed_outbox', 'event_agent_reviewed_outbox',
    'Event Agent reviewed outbox', '', 'secondary', 'event_import', false, 'none',
    '', 'event_import_reviewed_outbox', '{"manifest_identity":"tidewise:agent:event-reviewed-outbox"}'::jsonb, 'active'
)
ON CONFLICT (id) DO UPDATE SET
    ingest_channel = EXCLUDED.ingest_channel,
    provider_key = EXCLUDED.provider_key,
    connector_key = EXCLUDED.connector_key,
    parser_key = EXCLUDED.parser_key,
    source_type = EXCLUDED.source_type,
    source_name = EXCLUDED.source_name,
    source_level = EXCLUDED.source_level,
    topic_hint = EXCLUDED.topic_hint,
    auth_required = EXCLUDED.auth_required,
    auth_type = EXCLUDED.auth_type,
    credential_ref = EXCLUDED.credential_ref,
    usage_policy = EXCLUDED.usage_policy,
    source_config = EXCLUDED.source_config,
    status = EXCLUDED.status,
    updated_at = now();

-- +goose StatementBegin
DO $$
DECLARE
    expected RECORD;
    actual_id UUID;
BEGIN
    FOR expected IN
        SELECT * FROM (VALUES
            ('b0fe1994-0db2-526c-a57f-97fa73c1b595'::uuid, 'news_category', 1, 'geopolitics', '地缘政治'),
            ('b1a5438f-6e81-55e7-8ecb-33230b9ae965'::uuid, 'news_category', 2, 'macroeconomy', '宏观经济'),
            ('19fb07c0-aed3-5a1a-99b4-bba004cf2d00'::uuid, 'news_category', 3, 'monetary_policy', '货币政策'),
            ('80f6cb51-38ed-5fcc-8037-3aff25d1b767'::uuid, 'news_category', 4, 'fiscal_trade', '财政贸易'),
            ('06d1e3f4-ba81-5903-80d0-daabb27421af'::uuid, 'news_category', 5, 'usd_fx', '美元汇率'),
            ('80155a2e-33a9-545a-b57e-7bb253af699d'::uuid, 'news_category', 6, 'commodities', '大宗商品'),
            ('2b775f7a-24de-5b44-9fef-dd18f7480148'::uuid, 'news_category', 7, 'market_indices', '指数行情'),
            ('79b73443-5cc4-589b-9dd0-720d2af61e14'::uuid, 'news_category', 8, 'executive_commentary', '高层评论'),
            ('7947aa41-be9c-52ea-816e-8513b6c18d7d'::uuid, 'news_category', 9, 'capital_markets', '资本市场'),
            ('22a5afc5-20ed-55ce-bf77-54c26bbcc6ea'::uuid, 'news_category', 10, 'technology_industry', '科技产业'),
            ('173cabde-c2bf-5cdc-a026-08cd52a953f0'::uuid, 'index_category', 1, 'macro_economic_index', '宏观经济指数'),
            ('71e1deff-56b8-5f70-88ae-fcd4e267c429'::uuid, 'index_category', 2, 'inflation_price_index', '通胀物价指数'),
            ('d9a25979-00e6-5fe4-8807-4ac455d275cd'::uuid, 'index_category', 3, 'interest_credit_index', '利率与信用指数'),
            ('896f457d-3c40-5bad-bb91-3c7f196287c5'::uuid, 'index_category', 4, 'fx_index', '外汇汇率指数'),
            ('87de7402-7632-5a61-8f16-1432f9112c7e'::uuid, 'index_category', 5, 'equity_broad_index', '股票宽基指数'),
            ('22bf6fe5-7b11-5e80-abfa-430713657426'::uuid, 'index_category', 6, 'industry_theme_index', '行业主题指数'),
            ('ba56c6f1-2dfb-5f4c-a769-b95570e0a830'::uuid, 'index_category', 7, 'commodity_index', '大宗商品指数'),
            ('d4616900-4234-578b-9f35-2364c1009634'::uuid, 'index_category', 8, 'market_sentiment_index', '市场情绪指数'),
            ('b67b9650-7460-5708-9c10-089d566682b0'::uuid, 'index_category', 9, 'stock_trading_data', '个股与成交数据'),
            ('4f9ffa47-39c7-5a86-90a4-5ad06d91de4b'::uuid, 'index_category', 10, 'futures_contract', '期货合约品种'),
            ('e95a831e-f852-5838-a739-dbc59726a059'::uuid, 'index_category', 11, 'fund_etf_index', '基金与 ETF 指数'),
            ('6b2cf910-6aa3-5f8d-8016-8e9c0c4a2b09'::uuid, 'index_category', 12, 'options_derivatives', '期权与衍生品')
        ) AS t(id, tag_kind, display_order, code, name)
    LOOP
        SELECT id INTO actual_id FROM event_tag_defs WHERE tag_kind = expected.tag_kind AND code = expected.code;
        IF actual_id IS NOT NULL AND actual_id <> expected.id THEN
            RAISE EXCEPTION 'event tag %/% has UUID % instead of %', expected.tag_kind, expected.code, actual_id, expected.id;
        END IF;
        SELECT id INTO actual_id FROM event_tag_defs WHERE id = expected.id;
        IF actual_id IS NOT NULL AND NOT EXISTS (
            SELECT 1 FROM event_tag_defs WHERE id = expected.id AND tag_kind = expected.tag_kind AND code = expected.code
        ) THEN
            RAISE EXCEPTION 'event tag UUID % is occupied by another tag', expected.id;
        END IF;
    END LOOP;
END;
$$;
-- +goose StatementEnd

INSERT INTO event_tag_defs (id, tag_kind, code, name, is_active, display_order, updated_at)
VALUES
('b0fe1994-0db2-526c-a57f-97fa73c1b595','news_category','geopolitics','地缘政治',true,1,now()),
('b1a5438f-6e81-55e7-8ecb-33230b9ae965','news_category','macroeconomy','宏观经济',true,2,now()),
('19fb07c0-aed3-5a1a-99b4-bba004cf2d00','news_category','monetary_policy','货币政策',true,3,now()),
('80f6cb51-38ed-5fcc-8037-3aff25d1b767','news_category','fiscal_trade','财政贸易',true,4,now()),
('06d1e3f4-ba81-5903-80d0-daabb27421af','news_category','usd_fx','美元汇率',true,5,now()),
('80155a2e-33a9-545a-b57e-7bb253af699d','news_category','commodities','大宗商品',true,6,now()),
('2b775f7a-24de-5b44-9fef-dd18f7480148','news_category','market_indices','指数行情',true,7,now()),
('79b73443-5cc4-589b-9dd0-720d2af61e14','news_category','executive_commentary','高层评论',true,8,now()),
('7947aa41-be9c-52ea-816e-8513b6c18d7d','news_category','capital_markets','资本市场',true,9,now()),
('22a5afc5-20ed-55ce-bf77-54c26bbcc6ea','news_category','technology_industry','科技产业',true,10,now()),
('173cabde-c2bf-5cdc-a026-08cd52a953f0','index_category','macro_economic_index','宏观经济指数',true,1,now()),
('71e1deff-56b8-5f70-88ae-fcd4e267c429','index_category','inflation_price_index','通胀物价指数',true,2,now()),
('d9a25979-00e6-5fe4-8807-4ac455d275cd','index_category','interest_credit_index','利率与信用指数',true,3,now()),
('896f457d-3c40-5bad-bb91-3c7f196287c5','index_category','fx_index','外汇汇率指数',true,4,now()),
('87de7402-7632-5a61-8f16-1432f9112c7e','index_category','equity_broad_index','股票宽基指数',true,5,now()),
('22bf6fe5-7b11-5e80-abfa-430713657426','index_category','industry_theme_index','行业主题指数',true,6,now()),
('ba56c6f1-2dfb-5f4c-a769-b95570e0a830','index_category','commodity_index','大宗商品指数',true,7,now()),
('d4616900-4234-578b-9f35-2364c1009634','index_category','market_sentiment_index','市场情绪指数',true,8,now()),
('b67b9650-7460-5708-9c10-089d566682b0','index_category','stock_trading_data','个股与成交数据',true,9,now()),
('4f9ffa47-39c7-5a86-90a4-5ad06d91de4b','index_category','futures_contract','期货合约品种',true,10,now()),
('e95a831e-f852-5838-a739-dbc59726a059','index_category','fund_etf_index','基金与 ETF 指数',true,11,now()),
('6b2cf910-6aa3-5f8d-8016-8e9c0c4a2b09','index_category','options_derivatives','期权与衍生品',true,12,now())
ON CONFLICT (tag_kind, code) DO UPDATE SET
    name = EXCLUDED.name,
    is_active = EXCLUDED.is_active,
    display_order = EXCLUDED.display_order,
    updated_at = EXCLUDED.updated_at;

-- +goose Down
SELECT 'migration 000020 is forward-only; restore the pre-migration backup or use a reviewed forward recovery migration';
