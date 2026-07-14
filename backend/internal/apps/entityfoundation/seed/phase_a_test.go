package seed

import (
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestDefaultSeedPathsExcludeRetiredIndustryModels(t *testing.T) {
	joined := strings.Join(DefaultSeedPaths("seed-root"), "\n")
	for _, forbidden := range []string{"sectors.json", "sector_source_mappings.json", "industry_chains_v1.json", "covers_sector.json", "tracked_by_benchmark.json"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("default production seed paths contain retired input %q", forbidden)
		}
	}
}

func TestProductionManifestRejectsRetiredModels(t *testing.T) {
	tests := []Manifest{
		{Entities: []Entity{{Key: "sector:test", EntityType: domain.EntityTypeSector}}},
		{Entities: []Entity{{Key: "industry_chain:test", EntityType: domain.EntityTypeIndustryChain}}},
		{SectorSourceMappings: []SectorSourceMapping{{SectorEntityKey: "sector:test"}}},
		{IndustryChainMemberships: []IndustryChainMembershipSeed{{ID: "membership"}}},
		{Relationships: []Relationship{{Key: "legacy-sector-endpoint", From: "market:test", To: "sector:test", RelationType: "covers_sector"}}},
		{Relationships: []Relationship{{Key: "legacy-chain-endpoint", From: "industry_chain:test", To: "economy:test", RelationType: "scoped_to_economy"}}},
	}
	for _, manifest := range tests {
		if err := ValidateProductionManifest(manifest); err == nil {
			t.Fatal("ValidateProductionManifest() error = nil")
		}
	}
}

func TestMinimalChainNodeAndThemeProfiles(t *testing.T) {
	for _, tc := range []struct {
		entityType domain.EntityType
		data       string
	}{
		{domain.EntityTypeChainNode, `{"definition":"节点定义","boundary_note":null}`},
		{domain.EntityTypeTheme, `{"definition":"投研视角","boundary_note":"不等同于节点"}`},
	} {
		if err := validateProfileData(tc.entityType, []byte(tc.data)); err != nil {
			t.Fatalf("validateProfileData(%q) error = %v", tc.entityType, err)
		}
	}
	if fields := requiredProfileFields(domain.EntityTypeTheme); strings.Join(fields, ",") != "definition,boundary_note" {
		t.Fatalf("required theme fields = %v", fields)
	}
}

func TestProductionChainNodeRejectsLegacyOrMissingProfileFields(t *testing.T) {
	for _, profile := range []string{
		`{"chain_position":"upstream"}`,
		`{"definition":"节点","node_category":"component"}`,
		`{"definition":" "}`,
	} {
		manifest := Manifest{Entities: []Entity{{Key: "chain_node:test", EntityType: domain.EntityTypeChainNode, Profile: []byte(profile)}}}
		if err := ValidateProductionManifest(manifest); err == nil {
			t.Fatalf("ValidateProductionManifest(%s) error = nil", profile)
		}
	}
}
