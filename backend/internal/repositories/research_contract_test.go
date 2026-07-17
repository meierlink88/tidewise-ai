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

func TestResearchMigrationAllTablesHaveAuditColumns(t *testing.T) {
	content, err := os.ReadFile("../../migrations/000021_add_research_theme_anchor_foundation.sql")
	if err != nil {
		t.Fatalf("read research migration: %v", err)
	}
	sql := strings.ToLower(string(content))

	for _, table := range []string{
		"research_themes", "research_theme_chain_nodes", "research_theme_indices", "research_theme_events",
		"research_anchors", "research_anchor_chain_nodes", "research_anchor_indices", "research_anchor_events",
	} {
		t.Run(table, func(t *testing.T) {
			start := strings.Index(sql, "create table "+table+" (")
			if start < 0 {
				t.Fatalf("migration missing table %q", table)
			}
			definition := sql[start:]
			if end := strings.Index(definition, "\n);"); end >= 0 {
				definition = definition[:end]
			} else {
				t.Fatalf("migration table %q has no closing definition", table)
			}
			for _, column := range []string{
				"created_at timestamptz not null default now()",
				"updated_at timestamptz not null default now()",
			} {
				if !strings.Contains(definition, column) {
					t.Fatalf("migration table %q missing audit column %q", table, column)
				}
			}
		})
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

func TestResearchCountQueriesDeduplicateMainRowsAndEvents(t *testing.T) {
	tests := []struct {
		name              string
		query             string
		distinctMainCount string
	}{
		{name: "themes", query: countResearchThemesQuery, distinctMainCount: "count(distinct t.id)"},
		{name: "anchors", query: countResearchAnchorsQuery, distinctMainCount: "count(distinct a.id)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := strings.Join(strings.Fields(strings.ToLower(test.query)), " ")
			for _, required := range []string{test.distinctMainCount, "count(distinct e.event_id)", "left join"} {
				if !strings.Contains(query, required) {
					t.Fatalf("count query missing %q: %s", required, query)
				}
			}
			if strings.Contains(query, "select count(*)") {
				t.Fatalf("count query must not count joined rows: %s", query)
			}
		})
	}
}

func TestResearchChainNodeQueriesUseRelationEntityID(t *testing.T) {
	for name, query := range map[string]string{
		"theme list":    listResearchThemesQuery,
		"theme detail":  getResearchThemeQuery,
		"anchor list":   listResearchAnchorsQuery,
		"anchor detail": getResearchAnchorQuery,
	} {
		t.Run(name, func(t *testing.T) {
			query = strings.ToLower(query)
			if !strings.Contains(query, "jsonb_build_object('id', n.chain_node_entity_id") {
				t.Fatal("chain-node JSON id must use n.chain_node_entity_id")
			}
			if strings.Contains(query, "n.entity_id") {
				t.Fatal("chain-node query references nonexistent n.entity_id")
			}
		})
	}
}
