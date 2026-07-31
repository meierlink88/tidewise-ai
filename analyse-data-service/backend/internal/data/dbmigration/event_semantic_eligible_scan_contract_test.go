package dbmigration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEventSemanticEligibleScanMigrationAddsOnlyThePartialAccessPath(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "migrations",
		"000034_add_event_semantic_eligible_scan_index.sql",
	))
	if err != nil {
		t.Fatal(err)
	}
	normalized := strings.ToLower(string(payload))
	for _, required := range []string{
		"create index idx_events_event_semantic_eligible_scan",
		"on events(first_seen_at, id)",
		"event_status = 'confirmed'",
		"fact_status = 'verified'",
		"event_time is not null",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("migration is missing %q", required)
		}
	}
	for _, forbidden := range []string{"add column", "create table"} {
		if strings.Contains(normalized, forbidden) {
			t.Fatalf("migration unexpectedly contains %q", forbidden)
		}
	}
}
