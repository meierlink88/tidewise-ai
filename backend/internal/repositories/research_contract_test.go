package repositories

import (
	"os"
	"strings"
	"testing"
)

func TestResearchMigrationContract(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000021_add_research_theme_anchor_foundation.sql")
	if err != nil {
		t.Fatalf("read research migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, table := range []string{
		"research_themes", "research_theme_chain_nodes", "research_theme_indices", "research_theme_events",
		"research_anchors", "research_anchor_chain_nodes", "research_anchor_indices", "research_anchor_events",
	} {
		if !strings.Contains(sql, "create table "+table) {
			t.Fatalf("migration missing table %q", table)
		}
	}
	for _, fragment := range []string{
		"impact_level in ('high', 'focus', 'watch')",
		"evidence_role in ('driver', 'supporting', 'contradicting', 'context')",
		"window_end >= window_start",
		"on delete cascade",
		"references events(id)",
		"references chain_node_profiles(entity_id)",
		"references index_profiles(entity_id)",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing contract %q", fragment)
		}
	}
	if strings.Contains(sql, "display_order") || strings.Contains(sql, "research_conclusions") || strings.Contains(sql, "research_analysis_runs") {
		t.Fatal("migration contains an explicitly forbidden structure")
	}
}

func TestResearchReadQueriesArePostgresOnlyAndBatchAggregated(t *testing.T) {
	for name, query := range map[string]string{
		"theme list":    listResearchThemesQuery,
		"theme detail":  getResearchThemeQuery,
		"anchor list":   listResearchAnchorsQuery,
		"anchor detail": getResearchAnchorQuery,
	} {
		t.Run(name, func(t *testing.T) {
			value := strings.ToLower(query)
			for _, required := range []string{"jsonb_agg", "count(distinct event_id)"} {
				if !strings.Contains(value, required) {
					t.Fatalf("query missing %q", required)
				}
			}
			for _, forbidden := range []string{"neo4j", "raw_documents", "research_anchor_events r join research_theme_events"} {
				if strings.Contains(value, forbidden) {
					t.Fatalf("query contains forbidden fragment %q", forbidden)
				}
			}
		})
	}
}
