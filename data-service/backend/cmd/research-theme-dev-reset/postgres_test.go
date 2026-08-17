package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresResetClearsOnlyV1ResearchPublications(t *testing.T) {
	db := openIsolatedResetDatabase(t)
	prepareV1ResetSchema(t, db)

	report, err := runReset(context.Background(), db, resetOptions{
		Execute: true, ConfirmDatabase: localDatabaseName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Executed || !report.After.isZero() {
		t.Fatalf("reset report = %#v", report)
	}
	if report.ProtectedBefore != report.ProtectedAfter {
		t.Fatalf("protected data changed: %#v", report)
	}
	var total, enabled int
	if err := db.QueryRow(immutableTriggerStateSQL).Scan(&total, &enabled); err != nil {
		t.Fatal(err)
	}
	if total != 9 || enabled != 9 {
		t.Fatalf("immutable triggers total=%d enabled=%d", total, enabled)
	}
}

func openIsolatedResetDatabase(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run reset integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	var databaseName string
	if err := admin.QueryRowContext(ctx, currentDatabaseSQL).Scan(&databaseName); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	if databaseName != localDatabaseName {
		admin.Close()
		t.Skipf("reset integration test requires database %s, got %s", localDatabaseName, databaseName)
	}
	schema := fmt.Sprintf("tw_research_reset_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*config)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
		admin.Close()
	})
	return db
}

func prepareV1ResetSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	publicationTables := []struct {
		table, trigger string
	}{
		{"research_theme_import_receipts", "trg_research_theme_receipts_immutable"},
		{"research_themes", "trg_research_themes_immutable"},
		{"research_theme_impacts", "trg_research_theme_impacts_immutable"},
		{"research_theme_events", "trg_research_theme_events_immutable"},
		{"research_reasoning_tree_import_receipts", "trg_research_reasoning_tree_receipts_immutable"},
		{"research_reasoning_trees", "trg_research_reasoning_trees_immutable"},
		{"research_reasoning_tree_events", "trg_research_reasoning_tree_events_immutable"},
		{"research_reasoning_tree_nodes", "trg_research_reasoning_tree_nodes_immutable"},
		{"research_reasoning_tree_node_signals", "trg_research_reasoning_tree_node_signals_immutable"},
	}
	protectedTables := []string{
		"events", "entity_nodes", "industry", "concept", "chain_node", "industry_chain",
		"industry_chain_graph_edges", "index_profiles", "event_tag_defs",
		"event_tag_maps", "raw_documents",
	}
	for _, table := range append(publicationTableNames(publicationTables), protectedTables...) {
		if _, err := db.Exec(`CREATE TABLE ` + table + ` (id INTEGER PRIMARY KEY)`); err != nil {
			t.Fatalf("create %s: %v", table, err)
		}
		if _, err := db.Exec(`INSERT INTO ` + table + ` VALUES (1)`); err != nil {
			t.Fatalf("seed %s: %v", table, err)
		}
	}
	if _, err := db.Exec(`CREATE FUNCTION prevent_test_research_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'immutable'; END;
$$`); err != nil {
		t.Fatal(err)
	}
	for _, publication := range publicationTables {
		statement := `CREATE TRIGGER ` + publication.trigger +
			` BEFORE UPDATE OR DELETE OR TRUNCATE ON ` + publication.table +
			` FOR EACH STATEMENT EXECUTE FUNCTION prevent_test_research_mutation()`
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create trigger %s: %v", publication.trigger, err)
		}
	}
}

func publicationTableNames(values []struct{ table, trigger string }) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.table)
	}
	return result
}
