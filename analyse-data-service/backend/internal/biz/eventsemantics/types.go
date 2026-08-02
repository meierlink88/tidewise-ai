package eventsemantics

import (
	"encoding/json"
	"time"
)

type ReviewStatus string

const (
	StatusPendingReview   ReviewStatus = "pending_review"
	StatusNeedsReanalysis ReviewStatus = "needs_reanalysis"
	StatusQuarantined     ReviewStatus = "quarantined"
	StatusAccepted        ReviewStatus = "accepted"
	StatusRejected        ReviewStatus = "rejected"
	StatusSuperseded      ReviewStatus = "superseded"
)

type Event struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Summary    string     `json:"summary"`
	OccurredAt *time.Time `json:"occurred_at"`
	Status     string     `json:"event_status"`
	FactStatus string     `json:"fact_status"`
}

type Evidence struct {
	ID                   string     `json:"evidence_id"`
	Hash                 string     `json:"evidence_hash"`
	Statement            string     `json:"evidence_statement"`
	SourceLevel          string     `json:"source_level"`
	Relation             string     `json:"relation"`
	SupportsFields       []string   `json:"supports_fields"`
	RawDocumentID        string     `json:"raw_document_id"`
	SourceName           string     `json:"source_name"`
	SourceType           string     `json:"source_type"`
	SourceURL            string     `json:"source_url,omitempty"`
	Title                string     `json:"title"`
	PublishedAt          *time.Time `json:"published_at,omitempty"`
	FirstSeenAt          time.Time  `json:"first_seen_at"`
	KnowledgeAvailableAt time.Time  `json:"knowledge_available_at"`
	AcceptedAt           time.Time  `json:"accepted_at"`
	StatementSource      string     `json:"statement_source"`
}

type Entity struct {
	ID            string   `json:"entity_id"`
	Type          string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Status        string   `json:"status"`
}

type EntityRelation struct {
	ID           string `json:"entity_relation_id"`
	FromEntityID string `json:"from_entity_id"`
	ToEntityID   string `json:"to_entity_id"`
	Type         string `json:"relation_type"`
	Status       string `json:"status"`
}

type EntityTypeDefinition struct {
	TypeKey              string   `json:"type_key"`
	Version              int      `json:"version"`
	NameZH               string   `json:"name_zh"`
	NameEN               string   `json:"name_en"`
	BusinessDefinition   string   `json:"business_definition"`
	InclusionCriteria    []string `json:"inclusion_criteria"`
	ExclusionCriteria    []string `json:"exclusion_criteria"`
	EventLinkAllowed     bool     `json:"event_link_allowed"`
	SignalSubjectAllowed bool     `json:"signal_subject_allowed"`
	AllowedEventRoles    []string `json:"allowed_event_roles"`
	Status               string   `json:"status"`
}

type VariableDefinition struct {
	Key                   string   `json:"key"`
	Version               int      `json:"version"`
	NameZH                string   `json:"name_zh"`
	NameEN                string   `json:"name_en"`
	Domain                string   `json:"domain"`
	BusinessDefinition    string   `json:"business_definition"`
	ValueType             string   `json:"value_type"`
	Status                string   `json:"status"`
	AllowedDirections     []string `json:"allowed_directions"`
	AllowedUnits          []string `json:"allowed_units"`
	ApplicableEntityTypes []string `json:"applicable_entity_types"`
}

type DirectTransmissionRule struct {
	Key                     string `json:"rule_key"`
	Version                 int    `json:"version"`
	Status                  string `json:"status"`
	SourceEntityType        string `json:"source_entity_type"`
	SourceVariableKey       string `json:"source_variable_key"`
	SourceVariableVersion   int    `json:"source_variable_version"`
	SourceDirection         string `json:"source_direction"`
	RelationType            string `json:"relation_type"`
	TargetEntityType        string `json:"target_entity_type"`
	AffectedVariableKey     string `json:"affected_variable_key"`
	AffectedVariableVersion int    `json:"affected_variable_version"`
	AffectedDirection       string `json:"affected_direction"`
	ConditionSummary        string `json:"condition_summary"`
	MechanismTemplate       string `json:"mechanism_template"`
}

