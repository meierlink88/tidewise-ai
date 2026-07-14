package seed

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresPolymorphicJSONParametersUseExplicitTypes(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TIDEWISE_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var raw []byte
	err = db.QueryRowContext(ctx, `SELECT jsonb_build_object('value',$1)`, "untyped").Scan(&raw)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42P18" {
		t.Fatalf("bare polymorphic parameter error = %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT "+convergenceAuditMutationExpression(1, 2), "checksum", "initial").Scan(&raw); err != nil {
		t.Fatalf("typed audit provenance: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT "+convergenceOperationMutationExpression(1), "redirect").Scan(&raw); err != nil {
		t.Fatalf("typed operation provenance: %v", err)
	}
}

func TestPostgresChainNodeUpsertSQLPlansWithoutRetiredConvergenceTables(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TIDEWISE_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	key := "chain_node:artificial_intelligence"
	rows, err := db.QueryContext(context.Background(), "EXPLAIN "+buildEntityUpsert(), entitySeedUUID(key), key, "chain_node", "chain_node", "人工智能产业", "人工智能产业", []string{"人工智能"}, "active")
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
