-- Read-only diagnosis for the failed synthetic event-import integration run.
-- Execute only after separate R2 recovery authorization.
\set ON_ERROR_STOP on
BEGIN TRANSACTION READ ONLY;
DO $$
DECLARE
  version bigint;
  matching_tags int;
  receipt_columns int;
  receipt_constraints int;
BEGIN
  IF current_database() <> 'tidewise_local' OR current_user <> 'tidewise'
     OR host(inet_server_addr()) NOT IN ('127.0.0.1', '::1') THEN
    RAISE EXCEPTION 'recovery diagnosis requires local tidewise_local/tidewise loopback';
  END IF;
  SELECT max(version_id) FILTER (WHERE is_applied) INTO version FROM goose_db_version;
  IF version IS DISTINCT FROM 20 OR to_regclass('public.event_import_receipts') IS NULL THEN
    RAISE EXCEPTION 'recovery diagnosis requires completed 000020 receipt schema, got version %', version;
  END IF;
  SELECT count(*) INTO receipt_columns FROM information_schema.columns WHERE table_schema='public' AND table_name='event_import_receipts';
  SELECT count(*) INTO receipt_constraints FROM pg_constraint WHERE conrelid='event_import_receipts'::regclass;
  IF receipt_columns <> 12 OR receipt_constraints < 9 THEN
    RAISE EXCEPTION 'receipt schema drift: columns %, constraints %', receipt_columns, receipt_constraints;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM source_catalogs WHERE id='cd209afe-2ea9-54b8-bdd7-db64eebf0d71' AND status='active' AND source_config->>'manifest_identity'='tidewise:agent:event-reviewed-outbox') THEN
    RAISE EXCEPTION 'fixed Event Agent source is missing or drifted';
  END IF;
  SELECT count(*) INTO matching_tags FROM event_tag_defs
  WHERE is_active AND id::text IN (
    'b0fe1994-0db2-526c-a57f-97fa73c1b595','b1a5438f-6e81-55e7-8ecb-33230b9ae965','19fb07c0-aed3-5a1a-99b4-bba004cf2d00','80f6cb51-38ed-5fcc-8037-3aff25d1b767','06d1e3f4-ba81-5903-80d0-daabb27421af','80155a2e-33a9-545a-b57e-7bb253af699d','2b775f7a-24de-5b44-9fef-dd18f7480148','79b73443-5cc4-589b-9dd0-720d2af61e14','7947aa41-be9c-52ea-816e-8513b6c18d7d','22a5afc5-20ed-55ce-bf77-54c26bbcc6ea',
    '173cabde-c2bf-5cdc-a026-08cd52a953f0','71e1deff-56b8-5f70-88ae-fcd4e267c429','d9a25979-00e6-5fe4-8807-4ac455d275cd','896f457d-3c40-5bad-bb91-3c7f196287c5','87de7402-7632-5a61-8f16-1432f9112c7e','22bf6fe5-7b11-5e80-abfa-430713657426','ba56c6f1-2dfb-5f4c-a769-b95570e0a830','d4616900-4234-578b-9f35-2364c1009634','b67b9650-7460-5708-9c10-089d566682b0','4f9ffa47-39c7-5a86-90a4-5ad06d91de4b','e95a831e-f852-5838-a739-dbc59726a059','6b2cf910-6aa3-5f8d-8016-8e9c0c4a2b09'
  );
  IF matching_tags <> 22
     OR (SELECT count(*) FROM event_tag_defs WHERE tag_kind='news_category' AND is_active AND id::text IN ('b0fe1994-0db2-526c-a57f-97fa73c1b595','b1a5438f-6e81-55e7-8ecb-33230b9ae965','19fb07c0-aed3-5a1a-99b4-bba004cf2d00','80f6cb51-38ed-5fcc-8037-3aff25d1b767','06d1e3f4-ba81-5903-80d0-daabb27421af','80155a2e-33a9-545a-b57e-7bb253af699d','2b775f7a-24de-5b44-9fef-dd18f7480148','79b73443-5cc4-589b-9dd0-720d2af61e14','7947aa41-be9c-52ea-816e-8513b6c18d7d','22a5afc5-20ed-55ce-bf77-54c26bbcc6ea')) <> 10
     OR (SELECT count(*) FROM event_tag_defs WHERE tag_kind='index_category' AND is_active AND id::text IN ('173cabde-c2bf-5cdc-a026-08cd52a953f0','71e1deff-56b8-5f70-88ae-fcd4e267c429','d9a25979-00e6-5fe4-8807-4ac455d275cd','896f457d-3c40-5bad-bb91-3c7f196287c5','87de7402-7632-5a61-8f16-1432f9112c7e','22bf6fe5-7b11-5e80-abfa-430713657426','ba56c6f1-2dfb-5f4c-a769-b95570e0a830','d4616900-4234-578b-9f35-2364c1009634','b67b9650-7460-5708-9c10-089d566682b0','4f9ffa47-39c7-5a86-90a4-5ad06d91de4b','e95a831e-f852-5838-a739-dbc59726a059','6b2cf910-6aa3-5f8d-8016-8e9c0c4a2b09')) <> 12 THEN
    RAISE EXCEPTION 'expected 22 active frozen tags with 10/12 kind split, got %', matching_tags;
  END IF;
END $$;
WITH synthetic_events AS (SELECT id FROM events WHERE dedupe_key LIKE 'synthetic:event-import-integration-%')
SELECT 'event_import_receipts' AS table_name, count(*) AS row_count FROM event_import_receipts r JOIN synthetic_events e ON e.id=r.event_id WHERE r.idempotency_key LIKE 'event-import-integration-%'
UNION ALL SELECT 'event_tag_maps', count(*) FROM event_tag_maps m JOIN synthetic_events e ON e.id=m.event_id
UNION ALL SELECT 'event_sources', count(*) FROM event_sources s JOIN synthetic_events e ON e.id=s.event_id
UNION ALL SELECT 'events', count(*) FROM synthetic_events
UNION ALL SELECT 'raw_documents', count(*) FROM raw_documents WHERE source_id='cd209afe-2ea9-54b8-bdd7-db64eebf0d71' AND source_external_id LIKE 'synthetic:event-import-integration-%'
ORDER BY table_name;
COMMIT;
