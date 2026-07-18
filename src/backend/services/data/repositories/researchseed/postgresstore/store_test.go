package postgresstore

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchseed"
)

func TestStoreAppliesResolvedManifestInOneTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	manifest := testManifest()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(resolveChainNodeQuery)).WithArgs("算力").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("33333333-3333-5333-8333-333333333333"))
	mock.ExpectQuery(regexp.QuoteMeta(resolveEventQuery)).WithArgs("22222222-2222-5222-8222-222222222222").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("INSERT INTO research_themes").
		WithArgs(manifest.Themes[0].ID, manifest.AnalysisBatchID, manifest.Themes[0].Name, manifest.Themes[0].OneLineConclusion,
			manifest.Themes[0].ImpactLevel, manifest.Themes[0].TransmissionPath, manifest.Themes[0].TradingDirection,
			manifest.Themes[0].TransmissionStage, manifest.Themes[0].NextCheckpoint, manifest.Themes[0].IndexImpactSummary,
			now.Add(-72*time.Hour), now, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM research_theme_chain_nodes").WithArgs(manifest.Themes[0].ID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM research_theme_indices").WithArgs(manifest.Themes[0].ID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM research_theme_events").WithArgs(manifest.Themes[0].ID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO research_theme_chain_nodes").
		WithArgs(manifest.Themes[0].ID, "33333333-3333-5333-8333-333333333333", domain.ResearchRelationDriver, "影响").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO research_theme_events").
		WithArgs(manifest.Themes[0].ID, "22222222-2222-5222-8222-222222222222", domain.ResearchEvidenceDriver, "支持").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	report, err := New(db).Apply(context.Background(), manifest, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.ThemeCount != 1 || report.ChainNodeCount != 1 || report.EventCount != 1 || !report.PublishedAt.Equal(now) {
		t.Fatalf("report = %#v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreRollsBackBeforeWritesWhenMasterReferenceIsMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	manifest := testManifest()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(resolveChainNodeQuery)).WithArgs("算力").WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	if _, err := New(db).Apply(context.Background(), manifest, time.Now()); err == nil {
		t.Fatal("Apply() error = nil, want missing chain-node error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func testManifest() researchseed.Manifest {
	return researchseed.Manifest{
		AnalysisBatchID: "batch",
		Themes: []researchseed.Theme{{
			ID: "11111111-1111-5111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
			ImpactLevel: domain.ImpactLevelHigh, TransmissionPath: "事件 -> 影响",
			TradingDirection: "等待验证", TransmissionStage: domain.TransmissionStageValidation,
			NextCheckpoint: "尚未显现", IndexImpactSummary: "未观察",
			ChainNodes: []researchseed.ChainNodeReference{{Name: "算力", RelationRole: domain.ResearchRelationDriver, ImpactSummary: "影响"}},
			Events:     []researchseed.EventReference{{ID: "22222222-2222-5222-8222-222222222222", EvidenceRole: domain.ResearchEvidenceDriver, SupportedClaim: "支持"}},
		}},
	}
}
