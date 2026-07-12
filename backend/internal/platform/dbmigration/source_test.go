package dbmigration

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFileSourceListsVersionedMigrations(t *testing.T) {
	source := FileSource{Dir: filepath.Join("..", "..", "..", "migrations")}

	migrations, err := source.ListMigrations(context.Background())
	if err != nil {
		t.Fatalf("ListMigrations() error = %v", err)
	}

	if got, want := migrationVersions(migrations), []string{"000001", "000002", "000003", "000004", "000005", "000006", "000007", "000008", "000009", "000010"}; !reflect.DeepEqual(got, want) {
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
