package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestPostgresRepositoryGraphProjectionIntegration(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run PostgreSQL graph projection repository integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})

	ctx := context.Background()
	repo := NewPostgresRepository(db)
	runID := fmt.Sprintf("graph-integration-%d", time.Now().UnixNano())
	firstEntityID := NormalizeUUID(runID, "entity-a")
	secondEntityID := NormalizeUUID(runID, "entity-b")
	edgeID := NormalizeUUID(runID, "edge")
	projectionRunID := NormalizeUUID(runID + "-run")
	t.Cleanup(func() {
		for _, cleanup := range []struct {
			query string
			args  []any
		}{
			{`DELETE FROM graph_projection_run_items WHERE run_id = $1`, []any{projectionRunID}},
			{`DELETE FROM graph_projection_runs WHERE id = $1`, []any{projectionRunID}},
			{`DELETE FROM entity_edges WHERE id = $1`, []any{edgeID}},
			{`DELETE FROM entity_nodes WHERE id IN ($1, $2)`, []any{firstEntityID, secondEntityID}},
		} {
			if _, err := db.ExecContext(ctx, cleanup.query, cleanup.args...); err != nil {
				t.Errorf("clean graph projection integration data: %v", err)
			}
		}
	})

	if _, err := db.ExecContext(ctx, `
INSERT INTO entity_nodes (
    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
    ($1, $2, 'economy', 'economy', '中国', '中国', ARRAY['China', 'PRC']::text[], 'active'),
    ($3, $4, 'alliance_org', 'alliance', 'G20', '二十国集团', '{}'::text[], 'active')
ON CONFLICT (id) DO UPDATE SET
    entity_key = EXCLUDED.entity_key,
    updated_at = now()
`, firstEntityID, runID+"-economy:cn", secondEntityID, runID+"-alliance:g20"); err != nil {
		t.Fatalf("insert graph entities: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO entity_edges (
    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
) VALUES (
    $1, $2, $3, 'member_of', 'integration test', 'active'
) ON CONFLICT (id) DO UPDATE SET
    relation_type = EXCLUDED.relation_type,
    evidence_note = EXCLUDED.evidence_note,
    updated_at = now()
`, edgeID, firstEntityID, secondEntityID); err != nil {
		t.Fatalf("insert graph edge: %v", err)
	}

	nodes, err := repo.ListGraphEntityNodes(ctx)
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("ListGraphEntityNodes() returned no rows")
	}
	var projectedEntity *GraphEntityNode
	for i := range nodes {
		if nodes[i].ID == firstEntityID {
			projectedEntity = &nodes[i]
			break
		}
	}
	if projectedEntity == nil {
		t.Fatalf("ListGraphEntityNodes() missing inserted entity %s", firstEntityID)
	}
	if !reflect.DeepEqual(projectedEntity.Aliases, []string{"China", "PRC"}) {
		t.Fatalf("projected aliases = %v, want [China PRC]", projectedEntity.Aliases)
	}
	edges, err := repo.ListGraphEntityEdges(ctx)
	if err != nil {
		t.Fatalf("ListGraphEntityEdges() error = %v", err)
	}
	if len(edges) == 0 {
		t.Fatal("ListGraphEntityEdges() returned no rows")
	}

	started := time.Now()
	finished := started.Add(time.Second)
	run, err := repo.CreateGraphProjectionRun(ctx, GraphProjectionRun{
		ID:             runID + "-run",
		ProjectionType: GraphProjectionTypeEntityGraph,
		Mode:           GraphProjectionModeProjectEntities,
		Status:         GraphProjectionRunStatusRunning,
		StartedAt:      started,
		ConfigSummary:  map[string]any{"namespace": "tidewise"},
	})
	if err != nil {
		t.Fatalf("CreateGraphProjectionRun() error = %v", err)
	}
	if err := repo.RecordGraphProjectionRunItem(ctx, GraphProjectionRunItem{
		ID:       runID + "-item",
		RunID:    run.ID,
		ItemType: GraphProjectionRunItemTypeEntity,
		ItemKey:  firstEntityID,
		Status:   GraphProjectionRunItemStatusProjected,
	}); err != nil {
		t.Fatalf("RecordGraphProjectionRunItem() error = %v", err)
	}
	run.Status = GraphProjectionRunStatusSucceeded
	run.FinishedAt = &finished
	run.SourceRowCount = 2
	run.ProjectedCount = 2
	if err := repo.CompleteGraphProjectionRun(ctx, run); err != nil {
		t.Fatalf("CompleteGraphProjectionRun() error = %v", err)
	}
	runs, err := repo.RecentGraphProjectionRuns(ctx, 5)
	if err != nil {
		t.Fatalf("RecentGraphProjectionRuns() error = %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("RecentGraphProjectionRuns() returned no rows")
	}
}

func TestPostgresGraphProjectionSourcesOnlyContainActiveRows(t *testing.T) {
	dsn := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run PostgreSQL graph projection source integration test")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := NewPostgresRepository(db)
	nodes, err := repo.ListGraphEntityNodes(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	activeIDs := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if node.Status != domain.StatusActive {
			t.Fatalf("projection source contains inactive node %s with status %s", node.ID, node.Status)
		}
		if node.EntityType == domain.EntityTypeSector {
			if node.ClassificationCode != domain.SectorClassificationIndustry && node.ClassificationCode != domain.SectorClassificationTheme && node.ClassificationCode != domain.SectorClassificationMarket && node.ClassificationCode != domain.SectorClassificationStyle && node.ClassificationCode != domain.SectorClassificationRegion {
				t.Fatalf("projection sector %s has invalid classification %q", node.ID, node.ClassificationCode)
			}
		} else if node.ClassificationCode != "" {
			t.Fatalf("non-sector projection node %s has classification %q", node.ID, node.ClassificationCode)
		}
		activeIDs[node.ID] = struct{}{}
	}
	edges, err := repo.ListGraphEntityEdges(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityEdges() error = %v", err)
	}
	for _, edge := range edges {
		if edge.Status != domain.StatusActive {
			t.Fatalf("projection source contains inactive edge %s", edge.ID)
		}
		if _, ok := activeIDs[edge.FromEntityID]; !ok {
			t.Fatalf("projection edge %s has inactive or absent from endpoint %s", edge.ID, edge.FromEntityID)
		}
		if _, ok := activeIDs[edge.ToEntityID]; !ok {
			t.Fatalf("projection edge %s has inactive or absent to endpoint %s", edge.ID, edge.ToEntityID)
		}
	}
}
