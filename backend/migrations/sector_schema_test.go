package migrations_test

import (
	"strings"
	"testing"
)

func TestMarketSectorSchemaMigrationDefinesProfilesAndSourceMappings(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000010_add_market_sector_foundation.sql"))
	for _, fragment := range []string{
		"alter table sector_profiles",
		"add column if not exists classification_code text",
		"add column if not exists primary_market_entity_id uuid references entity_nodes(id)",
		"add column if not exists primary_economy_entity_id uuid references entity_nodes(id)",
		"add column if not exists methodology_url text",
		"add column if not exists review_status text",
		"classification_code in ('industry_sector', 'theme_sector', 'market_sector', 'style_sector', 'region_sector')",
		"create table if not exists sector_source_mappings",
		"sector_entity_id uuid not null references entity_nodes(id)",
		"source_taxonomy_type in ('concept', 'industry', 'index_sector')",
		"source_sector_name_normalized text not null",
		"source_market_scope text not null default ''",
		"mapping_status in ('candidate', 'approved', 'rejected', 'merged')",
		"create unique index if not exists uq_sector_source_mappings_code",
		"(source_system, source_taxonomy_type, source_sector_code)",
		"where source_sector_code <> ''",
		"create unique index if not exists uq_sector_source_mappings_name_scope",
		"(source_system, source_taxonomy_type, source_sector_name_normalized, source_market_scope)",
		"where source_sector_code = ''",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("market sector migration missing %q", fragment)
		}
	}
	if strings.Contains(sql, "source_sector_name_normalized, source_market_scope, snapshot_date") {
		t.Fatal("source mapping identity must not include snapshot_date")
	}
}

func TestMarketSectorSchemaMigrationIsNonDestructive(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000010_add_market_sector_foundation.sql"))
	for _, forbidden := range []string{"drop table", "drop column", "truncate", "delete from"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("market sector migration contains destructive fragment %q", forbidden)
		}
	}
	if !strings.Contains(sql, "rollback requires a reviewed forward migration or restored backup") {
		t.Fatal("market sector migration must document forward-only rollback")
	}
}
