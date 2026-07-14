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

func batchRelation(id, from, to string) domain.ChainNodeRelation {
	return domain.ChainNodeRelation{ID: id, FromChainNodeEntityID: from, ToChainNodeEntityID: to, RelationType: domain.ChainNodeRelationSubcategoryOf, Mechanism: from + " 全部属于 " + to, ConditionNote: "批准边界", EvidenceNote: "内部 artifact 与双遍 Review", Provenance: "artifact;sha256;derivation_rule;ffb243e;main-serenity", VerifiedAt: time.Date(2026, 7, 14, 6, 11, 6, 0, time.UTC), Status: domain.StatusActive}
}

func expectRelationPlanCreated(mock sqlmock.Sqlmock, relation domain.ChainNodeRelation) {
	mock.ExpectExec(chainNodeRelationTransactionLockSQL()).WithArgs(relation.FromChainNodeEntityID + "|" + string(relation.RelationType) + "|" + relation.ToChainNodeEntityID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(chainNodeRelationActiveEndpointsSQL(true)).WithArgs(relation.FromChainNodeEntityID, relation.ToChainNodeEntityID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(chainNodeRelationByIDSQL(true)).WithArgs(relation.ID).WillReturnRows(sqlmock.NewRows(chainNodeRelationColumns))
	mock.ExpectQuery(chainNodeRelationByTupleSQL(true)).WithArgs(relation.FromChainNodeEntityID, relation.ToChainNodeEntityID, relation.RelationType).WillReturnRows(sqlmock.NewRows(chainNodeRelationColumns))
}

func TestApplyChainNodeRelationBatchRollsBackAllWritesWhenSecondInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first := batchRelation("00000000-0000-5000-8000-000000000001", "00000000-0000-5000-8000-000000000011", "00000000-0000-5000-8000-000000000012")
	second := batchRelation("00000000-0000-5000-8000-000000000002", "00000000-0000-5000-8000-000000000013", "00000000-0000-5000-8000-000000000014")
	mock.ExpectBegin()
	expectRelationPlanCreated(mock, first)
	expectRelationPlanCreated(mock, second)
	mock.ExpectQuery(chainNodeRelationInsertSQL()).WithArgs(relationArgs(first)...).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(first.ID))
	mock.ExpectQuery(chainNodeRelationInsertSQL()).WithArgs(relationArgs(second)...).WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).ApplyChainNodeRelationBatch(context.Background(), []domain.ChainNodeRelation{first, second}); err == nil || !strings.Contains(err.Error(), "insert failed") {
		t.Fatalf("error=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyChainNodeRelationBatchPlansWholeBatchBeforeFirstInsert(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first := batchRelation("00000000-0000-5000-8000-000000000001", "00000000-0000-5000-8000-000000000011", "00000000-0000-5000-8000-000000000012")
	second := batchRelation("00000000-0000-5000-8000-000000000002", "00000000-0000-5000-8000-000000000013", "00000000-0000-5000-8000-000000000014")
	mock.ExpectBegin()
	expectRelationPlanCreated(mock, first)
	mock.ExpectExec(chainNodeRelationTransactionLockSQL()).WithArgs(second.FromChainNodeEntityID + "|" + string(second.RelationType) + "|" + second.ToChainNodeEntityID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(chainNodeRelationActiveEndpointsSQL(true)).WithArgs(second.FromChainNodeEntityID, second.ToChainNodeEntityID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).ApplyChainNodeRelationBatch(context.Background(), []domain.ChainNodeRelation{first, second}); err == nil {
		t.Fatal("error=nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyChainNodeRelationBatchRollsBackWhenPrecommitAssertionFails(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	relation := batchRelation("00000000-0000-5000-8000-000000000001", "00000000-0000-5000-8000-000000000011", "00000000-0000-5000-8000-000000000012")
	mock.ExpectBegin()
	expectRelationPlanCreated(mock, relation)
	mock.ExpectQuery(chainNodeRelationInsertSQL()).WithArgs(relationArgs(relation)...).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(relation.ID))
	mock.ExpectQuery(chainNodeRelationByIDSQL(true)).WithArgs(relation.ID).WillReturnError(errors.New("assert failed"))
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).ApplyChainNodeRelationBatch(context.Background(), []domain.ChainNodeRelation{relation}); err == nil || !strings.Contains(err.Error(), "assert failed") {
		t.Fatalf("error=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDryRunChainNodeRelationBatchUsesReadOnlySnapshotAndDoesNotWrite(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	relation := batchRelation("00000000-0000-5000-8000-000000000001", "00000000-0000-5000-8000-000000000011", "00000000-0000-5000-8000-000000000012")
	mock.ExpectBegin()
	mock.ExpectQuery(chainNodeRelationActiveEndpointsSQL(false)).WithArgs(relation.FromChainNodeEntityID, relation.ToChainNodeEntityID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(chainNodeRelationByIDSQL(false)).WithArgs(relation.ID).WillReturnRows(sqlmock.NewRows(chainNodeRelationColumns))
	mock.ExpectQuery(chainNodeRelationByTupleSQL(false)).WithArgs(relation.FromChainNodeEntityID, relation.ToChainNodeEntityID, relation.RelationType).WillReturnRows(sqlmock.NewRows(chainNodeRelationColumns))
	mock.ExpectCommit()
	report, err := NewPostgresRepository(db).DryRunChainNodeRelationBatch(context.Background(), []domain.ChainNodeRelation{relation})
	if err != nil || report.Created != 1 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDryRunChainNodeRelationBatchTreatsSameVerifiedInstantAsUnchanged(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	relation := batchRelation("00000000-0000-5000-8000-000000000001", "00000000-0000-5000-8000-000000000011", "00000000-0000-5000-8000-000000000012")
	databaseRelation := relation
	databaseRelation.VerifiedAt = relation.VerifiedAt.In(time.FixedZone("database-session", 8*60*60))
	mock.ExpectBegin()
	mock.ExpectQuery(chainNodeRelationActiveEndpointsSQL(false)).WithArgs(relation.FromChainNodeEntityID, relation.ToChainNodeEntityID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(chainNodeRelationByIDSQL(false)).WithArgs(relation.ID).WillReturnRows(sqlmock.NewRows(chainNodeRelationColumns).AddRow(relationArgs(databaseRelation)...))
	mock.ExpectCommit()
	report, err := NewPostgresRepository(db).DryRunChainNodeRelationBatch(context.Background(), []domain.ChainNodeRelation{relation})
	if err != nil || report.Created != 0 || report.Updated != 0 || report.Unchanged != 1 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyChainNodeRelationBatchAcceptsSameVerifiedInstantAtPrecommit(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	relation := batchRelation("00000000-0000-5000-8000-000000000001", "00000000-0000-5000-8000-000000000011", "00000000-0000-5000-8000-000000000012")
	databaseRelation := relation
	databaseRelation.VerifiedAt = relation.VerifiedAt.In(time.FixedZone("database-session", 8*60*60))
	mock.ExpectBegin()
	expectRelationPlanCreated(mock, relation)
	mock.ExpectQuery(chainNodeRelationInsertSQL()).WithArgs(relationArgs(relation)...).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(relation.ID))
	mock.ExpectQuery(chainNodeRelationByIDSQL(true)).WithArgs(relation.ID).WillReturnRows(sqlmock.NewRows(chainNodeRelationColumns).AddRow(relationArgs(databaseRelation)...))
	mock.ExpectCommit()
	report, err := NewPostgresRepository(db).ApplyChainNodeRelationBatch(context.Background(), []domain.ChainNodeRelation{relation})
	if err != nil || report.Created != 1 || report.Updated != 0 || report.Unchanged != 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEqualChainNodeRelationKeepsScalarFieldsExact(t *testing.T) {
	relation := batchRelation("00000000-0000-5000-8000-000000000001", "00000000-0000-5000-8000-000000000011", "00000000-0000-5000-8000-000000000012")
	sameInstant := relation
	sameInstant.VerifiedAt = relation.VerifiedAt.In(time.FixedZone("database-session", 8*60*60))
	if !equalChainNodeRelation(relation, sameInstant) {
		t.Fatal("same verified instant must compare equal")
	}
	drifted := sameInstant
	drifted.EvidenceNote += " changed"
	if equalChainNodeRelation(relation, drifted) {
		t.Fatal("scalar drift must not compare equal")
	}
	zeroVerifiedAt := relation
	zeroVerifiedAt.VerifiedAt = time.Time{}
	if equalChainNodeRelation(relation, zeroVerifiedAt) {
		t.Fatal("zero and non-zero verified_at must not compare equal")
	}
}
