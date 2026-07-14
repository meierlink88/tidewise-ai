package dbmigration

import (
	"strings"
	"testing"
)

func TestChainNodeRelationsMigrationStaticContract(t *testing.T) {
	s := strings.ToLower(readMigration(t, "000017_add_chain_node_relations.sql"))
	for _, required := range []string{"create table chain_node_relations", "create table chain_node_physical_constraints", "references chain_node_profiles(entity_id)", "is_subcategory_of", "is_component_of", "input_to", "depends_on", "from_chain_node_entity_id <> to_chain_node_entity_id", "unique (from_chain_node_entity_id, to_chain_node_entity_id, relation_type)", "lower(btrim(mechanism))", "condition_note is null or btrim(condition_note) <> ''", "chain_node_relations_input_dependency_mechanism_uidx", "chain_node_physical_constraints_node_subject_idx", "chain_node_physical_constraints_relation_subject_idx", "chain_node_relation_id", "constraint_type", "on delete restrict"} {
		if !strings.Contains(s, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
	for _, forbidden := range []string{"contains", "supplies_to", "substitutes_for", "transmits_to", "entity_edges"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("migration contains forbidden %q", forbidden)
		}
	}
	if strings.Count(s, "create unique index chain_node_physical_constraints") != 0 {
		t.Fatal("physical constraint semantic unique indexes require separate amendment approval")
	}
}

func TestIndustryChainFoundationMigrationContract(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000014_add_industry_chain_foundation.sql"))
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

func TestRefactorIndustryChainNodePhaseASchema(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000015_refactor_industry_chain_node_phase_a.sql"))
	for _, required := range []string{
		"current_setting('tidewise.phase_a_cleanup_write_authorized', true)",
		"reviewed_backup_verified",
		"cleanup write is not authorized",
		"create temp table phase_a_retired_entity_ids",
		"entity_type in ('sector', 'industry_chain', 'chain_node')",
		"delete from event_entity_links",
		"delete from entity_edges",
		"drop table industry_chain_physical_constraints",
		"drop table industry_chain_topology_edges",
		"drop table industry_chain_memberships",
		"drop table industry_chain_profiles",
		"drop table sector_source_mappings",
		"drop table sector_profiles",
		"drop table entity_convergence_reference_moves",
		"drop table entity_convergence_alias_moves",
		"drop table entity_convergences",
		"drop table entity_convergence_manifests",
		"drop function prevent_entity_convergence_audit_mutation",
		"drop column chain_position",
		"drop column node_category",
		"drop column unit_of_analysis",
		"drop column granularity_note",
		"delete from entity_nodes",
		"alter column definition set not null",
		"add column boundary_note text",
		"create table theme_profiles",
		"check (btrim(definition) <> '')",
		"check (boundary_note is null or btrim(boundary_note) <> '')",
		"check (btrim(boundary_note) <> '')",
		"unexpected reference",
		"raise exception 'migration 000015 is irreversible",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("phase A migration missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"chain_node_source_mappings",
		"unique (entity_key)",
		"research_theme",
		"chain_node_relations",
		"insert into entity_nodes",
		"update entity_nodes",
		"truncate ",
		" cascade",
		"select 'phase a rollback",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("phase A migration contains forbidden %q", forbidden)
		}
	}
}

func TestRefactorIndustryChainNodePhaseADeletesAreScoped(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000015_refactor_industry_chain_node_phase_a.sql"))
	for _, required := range []string{
		"where entity_id in (select id from phase_a_retired_entity_ids)",
		"where from_entity_id in (select id from phase_a_retired_entity_ids)",
		"or to_entity_id in (select id from phase_a_retired_entity_ids)",
		"where id in (select id from phase_a_retired_entity_ids)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("phase A cleanup missing scoped predicate %q", required)
		}
	}
}
