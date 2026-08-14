package dbmigration

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFileSourceListsVersionedMigrations(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFixture(t, dir, "000010_tenth.sql")
	writeMigrationFixture(t, dir, "000002_second.sql")
	source := FileSource{Dir: dir}

	migrations, err := source.ListMigrations(context.Background())
	if err != nil {
		t.Fatalf("ListMigrations() error = %v", err)
	}

	if got, want := migrationVersions(migrations), []string{"000002", "000010"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("migration versions = %v, want %v", got, want)
	}
	if migrations[0].Name != "000002_second.sql" {
		t.Fatalf("migration name = %q", migrations[0].Name)
	}
	if migrations[1].Name != "000010_tenth.sql" {
		t.Fatalf("migration name = %q", migrations[1].Name)
	}
	for _, migration := range migrations {
		if filepath.Dir(migration.Path) != dir {
			t.Fatalf("migration %s path = %q, want directory %q", migration.Name, migration.Path, dir)
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

func TestFileSourceRejectsInvalidFilename(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFixture(t, dir, "1_invalid.sql")

	if _, err := (FileSource{Dir: dir}).ListMigrations(context.Background()); err == nil {
		t.Fatal("ListMigrations() error = nil, want invalid filename error")
	}
}

func writeMigrationFixture(t *testing.T, dir string, name string) {
	t.Helper()

	content := []byte("-- +goose Up\nSELECT 1;\n-- +goose Down\nSELECT 1;\n")
	if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
		t.Fatalf("write migration fixture: %v", err)
	}
}
