package repositories

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestInMemoryRepositoryListsPhysicalConstraintsByPathIDs(t *testing.T) {
	repo := NewInMemoryRepository(nil)
	repo.SeedIndustryChainPhysicalConstraints([]domain.IndustryChainPhysicalConstraint{
		{ID: "node", IndustryChainEntityID: "chain-a", ChainNodeEntityID: "node-a", ConstraintType: domain.PhysicalConstraintBandwidth, ReviewStatus: domain.ReviewStatusApproved, Status: domain.StatusActive},
		{ID: "edge", IndustryChainEntityID: "chain-a", TopologyEdgeID: "edge-a", ConstraintType: domain.PhysicalConstraintLatency, ReviewStatus: domain.ReviewStatusApproved, Status: domain.StatusActive},
		{ID: "candidate", IndustryChainEntityID: "chain-a", ChainNodeEntityID: "node-a", ConstraintType: domain.PhysicalConstraintPowerCapacity, ReviewStatus: domain.ReviewStatusCandidate, Status: domain.StatusActive},
	})
	got, err := repo.ListPhysicalConstraints(context.Background(), PhysicalConstraintFilter{ChainIDs: []string{"chain-a"}, NodeIDs: []string{"node-a"}, TopologyEdgeIDs: []string{"edge-a"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("constraints = %d, want 2", len(got))
	}
}

func TestPhysicalConstraintQueryIsApprovedActiveAndPostgresOnly(t *testing.T) {
	query := strings.ToLower(listPhysicalConstraintsQuery)
	for _, fragment := range []string{"industry_chain_physical_constraints", "review_status = 'approved'", "status = 'active'", "industry_chain_entity_id = any", "chain_node_entity_id = any", "topology_edge_id = any"} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query missing %q", fragment)
		}
	}
	for _, forbidden := range []string{"neo4j", "observation_records", "severity"} {
		if strings.Contains(query, forbidden) {
			t.Fatalf("query contains %q", forbidden)
		}
	}
	_ = time.Time{}
}
