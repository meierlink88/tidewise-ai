package runtimeaudit

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresInspectReportsCurrentAndCandidateCatalogObjects(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run runtime audit integration tests")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	host := parsed.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatalf("runtime audit integration database must use a loopback host, got %q", host)
	}
	databaseName := strings.TrimPrefix(parsed.Path, "/")
	roleName := parsed.User.Username()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := database.Close(); closeErr != nil {
			t.Errorf("close runtime audit integration database: %v", closeErr)
		}
	})
	if err := database.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(database)
	if err != nil {
		t.Fatal(err)
	}

	present, err := store.Inspect(ctx, databaseName, roleName)
	if err != nil {
		t.Fatal(err)
	}
	if present.CurrentDatabase != databaseName || present.CurrentRole != roleName {
		t.Fatalf("connected identity = %#v", present)
	}
	if !present.DatabasePresent || !present.RolePresent {
		t.Fatalf("current catalog objects not found: %#v", present)
	}

	absent, err := store.Inspect(ctx, "tidewise_contract_absent_database", "tidewise_contract_absent_role")
	if err != nil {
		t.Fatal(err)
	}
	if absent.DatabasePresent || absent.RolePresent {
		t.Fatalf("unexpected catalog objects found: %#v", absent)
	}
}

func TestNewStoreRejectsMissingDatabase(t *testing.T) {
	if _, err := NewStore(nil); err == nil {
		t.Fatal("NewStore(nil) error = nil")
	}
}
