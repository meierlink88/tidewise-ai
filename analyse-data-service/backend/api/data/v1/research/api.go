package research

import (
	"context"
	"encoding/json"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

const (
	OperationPublishResearchTheme            = "data.v1.publishResearchTheme"
	OperationListResearchThemes              = "data.v1.listResearchThemes"
	OperationGetResearchTheme                = "data.v1.getResearchTheme"
	OperationListResearchThemeReasoningTrees = "data.v1.listResearchThemeReasoningTrees"
	OperationGetResearchThemeReasoningTree   = "data.v1.getResearchThemeReasoningTree"
	OperationListResearchAnalysisContext     = "data.v1.listResearchAnalysisContext"
	OperationSearchResearchGraph             = "data.v1.searchResearchGraph"
)

type Service interface {
	PublishResearchTheme(context.Context, *ResearchThemeImportRequest) (*v1.Response[ResearchThemeImportResult], error)
	ListResearchThemes(context.Context, *ListResearchThemesRequest) (*v1.Response[ResearchThemePage], error)
	GetResearchTheme(context.Context, *GetResearchThemeRequest) (*v1.Response[ResearchThemeDetail], error)
	ListResearchReasoningTrees(context.Context, *ReasoningTreeListRequest) (*v1.Response[ResearchReasoningTreeList], error)
	GetResearchReasoningTree(context.Context, *ReasoningTreeDetailRequest) (*v1.Response[ResearchReasoningTreeDetail], error)
	ListResearchAnalysisContext(context.Context, *ResearchAnalysisContextRequest) (*v1.Response[ResearchAnalysisContext], error)
	SearchResearchGraph(context.Context, *ResearchGraphSearchRequest) (*v1.Response[ResearchGraphSearchResult], error)
}

type ListResearchThemesRequest struct {
	WindowHours   string
	PublishedFrom string
	PublishedTo   string
	Limit         string
	Cursor        string
}

type GetResearchThemeRequest struct {
	ThemeID     string
	WindowHours string
}

type ReasoningTreeListRequest struct {
	ThemeID  string
	HasQuery bool
}

type ReasoningTreeDetailRequest struct {
	ThemeID         string
	ReasoningTreeID string
	HasQuery        bool
}

func BusinessOperations() []string {
	return []string{
		OperationPublishResearchTheme,
		OperationListResearchThemes,
		OperationGetResearchTheme,
		OperationListResearchThemeReasoningTrees,
		OperationGetResearchThemeReasoningTree,
		OperationListResearchAnalysisContext,
		OperationSearchResearchGraph,
	}
}

type ResearchThemeImportRequest struct {
	AnalysisBatchID      string                              `json:"analysis_batch_id"`
	AnalysisAsOf         string                              `json:"analysis_as_of"`
	DiscoveryWindowStart string                              `json:"discovery_window_start"`
	DiscoveryWindowEnd   string                              `json:"discovery_window_end"`
	Theme                ResearchThemeImportItem             `json:"theme"`
	ReasoningTrees       []ResearchReasoningTreeImportItem   `json:"reasoning_trees"`
	PublicationMode      string                              `json:"-"`
	Snapshot             *ResearchThemeSnapshotImportRequest `json:"-"`
}

type ResearchThemeSnapshotImportRequest struct {
	PublicationMode      string                                    `json:"publication_mode"`
	AnalysisBatchID      string                                    `json:"analysis_batch_id"`
	AnalysisAsOf         string                                    `json:"analysis_as_of"`
	DiscoveryWindowStart string                                    `json:"discovery_window_start"`
	DiscoveryWindowEnd   string                                    `json:"discovery_window_end"`
	Theme                ResearchThemeSnapshotItem                 `json:"theme"`
	ReasoningTrees       []ResearchReasoningTreeSnapshotImportItem `json:"reasoning_trees"`
}

type ResearchThemeSnapshotItem struct {
	ThemeKey                  string                        `json:"theme_key"`
	Title                     string                        `json:"title"`
	OneLineConclusion         string                        `json:"one_line_conclusion"`
	ConclusionDirection       string                        `json:"conclusion_direction"`
	ImpactStrength            string                        `json:"impact_strength"`
	AttentionLevel            *string                       `json:"attention_level"`
	ConclusionStatus          *string                       `json:"conclusion_status"`
	TransmissionStage         string                        `json:"transmission_stage"`
	InvestmentGuidanceAction  string                        `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string                        `json:"investment_guidance_summary"`
	TimeHorizonCategory       string                        `json:"time_horizon_category"`
	TimeHorizonSummary        *string                       `json:"time_horizon_summary"`
	TransmissionSummary       *string                       `json:"transmission_summary"`
	CheckpointSummary         *string                       `json:"checkpoint_summary"`
	RiskSummary               *string                       `json:"risk_summary"`
	Impacts                   []ResearchThemeSnapshotImpact `json:"impacts"`
	Events                    []ResearchThemeSnapshotEvent  `json:"events"`
}

type ResearchThemeSnapshotImpact struct {
	NodeKey         string  `json:"node_key"`
	DisplayName     string  `json:"display_name"`
	RelationRole    string  `json:"relation_role"`
	ImpactDirection string  `json:"impact_direction"`
	ImpactSummary   *string `json:"impact_summary"`
	DisplayOrder    int     `json:"display_order"`
}

type ResearchThemeSnapshotEvent struct {
	EventID        string   `json:"event_id"`
	EvidenceIDs    []string `json:"evidence_ids,omitempty"`
	EvidenceRole   string   `json:"evidence_role"`
	SupportedClaim *string  `json:"supported_claim"`
}

type ResearchReasoningTreeSnapshotImportItem struct {
	TreeKey                   string                                  `json:"tree_key"`
	DisplayName               string                                  `json:"display_name"`
	Title                     string                                  `json:"title"`
	DisplayOrder              int                                     `json:"display_order"`
	OneLineConclusion         string                                  `json:"one_line_conclusion"`
	FactSummary               *string                                 `json:"fact_summary"`
	TransmissionSummary       *string                                 `json:"transmission_summary"`
	ImpactDirection           string                                  `json:"impact_direction"`
	ImpactStrength            string                                  `json:"impact_strength"`
	ImpactSummary             *string                                 `json:"impact_summary"`
	ConclusionBoundarySummary *string                                 `json:"conclusion_boundary_summary"`
	SupportSummary            *string                                 `json:"support_summary"`
	CounterSummary            *string                                 `json:"counter_summary"`
	InvalidationConditions    []string                                `json:"invalidation_conditions"`
	Checkpoints               []ResearchReasoningTreeImportCheckpoint `json:"checkpoints"`
	Events                    []ResearchReasoningTreeSnapshotEvent    `json:"events"`
	Nodes                     []ResearchReasoningTreeSnapshotNode     `json:"nodes"`
}

type ResearchReasoningTreeSnapshotEvent struct {
	EventID      string   `json:"event_id"`
	EvidenceIDs  []string `json:"evidence_ids,omitempty"`
	EvidenceRole string   `json:"evidence_role"`
	DisplayOrder int      `json:"display_order"`
}

type ResearchReasoningTreeSnapshotNode struct {
	NodeKey               string                                `json:"node_key"`
	DisplayName           string                                `json:"display_name"`
	Position              int                                   `json:"position"`
	StateSummary          *string                               `json:"state_summary"`
	ImpactDirection       string                                `json:"impact_direction"`
	ImpactStrength        string                                `json:"impact_strength"`
	ImpactSummary         *string                               `json:"impact_summary"`
	ReasoningBasisSummary *string                               `json:"reasoning_basis_summary"`
	EvidenceGapSummary    *string                               `json:"evidence_gap_summary"`
	IncomingTransmission  *ResearchSnapshotIncomingTransmission `json:"incoming_transmission"`
	Signals               []ResearchReasoningTreeSnapshotSignal `json:"signals"`
}

type ResearchSnapshotIncomingTransmission struct {
	Title            *string `json:"title"`
	Mechanism        string  `json:"mechanism"`
	ConditionSummary *string `json:"condition_summary"`
}

type ResearchReasoningTreeSnapshotSignal struct {
	SignalKey      string  `json:"signal_key"`
	DisplaySummary string  `json:"display_summary"`
	Role           string  `json:"role"`
	DisplayOrder   int     `json:"display_order"`
	VariableName   *string `json:"variable_name"`
	Direction      *string `json:"direction"`
}

type ResearchThemeImportItem struct {
	ThemeKey                  string                      `json:"theme_key"`
	Title                     string                      `json:"title"`
	OneLineConclusion         string                      `json:"one_line_conclusion"`
	ConclusionDirection       string                      `json:"conclusion_direction"`
	ImpactStrength            string                      `json:"impact_strength"`
	AttentionLevel            *string                     `json:"attention_level"`
	ConclusionStatus          *string                     `json:"conclusion_status"`
	TransmissionStage         string                      `json:"transmission_stage"`
	InvestmentGuidanceAction  string                      `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string                      `json:"investment_guidance_summary"`
	TimeHorizonCategory       string                      `json:"time_horizon_category"`
	TimeHorizonSummary        *string                     `json:"time_horizon_summary"`
	TransmissionSummary       *string                     `json:"transmission_summary"`
	CheckpointSummary         *string                     `json:"checkpoint_summary"`
	RiskSummary               *string                     `json:"risk_summary"`
	Impacts                   []ResearchThemeImportImpact `json:"impacts"`
	Events                    []ResearchThemeImportEvent  `json:"events"`
}

type ResearchThemeImportImpact struct {
	ChainNodeEntityID string  `json:"chain_node_entity_id"`
	RelationRole      string  `json:"relation_role"`
	ImpactDirection   string  `json:"impact_direction"`
	ImpactSummary     *string `json:"impact_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ResearchThemeImportEvent struct {
	EventID        string  `json:"event_id"`
	EvidenceRole   string  `json:"evidence_role"`
	SupportedClaim *string `json:"supported_claim"`
}

type ResearchReasoningTreeImportItem struct {
	IndustryChainEntityID     string                                  `json:"industry_chain_entity_id"`
	Title                     string                                  `json:"title"`
	DisplayOrder              int                                     `json:"display_order"`
	OneLineConclusion         string                                  `json:"one_line_conclusion"`
	FactSummary               *string                                 `json:"fact_summary"`
	TransmissionSummary       *string                                 `json:"transmission_summary"`
	ImpactDirection           string                                  `json:"impact_direction"`
	ImpactStrength            string                                  `json:"impact_strength"`
	ImpactSummary             *string                                 `json:"impact_summary"`
	ConclusionBoundarySummary *string                                 `json:"conclusion_boundary_summary"`
	SupportSummary            *string                                 `json:"support_summary"`
	CounterSummary            *string                                 `json:"counter_summary"`
	InvalidationConditions    []string                                `json:"invalidation_conditions"`
	Checkpoints               []ResearchReasoningTreeImportCheckpoint `json:"checkpoints"`
	Events                    []ResearchReasoningTreeImportEvent      `json:"events"`
	Nodes                     []ResearchReasoningTreeImportNode       `json:"nodes"`
}

type ResearchReasoningTreeImportCheckpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type ResearchReasoningTreeImportEvent struct {
	EventID      string `json:"event_id"`
	EvidenceRole string `json:"evidence_role"`
	DisplayOrder int    `json:"display_order"`
}

type ResearchReasoningTreeImportNode struct {
	Position                         int                                   `json:"position"`
	ChainNodeEntityID                string                                `json:"chain_node_entity_id"`
	StateSummary                     *string                               `json:"state_summary"`
	ImpactDirection                  string                                `json:"impact_direction"`
	ImpactStrength                   string                                `json:"impact_strength"`
	ImpactSummary                    *string                               `json:"impact_summary"`
	ReasoningBasisSummary            *string                               `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string                               `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string                               `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string                               `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string                               `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string                               `json:"incoming_condition_summary"`
	IncomingLineage                  *ResearchReasoningTreeIncomingLineage `json:"incoming_lineage"`
	Signals                          []ResearchReasoningTreeImportSignal   `json:"signals"`
}

type ResearchReasoningTreeImportSignal struct {
	VariableSignalKey string                             `json:"variable_signal_key"`
	SignalRole        string                             `json:"signal_role"`
	SignalDirection   string                             `json:"signal_direction"`
	DisplaySummary    string                             `json:"display_summary"`
	DisplayOrder      int                                `json:"display_order"`
	Lineage           ResearchReasoningTreeSignalLineage `json:"lineage"`
}

type ResearchReasoningTreeSignalLineage struct {
	SourceKind                      string  `json:"source_kind"`
	VariableSignalID                *string `json:"variable_signal_id"`
	SemanticSubmissionID            *string `json:"semantic_submission_id"`
	EvidenceID                      *string `json:"evidence_id"`
	EvidenceHash                    *string `json:"evidence_hash"`
	UpstreamVariableSignalID        *string `json:"upstream_variable_signal_id"`
	UpstreamDirectImpactAssertionID *string `json:"upstream_direct_impact_assertion_id"`
	EntityRelationID                *string `json:"entity_relation_id"`
	IndustryChainGraphEdgeID        *string `json:"industry_chain_graph_edge_id"`
}

type ResearchReasoningTreeIncomingLineage struct {
	SourceKind                      string  `json:"source_kind"`
	DirectImpactAssertionID         *string `json:"direct_impact_assertion_id"`
	SemanticSubmissionID            *string `json:"semantic_submission_id"`
	EvidenceID                      *string `json:"evidence_id"`
	EvidenceHash                    *string `json:"evidence_hash"`
	AffectedVariableKey             *string `json:"affected_variable_key"`
	AffectedDirection               *string `json:"affected_direction"`
	UpstreamVariableSignalID        *string `json:"upstream_variable_signal_id"`
	UpstreamDirectImpactAssertionID *string `json:"upstream_direct_impact_assertion_id"`
	EntityRelationID                *string `json:"entity_relation_id"`
}
type ResearchAnalysisContextRequest struct {
	DiscoveryWindowStart   string
	DiscoveryWindowEnd     string
	AnalysisAsOf           string
	PredictionHorizonStart *string
	PredictionHorizonEnd   *string
	PageSize               int
	Cursor                 string
}

type ResearchResourceLimitDetails struct {
	Component     string `json:"component"`
	ActualRows    *int64 `json:"actual_rows,omitempty"`
	MaxRows       *int64 `json:"max_rows,omitempty"`
	ActualBytes   *int64 `json:"actual_bytes,omitempty"`
	MaxBytes      *int64 `json:"max_bytes,omitempty"`
	RetryGuidance string `json:"retry_guidance"`
}

type ResearchAnalysisContextInconsistentDetails struct {
	RetryGuidance string `json:"retry_guidance"`
}

type ResearchAnalysisContext struct {
	ContractVersion             string                                `json:"contract_version"`
	TBoxContractVersion         string                                `json:"tbox_contract_version"`
	TemporalSemantics           string                                `json:"temporal_semantics"`
	TemporalLimitation          string                                `json:"temporal_limitation"`
	EventPageFingerprint        string                                `json:"event_page_fingerprint"`
	ReferenceClosureFingerprint string                                `json:"reference_closure_fingerprint"`
	DiscoveryWindowStart        string                                `json:"discovery_window_start"`
	DiscoveryWindowEnd          string                                `json:"discovery_window_end"`
	AnalysisAsOf                string                                `json:"analysis_as_of"`
	PredictionHorizonStart      *string                               `json:"prediction_horizon_start,omitempty"`
	PredictionHorizonEnd        *string                               `json:"prediction_horizon_end,omitempty"`
	EventSemanticBundles        []ResearchAnalysisEventSemanticBundle `json:"event_semantic_bundles"`
	Dictionaries                ResearchAnalysisDictionaries          `json:"dictionaries"`
	NextCursor                  string                                `json:"next_cursor,omitempty"`
	HasMore                     bool                                  `json:"has_more"`
}

type ResearchAnalysisEventSemanticBundle struct {
	Event           ResearchAnalysisEvent            `json:"event"`
	Evidence        []ResearchAnalysisEvidence       `json:"evidence"`
	EntityLinks     []ResearchAnalysisEntityLink     `json:"entity_links"`
	VariableSignals []ResearchAnalysisVariableSignal `json:"variable_signals"`
}

type ResearchAnalysisEvidence struct {
	EvidenceID           string   `json:"evidence_id"`
	EvidenceHash         string   `json:"evidence_hash"`
	Statement            string   `json:"evidence_statement"`
	SourceLevel          string   `json:"source_level"`
	Relation             string   `json:"relation"`
	SupportsFields       []string `json:"supports_fields"`
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

type ResearchAnalysisEntity struct {
	EntityID      string   `json:"entity_id"`
	EntityType    string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Status        string   `json:"status"`
}

type ResearchAnalysisEntityRelation struct {
	EntityRelationID string `json:"entity_relation_id"`
	FromEntityID     string `json:"from_entity_id"`
	ToEntityID       string `json:"to_entity_id"`
	RelationType     string `json:"relation_type"`
	Status           string `json:"status"`
}

type ResearchAnalysisTransmissionRule struct {
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

type ResearchAnalysisEvent struct {
	ID                   string     `json:"id"`
	Title                string     `json:"title"`
	Summary              string     `json:"summary"`
	OccurredAt           *time.Time `json:"occurred_at"`
	FirstSeenAt          time.Time  `json:"first_seen_at"`
	KnowledgeAvailableAt time.Time  `json:"knowledge_available_at"`
	EventStatus          string     `json:"event_status"`
	FactStatus           string     `json:"fact_status"`
}

type ResearchAnalysisEntityLink struct {
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

type ResearchAnalysisVariableSignal struct {
	VariableSignalID         string                         `json:"variable_signal_id"`
	SemanticSubmissionID     string                         `json:"semantic_submission_id"`
	SourceEventID            string                         `json:"source_event_id"`
	SubjectEventEntityLinkID string                         `json:"subject_event_entity_link_id"`
	SubjectEntityID          string                         `json:"subject_entity_id"`
	VariableKey              string                         `json:"variable_key"`
	VariableVersion          int                            `json:"variable_version"`
	Direction                string                         `json:"direction"`
	AssertionModality        string                         `json:"assertion_modality"`
	EvidenceIDs              []string                       `json:"evidence_ids"`
	StatementAt              *time.Time                     `json:"statement_at"`
	ValidFrom                *time.Time                     `json:"valid_from"`
	ValidUntil               *time.Time                     `json:"valid_until"`
	ForecastPeriodStart      *time.Time                     `json:"forecast_period_start"`
	ForecastPeriodEnd        *time.Time                     `json:"forecast_period_end"`
	ExtractionConfidence     *float64                       `json:"extraction_confidence"`
	ReviewStatus             string                         `json:"review_status"`
	Measurements             []ResearchAnalysisMeasurement  `json:"measurements"`
	DirectImpacts            []ResearchAnalysisDirectImpact `json:"direct_impacts"`
}

type ResearchAnalysisMeasurement struct {
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

type ResearchAnalysisDirectImpact struct {
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

type ResearchAnalysisDictionaries struct {
	Entities                 []ResearchAnalysisEntity                  `json:"entities"`
	RelationDefinitions      []ResearchAnalysisRelationDefinition      `json:"relation_definitions"`
	EntityRelations          []ResearchAnalysisEntityRelation          `json:"entity_relations"`
	IndustryChains           []ResearchAnalysisIndustryChain           `json:"industry_chains"`
	IndustryChainMemberships []ResearchAnalysisIndustryChainMembership `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []ResearchAnalysisIndustryChainGraphEdge  `json:"industry_chain_graph_edges"`
	EntityTypeDefinitions    []ResearchAnalysisEntityTypeDefinition    `json:"entity_type_definitions"`
	VariableDefinitions      []ResearchAnalysisVariableDefinition      `json:"variable_definitions"`
	DirectTransmissionRules  []ResearchAnalysisTransmissionRule        `json:"direct_transmission_rules"`
	AcceptancePolicies       []ResearchAnalysisAcceptancePolicy        `json:"acceptance_policies"`
}

type ResearchAnalysisRelationDefinition struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type ResearchAnalysisIndustryChain struct {
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	Scope                 string `json:"scope"`
	TargetOutput          string `json:"target_output"`
	EndUse                string `json:"end_use"`
	Geography             string `json:"geography"`
	AsOfDate              string `json:"as_of_date"`
	ReviewStatus          string `json:"review_status"`
}

type ResearchAnalysisIndustryChainMembership struct {
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	ChainNodeEntityID     string `json:"chain_node_entity_id"`
	Position              int    `json:"position"`
	ContextualStage       string `json:"contextual_stage"`
	ReviewStatus          string `json:"review_status"`
	Status                string `json:"status"`
}

type ResearchAnalysisIndustryChainGraphEdge struct {
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

type ResearchAnalysisEntityTypeDefinition struct {
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

type ResearchAnalysisVariableDefinition struct {
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

type ResearchAnalysisAcceptancePolicy struct {
	PolicyKey   string          `json:"policy_key"`
	Version     int             `json:"version"`
	RetryBudget int             `json:"retry_budget"`
	Status      string          `json:"status"`
	Policy      json.RawMessage `json:"policy"`
}
type ResearchGraphSearchRequest struct {
	AnalysisAsOf          string                        `json:"analysis_as_of"`
	SeedEntityIDs         []string                      `json:"seed_entity_ids"`
	RelationFilters       []ResearchGraphRelationFilter `json:"relation_filters"`
	MaxDepth              int                           `json:"max_depth"`
	IndustryChainEntityID *string                       `json:"industry_chain_entity_id,omitempty"`
	NodeBudget            int                           `json:"node_budget"`
	EdgeBudget            int                           `json:"edge_budget"`
}

type ResearchGraphRelationFilter struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type ResearchGraphSearchResult struct {
	ContractVersion          string                                    `json:"contract_version"`
	AnalysisAsOf             string                                    `json:"analysis_as_of"`
	QueryFingerprint         string                                    `json:"query_fingerprint"`
	GraphFingerprint         string                                    `json:"graph_fingerprint"`
	ActualDepth              int                                       `json:"actual_depth"`
	Entities                 []ResearchAnalysisEntity                  `json:"entities"`
	RelationDefinitions      []ResearchAnalysisRelationDefinition      `json:"relation_definitions"`
	EntityRelations          []ResearchAnalysisEntityRelation          `json:"entity_relations"`
	IndustryChains           []ResearchAnalysisIndustryChain           `json:"industry_chains"`
	IndustryChainMemberships []ResearchAnalysisIndustryChainMembership `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []ResearchAnalysisIndustryChainGraphEdge  `json:"industry_chain_graph_edges"`
}
type ResearchThemeImportResult struct {
	ReceiptID                               string                    `json:"receipt_id"`
	AnalysisBatchID                         string                    `json:"analysis_batch_id"`
	PayloadHash                             string                    `json:"payload_hash"`
	ThemeID                                 string                    `json:"theme_id"`
	PublicationMode                         string                    `json:"publication_mode"`
	ReasoningTreeIDsByIndustryChainEntityID map[string]string         `json:"reasoning_tree_ids_by_industry_chain_entity_id"`
	ReasoningTreeIDsByTreeKey               map[string]string         `json:"reasoning_tree_ids_by_tree_key"`
	Counts                                  ResearchThemeImportCounts `json:"counts"`
	PublishedAt                             time.Time                 `json:"published_at"`
	ImportedAt                              time.Time                 `json:"imported_at"`
	Replayed                                bool                      `json:"replayed"`
}

type ResearchThemeImportCounts struct {
	Themes                 int `json:"themes"`
	Impacts                int `json:"impacts"`
	ThemeEventAssociations int `json:"theme_event_associations"`
	ReasoningTrees         int `json:"reasoning_trees"`
	Nodes                  int `json:"nodes"`
	TreeEventAssociations  int `json:"tree_event_associations"`
	SignalAssociations     int `json:"signal_associations"`
	Receipts               int `json:"receipts"`
}

type ResearchThemePage struct {
	WindowStart time.Time       `json:"window_start"`
	WindowEnd   time.Time       `json:"window_end"`
	AsOf        time.Time       `json:"as_of"`
	ThemeCount  int             `json:"theme_count"`
	EventCount  int             `json:"event_count"`
	Items       []ResearchTheme `json:"items"`
	NextCursor  *string         `json:"next_cursor"`
}

type ResearchTheme struct {
	ID                        string                `json:"id"`
	AnalysisBatchID           string                `json:"analysis_batch_id"`
	Title                     string                `json:"title"`
	OneLineConclusion         string                `json:"one_line_conclusion"`
	ConclusionDirection       string                `json:"conclusion_direction"`
	ImpactStrength            string                `json:"impact_strength"`
	AttentionLevel            *string               `json:"attention_level"`
	ConclusionStatus          *string               `json:"conclusion_status"`
	TransmissionStage         string                `json:"transmission_stage"`
	InvestmentGuidanceAction  string                `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string                `json:"investment_guidance_summary"`
	TimeHorizonCategory       string                `json:"time_horizon_category"`
	TimeHorizonSummary        *string               `json:"time_horizon_summary"`
	TransmissionSummary       *string               `json:"transmission_summary"`
	CheckpointSummary         *string               `json:"checkpoint_summary"`
	RiskSummary               *string               `json:"risk_summary"`
	AnalysisAsOf              time.Time             `json:"analysis_as_of"`
	WindowStart               time.Time             `json:"window_start"`
	WindowEnd                 time.Time             `json:"window_end"`
	PublishedAt               time.Time             `json:"published_at"`
	Impacts                   []ResearchThemeImpact `json:"impacts"`
	EvidenceEventCount        int                   `json:"evidence_event_count"`
	ReasoningTreeCount        int                   `json:"reasoning_tree_count"`
}

type ResearchThemeDetail struct {
	ThemeKey                   string          `json:"theme_key"`
	PublicationMode            string          `json:"publication_mode"`
	PublicationContractVersion int             `json:"publication_contract_version"`
	Theme                      ResearchTheme   `json:"theme"`
	Events                     []ResearchEvent `json:"events"`
}

type ResearchThemeImpact struct {
	NodeKey           string  `json:"node_key"`
	DisplayName       string  `json:"display_name"`
	ChainNodeEntityID string  `json:"chain_node_entity_id"`
	Name              string  `json:"name"`
	RelationRole      string  `json:"relation_role"`
	ImpactDirection   string  `json:"impact_direction"`
	ImpactSummary     *string `json:"impact_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ResearchEvent struct {
	EventID        string     `json:"event_id"`
	EvidenceIDs    []string   `json:"evidence_ids"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim *string    `json:"supported_claim"`
	DisplayOrder   int        `json:"display_order"`
}

type ResearchReasoningTreeSummary struct {
	TreeKey               string    `json:"tree_key"`
	DisplayName           string    `json:"display_name"`
	ReasoningTreeID       string    `json:"reasoning_tree_id"`
	IndustryChainEntityID string    `json:"industry_chain_entity_id"`
	IndustryChainName     string    `json:"industry_chain_name"`
	Title                 string    `json:"title"`
	DisplayOrder          int       `json:"display_order"`
	EventCount            int       `json:"event_count"`
	PublishedAt           time.Time `json:"published_at"`
}

type ResearchReasoningTreeList struct {
	Theme          ResearchTheme                  `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type ResearchReasoningTreeCheckpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type ResearchReasoningTreeGraphEdge struct {
	ID           string `json:"id"`
	RelationType string `json:"relation_type"`
	ReviewStatus string `json:"review_status"`
	Status       string `json:"status"`
}

type ResearchReasoningTreeSignal struct {
	SignalKey         string  `json:"signal_key"`
	VariableName      *string `json:"variable_name"`
	Direction         *string `json:"direction"`
	VariableSignalKey string  `json:"variable_signal_key"`
	SignalRole        string  `json:"signal_role"`
	SignalDirection   string  `json:"signal_direction"`
	DisplaySummary    string  `json:"display_summary"`
	DisplayOrder      int     `json:"display_order"`
}

type ResearchReasoningTreeNode struct {
	NodeKey                          string                          `json:"node_key"`
	DisplayName                      string                          `json:"display_name"`
	ID                               string                          `json:"id"`
	Position                         int                             `json:"position"`
	ChainNodeEntityID                string                          `json:"chain_node_entity_id"`
	Name                             string                          `json:"name"`
	StateSummary                     *string                         `json:"state_summary"`
	ImpactDirection                  string                          `json:"impact_direction"`
	ImpactStrength                   string                          `json:"impact_strength"`
	ImpactSummary                    *string                         `json:"impact_summary"`
	ReasoningBasisSummary            *string                         `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string                         `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string                         `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string                         `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string                         `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string                         `json:"incoming_condition_summary"`
	IncomingGraphEdge                *ResearchReasoningTreeGraphEdge `json:"incoming_graph_edge"`
	Signals                          []ResearchReasoningTreeSignal   `json:"signals"`
	PrimarySignal                    ResearchReasoningTreeSignal     `json:"primary_signal"`
	SignalDisplaySummary             string                          `json:"signal_display_summary"`
}

type ResearchReasoningTree struct {
	TreeKey                   string                            `json:"tree_key"`
	DisplayName               string                            `json:"display_name"`
	ReasoningTreeID           string                            `json:"reasoning_tree_id"`
	ThemeID                   string                            `json:"theme_id"`
	IndustryChainEntityID     string                            `json:"industry_chain_entity_id"`
	IndustryChainName         string                            `json:"industry_chain_name"`
	Title                     string                            `json:"title"`
	DisplayOrder              int                               `json:"display_order"`
	OneLineConclusion         string                            `json:"one_line_conclusion"`
	FactSummary               *string                           `json:"fact_summary"`
	TransmissionSummary       *string                           `json:"transmission_summary"`
	ImpactDirection           string                            `json:"impact_direction"`
	ImpactStrength            string                            `json:"impact_strength"`
	ImpactSummary             *string                           `json:"impact_summary"`
	ConclusionBoundarySummary *string                           `json:"conclusion_boundary_summary"`
	SupportSummary            *string                           `json:"support_summary"`
	CounterSummary            *string                           `json:"counter_summary"`
	InvalidationConditions    []string                          `json:"invalidation_conditions"`
	Checkpoints               []ResearchReasoningTreeCheckpoint `json:"checkpoints"`
	PublishedAt               time.Time                         `json:"published_at"`
	EventCount                int                               `json:"event_count"`
	Events                    []ResearchEvent                   `json:"events"`
	Nodes                     []ResearchReasoningTreeNode       `json:"nodes"`
}

type ResearchReasoningTreeDetail struct {
	ThemeID                    string                `json:"theme_id"`
	ThemeKey                   string                `json:"theme_key"`
	PublicationMode            string                `json:"publication_mode"`
	PublicationContractVersion int                   `json:"publication_contract_version"`
	ImpactNodeIDs              []string              `json:"impact_node_ids"`
	ReasoningTree              ResearchReasoningTree `json:"reasoning_tree"`
}
