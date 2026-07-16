package dbmigration

import (
	"strings"
	"testing"

	domainimport "github.com/meierlink88/tidewise-ai/backend/internal/domain/eventimport"
)

func TestEventImportMigrationFreezesSourceReceiptAndTagSeed(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000020_add_event_import_receipts_and_tag_seed.sql"))
	for _, required := range []string{
		"add column if not exists content_level varchar(32)",
		"create table if not exists event_import_receipts",
		"idempotency_key text not null unique",
		"payload_hash char(64) not null",
		"event_id uuid not null references events(id)",
		"chk_event_import_receipts_payload_hash",
		"cd209afe-2ea9-54b8-bdd7-db64eebf0d71",
		"manifest_identity",
		"insert into event_tag_defs",
		"on conflict (tag_kind, code) do update",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
	if len(domainimport.FrozenTags) != 22 {
		t.Fatalf("frozen tag fixture count = %d, want 22", len(domainimport.FrozenTags))
	}
	for _, tag := range domainimport.FrozenTags {
		if !strings.Contains(sql, "'"+tag.ID+"'") || !strings.Contains(sql, "'"+tag.Code+"'") || !strings.Contains(sql, "'"+strings.ToLower(tag.Name)+"'") {
			t.Fatalf("migration does not match frozen tag fixture %s/%s", tag.Kind, tag.Code)
		}
	}
	if strings.Contains(sql, "drop table") || strings.Contains(sql, "truncate") {
		t.Fatal("event import migration must not destructively reset data")
	}
}
