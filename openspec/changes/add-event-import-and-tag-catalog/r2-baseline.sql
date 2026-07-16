-- Emit one machine-readable baseline only after r2-preflight.sql has passed.
\set ON_ERROR_STOP on
BEGIN TRANSACTION READ ONLY;
WITH expected(id, tag_kind, code, name, display_order) AS (VALUES
 ('b0fe1994-0db2-526c-a57f-97fa73c1b595','news_category','geopolitics','地缘政治',1),('b1a5438f-6e81-55e7-8ecb-33230b9ae965','news_category','macroeconomy','宏观经济',2),('19fb07c0-aed3-5a1a-99b4-bba004cf2d00','news_category','monetary_policy','货币政策',3),('80f6cb51-38ed-5fcc-8037-3aff25d1b767','news_category','fiscal_trade','财政贸易',4),('06d1e3f4-ba81-5903-80d0-daabb27421af','news_category','usd_fx','美元汇率',5),('80155a2e-33a9-545a-b57e-7bb253af699d','news_category','commodities','大宗商品',6),('2b775f7a-24de-5b44-9fef-dd18f7480148','news_category','market_indices','指数行情',7),('79b73443-5cc4-589b-9dd0-720d2af61e14','news_category','executive_commentary','高层评论',8),('7947aa41-be9c-52ea-816e-8513b6c18d7d','news_category','capital_markets','资本市场',9),('22a5afc5-20ed-55ce-bf77-54c26bbcc6ea','news_category','technology_industry','科技产业',10),
 ('173cabde-c2bf-5cdc-a026-08cd52a953f0','index_category','macro_economic_index','宏观经济指数',1),('71e1deff-56b8-5f70-88ae-fcd4e267c429','index_category','inflation_price_index','通胀物价指数',2),('d9a25979-00e6-5fe4-8807-4ac455d275cd','index_category','interest_credit_index','利率与信用指数',3),('896f457d-3c40-5bad-bb91-3c7f196287c5','index_category','fx_index','外汇汇率指数',4),('87de7402-7632-5a61-8f16-1432f9112c7e','index_category','equity_broad_index','股票宽基指数',5),('22bf6fe5-7b11-5e80-abfa-430713657426','index_category','industry_theme_index','行业主题指数',6),('ba56c6f1-2dfb-5f4c-a769-b95570e0a830','index_category','commodity_index','大宗商品指数',7),('d4616900-4234-578b-9f35-2364c1009634','index_category','market_sentiment_index','市场情绪指数',8),('b67b9650-7460-5708-9c10-089d566682b0','index_category','stock_trading_data','个股与成交数据',9),('4f9ffa47-39c7-5a86-90a4-5ad06d91de4b','index_category','futures_contract','期货合约品种',10),('e95a831e-f852-5838-a739-dbc59726a059','index_category','fund_etf_index','基金与 ETF 指数',11),('6b2cf910-6aa3-5f8d-8016-8e9c0c4a2b09','index_category','options_derivatives','期权与衍生品',12)
), matching AS (
 SELECT count(*) AS n FROM expected e JOIN event_tag_defs d ON d.id::text=e.id AND d.tag_kind=e.tag_kind AND d.code=e.code AND d.name=e.name
)
SELECT json_build_object(
 'source_before', (SELECT count(*) FROM source_catalogs),
 'fixed_source_present_before', (SELECT count(*) FROM source_catalogs WHERE id='cd209afe-2ea9-54b8-bdd7-db64eebf0d71'),
 'tag_defs_before', (SELECT count(*) FROM event_tag_defs),
 'matching_frozen_tuple_count_before', (SELECT n FROM matching),
 'raw_documents_before', (SELECT count(*) FROM raw_documents),
 'events_before', (SELECT count(*) FROM events),
 'event_sources_before', (SELECT count(*) FROM event_sources),
 'event_tag_maps_before', (SELECT count(*) FROM event_tag_maps),
 'event_entity_links_before', (SELECT count(*) FROM event_entity_links),
 'receipt_table_present_before', to_regclass('public.event_import_receipts') IS NOT NULL
)::text AS r2_baseline_json;
COMMIT;
