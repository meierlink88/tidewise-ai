package dbmigration

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventimport"
)

func TestEventImportMigrationFreezesSourceReceiptAndTagSeed(t *testing.T) {
	rawSQL := readMigration(t, "000020_add_event_import_receipts_and_tag_seed.sql")
	sql := strings.ToLower(rawSQL)
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
	insertStart := strings.Index(rawSQL, "INSERT INTO event_tag_defs")
	if insertStart < 0 {
		t.Fatal("migration tag seed INSERT/ON CONFLICT block is missing")
	}
	conflictStart := strings.Index(rawSQL[insertStart:], "ON CONFLICT")
	if conflictStart < 0 {
		t.Fatal("migration tag seed ON CONFLICT block is missing")
	}
	seedSQL := rawSQL[insertStart : insertStart+conflictStart]
	tuplePattern := regexp.MustCompile(`\('([^']+)'\s*,\s*'([^']+)'\s*,\s*'([^']+)'\s*,\s*'([^']+)'\s*,\s*(true|false)\s*,\s*([0-9]+)\s*,\s*now\(\)\)`)
	matches := tuplePattern.FindAllStringSubmatch(seedSQL, -1)
	if len(matches) != len(domainimport.FrozenTags) {
		t.Fatalf("migration seed tuple count = %d, want %d", len(matches), len(domainimport.FrozenTags))
	}
	for index, match := range matches {
		order, err := strconv.Atoi(match[6])
		if err != nil {
			t.Fatal(err)
		}
		expected := domainimport.FrozenTags[index]
		if match[1] != expected.ID || match[2] != expected.Kind || match[3] != expected.Code || match[4] != expected.Name || match[5] != "true" || order != expected.DisplayOrder {
			t.Fatalf("migration seed tuple %d = %#v, want %#v", index, match[1:7], expected)
		}
	}
	if strings.Contains(sql, "drop table") || strings.Contains(sql, "truncate") {
		t.Fatal("event import migration must not destructively reset data")
	}
}
