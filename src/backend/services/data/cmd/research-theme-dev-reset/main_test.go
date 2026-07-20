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

	expectResetPreflight(mock, publicationCounts{
		ResearchThemes:               16,
		ResearchThemeChainNodes:      51,
		ResearchThemeEvents:          113,
		ResearchThemeImportReceipts:  4,
		ResearchAnchorImportReceipts: 1,
		ResearchAnchors:              2,
		ResearchAnchorChainNodes:     3,
		ResearchAnchorEvents:         4,
	}, protectedCounts{
		Events:            203,
		EntityNodes:       981,
		ChainNodeProfiles: 842,
		EventTagDefs:      22,
		EventTagMaps:      19,
		RawDocuments:      240,
	})
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

	protected := protectedCounts{
		Events:            203,
		EntityNodes:       981,
		ChainNodeProfiles: 842,
		EventTagDefs:      22,
		EventTagMaps:      19,
		RawDocuments:      240,
	}
	expectResetPreflight(mock, publicationCounts{
		ResearchThemes:               16,
		ResearchThemeChainNodes:      51,
		ResearchThemeEvents:          113,
		ResearchThemeImportReceipts:  4,
		ResearchAnchorImportReceipts: 1,
		ResearchAnchors:              2,
		ResearchAnchorChainNodes:     3,
		ResearchAnchorEvents:         4,
	}, protected)
	expectResetWrites(mock)
	expectReceiptTriggersEnabled(mock)
	expectPublicationCounts(mock, publicationCounts{})
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

func TestRunResetRollsBackWhenAnchorReceiptDeleteFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	expectResetPreflight(mock, publicationCounts{
		ResearchThemes:               1,
		ResearchThemeChainNodes:      1,
		ResearchThemeEvents:          1,
		ResearchThemeImportReceipts:  1,
		ResearchAnchorImportReceipts: 1,
		ResearchAnchors:              1,
		ResearchAnchorChainNodes:     2,
		ResearchAnchorEvents:         1,
	}, protectedCounts{})
	mock.ExpectExec(regexp.QuoteMeta(disableThemeReceiptTriggerSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(disableAnchorReceiptTriggerSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(deleteAnchorsSQL)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(deleteAnchorReceiptsSQL)).WillReturnError(errors.New("delete failed"))
	mock.ExpectRollback()

	_, err = runReset(context.Background(), db, resetOptions{Execute: true, ConfirmDatabase: "tidewise_local"})
	if err == nil || !strings.Contains(err.Error(), "delete research anchor import receipts") {
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

	before := protectedCounts{
		Events:            203,
		EntityNodes:       981,
		ChainNodeProfiles: 842,
		EventTagDefs:      22,
		EventTagMaps:      19,
		RawDocuments:      240,
	}
	after := before
	after.Events--
	expectResetPreflight(mock, publicationCounts{
		ResearchThemes:               1,
		ResearchThemeChainNodes:      1,
		ResearchThemeEvents:          1,
		ResearchThemeImportReceipts:  1,
		ResearchAnchorImportReceipts: 1,
		ResearchAnchors:              1,
		ResearchAnchorChainNodes:     2,
		ResearchAnchorEvents:         1,
	}, before)
	expectResetWrites(mock)
	expectReceiptTriggersEnabled(mock)
	expectPublicationCounts(mock, publicationCounts{})
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

func TestRunResetRollsBackWhenLockIsUnavailable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(currentDatabaseSQL)).WillReturnRows(
		sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"),
	)
	mock.ExpectQuery(regexp.QuoteMeta(acquireResetLockSQL)).WithArgs(resetLockKey).WillReturnRows(
		sqlmock.NewRows([]string{"locked"}).AddRow(false),
	)
	mock.ExpectRollback()

	_, err = runReset(context.Background(), db, resetOptions{})
	if err == nil || !strings.Contains(err.Error(), "another research publication reset") {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectResetPreflight(mock sqlmock.Sqlmock, publications publicationCounts, protected protectedCounts) {
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(currentDatabaseSQL)).WillReturnRows(
		sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"),
	)
	mock.ExpectQuery(regexp.QuoteMeta(acquireResetLockSQL)).WithArgs(resetLockKey).WillReturnRows(
		sqlmock.NewRows([]string{"locked"}).AddRow(true),
	)
	expectReceiptTriggersEnabled(mock)
	expectPublicationCounts(mock, publications)
	expectProtectedCounts(mock, protected)
}

func expectResetWrites(mock sqlmock.Sqlmock) {
	mock.ExpectExec(regexp.QuoteMeta(disableThemeReceiptTriggerSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(disableAnchorReceiptTriggerSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(deleteAnchorsSQL)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(deleteAnchorReceiptsSQL)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(deleteThemesSQL)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(deleteThemeReceiptsSQL)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(enableAnchorReceiptTriggerSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(enableThemeReceiptTriggerSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
}

func expectReceiptTriggersEnabled(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(themeReceiptTriggerStateSQL)).WillReturnRows(
		sqlmock.NewRows([]string{"tgenabled"}).AddRow("O"),
	)
	mock.ExpectQuery(regexp.QuoteMeta(anchorReceiptTriggerStateSQL)).WillReturnRows(
		sqlmock.NewRows([]string{"tgenabled"}).AddRow("O"),
	)
}

func expectPublicationCounts(mock sqlmock.Sqlmock, counts publicationCounts) {
	mock.ExpectQuery(regexp.QuoteMeta(publicationCountsSQL)).WillReturnRows(
		sqlmock.NewRows([]string{
			"research_themes",
			"research_theme_chain_nodes",
			"research_theme_indices",
			"research_theme_events",
			"research_theme_import_receipts",
			"research_anchor_import_receipts",
			"research_anchors",
			"research_anchor_chain_nodes",
			"research_anchor_events",
		}).AddRow(
			counts.ResearchThemes,
			counts.ResearchThemeChainNodes,
			counts.ResearchThemeIndices,
			counts.ResearchThemeEvents,
			counts.ResearchThemeImportReceipts,
			counts.ResearchAnchorImportReceipts,
			counts.ResearchAnchors,
			counts.ResearchAnchorChainNodes,
			counts.ResearchAnchorEvents,
		),
	)
}

func expectProtectedCounts(mock sqlmock.Sqlmock, counts protectedCounts) {
	mock.ExpectQuery(regexp.QuoteMeta(protectedCountsSQL)).WillReturnRows(
		sqlmock.NewRows([]string{
			"events",
			"entity_nodes",
			"chain_node_profiles",
			"index_profiles",
			"event_tag_defs",
			"event_tag_maps",
			"raw_documents",
		}).AddRow(
			counts.Events,
			counts.EntityNodes,
			counts.ChainNodeProfiles,
			counts.IndexProfiles,
			counts.EventTagDefs,
			counts.EventTagMaps,
			counts.RawDocuments,
		),
	)
}
