package industryrelationshipimport

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	maxManifestBytes = 2 << 20
	maxPayloadBytes  = 256 << 20
)

var shaPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func LoadDirectory(directory, expectedPackageSHA string) (Package, error) {
	cleanDirectory, err := filepath.Abs(strings.TrimSpace(directory))
	if err != nil {
		return Package{}, fmt.Errorf("resolve package directory: %w", err)
	}
	manifestPath := filepath.Join(cleanDirectory, "manifest.json")
	manifestBytes, err := readLimitedFile(manifestPath, maxManifestBytes)
	if err != nil {
		return Package{}, fmt.Errorf("read relationship manifest: %w", err)
	}
	var manifest Manifest
	if err := decodeStrict(manifestBytes, &manifest); err != nil {
		return Package{}, fmt.Errorf("decode relationship manifest: %w", err)
	}
	if err := validateManifest(cleanDirectory, manifest, expectedPackageSHA); err != nil {
		return Package{}, err
	}

	pkg := Package{
		Directory:      cleanDirectory,
		Manifest:       manifest,
		ManifestSHA256: hashHex(manifestBytes),
	}
	loads := []struct {
		name string
		out  any
	}{
		{"industry_chains", &pkg.IndustryChains},
		{"chain_node_additions", &pkg.ChainNodeAdditions},
		{"industry_chain_industry_relations", &pkg.IndustryMappings},
		{"industry_chain_concept_relations", &pkg.ConceptMappings},
		{"industry_chain_node_memberships", &pkg.Memberships},
		{"industry_chain_graph_edges", &pkg.GraphEdges},
		{"global_chain_node_relations", &pkg.GlobalRelations},
		{"relationship_evidence", &pkg.Evidence},
		{"concept_dispositions", &pkg.ConceptDispositions},
		{"node_dispositions", &pkg.NodeDispositions},
		{"unmapped_relation_candidates", &pkg.UnmappedCandidates},
		{"relationship_validation_report", &pkg.ValidationReport},
	}
	for _, load := range loads {
		descriptor := manifest.Files[load.name]
		content, err := readPackageFile(cleanDirectory, descriptor)
		if err != nil {
			return Package{}, fmt.Errorf("%s: %w", load.name, err)
		}
		if err := decodeStrict(content, load.out); err != nil {
			return Package{}, fmt.Errorf("decode %s: %w", load.name, err)
		}
		if descriptor.Count == nil || *descriptor.Count != payloadCount(load.out) {
			return Package{}, fmt.Errorf(
				"%s count %d does not match manifest %d",
				load.name, payloadCount(load.out), descriptorCount(descriptor),
			)
		}
	}
	if err := pkg.Validate(); err != nil {
		return Package{}, err
	}
	return pkg, nil
}

