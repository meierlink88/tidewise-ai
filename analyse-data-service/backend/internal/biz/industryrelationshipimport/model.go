package industryrelationshipimport

import (
	"encoding/json"
	"time"
)

const (
	ManifestSchemaVersion = "industry_relationship_import_manifest_v1"
	PackageStatusApproved = "approved"
	ApprovalBasis         = "user_explicit_delegated_review"
	RelationSpecVersion   = "entity-relationship-construction-v1"
)

var RequiredFiles = []string{
	"industry_chains",
	"chain_node_additions",
	"industry_chain_industry_relations",
	"industry_chain_concept_relations",
	"industry_chain_node_memberships",
	"industry_chain_graph_edges",
	"global_chain_node_relations",
	"relationship_evidence",
	"concept_dispositions",
	"node_dispositions",
	"unmapped_relation_candidates",
	"relationship_validation_report",
}

type FileDescriptor struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Count  *int   `json:"count,omitempty"`
}

type RelationSpecDescriptor struct {
	Version    string `json:"version"`
	Path       string `json:"path"`
	OriginPath string `json:"origin_path"`
	SHA256     string `json:"sha256"`
}

type SourceDatabaseDescriptor struct {
	Kind        string    `json:"kind"`
	Environment string    `json:"environment"`
	SnapshotAt  time.Time `json:"snapshot_at"`
}

