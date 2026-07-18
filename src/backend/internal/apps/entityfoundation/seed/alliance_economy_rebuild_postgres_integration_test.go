package seed

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresCleanupStatementVisibilityRequiresSeparateRemainingQuery(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TIDEWISE_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, statement := range []string{
		`CREATE TEMP TABLE ae_cleanup_nodes(kind text NOT NULL) ON COMMIT DROP`,
		`CREATE TEMP TABLE ae_cleanup_profiles(kind text NOT NULL) ON COMMIT DROP`,
		`CREATE TEMP TABLE ae_cleanup_edges(kind text NOT NULL) ON COMMIT DROP`,
		`INSERT INTO ae_cleanup_nodes(kind) SELECT 'economy' FROM generate_series(1,50)`,
		`INSERT INTO ae_cleanup_nodes(kind) VALUES ('alliance')`,
		`INSERT INTO ae_cleanup_profiles(kind) SELECT 'economy' FROM generate_series(1,50)`,
		`INSERT INTO ae_cleanup_profiles(kind) VALUES ('alliance')`,
		`INSERT INTO ae_cleanup_edges(kind) VALUES ('member_of')`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	var deletedEdges, deletedProfiles, deletedAlliances, oldAlliances, oldProfiles, oldEdges, oldEconomies, oldEconomyProfiles int
	err = tx.QueryRowContext(ctx, `WITH deleted_edges AS (DELETE FROM ae_cleanup_edges WHERE kind='member_of' RETURNING 1), deleted_profiles AS (DELETE FROM ae_cleanup_profiles WHERE kind='alliance' RETURNING 1), deleted_alliances AS (DELETE FROM ae_cleanup_nodes WHERE kind='alliance' RETURNING 1) SELECT (SELECT count(*) FROM deleted_edges), (SELECT count(*) FROM deleted_profiles), (SELECT count(*) FROM deleted_alliances), (SELECT count(*) FROM ae_cleanup_nodes WHERE kind='alliance'), (SELECT count(*) FROM ae_cleanup_profiles WHERE kind='alliance'), (SELECT count(*) FROM ae_cleanup_edges WHERE kind='member_of'), (SELECT count(*) FROM ae_cleanup_nodes WHERE kind='economy'), (SELECT count(*) FROM ae_cleanup_profiles WHERE kind='economy')`).Scan(&deletedEdges, &deletedProfiles, &deletedAlliances, &oldAlliances, &oldProfiles, &oldEdges, &oldEconomies, &oldEconomyProfiles)
	if err != nil {
		t.Fatal(err)
	}
	if deletedEdges != 1 || deletedProfiles != 1 || deletedAlliances != 1 || oldAlliances != 1 || oldProfiles != 1 || oldEdges != 1 || oldEconomies != 50 || oldEconomyProfiles != 50 {
		t.Fatalf("single statement result = edges=%d profiles=%d alliances=%d remaining=%d/%d/%d economy=%d/%d", deletedEdges, deletedProfiles, deletedAlliances, oldAlliances, oldProfiles, oldEdges, oldEconomies, oldEconomyProfiles)
	}

	var alliances, profiles, edges, economies, economyProfiles int
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM ae_cleanup_nodes WHERE kind='alliance'), (SELECT count(*) FROM ae_cleanup_profiles WHERE kind='alliance'), (SELECT count(*) FROM ae_cleanup_edges WHERE kind='member_of'), (SELECT count(*) FROM ae_cleanup_nodes WHERE kind='economy'), (SELECT count(*) FROM ae_cleanup_profiles WHERE kind='economy')`).Scan(&alliances, &profiles, &edges, &economies, &economyProfiles); err != nil {
		t.Fatal(err)
	}
	if alliances != 0 || profiles != 0 || edges != 0 || economies != 50 || economyProfiles != 50 {
		t.Fatalf("next statement remaining = alliance=%d profile=%d member_of=%d economy=%d economy_profile=%d", alliances, profiles, edges, economies, economyProfiles)
	}
}
