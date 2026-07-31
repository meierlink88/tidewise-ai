package dbmigration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEventSemanticAnchorResolutionMigrationIsForwardOnlyAndAdditive(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000035_add_event_semantic_anchor_resolution.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, fragment := range []string{
		"add column context_manifest jsonb",
		"alter column context_snapshot drop not null",
		"create table event_semantic_resolution_bindings",
		"path_fingerprint char(64)",
		"resolution_receipt jsonb",
		"semantic_submission_id uuid",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("anchor resolution migration missing %q", fragment)
		}
	}
	for _, forbidden := range []string{"update event_semantic_context_leases", "drop column context_snapshot"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("anchor resolution migration contains forbidden historical rewrite %q", forbidden)
		}
	}
}
