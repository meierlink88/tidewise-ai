package industryrelationshipimport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDirectoryVerifiesManifestPayloadAndPackageHashes(t *testing.T) {
	dir := t.TempDir()
	pkg := validTestPackage(t)
	spec := []byte("# relationship spec\n")
	if err := os.WriteFile(filepath.Join(dir, "relation_spec.md"), spec, 0o600); err != nil {
		t.Fatal(err)
	}
	inventory := SourceInventory{
		SchemaVersion: "industry_relationship_source_inventory_v1",
		DatabaseSource: SourceDatabaseDescriptor{
			Kind: "local_postgresql", Environment: "local",
			SnapshotAt: time.Date(2026, 7, 27, 4, 51, 54, 0, time.UTC),
		},
		EvidenceCutoffAt:    time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
		RelationSpecVersion: RelationSpecVersion,
		ReviewMode:          ApprovalBasis,
		AllowedRelationTypes: []string{
			"depends_on", "has_node", "input_to", "is_component_of",
			"is_subcategory_of", "mapped_to_concept", "mapped_to_industry",
		},
	}
	for _, kind := range []string{
		"industry_chain_registry",
		"industry_registry",
		"concept_registry",
		"chain_node_registry",
		"industry_mapping_decisions",
		"concept_mapping_decisions",
		"base_global_chain_node_relations",
		"concept_dispositions",
		"node_dispositions",
		"topology_review_contract",
		"relationship_build_execution_prompt",
	} {
		sourcePath := "outputs/test/" + kind + ".json"
		inventory.Sources = append(inventory.Sources, SourceInventoryEntry{
			SourceKind: kind, Path: sourcePath, SHA256: hashHex([]byte(sourcePath)),
		})
	}
	for index := 1; index <= 18; index++ {
		sourcePath := fmt.Sprintf("outputs/test/topology-m2-%02d.json", index)
		inventory.Sources = append(inventory.Sources, SourceInventoryEntry{
			SourceKind: "topology_work_item", Path: sourcePath, SHA256: hashHex([]byte(sourcePath)),
		})
	}
	sourceInventory, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	sourceInventory = append(sourceInventory, '\n')
	if err := os.WriteFile(filepath.Join(dir, "source_inventory.json"), sourceInventory, 0o600); err != nil {
		t.Fatal(err)
	}

	payloads := map[string]any{
		"industry_chains":                   pkg.IndustryChains,
		"chain_node_additions":              pkg.ChainNodeAdditions,
		"industry_chain_industry_relations": pkg.IndustryMappings,
		"industry_chain_concept_relations":  pkg.ConceptMappings,
		"industry_chain_node_memberships":   pkg.Memberships,
		"industry_chain_graph_edges":        pkg.GraphEdges,
		"global_chain_node_relations":       pkg.GlobalRelations,
		"relationship_evidence":             pkg.Evidence,
		"concept_dispositions":              pkg.ConceptDispositions,
		"node_dispositions":                 pkg.NodeDispositions,
		"unmapped_relation_candidates":      pkg.UnmappedCandidates,
		"relationship_validation_report":    pkg.ValidationReport,
	}
	files := make(map[string]FileDescriptor, len(payloads))
	for _, name := range RequiredFiles {
		content, err := json.MarshalIndent(payloads[name], "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		content = append(content, '\n')
		fileName := name + ".json"
		if err := os.WriteFile(filepath.Join(dir, fileName), content, 0o600); err != nil {
			t.Fatal(err)
		}
		count := 1
		if name != "relationship_validation_report" {
			count = rawPayloadCount(t, content)
		}
		files[name] = FileDescriptor{Path: fileName, SHA256: hashHex(content), Count: &count}
	}
	manifest := Manifest{
		SchemaVersion: ManifestSchemaVersion, PackageVersion: "test-v1",
		PackageStatus: PackageStatusApproved, ApprovalBasis: ApprovalBasis,
		GeneratedAt:        time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
		SourceSnapshotDate: "2026-07-27",
		RelationSpec: RelationSpecDescriptor{
			Version: RelationSpecVersion, Path: "relation_spec.md",
			OriginPath: "research/spec.md", SHA256: hashHex(spec),
		},
		SourceInventory: FileDescriptor{
			Path: "source_inventory.json", SHA256: hashHex(sourceInventory),
		},
		Files: files, ProjectionFiles: map[string]FileDescriptor{},
		PackageCounts: countsMap(pkg.Counts()),
	}
	packageSHA, err := manifestPackageSHA(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.PackageSHA256 = packageSHA
	manifestContent, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestContent = append(manifestContent, '\n')
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifestContent, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadDirectory(dir, packageSHA)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Manifest.PackageSHA256 != packageSHA ||
		len(loaded.IndustryChains) != 708 ||
		len(loaded.Memberships) != 1416 {
		t.Fatalf("loaded package = sha %s, chains %d, memberships %d",
			loaded.Manifest.PackageSHA256, len(loaded.IndustryChains), len(loaded.Memberships))
	}

	manifest.GeneratedAt = manifest.GeneratedAt.Add(time.Hour)
	tamperedManifest, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	tamperedManifest = append(tamperedManifest, '\n')
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), tamperedManifest, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDirectory(dir, packageSHA); err == nil {
		t.Fatal("LoadDirectory accepted generated_at drift outside the package SHA")
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifestContent, 0o600); err != nil {
		t.Fatal(err)
	}

	tamperedPath := filepath.Join(dir, files["industry_chains"].Path)
	if err := os.WriteFile(tamperedPath, []byte("[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDirectory(dir, packageSHA); err == nil {
		t.Fatal("LoadDirectory accepted a tampered payload")
	}
}

func rawPayloadCount(t *testing.T, content []byte) int {
	t.Helper()
	var rows []json.RawMessage
	if err := json.Unmarshal(content, &rows); err != nil {
		t.Fatal(err)
	}
	return len(rows)
}
