package dbmigration

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	postgresfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/postgres"
)

func TestEventPublicationMigrationPreservesHistoricalEvidenceContent(t *testing.T) {
	db := openEventPublicationTestDatabaseAt(t, 28)
	const (
		rawID          = "11111111-1111-4111-8111-111111111111"
		eventID        = "22222222-2222-4222-8222-222222222222"
		sourceID       = "33333333-3333-4333-8333-333333333333"
		themeReceiptID = "44444444-4444-4444-8444-444444444444"
		themeID        = "55555555-5555-4555-8555-555555555555"
		anchorReceipt  = "66666666-6666-4666-8666-666666666666"
		anchorID       = "77777777-7777-4777-8777-777777777777"
		chainNodeOne   = "88888888-8888-4888-8888-888888888888"
		chainNodeTwo   = "99999999-9999-4999-8999-999999999999"
	)
	if _, err := db.Exec(`
INSERT INTO raw_documents (
  id, source_id, ingest_channel, source_type, source_name, source_url, title,
  content_text, raw_mime_type, language, collected_at, content_hash, ingest_status
) VALUES (
  $1, 'cd209afe-2ea9-54b8-bdd7-db64eebf0d71', 'legacy', 'news', 'Legacy Source',
  'https://example.test/legacy', 'Legacy document', 'historical full content',
  'text/markdown', 'en', '2026-07-22T00:00:00Z', $2, 'collected'
)`, rawID, fmt.Sprintf("%064x", 7)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO events (
  id, title, summary, first_seen_at, event_status, fact_status, dedupe_key, fact_payload
) VALUES (
  $1, 'Historical Event', 'Historical summary', '2026-07-22T00:00:00Z',
  'confirmed', 'verified', 'event:historical', '{"legacy":true}'::jsonb
)`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO event_sources (
  id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
  evidence_relation, supports_fields
) VALUES (
  $1, $2, $3, 'primary', 'Historical excerpt', $4, 'supports', ARRAY['title']
)`, sourceID, eventID, rawID, fmt.Sprintf("%064x", 8)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE events SET primary_source_id = $2 WHERE id = $1`, eventID, sourceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO entity_nodes (
  id, entity_key, entity_type, layer_code, name, canonical_name, status
) VALUES
  ($1, 'chain_node:migration-preservation-one', 'chain_node', 'chain_node',
   'Migration Preservation One', 'Migration Preservation One', 'active'),
  ($2, 'chain_node:migration-preservation-two', 'chain_node', 'chain_node',
   'Migration Preservation Two', 'Migration Preservation Two', 'active')`,
		chainNodeOne, chainNodeTwo,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO chain_node_profiles (entity_id, definition, boundary_note, review_status)
VALUES
  ($1, 'First preservation node', 'First preservation boundary', 'approved'),
  ($2, 'Second preservation node', 'Second preservation boundary', 'approved')`,
		chainNodeOne, chainNodeTwo,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_theme_import_receipts (
  id, analysis_batch_id, publisher_subject, payload_hash, theme_ids_by_key,
  write_counts, published_at, imported_at
) VALUES (
  $1, 'migration-preservation-batch', 'migration-test', $2,
  jsonb_build_object('migration-preservation-theme', $3::text),
  '{"themes":1,"chain_node_associations":1,"event_associations":1,"receipts":1}'::jsonb,
  '2026-07-22T01:00:00Z', '2026-07-22T01:00:01Z'
)`, themeReceiptID, fmt.Sprintf("%064x", 9), themeID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_themes (
  id, analysis_batch_id, theme_key, import_receipt_id, name, one_line_conclusion,
  impact_level, transmission_path, trading_direction, transmission_stage,
  next_checkpoint, market_confirmation_summary, window_start, window_end, published_at
) VALUES (
  $1, 'migration-preservation-batch', 'migration-preservation-theme', $2,
  'Preserved Theme', 'Preserved theme conclusion', 'high',
  'Event to node transmission', 'Track the preserved direction', 'validation',
  'Verify the next checkpoint', 'Market evidence remains preserved',
  '2026-07-21T00:00:00Z', '2026-07-22T00:00:00Z', '2026-07-22T01:00:00Z'
)`, themeID, themeReceiptID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_theme_chain_nodes (
  theme_id, chain_node_entity_id, relation_role, impact_summary
) VALUES ($1, $2, 'driver', 'Preserved Theme node association')`,
		themeID, chainNodeOne,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_theme_events (theme_id, event_id, evidence_role, supported_claim)
VALUES ($1, $2, 'driver', 'Preserved Theme Event claim')`, themeID, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchor_import_receipts (
  id, theme_id, publisher_subject, payload_hash, anchor_ids_by_center_chain_node_id,
  write_counts, published_at, imported_at
) VALUES (
  $1, $2, 'migration-test', $3,
  jsonb_build_object($4::text, $5::text),
  '{"anchors":1,"event_associations":1,"path_nodes":2,"receipts":1}'::jsonb,
  '2026-07-22T01:00:00Z', '2026-07-22T01:00:01Z'
)`, anchorReceipt, themeID, fmt.Sprintf("%064x", 10), chainNodeOne, anchorID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchors (
  id, theme_id, center_chain_node_entity_id, import_receipt_id,
  one_line_conclusion, fact_summary, net_direction_summary, trading_direction,
  next_checkpoint, support_summary, counter_summary
) VALUES (
  $1, $2, $3, $4, 'Preserved Anchor conclusion', 'Preserved facts',
  'Preserved net direction', 'Preserved trading direction',
  'Preserved checkpoint', 'Preserved support summary', 'Preserved counter summary'
)`, anchorID, themeID, chainNodeOne, anchorReceipt); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchor_chain_nodes (
  anchor_id, position, chain_node_entity_id, change_direction,
  change_summary, impact_summary, incoming_transmission_mechanism
) VALUES
  ($1, 1, $2, 'increase', 'First preserved change', 'First preserved impact', NULL),
  ($1, 2, $3, 'increase', 'Second preserved change', 'Second preserved impact',
   'Preserved transmission mechanism')`,
		anchorID, chainNodeOne, chainNodeTwo,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchor_events (anchor_id, event_id, evidence_role, evidence_summary)
VALUES ($1, $2, 'driver', 'Preserved Anchor Event summary')`, anchorID, eventID); err != nil {
		t.Fatal(err)
	}

	applyEventPublicationMigration(t, db, 29)

	var content string
	var contractVersion int
	var artifactID *string
	if err := db.QueryRow(`
SELECT content_text, contract_version, artifact_id
FROM raw_documents WHERE id = $1`, rawID).Scan(&content, &contractVersion, &artifactID); err != nil {
		t.Fatal(err)
	}
	if content != "historical full content" || contractVersion != 1 || artifactID != nil {
		t.Fatalf("historical raw document = content %q, version %d, artifact %v", content, contractVersion, artifactID)
	}
	var linked int
	if err := db.QueryRow(`
SELECT count(*)
FROM events e
JOIN event_sources es ON es.id = e.primary_source_id AND es.event_id = e.id
JOIN raw_documents rd ON rd.id = es.raw_document_id
WHERE e.id = $1 AND rd.id = $2`, eventID, rawID).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if linked != 1 {
		t.Fatalf("historical Event evidence link count = %d, want 1", linked)
	}
	var preservedResearchRows int
	if err := db.QueryRow(`
SELECT
  (SELECT count(*) FROM research_themes
    WHERE id = $1 AND name = 'Preserved Theme')
  + (SELECT count(*) FROM research_theme_chain_nodes
    WHERE theme_id = $1 AND chain_node_entity_id = $3)
  + (SELECT count(*) FROM research_theme_events
    WHERE theme_id = $1 AND event_id = $2 AND supported_claim = 'Preserved Theme Event claim')
  + (SELECT count(*) FROM research_anchors
    WHERE id = $4 AND theme_id = $1 AND one_line_conclusion = 'Preserved Anchor conclusion')
  + (SELECT count(*) FROM research_anchor_chain_nodes
    WHERE anchor_id = $4 AND position IN (1, 2))
  + (SELECT count(*) FROM research_anchor_events
    WHERE anchor_id = $4 AND event_id = $2 AND evidence_summary = 'Preserved Anchor Event summary')`,
		themeID, eventID, chainNodeOne, anchorID,
	).Scan(&preservedResearchRows); err != nil {
		t.Fatal(err)
	}
	if preservedResearchRows != 7 {
		t.Fatalf("preserved Research row count = %d, want 7", preservedResearchRows)
	}
	for _, table := range []string{
		"events", "event_sources", "event_tag_defs", "event_tag_maps", "research_themes", "research_anchors",
	} {
		if !relationExists(t, db, table) {
			t.Fatalf("preserved table %s does not exist", table)
		}
	}
	for _, table := range []string{
		"source_catalogs", "ingestion_run_sources", "ingestion_runs",
		"ingestion_scheduler_configs", "raw_document_import_receipts", "event_import_receipts",
	} {
		if relationExists(t, db, table) {
			t.Fatalf("retired table %s still exists", table)
		}
	}
}

func TestEventPublicationV3MigrationPreservesEvidenceAndRemovesPrimarySemantics(t *testing.T) {
	db := openEventPublicationTestDatabaseAt(t, 37)
	const (
		rawID    = "a1000000-0000-4000-8000-000000000001"
		eventID  = "a1000000-0000-4000-8000-000000000002"
		sourceID = "a1000000-0000-4000-8000-000000000003"
	)
	if _, err := db.Exec(`
INSERT INTO raw_documents (
  id, contract_version, artifact_id, source_ref, source_type,
  source_name, source_url, title, content_text, raw_mime_type, language,
  collected_at, content_hash, ingest_status
) VALUES (
  $1, 2, 'artifact:migration-v3', 'source:migration-v3', 'news',
  'Migration Source', 'https://example.test/migration-v3', 'Migration V3 source',
  '', 'text/markdown', 'en', '2026-08-02T00:00:00Z', $2, 'collected'
)`, rawID, strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO events (
  id, title, summary, first_seen_at, event_status, fact_status, dedupe_key, fact_payload
) VALUES (
  $1, 'Migration V3 Event', 'Migration V3 summary', '2026-08-02T00:00:00Z',
  'confirmed', 'verified', 'event:migration-v3', '{}'::jsonb
)`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO event_sources (
  id, contract_version, event_id, raw_document_id, source_level,
  evidence_excerpt, evidence_hash, evidence_relation, supports_fields, is_primary
) VALUES (
  $1, 2, $2, $3, 'primary', 'Preserved evidence statement', $4,
  'supports', ARRAY['title'], true
)`, sourceID, eventID, rawID, strings.Repeat("b", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`UPDATE events SET primary_source_id = $2 WHERE id = $1`, eventID, sourceID,
	); err != nil {
		t.Fatal(err)
	}

	applyEventPublicationMigration(t, db, 38)

	var statement string
	var contractVersion int
	if err := db.QueryRow(`
SELECT evidence_statement, contract_version FROM event_sources WHERE id = $1
`, sourceID).Scan(&statement, &contractVersion); err != nil {
		t.Fatal(err)
	}
	if statement != "Preserved evidence statement" || contractVersion != 3 {
		t.Fatalf("migrated evidence = %q, version = %d", statement, contractVersion)
	}
	var retiredColumns int
	if err := db.QueryRow(`
SELECT count(*) FROM information_schema.columns
WHERE table_schema = current_schema()
  AND ((table_name = 'events' AND column_name = 'primary_source_id')
    OR (table_name = 'event_sources' AND column_name = 'is_primary'))
`).Scan(&retiredColumns); err != nil {
		t.Fatal(err)
	}
	if retiredColumns != 0 {
		t.Fatalf("retired primary Evidence columns remaining = %d", retiredColumns)
	}
}

func openEventPublicationTestDatabaseAt(t *testing.T, version int64) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_event_publication", migrationDir, version)
}

func applyEventPublicationMigration(t *testing.T, db *sql.DB, version int64) {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	postgresfixture.ApplyMigration(t, db, migrationDir, version)
}

func relationExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, name).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}
