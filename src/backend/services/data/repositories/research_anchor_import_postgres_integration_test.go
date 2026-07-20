package repositories_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchanchorimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
	app "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchanchorimport"
)

func TestResearchAnchorImportPostgresIntegration(t *testing.T) {
	db := openResearchAnchorImportDatabase(t)
	prepareResearchAnchorImportSchema(t, db)
	publication := readResearchAnchorPublication(t)
	seedResearchAnchorPrerequisites(t, db, publication)

	service := app.NewService(repositories.NewPostgresRepository(db))
	results := importResearchAnchorsConcurrently(t, service, publication)
	if results[0].Replayed == results[1].Replayed {
		t.Fatalf("concurrent replay flags = %v/%v, want one initial and one replay", results[0].Replayed, results[1].Replayed)
	}
	first, second := results[0], results[1]
	first.Replayed, second.Replayed = false, false
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("concurrent results differ: %#v / %#v", first, second)
	}
	assertResearchAnchorImportCounts(t, db, first)

	changed := publication
	changed.Anchors = append([]domainimport.Anchor(nil), publication.Anchors...)
	changed.Anchors[0].FactSummary = "changed payload"
	if _, err := service.Import(context.Background(), "service:ai-research-analyst", changed); !errors.Is(err, app.ErrPayloadConflict) {
		t.Fatalf("different payload error = %v, want ErrPayloadConflict", err)
	}
	assertResearchAnchorImportCounts(t, db, first)
}

func openResearchAnchorImportDatabase(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run Research Anchor import PostgreSQL integration tests")
	}
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	var databaseName string
	if err := admin.QueryRow(`SELECT current_database()`).Scan(&databaseName); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	if databaseName != "tidewise_local" {
		admin.Close()
		t.Fatalf("integration database = %q, want tidewise_local", databaseName)
	}
	schema := fmt.Sprintf("tw_anchor_import_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
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
	if err := db.Ping(); err != nil {
		db.Close()
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		_, _ = admin.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`)
		admin.Close()
	})
	return db
}

func prepareResearchAnchorImportSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE research_theme_import_receipts (id UUID PRIMARY KEY, publisher_subject TEXT NOT NULL)`,
		`CREATE TABLE research_themes (id UUID PRIMARY KEY, import_receipt_id UUID REFERENCES research_theme_import_receipts(id))`,
		`CREATE TABLE chain_node_profiles (entity_id UUID PRIMARY KEY)`,
		`CREATE TABLE events (id UUID PRIMARY KEY)`,
		`CREATE TABLE research_theme_chain_nodes (theme_id UUID NOT NULL REFERENCES research_themes(id), chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id), PRIMARY KEY(theme_id, chain_node_entity_id))`,
		`CREATE TABLE research_theme_events (theme_id UUID NOT NULL REFERENCES research_themes(id), event_id UUID NOT NULL REFERENCES events(id), PRIMARY KEY(theme_id, event_id))`,
		`CREATE TABLE research_anchor_import_receipts (
            id UUID PRIMARY KEY,
            theme_id UUID NOT NULL UNIQUE REFERENCES research_themes(id),
            publisher_subject TEXT NOT NULL,
            payload_hash CHAR(64) NOT NULL,
            anchor_ids_by_center_chain_node_id JSONB NOT NULL,
            write_counts JSONB NOT NULL,
            published_at TIMESTAMPTZ NOT NULL,
            imported_at TIMESTAMPTZ NOT NULL,
            UNIQUE(id, theme_id)
        )`,
		`CREATE TABLE research_anchors (
            id UUID PRIMARY KEY,
            theme_id UUID NOT NULL REFERENCES research_themes(id) ON DELETE CASCADE,
            center_chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
            import_receipt_id UUID NOT NULL,
            one_line_conclusion TEXT NOT NULL,
            fact_summary TEXT NOT NULL,
            net_direction_summary TEXT NOT NULL,
            trading_direction TEXT NOT NULL,
            next_checkpoint TEXT NOT NULL,
            UNIQUE(theme_id, center_chain_node_entity_id),
            FOREIGN KEY(import_receipt_id, theme_id) REFERENCES research_anchor_import_receipts(id, theme_id)
        )`,
		`CREATE TABLE research_anchor_events (
            anchor_id UUID NOT NULL REFERENCES research_anchors(id) ON DELETE CASCADE,
            event_id UUID NOT NULL REFERENCES events(id),
            evidence_role TEXT NOT NULL,
            evidence_summary TEXT NOT NULL,
            PRIMARY KEY(anchor_id, event_id)
        )`,
		`CREATE TABLE research_anchor_chain_nodes (
            anchor_id UUID NOT NULL REFERENCES research_anchors(id) ON DELETE CASCADE,
            position INTEGER NOT NULL,
            chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
            change_direction TEXT NOT NULL,
            change_summary TEXT NOT NULL,
            impact_summary TEXT NOT NULL,
            incoming_transmission_mechanism TEXT,
            PRIMARY KEY(anchor_id, position),
            UNIQUE(anchor_id, chain_node_entity_id)
        )`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
}

func readResearchAnchorPublication(t *testing.T) domainimport.Publication {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "testdata", "reasoning-tree-v1", "01-multi-anchor-import-request.json")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	publication, err := domainimport.DecodeStrict(file)
	if err != nil {
		t.Fatal(err)
	}
	return publication
}