type Context struct {
	ContextLeaseID          string                   `json:"context_lease_id"`
	AgentExecutionID        string                   `json:"agent_execution_id"`
	WorkerID                string                   `json:"worker_id"`
	LeaseExpiresAt          time.Time                `json:"lease_expires_at"`
	ManifestContractVersion string                   `json:"manifest_contract_version"`
	ContextFingerprint      string                   `json:"context_fingerprint"`
	EventFingerprint        string                   `json:"event_fingerprint"`
	EvidenceFingerprint     string                   `json:"evidence_fingerprint"`
	OntologyVersion         string                   `json:"ontology_version"`
	PolicyVersion           string                   `json:"acceptance_policy_version"`
	RouteContractVersion    string                   `json:"-"`
	Event                   Event                    `json:"event"`
	Evidence                []Evidence               `json:"evidence"`
	EntityTypes             []EntityTypeDefinition   `json:"entity_type_definitions"`
	Variables               []VariableDefinition     `json:"variable_definitions"`
	AssertionModalities     []string                 `json:"assertion_modalities"`
	MeasurementContract     MeasurementContract      `json:"measurement_contract"`
	Rules                   []DirectTransmissionRule `json:"-"`
	// Legacy in-memory compatibility only. Compact manifests never persist these ABox collections.
	Entities  []Entity         `json:"-"`
	Relations []EntityRelation `json:"-"`
}

type MeasurementContract struct {
	Representation      string `json:"representation"`
	MaxItemsPerSignal   int    `json:"max_items_per_signal"`
	MaxTextCharacters   int    `json:"max_text_characters"`
	RequiresEvidenceIDs bool   `json:"requires_evidence_ids"`
	NumericValidation   bool   `json:"numeric_validation"`
}

type EvidenceReference struct {
	EvidenceID  string `json:"evidence_id"`
	Fingerprint string `json:"evidence_fingerprint"`
}

type VersionReference struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

// ContextManifest persists only immutable identities, version references and fingerprints.
// The Context API hydrates the bounded Event/Evidence/TBox payload from these references.
type ContextManifest struct {
	ContextLeaseID          string              `json:"context_lease_id"`
	AgentExecutionID        string              `json:"agent_execution_id"`
	WorkerID                string              `json:"worker_id"`
	LeaseStatus             string              `json:"lease_status"`
	LeaseExpiresAt          time.Time           `json:"lease_expires_at"`
	ManifestContractVersion string              `json:"manifest_contract_version"`
	ManifestFingerprint     string              `json:"manifest_fingerprint"`
	ContextFingerprint      string              `json:"context_fingerprint"`
	EventID                 string              `json:"event_id"`
	EventFingerprint        string              `json:"event_fingerprint"`
	Evidence                []EvidenceReference `json:"evidence_references"`
	EvidenceFingerprint     string              `json:"evidence_fingerprint"`
	OntologyVersion         string              `json:"ontology_version"`
	PolicyVersion           string              `json:"acceptance_policy_version"`
	RouteContractVersion    string              `json:"-"`
	EntityTypes             []VersionReference  `json:"entity_type_references"`
	Variables               []VersionReference  `json:"variable_definition_references"`
	Rules                   []VersionReference  `json:"-"`
}

type MeasurementValue struct {
	Text        string   `json:"measurement_text"`
	EvidenceIDs []string `json:"evidence_ids"`
	// Legacy fields remain readable for historical snapshots only and are never part of V2 writes.
	Role             string  `json:"-"`
	Shape            string  `json:"-"`
	RawValue         *string `json:"-"`
	RawLower         *string `json:"-"`
	RawUpper         *string `json:"-"`
	RawUnit          string  `json:"-"`
	CanonicalValue   *string `json:"-"`
	CanonicalLower   *string `json:"-"`
	CanonicalUpper   *string `json:"-"`
	CanonicalUnit    string  `json:"-"`
	Currency         string  `json:"-"`
	Scale            string  `json:"-"`
	ComparisonBasis  string  `json:"-"`
	ComparisonPeriod string  `json:"-"`
	RawText          string  `json:"-"`
	IsApproximate    bool    `json:"-"`
	EvidenceID       string  `json:"-"`
}

type EntityLinkCandidate struct {
	Key                  string             `json:"candidate_key"`
	Mention              string             `json:"mention"`
	EntityID             string             `json:"entity_id"`
	ProjectedEntityType  string             `json:"projected_entity_type"`
	EntityRole           string             `json:"entity_role"`
	EvidenceIDs          []string           `json:"evidence_ids"`
	ResolutionMethod     string             `json:"resolution_method"`
	ResolutionConfidence string             `json:"resolution_confidence,omitempty"`
	ResolutionReceipt    *ResolutionReceipt `json:"-"`
}

