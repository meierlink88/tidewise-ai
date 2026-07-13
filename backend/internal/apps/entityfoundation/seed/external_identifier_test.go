package seed

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
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
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO entity_external_identifiers")).
		WithArgs(identifier.ID, identifier.EntityID, identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode, identifier.ExternalName, identifier.Status).
		WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("created"))
	result, err := NewPostgresRepository(db).UpsertExternalIdentifier(context.Background(), identifier)
	if err != nil || result.Action != WriteCreated {
		t.Fatalf("UpsertExternalIdentifier() = %+v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	statement := strings.ToLower(buildExternalIdentifierUpsert())
	for _, required := range []string{
		"entity_type = 'chain_node'",
		"status = 'active'",
		"on conflict (source_system, source_taxonomy_type, external_code)",
		"entity_external_identifiers.entity_id = excluded.entity_id",
		"identity_conflict",
		"invalid_target",
	} {
		if !strings.Contains(statement, required) {
			t.Fatalf("external identifier upsert missing %q", required)
		}
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
