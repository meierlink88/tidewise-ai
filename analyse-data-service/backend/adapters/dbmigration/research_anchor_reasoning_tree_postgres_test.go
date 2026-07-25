package dbmigration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestPostgresResearchAnchorReasoningTreeMigrationAcceptsFrozenSchema(t *testing.T) {
	db := openIsolatedMigrationDatabase(t)
	prepareLegacyResearchAnchorSchema(t, db)
	applyResearchAnchorMigration(t, db)

	var legacyIndexTable sql.NullString
	if err := db.QueryRow(`SELECT to_regclass('research_anchor_indices')::text`).Scan(&legacyIndexTable); err != nil {
		t.Fatal(err)
	}
	if legacyIndexTable.Valid {
		t.Fatalf("research_anchor_indices still exists as %q", legacyIndexTable.String)
	}

	const (
		themeID        = "00000000-0000-4000-8000-000000000001"
		otherThemeID   = "00000000-0000-4000-8000-000000000002"
		receiptID      = "00000000-0000-4000-8000-000000000003"
		otherReceiptID = "00000000-0000-4000-8000-000000000004"
		anchorID       = "00000000-0000-4000-8000-000000000005"
		otherAnchorID  = "00000000-0000-4000-8000-000000000006"
		centerID       = "00000000-0000-4000-8000-000000000007"
		otherCenterID  = "00000000-0000-4000-8000-000000000008"
		downstreamID   = "00000000-0000-4000-8000-000000000009"
		eventID        = "00000000-0000-4000-8000-000000000010"
		otherEventID   = "00000000-0000-4000-8000-000000000011"
		unknownID      = "00000000-0000-4000-8000-000000000012"
		candidateID    = "00000000-0000-4000-8000-000000000013"
		invalidReceipt = "00000000-0000-4000-8000-000000000014"
		thirdCenterID  = "00000000-0000-4000-8000-000000000015"
		mappingThemeID = "00000000-0000-4000-8000-000000000016"
	)

	for _, statement := range []string{
		`INSERT INTO research_themes (id) VALUES ('` + themeID + `'), ('` + otherThemeID + `'), ('` + mappingThemeID + `')`,
		`INSERT INTO chain_node_profiles (entity_id) VALUES ('` + centerID + `'), ('` + otherCenterID + `'), ('` + downstreamID + `'), ('` + thirdCenterID + `')`,
		`INSERT INTO events (id) VALUES ('` + eventID + `'), ('` + otherEventID + `')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	mapping := fmt.Sprintf(`{"%s":"%s","%s":"%s"}`, centerID, anchorID, otherCenterID, otherAnchorID)
	counts := `{"anchors":2,"event_associations":2,"path_nodes":4,"receipts":1}`
	if _, err := db.Exec(`
INSERT INTO research_anchor_import_receipts (
    id, theme_id, publisher_subject, payload_hash,
    anchor_ids_by_center_chain_node_id, write_counts, published_at, imported_at
) VALUES ($1, $2, 'analyst-service', $3, $4::jsonb, $5::jsonb, now(), now())`,
		receiptID, themeID, strings.Repeat("a", 64), mapping, counts,
	); err != nil {
		t.Fatal(err)
	}
	otherMapping := fmt.Sprintf(`{"%s":"%s"}`, centerID, candidateID)
	if _, err := db.Exec(`
INSERT INTO research_anchor_import_receipts (
    id, theme_id, publisher_subject, payload_hash,
    anchor_ids_by_center_chain_node_id, write_counts, published_at, imported_at
) VALUES ($1, $2, 'analyst-service', $3, $4::jsonb,
          '{"anchors":1,"event_associations":1,"path_nodes":2,"receipts":1}'::jsonb,
          now(), now())`, otherReceiptID, otherThemeID, strings.Repeat("b", 64), otherMapping); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchors (
    id, theme_id, center_chain_node_entity_id, import_receipt_id,
    one_line_conclusion, fact_summary, net_direction_summary, trading_direction, next_checkpoint
) VALUES
    ($1, $3, $4, $6, '结论一', '事实一', '净方向一', '交易指向一', '下一检查点一'),
    ($2, $3, $5, $6, '结论二', '事实二', '净方向二', '交易指向二', '下一检查点二')`,
		anchorID, otherAnchorID, themeID, centerID, otherCenterID, receiptID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES
    ($1, 1, $2, 'increase', '起点变化', '起点影响', NULL),
    ($1, 2, $3, 'increase', '下游变化', '下游影响', '需求向下游传导')`,
		anchorID, centerID, downstreamID,
	); err != nil {
		t.Fatal(err)
	}

	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 1, $2, 'increase', '起点变化', '起点影响', '首节点错误机制')`, otherAnchorID, otherCenterID)
	if _, err := db.Exec(`
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 1, $2, 'increase', '起点变化', '起点影响', NULL)`, otherAnchorID, otherCenterID); err != nil {
		t.Fatal(err)
	}
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 2, $2, 'increase', '下游变化', '下游影响', NULL)`, otherAnchorID, downstreamID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 2, $2, 'increase', '下游变化', '下游影响', '   ')`, otherAnchorID, downstreamID)
	if _, err := db.Exec(`
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 2, $2, 'increase', '下游变化', '下游影响', '需求向下游传导')`, otherAnchorID, downstreamID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`
INSERT INTO research_anchor_events (anchor_id, event_id, evidence_role, evidence_summary)
VALUES ($1, $2, 'driver', '事件直接支持中心节点判断')`, anchorID, eventID); err != nil {
		t.Fatal(err)
	}
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_events (anchor_id, event_id, evidence_role, evidence_summary)
VALUES ($1, $2, 'invalid', '证据说明')`, otherAnchorID, otherEventID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_events (anchor_id, event_id, evidence_role, evidence_summary)
VALUES ($1, $2, 'driver', '   ')`, otherAnchorID, otherEventID)
	if _, err := db.Exec(`
INSERT INTO research_anchor_events (anchor_id, event_id, evidence_role, evidence_summary)
VALUES ($1, $2, 'driver', '事件直接支持另一中心节点判断')`, otherAnchorID, otherEventID); err != nil {
		t.Fatal(err)
	}

	expectPostgresStatementFailureContaining(t, db, "immutable",
		`UPDATE research_anchor_import_receipts SET publisher_subject = 'other' WHERE id = $1`, receiptID)
	expectPostgresStatementFailureContaining(t, db, "immutable",
		`DELETE FROM research_anchor_import_receipts WHERE id = $1`, receiptID)
	expectPostgresStatementFailureContaining(t, db, "immutable",
		`TRUNCATE research_anchor_import_receipts CASCADE`)
	expectPostgresStatementFailure(t, db,
		`DELETE FROM research_themes WHERE id = $1`, themeID)
	expectPostgresStatementFailureContaining(t, db, "chk_research_anchor_import_receipts_anchor_ids", `
INSERT INTO research_anchor_import_receipts (
    id, theme_id, publisher_subject, payload_hash,
    anchor_ids_by_center_chain_node_id, write_counts, published_at, imported_at
) VALUES ($1, $2, 'analyst-service', $3, '{"node":"anchor"}'::jsonb,
          '{"anchors":1,"event_associations":1,"path_nodes":2,"receipts":1}'::jsonb,
          now(), now())`, invalidReceipt, mappingThemeID, strings.Repeat("c", 64))
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchors (
    id, theme_id, center_chain_node_entity_id, import_receipt_id,
    one_line_conclusion, fact_summary, net_direction_summary, trading_direction, next_checkpoint
) VALUES ($1, $2, $3, $4, '结论', '事实', '净方向', '交易指向', '下一检查点')`,
		candidateID, otherThemeID, unknownID, otherReceiptID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchors (
    id, theme_id, center_chain_node_entity_id, import_receipt_id,
    one_line_conclusion, fact_summary, net_direction_summary, trading_direction, next_checkpoint
) VALUES ($1, $2, $3, $4, '结论', '事实', '净方向', '交易指向', '下一检查点')`,
		candidateID, themeID, centerID, receiptID)
	expectPostgresStatementFailureContaining(t, db, "fk_research_anchors_receipt_theme", `
INSERT INTO research_anchors (
    id, theme_id, center_chain_node_entity_id, import_receipt_id,
    one_line_conclusion, fact_summary, net_direction_summary, trading_direction, next_checkpoint
) VALUES ($1, $2, $3, $4, '结论', '事实', '净方向', '交易指向', '下一检查点')`,
		candidateID, themeID, thirdCenterID, otherReceiptID)

	blankAnchorFields := []struct {
		name       string
		conclusion string
		fact       string
		net        string
		trading    string
		checkpoint string
	}{
		{name: "conclusion", conclusion: "   ", fact: "事实", net: "净方向", trading: "交易指向", checkpoint: "下一检查点"},
		{name: "fact", conclusion: "结论", fact: "   ", net: "净方向", trading: "交易指向", checkpoint: "下一检查点"},
		{name: "net direction", conclusion: "结论", fact: "事实", net: "   ", trading: "交易指向", checkpoint: "下一检查点"},
		{name: "trading direction", conclusion: "结论", fact: "事实", net: "净方向", trading: "   ", checkpoint: "下一检查点"},
		{name: "checkpoint", conclusion: "结论", fact: "事实", net: "净方向", trading: "交易指向", checkpoint: "   "},
	}
	for index, testCase := range blankAnchorFields {
		t.Run("rejects blank anchor "+testCase.name, func(t *testing.T) {
			expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchors (
    id, theme_id, center_chain_node_entity_id, import_receipt_id,
    one_line_conclusion, fact_summary, net_direction_summary, trading_direction, next_checkpoint
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				fmt.Sprintf("00000000-0000-4000-8000-%012d", 100+index), otherThemeID, centerID, otherReceiptID,
				testCase.conclusion, testCase.fact, testCase.net, testCase.trading, testCase.checkpoint)
		})
	}

	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 3, $2, 'invalid', '变化', '影响', '机制')`, anchorID, otherCenterID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 3, $2, 'increase', '   ', '影响', '机制')`, anchorID, otherCenterID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 3, $2, 'increase', '变化', '   ', '机制')`, anchorID, otherCenterID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 3, $2, 'increase', '变化', '影响', '机制')`, anchorID, unknownID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 1, $2, 'increase', '变化', '影响', NULL)`, anchorID, otherCenterID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_chain_nodes (
    anchor_id, position, chain_node_entity_id, change_direction,
    change_summary, impact_summary, incoming_transmission_mechanism
) VALUES ($1, 3, $2, 'increase', '变化', '影响', '机制')`, anchorID, downstreamID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_events (anchor_id, event_id, evidence_role, evidence_summary)
VALUES ($1, $2, 'driver', '证据说明')`, anchorID, unknownID)
	expectPostgresStatementFailure(t, db, `