type VariableSignalCandidate struct {
	Key                  string             `json:"candidate_key"`
	SubjectLinkKey       string             `json:"subject_link_key"`
	VariableKey          string             `json:"variable_key"`
	VariableVersion      int                `json:"variable_version"`
	Direction            string             `json:"direction"`
	AssertionModality    string             `json:"assertion_modality"`
	EvidenceIDs          []string           `json:"evidence_ids"`
	Measurements         []MeasurementValue `json:"measurements"`
	StatementAt          *time.Time         `json:"statement_at,omitempty"`
	ValidFrom            *time.Time         `json:"valid_from,omitempty"`
	ValidUntil           *time.Time         `json:"valid_until,omitempty"`
	ForecastPeriodStart  *time.Time         `json:"forecast_period_start,omitempty"`
	ForecastPeriodEnd    *time.Time         `json:"forecast_period_end,omitempty"`
	ExtractionConfidence string             `json:"extraction_confidence,omitempty"`
}

type DirectImpactCandidate struct {
	Key                     string   `json:"candidate_key"`
	SourceSignalKey         string   `json:"source_signal_key"`
	TargetEntityID          string   `json:"target_entity_id"`
	AffectedVariableKey     string   `json:"affected_variable_key"`
	AffectedVariableVersion int      `json:"affected_variable_version"`
	AffectedDirection       string   `json:"affected_direction"`
	DerivationType          string   `json:"derivation_type"`
	MechanismSummary        string   `json:"mechanism_summary"`
	EntityRelationID        string   `json:"entity_relation_id,omitempty"`
	RuleKey                 string   `json:"rule_key,omitempty"`
	RuleVersion             int      `json:"rule_version,omitempty"`
	EvidenceIDs             []string `json:"evidence_ids"`
	AssertionConfidence     string   `json:"assertion_confidence,omitempty"`
}

type Submission struct {
	ContextLeaseID          string                    `json:"context_lease_id"`
	EventID                 string                    `json:"event_id"`
	AgentExecutionID        string                    `json:"agent_execution_id"`
	AgentKey                string                    `json:"agent_key"`
	AgentVersion            string                    `json:"agent_version"`
	SupersedesSubmissionID  string                    `json:"supersedes_submission_id,omitempty"`
	GeneratorPromptHash     string                    `json:"generator_prompt_hash"`
	GeneratorModel          string                    `json:"generator_model"`
	ReviewerPromptHash      string                    `json:"reviewer_prompt_hash"`
	ReviewerModel           string                    `json:"reviewer_model"`
	AdjudicatorPromptHash   string                    `json:"adjudicator_prompt_hash,omitempty"`
	AdjudicatorModel        string                    `json:"adjudicator_model,omitempty"`
	OntologyVersion         string                    `json:"ontology_version"`
	AcceptancePolicyVersion string                    `json:"acceptance_policy_version"`
	EntityLinks             []EntityLinkCandidate     `json:"entity_links"`
	VariableSignals         []VariableSignalCandidate `json:"variable_signals"`
	// Historical persistence compatibility only. V2 never accepts or emits DirectImpact candidates.
	DirectImpacts []DirectImpactCandidate `json:"-"`
}

type CandidateDecision struct {
	CandidateKey string       `json:"candidate_key"`
	Status       ReviewStatus `json:"status"`
	ReasonCode   string       `json:"reason_code,omitempty"`
	RecordID     string       `json:"record_id,omitempty"`
}

type ReviewerWorkPackage struct {
	Event            Event                     `json:"event"`
	Evidence         []Evidence                `json:"evidence"`
	ResolvedEntities []Entity                  `json:"resolved_entities"`
	EntityLinks      []EntityLinkCandidate     `json:"entity_links"`
	VariableSignals  []VariableSignalCandidate `json:"variable_signals"`
	DirectImpacts    []DirectImpactCandidate   `json:"-"`
}

type PrecheckResult struct {
	EntityLinks         []CandidateDecision `json:"entity_links"`
	VariableSignals     []CandidateDecision `json:"variable_signals"`
	DirectImpacts       []CandidateDecision `json:"-"`
	ReviewerWorkPackage ReviewerWorkPackage `json:"reviewer_work_package"`
}