type SourceInventoryEntry struct {
	SourceKind string `json:"source_kind"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
}

type SourceInventory struct {
	SchemaVersion        string                   `json:"schema_version"`
	DatabaseSource       SourceDatabaseDescriptor `json:"database_source"`
	EvidenceCutoffAt     time.Time                `json:"evidence_cutoff_at"`
	RelationSpecVersion  string                   `json:"relation_spec_version"`
	ReviewMode           string                   `json:"review_mode"`
	AllowedRelationTypes []string                 `json:"allowed_relation_types"`
	Sources              []SourceInventoryEntry   `json:"sources"`
}

type Manifest struct {
	SchemaVersion      string                    `json:"schema_version"`
	PackageVersion     string                    `json:"package_version"`
	PackageStatus      string                    `json:"package_status"`
	ApprovalBasis      string                    `json:"approval_basis"`
	GeneratedAt        time.Time                 `json:"generated_at"`
	SourceSnapshotDate string                    `json:"source_snapshot_date"`
	RelationSpec       RelationSpecDescriptor    `json:"relation_spec"`
	SourceInventory    FileDescriptor            `json:"source_inventory"`
	Files              map[string]FileDescriptor `json:"files"`
	ProjectionFiles    map[string]FileDescriptor `json:"projection_files"`
	PackageCounts      map[string]int            `json:"package_counts"`
	PackageSHA256      string                    `json:"package_sha256"`
}

type IndustryChain struct {
	EntityID                  string   `json:"entity_id"`
	EntityKey                 string   `json:"entity_key"`
	EntityType                string   `json:"entity_type"`
	LayerCode                 string   `json:"layer_code"`
	Name                      string   `json:"name"`
	CanonicalName             string   `json:"canonical_name"`
	Aliases                   []string `json:"aliases"`
	Status                    string   `json:"status"`
	Scope                     string   `json:"scope"`
	TargetOutput              string   `json:"target_output"`
	EndUse                    string   `json:"end_use"`
	TechnologyRouteQualifier  *string  `json:"technology_route_qualifier"`
	Geography                 string   `json:"geography"`
	ObservableVariables       []string `json:"observable_variables"`
	AsOfDate                  string   `json:"as_of_date"`
	ReviewStatus              string   `json:"review_status"`
	ReviewNote                string   `json:"review_note"`
	RelationshipApprovalBasis string   `json:"approval_basis"`
}

type ChainNodeAddition struct {
	EntityID                  string            `json:"entity_id"`
	EntityKey                 string            `json:"entity_key"`
	EntityType                string            `json:"entity_type"`
	LayerCode                 string            `json:"layer_code"`
	Name                      string            `json:"name"`
	CanonicalName             string            `json:"canonical_name"`
	Aliases                   []string          `json:"aliases"`
	Status                    string            `json:"status"`
	Definition                string            `json:"definition"`
	BoundaryNote              string            `json:"boundary_note"`
	ObservableVariables       []string          `json:"observable_variables"`
	GateResults               map[string]string `json:"gate_results"`
	EvidenceIDs               []string          `json:"evidence_ids"`
	ReviewStatus              string            `json:"review_status"`
	RelationshipApprovalBasis string            `json:"approval_basis"`
	VerifiedAt                time.Time         `json:"verified_at"`
}

type EntityMapping struct {
	RelationID    string    `json:"relation_id"`
	RelationKey   string    `json:"relation_key"`
	FromKey       string    `json:"from_key"`
	FromEntityID  string    `json:"from_entity_id"`
	RelationType  string    `json:"relation_type"`
	ToKey         string    `json:"to_key"`
	ToEntityID    string    `json:"to_entity_id"`
	MappingReason string    `json:"mapping_reason"`
	EvidenceIDs   []string  `json:"evidence_ids"`
	EvidenceNote  string    `json:"evidence_note"`
	ReviewStatus  string    `json:"review_status"`
	Status        string    `json:"status"`
	VerifiedAt    time.Time `json:"verified_at"`
}

type Membership struct {
	RelationKey           string    `json:"relation_key"`
	IndustryChainEntityID string    `json:"industry_chain_entity_id"`
	ChainKey              string    `json:"chain_key"`
	ChainNodeEntityID     string    `json:"chain_node_entity_id"`
	NodeKey               string    `json:"node_key"`
	ContextualStage       string    `json:"contextual_stage"`
	Position              int       `json:"position"`
	InclusionReason       string    `json:"inclusion_reason"`
	EvidenceIDs           []string  `json:"evidence_ids"`
	SourceName            string    `json:"source_name"`
	SourceURL             string    `json:"source_url"`
	VerifiedAt            time.Time `json:"verified_at"`
	ReviewStatus          string    `json:"review_status"`
	Status                string    `json:"status"`
}

type GraphEdge struct {
	ID                    string    `json:"id"`
	RelationKey           string    `json:"relation_key"`
	IndustryChainEntityID string    `json:"industry_chain_entity_id"`
	ChainKey              string    `json:"chain_key"`
	FromChainNodeEntityID string    `json:"from_chain_node_entity_id"`
	FromNodeKey           string    `json:"from_node_key"`
	RelationType          string    `json:"relation_type"`
	ToChainNodeEntityID   string    `json:"to_chain_node_entity_id"`
	ToNodeKey             string    `json:"to_node_key"`
	Mechanism             string    `json:"mechanism"`
	ConditionNote         *string   `json:"condition_note"`
	SegmentKind           string    `json:"segment_kind"`
	OmittedStepNote       *string   `json:"omitted_step_note"`
	EvidenceIDs           []string  `json:"evidence_ids"`
	SourceName            string    `json:"source_name"`
	SourceURL             string    `json:"source_url"`
	VerifiedAt            time.Time `json:"verified_at"`
	ReviewStatus          string    `json:"review_status"`
	Status                string    `json:"status"`
}

type GlobalChainNodeRelation struct {
	ID                    string    `json:"id"`
	RelationKey           string    `json:"relation_key"`
	FromChainNodeEntityID string    `json:"from_chain_node_entity_id"`
	FromNodeKey           string    `json:"from_node_key"`
	FromName              string    `json:"from_name"`
	RelationType          string    `json:"relation_type"`
	ToChainNodeEntityID   string    `json:"to_chain_node_entity_id"`
	ToNodeKey             string    `json:"to_node_key"`
	ToName                string    `json:"to_name"`
	Mechanism             string    `json:"mechanism"`
	ConditionNote         *string   `json:"condition_note"`
	EvidenceNote          string    `json:"evidence_note"`
	Provenance            string    `json:"provenance"`
	Confidence            string    `json:"confidence"`
	ReviewStatus          string    `json:"review_status"`
	Status                string    `json:"status"`
	VerifiedAt            time.Time `json:"verified_at"`
}

type ValidationReport struct {
	SchemaVersion   string         `json:"schema_version"`
	Status          string         `json:"status"`
	ApprovalBasis   string         `json:"approval_basis"`
	VerifiedAt      time.Time      `json:"verified_at"`
	FrozenCounts    map[string]int `json:"frozen_counts"`
	PackageCounts   map[string]int `json:"package_counts"`
	HardGates       map[string]any `json:"hard_gates"`
	ClosedWorldNote string         `json:"closed_world_note"`
}

type Package struct {
	Directory           string
	Manifest            Manifest
	ManifestSHA256      string
	IndustryChains      []IndustryChain
	ChainNodeAdditions  []ChainNodeAddition
	IndustryMappings    []EntityMapping
	ConceptMappings     []EntityMapping
	Memberships         []Membership
	GraphEdges          []GraphEdge
	GlobalRelations     []GlobalChainNodeRelation
	Evidence            []json.RawMessage
	ConceptDispositions []json.RawMessage
	NodeDispositions    []json.RawMessage
	UnmappedCandidates  []json.RawMessage
	ValidationReport    ValidationReport
}

type Counts struct {
	IndustryChains      int `json:"industry_chain"`
	ChainNodeAdditions  int `json:"chain_node_additions"`
	IndustryMappings    int `json:"industry_chain_industry_relations"`
	ConceptMappings     int `json:"industry_chain_concept_relations"`
	Memberships         int `json:"industry_chain_node_memberships"`
	GraphEdges          int `json:"industry_chain_graph_edges"`
	GlobalRelations     int `json:"global_chain_node_relations"`
	Evidence            int `json:"relationship_evidence"`
	ConceptDispositions int `json:"concept_dispositions"`
	NodeDispositions    int `json:"node_dispositions"`
	UnmappedCandidates  int `json:"unmapped_relation_candidates"`
}

func (p Package) Counts() Counts {
	return Counts{
		IndustryChains: len(p.IndustryChains), ChainNodeAdditions: len(p.ChainNodeAdditions),
		IndustryMappings: len(p.IndustryMappings), ConceptMappings: len(p.ConceptMappings),
		Memberships: len(p.Memberships), GraphEdges: len(p.GraphEdges),
		GlobalRelations: len(p.GlobalRelations), Evidence: len(p.Evidence),
		ConceptDispositions: len(p.ConceptDispositions), NodeDispositions: len(p.NodeDispositions),
		UnmappedCandidates: len(p.UnmappedCandidates),
	}
}
