package seed

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	repoids "github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

const frozenChainNodeRelationManifestRelativePath = "openspec/changes/rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-r0/approved-candidate-manifest.json"
const frozenChainNodeRelationArchiveRelativePath = "openspec/changes/archive/2026-07-15-rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-r0/approved-candidate-manifest.json"
const frozenChainNodeRelationManifestFileSHA256 = "0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268"
const frozenChainNodeRelationManifestSHA256 = "b578e957df6e6249f745f2661f11a2d03c73434dab85fe8e2fb35f33bf14f2d9"
const frozenChainNodeRelationContentSHA256 = "e5adb1feb2abcda5bbeacd6e01baf68113417aba14c1dbf732b2dfa4528be67a"
const frozenAdditiveChainNodeRelationManifestRelativePath = "openspec/changes/archive/2026-07-15-rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-usable-map-r0/additive-final-candidate-manifest.json"
const frozenAdditiveChainNodeRelationManifestFileSHA256 = "9578cd18e3b629b1e8df11d517c94ad25597bb47826511217812e1e7794c2ed8"
const frozenAdditiveChainNodeRelationManifestSHA256 = "5a533399a77c430e9067bac5ff509362c8168965a198801d665c40723cee4487"
const frozenCombinedChainNodeRelationTupleSHA256 = "22809290b844104c140368a303d4e09336c9855f291b7ee624233150ca79b944"
const frozenChainNodeBaselineFileSHA256 = "a5475719cd874360116ba7e226d048c4ae9bc06006e1b4c23515198616120edb"
const frozenChainNodeIdentityMD5 = "d6b53dce56fb5ca72ec77eef816f0a4b"
const frozenChainNodeProfileMD5 = "2876324fb6bffa41967812702c6bc038"

type frozenChainNodeRelationManifest struct {
	ArtifactType      string                               `json:"artifact_type"`
	ArtifactVersion   string                               `json:"artifact_version"`
	ReviewState       string                               `json:"review_state"`
	ReadyForWrite     bool                                 `json:"ready_for_write"`
	WriteAuthorized   bool                                 `json:"write_authorized"`
	BaselineNodeCount int                                  `json:"baseline_node_count"`
	RelationCount     int                                  `json:"relation_count"`
	ByRelationType    map[domain.ChainNodeRelationType]int `json:"by_relation_type"`
	Relations         []frozenChainNodeRelation            `json:"relations"`
	ManifestSHA256    string                               `json:"manifest_sha256"`
}

type frozenChainNodeRelation struct {
	domain.ChainNodeRelation
	FromName       string                          `json:"from_name"`
	ToName         string                          `json:"to_name"`
	Sources        []frozenChainNodeRelationSource `json:"sources"`
	Counterexample string                          `json:"counterexample"`
	Confidence     string                          `json:"confidence"`
}

type frozenChainNodeRelationSource struct {
	Title        string  `json:"title"`
	URL          *string `json:"url"`
	PublishedAt  *string `json:"published_at"`
	AccessedAt   string  `json:"accessed_at"`
	Supports     string  `json:"supports"`
	ArtifactPath string  `json:"artifact_path"`
	SHA256       string  `json:"sha256"`
}

type frozenAdditiveChainNodeRelationManifest struct {
	ArtifactType                   string                               `json:"artifact_type"`
	ArtifactVersion                int                                  `json:"artifact_version"`
	ReviewState                    string                               `json:"review_state"`
	ReadyForWrite                  bool                                 `json:"ready_for_write"`
	WriteAuthorized                bool                                 `json:"write_authorized"`
	BaselineNodeCount              int                                  `json:"baseline_node_count"`
	SupersedesR0Checkpoint         string                               `json:"supersedes_r0_checkpoint"`
	ExistingBaselineReference      frozenAcceptedRelationReference      `json:"existing_baseline_reference"`
	ReviewedOriginalCandidateCount int                                  `json:"reviewed_original_candidate_count"`
	ExistingRelationCount          int                                  `json:"existing_relation_count"`
	ExistingByRelationType         map[domain.ChainNodeRelationType]int `json:"existing_by_relation_type"`
	NewRelationCount               int                                  `json:"new_relation_count"`
	NewByRelationType              map[domain.ChainNodeRelationType]int `json:"new_by_relation_type"`
	NewByEvidenceTier              map[string]int                       `json:"new_by_evidence_tier"`
	TotalRelationCount             int                                  `json:"total_relation_count"`
	TotalByRelationType            map[domain.ChainNodeRelationType]int `json:"total_by_relation_type"`
	NewRelationsSHA256             string                               `json:"new_relations_sha256"`
	TotalTupleSHA256               string                               `json:"total_tuple_sha256"`
	NewRelations                   []json.RawMessage                    `json:"new_relations"`
}

