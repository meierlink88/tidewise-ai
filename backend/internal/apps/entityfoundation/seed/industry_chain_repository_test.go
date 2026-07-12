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
	if first.Created != 5 || second.Unchanged != 5 {
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

func TestMemoryRepositoryRejectsIndustryChainIdentityMutationAtomically(t *testing.T) {
	repo := NewMemoryRepository()
	batch := validIndustryChainBatch()
	if _, err := repo.UpsertIndustryChainBatch(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	changed := batch
	changed.Memberships = append([]domain.IndustryChainMembership(nil), batch.Memberships...)
	changed.Memberships[0].ChainNodeEntityID = "node-c"
	changed.TopologyEdges = append([]domain.IndustryChainTopologyEdge(nil), batch.TopologyEdges...)
	changed.TopologyEdges[0].FromChainNodeEntityID = "node-c"
	changed.PhysicalConstraints = append([]domain.IndustryChainPhysicalConstraint(nil), batch.PhysicalConstraints...)
	changed.PhysicalConstraints[0].ChainNodeEntityID = "node-c"
	if _, err := repo.UpsertIndustryChainBatch(context.Background(), changed); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("identity mutation error = %v", err)
	}
	if got := repo.IndustryChainRowCount(); got != 5 {
		t.Fatalf("row count after rejected identity mutation = %d", got)
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
		if !strings.Contains(lower, "identity_conflict") {
			t.Fatalf("%s upsert lacks immutable identity guard", name)
		}
		if name == "constraint" && !strings.Contains(lower, "generated_by_ai") {
			t.Fatal("constraint upsert does not preserve AI provenance")
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

func TestIndustryChainRepositoriesShareAssociationAndApprovalValidation(t *testing.T) {
	invalid := validIndustryChainBatch()
	invalid.TopologyEdges[0].ToChainNodeEntityID = "missing"
	for name, run := range map[string]func(IndustryChainBatch) error{
		"memory": func(batch IndustryChainBatch) error {
			_, err := NewMemoryRepository().UpsertIndustryChainBatch(context.Background(), batch)
			return err
		},
		"postgres": func(batch IndustryChainBatch) error {
			db, _, _ := sqlmock.New()
			defer db.Close()
			_, err := NewPostgresRepository(db).UpsertIndustryChainBatch(context.Background(), batch)
			return err
		},
	} {
		if err := run(invalid); err == nil || !strings.Contains(err.Error(), "membership") {
			t.Fatalf("%s invalid association error = %v", name, err)
		}
	}

	aiApproved := validIndustryChainBatch()
	aiApproved.PhysicalConstraints[0].GeneratedByAI = true
	if _, err := NewMemoryRepository().UpsertIndustryChainBatch(context.Background(), aiApproved); err == nil || !strings.Contains(err.Error(), "human approval") {
		t.Fatalf("unapproved AI error = %v", err)
	}
	aiApproved.ApprovalGate = domain.IndustryChainApprovalGate{HumanApprovedConstraintIDs: map[string]struct{}{"constraint": {}}}
	if _, err := NewMemoryRepository().UpsertIndustryChainBatch(context.Background(), aiApproved); err != nil {
		t.Fatalf("human-approved AI error = %v", err)
	}
}

func TestPostgresRepositoryRollsBackIdentityConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("industry_chain_profiles").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("unchanged"))
	mock.ExpectQuery("industry_chain_memberships").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("identity_conflict"))
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).UpsertIndustryChainBatch(context.Background(), validIndustryChainBatch()); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("identity conflict error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func validIndustryChainBatch() IndustryChainBatch {
	verifiedAt := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	return IndustryChainBatch{
		Profiles: []domain.IndustryChainProfile{{EntityID: "chain", ChainCode: "test", Definition: "test", ScopeType: domain.IndustryChainScopeGlobal, Version: 1, ReviewStatus: domain.ReviewStatusApproved, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt}},
		Memberships: []domain.IndustryChainMembership{
			{ID: "membership-a", IndustryChainEntityID: "chain", ChainNodeEntityID: "node-a", StageCode: domain.IndustryChainStageUpstream, RoleCode: domain.IndustryChainRoleComponent, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: domain.StatusActive},
			{ID: "membership-b", IndustryChainEntityID: "chain", ChainNodeEntityID: "node-b", StageCode: domain.IndustryChainStageDownstream, RoleCode: domain.IndustryChainRoleProduct, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: domain.StatusActive},
		},
		TopologyEdges:       []domain.IndustryChainTopologyEdge{{ID: "edge", IndustryChainEntityID: "chain", FromChainNodeEntityID: "node-a", ToChainNodeEntityID: "node-b", RelationType: domain.IndustryChainRelationSuppliesTo, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: domain.StatusActive}},
		PhysicalConstraints: []domain.IndustryChainPhysicalConstraint{{ID: "constraint", IndustryChainEntityID: "chain", ChainNodeEntityID: "node-a", ConstraintType: domain.PhysicalConstraintPowerCapacity, Mechanism: "power", SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, ReviewStatus: domain.ReviewStatusApproved, Status: domain.StatusActive}},
	}
}
