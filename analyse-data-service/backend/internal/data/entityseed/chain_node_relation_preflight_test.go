package entityseed

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPreflightFrozenChainNodeRelationDataRequiresExactGoose18SchemaAndBaseline(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(relationDataBaselineSQL).WillReturnRows(sqlmock.NewRows([]string{"database", "version", "goose", "nodes", "profiles", "external", "edges", "relations", "subcategory", "component", "input", "depends", "constraints"}).AddRow("tidewise_local", "16.14", 18, 842, 842, 1169, 241, 100, 95, 1, 3, 1, 0))
	mock.ExpectQuery(relationDataSchemaSQL).WillReturnRows(sqlmock.NewRows([]string{"relation_columns", "constraint_columns", "relation_checks", "relation_fks", "relation_pks", "relation_uniques", "constraint_checks", "constraint_fks", "constraint_pks", "relation_indexes", "constraint_indexes", "triggers"}).AddRow(relationColumnSignature, physicalConstraintColumnSignature, 7, 2, 1, 1, 7, 2, 1, 4, 3, 0))
	report, err := NewPostgresRepository(db).PreflightFrozenChainNodeRelationData(context.Background())
	if err != nil || !report.SchemaValid || report.GooseVersion != 18 || report.ExistingRelations != 100 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPreflightFrozenChainNodeRelationDataRejectsSchemaDrift(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(relationDataBaselineSQL).WillReturnRows(sqlmock.NewRows([]string{"database", "version", "goose", "nodes", "profiles", "external", "edges", "relations", "subcategory", "component", "input", "depends", "constraints"}).AddRow("tidewise_local", "16.14", 18, 842, 842, 1169, 241, 100, 95, 1, 3, 1, 0))
	mock.ExpectQuery(relationDataSchemaSQL).WillReturnRows(sqlmock.NewRows([]string{"relation_columns", "constraint_columns", "relation_checks", "relation_fks", "relation_pks", "relation_uniques", "constraint_checks", "constraint_fks", "constraint_pks", "relation_indexes", "constraint_indexes", "triggers"}).AddRow(relationColumnSignature, physicalConstraintColumnSignature, 6, 2, 1, 1, 7, 2, 1, 4, 3, 0))
	if _, err := NewPostgresRepository(db).PreflightFrozenChainNodeRelationData(context.Background()); err == nil {
		t.Fatal("schema drift error = nil")
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
	mock.ExpectQuery(relationDataBaselineSQL).WillReturnRows(sqlmock.NewRows([]string{"database", "version", "goose", "nodes", "profiles", "external", "edges", "relations", "subcategory", "component", "input", "depends", "constraints"}).AddRow("tidewise_local", "16.14", 18, 842, 842, 1169, 241, 212, 108, 3, 93, 8, 0))
	mock.ExpectQuery(relationDataSchemaSQL).WillReturnRows(sqlmock.NewRows([]string{"relation_columns", "constraint_columns", "relation_checks", "relation_fks", "relation_pks", "relation_uniques", "constraint_checks", "constraint_fks", "constraint_pks", "relation_indexes", "constraint_indexes", "triggers"}).AddRow(relationColumnSignature, physicalConstraintColumnSignature, 7, 2, 1, 1, 7, 2, 1, 4, 3, 0))
	mock.ExpectQuery(frozenChainNodeRelationAggregateSQL).WillReturnRows(sqlmock.NewRows([]string{"total", "subcategory", "component", "input", "depends", "incomplete", "self", "duplicate", "orphan"}).AddRow(212, 108, 3, 93, 8, 0, 0, 0, 0))
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
