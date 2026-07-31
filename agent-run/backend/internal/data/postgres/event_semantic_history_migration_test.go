package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestEventSemanticHistoryMigrationAddsOnlySkippedToTheExistingStatus(t *testing.T) {
	payload, err := os.ReadFile("migrations/010_event_semantic_history_skip.sql")
	if err != nil {
		t.Fatal(err)
	}
	normalized := strings.ToLower(string(payload))
	if !strings.Contains(
		normalized,
		"status in ('pending', 'running', 'succeeded', 'failed', 'skipped')",
	) {
		t.Fatalf("migration does not add the history-only skipped state: %s", payload)
	}
	for _, forbidden := range []string{
		"terminal_reason_code", "create table", "add column",
	} {
		if strings.Contains(normalized, forbidden) {
			t.Fatalf("migration unexpectedly contains %q", forbidden)
		}
	}
}
