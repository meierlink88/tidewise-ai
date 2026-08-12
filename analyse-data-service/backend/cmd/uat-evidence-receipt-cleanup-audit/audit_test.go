package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresCleanupAuditReadsFormalEvidenceFacts(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TIDEWISE_TEST_DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("tw_receipt_cleanup_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, cleanupErr := admin.ExecContext(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE"); cleanupErr != nil {
			t.Errorf("drop cleanup schema: %v", cleanupErr)
		}
		if closeErr := admin.Close(); closeErr != nil {
			t.Errorf("close cleanup database: %v", closeErr)
		}
	})

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsedURL.Query()
	query.Set("search_path", schema)
	parsedURL.RawQuery = query.Encode()
	database, err := sql.Open("pgx", parsedURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.ExecContext(ctx, `
		CREATE TABLE raw_evidences (raw_evidence_id text PRIMARY KEY, raw_text text NOT NULL);
		CREATE TABLE evidences (evidence_id text PRIMARY KEY, raw_evidence_id text NOT NULL, source_what text NOT NULL);
		INSERT INTO raw_evidences VALUES ('raw-1', 'first'), ('raw-2', 'second');
		INSERT INTO evidences VALUES ('evidence-1', 'raw-1', 'fact');
	`); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run(ctx, database, &output); err != nil {
		t.Fatal(err)
	}
	var report cleanupAuditReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if raw := report.ProtectedTables["raw_evidences"]; raw.RowCount != 2 || len(raw.RowFingerprints) != 2 || raw.RowFingerprints["raw-1"] == "" {
		t.Fatalf("raw audit = %#v", raw)
	}
	if evidence := report.ProtectedTables["evidences"]; evidence.RowCount != 1 || len(evidence.RowFingerprints) != 1 || evidence.RowFingerprints["evidence-1"] == "" {
		t.Fatalf("Evidence audit = %#v", evidence)
	}
}
