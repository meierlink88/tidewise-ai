package seed

import (
	"context"
	"database/sql"
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

func TestMemoryRepositoryTopologyOnlyUsesPersistedMemberships(t *testing.T) {
	repo := NewMemoryRepository()
	full := validIndustryChainBatch()
	if _, err := repo.UpsertIndustryChainBatch(context.Background(), IndustryChainBatch{Memberships: full.Memberships}); err != nil {
		t.Fatalf("seed memberships: %v", err)
	}
	report, err := repo.UpsertIndustryChainBatch(context.Background(), IndustryChainBatch{TopologyEdges: full.TopologyEdges})
	if err != nil {
		t.Fatalf("topology-only batch: %v", err)
	}
	if report.Created != 1 {
		t.Fatalf("report = %+v", report)
	}
	if _, err := NewMemoryRepository().UpsertIndustryChainBatch(context.Background(), IndustryChainBatch{TopologyEdges: full.TopologyEdges}); err == nil || !strings.Contains(err.Error(), "membership") {
		t.Fatalf("missing persisted membership error = %v", err)
	}
}

func TestPostgresRepositoryTopologyOnlyChecksPersistedMemberships(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status.*FOR SHARE").WithArgs("chain", "node-a").WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("active"))
	mock.ExpectQuery("SELECT status.*FOR SHARE").WithArgs("chain", "node-b").WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("active"))
	mock.ExpectQuery("INSERT INTO industry_chain_topology_edges").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("created"))
	mock.ExpectCommit()

	full := validIndustryChainBatch()
	full.TopologyEdges[0].FromChainNodeEntityID = "node-b"
	full.TopologyEdges[0].ToChainNodeEntityID = "node-a"
	report, err := NewPostgresRepository(db).UpsertIndustryChainBatch(context.Background(), IndustryChainBatch{TopologyEdges: full.TopologyEdges})
	if err != nil {
		t.Fatalf("topology-only batch: %v", err)
	}
	if report.Created != 1 {
		t.Fatalf("report = %+v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRepositoryTopologyOnlyRollsBackMissingOrInactiveMembership(t *testing.T) {
	for name, rows := range map[string]*sqlmock.Rows{
		"missing":  nil,
		"inactive": sqlmock.NewRows([]string{"status"}).AddRow("inactive"),
	} {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			query := mock.ExpectQuery("SELECT status.*FOR SHARE").WithArgs("chain", "node-a")
			if rows == nil {
				query.WillReturnError(sql.ErrNoRows)
			} else {
				query.WillReturnRows(rows)
			}
			mock.ExpectRollback()

			full := validIndustryChainBatch()
			if _, err := NewPostgresRepository(db).UpsertIndustryChainBatch(context.Background(), IndustryChainBatch{TopologyEdges: full.TopologyEdges}); err == nil || !strings.Contains(err.Error(), "membership") {
				t.Fatalf("topology-only error = %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTopologyMembershipStatusQueryUsesUpdateConflictingSharedLock(t *testing.T) {
	lower := strings.ToLower(industryChainMembershipStatusSQL)
	if !strings.Contains(lower, "for share") {
		t.Fatal("topology endpoint validation must lock membership rows FOR SHARE")
	}
	if !strings.Contains(strings.ToLower(industryChainMembershipUpsertSQL), "do update") {
		t.Fatal("membership upsert must update the locked membership row")
	}
}

func TestMemoryRepositoryConstraintOnlyUsesPersistedSubjectsAndApprovalGate(t *testing.T) {
	repo := NewMemoryRepository()
	full := validIndustryChainBatch()
	if _, err := repo.UpsertIndustryChainBatch(context.Background(), IndustryChainBatch{Memberships: full.Memberships, TopologyEdges: full.TopologyEdges}); err != nil {
		t.Fatalf("seed subjects: %v", err)
	}
	constraint := full.PhysicalConstraints[0]
	constraint.GeneratedByAI = true
	batch := IndustryChainBatch{PhysicalConstraints: []domain.IndustryChainPhysicalConstraint{constraint}, ApprovalGate: domain.IndustryChainApprovalGate{HumanApprovedConstraintIDs: map[string]struct{}{"constraint": {}}}}
	if _, err := repo.UpsertIndustryChainBatch(context.Background(), batch); err != nil {
		t.Fatalf("constraint-only batch: %v", err)
	}
	if _, err := NewMemoryRepository().UpsertIndustryChainBatch(context.Background(), batch); err == nil || !strings.Contains(err.Error(), "membership") {
		t.Fatalf("missing persisted subject error = %v", err)
	}
	batch.ApprovalGate = domain.IndustryChainApprovalGate{}
	if _, err := repo.UpsertIndustryChainBatch(context.Background(), batch); err == nil || !strings.Contains(err.Error(), "human approval") {
		t.Fatalf("missing approval gate error = %v", err)
	}
}

func TestPostgresRepositoryConstraintOnlyLocksPersistedNodeSubject(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status.*FOR SHARE").WithArgs("chain", "node-a").WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("active"))
	mock.ExpectQuery("INSERT INTO industry_chain_physical_constraints").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("created"))
	mock.ExpectCommit()
	full := validIndustryChainBatch()
	constraint := full.PhysicalConstraints[0]
	constraint.GeneratedByAI = true
	batch := IndustryChainBatch{PhysicalConstraints: []domain.IndustryChainPhysicalConstraint{constraint}, ApprovalGate: domain.IndustryChainApprovalGate{HumanApprovedConstraintIDs: map[string]struct{}{"constraint": {}}}}
	if _, err := NewPostgresRepository(db).UpsertIndustryChainBatch(context.Background(), batch); err != nil {
		t.Fatalf("constraint-only batch: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRepositoryConstraintOnlyLocksSubjectsInStableOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status.*FOR SHARE").WithArgs("chain", "node-a").WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("active"))
	mock.ExpectQuery("SELECT status.*FOR SHARE").WithArgs("chain", "node-b").WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("active"))
	mock.ExpectQuery("INSERT INTO industry_chain_physical_constraints").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("created"))
	mock.ExpectQuery("INSERT INTO industry_chain_physical_constraints").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("created"))
	mock.ExpectCommit()
	full := validIndustryChainBatch()
	first := full.PhysicalConstraints[0]
	first.ID, first.ChainNodeEntityID = "constraint-b", "node-b"
	second := full.PhysicalConstraints[0]
	second.ID = "constraint-a"
	batch := IndustryChainBatch{PhysicalConstraints: []domain.IndustryChainPhysicalConstraint{first, second}}
	if _, err := NewPostgresRepository(db).UpsertIndustryChainBatch(context.Background(), batch); err != nil {
		t.Fatalf("constraint-only batch: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRepositoryConstraintOnlyLocksPersistedTopologySubject(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT industry_chain_entity_id, status.*FOR SHARE").WithArgs("edge").WillReturnRows(sqlmock.NewRows([]string{"industry_chain_entity_id", "status"}).AddRow("chain", "active"))
	mock.ExpectQuery("INSERT INTO industry_chain_physical_constraints").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("created"))
	mock.ExpectCommit()
	full := validIndustryChainBatch()
	constraint := full.PhysicalConstraints[0]
	constraint.ChainNodeEntityID, constraint.TopologyEdgeID = "", "edge"
	if _, err := NewPostgresRepository(db).UpsertIndustryChainBatch(context.Background(), IndustryChainBatch{PhysicalConstraints: []domain.IndustryChainPhysicalConstraint{constraint}}); err != nil {
		t.Fatalf("constraint-only batch: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRepositoryConstraintOnlyRollsBackIdentityConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status.*FOR SHARE").WithArgs("chain", "node-a").WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("active"))
	mock.ExpectQuery("INSERT INTO industry_chain_physical_constraints").WillReturnRows(sqlmock.NewRows([]string{"action"}).AddRow("identity_conflict"))
	mock.ExpectRollback()
	full := validIndustryChainBatch()
	if _, err := NewPostgresRepository(db).UpsertIndustryChainBatch(context.Background(), IndustryChainBatch{PhysicalConstraints: full.PhysicalConstraints}); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("identity conflict error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRepositoryConstraintOnlyRollsBackMissingOrInactiveSubject(t *testing.T) {
	for name, rows := range map[string]*sqlmock.Rows{"missing": nil, "inactive": sqlmock.NewRows([]string{"status"}).AddRow("inactive")} {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			query := mock.ExpectQuery("SELECT status.*FOR SHARE").WithArgs("chain", "node-a")
			if rows == nil {
				query.WillReturnError(sql.ErrNoRows)
			} else {
				query.WillReturnRows(rows)
			}
			mock.ExpectRollback()
			full := validIndustryChainBatch()
			batch := IndustryChainBatch{PhysicalConstraints: full.PhysicalConstraints}
			if _, err := NewPostgresRepository(db).UpsertIndustryChainBatch(context.Background(), batch); err == nil || !strings.Contains(err.Error(), "membership") {
				t.Fatalf("constraint-only error = %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
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
