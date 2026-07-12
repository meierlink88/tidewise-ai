package repositories

import (
	"strings"
	"testing"
)

func TestPostgresGraphProjectionQueriesRequireActiveRowsAndEndpoints(t *testing.T) {
	nodes := strings.ToLower(graphEntityNodesQuery)
	for _, fragment := range []string{"left join sector_profiles", "classification_code", "where node.status = 'active'"} {
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
	} {
		if !strings.Contains(edges, fragment) {
			t.Fatalf("edge projection query missing %q: %s", fragment, graphEntityEdgesQuery)
		}
	}
}
