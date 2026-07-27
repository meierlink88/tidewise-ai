package architecture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type industryRelationshipBuildAudit struct {
	SchemaVersion     string                        `json:"schema_version"`
	PackageVersion    string                        `json:"package_version"`
	BuildStatus       string                        `json:"build_status"`
	ReviewStatus      string                        `json:"review_status"`
	ApprovalBasis     string                        `json:"approval_basis"`
	ImportPackage     buildAuditImportPackage       `json:"import_package"`
	RelationSpec      buildAuditRelationSpec        `json:"relation_spec"`
	InputRegistries   map[string]buildAuditRegistry `json:"input_registries"`
	RelationDecisions map[string]buildAuditDecision `json:"relation_decisions"`
	EntityDecisions   map[string]buildAuditDecision `json:"entity_decisions"`
	SemanticReview    struct {
		WarningCount         int    `json:"warning_count"`
		ReviewedWarningCount int    `json:"reviewed_warning_count"`
		PendingWarningCount  int    `json:"pending_warning_count"`
		ErrorCount           int    `json:"error_count"`
		Status               string `json:"status"`
	} `json:"semantic_review"`
	OutputCounts          map[string]int `json:"output_counts"`
	DatabaseWriteAtFreeze struct {
		Executed              bool   `json:"executed"`
		Status                string `json:"status"`
		RuntimeRecordContract string `json:"runtime_record_contract"`
	} `json:"database_write_at_freeze"`
}

type buildAuditImportPackage struct {
	Path                  string `json:"path"`
	PackageSHA256         string `json:"package_sha256"`
	ManifestSHA256        string `json:"manifest_sha256"`
	SourceInventorySHA256 string `json:"source_inventory_sha256"`
}

