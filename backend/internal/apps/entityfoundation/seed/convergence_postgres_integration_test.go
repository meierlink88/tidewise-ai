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

func TestPostgresCurrentConvergenceAliasOwnershipIsRecoverable(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TIDEWISE_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.QueryContext(context.Background(), currentConvergenceOwnedAliasesQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	count := 0
	missing := 0
	for rows.Next() {
		var entityID, alias string
		var present bool
		if err := rows.Scan(&entityID, &alias, &present); err != nil {
			t.Fatal(err)
		}
		count++
		if !present {
			missing++
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 29 || missing != 29 {
		t.Fatalf("owned aliases count=%d missing=%d", count, missing)
	}
}

func TestPostgresEntityUpsertAliasOwnershipSQLPlansWithoutWriting(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TIDEWISE_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	key := "sector:theme_artificial_intelligence"
	rows, err := db.QueryContext(context.Background(), "EXPLAIN "+buildEntityUpsert(), entitySeedUUID(key), key, "sector", "sector", "人工智能", "人工智能", []string{"Artificial Intelligence"}, "active")
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
