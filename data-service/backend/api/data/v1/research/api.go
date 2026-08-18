package research

import (
	"context"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationPublishResearchTheme            = "data.v1.publishResearchTheme"
	OperationListResearchThemes              = "data.v1.listResearchThemes"
	OperationGetResearchTheme                = "data.v1.getResearchTheme"
	OperationListResearchThemeReasoningTrees = "data.v1.listResearchThemeReasoningTrees"
	OperationGetResearchThemeReasoningTree   = "data.v1.getResearchThemeReasoningTree"
	OperationSearchResearchGraph             = "data.v1.searchResearchGraph"
)

type Service interface {
	PublishResearchTheme(context.Context, *ResearchThemeImportRequest) (*v1.Response[ResearchThemeImportResult], error)
	ListResearchThemes(context.Context, *ListResearchThemesRequest) (*v1.Response[ResearchThemePage], error)
	GetResearchTheme(context.Context, *GetResearchThemeRequest) (*v1.Response[ResearchThemeDetail], error)
	ListResearchReasoningTrees(context.Context, *ReasoningTreeListRequest) (*v1.Response[ResearchReasoningTreeList], error)
	GetResearchReasoningTree(context.Context, *ReasoningTreeDetailRequest) (*v1.Response[ResearchReasoningTreeDetail], error)
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
		OperationSearchResearchGraph,
	}
}

