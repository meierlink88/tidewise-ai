package graphprojection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestProjectorProjectsEntitiesAndRelationships(t *testing.T) {
	repo := newProjectorRepository()
	repo.nodes = []repositories.GraphEntityNode{
		projectorNode("entity-1", "economy:cn"),
		projectorNode("entity-2", "alliance:g20"),
	}
	repo.edges = []repositories.GraphEntityEdge{
		{ID: "edge-1", FromEntityID: "entity-1", ToEntityID: "entity-2", RelationType: "member_of", Status: domain.StatusActive, UpdatedAt: repo.now},
	}
	writer := &recordingGraphWriter{}

	report, err := NewProjector(repo, writer, "tidewise", func() time.Time { return repo.now }).ProjectEntities(context.Background(), ProjectOptions{Mode: repositories.GraphProjectionModeProjectEntities})
	if err != nil {
		t.Fatalf("ProjectEntities() error = %v", err)
	}

	if report.Status != repositories.GraphProjectionRunStatusSucceeded {
		t.Fatalf("report status = %q, want succeeded", report.Status)
	}
	if report.SourceRowCount != 3 || report.ProjectedCount != 3 || report.SkippedCount != 0 || report.FailedCount != 0 {
		t.Fatalf("report counts = %+v", report)
	}
	if len(writer.nodes) != 2 || len(writer.relationships) != 1 {
		t.Fatalf("writer nodes/relationships = %d/%d, want 2/1", len(writer.nodes), len(writer.relationships))
	}
	if len(repo.completedRuns) != 1 || repo.completedRuns[0].Status != repositories.GraphProjectionRunStatusSucceeded {
		t.Fatalf("completed runs = %+v", repo.completedRuns)
	}
}

func TestProjectorWritesBenchmarkChainNodeSectorPathWithoutOverseasMarketCover(t *testing.T) {
	repo := newProjectorRepository()
	benchmark := projectorNode("benchmark", "benchmark:global")
	benchmark.EntityType = domain.EntityTypeBenchmark
	chain := projectorNode("chain", "industry_chain:test")
	chain.EntityType = domain.EntityTypeIndustryChain
	node := projectorNode("node", "chain_node:test")
	node.EntityType = domain.EntityTypeChainNode
	sector := projectorNode("sector", "sector:cn")
	sector.EntityType = domain.EntityTypeSector
	sector.ClassificationCode = domain.SectorClassificationTheme
	repo.nodes = []repositories.GraphEntityNode{benchmark, chain, node, sector}
	repo.edges = []repositories.GraphEntityEdge{
		{ID: "observed", FromEntityID: chain.ID, ToEntityID: benchmark.ID, RelationType: "observed_by_benchmark", Status: domain.StatusActive},
		{ID: "membership", FromEntityID: node.ID, ToEntityID: chain.ID, RelationType: "member_of_chain", Status: domain.StatusActive},
		{ID: "sector-map", FromEntityID: node.ID, ToEntityID: sector.ID, RelationType: "mapped_to_sector", Status: domain.StatusActive},
	}
	writer := &recordingGraphWriter{}
	if _, err := NewProjector(repo, writer, "tidewise", func() time.Time { return repo.now }).ProjectEntities(context.Background(), ProjectOptions{}); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, relationship := range writer.relationships {
		got[relationship.RelationshipType] = true
		if relationship.RelationshipType == "COVERS_SECTOR" {
			t.Fatal("projected overseas market COVERS_SECTOR")
		}
	}
	for _, want := range []string{"OBSERVED_BY_BENCHMARK", "MEMBER_OF_CHAIN", "MAPPED_TO_SECTOR"} {
		if !got[want] {
			t.Fatalf("missing projected path relation %s: %+v", want, writer.relationships)
		}
	}
}

func TestProjectorDefensivelyExcludesInactiveNodesAndEdges(t *testing.T) {
	repo := newProjectorRepository()
	activeA := projectorNode("active-a", "economy:active-a")
	activeB := projectorNode("active-b", "market:active-b")
	inactive := projectorNode("inactive", "sector:legacy")
	inactive.Status = domain.StatusInactive
	repo.nodes = []repositories.GraphEntityNode{activeA, inactive, activeB}
	repo.edges = []repositories.GraphEntityEdge{
		{ID: "active-edge", FromEntityID: activeA.ID, ToEntityID: activeB.ID, RelationType: "has_market", Status: domain.StatusActive, UpdatedAt: repo.now},
		{ID: "inactive-endpoint", FromEntityID: activeB.ID, ToEntityID: inactive.ID, RelationType: "covers_sector", Status: domain.StatusActive, UpdatedAt: repo.now},
		{ID: "inactive-edge", FromEntityID: activeA.ID, ToEntityID: activeB.ID, RelationType: "has_market", Status: domain.StatusInactive, UpdatedAt: repo.now},
	}
	writer := &recordingGraphWriter{}

	report, err := NewProjector(repo, writer, "tidewise", func() time.Time { return repo.now }).ProjectEntities(context.Background(), ProjectOptions{})
	if err != nil {
		t.Fatalf("ProjectEntities() error = %v", err)
	}
	if report.SourceRowCount != 3 || report.ProjectedCount != 3 || report.SkippedCount != 0 || report.FailedCount != 0 {
		t.Fatalf("report counts = %+v, want three active source rows projected", report)
	}
	if len(writer.nodes) != 2 || len(writer.relationships) != 1 || writer.relationships[0].EdgeID != "active-edge" {
		t.Fatalf("writer nodes/relationships = %+v/%+v", writer.nodes, writer.relationships)
	}
	for _, node := range writer.nodes {
		if node.Status != string(domain.StatusActive) || node.Namespace != "tidewise" {
			t.Fatalf("projected node = %+v, want active tidewise node", node)
		}
	}
}

