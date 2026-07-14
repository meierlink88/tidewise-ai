package main

import (
	"os"
	"path/filepath"
	"testing"

	entityseed "github.com/meierlink88/tidewise-ai/backend/internal/apps/entityfoundation/seed"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
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

func TestLoadRelationDryRunManifestRejectsPhysicalConstraints(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relations.json")
	content := []byte(`{"relations":[],"physical_constraints":[{"id":"constraint","chain_node_entity_id":"node","constraint_type":"process_yield","description":"具体工艺良率限制合格产出","evidence_note":"强证据","provenance":"review","status":"active","verified_at":"2026-07-14T00:00:00Z"}]}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRelationDryRunManifest(path); err == nil {
		t.Fatal("loadRelationDryRunManifest() error = nil")
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
