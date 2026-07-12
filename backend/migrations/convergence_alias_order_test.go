package migrations_test

import (
	"context"
	"database/sql"
	"os"
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
	migrations, err := goose.CollectMigrations(".", 12, 13)
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
