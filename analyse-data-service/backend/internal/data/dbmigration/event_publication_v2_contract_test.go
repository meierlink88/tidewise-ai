package dbmigration

import (
	"regexp"
	"strings"
	"testing"
)

const eventPublicationV2Migration = "000029_add_event_publication_v2.sql"

func TestEventPublicationV2MigrationDefinesFrozenSchemaContract(t *testing.T) {
	raw := readMigration(t, eventPublicationV2Migration)
	up, down := migrationSections(t, raw)
	normalized := strings.ToLower(up)

	for _, fragment := range []string{
		"alter table raw_documents",
		"add column contract_version",
		"add column artifact_id",
		"add column source_ref",
		"drop column source_id",
		"ux_raw_documents_artifact_id",
		"chk_raw_documents_v2_contract",
		"alter table event_sources",
		"add column contract_version",
		"add column is_primary",
		"ux_event_sources_v2_event_document",
		"ux_event_sources_v2_primary",
		"chk_event_sources_v2_contract",
		"create table event_publication_receipts",
		"caller_subject",
		"extractor_execution_id",
		"extractor_agent_version",
		"collector_executions",
		"event_ids",
		"raw_document_ids",
		"event_source_ids",
		"event_tag_map_ids",
		"review_metadata",
		"write_counts",
		"create trigger trg_event_publication_receipts_immutable",
		"drop table ingestion_run_sources",
		"drop table ingestion_runs",
		"drop table ingestion_scheduler_configs",
		"drop table source_catalogs",
		"drop table raw_document_import_receipts",
		"drop table event_import_receipts",
		"drop function prevent_raw_document_import_receipt_mutation",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("Event Publication V2 migration Up must contain %q", fragment)
		}
	}

	for _, preserved := range []string{
		"content_text",
		"events",
		"event_sources",
		"event_tag_defs",
		"event_tag_maps",
		"research_themes",
		"research_anchors",
	} {
		if regexp.MustCompile(`(?m)^\s*drop\s+(table|column)\s+(if\s+exists\s+)?` + regexp.QuoteMeta(preserved) + `\b`).MatchString(normalized) {
			t.Errorf("Event Publication V2 migration must preserve %s", preserved)
		}
	}

	businessDML := regexp.MustCompile(`(?mi)^\s*(insert\s+into|update\s+|delete\s+from|truncate\s+)`)
	if match := businessDML.FindString(up); match != "" {
		t.Fatalf("Event Publication V2 migration must contain no business DML, found %q", strings.TrimSpace(match))
	}
	if !strings.Contains(strings.ToLower(down), "migration 000029 is forward-only") || !strings.Contains(strings.ToLower(down), "raise exception") {
		t.Fatal("Event Publication V2 migration Down must fail closed as forward-only")
	}
}