type frozenAcceptedRelationReference struct {
	Path                 string `json:"path"`
	FileSHA256           string `json:"file_sha256"`
	ContentSHA256        string `json:"content_sha256"`
	ManifestSHA256       string `json:"manifest_sha256"`
	PreservedContentDiff int    `json:"preserved_content_diff"`
}

type frozenChainNodeBaseline struct {
	ArtifactType            string                          `json:"artifact_type"`
	ArtifactVersion         string                          `json:"artifact_version"`
	Environment             string                          `json:"environment"`
	SourceOfTruth           string                          `json:"source_of_truth"`
	FrozenAt                time.Time                       `json:"frozen_at"`
	Count                   int                             `json:"count"`
	IdentityMD5             string                          `json:"identity_md5"`
	ProfileMD5              string                          `json:"profile_md5"`
	ArtifactIdentityRowDiff int                             `json:"artifact_identity_row_diff"`
	SourceArtifacts         []frozenChainNodeSourceArtifact `json:"source_artifacts"`
	Nodes                   []frozenChainNodeIdentity       `json:"nodes"`
}

type frozenChainNodeSourceArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type frozenChainNodeIdentity struct {
	EntityID      string        `json:"entity_id"`
	EntityKey     string        `json:"entity_key"`
	Name          string        `json:"name"`
	CanonicalName string        `json:"canonical_name"`
	Status        domain.Status `json:"status"`
}

type ChainNodeRelationDataPreflightReport struct {
	DatabaseName         string `json:"database_name"`
	ServerVersion        string `json:"server_version"`
	GooseVersion         int    `json:"goose_version"`
	ActiveChainNodes     int    `json:"active_chain_nodes"`
	ChainNodeProfiles    int    `json:"chain_node_profiles"`
	ExternalIdentifiers  int    `json:"external_identifiers"`
	EntityEdges          int    `json:"entity_edges"`
	ExistingRelations    int    `json:"existing_relations"`
	SubcategoryRelations int    `json:"subcategory_relations"`
	ComponentRelations   int    `json:"component_relations"`
	InputRelations       int    `json:"input_relations"`
	DependsRelations     int    `json:"depends_relations"`
	ExistingConstraints  int    `json:"existing_constraints"`
	SchemaValid          bool   `json:"schema_valid"`
}

func LoadFrozenChainNodeRelationManifest(path string) (ChainNodeRelationManifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ChainNodeRelationManifest{}, err
	}
	if err := validateFrozenChainNodeRelationFileIdentity(path, content); err != nil {
		return ChainNodeRelationManifest{}, err
	}
	var frozen frozenChainNodeRelationManifest
	if err := decodeFrozenChainNodeJSON(content, &frozen); err != nil {
		return ChainNodeRelationManifest{}, fmt.Errorf("decode frozen chain node relation manifest: %w", err)
	}
	endpoints, err := loadFrozenChainNodeEndpointBaseline()
	if err != nil {
		return ChainNodeRelationManifest{}, err
	}
	manifest := ChainNodeRelationManifest{Relations: make([]domain.ChainNodeRelation, 0, len(frozen.Relations))}
	for _, relation := range frozen.Relations {
		manifest.Relations = append(manifest.Relations, relation.ChainNodeRelation)
	}
	if err := validateFrozenChainNodeRelationMetadata(frozen, manifest); err != nil {
		return ChainNodeRelationManifest{}, err
	}
	if err := validateFrozenChainNodeRelationEndpoints(manifest.Relations, endpoints); err != nil {
		return ChainNodeRelationManifest{}, err
	}
	return manifest, nil
}

