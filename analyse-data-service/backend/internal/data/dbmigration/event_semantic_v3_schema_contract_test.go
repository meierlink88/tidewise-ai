package dbmigration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEventSemanticV3MigrationExtendsExistingEntityTypeDefinitions(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000037_event_semantic_entity_first_v3.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, fragment := range []string{
		"alter table entity_type_definitions",
		"add column name_zh text",
		"add column name_en text",
		"add column business_definition text",
		"add column inclusion_criteria text[]",
		"add column exclusion_criteria text[]",
		"add column event_link_allowed boolean",
		"every active entity type definition requires complete v3 semantics",
		"alter column name_zh set not null",
		"cardinality(inclusion_criteria) > 0",
		"array_to_string(inclusion_criteria",
		"[[:space:]]*",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("Event Semantic V3 migration missing %q", fragment)
		}
	}
	for _, typeKey := range []string{
		"alliance_org", "chain_node", "commodity", "company", "concept", "industry",
		"industry_chain", "person", "policy_body", "product", "sector", "security",
	} {
		if !strings.Contains(sql, "('"+typeKey+"', 1") {
			t.Fatalf("Event Semantic V3 migration does not author %q", typeKey)
		}
	}
	for _, forbidden := range []string{"create table entity_type_definitions", "drop table", "delete from entity_type_definitions"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("Event Semantic V3 migration contains forbidden fragment %q", forbidden)
		}
	}
}
