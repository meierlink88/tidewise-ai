package seed

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestFirstBatchExternalIdentifierValidation(t *testing.T) {
	valid := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	if err := validateFirstBatchExternalIdentifier(valid); err != nil {
		t.Fatalf("validateFirstBatchExternalIdentifier() error = %v", err)
	}

	for _, candidate := range []domain.EntityExternalIdentifier{
		firstBatchExternalIdentifier("chain_node:3d_printing", "wind", "concept_sector", "BK0619", "3D打印"),
		firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "概念板块、行业板块", "BK0619", "3D打印"),
		firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "", "BK0619", "3D打印"),
	} {
		if err := validateFirstBatchExternalIdentifier(candidate); err == nil {
			t.Fatalf("validateFirstBatchExternalIdentifier(%+v) error = nil", candidate)
		}
	}
}

func TestMemoryRepositoryUpsertsExternalIdentifiersIdempotentlyAndRejectsRebinding(t *testing.T) {
	repo := NewMemoryRepository()
	for _, entity := range []Entity{
		{Key: "chain_node:3d_printing", EntityType: domain.EntityTypeChainNode, LayerCode: "chain_node", Name: "3D打印", CanonicalName: "3D打印", Status: domain.StatusActive},
		{Key: "chain_node:additive_manufacturing", EntityType: domain.EntityTypeChainNode, LayerCode: "chain_node", Name: "增材制造", CanonicalName: "增材制造", Status: domain.StatusActive},
	} {
		if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
			t.Fatal(err)
		}
	}

	identifier := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	created, err := repo.UpsertExternalIdentifier(context.Background(), identifier)
	if err != nil || created.Action != WriteCreated {
		t.Fatalf("first upsert = %+v, %v", created, err)
	}
	unchanged, err := repo.UpsertExternalIdentifier(context.Background(), identifier)
	if err != nil || unchanged.Action != WriteUnchanged {
		t.Fatalf("second upsert = %+v, %v", unchanged, err)
	}

	identifier.ExternalName = "3D 打印"
	updated, err := repo.UpsertExternalIdentifier(context.Background(), identifier)
	if err != nil || updated.Action != WriteUpdated {
		t.Fatalf("updated upsert = %+v, %v", updated, err)
	}

	rebound := firstBatchExternalIdentifier("chain_node:additive_manufacturing", "eastmoney", "concept_sector", "BK0619", "增材制造")
	if _, err := repo.UpsertExternalIdentifier(context.Background(), rebound); err == nil || !strings.Contains(err.Error(), "identity conflict") {
		t.Fatalf("rebound error = %v", err)
	}
}

func TestPostgresExternalIdentifierUpsertEnforcesTargetAndIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	identifier := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	identity := externalIdentifierIdentity(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).
		WithArgs(identity).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM entity_nodes")).
		WithArgs(identifier.EntityID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(identifier.EntityID))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, entity_id, external_name, status FROM entity_external_identifiers")).
		WithArgs(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode).
		WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "external_name", "status"}))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO entity_external_identifiers")).
		WithArgs(identifier.ID, identifier.EntityID, identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode, identifier.ExternalName, identifier.Status).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(identifier.ID))
	mock.ExpectCommit()
	result, err := NewPostgresRepository(db).UpsertExternalIdentifier(context.Background(), identifier)
	if err != nil || result.Action != WriteCreated {
		t.Fatalf("UpsertExternalIdentifier() = %+v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	statement := strings.ToLower(externalIdentifierTargetSQL() + externalIdentifierSelectSQL() + externalIdentifierInsertSQL())
	for _, required := range []string{
		"entity_type = 'chain_node'",
		"status = 'active'",
		"for share",
		"for update",
		"on conflict (source_system, source_taxonomy_type, external_code) do nothing",
	} {
		if !strings.Contains(statement, required) {
			t.Fatalf("external identifier transaction SQL missing %q", required)
		}
	}

}

func TestPostgresExternalIdentifierUpsertRollsBackSerializedRebindConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	identifier := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	identity := externalIdentifierIdentity(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).
		WithArgs(identity).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM entity_nodes")).
		WithArgs(identifier.EntityID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(identifier.EntityID))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, entity_id, external_name, status FROM entity_external_identifiers")).
		WithArgs(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode).
		WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "external_name", "status"}))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO entity_external_identifiers")).
		WithArgs(identifier.ID, identifier.EntityID, identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode, identifier.ExternalName, identifier.Status).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, entity_id, external_name, status FROM entity_external_identifiers")).
		WithArgs(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode).
		WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "external_name", "status"}).AddRow(identifier.ID, entitySeedUUID("chain_node:other"), identifier.ExternalName, identifier.Status))
	mock.ExpectRollback()

	if _, err := NewPostgresRepository(db).UpsertExternalIdentifier(context.Background(), identifier); err == nil || !strings.Contains(err.Error(), "identity conflict") {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresExternalIdentifierUpsertUpdatesSameEntityAfterLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	identifier := firstBatchExternalIdentifier("chain_node:3d_printing", "eastmoney", "concept_sector", "BK0619", "3D打印")
	identity := externalIdentifierIdentity(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).WithArgs(identity).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM entity_nodes")).WithArgs(identifier.EntityID).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(identifier.EntityID))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, entity_id, external_name, status FROM entity_external_identifiers")).
		WithArgs(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode).
		WillReturnRows(sqlmock.NewRows([]string{"id", "entity_id", "external_name", "status"}).AddRow(identifier.ID, identifier.EntityID, "旧名称", domain.StatusInactive))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE entity_external_identifiers")).
		WithArgs(identifier.ExternalName, identifier.Status, identifier.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := NewPostgresRepository(db).UpsertExternalIdentifier(context.Background(), identifier)
	if err != nil || result.Action != WriteUpdated {
		t.Fatalf("result = %+v, error = %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func firstBatchExternalIdentifier(entityKey, source, taxonomy, code, name string) domain.EntityExternalIdentifier {
	identity := externalIdentifierIdentity(source, taxonomy, code)
	return domain.EntityExternalIdentifier{
		ID:                 externalIdentifierSeedUUID(identity),
		EntityID:           entitySeedUUID(entityKey),
		SourceSystem:       source,
		SourceTaxonomyType: taxonomy,
		ExternalCode:       code,
		ExternalName:       name,
		Status:             domain.StatusActive,
	}
}
