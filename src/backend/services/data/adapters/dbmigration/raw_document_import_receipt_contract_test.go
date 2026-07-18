package dbmigration

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	historicalMigrationManifestSHA256       = "2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc"
	rawDocumentImportReceiptMigrationSHA256 = "3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26"
)

func TestRawDocumentImportReceiptMigrationDefinesFrozenContract(t *testing.T) {
	rawSQL := readMigration(t, "000022_add_raw_document_import_receipts.sql")
	compact := compactMigrationSQL(rawSQL)

	for _, required := range []string{
		"-- +goose Up",
		"CREATE TABLE raw_document_import_receipts (",
		"id UUID NOT NULL,",
		"caller_identity TEXT NOT NULL,",
		"idempotency_key TEXT NOT NULL,",
		"payload_hash CHAR(64) NOT NULL,",
		"raw_document_ids UUID[] NOT NULL,",
		"result_payload JSONB NOT NULL,",
		"imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),",
		"CONSTRAINT raw_document_import_receipts_pkey PRIMARY KEY (id)",
		"CONSTRAINT uq_raw_document_import_receipts_caller_key UNIQUE (caller_identity, idempotency_key)",
		"CONSTRAINT chk_raw_document_import_receipts_caller CHECK (char_length(caller_identity) BETWEEN 1 AND 200 AND btrim(caller_identity) <> '')",
		"CONSTRAINT chk_raw_document_import_receipts_key CHECK (char_length(idempotency_key) BETWEEN 1 AND 200 AND btrim(idempotency_key) <> '')",
		"CONSTRAINT chk_raw_document_import_receipts_payload_hash CHECK (payload_hash ~ '^[0-9a-f]{64}$')",
		"CONSTRAINT chk_raw_document_import_receipts_raw_ids CHECK (array_ndims(raw_document_ids) = 1 AND cardinality(raw_document_ids) >= 1 AND array_position(raw_document_ids, NULL::uuid) IS NULL)",
		"CONSTRAINT chk_raw_document_import_receipts_result CHECK (",
		"jsonb_typeof(result_payload) = 'object'",
		"result_payload ?& ARRAY['receipt_id', 'payload_hash', 'raw_document_ids', 'items', 'imported_at']::text[]",
		"result_payload -> 'receipt_id' = to_jsonb(id::text)",
		"result_payload -> 'payload_hash' = to_jsonb(payload_hash::text)",
		"result_payload -> 'raw_document_ids' = to_jsonb(raw_document_ids)",
		"jsonb_array_length(result_payload -> 'items') = cardinality(raw_document_ids)",
		"jsonb_path_query_array(result_payload, '$.items[*].raw_document_id') = to_jsonb(raw_document_ids)",
		"jsonb_array_length(jsonb_path_query_array(result_payload, '$.items[*].disposition')) = cardinality(raw_document_ids)",
		"jsonb_path_query_array(result_payload, '$.items[*].disposition') <@ '[\"created\", \"reused\"]'::jsonb",
		"(result_payload ->> 'imported_at')::timestamptz = imported_at",
		"CREATE INDEX idx_raw_document_import_receipts_imported_at ON raw_document_import_receipts (imported_at);",
		"CREATE FUNCTION prevent_raw_document_import_receipt_mutation() RETURNS trigger LANGUAGE plpgsql",
		"ERRCODE = '55000'",
		"MESSAGE = 'raw document import receipts are immutable'",
		"CREATE TRIGGER trg_raw_document_import_receipts_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON raw_document_import_receipts FOR EACH STATEMENT EXECUTE FUNCTION prevent_raw_document_import_receipt_mutation();",
		"-- +goose Down",
		"MESSAGE = 'migration 000022 is forward-only; use a reviewed forward migration or restore the pre-migration backup'",
	} {
		if !strings.Contains(compact, required) {
			t.Fatalf("000022 migration missing frozen contract fragment %q", required)
		}
	}

	tableBody := migrationTableBody(t, rawSQL, "raw_document_import_receipts")
	columnPattern := regexp.MustCompile(`(?m)^\s{4}([a-z_]+)\s+(UUID|TEXT|CHAR\(64\)|UUID\[\]|JSONB|TIMESTAMPTZ)\s+NOT NULL(?:\s+DEFAULT\s+now\(\))?,\s*$`)
	matches := columnPattern.FindAllStringSubmatch(tableBody, -1)
	var columns []string
	for _, match := range matches {
		columns = append(columns, match[1])
	}
	wantColumns := []string{"id", "caller_identity", "idempotency_key", "payload_hash", "raw_document_ids", "result_payload", "imported_at"}
	if fmt.Sprint(columns) != fmt.Sprint(wantColumns) {
		t.Fatalf("000022 columns = %v, want exact frozen seven columns %v", columns, wantColumns)
	}
}

