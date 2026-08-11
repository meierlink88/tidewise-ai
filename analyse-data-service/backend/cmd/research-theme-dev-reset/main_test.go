package main

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
)

func TestValidateResetSafetyGates(t *testing.T) {
	valid := conf.Config{
		App:      conf.AppConfig{Env: conf.EnvLocal},
		Database: conf.DatabaseConfig{Host: "host.docker.internal", Name: localDatabaseName},
	}
	if err := validateResetTarget(valid); err != nil {
		t.Fatal(err)
	}
	valid.Database.Host = "db.internal"
	if err := validateResetTarget(valid); err == nil {
		t.Fatal("unapproved infrastructure target was accepted")
	}
	if err := validateExecutionGate(resetOptions{Execute: true}); err == nil {
		t.Fatal("execute without exact database confirmation was accepted")
	}
}

func TestRunResetDryRunOnlyReadsCounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	before := publicationCounts{ResearchThemes: 1, ResearchThemeImpacts: 3, ResearchReasoningTrees: 2}
	protected := protectedCounts{Events: 2, EntityNodes: 5, ChainNodeProfiles: 3}
	expectPreflight(mock, before, protected)
	mock.ExpectCommit()

	report, err := runReset(context.Background(), db, resetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Executed || report.After != before || report.ProtectedAfter != protected {
		t.Fatalf("dry-run report = %#v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunResetDeletesOnlyResearchPublicationsAndRestoresTriggers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	protected := protectedCounts{Events: 2, EntityNodes: 5, ChainNodeProfiles: 3, IndustryChainDefinitions: 1}
	expectPreflight(mock, publicationCounts{ResearchThemes: 1, ResearchReasoningTrees: 2}, protected)
	mock.ExpectExec(regexp.QuoteMeta(disablePublicationTriggersSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(deletePublicationsSQL)).WillReturnResult(sqlmock.NewResult(0, 9))
	mock.ExpectExec(regexp.QuoteMeta(enablePublicationTriggersSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	expectTriggersEnabled(mock)
	expectPublicationCounts(mock, publicationCounts{})
	expectProtectedCounts(mock, protected)
	mock.ExpectCommit()

	report, err := runReset(context.Background(), db, resetOptions{Execute: true, ConfirmDatabase: localDatabaseName})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Executed || !report.After.isZero() || report.ProtectedBefore != report.ProtectedAfter {
		t.Fatalf("execute report = %#v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunResetRollsBackWhenPublicationDeleteFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	expectPreflight(mock, publicationCounts{ResearchThemes: 1}, protectedCounts{})
	mock.ExpectExec(regexp.QuoteMeta(disablePublicationTriggersSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(deletePublicationsSQL)).WillReturnError(errors.New("delete failed"))
	mock.ExpectRollback()

	_, err = runReset(context.Background(), db, resetOptions{Execute: true, ConfirmDatabase: localDatabaseName})
	if err == nil || !strings.Contains(err.Error(), "delete Research Theme and Reason Tree publications") {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectPreflight(mock sqlmock.Sqlmock, publications publicationCounts, protected protectedCounts) {
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(currentDatabaseSQL)).WillReturnRows(
		sqlmock.NewRows([]string{"current_database"}).AddRow(localDatabaseName),
	)
	mock.ExpectQuery(regexp.QuoteMeta(acquireResetLockSQL)).WithArgs(resetLockKey).WillReturnRows(
		sqlmock.NewRows([]string{"locked"}).AddRow(true),
	)
	expectTriggersEnabled(mock)
	expectPublicationCounts(mock, publications)
	expectProtectedCounts(mock, protected)
}

func expectTriggersEnabled(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(immutableTriggerStateSQL)).WillReturnRows(
		sqlmock.NewRows([]string{"total", "enabled"}).AddRow(9, 9),
	)
}

func expectPublicationCounts(mock sqlmock.Sqlmock, counts publicationCounts) {
	mock.ExpectQuery(regexp.QuoteMeta(publicationCountsSQL)).WillReturnRows(
		sqlmock.NewRows([]string{
			"research_theme_import_receipts", "research_themes", "research_theme_impacts",
			"research_theme_events", "research_reasoning_tree_import_receipts",
			"research_reasoning_trees", "research_reasoning_tree_events",
			"research_reasoning_tree_nodes", "research_reasoning_tree_node_signals",
		}).AddRow(
			counts.ResearchThemeImportReceipts, counts.ResearchThemes,
			counts.ResearchThemeImpacts, counts.ResearchThemeEvents,
			counts.ResearchReasoningTreeImportReceipts, counts.ResearchReasoningTrees,
			counts.ResearchReasoningTreeEvents, counts.ResearchReasoningTreeNodes,
			counts.ResearchReasoningTreeNodeSignals,
		),
	)
}

func expectProtectedCounts(mock sqlmock.Sqlmock, counts protectedCounts) {
	mock.ExpectQuery(regexp.QuoteMeta(protectedCountsSQL)).WillReturnRows(
		sqlmock.NewRows([]string{
			"events", "entity_nodes", "chain_node_profiles", "industry_chain_definitions",
			"industry_chain_graph_edges", "index_profiles", "event_tag_defs",
			"event_tag_maps", "raw_documents",
		}).AddRow(
			counts.Events, counts.EntityNodes, counts.ChainNodeProfiles,
			counts.IndustryChainDefinitions, counts.IndustryChainGraphEdges,
			counts.IndexProfiles, counts.EventTagDefs, counts.EventTagMaps, counts.RawDocuments,
		),
	)
}