INSERT INTO research_anchor_events (anchor_id, event_id, evidence_role, evidence_summary)
VALUES ($1, $2, 'supporting', '重复证据说明')`, anchorID, eventID)

	var receiptCount int
	if err := db.QueryRow(`SELECT count(*) FROM research_anchor_import_receipts`).Scan(&receiptCount); err != nil {
		t.Fatal(err)
	}
	if receiptCount != 2 {
		t.Fatalf("receipt rows after rejected mutations = %d, want 2", receiptCount)
	}
}

func TestPostgresResearchAnchorReasoningTreeMigrationRejectsNonemptyLegacyTables(t *testing.T) {
	for _, table := range []string{
		"research_anchors",
		"research_anchor_chain_nodes",
		"research_anchor_indices",
		"research_anchor_events",
	} {
		t.Run(table, func(t *testing.T) {
			db := openIsolatedMigrationDatabase(t)
			prepareLegacyResearchAnchorSchema(t, db)

			const sentinelID = "00000000-0000-4000-8000-000000000021"
			if _, err := db.Exec(`INSERT INTO research_anchors (id) VALUES ($1)`, sentinelID); err != nil {
				t.Fatal(err)
			}
			if table != "research_anchors" {
				if _, err := db.Exec(`INSERT INTO `+table+` (anchor_id) VALUES ($1)`, sentinelID); err != nil {
					t.Fatal(err)
				}
			}

			err := runResearchAnchorMigration(db)
			if err == nil || !strings.Contains(err.Error(), "requires all legacy research anchor tables to be empty") {
				t.Fatalf("migration error = %v, want fail-closed legacy data error", err)
			}

			var count int
			if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Fatalf("legacy %s rows = %d, want 1 after rollback", table, count)
			}
			var receiptTable sql.NullString
			if err := db.QueryRow(`SELECT to_regclass('research_anchor_import_receipts')::text`).Scan(&receiptTable); err != nil {
				t.Fatal(err)
			}
			if receiptTable.Valid {
				t.Fatalf("new receipt table exists after failed migration as %q", receiptTable.String)
			}
		})
	}
}

func openIsolatedMigrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run Research Anchor migration integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("tw_anchor_migration_%d", time.Now().UnixNano())
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

func prepareLegacyResearchAnchorSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE research_themes (id UUID PRIMARY KEY)`,
		`CREATE TABLE chain_node_profiles (entity_id UUID PRIMARY KEY)`,
		`CREATE TABLE events (id UUID PRIMARY KEY)`,
		`CREATE TABLE research_anchors (id UUID PRIMARY KEY)`,
		`CREATE TABLE research_anchor_chain_nodes (anchor_id UUID NOT NULL REFERENCES research_anchors(id))`,
		`CREATE TABLE research_anchor_indices (anchor_id UUID NOT NULL REFERENCES research_anchors(id))`,
		`CREATE TABLE research_anchor_events (anchor_id UUID NOT NULL REFERENCES research_anchors(id))`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := goose.EnsureDBVersionContext(context.Background(), db); err != nil {
		t.Fatal(err)
	}
}

func applyResearchAnchorMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := runResearchAnchorMigration(db); err != nil {
		t.Fatal(err)
	}
}

func runResearchAnchorMigration(db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	migrations, err := goose.CollectMigrations(migrationDirectory(), 24, 25)
	if err != nil {
		return err
	}
	if len(migrations) != 1 {
		return fmt.Errorf("research anchor migrations = %d, want 1", len(migrations))
	}
	return migrations[0].UpContext(context.Background(), db)
}

func expectPostgresStatementFailure(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err == nil {
		t.Fatalf("statement unexpectedly succeeded: %s", statement)
	}
}

func expectPostgresStatementFailureContaining(t *testing.T, db *sql.DB, fragment, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err == nil {
		t.Fatalf("statement unexpectedly succeeded: %s", statement)
	} else if !strings.Contains(err.Error(), fragment) {
		t.Fatalf("statement error = %v, want %q", err, fragment)
	}
}
