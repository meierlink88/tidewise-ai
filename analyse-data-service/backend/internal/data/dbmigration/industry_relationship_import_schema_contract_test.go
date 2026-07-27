package dbmigration

import (
	"regexp"
	"strings"
	"testing"
)

const industryRelationshipImportMigration = "000030_add_industry_relationship_import_contract.sql"

func TestIndustryRelationshipImportMigrationIsAdditiveSchemaOnly(t *testing.T) {
	raw := readMigration(t, industryRelationshipImportMigration)
	up, down := migrationSections(t, raw)
	normalized := strings.ToLower(up)

	for _, fragment := range []string{
		"alter table industry_chain_definitions",
		"add column technology_route_qualifier text",
		"add column observable_variables text[]",
		"alter table industry_chain_node_memberships",
		"add column inclusion_reason text",
		"add column evidence_ids text[]",
		"add column source_name text",
		"add column source_url text",
		"add column verified_at timestamptz",
		"alter table industry_chain_graph_edges",
		"create table industry_relationship_import_receipts",
		"package_sha256 text not null unique",
		"approval_basis = 'user_explicit_delegated_review'",
		"create trigger trg_industry_relationship_import_receipts_immutable",
		"prevent_industry_relationship_import_receipt_mutation",
		"industry chain memberships exist without v1 inclusion/provenance",
		"industry chain graph edges exist without v1 provenance",
		"industry chain definitions exist without v1 route/observable variables",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("industry relationship import migration Up must contain %q", fragment)
		}
	}

	for _, fragment := range []string{
		"inclusion_reason set not null",
		"source_name set not null",
		"source_url set not null",
		"verified_at set not null",
		"check (btrim(inclusion_reason) <> '')",
		"cardinality(evidence_ids) > 0",
		"cardinality(observable_variables) > 0",
		"check (source_url ~ '^(https?://|artifact://)')",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("industry relationship import migration must enforce %q", fragment)
		}
	}

	dml := regexp.MustCompile(`(?mi)^\s*(insert\s+into|update\s+|delete\s+from|truncate\s+)`)
	if match := dml.FindString(up); match != "" {
		t.Fatalf("industry relationship migration must contain no business DML, found %q", strings.TrimSpace(match))
	}
	if !strings.Contains(strings.ToLower(down), "migration 000030 is forward-only") ||
		!strings.Contains(strings.ToLower(down), "raise exception") {
		t.Fatal("industry relationship migration Down must fail closed as forward-only")
	}
}
