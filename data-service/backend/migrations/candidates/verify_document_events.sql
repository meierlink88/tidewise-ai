-- Disposable database only: baseline ledger at 93, no business rows.
-- All candidate DDL and synthetic fixtures are rolled back.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL tidewise.document_event_cutover = 'issue-513-reviewed';
INSERT INTO raw_evidences(id, source_id, source_name, source_level, source_url, is_original, raw_text, collected_at)
VALUES ('RAW11111111-1111-4111-8111-111111111111', 'test-source', 'Test source', 'L1_OFFICIAL',
        'https://example.invalid/article', true, '/raw-evidence/test.md', '2026-09-16T00:00:00Z');
\ir document_events.sql

CREATE FUNCTION pg_temp.expect_error(statement TEXT, expected_state TEXT) RETURNS VOID LANGUAGE plpgsql AS $$
BEGIN
    BEGIN
        EXECUTE statement;
    EXCEPTION WHEN OTHERS THEN
        IF SQLSTATE = expected_state THEN RETURN; END IF;
        RAISE;
    END;
    RAISE EXCEPTION 'Expected SQLSTATE %, statement succeeded: %', expected_state, statement;
END $$;

INSERT INTO events(id,title,summary,collected_at,keywords)
VALUES ('EVT11111111-1111-4111-8111-111111111111', repeat('原文事实',100), '完整摘要',
        '2026-09-16T01:00:00Z', ARRAY['12人死亡','上涨1.2%','税率249.13%','约两成','2026年收入100亿元']);
INSERT INTO event_evidence_links(id,event_id,raw_evidence_id)
VALUES ('EEL11111111-1111-4111-8111-111111111111', 'EVT11111111-1111-4111-8111-111111111111', 'RAW11111111-1111-4111-8111-111111111111');
-- Empty semantic is valid and a deferred origin link resolves atomically.
SET CONSTRAINTS ALL IMMEDIATE;
SET CONSTRAINTS ALL DEFERRED;
INSERT INTO event_semantics(id,event_id,actor,action,target,announced_time,effective_time,planned_execution_time,executed_time,statement_type,action_status,assertion_status)
VALUES ('ESM11111111-1111-4111-8111-111111111111','EVT11111111-1111-4111-8111-111111111111',NULL,'建设工厂','新工厂','上周',NULL,'明年第三季度',NULL,'GENERAL','PLANNED','CONFIRMED'),
       ('ESM22222222-2222-4222-8222-222222222222','EVT11111111-1111-4111-8111-111111111111','政府','公布终裁',NULL,'9月11日','10月1日',NULL,NULL,'POLICY','OCCURRED','CONFIRMED'),
       ('ESM33333333-3333-4333-8333-333333333333','EVT11111111-1111-4111-8111-111111111111','某公司','完成收购','另一公司',NULL,NULL,'原定10月1日','实际10月3日','GENERAL','OCCURRED','UNCONFIRMED');

SELECT pg_temp.expect_error($q$UPDATE events SET keywords=ARRAY['1','2','3','4','5','6']$q$,'23514');
SELECT pg_temp.expect_error($q$UPDATE events SET keywords=ARRAY[NULL::text]$q$,'23514');
SELECT pg_temp.expect_error($q$UPDATE event_semantics SET assertion_status='RUMOR'$q$,'23514');
SELECT pg_temp.expect_error($q$UPDATE event_evidence_links SET raw_evidence_id='RAW22222222-2222-4222-8222-222222222222'$q$,'23503');
SELECT pg_temp.expect_error($q$DELETE FROM raw_evidences$q$,'23503');
SELECT pg_temp.expect_error($q$TRUNCATE event_evidence_links$q$,'23514');
SELECT pg_temp.expect_error($q$
 INSERT INTO events(id,title,summary,collected_at) VALUES ('EVT22222222-2222-4222-8222-222222222222','第二条','摘要',now());
 INSERT INTO event_evidence_links(id,event_id,raw_evidence_id) VALUES ('EEL22222222-2222-4222-8222-222222222222','EVT22222222-2222-4222-8222-222222222222','RAW11111111-1111-4111-8111-111111111111');
$q$,'23505');
SELECT pg_temp.expect_error($q$
 DELETE FROM event_evidence_links;
 SET CONSTRAINTS ALL IMMEDIATE;
$q$,'23514');
SELECT pg_temp.expect_error($q$
 INSERT INTO events(id,title,summary,collected_at) VALUES ('EVT33333333-3333-4333-8333-333333333333','无来源','摘要',now());
 SET CONSTRAINTS ALL IMMEDIATE;
$q$,'23514');

INSERT INTO report_archive(id,publisher_report_id,content_hash,report,published_at,event_counts)
VALUES ('RPT11111111-1111-4111-8111-111111111111','test-report',repeat('a',64),'{"event_ids":["EVT11111111-1111-4111-8111-111111111111"]}',now(),'{"/summary/event_ids":1}');
INSERT INTO report_event_links(id,report_id,event_id,scope_type,scope_path,position)
VALUES ('RPE11111111-1111-4111-8111-111111111111','RPT11111111-1111-4111-8111-111111111111','EVT11111111-1111-4111-8111-111111111111','section_summary','/summary/event_ids',1);
SELECT pg_temp.expect_error($q$UPDATE report_event_links SET position=2$q$,'55000');
SELECT pg_temp.expect_error($q$DELETE FROM report_archive$q$,'55000');

DO $$
BEGIN
    IF (SELECT count(*) FROM report_event_links r JOIN events e ON e.id=r.event_id
        JOIN event_evidence_links l ON l.event_id=e.id JOIN raw_evidences raw ON raw.id=l.raw_evidence_id) <> 1 THEN
        RAISE EXCEPTION 'Report -> Event -> Raw provenance failed';
    END IF;
    IF (SELECT published_at IS NOT NULL FROM events LIMIT 1) THEN RAISE EXCEPTION 'unknown news date was invented'; END IF;
    IF to_regclass('evidences') IS NOT NULL OR to_regclass('event_actor_links') IS NOT NULL
       OR to_regclass('event_asset_links') IS NOT NULL THEN RAISE EXCEPTION 'retired tables remain'; END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='event_semantics' AND column_name='position') THEN
        RAISE EXCEPTION 'semantic position is forbidden';
    END IF;
END $$;
SET CONSTRAINTS ALL IMMEDIATE;
ROLLBACK;
SELECT 'PASS document Event persistence and provenance (all changes rolled back)' AS result;