func LoadFrozenAdditiveChainNodeRelationManifest(path string) (ChainNodeRelationManifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ChainNodeRelationManifest{}, err
	}
	if err := validateFrozenAdditiveChainNodeRelationFileIdentity(path, content); err != nil {
		return ChainNodeRelationManifest{}, err
	}
	var frozen frozenAdditiveChainNodeRelationManifest
	if err := decodeFrozenChainNodeJSON(content, &frozen); err != nil {
		return ChainNodeRelationManifest{}, fmt.Errorf("decode frozen additive chain node relation manifest: %w", err)
	}
	acceptedPath, err := frozenChainNodeRelationArtifactPath("approved-candidate-manifest.json")
	if err != nil {
		return ChainNodeRelationManifest{}, err
	}
	accepted, err := LoadFrozenChainNodeRelationManifest(acceptedPath)
	if err != nil {
		return ChainNodeRelationManifest{}, err
	}
	additive := ChainNodeRelationManifest{Relations: make([]domain.ChainNodeRelation, 0, len(frozen.NewRelations))}
	for _, raw := range frozen.NewRelations {
		var relation domain.ChainNodeRelation
		if err := json.Unmarshal(raw, &relation); err != nil {
			return ChainNodeRelationManifest{}, fmt.Errorf("decode frozen additive chain node relation: %w", err)
		}
		additive.Relations = append(additive.Relations, relation)
	}
	combined := ChainNodeRelationManifest{Relations: make([]domain.ChainNodeRelation, 0, len(accepted.Relations)+len(additive.Relations))}
	combined.Relations = append(combined.Relations, accepted.Relations...)
	combined.Relations = append(combined.Relations, additive.Relations...)
	if err := validateFrozenAdditiveChainNodeRelationMetadata(frozen, accepted, additive, combined); err != nil {
		return ChainNodeRelationManifest{}, err
	}
	endpoints, err := loadFrozenChainNodeEndpointBaseline()
	if err != nil {
		return ChainNodeRelationManifest{}, err
	}
	if err := validateFrozenChainNodeRelationEndpoints(combined.Relations, endpoints); err != nil {
		return ChainNodeRelationManifest{}, err
	}
	if err := validateFrozenChainNodeRelationIDs(combined.Relations); err != nil {
		return ChainNodeRelationManifest{}, err
	}
	if err := domain.ValidateChainNodeRelationBatch(combined.Relations); err != nil {
		return ChainNodeRelationManifest{}, err
	}
	return combined, nil
}

func validateFrozenAdditiveChainNodeRelationFileIdentity(path string, content []byte) error {
	expected, err := frozenAdditiveChainNodeRelationArtifactPath()
	if err != nil {
		return err
	}
	actual, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if filepath.Clean(actual) != filepath.Clean(expected) {
		return fmt.Errorf("additive chain node relation manifest path does not match frozen artifact")
	}
	if fmt.Sprintf("%x", sha256.Sum256(content)) != frozenAdditiveChainNodeRelationManifestFileSHA256 {
		return fmt.Errorf("additive chain node relation manifest file checksum does not match frozen artifact")
	}
	return nil
}