func validateManifest(directory string, manifest Manifest, expectedPackageSHA string) error {
	if manifest.SchemaVersion != ManifestSchemaVersion {
		return fmt.Errorf("manifest schema_version %q is unsupported", manifest.SchemaVersion)
	}
	if manifest.PackageStatus != PackageStatusApproved || manifest.ApprovalBasis != ApprovalBasis {
		return errors.New("relationship package is not approved under the delegated-review contract")
	}
	if strings.TrimSpace(manifest.PackageVersion) == "" ||
		strings.TrimSpace(manifest.SourceSnapshotDate) == "" ||
		strings.TrimSpace(manifest.RelationSpec.OriginPath) == "" ||
		manifest.GeneratedAt.IsZero() {
		return errors.New("package_version, generated_at, source_snapshot_date and relation Spec origin_path are required")
	}
	expectedGeneratedAt, err := time.Parse("2006-01-02", manifest.SourceSnapshotDate)
	if err != nil || !manifest.GeneratedAt.Equal(expectedGeneratedAt) {
		return errors.New("generated_at must equal source_snapshot_date at 00:00:00Z")
	}
	if manifest.RelationSpec.Version != RelationSpecVersion {
		return fmt.Errorf("relation Spec version %q is unsupported", manifest.RelationSpec.Version)
	}
	if !shaPattern.MatchString(manifest.RelationSpec.SHA256) ||
		!shaPattern.MatchString(manifest.PackageSHA256) {
		return errors.New("manifest contains an invalid SHA-256")
	}
	if len(manifest.Files) != len(RequiredFiles) {
		return fmt.Errorf("manifest contains %d database files, want %d", len(manifest.Files), len(RequiredFiles))
	}
	for _, name := range RequiredFiles {
		descriptor, exists := manifest.Files[name]
		if !exists {
			return fmt.Errorf("manifest is missing database file %q", name)
		}
		if err := validateFileDescriptor(descriptor, true); err != nil {
			return fmt.Errorf("manifest file %q: %w", name, err)
		}
	}
	for name, descriptor := range manifest.ProjectionFiles {
		if err := validateFileDescriptor(descriptor, false); err != nil {
			return fmt.Errorf("projection file %q: %w", name, err)
		}
		content, err := readPackageFile(directory, descriptor)
		if err != nil {
			return fmt.Errorf("projection file %q: %w", name, err)
		}
		if hashHex(content) != descriptor.SHA256 {
			return fmt.Errorf("projection file %q SHA-256 mismatch", name)
		}
	}
	specDescriptor := FileDescriptor{Path: manifest.RelationSpec.Path, SHA256: manifest.RelationSpec.SHA256}
	if err := validateFileDescriptor(specDescriptor, false); err != nil {
		return fmt.Errorf("relation Spec: %w", err)
	}
	spec, err := readPackageFile(directory, specDescriptor)
	if err != nil {
		return fmt.Errorf("read relation Spec: %w", err)
	}
	if hashHex(spec) != manifest.RelationSpec.SHA256 {
		return errors.New("relation Spec SHA-256 mismatch")
	}
	if manifest.SourceInventory.Path != "source_inventory.json" {
		return errors.New("source inventory path must be source_inventory.json")
	}
	if err := validateFileDescriptor(manifest.SourceInventory, false); err != nil {
		return fmt.Errorf("source inventory: %w", err)
	}
	sourceInventory, err := readPackageFile(directory, manifest.SourceInventory)
	if err != nil {
		return fmt.Errorf("read source inventory: %w", err)
	}
	if hashHex(sourceInventory) != manifest.SourceInventory.SHA256 {
		return errors.New("source inventory SHA-256 mismatch")
	}
	var inventory SourceInventory
	if err := decodeStrict(sourceInventory, &inventory); err != nil {
		return fmt.Errorf("decode source inventory: %w", err)
	}
	if err := validateSourceInventory(inventory); err != nil {
		return err
	}
	calculated, err := manifestPackageSHA(manifest)
	if err != nil {
		return err
	}
	if calculated != manifest.PackageSHA256 {
		return fmt.Errorf("package SHA-256 mismatch: calculated %s", calculated)
	}
	expectedPackageSHA = strings.TrimSpace(expectedPackageSHA)
	if expectedPackageSHA != "" && expectedPackageSHA != manifest.PackageSHA256 {
		return fmt.Errorf("package SHA-256 %s does not match expected %s", manifest.PackageSHA256, expectedPackageSHA)
	}
	return nil
}

func validateSourceInventory(inventory SourceInventory) error {
	if inventory.SchemaVersion != "industry_relationship_source_inventory_v1" ||
		inventory.DatabaseSource.Kind != "local_postgresql" ||
		inventory.DatabaseSource.Environment != "local" ||
		inventory.DatabaseSource.SnapshotAt.IsZero() ||
		inventory.EvidenceCutoffAt.IsZero() ||
		inventory.RelationSpecVersion != RelationSpecVersion ||
		inventory.ReviewMode != ApprovalBasis {
		return errors.New("source inventory does not match the frozen delegated-review contract")
	}
	expectedRelationTypes := []string{
		"depends_on",
		"has_node",
		"input_to",
		"is_component_of",
		"is_subcategory_of",
		"mapped_to_concept",
		"mapped_to_industry",
	}
	if !reflect.DeepEqual(inventory.AllowedRelationTypes, expectedRelationTypes) {
		return errors.New("source inventory relation types do not match the V1 contract")
	}
	requiredKinds := map[string]int{
		"industry_chain_registry":             1,
		"industry_registry":                   1,
		"concept_registry":                    1,
		"chain_node_registry":                 1,
		"industry_mapping_decisions":          1,
		"concept_mapping_decisions":           1,
		"base_global_chain_node_relations":    1,
		"concept_dispositions":                1,
		"node_dispositions":                   1,
		"topology_review_contract":            1,
		"relationship_build_execution_prompt": 1,
		"topology_work_item":                  18,
	}
	gotKinds := make(map[string]int, len(requiredKinds))
	paths := make(map[string]struct{}, len(inventory.Sources))
	for index, source := range inventory.Sources {
		if _, duplicate := paths[source.Path]; duplicate {
			return fmt.Errorf("source inventory duplicates path %s", source.Path)
		}
		paths[source.Path] = struct{}{}
		if source.SourceKind == "" ||
			(!strings.HasPrefix(source.Path, "outputs/") &&
				!strings.HasPrefix(source.Path, "research/")) ||
			!shaPattern.MatchString(source.SHA256) {
			return fmt.Errorf("source inventory entry %d is invalid", index)
		}
		gotKinds[source.SourceKind]++
	}
	for kind, want := range requiredKinds {
		if gotKinds[kind] != want {
			return fmt.Errorf("source inventory kind %s count=%d, want %d", kind, gotKinds[kind], want)
		}
	}
	return nil
}

