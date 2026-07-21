package dbmigration

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestResearchAnchorBranchSummariesMigrationContract(t *testing.T) {
	raw := readMigration(t, filepath.Join(migrationDirectory(), "000026_add_research_anchor_branch_summaries.sql"))
	up, down := migrationSections(t, raw)
	up = strings.ToLower(up)
	down = strings.ToLower(down)

	for _, fragment := range []string{
		"research_anchor_import_receipts",
		"research_anchors",
		"research_anchor_events",
		"research_anchor_chain_nodes",
		"raise exception",
		"add column support_summary text",
		"add column counter_summary text",
		"set not null",
		"chk_research_anchors_support_summary_nonblank",
		"chk_research_anchors_counter_summary_nonblank",
	} {
		if !strings.Contains(up, fragment) {
			t.Errorf("branch summary migration Up must contain %q", fragment)
		}
	}
	if !strings.Contains(down, "migration 000026 is forward-only") || !strings.Contains(down, "raise exception") {
		t.Fatal("branch summary migration Down must fail closed as forward-only")
	}
}

func TestPostgresResearchAnchorBranchSummariesMigrationRequiresEmptyPublicationTables(t *testing.T) {
	for _, nonemptyTable := range []string{
		"research_anchor_import_receipts",
		"research_anchors",
		"research_anchor_events",
		"research_anchor_chain_nodes",
	} {
		t.Run(nonemptyTable, func(t *testing.T) {
			db := openIsolatedMigrationDatabase(t)
			prepareResearchAnchorBranchSummarySchema(t, db)
			if _, err := db.Exec(`INSERT INTO ` + nonemptyTable + ` (id) VALUES (1)`); err != nil {
				t.Fatal(err)
			}

			err := runResearchAnchorBranchSummaryMigration(db)
			if err == nil || !strings.Contains(err.Error(), "requires empty Research Anchor publication tables") {
				t.Fatalf("migration error = %v, want fail-closed nonempty-table error", err)
			}
			var supportColumn sql.NullString
			if err := db.QueryRow(`SELECT column_name FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'research_anchors' AND column_name = 'support_summary'`).Scan(&supportColumn); err != sql.ErrNoRows {
				t.Fatalf("support_summary column query error = %v, want sql.ErrNoRows after rollback", err)
			}
		})
	}
}

func TestPostgresResearchAnchorBranchSummariesMigrationAddsConstrainedColumns(t *testing.T) {
	db := openIsolatedMigrationDatabase(t)
	prepareResearchAnchorBranchSummarySchema(t, db)
	if err := runResearchAnchorBranchSummaryMigration(db); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`INSERT INTO research_anchors (id, support_summary, counter_summary) VALUES (1, '当前支持', NULL)`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO research_anchors (id, support_summary, counter_summary) VALUES (2, NULL, NULL)`,
		`INSERT INTO research_anchors (id, support_summary, counter_summary) VALUES (3, '   ', NULL)`,
		`INSERT INTO research_anchors (id, support_summary, counter_summary) VALUES (4, '当前支持', '   ')`,
	} {
		expectPostgresStatementFailure(t, db, statement)
	}
}

func prepareResearchAnchorBranchSummarySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE research_anchor_import_receipts (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE research_anchors (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE research_anchor_events (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE research_anchor_chain_nodes (id INTEGER PRIMARY KEY)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := goose.EnsureDBVersionContext(context.Background(), db); err != nil {
		t.Fatal(err)
	}
}

func runResearchAnchorBranchSummaryMigration(db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	migrations, err := goose.CollectMigrations(migrationDirectory(), 25, 26)
	if err != nil {
		return err
	}
	if len(migrations) != 1 {
		return fmt.Errorf("research anchor branch summary migrations = %d, want 1", len(migrations))
	}
	return migrations[0].UpContext(context.Background(), db)
}
