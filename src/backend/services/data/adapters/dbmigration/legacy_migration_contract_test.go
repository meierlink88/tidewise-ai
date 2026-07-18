package dbmigration

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestConvergenceAliasOrderNormalizationIsAuditDriven(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000013_normalize_current_convergence_alias_order.sql"))
	for _, fragment := range []string{"max(manifest_version)", "entity_convergence_alias_moves", "order by am.moved_at,am.id", "with ordinality", "is distinct from", "reviewed forward migration"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}

func TestPostgresConvergenceAliasOrderNormalizationPlansWithoutWriting(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TIDEWISE_TEST_DATABASE_URL is not set")
	}
	contents := readMigration(t, "000013_normalize_current_convergence_alias_order.sql")
	parts := strings.Split(contents, "-- +goose Down")
	upSQL := strings.TrimSpace(strings.Replace(parts[0], "-- +goose Up", "", 1))
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.QueryContext(context.Background(), "EXPLAIN "+upSQL)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	lines := 0
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		lines++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if lines == 0 {
		t.Fatal("EXPLAIN returned no plan")
	}
}

func TestConvergenceAliasOrderNormalizationExecutesThroughGoose(t *testing.T) {
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	migrations, err := goose.CollectMigrations(migrationDirectory(), 12, 13)
	if err != nil {
		t.Fatal(err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec(`(?s)WITH current_manifest AS .*UPDATE entity_nodes`).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`INSERT INTO .*goose_db_version`).WithArgs(int64(13), true).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := migrations[0].UpContext(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConvergenceAliasRepairIsForwardIdempotentAndAuditDriven(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000012_restore_current_convergence_aliases.sql"))
	for _, fragment := range []string{"max(manifest_version)", "entity_convergence_alias_moves", "entity_convergences", "unnest(n.aliases || ca.aliases)", "is distinct from", "not (", "any(n.aliases)", "reviewed forward migration"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("alias repair missing %q", fragment)
		}
	}
	for _, forbidden := range []string{"人工智能", "算力", "delete from", "truncate "} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("alias repair contains %q", forbidden)
		}
	}
}

func TestPostgresConvergenceAliasRepairPlansWithoutWriting(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TIDEWISE_TEST_DATABASE_URL is not set")
	}
	contents := readMigration(t, "000012_restore_current_convergence_aliases.sql")
	parts := strings.Split(contents, "-- +goose Down")
	upSQL := strings.TrimSpace(strings.Replace(parts[0], "-- +goose Up", "", 1))
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.QueryContext(context.Background(), "EXPLAIN "+upSQL)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	lines := 0
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		lines++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if lines == 0 {
		t.Fatal("EXPLAIN returned no plan")
	}
}

func TestConvergenceAliasRepairExecutesThroughGoose(t *testing.T) {
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	migrations, err := goose.CollectMigrations(migrationDirectory(), 11, 12)
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
	mock.ExpectExec(`(?s)WITH current_manifest AS .*UPDATE entity_nodes`).WillReturnResult(sqlmock.NewResult(0, 24))
	mock.ExpectExec(`INSERT INTO .*goose_db_version`).WithArgs(int64(12), true).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := migrations[0].UpContext(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConvergenceAliasRepairFailureRollsBack(t *testing.T) {
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	migrations, err := goose.CollectMigrations(migrationDirectory(), 11, 12)
	if err != nil {
		t.Fatal(err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec(`(?s)WITH current_manifest AS .*UPDATE entity_nodes`).WillReturnError(errors.New("forced repair failure"))
	mock.ExpectRollback()
	if err := migrations[0].UpContext(context.Background(), db); err == nil || !strings.Contains(err.Error(), "forced repair failure") {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

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
	migrations, err := goose.CollectMigrations(migrationDirectory(), 10, 11)
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

func TestMarketSectorSchemaMigrationDefinesProfilesAndSourceMappings(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000010_add_market_sector_foundation.sql"))
	for _, fragment := range []string{
		"alter table sector_profiles",
		"add column if not exists classification_code text",
		"update sector_profiles",
		"set classification_code = case",
		"when lower(trim(sector_type)) = 'industry' then 'industry_sector'",
		"when lower(trim(sector_type)) = 'concept' then 'theme_sector'",
		"else 'market_sector'",
		"alter column classification_code set default 'market_sector'",
		"alter column classification_code set not null",
		"add column if not exists primary_market_entity_id uuid references entity_nodes(id)",
		"add column if not exists primary_economy_entity_id uuid references entity_nodes(id)",
		"add column if not exists methodology_url text",
		"add column if not exists review_status text",
		"classification_code in ('industry_sector', 'theme_sector', 'market_sector', 'style_sector', 'region_sector')",
		"create table if not exists sector_source_mappings",
		"sector_entity_id uuid not null references entity_nodes(id)",
		"source_taxonomy_type in ('concept', 'industry', 'index_sector')",
		"source_sector_name_normalized text not null",
		"source_market_scope text not null default ''",
		"mapping_status in ('candidate', 'approved', 'rejected', 'merged')",
		"create unique index if not exists uq_sector_source_mappings_code",
		"(source_system, source_taxonomy_type, source_sector_code)",
		"where source_sector_code <> ''",
		"create unique index if not exists uq_sector_source_mappings_name_scope",
		"(source_system, source_taxonomy_type, source_sector_name_normalized, source_market_scope)",
		"where source_sector_code = ''",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("market sector migration missing %q", fragment)
		}
	}
	if strings.Contains(sql, "source_sector_name_normalized, source_market_scope, snapshot_date") {
		t.Fatal("source mapping identity must not include snapshot_date")
	}
	addColumn := strings.Index(sql, "add column if not exists classification_code text")
	backfill := strings.Index(sql, "update sector_profiles")
	setNotNull := strings.Index(sql, "alter column classification_code set not null")
	if addColumn < 0 || backfill <= addColumn || setNotNull <= backfill {
		t.Fatalf("classification migration order = add:%d backfill:%d not-null:%d", addColumn, backfill, setNotNull)
	}
	if strings.Contains(sql[addColumn:backfill], "classification_code text not null default") {
		t.Fatal("classification column must not apply a blanket default before deterministic backfill")
	}
}

func TestMarketSectorSchemaMigrationIsNonDestructive(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000010_add_market_sector_foundation.sql"))
	for _, forbidden := range []string{"drop table", "drop column", "truncate", "delete from"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("market sector migration contains destructive fragment %q", forbidden)
		}
	}
	if !strings.Contains(sql, "rollback requires a reviewed forward migration or restored backup") {
		t.Fatal("market sector migration must document forward-only rollback")
	}
}
