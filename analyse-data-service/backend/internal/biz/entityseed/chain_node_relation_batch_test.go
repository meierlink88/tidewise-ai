package entityseed

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

func batchRelation(id, from, to string) model.ChainNodeRelation {
	return model.ChainNodeRelation{ID: id, FromChainNodeEntityID: from, ToChainNodeEntityID: to, RelationType: model.ChainNodeRelationSubcategoryOf, Mechanism: from + " 全部属于 " + to, ConditionNote: "批准边界", EvidenceNote: "内部 artifact 与双遍 Review", Provenance: "artifact;sha256;derivation_rule;ffb243e;main-serenity", VerifiedAt: time.Date(2026, 7, 14, 6, 11, 6, 0, time.UTC), Status: model.StatusActive}
}

func expectRelationPlanCreated(mock sqlmock.Sqlmock, relation model.ChainNodeRelation) {
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
	if _, err := NewPostgresRepository(db).ApplyChainNodeRelationBatch(context.Background(), []model.ChainNodeRelation{first, second}); err == nil || !strings.Contains(err.Error(), "insert failed") {
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
	if _, err := NewPostgresRepository(db).ApplyChainNodeRelationBatch(context.Background(), []model.ChainNodeRelation{first, second}); err == nil {
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
	if _, err := NewPostgresRepository(db).ApplyChainNodeRelationBatch(context.Background(), []model.ChainNodeRelation{relation}); err == nil || !strings.Contains(err.Error(), "assert failed") {
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
	report, err := NewPostgresRepository(db).DryRunChainNodeRelationBatch(context.Background(), []model.ChainNodeRelation{relation})
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
	report, err := NewPostgresRepository(db).DryRunChainNodeRelationBatch(context.Background(), []model.ChainNodeRelation{relation})
	if err != nil || report.Created != 0 || report.Updated != 0 || report.Unchanged != 1 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateFrozenChainNodeRelationDryRunBaselineAcceptsOnlyFrozenStates(t *testing.T) {
	tests := []struct {
		name    string
		report  ChainNodeRelationDataPreflightReport
		wantErr bool
	}{
		{
			name: "before write",
			report: ChainNodeRelationDataPreflightReport{
				ExistingRelations:    100,
				SubcategoryRelations: 95,
				ComponentRelations:   1,
				InputRelations:       3,
				DependsRelations:     1,
			},
		},
		{
			name: "after write",
			report: ChainNodeRelationDataPreflightReport{
				ExistingRelations:    212,
				SubcategoryRelations: 108,
				ComponentRelations:   3,
				InputRelations:       93,
				DependsRelations:     8,
			},
		},
		{
			name: "retired historical baseline",
			report: ChainNodeRelationDataPreflightReport{
				ExistingRelations:    96,
				SubcategoryRelations: 95,
				ComponentRelations:   1,
			},
			wantErr: true,
		},
		{
			name: "before write type drift",
			report: ChainNodeRelationDataPreflightReport{
				ExistingRelations:    100,
				SubcategoryRelations: 95,
				ComponentRelations:   1,
				InputRelations:       4,
				DependsRelations:     0,
			},
			wantErr: true,
		},
		{
			name: "after write type drift",
			report: ChainNodeRelationDataPreflightReport{
				ExistingRelations:    212,
				SubcategoryRelations: 108,
				ComponentRelations:   3,
				InputRelations:       94,
				DependsRelations:     7,
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateFrozenChainNodeRelationDryRunBaseline(test.report)
			if (err != nil) != test.wantErr {
				t.Fatalf("error=%v wantErr=%v", err, test.wantErr)
			}
		})
	}
}

func TestValidateFrozenChainNodeRelationPlanProtectsAcceptedHundredAndFinalCounts(t *testing.T) {
	typeCounts := map[model.ChainNodeRelationType]int{
		model.ChainNodeRelationSubcategoryOf: 108,
		model.ChainNodeRelationComponentOf:   3,
		model.ChainNodeRelationInputTo:       93,
		model.ChainNodeRelationDependsOn:     8,
	}
	before := ChainNodeRelationDataPreflightReport{ExistingRelations: 100, SubcategoryRelations: 95, ComponentRelations: 1, InputRelations: 3, DependsRelations: 1}
	post := ChainNodeRelationDataPreflightReport{ExistingRelations: 212, SubcategoryRelations: 108, ComponentRelations: 3, InputRelations: 93, DependsRelations: 8}
	if err := validateFrozenChainNodeRelationPlan(before, ChainNodeRelationReport{Created: 112, Updated: 0, Unchanged: 100, ByRelationType: typeCounts}); err != nil {
		t.Fatal(err)
	}
	if err := validateFrozenChainNodeRelationPlan(post, ChainNodeRelationReport{Created: 0, Updated: 0, Unchanged: 212, ByRelationType: typeCounts}); err != nil {
		t.Fatal(err)
	}
	tests := []ChainNodeRelationReport{
		{Created: 111, Updated: 0, Unchanged: 101, ByRelationType: typeCounts},
		{Created: 112, Updated: 1, Unchanged: 99, ByRelationType: typeCounts},
		{Created: 112, Updated: 0, Unchanged: 100, ByRelationType: map[model.ChainNodeRelationType]int{model.ChainNodeRelationSubcategoryOf: 107, model.ChainNodeRelationComponentOf: 3, model.ChainNodeRelationInputTo: 94, model.ChainNodeRelationDependsOn: 8}},
	}
	for _, report := range tests {
		if err := validateFrozenChainNodeRelationPlan(before, report); err == nil {
			t.Fatalf("plan drift accepted: %+v", report)
		}
	}
}

func TestValidateFrozenChainNodeRelationActionsRejectsBalancedAcceptedBaselineDrift(t *testing.T) {
	before := ChainNodeRelationDataPreflightReport{ExistingRelations: 100, SubcategoryRelations: 95, ComponentRelations: 1, InputRelations: 3, DependsRelations: 1}
	planned := frozenChainNodeRelationPlans(WriteUnchanged, WriteCreated)
	if err := validateFrozenChainNodeRelationActions(before, planned); err != nil {
		t.Fatal(err)
	}
	planned[0].action = WriteCreated
	planned[100].action = WriteUnchanged
	if report := relationReport(planned); report.Created != 112 || report.Updated != 0 || report.Unchanged != 100 {
		t.Fatalf("balanced drift changed aggregate report: %+v", report)
	}
	if err := validateFrozenChainNodeRelationActions(before, planned); err == nil {
		t.Fatal("accepted baseline drift passed because aggregate counts balanced")
	}
}

func frozenChainNodeRelationPlans(acceptedAction, additiveAction WriteAction) []plannedChainNodeRelation {
	planned := make([]plannedChainNodeRelation, 0, 212)
	appendPlans := func(count int, relationType model.ChainNodeRelationType, action WriteAction) {
		for range count {
			planned = append(planned, plannedChainNodeRelation{item: model.ChainNodeRelation{RelationType: relationType}, action: action})
		}
	}
	appendPlans(95, model.ChainNodeRelationSubcategoryOf, acceptedAction)
	appendPlans(1, model.ChainNodeRelationComponentOf, acceptedAction)
	appendPlans(3, model.ChainNodeRelationInputTo, acceptedAction)
	appendPlans(1, model.ChainNodeRelationDependsOn, acceptedAction)
	appendPlans(13, model.ChainNodeRelationSubcategoryOf, additiveAction)
	appendPlans(2, model.ChainNodeRelationComponentOf, additiveAction)
	appendPlans(90, model.ChainNodeRelationInputTo, additiveAction)
	appendPlans(7, model.ChainNodeRelationDependsOn, additiveAction)
	return planned
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
	report, err := NewPostgresRepository(db).ApplyChainNodeRelationBatch(context.Background(), []model.ChainNodeRelation{relation})
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