func validateFrozenAdditiveChainNodeRelationMetadata(frozen frozenAdditiveChainNodeRelationManifest, accepted, additive, combined ChainNodeRelationManifest) error {
	wantAccepted := map[domain.ChainNodeRelationType]int{
		domain.ChainNodeRelationSubcategoryOf: 95,
		domain.ChainNodeRelationComponentOf:   1,
		domain.ChainNodeRelationInputTo:       3,
		domain.ChainNodeRelationDependsOn:     1,
	}
	wantAdditive := map[domain.ChainNodeRelationType]int{
		domain.ChainNodeRelationSubcategoryOf: 13,
		domain.ChainNodeRelationComponentOf:   2,
		domain.ChainNodeRelationInputTo:       90,
		domain.ChainNodeRelationDependsOn:     7,
	}
	wantCombined := map[domain.ChainNodeRelationType]int{
		domain.ChainNodeRelationSubcategoryOf: 108,
		domain.ChainNodeRelationComponentOf:   3,
		domain.ChainNodeRelationInputTo:       93,
		domain.ChainNodeRelationDependsOn:     8,
	}
	if frozen.ArtifactType != "chain_node_usable_map_additive_manifest" || frozen.ArtifactVersion != 2 || frozen.ReviewState != "ready_for_independent_r0_review" || frozen.ReadyForWrite || frozen.WriteAuthorized || frozen.BaselineNodeCount != 842 || frozen.SupersedesR0Checkpoint != "4b7b80cb639f9991cdb23c3a50007563b1aba4a3" || frozen.ReviewedOriginalCandidateCount != 156 {
		return fmt.Errorf("additive relation manifest review metadata drifted")
	}
	reference := frozen.ExistingBaselineReference
	if reference.Path != frozenChainNodeRelationManifestRelativePath || reference.FileSHA256 != frozenChainNodeRelationManifestFileSHA256 || reference.ContentSHA256 != frozenChainNodeRelationContentSHA256 || reference.ManifestSHA256 != frozenChainNodeRelationManifestSHA256 || reference.PreservedContentDiff != 0 {
		return fmt.Errorf("additive relation accepted baseline reference drifted")
	}
	if frozen.ExistingRelationCount != 100 || len(accepted.Relations) != 100 || !equalFrozenChainNodeRelationCounts(frozen.ExistingByRelationType, wantAccepted) {
		return fmt.Errorf("additive relation accepted baseline requires 100=95/1/3/1")
	}
	if frozen.NewRelationCount != 112 || len(frozen.NewRelations) != 112 || len(additive.Relations) != 112 || !equalFrozenChainNodeRelationCounts(frozen.NewByRelationType, wantAdditive) || !equalFrozenEvidenceTierCounts(frozen.NewByEvidenceTier) {
		return fmt.Errorf("additive relation manifest requires 112=13/2/90/7 and tier 16/96")
	}
	if frozen.TotalRelationCount != 212 || len(combined.Relations) != 212 || !equalFrozenChainNodeRelationCounts(frozen.TotalByRelationType, wantCombined) || frozen.NewRelationsSHA256 != frozenAdditiveChainNodeRelationManifestSHA256 || frozen.TotalTupleSHA256 != frozenCombinedChainNodeRelationTupleSHA256 {
		return fmt.Errorf("combined relation manifest requires 212=108/3/93/8 and frozen hashes")
	}
	return nil
}

func equalFrozenChainNodeRelationCounts(got, want map[domain.ChainNodeRelationType]int) bool {
	if len(got) != len(want) {
		return false
	}
	for relationType, count := range want {
		if got[relationType] != count {
			return false
		}
	}
	return true
}

func equalFrozenEvidenceTierCounts(got map[string]int) bool {
	return len(got) == 2 && got["tier_1"] == 16 && got["tier_2"] == 96
}

func validateFrozenChainNodeRelationIDs(relations []domain.ChainNodeRelation) error {
	seen := make(map[string]struct{}, len(relations))
	for _, relation := range relations {
		if _, duplicate := seen[relation.ID]; duplicate {
			return fmt.Errorf("duplicate chain node relation id %q", relation.ID)
		}
		seen[relation.ID] = struct{}{}
	}
	return nil
}

func validateFrozenChainNodeRelationFileIdentity(path string, content []byte) error {
	expected, err := frozenChainNodeRelationArtifactPath("approved-candidate-manifest.json")
	if err != nil {
		return err
	}
	actual, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if filepath.Clean(actual) != filepath.Clean(expected) {
		return fmt.Errorf("chain node relation manifest path does not match frozen artifact")
	}
	if fmt.Sprintf("%x", sha256.Sum256(content)) != frozenChainNodeRelationManifestFileSHA256 {
		return fmt.Errorf("chain node relation manifest file checksum does not match frozen artifact")
	}
	return nil
}

