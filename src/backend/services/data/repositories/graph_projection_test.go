package repositories

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestInMemoryRepositoryListsEntityGraphSnapshot(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Date(2026, 7, 10, 9, 30, 0, 0, time.UTC)

	repo.SeedGraphEntity(GraphEntityNode{
		ID:            "entity-1",
		EntityKey:     "economy:cn",
		EntityType:    domain.EntityTypeEconomy,
		LayerCode:     "economy",
		Name:          "中国",
		CanonicalName: "中国",
		Aliases:       []string{"China"},
		Status:        domain.StatusActive,
		UpdatedAt:     now,
	})
	repo.SeedGraphEntity(GraphEntityNode{
		ID:            "entity-2",
		EntityType:    domain.EntityTypeAllianceOrg,
		LayerCode:     "alliance",
		Name:          "G20",
		CanonicalName: "二十国集团",
		Status:        domain.StatusActive,
		UpdatedAt:     now.Add(time.Minute),
	})
	repo.SeedGraphEdge(GraphEntityEdge{
		ID:           "edge-1",
		FromEntityID: "entity-1",
		ToEntityID:   "entity-2",
		RelationType: "member_of",
		EvidenceNote: "基础关系",
		Status:       domain.StatusActive,
		UpdatedAt:    now.Add(2 * time.Minute),
	})

	nodes, err := repo.ListGraphEntityNodes(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	edges, err := repo.ListGraphEntityEdges(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityEdges() error = %v", err)
	}

	if got, want := graphEntityNodeIDs(nodes), []string{"entity-1", "entity-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("node ids = %v, want %v", got, want)
	}
	if nodes[1].EntityKey != "alliance_org:entity-2" {
		t.Fatalf("fallback entity key = %q, want alliance_org:entity-2", nodes[1].EntityKey)
	}
	if !reflect.DeepEqual(nodes[0].Aliases, []string{"China"}) {
		t.Fatalf("node aliases = %v, want [China]", nodes[0].Aliases)
	}
	if len(edges) != 1 || edges[0].RelationType != "member_of" {
		t.Fatalf("edges = %+v, want member_of edge", edges)
	}
}

func TestInMemoryRepositoryListsOnlyActiveGraphProjectionInputs(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	for _, node := range []GraphEntityNode{
		{ID: "active-a", EntityType: domain.EntityTypeEconomy, LayerCode: "economy", Name: "A", CanonicalName: "A", Status: domain.StatusActive, UpdatedAt: now},
		{ID: "active-b", EntityType: domain.EntityTypeMarket, LayerCode: "market", Name: "B", CanonicalName: "B", Status: domain.StatusActive, UpdatedAt: now},
		{ID: "inactive", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "旧板块", CanonicalName: "旧板块", Status: domain.StatusInactive, UpdatedAt: now},
	} {
		repo.SeedGraphEntity(node)
	}
	for _, edge := range []GraphEntityEdge{
		{ID: "active-edge", FromEntityID: "active-a", ToEntityID: "active-b", RelationType: "has_market", Status: domain.StatusActive, UpdatedAt: now},
		{ID: "inactive-endpoint", FromEntityID: "active-b", ToEntityID: "inactive", RelationType: "covers_sector", Status: domain.StatusActive, UpdatedAt: now},
		{ID: "inactive-edge", FromEntityID: "active-a", ToEntityID: "active-b", RelationType: "has_market", Status: domain.StatusInactive, UpdatedAt: now},
	} {
		repo.SeedGraphEdge(edge)
	}

	nodes, err := repo.ListGraphEntityNodes(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	edges, err := repo.ListGraphEntityEdges(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityEdges() error = %v", err)
	}
	if got, want := graphEntityNodeIDs(nodes), []string{"active-a", "active-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("node ids = %v, want %v", got, want)
	}
	if len(edges) != 1 || edges[0].ID != "active-edge" {
		t.Fatalf("edges = %+v, want only active-edge", edges)
	}
}

func TestInMemoryRepositoryPreservesSectorClassificationForProjection(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	repo.SeedGraphEntity(GraphEntityNode{
		ID: "sector", EntityKey: "sector:industry_test", EntityType: domain.EntityTypeSector,
		LayerCode: "sector", Name: "测试行业", CanonicalName: "测试行业",
		ClassificationCode: domain.SectorClassificationIndustry, Status: domain.StatusActive, UpdatedAt: now,
	})
	repo.SeedGraphEntity(GraphEntityNode{
		ID: "market", EntityKey: "market:test", EntityType: domain.EntityTypeMarket,
		LayerCode: "market", Name: "测试市场", CanonicalName: "测试市场", Status: domain.StatusActive, UpdatedAt: now,
	})

	nodes, err := repo.ListGraphEntityNodes(context.Background())
	if err != nil {
		t.Fatalf("ListGraphEntityNodes() error = %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes = %+v, want two active nodes", nodes)
	}
	if nodes[1].ClassificationCode != domain.SectorClassificationIndustry {
		t.Fatalf("sector classification = %q", nodes[1].ClassificationCode)
	}
	if nodes[0].ClassificationCode != "" {
		t.Fatalf("non-sector classification = %q, want empty", nodes[0].ClassificationCode)
	}
}

func TestInMemoryRepositoryRecordsGraphProjectionRuns(t *testing.T) {
	repo := NewInMemoryRepository()
	started := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	finished := started.Add(2 * time.Second)

	run, err := repo.CreateGraphProjectionRun(context.Background(), GraphProjectionRun{
		ID:             "run-1",
		ProjectionType: GraphProjectionTypeEntityGraph,
		Mode:           GraphProjectionModeProjectEntities,
		Status:         GraphProjectionRunStatusRunning,
		StartedAt:      started,
		ConfigSummary:  map[string]any{"namespace": "tidewise"},
	})
	if err != nil {
		t.Fatalf("CreateGraphProjectionRun() error = %v", err)
	}
	if run.Status != GraphProjectionRunStatusRunning {
		t.Fatalf("created run status = %q", run.Status)
	}

	if err := repo.RecordGraphProjectionRunItem(context.Background(), GraphProjectionRunItem{
		ID:           "item-1",
		RunID:        "run-1",
		ItemType:     GraphProjectionRunItemTypeRelationship,
		ItemKey:      "edge-1",
		Status:       GraphProjectionRunItemStatusFailed,
		ErrorMessage: "missing endpoint",
	}); err != nil {
		t.Fatalf("RecordGraphProjectionRunItem() error = %v", err)
	}

	run.Status = GraphProjectionRunStatusPartial
	run.FinishedAt = &finished
	run.SourceRowCount = 3
	run.ProjectedCount = 2
	run.SkippedCount = 0
	run.FailedCount = 1
	run.ErrorSummary = "1 relationship failed"
	if err := repo.CompleteGraphProjectionRun(context.Background(), run); err != nil {
		t.Fatalf("CompleteGraphProjectionRun() error = %v", err)
	}

	runs, err := repo.RecentGraphProjectionRuns(context.Background(), 5)
	if err != nil {
		t.Fatalf("RecentGraphProjectionRuns() error = %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("recent run count = %d, want 1", len(runs))
	}
	if runs[0].Status != GraphProjectionRunStatusPartial || runs[0].FailedCount != 1 {
		t.Fatalf("recent run = %+v, want partial with one failure", runs[0])
	}
}

func graphEntityNodeIDs(items []GraphEntityNode) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestPostgresGraphProjectionQueriesRequireActiveRowsAndEndpoints(t *testing.T) {
	nodes := strings.ToLower(graphEntityNodesQuery)
	for _, fragment := range []string{"from entity_nodes node", "where node.status = 'active'", "node.entity_type in ('alliance_org', 'economy', 'chain_node')"} {
		if !strings.Contains(nodes, fragment) {
			t.Fatalf("node projection query missing %q: %s", fragment, graphEntityNodesQuery)
		}
	}
	for _, forbidden := range []string{"sector_profiles", "industry_chain_profiles", "classification_code"} {
		if strings.Contains(nodes, forbidden) {
			t.Fatalf("node projection query contains retired source %q", forbidden)
		}
	}

	edges := strings.ToLower(graphEntityEdgesQuery)
	for _, fragment := range []string{
		"join entity_nodes from_node",
		"join entity_nodes to_node",
		"edge.status = 'active'",
		"from_node.status = 'active'",
		"to_node.status = 'active'",
		"from_node.entity_type in ('alliance_org', 'economy', 'chain_node')",
		"to_node.entity_type in ('alliance_org', 'economy', 'chain_node')",
		"from chain_node_relations relation",
		"relation.status = 'active'",
		"'postgres_entity_edges'",
		"'postgres_chain_node_relations'",
	} {
		if !strings.Contains(edges, fragment) {
			t.Fatalf("edge projection query missing %q: %s", fragment, graphEntityEdgesQuery)
		}
	}
	for _, forbidden := range []string{"sector_profiles", "industry_chain_profiles", "industry_chain_memberships", "industry_chain_topology_edges", "industry_chain_physical_constraints", "observation_records", "industry_chain_node_observations"} {
		if strings.Contains(edges, forbidden) {
			t.Fatalf("edge projection query contains forbidden source %q", forbidden)
		}
	}
}

func TestGraphEntityEdgeCarriesProjectionSource(t *testing.T) {
	if _, ok := reflect.TypeOf(GraphEntityEdge{}).FieldByName("Source"); !ok {
		t.Fatal("GraphEntityEdge is missing projection Source")
	}
}
