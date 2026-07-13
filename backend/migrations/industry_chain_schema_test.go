package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestIndustryChainFoundationMigrationContract(t *testing.T) {
	data, err := os.ReadFile("000014_add_industry_chain_foundation.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := strings.ToLower(string(data))
	for _, fragment := range []string{
		"alter table chain_node_profiles",
		"add column if not exists node_category",
		"add column if not exists definition",
		"add column if not exists unit_of_analysis",
		"add column if not exists granularity_note",
		"create table industry_chain_profiles",
		"create table industry_chain_memberships",
		"create table industry_chain_topology_edges",
		"create table industry_chain_physical_constraints",
		"unique (industry_chain_entity_id, chain_node_entity_id)",
		"check (from_chain_node_entity_id <> to_chain_node_entity_id)",
		"check ((chain_node_entity_id is not null)::integer + (topology_edge_id is not null)::integer = 1)",
		"supplies_to", "depends_on", "substitutes_for",
		"power_capacity", "thermal_dissipation", "physical_expansion_cycle",
		"create index industry_chain_memberships_chain_status_order_idx",
		"create index industry_chain_physical_constraints_chain_idx",
		"generated_by_ai boolean not null default false",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"industry_chain_metric_definitions",
		"industry_chain_metric_bindings",
		"observation_records",
		"industry_chain_node_observations",
		"industry_chain_flow_observations",
		"severity",
	} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("migration contains forbidden scope %q", forbidden)
		}
	}
}
