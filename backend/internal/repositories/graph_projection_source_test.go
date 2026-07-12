package repositories

import (
	"strings"
	"testing"
)

func TestPostgresGraphProjectionQueriesRequireActiveRowsAndEndpoints(t *testing.T) {
	nodes := strings.ToLower(graphEntityNodesQuery)
	for _, fragment := range []string{"left join sector_profiles", "left join industry_chain_profiles", "classification_code", "where node.status = 'active'", "chain.review_status = 'approved'"} {
		if !strings.Contains(nodes, fragment) {
			t.Fatalf("node projection query missing %q: %s", fragment, graphEntityNodesQuery)
		}
	}

	edges := strings.ToLower(graphEntityEdgesQuery)
	for _, fragment := range []string{
		"join entity_nodes from_node",
		"join entity_nodes to_node",
		"edge.status = 'active'",
		"from_node.status = 'active'",
		"to_node.status = 'active'",
		"industry_chain_memberships",
		"member_of_chain",
		"industry_chain_topology_edges",
		"supplies_to",
	} {
		if !strings.Contains(edges, fragment) {
			t.Fatalf("edge projection query missing %q: %s", fragment, graphEntityEdgesQuery)
		}
	}
	for _, forbidden := range []string{"industry_chain_physical_constraints", "observation_records", "industry_chain_node_observations"} {
		if strings.Contains(edges, forbidden) {
			t.Fatalf("edge projection query contains forbidden source %q", forbidden)
		}
	}
}
