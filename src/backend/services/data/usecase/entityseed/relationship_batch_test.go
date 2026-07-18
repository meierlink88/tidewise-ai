package seed

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestPostgresRelationshipBatchLocksPersistedEndpointsAndCommits(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	relationship := reviewedSectorMapping("relationship:test", "chain_node:test", "sector:test")
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT entity_key, entity_type, status.*FOR SHARE").WithArgs("chain_node:test").WillReturnRows(sqlmock.NewRows([]string{"entity_key", "entity_type", "status"}).AddRow("chain_node:test", domain.EntityTypeChainNode, domain.StatusActive))
	mock.ExpectQuery("SELECT entity_key, entity_type, status.*FOR SHARE").WithArgs("sector:test").WillReturnRows(sqlmock.NewRows([]string{"entity_key", "entity_type", "status"}).AddRow("sector:test", domain.EntityTypeSector, domain.StatusActive))
	mock.ExpectQuery("WITH existing AS").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("created"))
	mock.ExpectCommit()
	results, err := NewPostgresRepository(db).UpsertRelationshipBatch(context.Background(), []Relationship{relationship})
	if err != nil || len(results) != 1 || results[0].Action != WriteCreated {
		t.Fatalf("UpsertRelationshipBatch() = %+v, %v", results, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRelationshipBatchRollsBackMissingEndpointAndIdentityConflict(t *testing.T) {
	for _, test := range []struct {
		name       string
		secondRows *sqlmock.Rows
		action     string
	}{
		{name: "missing endpoint"},
		{name: "identity conflict", secondRows: sqlmock.NewRows([]string{"entity_key", "entity_type", "status"}).AddRow("sector:test", domain.EntityTypeSector, domain.StatusActive), action: "identity_conflict"},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT entity_key, entity_type, status.*FOR SHARE").WithArgs("chain_node:test").WillReturnRows(sqlmock.NewRows([]string{"entity_key", "entity_type", "status"}).AddRow("chain_node:test", domain.EntityTypeChainNode, domain.StatusActive))
			endpoint := mock.ExpectQuery("SELECT entity_key, entity_type, status.*FOR SHARE").WithArgs("sector:test")
			if test.secondRows == nil {
				endpoint.WillReturnRows(sqlmock.NewRows([]string{"entity_key", "entity_type", "status"}))
			} else {
				endpoint.WillReturnRows(test.secondRows)
				mock.ExpectQuery("WITH existing AS").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow(test.action))
			}
			mock.ExpectRollback()
			_, err = NewPostgresRepository(db).UpsertRelationshipBatch(context.Background(), []Relationship{reviewedSectorMapping("relationship:test", "chain_node:test", "sector:test")})
			if err == nil || (!strings.Contains(err.Error(), "missing") && !strings.Contains(err.Error(), "identity_conflict")) {
				t.Fatalf("error = %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMemoryRelationshipBatchRejectsPolicyAndIdentityAtomically(t *testing.T) {
	repo := NewMemoryRepository()
	for _, entity := range []Entity{
		{Key: "chain_node:test", EntityType: domain.EntityTypeChainNode, LayerCode: "industry_chain", Name: "节点", CanonicalName: "节点", Aliases: []string{"Test Node"}, Status: domain.StatusActive},
		{Key: "sector:test", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "板块", CanonicalName: "板块", Aliases: []string{"Test Sector"}, Status: domain.StatusActive},
		{Key: "sector:other", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "其他", CanonicalName: "其他", Aliases: []string{"Other Sector"}, Status: domain.StatusActive},
	} {
		if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
			t.Fatal(err)
		}
	}
	original := reviewedSectorMapping("relationship:identity", "chain_node:test", "sector:test")
	if _, err := repo.UpsertRelationshipBatch(context.Background(), []Relationship{original}); err != nil {
		t.Fatal(err)
	}
	first := reviewedSectorMapping("relationship:first", "chain_node:test", "sector:test")
	conflict := reviewedSectorMapping("relationship:identity", "chain_node:test", "sector:other")
	if _, err := repo.UpsertRelationshipBatch(context.Background(), []Relationship{first, conflict}); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("identity error = %v", err)
	}
	results, err := repo.UpsertRelationshipBatch(context.Background(), []Relationship{first})
	if err != nil || results[0].Action != WriteCreated {
		t.Fatalf("atomic rollback result = %+v, %v", results, err)
	}
	wrongDirection := reviewedSectorMapping("relationship:wrong", "sector:test", "chain_node:test")
	if _, err := repo.UpsertRelationshipBatch(context.Background(), []Relationship{wrongDirection}); err == nil || !strings.Contains(err.Error(), "does not allow") {
		t.Fatalf("policy error = %v", err)
	}
}

func reviewedSectorMapping(key, from, to string) Relationship {
	return Relationship{Key: key, From: from, To: to, RelationType: "mapped_to_sector", EvidenceNote: "composite curation", SourceName: "Tidewise review", SourceURL: "https://example.com/review", VerifiedAt: time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC), Status: domain.StatusActive}
}
