package dbmigration

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestMigrationChainIsOrderedAndGooseCompatible(t *testing.T) {
	entries, err := os.ReadDir(migrationDirectory())
	if err != nil {
		t.Fatal(err)
	}
	namePattern := regexp.MustCompile(`^\d{6}_[a-z0-9_]+\.sql$`)
	versions := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf("migration directory contains subdirectory %q", entry.Name())
		}
		if entry.Name() == "README.md" {
			continue
		}
		if !namePattern.MatchString(entry.Name()) {
			t.Fatalf("migration file %q must use 000001_name.sql format", entry.Name())
		}
		version := entry.Name()[:6]
		if previous := versions[version]; previous != "" {
			t.Fatalf("migration version %s is duplicated by %q and %q", version, previous, entry.Name())
		}
		versions[version] = entry.Name()
		content := readMigration(t, entry.Name())
		if !strings.Contains(content, "-- +goose Up") || !strings.Contains(content, "-- +goose Down") {
			t.Fatalf("migration %q must define Goose Up and Down sections", entry.Name())
		}
	}
	if len(versions) == 0 {
		t.Fatal("migration chain is empty")
	}
}

func TestMigrationChainRejectsCatastrophicResetStatements(t *testing.T) {
	entries, err := os.ReadDir(migrationDirectory())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content := strings.ToLower(readMigration(t, entry.Name()))
		for _, forbidden := range []string{"drop database", "drop schema", "truncate table"} {
			if strings.Contains(content, forbidden) {
				t.Fatalf("migration %q contains catastrophic reset statement %q", entry.Name(), forbidden)
			}
		}
		if entry.Name() != "000009_migrate_benchmark_metrics.sql" &&
			entry.Name() != "000015_refactor_industry_chain_node_phase_a.sql" &&
			strings.Contains(content, "delete from") {
			t.Fatalf("migration %q contains an unreviewed data deletion", entry.Name())
		}
	}
}

func TestResearchThemeReasoningTreeV1MigrationOwnsOnlyPublicationTables(t *testing.T) {
	content := strings.ToLower(readMigration(t, "000031_rebuild_research_theme_reasoning_trees_v1.sql"))
	for _, required := range []string{
		"create table research_theme_import_receipts",
		"create table research_themes",
		"create table research_theme_impacts",
		"create table research_theme_events",
		"create table research_reasoning_tree_import_receipts",
		"create table research_reasoning_trees",
		"create table research_reasoning_tree_events",
		"create table research_reasoning_tree_nodes",
		"create table research_reasoning_tree_node_signals",
		"trg_research_reasoning_tree_node_signals_immutable",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("V1 migration is missing %q", required)
		}
	}
	for _, protected := range []string{
		"events", "entity_nodes", "chain_node_profiles", "industry_chain_definitions",
		"industry_chain_graph_edges", "index_profiles", "event_tag_defs",
		"event_tag_maps", "raw_documents",
	} {
		if strings.Contains(content, "drop table "+protected) ||
			strings.Contains(content, "delete from "+protected) {
			t.Fatalf("V1 migration mutates protected table %q", protected)
		}
	}
}

func TestPostgresAppliesTheCompleteForwardMigrationChain(t *testing.T) {
	db := openIsolatedMigrationDatabase(t)
	executor := NewGooseExecutor(db, migrationDirectory())

	before, err := executor.Pending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(before) == 0 {
		t.Fatal("fresh database unexpectedly has no pending migrations")
	}
	if _, err := executor.Apply(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	after, err := executor.Pending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 0 {
		t.Fatalf("forward migration chain left %d migrations pending", len(after))
	}

	for _, table := range []string{
		"events",
		"event_publication_receipts",
		"research_theme_import_receipts",
		"research_reasoning_tree_import_receipts",
		"research_reasoning_trees",
		"research_reasoning_tree_nodes",
		"research_reasoning_tree_node_signals",
	} {
		var relation string
		if err := db.QueryRow(`SELECT COALESCE(to_regclass($1)::text, '')`, table).Scan(&relation); err != nil {
			t.Fatal(err)
		}
		if relation == "" {
			t.Fatalf("critical table %q is missing after the forward chain", table)
		}
	}
	for _, constraint := range []string{
		"uq_entity_external_identifier_identity",
		"chk_industry_profile_level",
		"uq_research_themes_batch_theme_key",
		"chk_research_theme_import_receipts_payload_hash",
		"uq_research_reasoning_trees_theme_chain",
		"chk_research_reasoning_tree_node_signals_key",
		"chk_event_publication_receipts_version",
	} {
		var exists bool
		if err := db.QueryRow(
			`SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = $1)`,
			constraint,
		).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("critical constraint %q is missing after the forward chain", constraint)
		}
	}
	for _, index := range []string{
		"ux_raw_documents_artifact_id",
		"ux_event_sources_v3_event_document",
	} {
		var relation string
		if err := db.QueryRow(`SELECT COALESCE(to_regclass($1)::text, '')`, index).Scan(&relation); err != nil {
			t.Fatal(err)
		}
		if relation == "" {
			t.Fatalf("critical index %q is missing after the forward chain", index)
		}
	}
	var removedPrimaryIndex string
	if err := db.QueryRow(
		`SELECT COALESCE(to_regclass('ux_event_sources_v2_primary')::text, '')`,
	).Scan(&removedPrimaryIndex); err != nil {
		t.Fatal(err)
	}
	if removedPrimaryIndex != "" {
		t.Fatal("V2 primary Event Evidence index still exists after the V3 forward chain")
	}
	if _, err := db.Exec(`
		UPDATE entity_type_definitions
		SET inclusion_criteria = ARRAY['valid boundary', '   ']::text[]
		WHERE status = 'active'
		  AND (type_key, version) = (
		      SELECT type_key, version FROM entity_type_definitions
		      WHERE status = 'active' ORDER BY type_key, version LIMIT 1
		  )
	`); err == nil {
		t.Fatal("V3 Entity Type boundary constraint accepted a whitespace-only criterion")
	}
}

func migrationVersions(migrations []Migration) []string {
	versions := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		versions = append(versions, migration.Version)
	}
	return versions
}

func migrationSections(t *testing.T, sql string) (string, string) {
	t.Helper()
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"
	upStart := strings.Index(sql, upMarker)
	downStart := strings.Index(sql, downMarker)
	if upStart < 0 || downStart <= upStart {
		t.Fatal("migration must contain ordered Goose Up and Down markers")
	}
	return sql[upStart:downStart], sql[downStart:]
}

func readMigration(t *testing.T, path string) string {
	t.Helper()
	if filepath.Base(path) == path {
		path = filepath.Join(migrationDirectory(), path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	return string(content)
}

func migrationDirectory() string {
	return filepath.Join("..", "..", "..", "migrations")
}
