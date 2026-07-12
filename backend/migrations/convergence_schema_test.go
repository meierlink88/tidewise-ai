package migrations_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/pressly/goose/v3"
)

func TestSectorConvergenceMigrationDefinesAppendOnlyVersionedAudit(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000011_add_sector_convergence.sql"))
	for _, fragment := range []string{
		"create table entity_convergence_manifests",
		"manifest_version bigint primary key",
		"previous_manifest_version bigint",
		"manifest_checksum text not null unique",
		"create table entity_convergences",
		"id uuid primary key",
		"legacy_entity_id uuid not null references entity_nodes(id)",
		"target_entity_id uuid references entity_nodes(id)",
		"target_entity_type text",
		"reason text not null check (btrim(reason) <> '')",
		"target_entity_type = 'sector'",
		"target_entity_type = 'index'",
		"unique (legacy_entity_id, manifest_version)",
		"create table entity_convergence_reference_moves",
		"create table entity_convergence_alias_moves",
		"raise exception 'convergence audit is append-only'",
		"before update or delete on entity_convergence_manifests",
		"before update or delete on entity_convergences",
		"before update or delete on entity_convergence_reference_moves",
		"before update or delete on entity_convergence_alias_moves",
		"-- +goose down",
		"reviewed forward migration or restored backup",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("convergence migration missing %q", fragment)
		}
	}
}

func TestSectorConvergenceMigrationExecutesPLpgSQLAsOneGooseStatement(t *testing.T) {
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	migrations, err := goose.CollectMigrations(".", 10, 11)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 {
		t.Fatalf("migrations = %d", len(migrations))
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	for _, table := range []string{"entity_convergence_manifests", "entity_convergences", "entity_convergence_reference_moves", "entity_convergence_alias_moves"} {
		mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE " + table)).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mock.ExpectExec(`(?s)CREATE OR REPLACE FUNCTION prevent_entity_convergence_audit_mutation\(\).*RETURNS trigger LANGUAGE plpgsql AS \$\$.*BEGIN.*RAISE EXCEPTION 'convergence audit is append-only';.*END;.*\$\$;`).WillReturnResult(sqlmock.NewResult(0, 0))
	for _, trigger := range []string{"entity_convergence_manifests_append_only", "entity_convergences_append_only", "entity_convergence_reference_moves_append_only", "entity_convergence_alias_moves_append_only"} {
		mock.ExpectExec(regexp.QuoteMeta("CREATE TRIGGER " + trigger)).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mock.ExpectExec(`INSERT INTO .*goose_db_version`).WithArgs(int64(11), true).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := migrations[0].UpContext(context.Background(), db); err != nil {
		t.Fatalf("UpContext() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
