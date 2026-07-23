package dbmigration

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
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
	if len(matches) != 22 {
		t.Fatalf("migration seed tuple count = %d, want 22", len(matches))
	}
	seen := make(map[string]struct{}, len(matches))
	kindCounts := map[string]int{}
	for index, match := range matches {
		order, err := strconv.Atoi(match[6])
		if err != nil {
			t.Fatal(err)
		}
		identity := match[2] + "\x00" + match[3]
		if _, duplicate := seen[identity]; duplicate {
			t.Fatalf("migration seed tuple %d duplicates %q", index, identity)
		}
		seen[identity] = struct{}{}
		kindCounts[match[2]]++
		if match[1] == "" || match[3] == "" || match[4] == "" || match[5] != "true" {
			t.Fatalf("migration seed tuple %d is incomplete: %#v", index, match[1:7])
		}
		wantOrder := kindCounts[match[2]]
		if order != wantOrder {
			t.Fatalf("migration seed tuple %d display order = %d, want %d within %s", index, order, wantOrder, match[2])
		}
	}
	if kindCounts["news_category"] != 10 || kindCounts["index_category"] != 12 {
		t.Fatalf("migration tag kind counts = %#v, want news/index 10/12", kindCounts)
	}
	if strings.Contains(sql, "drop table") || strings.Contains(sql, "truncate") {
		t.Fatal("event import migration must not destructively reset data")
	}
}
