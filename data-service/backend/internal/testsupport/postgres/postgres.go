package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func OpenIsolated(t testing.TB, schemaPrefix, migrationDir string, version int64) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	host := parsed.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatalf("integration database must use a loopback host, got %q", host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("%s_%d", schemaPrefix, time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}

	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema
	config.RuntimeParams["tidewise.phase_a_cleanup_write_authorized"] = "reviewed_backup_verified"
	config.RuntimeParams["tidewise.external_identifier_schema_write_authorized"] = "reviewed_backup_verified"
	config.RuntimeParams["tidewise.alliance_economy_schema_write_authorized"] = "reviewed_local_cleanup_verified"
	db := stdlib.OpenDB(*config)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		_ = admin.Close()
		t.Fatal(err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if version == 0 {
		err = goose.UpContext(ctx, db, migrationDir)
	} else {
		err = goose.UpToContext(ctx, db, migrationDir, version)
	}
	if err != nil {
		t.Fatalf("apply migrations in isolated schema: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
		_ = admin.Close()
	})
	return db
}

func ApplyMigration(t testing.TB, db *sql.DB, migrationDir string, version int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := goose.UpToContext(ctx, db, migrationDir, version); err != nil {
		t.Fatalf("apply migration %d: %v", version, err)
	}
}
