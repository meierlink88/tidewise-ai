package dbmigration

import (
	"strings"
	"testing"
)

func TestEventSchemaMigrationIsAdditiveAndConstrained(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000019_add_event_fact_contract.sql"))

	for _, required := range []string{
		"alter table events",
		"add column if not exists fact_payload jsonb not null default '{}'::jsonb",
		"alter table event_sources",
		"add column if not exists evidence_relation varchar(32)",
		"add column if not exists supports_fields text[] not null default '{}'::text[]",
		"chk_event_sources_evidence_relation",
		"evidence_relation in ('supports', 'contradicts', 'context')",
		"alter table event_tag_maps",
		"add column if not exists confidence numeric(5,4)",
		"add column if not exists assignment_reason text not null default ''",
		"chk_event_tag_maps_confidence",
		"confidence >= 0 and confidence <= 1",
		"create unique index if not exists ux_event_sources_event_document_evidence",
		"on event_sources (event_id, raw_document_id, evidence_hash)",
		"migration 000019 is irreversible",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("event schema migration missing %q", required)
		}
	}

	for _, forbidden := range []string{
		"event_entity_links",
		"insert into ",
		"update ",
		"delete from ",
		"truncate ",
		"drop table",
		"drop column",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("event schema migration contains forbidden %q", forbidden)
		}
	}
}