type buildAuditRelationSpec struct {
	Version string `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
}

type buildAuditRegistry struct {
	SchemaVersion         string `json:"schema_version"`
	Path                  string `json:"path"`
	SHA256                string `json:"sha256"`
	InputCount            int    `json:"input_count"`
	DeduplicatedCount     int    `json:"deduplicated_count"`
	DuplicateRemovedCount int    `json:"duplicate_removed_count"`
	RejectedCount         int    `json:"rejected_count"`
	PendingCount          int    `json:"pending_count"`
}

type buildAuditDecision struct {
	OutputCountKey    string `json:"output_count_key"`
	CandidateCount    int    `json:"candidate_count"`
	DeduplicatedCount int    `json:"deduplicated_count"`
	ApprovedCount     int    `json:"approved_count"`
	RejectedCount     int    `json:"rejected_count"`
	PendingCount      int    `json:"pending_count"`
}

func TestIndustryRelationshipBuildAuditPinsFrozenInputsAndDecisionClosure(t *testing.T) {
	root := repositoryRoot()
	auditPath := filepath.Join(
		root,
		"analyse-data-service",
		"backend",
		"data",
		"industry_relationships",
		"audit",
		"2026-07-27-v1",
		"relationship_build_manifest.json",
	)
	var audit industryRelationshipBuildAudit
	decodeJSONContract(t, auditPath, &audit)
	if audit.SchemaVersion != "entity_relationship_build_manifest_v1" ||
		audit.PackageVersion != "2026-07-27-v1" ||
		audit.BuildStatus != "approved" ||
		audit.ReviewStatus != "approved" ||
		audit.ApprovalBasis != "user_explicit_delegated_review" {
		t.Fatalf("build audit is not the approved V1 contract: %#v", audit)
	}

	packageRoot := filepath.Join(
		root,
		"analyse-data-service",
		"backend",
		"data",
		"industry_relationships",
		"2026-07-27-v1",
	)
	manifestBytes := readContractBytes(t, filepath.Join(packageRoot, "manifest.json"))
	sourceInventoryBytes := readContractBytes(t, filepath.Join(packageRoot, "source_inventory.json"))
	var importManifest struct {
		PackageSHA256 string         `json:"package_sha256"`
		PackageCounts map[string]int `json:"package_counts"`
		RelationSpec  struct {
			Version    string `json:"version"`
			OriginPath string `json:"origin_path"`
			SHA256     string `json:"sha256"`
		} `json:"relation_spec"`
		SourceInventory struct {
			SHA256 string `json:"sha256"`
		} `json:"source_inventory"`
	}
	if err := json.Unmarshal(manifestBytes, &importManifest); err != nil {
		t.Fatalf("decode Industry relationship import manifest: %v", err)
	}
	if audit.ImportPackage.PackageSHA256 != importManifest.PackageSHA256 ||
		audit.ImportPackage.ManifestSHA256 != sha256Hex(manifestBytes) ||
		audit.ImportPackage.SourceInventorySHA256 != sha256Hex(sourceInventoryBytes) ||
		audit.ImportPackage.SourceInventorySHA256 != importManifest.SourceInventory.SHA256 {
		t.Fatal("build audit does not pin the exact import package, manifest and source inventory")
	}
	if audit.RelationSpec.Version != importManifest.RelationSpec.Version ||
		audit.RelationSpec.Path != importManifest.RelationSpec.OriginPath ||
		audit.RelationSpec.SHA256 != importManifest.RelationSpec.SHA256 {
		t.Fatal("build audit relation Spec does not match the import manifest")
	}
	if !reflect.DeepEqual(audit.OutputCounts, importManifest.PackageCounts) {
		t.Fatal("build audit output counts do not match the import package")
	}

	var sourceInventory struct {
		Sources []struct {
			SourceKind string `json:"source_kind"`
			Path       string `json:"path"`
			SHA256     string `json:"sha256"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(sourceInventoryBytes, &sourceInventory); err != nil {
		t.Fatalf("decode Industry relationship source inventory: %v", err)
	}
	inventoryByKind := make(map[string]struct {
		Path   string
		SHA256 string
	}, len(sourceInventory.Sources))
	for _, source := range sourceInventory.Sources {
		inventoryByKind[source.SourceKind] = struct {
			Path   string
			SHA256 string
		}{Path: source.Path, SHA256: source.SHA256}
	}
	registryKinds := map[string]struct {
		SourceKind    string
		SchemaVersion string
		Count         int
	}{
		"industry_chain": {"industry_chain_registry", "approved_industry_chain_registry_v1", 708},
		"industry":       {"industry_registry", "canonical_industry_registry_v1", 512},
		"concept":        {"concept_registry", "canonical_concept_registry_v1", 194},
		"chain_node":     {"chain_node_registry", "canonical_node_registry_v1", 588},
	}
	for name, expected := range registryKinds {
		registry, ok := audit.InputRegistries[name]
		if !ok {
			t.Fatalf("build audit is missing %s registry metrics", name)
		}
		source := inventoryByKind[expected.SourceKind]
		if registry.SchemaVersion != expected.SchemaVersion ||
			registry.Path != source.Path ||
			registry.SHA256 != source.SHA256 ||
			registry.InputCount != expected.Count ||
			registry.DeduplicatedCount != expected.Count ||
			registry.DuplicateRemovedCount != 0 ||
			registry.RejectedCount != 0 ||
			registry.PendingCount != 0 {
			t.Fatalf("build audit %s registry metrics are inconsistent: %#v", name, registry)
		}
	}

	requiredDecisions := map[string]string{
		"mapped_to_industry":         "industry_chain_industry_relations",
		"mapped_to_concept":          "industry_chain_concept_relations",
		"has_node":                   "industry_chain_node_memberships",
		"chain_graph_edge":           "industry_chain_graph_edges",
		"global_chain_node_relation": "global_chain_node_relations",
	}
	for name, outputCountKey := range requiredDecisions {
		decision, ok := audit.RelationDecisions[name]
		if !ok {
			t.Fatalf("build audit is missing %s decision metrics", name)
		}
		if decision.OutputCountKey != outputCountKey ||
			decision.CandidateCount <= 0 ||
			decision.DeduplicatedCount <= 0 ||
			decision.DeduplicatedCount > decision.CandidateCount ||
			decision.ApprovedCount != importManifest.PackageCounts[outputCountKey] ||
			decision.RejectedCount < 0 ||
			decision.PendingCount != 0 ||
			decision.ApprovedCount <= 0 {
			t.Fatalf("build audit %s decision metrics are inconsistent: %#v", name, decision)
		}
	}
	additions := audit.EntityDecisions["chain_node_additions"]
	if additions.CandidateCount <= 0 ||
		additions.DeduplicatedCount != additions.CandidateCount ||
		additions.ApprovedCount != importManifest.PackageCounts[additions.OutputCountKey] ||
		additions.RejectedCount != 0 ||
		additions.PendingCount != 0 {
		t.Fatalf("build audit Chain Node addition metrics are inconsistent: %#v", additions)
	}
	if audit.SemanticReview.WarningCount != audit.SemanticReview.ReviewedWarningCount ||
		audit.SemanticReview.PendingWarningCount != 0 ||
		audit.SemanticReview.ErrorCount != 0 ||
		audit.SemanticReview.Status != "passed" {
		t.Fatalf("build audit semantic review is not closed: %#v", audit.SemanticReview)
	}
	if audit.DatabaseWriteAtFreeze.Executed ||
		audit.DatabaseWriteAtFreeze.Status != "not_written_at_package_freeze" ||
		audit.DatabaseWriteAtFreeze.RuntimeRecordContract != "industry_relationship_import_receipts" {
		t.Fatalf("build audit database-write semantics are invalid: %#v", audit.DatabaseWriteAtFreeze)
	}
}

func decodeJSONContract(t *testing.T, path string, target any) {
	t.Helper()
	if err := json.Unmarshal(readContractBytes(t, path), target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func readContractBytes(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
