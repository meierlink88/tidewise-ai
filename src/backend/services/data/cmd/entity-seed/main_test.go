package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	entityseed "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/entityseed"
)

func TestValidateCommandOptionsRejectsRetiredApplyScopes(t *testing.T) {
	for _, retired := range []string{
		"industry-chain-master",
		"industry-chain-membership",
		"industry-chain-topology",
		"industry-chain-physical-constraint",
		"industry-chain-sector-mapping",
	} {
		if _, _, err := validateCommandOptions(commandOptions{applyScope: retired}); err == nil {
			t.Fatalf("validateCommandOptions(%q) error = nil", retired)
		}
	}
}

func TestValidateCommandOptionsFailsClosedForMappingMode(t *testing.T) {
	tests := []commandOptions{
		{mappingDryRun: true},
		{mappingPreflight: true},
		{mappingApprovedFirstBatch: true},
		{mappingManifest: "reviewed.json", mappingDryRun: true, mappingPreflight: true},
		{mappingManifest: "reviewed.json", seedDir: "non-default"},
		{mappingManifest: "reviewed.json", manifestFile: "entities.json"},
	}
	for _, options := range tests {
		if _, _, err := validateCommandOptions(options); err == nil {
			t.Fatalf("validateCommandOptions(%+v) error = nil", options)
		}
	}
	_, mode, err := validateCommandOptions(commandOptions{mappingManifest: "reviewed.json", seedDir: entityseed.DefaultSeedDir})
	if err != nil || !mode {
		t.Fatalf("mapping mode = %t, %v", mode, err)
	}
}

func TestValidateRelationCommandOptionsRequiresExactlyOneIsolatedMode(t *testing.T) {
	for _, options := range []relationCommandOptions{{dryRun: true}, {approvedWrite: true}, {manifest: "relations.json"}, {manifest: "relations.json", dryRun: true, approvedWrite: true}, {manifest: "relations.json", dryRun: true, seedDir: "other"}, {manifest: "relations.json", approvedWrite: true, mappingManifest: "mapping.json"}} {
		if err := validateRelationCommandOptions(options); err == nil {
			t.Fatalf("validateRelationCommandOptions(%+v) error=nil", options)
		}
	}
	if err := validateRelationCommandOptions(relationCommandOptions{manifest: "relations.json", dryRun: true, seedDir: entityseed.DefaultSeedDir}); err != nil {
		t.Fatal(err)
	}
	if err := validateRelationCommandOptions(relationCommandOptions{manifest: "relations.json", approvedWrite: true, seedDir: entityseed.DefaultSeedDir}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRelationDryRunManifestRejectsNonFrozenPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relations.json")
	content := []byte(`{"relations":[]}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRelationDryRunManifest(path); err == nil {
		t.Fatal("loadRelationDryRunManifest() error = nil")
	}
}

func TestLoadRelationDryRunManifestReadsFrozenAdditiveRelations(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "relationships", "reviewed_chain_node_relations", "additive-final-candidate-manifest.json")
	manifest, err := loadRelationDryRunManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(manifest.Relations); got != 212 {
		t.Fatalf("relations = %d, want 212", got)
	}
}

func TestManifestPreflightProofUsesExplicitFileAndCountsChainNodeProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chain-nodes.json")
	content := []byte(`{"entities":[{"key":"chain_node:test","entity_type":"chain_node","layer_code":"chain_node","name":"测试节点","canonical_name":"测试节点","status":"active","profile":{"definition":"用于验证显式节点 seed 文件的稳定产业节点。"}}]}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := entityseed.Manifest{Entities: []entityseed.Entity{{EntityType: domain.EntityTypeChainNode, Profile: content}}}
	proof, err := manifestPreflightProof(path, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if proof.EntityCount != 1 || proof.ChainNodeCount != 1 || proof.ProfileCount != 1 || proof.SHA256 == "" {
		t.Fatalf("proof = %+v", proof)
	}
}

func TestLoadManifestUsesExplicitManifestFileWithoutDefaultSeedPaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chain-nodes.json")
	if err := os.WriteFile(path, []byte(`{"entities":[{"key":"chain_node:test","entity_type":"chain_node","layer_code":"chain_node","name":"测试节点","canonical_name":"测试节点","status":"active","profile":{"definition":"用于验证显式节点 seed 文件的稳定产业节点。"}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifest("missing-default-seed-root", path)
	if err != nil {
		t.Fatalf("loadManifest() error = %v", err)
	}
	if len(manifest.Entities) != 1 || manifest.Entities[0].Key != "chain_node:test" {
		t.Fatalf("manifest = %+v", manifest)
	}
}

func TestValidateAllianceEconomyCommandOptionsRequiresOneIsolatedMode(t *testing.T) {
	valid := []allianceEconomyCommandOptions{
		{manifest: "approved.json", dependencyAudit: true, seedDir: entityseed.DefaultSeedDir},
		{manifest: "approved.json", cleanupApprovedLocal: true, dependencyChecksum: "sha256", seedDir: entityseed.DefaultSeedDir},
		{manifest: "approved.json", rebuildApprovedLocal: true, seedDir: entityseed.DefaultSeedDir},
	}
	for _, options := range valid {
		if err := validateAllianceEconomyCommandOptions(options); err != nil {
			t.Fatalf("validateAllianceEconomyCommandOptions(%+v) error = %v", options, err)
		}
	}

	invalid := []allianceEconomyCommandOptions{
		{dependencyAudit: true, seedDir: entityseed.DefaultSeedDir},
		{manifest: "approved.json", seedDir: entityseed.DefaultSeedDir},
		{manifest: "approved.json", dependencyAudit: true, rebuildApprovedLocal: true, seedDir: entityseed.DefaultSeedDir},
		{manifest: "approved.json", cleanupApprovedLocal: true, seedDir: entityseed.DefaultSeedDir},
		{manifest: "approved.json", rebuildApprovedLocal: true, manifestFile: "other.json", seedDir: entityseed.DefaultSeedDir},
		{manifest: "approved.json", rebuildApprovedLocal: true, relationManifest: "relations.json", seedDir: entityseed.DefaultSeedDir},
	}
	for _, options := range invalid {
		if err := validateAllianceEconomyCommandOptions(options); err == nil {
			t.Fatalf("validateAllianceEconomyCommandOptions(%+v) error = nil", options)
		}
	}
}

func TestValidateAllianceEconomyLocalTarget(t *testing.T) {
	valid := config.Config{App: config.AppConfig{Env: config.EnvLocal}, Database: config.DatabaseConfig{Name: "tidewise_local"}}
	if err := validateAllianceEconomyLocalTarget(valid); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []config.Config{
		{App: config.AppConfig{Env: config.EnvUAT}, Database: config.DatabaseConfig{Name: "tidewise_local"}},
		{App: config.AppConfig{Env: config.EnvProd}, Database: config.DatabaseConfig{Name: "tidewise_prod"}},
		{App: config.AppConfig{Env: config.EnvLocal}, Database: config.DatabaseConfig{Name: "shared_local"}},
	} {
		if err := validateAllianceEconomyLocalTarget(invalid); err == nil {
			t.Fatalf("validateAllianceEconomyLocalTarget(%+v) error = nil", invalid)
		}
	}
}
