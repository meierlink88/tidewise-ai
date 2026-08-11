package researchanalysiscontext

import (
	"encoding/json"
	"time"
)

type EventSemanticBundle struct {
	Event           Event            `json:"event"`
	Evidence        []Evidence       `json:"evidence"`
	EntityLinks     []EntityLink     `json:"entity_links"`
	VariableSignals []VariableSignal `json:"variable_signals"`
}

type Event struct {
	ID                   string     `json:"id"`
	Title                string     `json:"title"`
	Summary              string     `json:"summary"`
	OccurredAt           *time.Time `json:"occurred_at"`
	FirstSeenAt          time.Time  `json:"first_seen_at"`
	KnowledgeAvailableAt time.Time  `json:"knowledge_available_at"`
	EventStatus          string     `json:"event_status"`
	FactStatus           string     `json:"fact_status"`
}

type Evidence struct {
	EvidenceID           string     `json:"evidence_id"`
	EvidenceHash         string     `json:"evidence_hash"`
	Statement            string     `json:"evidence_statement"`
	SourceLevel          string     `json:"source_level"`
	Relation             string     `json:"relation"`
	SupportsFields       []string   `json:"supports_fields"`
	RawDocumentID        string     `json:"raw_document_id"`
	SourceName           string     `json:"source_name"`
	SourceType           string     `json:"source_type"`
	SourceURL            *string    `json:"source_url"`
	Title                string     `json:"title"`
	PublishedAt          *time.Time `json:"published_at"`
	FirstSeenAt          time.Time  `json:"first_seen_at"`
	KnowledgeAvailableAt time.Time  `json:"knowledge_available_at"`
	AcceptedAt           time.Time  `json:"accepted_at"`
	StatementSource      string     `json:"statement_source"`
}

type EntityLink struct {
	EventEntityLinkID    string   `json:"event_entity_link_id"`
	SemanticSubmissionID string   `json:"semantic_submission_id"`
	EntityID             string   `json:"entity_id"`
	EntityRole           string   `json:"entity_role"`
	ResolvedMention      *string  `json:"resolved_mention"`
	ResolutionMethod     *string  `json:"resolution_method"`
	ResolutionConfidence *float64 `json:"resolution_confidence"`
	EvidenceIDs          []string `json:"evidence_ids"`
	ReviewStatus         string   `json:"review_status"`
}

type VariableSignal struct {
	VariableSignalID         string         `json:"variable_signal_id"`
	SemanticSubmissionID     string         `json:"semantic_submission_id"`
	SourceEventID            string         `json:"source_event_id"`
	SubjectEventEntityLinkID string         `json:"subject_event_entity_link_id"`
	SubjectEntityID          string         `json:"subject_entity_id"`
	VariableKey              string         `json:"variable_key"`
	VariableVersion          int            `json:"variable_version"`
	Direction                string         `json:"direction"`
	AssertionModality        string         `json:"assertion_modality"`
	EvidenceIDs              []string       `json:"evidence_ids"`
	StatementAt              *time.Time     `json:"statement_at"`
	ValidFrom                *time.Time     `json:"valid_from"`
	ValidUntil               *time.Time     `json:"valid_until"`
	ForecastPeriodStart      *time.Time     `json:"forecast_period_start"`
	ForecastPeriodEnd        *time.Time     `json:"forecast_period_end"`
	ExtractionConfidence     *float64       `json:"extraction_confidence"`
	ReviewStatus             string         `json:"review_status"`
	Measurements             []Measurement  `json:"measurements"`
	DirectImpacts            []DirectImpact `json:"direct_impacts"`
}

type Measurement struct {
	MeasurementID    string  `json:"measurement_id"`
	MeasurementRole  string  `json:"measurement_role"`
	ValueShape       string  `json:"value_shape"`
	RawValue         *string `json:"raw_value"`
	RawLower         *string `json:"raw_lower"`
	RawUpper         *string `json:"raw_upper"`
	RawUnit          *string `json:"raw_unit"`
	CanonicalValue   *string `json:"canonical_value"`
	CanonicalLower   *string `json:"canonical_lower"`
	CanonicalUpper   *string `json:"canonical_upper"`
	CanonicalUnit    *string `json:"canonical_unit"`
	Currency         *string `json:"currency"`
	Scale            *string `json:"scale"`
	ComparisonBasis  *string `json:"comparison_basis"`
	ComparisonPeriod *string `json:"comparison_period"`
	RawText          string  `json:"raw_text"`
	IsApproximate    bool    `json:"is_approximate"`
	EvidenceID       string  `json:"evidence_id"`
}

