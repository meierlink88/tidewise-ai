-- Scoped synthetic-fixture cleanup. Execute only after separate R2 recovery authorization.
\set ON_ERROR_STOP on
BEGIN;
DELETE FROM event_import_receipts r USING events e
WHERE r.event_id=e.id AND r.idempotency_key LIKE 'event-import-integration-%' AND e.dedupe_key LIKE 'synthetic:event-import-integration-%';
DELETE FROM event_tag_maps m USING events e WHERE m.event_id=e.id AND e.dedupe_key LIKE 'synthetic:event-import-integration-%';
DELETE FROM event_sources s USING events e WHERE s.event_id=e.id AND e.dedupe_key LIKE 'synthetic:event-import-integration-%';
DELETE FROM events WHERE dedupe_key LIKE 'synthetic:event-import-integration-%';
DELETE FROM raw_documents WHERE source_id='cd209afe-2ea9-54b8-bdd7-db64eebf0d71' AND source_external_id LIKE 'synthetic:event-import-integration-%';
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM event_import_receipts WHERE idempotency_key LIKE 'event-import-integration-%')
     OR EXISTS (SELECT 1 FROM events WHERE dedupe_key LIKE 'synthetic:event-import-integration-%')
     OR EXISTS (SELECT 1 FROM event_sources s JOIN events e ON e.id=s.event_id WHERE e.dedupe_key LIKE 'synthetic:event-import-integration-%')
     OR EXISTS (SELECT 1 FROM event_tag_maps m JOIN events e ON e.id=m.event_id WHERE e.dedupe_key LIKE 'synthetic:event-import-integration-%')
     OR EXISTS (SELECT 1 FROM raw_documents WHERE source_id='cd209afe-2ea9-54b8-bdd7-db64eebf0d71' AND source_external_id LIKE 'synthetic:event-import-integration-%') THEN
    RAISE EXCEPTION 'synthetic event-import integration cleanup left residual rows';
  END IF;
END $$;
COMMIT;
