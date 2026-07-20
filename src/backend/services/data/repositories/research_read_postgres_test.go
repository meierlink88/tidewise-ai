package repositories

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresResearchThemeReadSmoke(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run the PostgreSQL research read smoke test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	defer db.Close()

	asOf := time.Now().UTC()
	_, err = NewPostgresRepository(db).ListResearchThemes(context.Background(), ResearchThemeListFilter{
		WindowStart: asOf.Add(-24 * time.Hour),
		AsOf:        asOf,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("list research themes: %v", err)
	}
}