func validateFrozenChainNodeRelationMetadata(frozen frozenChainNodeRelationManifest, manifest ChainNodeRelationManifest) error {
	if frozen.ArtifactType != "chain_node_relation_approved_candidate_manifest" || frozen.ArtifactVersion != "1.0.0" || frozen.ReviewState != "ready_for_human_freeze_review" || frozen.ReadyForWrite || frozen.WriteAuthorized {
		return fmt.Errorf("chain node relation manifest review metadata drifted")
	}
	if frozen.ManifestSHA256 != frozenChainNodeRelationManifestSHA256 {
		return fmt.Errorf("chain node relation manifest semantic checksum drifted")
	}
	if frozen.BaselineNodeCount != 842 || frozen.RelationCount != 100 || len(manifest.Relations) != 100 || len(manifest.PhysicalConstraints) != 0 {
		return fmt.Errorf("approved relation manifest requires 842 baseline nodes, 100 relations and zero physical constraints")
	}
	counts := map[domain.ChainNodeRelationType]int{}
	for index, relation := range manifest.Relations {
		counts[relation.RelationType]++
		identity := relation.FromChainNodeEntityID + "|" + string(relation.RelationType) + "|" + relation.ToChainNodeEntityID
		if relation.ID != repoids.NormalizeUUID("chain_node_relation", identity) {
			return fmt.Errorf("relation %q deterministic id mismatch", relation.ID)
		}
		if relation.Status != domain.StatusActive || relation.VerifiedAt.IsZero() {
			return fmt.Errorf("relation %q must be active and verified", relation.ID)
		}
		metadata := frozen.Relations[index]
		if strings.TrimSpace(metadata.FromName) == "" || strings.TrimSpace(metadata.ToName) == "" || strings.TrimSpace(metadata.Counterexample) == "" || metadata.Confidence != "high" || len(metadata.Sources) == 0 {
			return fmt.Errorf("relation %q review evidence metadata is incomplete", relation.ID)
		}
	}
	want := map[domain.ChainNodeRelationType]int{
		domain.ChainNodeRelationSubcategoryOf: 95,
		domain.ChainNodeRelationComponentOf:   1,
		domain.ChainNodeRelationInputTo:       3,
		domain.ChainNodeRelationDependsOn:     1,
	}
	for relationType, count := range want {
		if counts[relationType] != count || frozen.ByRelationType[relationType] != count {
			return fmt.Errorf("approved relation type counts do not match 95/1/3/1")
		}
	}
	if len(frozen.ByRelationType) != len(want) {
		return fmt.Errorf("approved relation type set drifted")
	}
	return domain.ValidateChainNodeRelationBatch(manifest.Relations)
}

func loadFrozenChainNodeEndpointBaseline() (map[string]struct{}, error) {
	path, err := frozenChainNodeRelationArtifactPath("chain-node-baseline.json")
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(content)) != frozenChainNodeBaselineFileSHA256 {
		return nil, fmt.Errorf("chain node baseline file checksum does not match frozen artifact")
	}
	var baseline frozenChainNodeBaseline
	if err := decodeFrozenChainNodeJSON(content, &baseline); err != nil {
		return nil, fmt.Errorf("decode frozen chain node baseline: %w", err)
	}
	if baseline.ArtifactType != "active_chain_node_frozen_baseline" || baseline.ArtifactVersion != "1.0.0" || baseline.Environment != "local" || baseline.SourceOfTruth != "PostgreSQL" || baseline.FrozenAt.IsZero() || baseline.Count != 842 || baseline.IdentityMD5 != frozenChainNodeIdentityMD5 || baseline.ProfileMD5 != frozenChainNodeProfileMD5 || baseline.ArtifactIdentityRowDiff != 0 || len(baseline.Nodes) != 842 {
		return nil, fmt.Errorf("chain node baseline metadata drifted")
	}
	endpoints := make(map[string]struct{}, len(baseline.Nodes))
	for _, node := range baseline.Nodes {
		if node.EntityID == "" || node.EntityKey == "" || node.Name == "" || node.CanonicalName == "" || node.Status != domain.StatusActive {
			return nil, fmt.Errorf("chain node baseline contains incomplete identity")
		}
		if _, duplicate := endpoints[node.EntityID]; duplicate {
			return nil, fmt.Errorf("chain node baseline contains duplicate endpoint %q", node.EntityID)
		}
		endpoints[node.EntityID] = struct{}{}
	}
	return endpoints, nil
}

