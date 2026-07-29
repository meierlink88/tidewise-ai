package dbmigration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEventSemanticsPhaseOneMigrationOwnsTBoxSubmissionsCandidatesAndAcceptedFacts(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000032_add_event_semantics_phase_one.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, fragment := range []string{
		"create table entity_type_definitions",
		"create table variable_definitions",
		"create table variable_definition_entity_types",
		"create table direct_transmission_rules",
		"create table event_semantic_context_leases",
		"context_snapshot jsonb",
		"create table event_semantic_submissions",
		"create table event_semantic_candidate_snapshots",
		"create table event_semantic_review_snapshots",
		"create table variable_signals",
		"create table variable_signal_measurements",
		"create table direct_impact_assertions",
		"alter table event_entity_links",
		"semantic_submission_id uuid references event_semantic_submissions",
		"evidence_ids uuid[]",
		"canonical_payload_hash char(64)",
		"unique (agent_execution_id)",
		"where review_status = 'accepted'",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("Event semantics migration missing %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"create table observations",
		"create table research_themes",
		"create table reasoning_trees",
		"neo4j",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("Event semantics migration contains out-of-scope fragment %q", forbidden)
		}
	}
}

func TestEventSemanticsPhaseOneSeedsOnlyFrozenVariableCatalog(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000032_add_event_semantics_phase_one.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, key := range []string{
		"market_supply", "market_demand", "market_price", "production_volume", "sales_volume",
		"order_quantity", "order_value", "revenue", "net_profit", "gross_margin",
		"policy_support_intensity", "regulatory_restriction_intensity",
	} {
		if !strings.Contains(sql, "'"+key+"'") {
			t.Fatalf("Event semantics migration missing frozen variable %q", key)
		}
	}
	for _, deferred := range []string{"'capacity'", "'capacity_utilization'", "'input_cost'"} {
		if strings.Contains(sql, deferred) {
			t.Fatalf("Event semantics migration must not seed deferred variable %s", deferred)
		}
	}
}
