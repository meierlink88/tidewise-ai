-- Read-only R2 preflight. Execute only after explicit R2 authorization.
\set ON_ERROR_STOP on

SELECT current_database() AS database_name, current_user AS database_user,
       inet_server_addr() AS server_addr, inet_server_port() AS server_port,
       current_setting('server_version') AS server_version;

SELECT version_id, is_applied, tstamp
FROM goose_db_version ORDER BY id DESC;

SELECT table_name, column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name IN ('source_catalogs','raw_documents','events','event_sources',
    'event_tag_defs','event_tag_maps','event_import_receipts','event_entity_links')
ORDER BY table_name, ordinal_position;

SELECT conrelid::regclass AS table_name, conname, contype,
       pg_get_constraintdef(oid) AS definition
FROM pg_constraint
WHERE conrelid::regclass::text IN ('source_catalogs','raw_documents','events',
  'event_sources','event_tag_defs','event_tag_maps','event_import_receipts',
  'event_entity_links')
ORDER BY table_name, conname;

SELECT tablename, indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename IN ('source_catalogs','raw_documents','events','event_sources',
    'event_tag_defs','event_tag_maps','event_import_receipts','event_entity_links')
ORDER BY tablename, indexname;

SELECT id, status, ingest_channel, source_type,
       source_config->>'manifest_identity' AS manifest_identity
FROM source_catalogs
WHERE id = 'cd209afe-2ea9-54b8-bdd7-db64eebf0d71'
   OR source_config->>'manifest_identity' = 'tidewise:agent:event-reviewed-outbox'
ORDER BY id;

WITH expected(id, tag_kind, code, name, display_order) AS (VALUES
 ('b0fe1994-0db2-526c-a57f-97fa73c1b595','news_category','geopolitics','地缘政治',1),
 ('b1a5438f-6e81-55e7-8ecb-33230b9ae965','news_category','macroeconomy','宏观经济',2),
 ('19fb07c0-aed3-5a1a-99b4-bba004cf2d00','news_category','monetary_policy','货币政策',3),
 ('80f6cb51-38ed-5fcc-8037-3aff25d1b767','news_category','fiscal_trade','财政贸易',4),
 ('06d1e3f4-ba81-5903-80d0-daabb27421af','news_category','usd_fx','美元汇率',5),
 ('80155a2e-33a9-545a-b57e-7bb253af699d','news_category','commodities','大宗商品',6),
 ('2b775f7a-24de-5b44-9fef-dd18f7480148','news_category','market_indices','指数行情',7),
 ('79b73443-5cc4-589b-9dd0-720d2af61e14','news_category','executive_commentary','高层评论',8),
 ('7947aa41-be9c-52ea-816e-8513b6c18d7d','news_category','capital_markets','资本市场',9),
 ('22a5afc5-20ed-55ce-bf77-54c26bbcc6ea','news_category','technology_industry','科技产业',10),
 ('173cabde-c2bf-5cdc-a026-08cd52a953f0','index_category','macro_economic_index','宏观经济指数',1),
 ('71e1deff-56b8-5f70-88ae-fcd4e267c429','index_category','inflation_price_index','通胀物价指数',2),
 ('d9a25979-00e6-5fe4-8807-4ac455d275cd','index_category','interest_credit_index','利率与信用指数',3),
 ('896f457d-3c40-5bad-bb91-3c7f196287c5','index_category','fx_index','外汇汇率指数',4),
 ('87de7402-7632-5a61-8f16-1432f9112c7e','index_category','equity_broad_index','股票宽基指数',5),
 ('22bf6fe5-7b11-5e80-abfa-430713657426','index_category','industry_theme_index','行业主题指数',6),
 ('ba56c6f1-2dfb-5f4c-a769-b95570e0a830','index_category','commodity_index','大宗商品指数',7),
 ('d4616900-4234-578b-9f35-2364c1009634','index_category','market_sentiment_index','市场情绪指数',8),
 ('b67b9650-7460-5708-9c10-089d566682b0','index_category','stock_trading_data','个股与成交数据',9),
 ('4f9ffa47-39c7-5a86-90a4-5ad06d91de4b','index_category','futures_contract','期货合约品种',10),
 ('e95a831e-f852-5838-a739-dbc59726a059','index_category','fund_etf_index','基金与 ETF 指数',11),
 ('6b2cf910-6aa3-5f8d-8016-8e9c0c4a2b09','index_category','options_derivatives','期权与衍生品',12)
)
SELECT e.id, e.tag_kind, e.code, e.name, e.display_order,
       d.id AS actual_id, d.tag_kind AS actual_kind, d.code AS actual_code,
       d.name AS actual_name, d.display_order AS actual_display_order,
       d.is_active
FROM expected e
LEFT JOIN event_tag_defs d ON d.tag_kind = e.tag_kind AND d.code = e.code
WHERE d.id IS NOT NULL
  AND (d.id::text <> e.id OR d.name <> e.name
       OR d.display_order <> e.display_order OR d.is_active IS NOT TRUE);

SELECT 'source_catalogs' AS table_name, count(*) FROM source_catalogs
UNION ALL SELECT 'raw_documents', count(*) FROM raw_documents
UNION ALL SELECT 'events', count(*) FROM events
UNION ALL SELECT 'event_sources', count(*) FROM event_sources
UNION ALL SELECT 'event_tag_defs', count(*) FROM event_tag_defs
UNION ALL SELECT 'event_tag_maps', count(*) FROM event_tag_maps
UNION ALL SELECT 'event_entity_links', count(*) FROM event_entity_links
ORDER BY table_name;

SELECT (to_regclass('public.event_import_receipts') IS NOT NULL)
       AS receipt_table_present \gset
\if :receipt_table_present
SELECT 'event_import_receipts' AS table_name, count(*) AS row_count
FROM event_import_receipts;
\else
SELECT 'event_import_receipts' AS table_name, 0 AS row_count,
       'table absent before 000020' AS note;
\endif
