package v1

import "encoding/json"

type EventSemanticContextLeaseRequest struct {
	EventID                string `json:"event_id"`
	SupersedesSubmissionID string `json:"supersedes_submission_id,omitempty"`
	AgentExecutionID       string `json:"agent_execution_id"`
	WorkerID               string `json:"worker_id"`
	LeaseSeconds           int    `json:"lease_seconds"`
}

type EventSemanticContextLease struct {
	ContextLeaseID         string `json:"context_lease_id"`
	EventID                string `json:"event_id"`
	SupersedesSubmissionID string `json:"supersedes_submission_id,omitempty"`
	Status                 string `json:"status"`
	LeaseExpiresAt         string `json:"lease_expires_at"`
}

type EligibleEventSemanticEventsRequest struct {
	Limit  int
	Cursor string
}

type EligibleEventSemanticEvent struct {
	EventID string `json:"event_id"`
}

type EligibleEventSemanticEvents struct {
	Events     []EligibleEventSemanticEvent `json:"events"`
	NextCursor string                       `json:"next_cursor,omitempty"`
}

type EventSemanticContextRequest struct {
	ContextLeaseID string
}

type EventSemanticEvent struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Summary     string  `json:"summary"`
	OccurredAt  *string `json:"occurred_at"`
	EventStatus string  `json:"event_status"`
	FactStatus  string  `json:"fact_status"`
}

type EventSemanticEvidence struct {
	EvidenceID           string   `json:"evidence_id"`
	EvidenceHash         string   `json:"evidence_hash"`
	Excerpt              string   `json:"excerpt"`
	SourceLevel          string   `json:"source_level"`
	Relation             string   `json:"relation"`
	SupportsFields       []string `json:"supports_fields"`
	IsPrimary            bool     `json:"is_primary"`
	RawDocumentID        string   `json:"raw_document_id"`
	SourceName           string   `json:"source_name"`
	SourceType           string   `json:"source_type"`
	SourceURL            string   `json:"source_url,omitempty"`
	Title                string   `json:"title"`
	PublishedAt          *string  `json:"published_at,omitempty"`
	FirstSeenAt          string   `json:"first_seen_at"`
	KnowledgeAvailableAt string   `json:"knowledge_available_at"`
	AcceptedAt           string   `json:"accepted_at"`
	StatementSource      string   `json:"statement_source"`
}

type EventSemanticEntity struct {
	EntityID      string   `json:"entity_id"`
	EntityType    string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Status        string   `json:"status"`
}

type EventSemanticEntityRelation struct {
	EntityRelationID string `json:"entity_relation_id"`
	FromEntityID     string `json:"from_entity_id"`
	ToEntityID       string `json:"to_entity_id"`
	RelationType     string `json:"relation_type"`
	Status           string `json:"status"`
}

type EventSemanticVariableDefinition struct {
	Key                   string   `json:"key"`
	Version               int      `json:"version"`
	NameZH                string   `json:"name_zh"`
	NameEN                string   `json:"name_en"`
	Domain                string   `json:"domain"`
	ValueType             string   `json:"value_type"`
	Status                string   `json:"status"`
	AllowedDirections     []string `json:"allowed_directions"`
	ApplicableEntityTypes []string `json:"applicable_entity_types"`
}

