package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestChainNodeRelationsMigrationStaticContract(t *testing.T) {
	b, err := os.ReadFile("000017_add_chain_node_relations.sql")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ToLower(string(b))
	for _, required := range []string{"create table chain_node_relations", "create table chain_node_physical_constraints", "references chain_node_profiles(entity_id)", "is_subcategory_of", "is_component_of", "input_to", "depends_on", "from_chain_node_entity_id <> to_chain_node_entity_id", "unique (from_chain_node_entity_id, to_chain_node_entity_id, relation_type)", "lower(btrim(mechanism))", "condition_note is null or btrim(condition_note) <> ''", "chain_node_relations_input_dependency_mechanism_uidx", "chain_node_relation_id", "constraint_type", "on delete restrict"} {
		if !strings.Contains(s, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
	for _, forbidden := range []string{"contains", "supplies_to", "substitutes_for", "transmits_to", "entity_edges"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("migration contains forbidden %q", forbidden)
		}
	}
}
