package repositories

import (
	"strings"
	"testing"
)

func TestPostgresGraphProjectionQueriesRequireActiveRowsAndEndpoints(t *testing.T) {
	nodes := strings.ToLower(graphEntityNodesQuery)
	if !strings.Contains(nodes, "where status = 'active'") {
		t.Fatalf("node projection query does not require active status: %s", graphEntityNodesQuery)
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