type ContextLease struct {
	ID                     string
	EventID                string
	SupersedesSubmissionID string
	Status                 string
	LeaseExpiresAt         time.Time
}

type ContextLeaseRequest struct {
	EventID                string
	SupersedesSubmissionID string
	AgentExecutionID       string
	WorkerID               string
	Lease                  time.Duration
}

type EligibleEvent struct {
	EventID     string
	FirstSeenAt time.Time
}

type EligibleEventCursor struct {
	FirstSeenAt time.Time
	EventID     string
}

type EligibleEventPage struct {
	Events     []EligibleEvent
	NextCursor string
}

type EntityMention struct {
	Mention            string
	AllowedEntityTypes []string
}

type EntityResolution struct {
	Mention    string
	Candidates []Entity
	Ambiguous  bool
}

type DirectTarget struct {
	Entity   Entity
	Relation EntityRelation
}

type ResolutionRoute struct {
	ID                  string            `json:"route_id"`
	ContractVersion     string            `json:"route_contract_version"`
	TargetEntityType    string            `json:"target_entity_type"`
	AnchorEntityType    string            `json:"anchor_entity_type"`
	MappingRelationType string            `json:"mapping_relation_type"`
	Partitions          []string          `json:"partitions"`
	PartitionLabels     map[string]string `json:"partition_labels"`
	Direction           string            `json:"direction"`
	Purpose             string            `json:"purpose"`
	NextOperation       string            `json:"next_operation"`
	OrderingContract    string            `json:"ordering_contract"`
}

type ResolutionAnchor struct {
	Entity            Entity `json:"entity"`
	Partition         string `json:"partition"`
	Description       string `json:"description"`
	HierarchyIdentity string `json:"hierarchy_identity"`
}

type ResolutionReceipt struct {
	RouteID               string `json:"route_id"`
	RouteContractVersion  string `json:"route_contract_version"`
	AnchorEntityID        string `json:"anchor_entity_id"`
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	MappingRelationID     string `json:"mapping_relation_id"`
	TargetEntityID        string `json:"target_entity_id"`
	MembershipPosition    int    `json:"membership_position"`
	MembershipUpdatedAt   string `json:"membership_updated_at"`
	PathFingerprint       string `json:"path_fingerprint"`
}

type ResolutionCandidate struct {
	Entity                  Entity            `json:"entity"`
	Description             string            `json:"description"`
	MatchedAnchorEntityIDs  []string          `json:"matched_anchor_entity_ids"`
	IndustryChainEntityName string            `json:"industry_chain_entity_name"`
	Receipt                 ResolutionReceipt `json:"resolution_receipt"`
}

type ResolutionAnchorPage struct {
	Anchors    []ResolutionAnchor
	NextCursor string
}

type ResolutionCandidatePage struct {
	Candidates []ResolutionCandidate
	NextCursor string
}

// ResolutionKeyset is the stable database ordering boundary carried by an opaque API cursor.
type ResolutionKeyset struct {
	CanonicalName string
	EntityID      string
}

type ReviewItem struct {
	CandidateType string
	CandidateKey  string
	Decision      string
	ReasonCodes   []string
	EvidenceIDs   []string
}

type ReviewSubmission struct {
	SubmissionID         string
	ReviewerExecutionKey string
	PromptHash           string
	Model                string
	Items                []ReviewItem
}

type SubmissionResult struct {
	SubmissionID            string
	EventID                 string
	Status                  ReviewStatus
	CanonicalPayloadHash    string
	Replayed                bool
	Precheck                PrecheckResult
	ContextLeaseID          string
	AgentExecutionID        string
	AgentKey                string
	AgentVersion            string
	SupersedesSubmissionID  string
	GeneratorPromptHash     string
	GeneratorModel          string
	ReviewerPromptHash      string
	ReviewerModel           string
	AdjudicatorPromptHash   string
	AdjudicatorModel        string
	OntologyVersion         string
	AcceptancePolicyVersion string
	CandidateSnapshot       json.RawMessage
	ReviewSnapshots         []ReviewSnapshot
	CreatedAt               time.Time
	FinalizedAt             *time.Time
}

type ReviewSnapshot struct {
	ReviewerExecutionKey string
	CanonicalPayloadHash string
	Payload              json.RawMessage
	CreatedAt            time.Time
}

type EventSemanticsResult struct {
	EventID     string
	Submissions []SubmissionResult
}