type ResearchThemeImportRequest struct {
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

type ResearchReasoningTreeImportCheckpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
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

type ResearchResourceLimitDetails struct {
	Component     string `json:"component"`
	ActualRows    *int64 `json:"actual_rows,omitempty"`
	MaxRows       *int64 `json:"max_rows,omitempty"`
	ActualBytes   *int64 `json:"actual_bytes,omitempty"`
	MaxBytes      *int64 `json:"max_bytes,omitempty"`
	RetryGuidance string `json:"retry_guidance"`
}

type ResearchGraphEntity struct {
	EntityID      string   `json:"entity_id"`
	EntityType    string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Status        string   `json:"status"`
}

type ResearchGraphEntityRelation struct {
	EntityRelationID string `json:"entity_relation_id"`
	FromEntityID     string `json:"from_entity_id"`
	ToEntityID       string `json:"to_entity_id"`
	RelationType     string `json:"relation_type"`
	Status           string `json:"status"`
}

type ResearchGraphRelationDefinition struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type ResearchGraphIndustryChain struct {
	IndustryChainID string `json:"industry_chain_id"`
	Scope           string `json:"scope"`
	TargetOutput    string `json:"target_output"`
	EndUse          string `json:"end_use"`
	Geography       string `json:"geography"`
	AsOfDate        string `json:"as_of_date"`
	ReviewStatus    string `json:"review_status"`
}

type ResearchGraphIndustryChainMembership struct {
	IndustryChainID string `json:"industry_chain_id"`
	ChainNodeID     string `json:"chain_node_id"`
	Position        int    `json:"position"`
	ContextualStage string `json:"contextual_stage"`
	ReviewStatus    string `json:"review_status"`
	Status          string `json:"status"`
}

type ResearchGraphIndustryChainGraphEdge struct {
	IndustryChainGraphEdgeID string  `json:"industry_chain_graph_edge_id"`
	IndustryChainID          string  `json:"industry_chain_id"`
	FromChainNodeID          string  `json:"from_chain_node_id"`
	ToChainNodeID            string  `json:"to_chain_node_id"`
	RelationType             string  `json:"relation_type"`
	Mechanism                string  `json:"mechanism"`
	ConditionNote            *string `json:"condition_note"`
	SegmentKind              string  `json:"segment_kind"`
	OmittedStepNote          *string `json:"omitted_step_note"`
	ReviewStatus             string  `json:"review_status"`
	Status                   string  `json:"status"`
}

type ResearchGraphSearchRequest struct {
	AnalysisAsOf    string                        `json:"analysis_as_of"`
	SeedEntityIDs   []string                      `json:"seed_entity_ids"`
	RelationFilters []ResearchGraphRelationFilter `json:"relation_filters"`
	MaxDepth        int                           `json:"max_depth"`
	IndustryChainID *string                       `json:"industry_chain_id,omitempty"`
	NodeBudget      int                           `json:"node_budget"`
	EdgeBudget      int                           `json:"edge_budget"`
}

type ResearchGraphRelationFilter struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type ResearchGraphSearchResult struct {
	ContractVersion          string                                 `json:"contract_version"`
	AnalysisAsOf             string                                 `json:"analysis_as_of"`
	QueryFingerprint         string                                 `json:"query_fingerprint"`
	GraphFingerprint         string                                 `json:"graph_fingerprint"`
	ActualDepth              int                                    `json:"actual_depth"`
	Entities                 []ResearchGraphEntity                  `json:"entities"`
	RelationDefinitions      []ResearchGraphRelationDefinition      `json:"relation_definitions"`
	EntityRelations          []ResearchGraphEntityRelation          `json:"entity_relations"`
	IndustryChains           []ResearchGraphIndustryChain           `json:"industry_chains"`
	IndustryChainMemberships []ResearchGraphIndustryChainMembership `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []ResearchGraphIndustryChainGraphEdge  `json:"industry_chain_graph_edges"`
}
type ResearchThemeImportResult struct {
	ReceiptID                 string                    `json:"receipt_id"`
	AnalysisBatchID           string                    `json:"analysis_batch_id"`
	PayloadHash               string                    `json:"payload_hash"`
	ThemeID                   string                    `json:"theme_id"`
	PublicationMode           string                    `json:"publication_mode"`
	ReasoningTreeIDsByTreeKey map[string]string         `json:"reasoning_tree_ids_by_tree_key"`
	Counts                    ResearchThemeImportCounts `json:"counts"`
	PublishedAt               time.Time                 `json:"published_at"`
	ImportedAt                time.Time                 `json:"imported_at"`
	Replayed                  bool                      `json:"replayed"`
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
	NodeKey         string  `json:"node_key"`
	DisplayName     string  `json:"display_name"`
	RelationRole    string  `json:"relation_role"`
	ImpactDirection string  `json:"impact_direction"`
	ImpactSummary   *string `json:"impact_summary"`
	DisplayOrder    int     `json:"display_order"`
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
	TreeKey         string    `json:"tree_key"`
	DisplayName     string    `json:"display_name"`
	ReasoningTreeID string    `json:"reasoning_tree_id"`
	Title           string    `json:"title"`
	DisplayOrder    int       `json:"display_order"`
	EventCount      int       `json:"event_count"`
	PublishedAt     time.Time `json:"published_at"`
}

type ResearchReasoningTreeList struct {
	Theme          ResearchTheme                  `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type ResearchReasoningTreeCheckpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type ResearchReasoningTreeSignal struct {
	SignalKey      string  `json:"signal_key"`
	VariableName   *string `json:"variable_name"`
	Direction      *string `json:"direction"`
	SignalRole     string  `json:"signal_role"`
	DisplaySummary string  `json:"display_summary"`
	DisplayOrder   int     `json:"display_order"`
}

type ResearchReasoningTreeNode struct {
	NodeKey                       string                        `json:"node_key"`
	DisplayName                   string                        `json:"display_name"`
	ID                            string                        `json:"id"`
	Position                      int                           `json:"position"`
	StateSummary                  *string                       `json:"state_summary"`
	ImpactDirection               string                        `json:"impact_direction"`
	ImpactStrength                string                        `json:"impact_strength"`
	ImpactSummary                 *string                       `json:"impact_summary"`
	ReasoningBasisSummary         *string                       `json:"reasoning_basis_summary"`
	EvidenceGapSummary            *string                       `json:"evidence_gap_summary"`
	IncomingTransmissionTitle     *string                       `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism *string                       `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary      *string                       `json:"incoming_condition_summary"`
	Signals                       []ResearchReasoningTreeSignal `json:"signals"`
	PrimarySignal                 ResearchReasoningTreeSignal   `json:"primary_signal"`
	SignalDisplaySummary          string                        `json:"signal_display_summary"`
}

type ResearchReasoningTree struct {
	TreeKey                   string                            `json:"tree_key"`
	DisplayName               string                            `json:"display_name"`
	ReasoningTreeID           string                            `json:"reasoning_tree_id"`
	ThemeID                   string                            `json:"theme_id"`
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
