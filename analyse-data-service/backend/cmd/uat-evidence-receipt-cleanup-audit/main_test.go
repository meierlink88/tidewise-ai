package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCleanupAuditReportsObjectsAndProtectedEvidenceIdentities(t *testing.T) {
	database, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.MatchExpectationsInOrder(false)
	for _, query := range []string{
		"SELECT to_regclass('public.raw_evidence_publication_receipts') IS NOT NULL",
		"SELECT to_regclass('public.evidence_publication_receipts') IS NOT NULL",
		"SELECT to_regprocedure('public.prevent_evidence_publication_receipt_mutation()') IS NOT NULL",
	} {
		mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT raw_evidence_id::text, md5(to_jsonb(source_row)::text) FROM raw_evidences source_row ORDER BY raw_evidence_id")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fingerprint"}).AddRow("raw-1", "raw-hash-1").AddRow("raw-2", "raw-hash-2"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT evidence_id::text, md5(to_jsonb(source_row)::text) FROM evidences source_row ORDER BY evidence_id")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fingerprint"}).AddRow("evidence-1", "evidence-hash-1"))

	var output bytes.Buffer
	if err := run(context.Background(), database, &output); err != nil {
		t.Fatal(err)
	}
	var report cleanupAuditReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.ContractVersion != auditContractVersion || len(report.Objects) != 3 {
		t.Fatalf("unexpected report: %#v", report)
	}
	for name, exists := range report.Objects {
		if !exists {
			t.Fatalf("object %s should exist", name)
		}
	}
	wantRawHash := sha256.Sum256([]byte("raw-1:raw-hash-1\nraw-2:raw-hash-2\n"))
	if got := report.ProtectedTables["raw_evidences"]; got.RowCount != 2 || got.Fingerprint != hex.EncodeToString(wantRawHash[:]) || len(got.RowFingerprints) != 2 || got.RowFingerprints["raw-1"] != "raw-hash-1" || got.RowFingerprints["raw-2"] != "raw-hash-2" {
		t.Fatalf("raw_evidences audit = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupAuditRejectsMissingDependencies(t *testing.T) {
	if err := run(context.Background(), nil, &bytes.Buffer{}); err == nil {
		t.Fatal("expected missing database error")
	}
}
