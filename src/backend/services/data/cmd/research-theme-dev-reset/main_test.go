package main

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
)

const testResetLockKey = "tidewise:research-theme-dev-reset:v1"

func TestValidateResetTargetRequiresLocalLoopbackDatabase(t *testing.T) {
	valid := []config.Config{
		{
			App:      config.AppConfig{Env: config.EnvLocal},
			Database: config.DatabaseConfig{Host: "localhost", Name: "tidewise_local"},
		},
		{
			App: config.AppConfig{Env: config.EnvLocal},
			Secrets: config.SecretConfig{
				DatabaseURL: "postgres://user:secret@127.0.0.1:5432/tidewise_local?sslmode=disable",
			},
		},
		{
			App: config.AppConfig{Env: config.EnvLocal},
			Secrets: config.SecretConfig{
				DatabaseURL: "postgres://user:secret@[::1]:5432/tidewise_local?sslmode=disable",
			},
		},
	}
	for index, cfg := range valid {
		if err := validateResetTarget(cfg); err != nil {
			t.Fatalf("valid target %d error = %v", index, err)
		}
	}

	invalid := []config.Config{
		{App: config.AppConfig{Env: config.EnvUAT}, Database: config.DatabaseConfig{Host: "localhost", Name: "tidewise_local"}},
		{App: config.AppConfig{Env: config.EnvProd}, Database: config.DatabaseConfig{Host: "127.0.0.1", Name: "tidewise_local"}},
		{App: config.AppConfig{Env: config.EnvLocal}, Database: config.DatabaseConfig{Host: "postgres", Name: "tidewise_local"}},
		{App: config.AppConfig{Env: config.EnvLocal}, Database: config.DatabaseConfig{Host: "db.internal", Name: "tidewise_local"}},
		{App: config.AppConfig{Env: config.EnvLocal}, Database: config.DatabaseConfig{Host: "localhost", Name: "tidewise_shared"}},
		{App: config.AppConfig{Env: config.EnvLocal}, Secrets: config.SecretConfig{DatabaseURL: "://invalid"}},
	}
	for index, cfg := range invalid {
		if err := validateResetTarget(cfg); err == nil {
			t.Fatalf("invalid target %d was accepted", index)
		}
	}
}

func TestValidateExecutionGate(t *testing.T) {
	if err := validateExecutionGate(resetOptions{}); err != nil {
		t.Fatalf("dry-run gate error = %v", err)
	}
	if err := validateExecutionGate(resetOptions{Execute: true, ConfirmDatabase: "tidewise_local"}); err != nil {
		t.Fatalf("execute gate error = %v", err)
	}

	for _, options := range []resetOptions{
		{Execute: true},
		{Execute: true, ConfirmDatabase: "tidewise-local"},
		{Execute: true, ConfirmDatabase: "TIDEWISE_LOCAL"},
	} {
		if err := validateExecutionGate(options); err == nil || !strings.Contains(err.Error(), "--confirm-database tidewise_local") {
			t.Fatalf("options %#v error = %v", options, err)
		}
	}
}

func TestRunResetDryRunOnlyReadsCounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	expectResetPreflight(mock, themeCountValues{16, 51, 0, 113, 4}, protectedCountValues{203, 981, 842, 0, 22, 19, 240, 2, 3, 0, 4})
	mock.ExpectCommit()

	report, err := runReset(context.Background(), db, resetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Mode != "dry-run" || report.Executed {
		t.Fatalf("report mode = %#v", report)
	}
	if report.Before.ResearchThemes != 16 || report.After != report.Before {
		t.Fatalf("theme counts = %#v", report)
	}
	if report.ProtectedBefore != report.ProtectedAfter || !report.TriggerRestored {
		t.Fatalf("protection report = %#v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunResetExecuteDeletesThemeDataAndRestoresTrigger(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	protected := protectedCountValues{203, 981, 842, 0, 22, 19, 240, 2, 3, 0, 4}
	expectResetPreflight(mock, themeCountValues{16, 51, 0, 113, 4}, protected)
	expectResetWrites(mock)
	expectTriggerEnabled(mock)
	expectThemeCounts(mock, themeCountValues{})
	expectProtectedCounts(mock, protected)
	mock.ExpectCommit()

	report, err := runReset(context.Background(), db, resetOptions{Execute: true, ConfirmDatabase: "tidewise_local"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Mode != "execute" || !report.Executed || !report.After.isZero() || !report.TriggerRestored {
		t.Fatalf("report = %#v", report)
	}
	if report.ProtectedBefore != report.ProtectedAfter {
		t.Fatalf("protected data changed: %#v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunResetRollsBackWhenReceiptDeleteFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	expectResetPreflight(mock, themeCountValues{1, 1, 0, 1, 1}, protectedCountValues{})
	mock.ExpectExec(regexp.QuoteMeta(disableReceiptTriggerSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(deleteThemesSQL)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(deleteReceiptsSQL)).WillReturnError(errors.New("delete failed"))
	mock.ExpectRollback()

	_, err = runReset(context.Background(), db, resetOptions{Execute: true, ConfirmDatabase: "tidewise_local"})
	if err == nil || !strings.Contains(err.Error(), "delete research theme import receipts") {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunResetRollsBackWhenProtectedCountsChange(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	before := protectedCountValues{203, 981, 842, 0, 22, 19, 240, 2, 3, 0, 4}
	after := before
	after.Events--
	expectResetPreflight(mock, themeCountValues{1, 1, 0, 1, 1}, before)
	expectResetWrites(mock)
	expectTriggerEnabled(mock)
	expectThemeCounts(mock, themeCountValues{})
	expectProtectedCounts(mock, after)
	mock.ExpectRollback()

	_, err = runReset(context.Background(), db, resetOptions{Execute: true, ConfirmDatabase: "tidewise_local"})
	if err == nil || !strings.Contains(err.Error(), "protected data counts changed") {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type themeCountValues struct {
	ResearchThemes              int64
	ResearchThemeChainNodes     int64
	ResearchThemeIndices        int64
	ResearchThemeEvents         int64
	ResearchThemeImportReceipts int64
}

type protectedCountValues struct {
	Events                   int64
	EntityNodes              int64
	ChainNodeProfiles        int64
	IndexProfiles            int64
	EventTagDefs             int64
	EventTagMaps             int64
	RawDocuments             int64
	ResearchAnchors          int64
	ResearchAnchorChainNodes int64
	ResearchAnchorIndices    int64
	ResearchAnchorEvents     int64
}

func expectResetPreflight(mock sqlmock.Sqlmock, themes themeCountValues, protected protectedCountValues) {
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(currentDatabaseSQL)).WillReturnRows(
		sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"),
	)
	mock.ExpectQuery(regexp.QuoteMeta(acquireResetLockSQL)).WithArgs(testResetLockKey).WillReturnRows(
		sqlmock.NewRows([]string{"locked"}).AddRow(true),
	)
	expectTriggerEnabled(mock)
	expectThemeCounts(mock, themes)
	expectProtectedCounts(mock, protected)
}

func expectResetWrites(mock sqlmock.Sqlmock) {
	mock.ExpectExec(regexp.QuoteMeta(disableReceiptTriggerSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(deleteThemesSQL)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(deleteReceiptsSQL)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(enableReceiptTriggerSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
}

func expectTriggerEnabled(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(receiptTriggerStateSQL)).WillReturnRows(
		sqlmock.NewRows([]string{"tgenabled"}).AddRow("O"),
	)
}

func expectThemeCounts(mock sqlmock.Sqlmock, counts themeCountValues) {
	mock.ExpectQuery(regexp.QuoteMeta(themeCountsSQL)).WillReturnRows(
		sqlmock.NewRows([]string{
			"research_themes",
			"research_theme_chain_nodes",
			"research_theme_indices",
			"research_theme_events",
			"research_theme_import_receipts",
		}).AddRow(
			counts.ResearchThemes,
			counts.ResearchThemeChainNodes,
			counts.ResearchThemeIndices,
			counts.ResearchThemeEvents,
			counts.ResearchThemeImportReceipts,
		),
	)
}

func expectProtectedCounts(mock sqlmock.Sqlmock, counts protectedCountValues) {
	mock.ExpectQuery(regexp.QuoteMeta(protectedCountsSQL)).WillReturnRows(
		sqlmock.NewRows([]string{
			"events",
			"entity_nodes",
			"chain_node_profiles",
			"index_profiles",
			"event_tag_defs",
			"event_tag_maps",
			"raw_documents",
			"research_anchors",
			"research_anchor_chain_nodes",
			"research_anchor_indices",
			"research_anchor_events",
		}).AddRow(
			counts.Events,
			counts.EntityNodes,
			counts.ChainNodeProfiles,
			counts.IndexProfiles,
			counts.EventTagDefs,
			counts.EventTagMaps,
			counts.RawDocuments,
			counts.ResearchAnchors,
			counts.ResearchAnchorChainNodes,
			counts.ResearchAnchorIndices,
			counts.ResearchAnchorEvents,
		),
	)
}