func TestRawDocumentImportReceiptMigrationIsTransactionalForwardOnlyAndFailClosed(t *testing.T) {
	rawSQL := readMigration(t, "000022_add_raw_document_import_receipts.sql")
	lower := strings.ToLower(rawSQL)
	up, down := migrationSections(t, rawSQL)

	for _, forbidden := range []string{"-- +goose no transaction", "if not exists", "or replace"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("000022 migration contains fail-open fragment %q", forbidden)
		}
	}
	forbiddenUpStatement := regexp.MustCompile(`(?im)^\s*(insert\s+into|update\s+|delete\s+from|truncate\s+|alter\s+table|drop\s+)`)
	if statement := forbiddenUpStatement.FindString(up); statement != "" {
		t.Fatalf("000022 Up contains existing-data or destructive statement %q", strings.TrimSpace(statement))
	}
	for _, forbidden := range []string{"event_import_receipts", "insert into events", "insert into source_catalogs", "insert into event_tag"} {
		if strings.Contains(strings.ToLower(up), forbidden) {
			t.Fatalf("000022 Up crosses the raw receipt boundary with %q", forbidden)
		}
	}
	if forbiddenRuntimeObject := regexp.MustCompile(`(?i)\b(jobs?|runs?)\b`).FindString(up); forbiddenRuntimeObject != "" {
		t.Fatalf("000022 Up creates forbidden runtime state %q", forbiddenRuntimeObject)
	}
	if strings.Count(up, "-- +goose StatementBegin") != 1 || strings.Count(up, "-- +goose StatementEnd") != 1 {
		t.Fatal("000022 Up must wrap exactly the trigger function body in one Goose statement block")
	}
	if !strings.Contains(down, "-- +goose StatementBegin") || !strings.Contains(down, "DO $$") || !strings.Contains(down, "RAISE EXCEPTION USING") || !strings.Contains(down, "-- +goose StatementEnd") {
		t.Fatal("000022 Down must be one executable failing DO block")
	}
	forbiddenDown := regexp.MustCompile(`(?im)^\s*(select\s+|drop\s+|alter\s+|delete\s+|truncate\s+|update\s+|insert\s+)`)
	if statement := forbiddenDown.FindString(down); statement != "" {
		t.Fatalf("000022 Down must fail instead of succeeding or mutating: %q", strings.TrimSpace(statement))
	}
}

func TestRawDocumentImportReceiptMigrationPreservesHistoricalManifest(t *testing.T) {
	entries, err := filepath.Glob(filepath.Join(migrationDirectory(), "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)

	var manifest strings.Builder
	historicalCount := 0
	for _, path := range entries {
		name := filepath.Base(path)
		if name >= "000022_" {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(content)
		fmt.Fprintf(&manifest, "%x  backend/migrations/%s\n", sum, name)
		historicalCount++
	}
	if historicalCount != 21 {
		t.Fatalf("historical migration count = %d, want 21", historicalCount)
	}
	aggregate := fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.String())))
	if aggregate != historicalMigrationManifestSHA256 {
		t.Fatalf("historical migration aggregate hash = %s, want %s", aggregate, historicalMigrationManifestSHA256)
	}
}

func TestRawDocumentImportReceiptMigrationHashIsFrozen(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(migrationDirectory(), "000022_add_raw_document_import_receipts.sql"))
	if err != nil {
		t.Fatal(err)
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(content))
	if actual != rawDocumentImportReceiptMigrationSHA256 {
		t.Fatalf("000022 migration hash = %s, want frozen %s", actual, rawDocumentImportReceiptMigrationSHA256)
	}
}

func compactMigrationSQL(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func migrationTableBody(t *testing.T, sql, table string) string {
	t.Helper()
	startToken := "CREATE TABLE " + table + " ("
	start := strings.Index(sql, startToken)
	if start < 0 {
		t.Fatalf("migration does not create table %s", table)
	}
	start += len(startToken)
	end := strings.Index(sql[start:], "\n);\n")
	if end < 0 {
		t.Fatalf("migration table %s has no closing statement", table)
	}
	return sql[start : start+end]
}

func migrationSections(t *testing.T, sql string) (string, string) {
	t.Helper()
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"
	upStart := strings.Index(sql, upMarker)
	downStart := strings.Index(sql, downMarker)
	if upStart < 0 || downStart <= upStart {
		t.Fatal("migration must contain ordered Goose Up and Down markers")
	}
	return sql[upStart:downStart], sql[downStart:]
}
