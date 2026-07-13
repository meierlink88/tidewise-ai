package seed

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestIndustryChainExecutableSeedV1(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	manifest, err := LoadFiles(DefaultSeedPaths(root)...)
	if err != nil {
		t.Fatalf("LoadFiles() error = %v", err)
	}
	chains, nodes := 0, 0
	for _, entity := range manifest.Entities {
		switch entity.EntityType {
		case domain.EntityTypeIndustryChain:
			chains++
		case domain.EntityTypeChainNode:
			nodes++
		}
	}
	if chains != 2 || nodes != 54 {
		t.Fatalf("chains/nodes = %d/%d, want 2/54 including existing 33", chains, nodes)
	}
	if len(manifest.IndustryChainMemberships) != 27 || len(manifest.IndustryChainTopologyEdges) != 24 {
		t.Fatalf("memberships/topology = %d/%d", len(manifest.IndustryChainMemberships), len(manifest.IndustryChainTopologyEdges))
	}
	if len(manifest.IndustryChainPhysicalConstraints) != 4 {
		t.Fatalf("executable physical constraints = %d, want 4", len(manifest.IndustryChainPhysicalConstraints))
	}
	approvedMappings := map[string]Relationship{}
	for _, relationship := range manifest.Relationships {
		if relationship.RelationType == "mapped_to_sector" {
			approvedMappings[relationship.From+"->"+relationship.To] = relationship
		}
	}
	wantMappings := []string{
		"industry_chain:ai_compute_infrastructure->sector:theme_computing_infrastructure",
		"chain_node:data_center->sector:theme_data_centers_cloud",
		"industry_chain:semiconductor_manufacturing->sector:industry_semiconductors_electronics",
		"chain_node:lithography_machine->sector:industry_star_semiconductor_materials_equipment",
		"chain_node:deposition_equipment->sector:industry_star_semiconductor_materials_equipment",
		"chain_node:etch_equipment->sector:industry_star_semiconductor_materials_equipment",
	}
	if len(approvedMappings) != len(wantMappings) {
		t.Fatalf("approved mapped_to_sector = %d, want %d", len(approvedMappings), len(wantMappings))
	}
	for _, key := range wantMappings {
		mapping, ok := approvedMappings[key]
		if !ok {
			t.Fatalf("missing approved mapped_to_sector %s", key)
		}
		if mapping.SourceName == "" || mapping.SourceURL == "" || mapping.VerifiedAt.IsZero() || !strings.Contains(mapping.EvidenceNote, "composite curation") {
			t.Fatalf("mapping %s lacks composite provenance: %+v", key, mapping)
		}
	}
	approved := map[string]IndustryChainPhysicalConstraintSeed{}
	for _, constraint := range manifest.IndustryChainPhysicalConstraints {
		approved[constraint.ID] = constraint
		if constraint.ReviewStatus != domain.ReviewStatusApproved || !constraint.GeneratedByAI || !constraint.ApprovedByHuman {
			t.Fatalf("approved constraint gate/provenance = %+v", constraint)
		}
	}
	if approved["constraint:ai:data_center:infrastructure_access"].SourceURL != "https://www.iea.org/reports/energy-and-ai/executive-summary" {
		t.Fatal("infrastructure access did not use direct P2 provenance")
	}
	if approved["constraint:semi:advanced_packaging:reliability"].SourceURL != "https://3dfabric.tsmc.com/schinese/dedicatedFoundry/technology/del/cowos_publications_ECTC2013.htm" {
		t.Fatal("packaging reliability did not use direct P6 provenance")
	}
	pilotKeys := map[string]struct{}{}
	for _, membership := range manifest.IndustryChainMemberships {
		pilotKeys[membership.ChainNodeKey] = struct{}{}
	}
	if len(pilotKeys) != 26 {
		t.Fatalf("unique pilot node keys = %d, want 26", len(pilotKeys))
	}
	entities := map[string]Entity{}
	profiles := map[string]json.RawMessage{}
	for _, entity := range manifest.Entities {
		entities[entity.Key] = entity
		profiles[entity.Key] = entity.Profile
	}
	for _, profile := range manifest.Profiles {
		profiles[profile.EntityKey] = profile.Data
	}
	for key := range pilotKeys {
		entity := entities[key]
		if entity.Name == "" || !aliasesContainUnicodeScript(entity.Aliases, unicode.Latin) {
			t.Fatalf("pilot node %s lacks Chinese primary name or English alias", key)
		}
		for _, field := range []string{"node_category", "definition", "unit_of_analysis", "granularity_note"} {
			if profileString(t, profiles[key], field) == "" {
				t.Fatalf("pilot node %s profile missing %s", key, field)
			}
		}
	}
	for _, edge := range manifest.IndustryChainTopologyEdges {
		if edge.RelationType == domain.IndustryChainRelationSubstitutesFor {
			t.Fatal("executable seed contains speculative substitutes_for")
		}
	}
}

func TestIndustryChainExecutableSeedFlowsThroughServiceReport(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	manifest, err := LoadFiles(DefaultSeedPaths(root)...)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMemoryRepository()
	report, err := NewService(repo).Apply(context.Background(), manifest, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if report.IndustryChainCounts["membership"] != 27 || report.IndustryChainCounts["topology"] != 24 || report.IndustryChainCounts["physical_constraint"] != 4 {
		t.Fatalf("industry chain report = %+v", report.IndustryChainCounts)
	}
	if got := repo.IndustryChainRowCount(); got != 55 {
		t.Fatalf("industry chain repository rows = %d, want 55", got)
	}
}

func TestIndustryChainCandidateFixtureIsReviewOnly(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	reviewPath := filepath.Join(root, "review", "industry_chain_candidates_v1.json")
	for _, path := range DefaultSeedPaths(root) {
		if path == reviewPath {
			t.Fatal("candidate review fixture is executable")
		}
	}
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("read review fixture: %v", err)
	}
	var fixture struct {
		ManifestVersion     int                                   `json:"manifest_version"`
		PhysicalConstraints []IndustryChainPhysicalConstraintSeed `json:"physical_constraints"`
		MappedToSector      []struct {
			ReviewStatus string `json:"review_status"`
		} `json:"mapped_to_sector"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.ManifestVersion != 1 || len(fixture.PhysicalConstraints) != 11 || len(fixture.MappedToSector) != 6 {
		t.Fatalf("review fixture counts = version %d, constraints %d, sector mappings %d", fixture.ManifestVersion, len(fixture.PhysicalConstraints), len(fixture.MappedToSector))
	}
	for _, constraint := range fixture.PhysicalConstraints {
		if constraint.ReviewStatus != domain.ReviewStatusCandidate || !constraint.GeneratedByAI || constraint.ApprovedByHuman {
			t.Fatalf("writable constraint leaked into review fixture: %+v", constraint)
		}
	}
	for _, mapping := range fixture.MappedToSector {
		if !strings.EqualFold(mapping.ReviewStatus, "candidate") {
			t.Fatalf("sector mapping review status = %q", mapping.ReviewStatus)
		}
	}
}
