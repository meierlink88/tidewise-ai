package seed

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestMemoryRepositoryUpsertsIndustryChainBatchAtomicallyAndIdempotently(t *testing.T) {
	repo := NewMemoryRepository()
	batch := validIndustryChainBatch()
	first, err := repo.UpsertIndustryChainBatch(context.Background(), batch)
	if err != nil {
		t.Fatalf("UpsertIndustryChainBatch(first) error = %v", err)
	}
	second, err := repo.UpsertIndustryChainBatch(context.Background(), batch)
	if err != nil {
		t.Fatalf("UpsertIndustryChainBatch(second) error = %v", err)
	}
	if first.Created != 4 || second.Unchanged != 4 {
		t.Fatalf("reports = %+v / %+v", first, second)
	}

	invalid := batch
	invalid.TopologyEdges[0].ToChainNodeEntityID = "node-a"
	empty := NewMemoryRepository()
	if _, err := empty.UpsertIndustryChainBatch(context.Background(), invalid); err == nil {
		t.Fatal("invalid batch error = nil")
	}
	if got := empty.IndustryChainRowCount(); got != 0 {
		t.Fatalf("row count after rejected batch = %d", got)
	}
}

func TestIndustryChainPostgresUpsertsExposeUnchangedAction(t *testing.T) {
	for name, statement := range map[string]string{
		"profile":    industryChainProfileUpsertSQL,
		"membership": industryChainMembershipUpsertSQL,
		"topology":   industryChainTopologyUpsertSQL,
		"constraint": industryChainConstraintUpsertSQL,
	} {
		lower := strings.ToLower(statement)
		if !strings.Contains(lower, "is distinct from") || !strings.Contains(lower, "'unchanged'") {
			t.Fatalf("%s upsert lacks idempotent unchanged semantics", name)
		}
	}
}

func TestPostgresRepositoryRollsBackIndustryChainBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO industry_chain_profiles").WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()

	repo := NewPostgresRepository(db)
	if _, err := repo.UpsertIndustryChainBatch(context.Background(), validIndustryChainBatch()); err == nil {
		t.Fatal("UpsertIndustryChainBatch() error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func validIndustryChainBatch() IndustryChainBatch {
	verifiedAt := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	return IndustryChainBatch{
		Profiles:            []domain.IndustryChainProfile{{EntityID: "chain", ChainCode: "test", Definition: "test", ScopeType: domain.IndustryChainScopeGlobal, Version: 1, ReviewStatus: domain.ReviewStatusApproved, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt}},
		Memberships:         []domain.IndustryChainMembership{{ID: "membership", IndustryChainEntityID: "chain", ChainNodeEntityID: "node-a", StageCode: domain.IndustryChainStageUpstream, RoleCode: domain.IndustryChainRoleComponent, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: domain.StatusActive}},
		TopologyEdges:       []domain.IndustryChainTopologyEdge{{ID: "edge", IndustryChainEntityID: "chain", FromChainNodeEntityID: "node-a", ToChainNodeEntityID: "node-b", RelationType: domain.IndustryChainRelationSuppliesTo, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: domain.StatusActive}},
		PhysicalConstraints: []domain.IndustryChainPhysicalConstraint{{ID: "constraint", IndustryChainEntityID: "chain", ChainNodeEntityID: "node-a", ConstraintType: domain.PhysicalConstraintPowerCapacity, Mechanism: "power", SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, ReviewStatus: domain.ReviewStatusApproved, Status: domain.StatusActive}},
	}
}
