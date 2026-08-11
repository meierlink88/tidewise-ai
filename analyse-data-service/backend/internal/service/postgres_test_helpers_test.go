package service

import (
	"database/sql"
	"path/filepath"
	"testing"

	postgresfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/postgres"
)

func openEventPublicationTestDatabase(t *testing.T) *sql.DB {
	return openEventPublicationTestDatabaseAt(t, 0)
}

func openEventPublicationTestDatabaseAt(t *testing.T, version int64) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_service_integration", migrationDir, version)
}

func applyEventPublicationMigration(t *testing.T, db *sql.DB, version int64) {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	postgresfixture.ApplyMigration(t, db, migrationDir, version)
}