func seedResearchAnchorPrerequisites(t *testing.T, db *sql.DB, publication domainimport.Publication) {
	t.Helper()
	const themeReceiptID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	if _, err := db.Exec(`INSERT INTO research_theme_import_receipts (id, publisher_subject) VALUES ($1, $2)`, themeReceiptID, "service:ai-research-analyst"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO research_themes (id, import_receipt_id) VALUES ($1, $2)`, publication.ThemeID, themeReceiptID); err != nil {
		t.Fatal(err)
	}
	nodes := map[string]struct{}{}
	events := map[string]struct{}{}
	centers := map[string]struct{}{}
	for _, anchor := range publication.Anchors {
		centers[anchor.CenterChainNodeID] = struct{}{}
		for _, event := range anchor.Events {
			events[event.EventID] = struct{}{}
		}
		for _, node := range anchor.PathNodes {
			nodes[node.ChainNodeID] = struct{}{}
		}
	}
	for nodeID := range nodes {
		if _, err := db.Exec(`INSERT INTO chain_node_profiles (entity_id) VALUES ($1)`, nodeID); err != nil {
			t.Fatal(err)
		}
	}
	for centerID := range centers {
		if _, err := db.Exec(`INSERT INTO research_theme_chain_nodes (theme_id, chain_node_entity_id) VALUES ($1, $2)`, publication.ThemeID, centerID); err != nil {
			t.Fatal(err)
		}
	}
	for eventID := range events {
		if _, err := db.Exec(`INSERT INTO events (id) VALUES ($1)`, eventID); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO research_theme_events (theme_id, event_id) VALUES ($1, $2)`, publication.ThemeID, eventID); err != nil {
			t.Fatal(err)
		}
	}
}

func importResearchAnchorsConcurrently(t *testing.T, service *app.Service, publication domainimport.Publication) []app.Result {
	t.Helper()
	ready := make(chan struct{}, 2)
	start := make(chan struct{})
	results := make(chan app.Result, 2)
	errs := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ready <- struct{}{}
			<-start
			result, err := service.Import(context.Background(), "service:ai-research-analyst", publication)
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}()
	}
	<-ready
	<-ready
	close(start)
	workers.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var imported []app.Result
	for result := range results {
		imported = append(imported, result)
	}
	if len(imported) != 2 {
		t.Fatalf("concurrent imports = %d successful results, want 2", len(imported))
	}
	return imported
}

func assertResearchAnchorImportCounts(t *testing.T, db *sql.DB, result app.Result) {
	t.Helper()
	checks := []struct {
		query string
		want  int
	}{
		{`SELECT count(*) FROM research_anchor_import_receipts WHERE id = $1`, 1},
		{`SELECT count(*) FROM research_anchors WHERE import_receipt_id = $1`, result.Counts.Anchors},
		{`SELECT count(*) FROM research_anchor_events e JOIN research_anchors a ON a.id = e.anchor_id WHERE a.import_receipt_id = $1`, result.Counts.EventAssociations},
		{`SELECT count(*) FROM research_anchor_chain_nodes n JOIN research_anchors a ON a.id = n.anchor_id WHERE a.import_receipt_id = $1`, result.Counts.PathNodes},
	}
	for _, check := range checks {
		var count int
		if err := db.QueryRow(check.query, result.ReceiptID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != check.want {
			t.Fatalf("count for %q = %d, want %d", check.query, count, check.want)
		}
	}
}
