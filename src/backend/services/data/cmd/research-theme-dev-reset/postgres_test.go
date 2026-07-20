package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestPostgresRunResetClearsResearchPublicationsAndPreservesMasterData(t *testing.T) {
	db := openIsolatedResetDatabase(t)
	prepareResetSchema(t, db)

	dryRun, err := runReset(context.Background(), db, resetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.Executed || dryRun.Mode != "dry-run" || dryRun.Before.isZero() || dryRun.After != dryRun.Before {
		t.Fatalf("dry-run report = %#v", dryRun)
	}
	if dryRun.ProtectedBefore != dryRun.ProtectedAfter {
		t.Fatalf("dry-run protected counts changed: %#v", dryRun)
	}

	report, err := runReset(context.Background(), db, resetOptions{
		Execute:         true,
		ConfirmDatabase: localDatabaseName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Executed || report.Mode != "execute" || !report.After.isZero() {
		t.Fatalf("reset report = %#v", report)
	}
	if report.ProtectedBefore != report.ProtectedAfter {
		t.Fatalf("protected counts changed: %#v", report)
	}
	if !report.TriggerRestored {
		t.Fatal("reset did not report restored immutable receipt triggers")
	}

	for _, query := range []string{themeReceiptTriggerStateSQL, anchorReceiptTriggerStateSQL} {
		var state string
		if err := db.QueryRow(query).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if state != "O" {
			t.Fatalf("receipt trigger state = %q, want O", state)
		}
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

func prepareResetSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE research_theme_import_receipts (id UUID PRIMARY KEY)`,
		`CREATE TABLE research_themes (
			id UUID PRIMARY KEY,
			import_receipt_id UUID REFERENCES research_theme_import_receipts(id)
		)`,
		`CREATE TABLE events (id UUID PRIMARY KEY)`,
		`CREATE TABLE entity_nodes (id UUID PRIMARY KEY)`,
		`CREATE TABLE chain_node_profiles (entity_id UUID PRIMARY KEY)`,
		`CREATE TABLE index_profiles (entity_id UUID PRIMARY KEY)`,
		`CREATE TABLE research_theme_chain_nodes (
			theme_id UUID REFERENCES research_themes(id) ON DELETE CASCADE,
			chain_node_entity_id UUID REFERENCES chain_node_profiles(entity_id)
		)`,
		`CREATE TABLE research_theme_indices (
			theme_id UUID REFERENCES research_themes(id) ON DELETE CASCADE,
			index_entity_id UUID REFERENCES index_profiles(entity_id)
		)`,
		`CREATE TABLE research_theme_events (
			theme_id UUID REFERENCES research_themes(id) ON DELETE CASCADE,
			event_id UUID REFERENCES events(id)
		)`,
		`CREATE TABLE research_anchors (id UUID PRIMARY KEY)`,
		`CREATE TABLE research_anchor_chain_nodes (anchor_id UUID NOT NULL REFERENCES research_anchors(id))`,
		`CREATE TABLE research_anchor_indices (anchor_id UUID NOT NULL REFERENCES research_anchors(id))`,
		`CREATE TABLE research_anchor_events (anchor_id UUID NOT NULL REFERENCES research_anchors(id))`,
		`CREATE TABLE event_tag_defs (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE event_tag_maps (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE raw_documents (id INTEGER PRIMARY KEY)`,
		`CREATE FUNCTION prevent_test_theme_receipt_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'immutable'; END; $$`,
		`CREATE TRIGGER trg_research_theme_import_receipts_immutable
		BEFORE UPDATE OR DELETE OR TRUNCATE ON research_theme_import_receipts
		FOR EACH STATEMENT EXECUTE FUNCTION prevent_test_theme_receipt_mutation()`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("execute %q: %v", statement, err)
		}
	}

	if _, err := goose.EnsureDBVersionContext(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	migrations, err := goose.CollectMigrations(filepath.Join("..", "..", "..", "..", "migrations"), 24, 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 {
		t.Fatalf("Research Anchor migrations = %d, want 1", len(migrations))
	}
	if err := migrations[0].UpContext(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	const (
		themeReceiptID = "00000000-0000-4000-8000-000000000101"
		themeID        = "00000000-0000-4000-8000-000000000102"
		anchorReceipt  = "00000000-0000-4000-8000-000000000103"
		anchorID       = "00000000-0000-4000-8000-000000000104"
		centerID       = "00000000-0000-4000-8000-000000000105"
		downstreamID   = "00000000-0000-4000-8000-000000000106"
		eventID        = "00000000-0000-4000-8000-000000000107"
		indexID        = "00000000-0000-4000-8000-000000000108"
		entityID       = "00000000-0000-4000-8000-000000000109"
	)
	mapping := fmt.Sprintf(`{"%s":"%s"}`, centerID, anchorID)
	for _, statement := range []string{
		`INSERT INTO events VALUES ('` + eventID + `')`,
		`INSERT INTO entity_nodes VALUES ('` + entityID + `')`,
		`INSERT INTO chain_node_profiles VALUES ('` + centerID + `'), ('` + downstreamID + `')`,
		`INSERT INTO index_profiles VALUES ('` + indexID + `')`,
		`INSERT INTO event_tag_defs VALUES (1)`,
		`INSERT INTO event_tag_maps VALUES (1)`,
		`INSERT INTO raw_documents VALUES (1)`,
		`INSERT INTO research_theme_import_receipts VALUES ('` + themeReceiptID + `')`,
		`INSERT INTO research_themes VALUES ('` + themeID + `', '` + themeReceiptID + `')`,
		`INSERT INTO research_theme_chain_nodes VALUES ('` + themeID + `', '` + centerID + `')`,
		`INSERT INTO research_theme_indices VALUES ('` + themeID + `', '` + indexID + `')`,
		`INSERT INTO research_theme_events VALUES ('` + themeID + `', '` + eventID + `')`,
		`INSERT INTO research_anchor_import_receipts (
			id, theme_id, publisher_subject, payload_hash,
			anchor_ids_by_center_chain_node_id, write_counts, published_at, imported_at
		) VALUES ('` + anchorReceipt + `', '` + themeID + `', 'analyst-service', '` + strings.Repeat("a", 64) + `',
			'` + mapping + `'::jsonb,
			'{"anchors":1,"event_associations":1,"path_nodes":2,"receipts":1}'::jsonb,
			now(), now())`,
		`INSERT INTO research_anchors (
			id, theme_id, center_chain_node_entity_id, import_receipt_id,
			one_line_conclusion, fact_summary, net_direction_summary, trading_direction, next_checkpoint
		) VALUES ('` + anchorID + `', '` + themeID + `', '` + centerID + `', '` + anchorReceipt + `',
			'结论', '事实', '净方向', '交易指向', '下一检查点')`,
		`INSERT INTO research_anchor_chain_nodes (
			anchor_id, position, chain_node_entity_id, change_direction,
			change_summary, impact_summary, incoming_transmission_mechanism
		) VALUES
			('` + anchorID + `', 1, '` + centerID + `', 'increase', '起点变化', '起点影响', NULL),
			('` + anchorID + `', 2, '` + downstreamID + `', 'increase', '下游变化', '下游影响', '需求传导')`,
		`INSERT INTO research_anchor_events (anchor_id, event_id, evidence_role, evidence_summary)
		VALUES ('` + anchorID + `', '` + eventID + `', 'driver', '事件支持中心节点判断')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("execute fixture %q: %v", statement, err)
		}
	}
}
