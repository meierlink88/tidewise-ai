package seed

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestApplyExternalIdentifierBatchRollsBackBeforeWritesWhenTargetIsInvalid(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	item := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	mock.ExpectBegin()
	mock.ExpectExec("pg_advisory_xact_lock").WithArgs(externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id FROM entity_nodes").WithArgs(item.EntityID).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).ApplyExternalIdentifierBatch(context.Background(), []ExternalIdentifierMapping{mappingFromIdentifier(item)}); err == nil {
		t.Fatal("ApplyExternalIdentifierBatch() error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyExternalIdentifierBatchRollsBackAllWritesWhenSecondInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	second := firstBatchExternalIdentifier("chain_node:additive_manufacturing", "ths", "concept_sector", "301001", "增材制造")
	mock.ExpectBegin()
	for _, item := range []domain.EntityExternalIdentifier{first, second} {
		identity := externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode)
		mock.ExpectExec(regexp.QuoteMeta(externalIdentifierTransactionLockSQL())).WithArgs(identity).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM entity_nodes")).WithArgs(item.EntityID).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(item.EntityID))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, entity_id, external_name, status FROM entity_external_identifiers")).WithArgs(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode).WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "external_name", "status"}))
		mock.ExpectQuery(regexp.QuoteMeta("WHERE id = $1::uuid")).WithArgs(item.ID).WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "external_name", "status"}))
	}
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO entity_external_identifiers")).WithArgs(first.ID, first.EntityID, first.SourceSystem, first.SourceTaxonomyType, first.ExternalCode, first.ExternalName, first.Status).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(first.ID))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO entity_external_identifiers")).WithArgs(second.ID, second.EntityID, second.SourceSystem, second.SourceTaxonomyType, second.ExternalCode, second.ExternalName, second.Status).WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).ApplyExternalIdentifierBatch(context.Background(), []ExternalIdentifierMapping{mappingFromIdentifier(first), mappingFromIdentifier(second)}); err == nil || !strings.Contains(err.Error(), "insert") {
		t.Fatalf("ApplyExternalIdentifierBatch() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyExternalIdentifierBatchDoesNotInsertWhenSecondTargetFailsPlanning(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	second := firstBatchExternalIdentifier("chain_node:additive_manufacturing", "ths", "concept_sector", "301001", "增材制造")
	mock.ExpectBegin()
	for _, item := range []domain.EntityExternalIdentifier{first, second} {
		identity := externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode)
		mock.ExpectExec(regexp.QuoteMeta(externalIdentifierTransactionLockSQL())).WithArgs(identity).WillReturnResult(sqlmock.NewResult(0, 1))
		rows := sqlmock.NewRows([]string{"id"})
		if item.ID == first.ID {
			rows.AddRow(item.EntityID)
		}
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM entity_nodes")).WithArgs(item.EntityID).WillReturnRows(rows)
		if item.ID == first.ID {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT id, entity_id, external_name, status FROM entity_external_identifiers")).WithArgs(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode).WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "external_name", "status"}))
			mock.ExpectQuery(regexp.QuoteMeta("WHERE id = $1::uuid")).WithArgs(item.ID).WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "source_system", "source_taxonomy_type", "external_code", "external_name", "status"}))
		}
	}
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).ApplyExternalIdentifierBatch(context.Background(), []ExternalIdentifierMapping{mappingFromIdentifier(first), mappingFromIdentifier(second)}); err == nil {
		t.Fatal("ApplyExternalIdentifierBatch() error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyFrozenFirstBatchExternalIdentifiersRejectsExistingRowsBeforePlanning(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	item := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM entity_external_identifiers")).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).ApplyFrozenFirstBatchExternalIdentifiers(context.Background(), []ExternalIdentifierMapping{mappingFromIdentifier(item)}); err == nil || !strings.Contains(err.Error(), "zero existing") {
		t.Fatalf("ApplyFrozenFirstBatchExternalIdentifiers() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDryRunExternalIdentifierBatchUsesUnlockedTargetSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	item := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	mock.ExpectBegin()
	mock.ExpectQuery(externalIdentifierTargetSnapshotSQL()).WithArgs(item.EntityID).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(item.EntityID))
	mock.ExpectQuery(externalIdentifierSnapshotSQL()).WithArgs(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode).WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "external_name", "status"}))
	mock.ExpectQuery(externalIdentifierSnapshotByIDSQL()).WithArgs(item.ID).WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "source_system", "source_taxonomy_type", "external_code", "external_name", "status"}))
	mock.ExpectRollback()
	report, err := NewPostgresRepository(db).DryRunExternalIdentifierBatch(context.Background(), []ExternalIdentifierMapping{mappingFromIdentifier(item)})
	if err != nil || report.Created != 1 || report.Updated != 0 || report.Unchanged != 0 {
		t.Fatalf("DryRunExternalIdentifierBatch() = %+v, %v", report, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadExternalIdentifierMappingFileDecodesSnakeCaseAndRejectsDuplicateTriple(t *testing.T) {
	item := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	path := filepath.Join(t.TempDir(), "mappings.json")
	content := `{"mappings":[{"id":"` + item.ID + `","entity_id":"` + item.EntityID + `","source_system":"eastmoney","source_taxonomy_type":"concept_sector","external_code":"BK0619","external_name":"3D打印","status":"active"}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadExternalIdentifierMappingFile(path)
	if err != nil || len(manifest.Mappings) != 1 || manifest.Mappings[0].EntityID != item.EntityID {
		t.Fatalf("LoadExternalIdentifierMappingFile() = %+v, %v", manifest, err)
	}
	duplicate := strings.TrimSuffix(content, `]}`) + `,` + strings.TrimPrefix(content, `{"mappings":[`)
	if err := os.WriteFile(path, []byte(duplicate), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadExternalIdentifierMappingFile(path); err == nil || !strings.Contains(err.Error(), "duplicate external identifier") {
		t.Fatalf("duplicate manifest error = %v", err)
	}
}
