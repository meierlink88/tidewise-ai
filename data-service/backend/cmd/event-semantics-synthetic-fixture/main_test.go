package main

import (
	"context"
	"path/filepath"
	"testing"

	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestValidateLocalDatabaseURLRejectsUnapprovedOrIncompleteTargets(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name: "external local PostgreSQL URL",
			raw:  "postgres://fixture:secret@host.docker.internal:5432/tidewise_local?sslmode=disable",
		},
		{
			name:    "remote host",
			raw:     "postgres://fixture:secret@database.example/tidewise",
			wantErr: true,
		},
		{
			name:    "missing database",
			raw:     "postgres://fixture:secret@host.docker.internal:5432",
			wantErr: true,
		},
		{
			name:    "non PostgreSQL scheme",
			raw:     "https://host.docker.internal/tidewise",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateLocalDatabaseURL(test.raw)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateLocalDatabaseURL(%q) error = %v", test.raw, err)
			}
		})
	}
}

func TestSyntheticEntitySeedRunsAgainstCurrentLedger(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_semantic_fixture", migrationDir, 0)
	for _, statement := range syntheticEntitySeedStatements {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatalf("seed current synthetic Entity fixture: %v\n%s", err, statement)
		}
	}
	var industries, relations int
	if err := db.QueryRow(`SELECT count(*) FROM industry WHERE classification_system = 'synthetic'`).Scan(&industries); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM entity_edges WHERE id IN (
		'ERL22000000-0000-4000-8000-000000000003',
		'ERL23000000-0000-4000-8000-000000000003'
	)`).Scan(&relations); err != nil {
		t.Fatal(err)
	}
	if industries != 3 || relations != 2 {
		t.Fatalf("synthetic fixture industries=%d relations=%d", industries, relations)
	}
}