type EventSemanticTransmissionRule struct {
	RuleKey                 string `json:"rule_key"`
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

type EventSemanticContext struct {
	ContextLeaseID          string                            `json:"context_lease_id"`
	OntologyVersion         string                            `json:"ontology_version"`
	AcceptancePolicyVersion string                            `json:"acceptance_policy_version"`
	Event                   EventSemanticEvent                `json:"event"`
	Evidence                []EventSemanticEvidence           `json:"evidence"`
	Entities                []EventSemanticEntity             `json:"entities"`
	Relations               []EventSemanticEntityRelation     `json:"relations"`
	VariableDefinitions     []EventSemanticVariableDefinition `json:"variable_definitions"`
	DirectTransmissionRules []EventSemanticTransmissionRule   `json:"direct_transmission_rules"`
}

type EventSemanticEntityMention struct {
	Mention            string   `json:"mention"`
	AllowedEntityTypes []string `json:"allowed_entity_types"`
}

type EventSemanticEntityResolutionRequest struct {
	ContextLeaseID string                       `json:"context_lease_id"`
	Mentions       []EventSemanticEntityMention `json:"mentions"`
}

type EventSemanticEntityResolution struct {
	Mention    string                `json:"mention"`
	Candidates []EventSemanticEntity `json:"candidates"`
	Ambiguous  bool                  `json:"ambiguous"`
}

type EventSemanticEntityResolutionResult struct {
	Resolutions []EventSemanticEntityResolution `json:"resolutions"`
}

type EventSemanticDirectTargetSearchRequest struct {
	ContextLeaseID     string   `json:"context_lease_id"`
	SubjectEntityID    string   `json:"subject_entity_id"`
	AllowedTargetTypes []string `json:"allowed_target_types"`
}

type EventSemanticDirectTarget struct {
	Entity   EventSemanticEntity         `json:"entity"`
	Relation EventSemanticEntityRelation `json:"relation"`
}

type EventSemanticDirectTargetSearchResult struct {
	Targets []EventSemanticDirectTarget `json:"targets"`
}

type EventSemanticMeasurement struct {
	MeasurementRole  string  `json:"measurement_role"`
	ValueShape       string  `json:"value_shape"`
	RawValue         *string `json:"raw_value,omitempty"`
	RawLower         *string `json:"raw_lower,omitempty"`
	RawUpper         *string `json:"raw_upper,omitempty"`
	RawUnit          string  `json:"raw_unit,omitempty"`
	CanonicalValue   *string `json:"canonical_value,omitempty"`
	CanonicalLower   *string `json:"canonical_lower,omitempty"`
	CanonicalUpper   *string `json:"canonical_upper,omitempty"`
	CanonicalUnit    string  `json:"canonical_unit,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	Scale            string  `json:"scale,omitempty"`
	ComparisonBasis  string  `json:"comparison_basis,omitempty"`
	ComparisonPeriod string  `json:"comparison_period,omitempty"`
	RawText          string  `json:"raw_text"`
	IsApproximate    bool    `json:"is_approximate"`
	EvidenceID       string  `json:"evidence_id"`
}

type EventSemanticEntityLinkCandidate struct {
	CandidateKey         string   `json:"candidate_key"`
	Mention              string   `json:"mention"`
	EntityID             string   `json:"entity_id"`
	EntityRole           string   `json:"entity_role"`
	EvidenceIDs          []string `json:"evidence_ids"`
	ResolutionMethod     string   `json:"resolution_method"`
	ResolutionConfidence string   `json:"resolution_confidence,omitempty"`
}

type EventSemanticVariableSignalCandidate struct {
	CandidateKey         string                     `json:"candidate_key"`
	SubjectLinkKey       string                     `json:"subject_link_key"`
	VariableKey          string                     `json:"variable_key"`
	VariableVersion      int                        `json:"variable_version"`
	Direction            string                     `json:"direction"`
	AssertionModality    string                     `json:"assertion_modality"`
	EvidenceIDs          []string                   `json:"evidence_ids"`
	Measurements         []EventSemanticMeasurement `json:"measurements"`
	StatementAt          *string                    `json:"statement_at,omitempty"`
	ValidFrom            *string                    `json:"valid_from,omitempty"`
	ValidUntil           *string                    `json:"valid_until,omitempty"`
	ForecastPeriodStart  *string                    `json:"forecast_period_start,omitempty"`
	ForecastPeriodEnd    *string                    `json:"forecast_period_end,omitempty"`
	ExtractionConfidence string                     `json:"extraction_confidence,omitempty"`
}

type EventSemanticDirectImpactCandidate struct {
	CandidateKey            string   `json:"candidate_key"`
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

type EventSemanticSubmissionRequest struct {
	ContextLeaseID          string                                 `json:"context_lease_id"`
	EventID                 string                                 `json:"event_id"`
	AgentExecutionID        string                                 `json:"agent_execution_id"`
	AgentKey                string                                 `json:"agent_key"`
	AgentVersion            string                                 `json:"agent_version"`
	SupersedesSubmissionID  string                                 `json:"supersedes_submission_id,omitempty"`
	GeneratorPromptHash     string                                 `json:"generator_prompt_hash"`
	GeneratorModel          string                                 `json:"generator_model"`
	ReviewerPromptHash      string                                 `json:"reviewer_prompt_hash"`
	ReviewerModel           string                                 `json:"reviewer_model"`
	AdjudicatorPromptHash   string                                 `json:"adjudicator_prompt_hash,omitempty"`
	AdjudicatorModel        string                                 `json:"adjudicator_model,omitempty"`
	OntologyVersion         string                                 `json:"ontology_version"`
	AcceptancePolicyVersion string                                 `json:"acceptance_policy_version"`
	EntityLinks             []EventSemanticEntityLinkCandidate     `json:"entity_links"`
	VariableSignals         []EventSemanticVariableSignalCandidate `json:"variable_signals"`
	DirectImpacts           []EventSemanticDirectImpactCandidate   `json:"direct_impacts"`
}

type EventSemanticCandidateDecision struct {
	CandidateKey string `json:"candidate_key"`
	Status       string `json:"status"`
	ReasonCode   string `json:"reason_code,omitempty"`
	RecordID     string `json:"record_id,omitempty"`
}

type EventSemanticReviewSnapshot struct {
	ReviewerExecutionKey string          `json:"reviewer_execution_key"`
	CanonicalPayloadHash string          `json:"canonical_payload_hash"`
	Payload              json.RawMessage `json:"payload"`
	CreatedAt            string          `json:"created_at"`
}

type EventSemanticReviewerWorkPackage struct {
	Event           EventSemanticEvent                     `json:"event"`
	Evidence        []EventSemanticEvidence                `json:"evidence"`
	EntityLinks     []EventSemanticEntityLinkCandidate     `json:"entity_links"`
	VariableSignals []EventSemanticVariableSignalCandidate `json:"variable_signals"`
	DirectImpacts   []EventSemanticDirectImpactCandidate   `json:"direct_impacts"`
}

type EventSemanticSubmissionResult struct {
	SubmissionID            string                            `json:"submission_id"`
	EventID                 string                            `json:"event_id"`
	Status                  string                            `json:"status"`
	CanonicalPayloadHash    string                            `json:"canonical_payload_hash"`
	Replayed                bool                              `json:"replayed"`
	EntityLinks             []EventSemanticCandidateDecision  `json:"entity_links"`
	VariableSignals         []EventSemanticCandidateDecision  `json:"variable_signals"`
	DirectImpacts           []EventSemanticCandidateDecision  `json:"direct_impacts"`
	ReviewerWorkPackage     *EventSemanticReviewerWorkPackage `json:"reviewer_work_package,omitempty"`
	AuditWorkPackage        *EventSemanticReviewerWorkPackage `json:"audit_work_package,omitempty"`
	ContextLeaseID          string                            `json:"context_lease_id,omitempty"`
	AgentExecutionID        string                            `json:"agent_execution_id,omitempty"`
	AgentKey                string                            `json:"agent_key,omitempty"`
	AgentVersion            string                            `json:"agent_version,omitempty"`
	SupersedesSubmissionID  string                            `json:"supersedes_submission_id,omitempty"`
	GeneratorPromptHash     string                            `json:"generator_prompt_hash,omitempty"`
	GeneratorModel          string                            `json:"generator_model,omitempty"`
	ReviewerPromptHash      string                            `json:"reviewer_prompt_hash,omitempty"`
	ReviewerModel           string                            `json:"reviewer_model,omitempty"`
	AdjudicatorPromptHash   string                            `json:"adjudicator_prompt_hash,omitempty"`
	AdjudicatorModel        string                            `json:"adjudicator_model,omitempty"`
	OntologyVersion         string                            `json:"ontology_version,omitempty"`
	AcceptancePolicyVersion string                            `json:"acceptance_policy_version,omitempty"`
	CandidateSnapshot       json.RawMessage                   `json:"candidate_snapshot,omitempty"`
	ReviewSnapshots         []EventSemanticReviewSnapshot     `json:"review_snapshots,omitempty"`
	CreatedAt               string                            `json:"created_at,omitempty"`
	FinalizedAt             *string                           `json:"finalized_at,omitempty"`
}

type EventSemanticReviewItem struct {
	CandidateType string   `json:"candidate_type"`
	CandidateKey  string   `json:"candidate_key"`
	Decision      string   `json:"decision"`
	ReasonCodes   []string `json:"reason_codes"`
	EvidenceIDs   []string `json:"evidence_ids"`
}

type EventSemanticReviewRequest struct {
	SubmissionID         string                    `json:"-"`
	ReviewerExecutionKey string                    `json:"reviewer_execution_key"`
	PromptHash           string                    `json:"prompt_hash"`
	Model                string                    `json:"model"`
	Items                []EventSemanticReviewItem `json:"items"`
}

type GetEventSemanticsRequest struct {
	EventID string
}

type EventSemanticsResult struct {
	EventID     string                          `json:"event_id"`
	Submissions []EventSemanticSubmissionResult `json:"submissions"`
}
