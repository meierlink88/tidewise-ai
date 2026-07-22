package dbmigration

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFileSourceListsVersionedMigrations(t *testing.T) {
	source := FileSource{Dir: filepath.Join("..", "..", "..", "..", "migrations")}

	migrations, err := source.ListMigrations(context.Background())
	if err != nil {
		t.Fatalf("ListMigrations() error = %v", err)
	}

	if got, want := migrationVersions(migrations), []string{"000001", "000002", "000003", "000004", "000005", "000006", "000007", "000008", "000009", "000010", "000011", "000012", "000013", "000014", "000015", "000016", "000017", "000018", "000019", "000020", "000021", "000022", "000023", "000024", "000025", "000026", "000027"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("migration versions = %v, want %v", got, want)
	}
	if migrations[0].Name != "000001_init_event_knowledge_schema.sql" {
		t.Fatalf("migration name = %q", migrations[0].Name)
	}
	if migrations[1].Name != "000002_add_alliance_org_profiles.sql" {
		t.Fatalf("migration name = %q", migrations[1].Name)
	}
	if migrations[2].Name != "000003_add_sector_seed_snapshot_fields.sql" {
		t.Fatalf("migration name = %q", migrations[2].Name)
	}
	if migrations[3].Name != "000004_add_source_catalog_source_config.sql" {
		t.Fatalf("migration name = %q", migrations[3].Name)
	}
	if migrations[4].Name != "000005_add_ingestion_scheduler.sql" {
		t.Fatalf("migration name = %q", migrations[4].Name)
	}
	if migrations[5].Name != "000006_add_graph_projection_runs.sql" {
		t.Fatalf("migration name = %q", migrations[5].Name)
	}
	if migrations[6].Name != "000007_add_entity_edge_provenance.sql" {
		t.Fatalf("migration name = %q", migrations[6].Name)
	}
	if migrations[9].Name != "000010_add_market_sector_foundation.sql" {
		t.Fatalf("migration name = %q", migrations[9].Name)
	}
	if migrations[10].Name != "000011_add_sector_convergence.sql" {
		t.Fatalf("migration name = %q", migrations[10].Name)
	}
	if migrations[11].Name != "000012_restore_current_convergence_aliases.sql" {
		t.Fatalf("migration name = %q", migrations[11].Name)
	}
	if migrations[12].Name != "000013_normalize_current_convergence_alias_order.sql" {
		t.Fatalf("migration name = %q", migrations[12].Name)
	}
	if migrations[13].Name != "000014_add_industry_chain_foundation.sql" {
		t.Fatalf("migration name = %q", migrations[13].Name)
	}
	if migrations[14].Name != "000015_refactor_industry_chain_node_phase_a.sql" {
		t.Fatalf("migration name = %q", migrations[14].Name)
	}
	if migrations[15].Name != "000016_add_entity_external_identifiers.sql" {
		t.Fatalf("migration name = %q", migrations[15].Name)
	}
	if migrations[16].Name != "000017_add_chain_node_relations.sql" {
		t.Fatalf("migration name = %q", migrations[16].Name)
	}
	if migrations[17].Name != "000018_reinitialize_alliance_economy_foundation.sql" {
		t.Fatalf("migration name = %q", migrations[17].Name)
	}
	if migrations[18].Name != "000019_add_event_fact_contract.sql" {
		t.Fatalf("migration name = %q", migrations[18].Name)
	}
	if migrations[19].Name != "000020_add_event_import_receipts_and_tag_seed.sql" {
		t.Fatalf("migration name = %q", migrations[19].Name)
	}
	if migrations[20].Name != "000021_add_research_theme_anchor_foundation.sql" {
		t.Fatalf("migration name = %q", migrations[20].Name)
	}
	if migrations[21].Name != "000022_add_raw_document_import_receipts.sql" {
		t.Fatalf("migration name = %q", migrations[21].Name)
	}
	if migrations[22].Name != "000023_correct_research_theme_transmission_stages.sql" {
		t.Fatalf("migration name = %q", migrations[22].Name)
	}
	if migrations[23].Name != "000024_add_research_theme_imports.sql" {
		t.Fatalf("migration name = %q", migrations[23].Name)
	}
	if migrations[24].Name != "000025_rebuild_research_anchor_reasoning_trees.sql" {
		t.Fatalf("migration name = %q", migrations[24].Name)
	}
	if migrations[25].Name != "000026_add_research_anchor_branch_summaries.sql" {
		t.Fatalf("migration name = %q", migrations[25].Name)
	}
	if migrations[26].Name != "000027_add_typed_master_data_schema.sql" {
		t.Fatalf("migration name = %q", migrations[26].Name)
	}
	for _, migration := range migrations {
		if migration.Path == "" {
			t.Fatalf("migration %s path is empty", migration.Name)
		}
	}
}

func TestFileSourceRejectsDuplicateVersions(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFixture(t, dir, "000001_first.sql")
	writeMigrationFixture(t, dir, "000001_second.sql")

	source := FileSource{Dir: dir}

	if _, err := source.ListMigrations(context.Background()); err == nil {
		t.Fatal("ListMigrations() error = nil, want duplicate version error")
	}
}

func writeMigrationFixture(t *testing.T, dir string, name string) {
	t.Helper()

	content := []byte("-- +goose Up\nSELECT 1;\n-- +goose Down\nSELECT 1;\n")
	if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
		t.Fatalf("write migration fixture: %v", err)
	}
}
