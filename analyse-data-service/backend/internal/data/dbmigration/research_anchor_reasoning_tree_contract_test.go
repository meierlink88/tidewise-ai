package dbmigration

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResearchAnchorReasoningTreeMigrationDefinesFrozenSchema(t *testing.T) {
	content := strings.ToLower(readMigration(t, filepath.Join(migrationDirectory(), "000025_rebuild_research_anchor_reasoning_trees.sql")))
	parts := strings.Split(content, "-- +goose down")
	if len(parts) != 2 {
		t.Fatal("reasoning tree migration must define exactly one goose Down section")
	}
	up, down := parts[0], parts[1]
	normalizedUp := strings.Join(strings.Fields(up), " ")

	for _, fragment := range []string{
		"do $$",
		"research_anchors",
		"research_anchor_chain_nodes",
		"research_anchor_events",
		"research_anchor_indices",
		"raise exception",
		"drop table research_anchor_indices",
		"drop table research_anchor_events",
		"drop table research_anchor_chain_nodes",
		"drop table research_anchors",
		"create table research_anchor_import_receipts",
		"theme_id uuid not null unique references research_themes(id)",
		"publisher_subject text not null",
		"payload_hash char(64) not null",
		"anchor_ids_by_center_chain_node_id jsonb not null",
		`@.key like_regex "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"`,
		`@.value like_regex "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"`,
		"write_counts jsonb not null",
		"unique (id, theme_id)",
		"create table research_anchors",
		"center_chain_node_entity_id uuid not null references chain_node_profiles(entity_id)",
		"foreign key (import_receipt_id, theme_id)",
		"references research_anchor_import_receipts(id, theme_id)",
		"unique (theme_id, center_chain_node_entity_id)",
		"create table research_anchor_chain_nodes",
		"position integer not null",
		"primary key (anchor_id, position)",
		"unique (anchor_id, chain_node_entity_id)",
		"change_direction in ('increase', 'decrease', 'mixed', 'unchanged', 'uncertain')",
		"position = 1 and incoming_transmission_mechanism is null",
		"position > 1 and btrim(incoming_transmission_mechanism) <> ''",
		"create table research_anchor_events",
		"primary key (anchor_id, event_id)",
		"evidence_role in ('driver', 'supporting', 'contradicting', 'context')",
		"create function prevent_research_anchor_import_receipt_mutation",
		"create trigger trg_research_anchor_import_receipts_immutable",
	} {
		if !strings.Contains(normalizedUp, fragment) {
			t.Errorf("reasoning tree migration Up must contain %q", fragment)
		}
	}

	if strings.Contains(normalizedUp, "create table research_anchor_indices") {
		t.Fatal("reasoning tree migration must not rebuild research_anchor_indices")
	}
	if strings.Contains(normalizedUp, "create index idx_research_anchors_theme_id") {
		t.Fatal("reasoning tree migration must rely on the theme/center unique index instead of creating a duplicate index")
	}
	for _, legacyColumn := range []string{
		"anchor_type text",
		"importance text",
		"analysis_batch_id text",
		"transmission_path text",
		"updated_at timestamptz",
	} {
		if strings.Contains(normalizedUp, legacyColumn) {
			t.Errorf("reasoning tree migration must not recreate legacy column %q", legacyColumn)
		}
	}
	if !strings.Contains(down, "migration 000025 is forward-only") || !strings.Contains(down, "raise exception") {
		t.Fatal("reasoning tree migration Down must fail closed as forward-only")
	}
}

func TestResearchAnchorReasoningTreeMigrationChecksLegacyTablesBeforeDDL(t *testing.T) {
	content := strings.ToLower(readMigration(t, filepath.Join(migrationDirectory(), "000025_rebuild_research_anchor_reasoning_trees.sql")))
	lock := strings.Index(content, "lock table")
	assertion := strings.Index(content, "do $$")
	firstDDL := strings.Index(content, "drop table")
	if lock == -1 || assertion == -1 || firstDDL == -1 || lock > assertion || assertion > firstDDL {
		t.Fatal("legacy Anchor tables must be locked before the emptiness assertion and destructive DDL")
	}
}