func validateFrozenChainNodeRelationEndpoints(relations []domain.ChainNodeRelation, endpoints map[string]struct{}) error {
	if len(endpoints) != 842 {
		return fmt.Errorf("chain node endpoint baseline requires 842 identities")
	}
	for _, relation := range relations {
		for _, endpoint := range []string{relation.FromChainNodeEntityID, relation.ToChainNodeEntityID} {
			if _, ok := endpoints[endpoint]; !ok {
				return fmt.Errorf("relation %q endpoint %q is outside frozen 842 baseline", relation.ID, endpoint)
			}
		}
	}
	return nil
}

func frozenChainNodeRelationArtifactPath(name string) (string, error) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve frozen chain node relation artifact path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "..", "..", ".."))
	return filepath.Join(repositoryRoot, filepath.Dir(frozenChainNodeRelationArchiveRelativePath), name), nil
}

func frozenAdditiveChainNodeRelationArtifactPath() (string, error) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve frozen additive chain node relation artifact path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "..", "..", ".."))
	return filepath.Join(repositoryRoot, frozenAdditiveChainNodeRelationManifestRelativePath), nil
}

func decodeFrozenChainNodeJSON(content []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("frozen artifact must contain a single JSON document")
	}
	return nil
}

const relationDataBaselineSQL = `SELECT current_database(), current_setting('server_version'),
 (SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1),
 (SELECT count(*) FROM entity_nodes WHERE entity_type='chain_node' AND status='active'),
 (SELECT count(*) FROM chain_node_profiles p JOIN entity_nodes e ON e.id=p.entity_id WHERE e.entity_type='chain_node' AND e.status='active'),
 (SELECT count(*) FROM entity_external_identifiers),
 (SELECT count(*) FROM entity_edges),
 (SELECT count(*) FROM chain_node_relations),
 (SELECT count(*) FROM chain_node_relations WHERE relation_type='is_subcategory_of'),
 (SELECT count(*) FROM chain_node_relations WHERE relation_type='is_component_of'),
 (SELECT count(*) FROM chain_node_relations WHERE relation_type='input_to'),
 (SELECT count(*) FROM chain_node_relations WHERE relation_type='depends_on'),
 (SELECT count(*) FROM chain_node_physical_constraints)`

const relationDataSchemaSQL = `SELECT
 (SELECT string_agg(column_name||':'||udt_name||':'||is_nullable||':'||COALESCE(column_default,''),',' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='chain_node_relations'),
 (SELECT string_agg(column_name||':'||udt_name||':'||is_nullable||':'||COALESCE(column_default,''),',' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='chain_node_physical_constraints'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_relations'::regclass AND contype='c'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_relations'::regclass AND contype='f'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_relations'::regclass AND contype='p'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_relations'::regclass AND contype='u'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_physical_constraints'::regclass AND contype='c'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_physical_constraints'::regclass AND contype='f'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_physical_constraints'::regclass AND contype='p'),
 (SELECT count(*) FROM pg_indexes WHERE schemaname=current_schema() AND tablename='chain_node_relations' AND indexname IN ('chain_node_relations_pkey','chain_node_relations_from_chain_node_entity_id_to_chain_nod_key','chain_node_relations_to_type_idx','chain_node_relations_input_dependency_mechanism_uidx')),
 (SELECT count(*) FROM pg_indexes WHERE schemaname=current_schema() AND tablename='chain_node_physical_constraints' AND indexname IN ('chain_node_physical_constraints_pkey','chain_node_physical_constraints_node_subject_idx','chain_node_physical_constraints_relation_subject_idx')),
 (SELECT count(*) FROM pg_trigger WHERE tgrelid IN ('chain_node_relations'::regclass,'chain_node_physical_constraints'::regclass) AND NOT tgisinternal)`