func validateFileDescriptor(descriptor FileDescriptor, requireCount bool) error {
	if !safeRelativePath(descriptor.Path) {
		return fmt.Errorf("path %q is not a safe package-relative path", descriptor.Path)
	}
	if !shaPattern.MatchString(descriptor.SHA256) {
		return errors.New("sha256 must be 64 lowercase hexadecimal characters")
	}
	if requireCount && (descriptor.Count == nil || *descriptor.Count < 0) {
		return errors.New("count must be non-negative")
	}
	return nil
}

func descriptorCount(descriptor FileDescriptor) int {
	if descriptor.Count == nil {
		return -1
	}
	return *descriptor.Count
}

func safeRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) {
		return false
	}
	clean := filepath.Clean(value)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func readPackageFile(directory string, descriptor FileDescriptor) ([]byte, error) {
	target := filepath.Join(directory, filepath.Clean(descriptor.Path))
	content, err := readLimitedFile(target, maxPayloadBytes)
	if err != nil {
		return nil, err
	}
	if hashHex(content) != descriptor.SHA256 {
		return nil, errors.New("SHA-256 mismatch")
	}
	return content, nil
}

func readLimitedFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	return content, nil
}

func decodeStrict(content []byte, output any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("unexpected trailing token %v", token)
		}
		return err
	}
	return nil
}

func payloadCount(value any) int {
	switch target := value.(type) {
	case *[]IndustryChain:
		return len(*target)
	case *[]ChainNodeAddition:
		return len(*target)
	case *[]EntityMapping:
		return len(*target)
	case *[]Membership:
		return len(*target)
	case *[]GraphEdge:
		return len(*target)
	case *[]GlobalChainNodeRelation:
		return len(*target)
	case *[]json.RawMessage:
		return len(*target)
	case *ValidationReport:
		return 1
	default:
		panic(fmt.Sprintf("unsupported relationship package payload %T", value))
	}
}

func manifestPackageSHA(manifest Manifest) (string, error) {
	core := map[string]any{
		"schema_version":       manifest.SchemaVersion,
		"package_version":      manifest.PackageVersion,
		"package_status":       manifest.PackageStatus,
		"approval_basis":       manifest.ApprovalBasis,
		"source_snapshot_date": manifest.SourceSnapshotDate,
		"relation_spec":        manifest.RelationSpec,
		"source_inventory":     manifest.SourceInventory,
		"files":                manifest.Files,
		"projection_files":     manifest.ProjectionFiles,
		"package_counts":       manifest.PackageCounts,
	}
	encoded, err := json.Marshal(core)
	if err != nil {
		return "", fmt.Errorf("encode package SHA input: %w", err)
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		return "", fmt.Errorf("normalize package SHA input: %w", err)
	}
	canonical, err := canonicalJSON(generic)
	if err != nil {
		return "", fmt.Errorf("canonicalize package SHA input: %w", err)
	}
	return hashHex(canonical), nil
}

func canonicalJSON(value any) ([]byte, error) {
	switch typed := value.(type) {
	case nil, bool, string, float64:
		return json.Marshal(typed)
	case []any:
		var output bytes.Buffer
		output.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				output.WriteByte(',')
			}
			encoded, err := canonicalJSON(item)
			if err != nil {
				return nil, err
			}
			output.Write(encoded)
		}
		output.WriteByte(']')
		return output.Bytes(), nil
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var output bytes.Buffer
		output.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				output.WriteByte(',')
			}
			encodedKey, _ := json.Marshal(key)
			encodedValue, err := canonicalJSON(typed[key])
			if err != nil {
				return nil, err
			}
			output.Write(encodedKey)
			output.WriteByte(':')
			output.Write(encodedValue)
		}
		output.WriteByte('}')
		return output.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported canonical JSON value %T", value)
	}
}

func hashHex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