func TestProjectorFailsInvalidSectorClassificationWithoutPollutingOtherNodes(t *testing.T) {
	repo := newProjectorRepository()
	valid := projectorNode("market", "market:test")
	valid.EntityType = domain.EntityTypeMarket
	invalid := projectorNode("sector", "sector:invalid")
	invalid.EntityType = domain.EntityTypeSector
	invalid.LayerCode = "sector"
	repo.nodes = []repositories.GraphEntityNode{valid, invalid}
	repo.edges = []repositories.GraphEntityEdge{{
		ID: "edge", FromEntityID: valid.ID, ToEntityID: invalid.ID,
		RelationType: "covers_sector", Status: domain.StatusActive, UpdatedAt: repo.now,
	}}
	writer := &recordingGraphWriter{}

	report, err := NewProjector(repo, writer, "tidewise", func() time.Time { return repo.now }).ProjectEntities(context.Background(), ProjectOptions{Mode: repositories.GraphProjectionModeRebuildEntities})
	if err != nil {
		t.Fatalf("ProjectEntities() error = %v", err)
	}
	if report.Status != repositories.GraphProjectionRunStatusPartial || report.SourceRowCount != 3 || report.ProjectedCount != 1 || report.FailedCount != 1 || report.SkippedCount != 1 {
		t.Fatalf("report = %+v, want one projected non-sector and failed sector plus skipped edge", report)
	}
	if len(writer.nodes) != 1 || writer.nodes[0].EntityID != valid.ID || writer.nodes[0].ClassificationCode != "" || writer.nodes[0].Namespace != "tidewise" {
		t.Fatalf("writer nodes = %+v", writer.nodes)
	}
	if len(writer.deletedNamespaces) != 1 || writer.deletedNamespaces[0] != "tidewise" {
		t.Fatalf("deleted namespaces = %+v", writer.deletedNamespaces)
	}
}

func TestProjectorExcludesBenchmarkObservationsFromSourceRowsAndGraphWrites(t *testing.T) {
	repo := newProjectorRepository()
	benchmarkNode := projectorNode("benchmark-1", "benchmark:us_10y_treasury_yield")
	benchmarkNode.EntityType = domain.EntityTypeBenchmark
	benchmarkNode.LayerCode = "market"
	repo.nodes = []repositories.GraphEntityNode{benchmarkNode}
	repo.observations = []domain.BenchmarkObservation{
		{
			ID:                "observation-1",
			BenchmarkEntityID: "benchmark-1",
			ObservedAt:        repo.now,
			Value:             "4.25",
			Unit:              "percent",
			SourceName:        "US Treasury",
			QualityStatus:     domain.BenchmarkObservationQualityValidated,
		},
	}
	writer := &recordingGraphWriter{}

	report, err := NewProjector(repo, writer, "tidewise", func() time.Time { return repo.now }).ProjectEntities(context.Background(), ProjectOptions{})
	if err != nil {
		t.Fatalf("ProjectEntities() error = %v", err)
	}

	if report.SourceRowCount != 1 {
		t.Fatalf("SourceRowCount = %d, want only one entity row and no observation rows", report.SourceRowCount)
	}
	if len(writer.nodes) != 1 || len(writer.relationships) != 0 {
		t.Fatalf("writer nodes/relationships = %d/%d, want 1/0 with no observation writes", len(writer.nodes), len(writer.relationships))
	}
	if writer.nodes[0].EntityID != "benchmark-1" {
		t.Fatalf("writer node entity ID = %q, want benchmark definition only", writer.nodes[0].EntityID)
	}
}

func TestProjectorRecordsSkippedRelationshipAsPartial(t *testing.T) {
	repo := newProjectorRepository()
	repo.nodes = []repositories.GraphEntityNode{projectorNode("entity-1", "economy:cn")}
	repo.edges = []repositories.GraphEntityEdge{
		{ID: "edge-missing", FromEntityID: "entity-1", ToEntityID: "missing", RelationType: "member_of", Status: domain.StatusActive, UpdatedAt: repo.now},
	}

	report, err := NewProjector(repo, &recordingGraphWriter{}, "tidewise", func() time.Time { return repo.now }).ProjectEntities(context.Background(), ProjectOptions{})
	if err != nil {
		t.Fatalf("ProjectEntities() error = %v", err)
	}

	if report.Status != repositories.GraphProjectionRunStatusPartial || report.SkippedCount != 1 {
		t.Fatalf("report = %+v, want partial with one skipped", report)
	}
	if len(repo.items) != 1 || repo.items[0].Status != repositories.GraphProjectionRunItemStatusSkipped {
		t.Fatalf("run items = %+v, want skipped item", repo.items)
	}
}

