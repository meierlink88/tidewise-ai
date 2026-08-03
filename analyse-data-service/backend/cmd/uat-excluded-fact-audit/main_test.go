package main

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRunAuditsEveryExcludedFactTableDeterministically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	for _, table := range auditedTables {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT to_regclass($1)")).
			WithArgs(table).
			WillReturnRows(sqlmock.NewRows([]string{"to_regclass"}).AddRow(table))
		mock.ExpectQuery(regexp.QuoteMeta(auditQuery(table))).
			WillReturnRows(sqlmock.NewRows([]string{"count", "fingerprint"}).
				AddRow(3, "0123456789abcdef0123456789abcdef"))
	}
	var output bytes.Buffer
	if err := run(context.Background(), database, &output); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	var report auditReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.ContractVersion != contractVersion || len(report.Tables) != len(auditedTables) {
		t.Fatalf("audit report = %#v", report)
	}
	for _, table := range auditedTables {
		value := report.Tables[table]
		if value.RowCount != 3 || value.Fingerprint != "0123456789abcdef0123456789abcdef" {
			t.Fatalf("table %s audit = %#v", table, value)
		}
	}
}

func TestAuditTableNormalizesNotYetMigratedTableAsEmpty(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT to_regclass($1)")).
		WithArgs("event_semantic_resolution_bindings").
		WillReturnRows(sqlmock.NewRows([]string{"to_regclass"}).AddRow(nil))
	value, err := auditTable(context.Background(), database, "event_semantic_resolution_bindings")
	if err != nil {
		t.Fatal(err)
	}
	if value.RowCount != 0 || value.Fingerprint != emptyTableFingerprint {
		t.Fatalf("not-yet-migrated table audit = %#v", value)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditQueryUsesOnlyFrozenTableNames(t *testing.T) {
	for _, table := range auditedTables {
		if !regexp.MustCompile(`^[a-z_]+$`).MatchString(table) {
			t.Fatalf("unsafe audited table name %q", table)
		}
		query := auditQuery(table)
		if !regexp.MustCompile(`FROM ` + regexp.QuoteMeta(table) + ` source_row`).MatchString(query) {
			t.Fatalf("audit query does not target %q: %s", table, query)
		}
	}
}

func TestAuditRowsNormalizeForwardOnlyEventSchemaChanges(t *testing.T) {
	tests := map[string][]string{
		"events": {
			"to_jsonb(source_row) - 'primary_source_id'",
		},
		"event_sources": {
			"- 'evidence_excerpt' - 'is_primary' - 'contract_version'",
			"'evidence_statement'",
			"WHEN to_jsonb(source_row) ->> 'contract_version' = '2' THEN '3'::jsonb",
		},
		"event_semantic_context_leases": {
			"'context_manifest', to_jsonb(source_row) -> 'context_manifest'",
		},
	}
	for table, required := range tests {
		expression := auditRowExpression(table)
		for _, fragment := range required {
			if !regexp.MustCompile(regexp.QuoteMeta(fragment)).MatchString(expression) {
				t.Fatalf("%s audit normalization missing %q: %s", table, fragment, expression)
			}
		}
	}
}
