package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCommandOptionsRejectsRetiredApplyScopes(t *testing.T) {
	for _, retired := range []string{
		"industry-chain-master",
		"industry-chain-membership",
		"industry-chain-topology",
		"industry-chain-physical-constraint",
		"industry-chain-sector-mapping",
	} {
		if _, err := validateCommandOptions(commandOptions{applyScope: retired}); err == nil {
			t.Fatalf("validateCommandOptions(%q) error = nil", retired)
		}
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