type DirectImpact struct {
	DirectImpactAssertionID string     `json:"direct_impact_assertion_id"`
	SemanticSubmissionID    string     `json:"semantic_submission_id"`
	SourceVariableSignalID  string     `json:"source_variable_signal_id"`
	TargetEntityID          string     `json:"target_entity_id"`
	AffectedVariableKey     string     `json:"affected_variable_key"`
	AffectedVariableVersion int        `json:"affected_variable_version"`
	AffectedDirection       string     `json:"affected_direction"`
	DerivationType          string     `json:"derivation_type"`
	MechanismSummary        string     `json:"mechanism_summary"`
	EvidenceIDs             []string   `json:"evidence_ids"`
	EntityRelationID        *string    `json:"entity_relation_id"`
	RuleKey                 *string    `json:"rule_key"`
	RuleVersion             *int       `json:"rule_version"`
	AssertionConfidence     *float64   `json:"assertion_confidence"`
	EffectiveFrom           *time.Time `json:"effective_from"`
	EffectiveTo             *time.Time `json:"effective_to"`
	ReviewStatus            string     `json:"review_status"`
}

type Dictionaries struct {
	Entities                 []Entity                  `json:"entities"`
	RelationDefinitions      []RelationDefinition      `json:"relation_definitions"`
	EntityRelations          []EntityRelation          `json:"entity_relations"`
	IndustryChains           []IndustryChain           `json:"industry_chains"`
	IndustryChainMemberships []IndustryChainMembership `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []IndustryChainGraphEdge  `json:"industry_chain_graph_edges"`
	EntityTypeDefinitions    []EntityTypeContext       `json:"entity_type_definitions"`
	VariableDefinitions      []VariableDefinition      `json:"variable_definitions"`
	DirectTransmissionRules  []DirectTransmissionRule  `json:"direct_transmission_rules"`
	AcceptancePolicies       []AcceptancePolicy        `json:"acceptance_policies"`
}

type Entity struct {
	EntityID      string   `json:"entity_id"`
	EntityType    string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Status        string   `json:"status"`
}

type RelationDefinition struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type EntityRelation struct {
	EntityRelationID string `json:"entity_relation_id"`
	FromEntityID     string `json:"from_entity_id"`
	ToEntityID       string `json:"to_entity_id"`
	RelationType     string `json:"relation_type"`
	Status           string `json:"status"`
}

type IndustryChain struct {
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	Scope                 string `json:"scope"`
	TargetOutput          string `json:"target_output"`
	EndUse                string `json:"end_use"`
	Geography             string `json:"geography"`
	AsOfDate              string `json:"as_of_date"`
	ReviewStatus          string `json:"review_status"`
}

type IndustryChainMembership struct {
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	ChainNodeEntityID     string `json:"chain_node_entity_id"`
	Position              int    `json:"position"`
	ContextualStage       string `json:"contextual_stage"`
	ReviewStatus          string `json:"review_status"`
	Status                string `json:"status"`
}

type IndustryChainGraphEdge struct {
	IndustryChainGraphEdgeID string  `json:"industry_chain_graph_edge_id"`
	IndustryChainEntityID    string  `json:"industry_chain_entity_id"`
	FromChainNodeEntityID    string  `json:"from_chain_node_entity_id"`
	ToChainNodeEntityID      string  `json:"to_chain_node_entity_id"`
	RelationType             string  `json:"relation_type"`
	Mechanism                string  `json:"mechanism"`
	ConditionNote            *string `json:"condition_note"`
	SegmentKind              string  `json:"segment_kind"`
	OmittedStepNote          *string `json:"omitted_step_note"`
	ReviewStatus             string  `json:"review_status"`
	Status                   string  `json:"status"`
}

type EntityTypeContext struct {
	TypeKey              string   `json:"type_key"`
	Version              int      `json:"version"`
	NameZH               string   `json:"name_zh"`
	NameEN               string   `json:"name_en"`
	BusinessDefinition   string   `json:"business_definition"`
	InclusionCriteria    []string `json:"inclusion_criteria"`
	ExclusionCriteria    []string `json:"exclusion_criteria"`
	EventLinkAllowed     bool     `json:"event_link_allowed"`
	SignalSubjectAllowed bool     `json:"signal_subject_allowed"`
	DirectTargetMode     string   `json:"direct_target_mode"`
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
	AllowedDirections     []string `json:"allowed_directions"`
	CanonicalUnit         *string  `json:"canonical_unit"`
	Status                string   `json:"status"`
	ApplicableEntityTypes []string `json:"applicable_entity_types"`
}

type DirectTransmissionRule struct {
	RuleKey                 string `json:"rule_key"`
	Version                 int    `json:"version"`
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
	Status                  string `json:"status"`
}

type AcceptancePolicy struct {
	PolicyKey   string          `json:"policy_key"`
	Version     int             `json:"version"`
	RetryBudget int             `json:"retry_budget"`
	Status      string          `json:"status"`
	Policy      json.RawMessage `json:"policy"`
}
