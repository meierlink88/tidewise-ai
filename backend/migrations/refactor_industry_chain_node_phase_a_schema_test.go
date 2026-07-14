package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestRefactorIndustryChainNodePhaseASchema(t *testing.T) {
	data, err := os.ReadFile("000015_refactor_industry_chain_node_phase_a.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
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
	data, err := os.ReadFile("000015_refactor_industry_chain_node_phase_a.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
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
