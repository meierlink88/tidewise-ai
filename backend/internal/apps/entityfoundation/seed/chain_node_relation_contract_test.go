package seed

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestFrozenFirstBatchChainNodeRelationManifestMatchesApprovedArtifact(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "openspec", "changes", "refactor-industry-chain-node-foundation", "relation-candidate-artifacts", "relation-write-manifest.json")
	manifest, err := LoadChainNodeRelationManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateFrozenFirstBatchChainNodeRelationManifest(path, manifest); err != nil {
		t.Fatal(err)
	}
}

func TestPreflightFrozenChainNodeRelationDataRequiresExactGoose17SchemaAndBaseline(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(relationDataBaselineSQL).WillReturnRows(sqlmock.NewRows([]string{"database", "version", "goose", "nodes", "profiles", "external", "edges", "relations", "constraints"}).AddRow("tidewise_local", "16.14", 17, 842, 842, 1169, 331, 0, 0))
	mock.ExpectQuery(relationDataSchemaSQL).WillReturnRows(sqlmock.NewRows([]string{"relation_columns", "constraint_columns", "relation_checks", "relation_fks", "relation_pks", "relation_uniques", "constraint_checks", "constraint_fks", "constraint_pks", "relation_indexes", "constraint_indexes", "triggers"}).AddRow(relationColumnSignature, physicalConstraintColumnSignature, 7, 2, 1, 1, 7, 2, 1, 4, 3, 0))
	report, err := NewPostgresRepository(db).PreflightFrozenChainNodeRelationData(context.Background())
	if err != nil || !report.SchemaValid || report.GooseVersion != 17 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyFrozenChainNodeRelationPostWriteChecksProtectedBaselineAndExactAggregate(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(relationDataBaselineSQL).WillReturnRows(sqlmock.NewRows([]string{"database", "version", "goose", "nodes", "profiles", "external", "edges", "relations", "constraints"}).AddRow("tidewise_local", "16.14", 17, 842, 842, 1169, 331, 96, 0))
	mock.ExpectQuery(relationDataSchemaSQL).WillReturnRows(sqlmock.NewRows([]string{"relation_columns", "constraint_columns", "relation_checks", "relation_fks", "relation_pks", "relation_uniques", "constraint_checks", "constraint_fks", "constraint_pks", "relation_indexes", "constraint_indexes", "triggers"}).AddRow(relationColumnSignature, physicalConstraintColumnSignature, 7, 2, 1, 1, 7, 2, 1, 4, 3, 0))
	mock.ExpectQuery(frozenChainNodeRelationAggregateSQL).WillReturnRows(sqlmock.NewRows([]string{"total", "subcategory", "component", "input", "depends", "incomplete", "self", "duplicate", "orphan"}).AddRow(96, 95, 1, 0, 0, 0, 0, 0, 0))
	if err := verifyFrozenChainNodeRelationPostWrite(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