const relationColumnSignature = "id:uuid:NO:,from_chain_node_entity_id:uuid:NO:,to_chain_node_entity_id:uuid:NO:,relation_type:text:NO:,mechanism:text:NO:,condition_note:text:YES:,evidence_note:text:NO:,provenance:text:NO:,verified_at:timestamptz:NO:,status:text:NO:'active'::text,created_at:timestamptz:NO:now(),updated_at:timestamptz:NO:now()"
const physicalConstraintColumnSignature = "id:uuid:NO:,chain_node_entity_id:uuid:YES:,chain_node_relation_id:uuid:YES:,constraint_type:text:NO:,description:text:NO:,condition_note:text:YES:,evidence_note:text:NO:,provenance:text:NO:,verified_at:timestamptz:NO:,status:text:NO:'active'::text,created_at:timestamptz:NO:now(),updated_at:timestamptz:NO:now()"

func readChainNodeRelationDataBaseline(ctx context.Context, db postgresExecutor) (ChainNodeRelationDataPreflightReport, error) {
	var report ChainNodeRelationDataPreflightReport
	if err := db.QueryRowContext(ctx, relationDataBaselineSQL).Scan(&report.DatabaseName, &report.ServerVersion, &report.GooseVersion, &report.ActiveChainNodes, &report.ChainNodeProfiles, &report.ExternalIdentifiers, &report.EntityEdges, &report.ExistingRelations, &report.SubcategoryRelations, &report.ComponentRelations, &report.InputRelations, &report.DependsRelations, &report.ExistingConstraints); err != nil {
		return report, err
	}
	var relationColumns, constraintColumns string
	var relationChecks, relationFKs, relationPKs, relationUniques, constraintChecks, constraintFKs, constraintPKs, relationIndexes, constraintIndexes, triggers int
	if err := db.QueryRowContext(ctx, relationDataSchemaSQL).Scan(&relationColumns, &constraintColumns, &relationChecks, &relationFKs, &relationPKs, &relationUniques, &constraintChecks, &constraintFKs, &constraintPKs, &relationIndexes, &constraintIndexes, &triggers); err != nil {
		return report, err
	}
	report.SchemaValid = relationColumns == relationColumnSignature && constraintColumns == physicalConstraintColumnSignature && relationChecks == 7 && relationFKs == 2 && relationPKs == 1 && relationUniques == 1 && constraintChecks == 7 && constraintFKs == 2 && constraintPKs == 1 && relationIndexes == 4 && constraintIndexes == 3 && triggers == 0
	return report, nil
}

func assertChainNodeRelationDataBaseline(ctx context.Context, db postgresExecutor, expectedRelations int) (ChainNodeRelationDataPreflightReport, error) {
	report, err := readChainNodeRelationDataBaseline(ctx, db)
	if err != nil {
		return report, err
	}
	if report.DatabaseName != "tidewise_local" || !strings.HasPrefix(report.ServerVersion, "16.14") || report.GooseVersion != 18 || report.ActiveChainNodes != 842 || report.ChainNodeProfiles != 842 || report.ExternalIdentifiers != 1169 || report.EntityEdges != 241 || report.ExistingRelations != expectedRelations || report.ExistingConstraints != 0 || !report.SchemaValid {
		return report, fmt.Errorf("relation data preflight baseline mismatch: %+v", report)
	}
	return report, nil
}

func preflightChainNodeRelationData(ctx context.Context, db postgresExecutor) (ChainNodeRelationDataPreflightReport, error) {
	report, err := assertChainNodeRelationDataBaseline(ctx, db, 100)
	if err != nil {
		return report, err
	}
	if err := validateFrozenChainNodeRelationDryRunBaseline(report); err != nil {
		return report, err
	}
	return report, nil
}

func (r PostgresRepository) PreflightFrozenChainNodeRelationData(ctx context.Context) (ChainNodeRelationDataPreflightReport, error) {
	if r.root == nil {
		return ChainNodeRelationDataPreflightReport{}, fmt.Errorf("postgres root database is required")
	}
	return preflightChainNodeRelationData(ctx, r.root)
}
