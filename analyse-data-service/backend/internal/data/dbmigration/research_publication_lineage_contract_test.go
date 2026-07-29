package dbmigration

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicResearchPublicationLineageMigrationContract(t *testing.T) {
	raw := readMigration(t, filepath.Join(
		migrationDirectory(), "000033_add_atomic_research_publication_lineage.sql",
	))
	up, down := migrationSections(t, raw)
	up = strings.ToLower(up)
	down = strings.ToLower(down)

	for _, fragment := range []string{
		"publication_contract_version",
		"aggregate_theme_id",
		"reasoning_tree_ids_by_industry_chain_entity_id",
		"aggregate_write_counts",
		"incoming_source_kind",
		"direct_impact_assertion_id",
		"direct_impact_semantic_submission_id",
		"direct_impact_evidence_hash",
		"source_kind",
		"variable_signal_id",
		"semantic_submission_id",
		"evidence_hash",
		"analyst_inference",
		"formal_direct_impact",
		"formal_signal",
	} {
		if !strings.Contains(up, fragment) {
			t.Errorf("research publication lineage migration Up must contain %q", fragment)
		}
	}
	if !strings.Contains(down, "migration 000033 is forward-only") ||
		!strings.Contains(down, "raise exception") {
		t.Fatal("research publication lineage migration Down must fail closed")
	}
}