func TestProjectorMarksRunFailedWhenWriterFails(t *testing.T) {
	expected := errors.New("neo4j write failed")
	repo := newProjectorRepository()
	repo.nodes = []repositories.GraphEntityNode{projectorNode("entity-1", "economy:cn")}
	writer := &recordingGraphWriter{upsertEntitiesErr: expected}

	report, err := NewProjector(repo, writer, "tidewise", func() time.Time { return repo.now }).ProjectEntities(context.Background(), ProjectOptions{})
	if !errors.Is(err, expected) {
		t.Fatalf("ProjectEntities() error = %v, want %v", err, expected)
	}
	if report.Status != repositories.GraphProjectionRunStatusFailed || report.FailedCount != 1 {
		t.Fatalf("report = %+v, want failed with one failure", report)
	}
	if len(repo.completedRuns) != 1 || repo.completedRuns[0].Status != repositories.GraphProjectionRunStatusFailed {
		t.Fatalf("completed runs = %+v, want failed run", repo.completedRuns)
	}
}

func TestProjectorRebuildDeletesNamespaceBeforeProjection(t *testing.T) {
	repo := newProjectorRepository()
	repo.nodes = []repositories.GraphEntityNode{projectorNode("entity-1", "economy:cn")}
	writer := &recordingGraphWriter{}

	_, err := NewProjector(repo, writer, "tidewise", func() time.Time { return repo.now }).ProjectEntities(context.Background(), ProjectOptions{Mode: repositories.GraphProjectionModeRebuildEntities})
	if err != nil {
		t.Fatalf("ProjectEntities() error = %v", err)
	}
	if len(writer.deletedNamespaces) != 1 || writer.deletedNamespaces[0] != "tidewise" {
		t.Fatalf("deleted namespaces = %+v, want tidewise", writer.deletedNamespaces)
	}
}

func TestProjectorCanRunRepeatedlyWithStableInputs(t *testing.T) {
	repo := newProjectorRepository()
	repo.nodes = []repositories.GraphEntityNode{projectorNode("entity-1", "economy:cn")}
	writer := &recordingGraphWriter{}
	projector := NewProjector(repo, writer, "tidewise", func() time.Time { return repo.now })

	if _, err := projector.ProjectEntities(context.Background(), ProjectOptions{}); err != nil {
		t.Fatalf("ProjectEntities(first) error = %v", err)
	}
	if _, err := projector.ProjectEntities(context.Background(), ProjectOptions{}); err != nil {
		t.Fatalf("ProjectEntities(second) error = %v", err)
	}

	if len(repo.createdRuns) != 2 || len(writer.nodes) != 2 {
		t.Fatalf("created runs / node writes = %d/%d, want 2/2", len(repo.createdRuns), len(writer.nodes))
	}
}

func projectorNode(id string, key string) repositories.GraphEntityNode {
	return repositories.GraphEntityNode{
		ID:            id,
		EntityKey:     key,
		EntityType:    domain.EntityTypeEconomy,
		LayerCode:     "economy",
		Name:          key,
		CanonicalName: key,
		Status:        domain.StatusActive,
		UpdatedAt:     time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
	}
}

type projectorRepository struct {
	nodes        []repositories.GraphEntityNode
	edges        []repositories.GraphEntityEdge
	observations []domain.BenchmarkObservation
	now          time.Time

	createdRuns   []repositories.GraphProjectionRun
	completedRuns []repositories.GraphProjectionRun
	items         []repositories.GraphProjectionRunItem
}

func newProjectorRepository() *projectorRepository {
	return &projectorRepository{now: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)}
}

func (r *projectorRepository) ListGraphEntityNodes(context.Context) ([]repositories.GraphEntityNode, error) {
	return append([]repositories.GraphEntityNode(nil), r.nodes...), nil
}

func (r *projectorRepository) ListGraphEntityEdges(context.Context) ([]repositories.GraphEntityEdge, error) {
	return append([]repositories.GraphEntityEdge(nil), r.edges...), nil
}

func (r *projectorRepository) CreateGraphProjectionRun(_ context.Context, run repositories.GraphProjectionRun) (repositories.GraphProjectionRun, error) {
	r.createdRuns = append(r.createdRuns, run)
	return run, nil
}

func (r *projectorRepository) RecordGraphProjectionRunItem(_ context.Context, item repositories.GraphProjectionRunItem) error {
	r.items = append(r.items, item)
	return nil
}

func (r *projectorRepository) CompleteGraphProjectionRun(_ context.Context, run repositories.GraphProjectionRun) error {
	r.completedRuns = append(r.completedRuns, run)
	return nil
}

func (r *projectorRepository) RecentGraphProjectionRuns(context.Context, int) ([]repositories.GraphProjectionRun, error) {
	return append([]repositories.GraphProjectionRun(nil), r.completedRuns...), nil
}
