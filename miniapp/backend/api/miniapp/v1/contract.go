package v1

import "context"

const (
	APIPrefix                                = "/api/miniapp/v1"
	OperationListResearchThemes              = "/miniapp.v1.ResearchService/ListResearchThemes"
	OperationGetResearchTheme                = "/miniapp.v1.ResearchService/GetResearchTheme"
	OperationListResearchThemeReasoningTrees = "/miniapp.v1.ResearchService/ListResearchThemeReasoningTrees"
	OperationGetResearchThemeReasoningTree   = "/miniapp.v1.ResearchService/GetResearchThemeReasoningTree"
)

type ResearchHTTPServer interface {
	ListResearchThemes(context.Context, *ListResearchThemesRequest) (*ResearchThemeListResponse, error)
	GetResearchTheme(context.Context, *GetResearchThemeRequest) (*ResearchThemeDetailResponse, error)
	ListResearchThemeReasoningTrees(context.Context, *ListResearchThemeReasoningTreesRequest) (*ResearchReasoningTreeListResponse, error)
	GetResearchThemeReasoningTree(context.Context, *GetResearchThemeReasoningTreeRequest) (*ResearchReasoningTreeDetailResponse, error)
}
type ListResearchThemesRequest struct {
	Period             string
	WindowHours, Limit int
	Cursor             string
}
type GetResearchThemeRequest struct {
	ThemeID     string
	WindowHours int
}
type ListResearchThemeReasoningTreesRequest struct{ ThemeID string }
type GetResearchThemeReasoningTreeRequest struct{ ThemeID, ReasoningTreeID string }

type ResearchThemeListResponse struct {
	WindowStart string              `json:"window_start"`
	WindowEnd   string              `json:"window_end"`
	AsOf        string              `json:"as_of"`
	ThemeCount  int                 `json:"theme_count"`
	EventCount  int                 `json:"event_count"`
	Items       []ResearchThemeItem `json:"items"`
	NextCursor  *string             `json:"next_cursor"`
}
type ResearchThemeItem struct {
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
	AnalysisAsOf              string                `json:"analysis_as_of"`
	WindowStart               string                `json:"window_start"`
	WindowEnd                 string                `json:"window_end"`
	PublishedAt               string                `json:"published_at"`
	Impacts                   []ResearchThemeImpact `json:"impacts"`
	EvidenceEventCount        int                   `json:"evidence_event_count"`
	ReasoningTreeCount        int                   `json:"reasoning_tree_count"`
}
type ResearchThemeImpact struct {
	NodeKey         string  `json:"node_key"`
	DisplayName     string  `json:"display_name"`
	RelationRole    string  `json:"relation_role"`
	ImpactDirection string  `json:"impact_direction"`
	ImpactSummary   *string `json:"impact_summary"`
	DisplayOrder    int     `json:"display_order"`
}
type ResearchThemeDetailResponse struct {
	ResearchThemeItem
	Events []ResearchEvent `json:"events"`
}
type ResearchEvent struct {
	EventID        string   `json:"event_id"`
	EvidenceIDs    []string `json:"evidence_ids"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	EventTime      *string  `json:"event_time"`
	EvidenceRole   string   `json:"evidence_role"`
	SupportedClaim *string  `json:"supported_claim"`
	DisplayOrder   int      `json:"display_order"`
}

type ResearchReasoningTreeListResponse struct {
	Theme          ResearchThemeItem              `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}
type ResearchReasoningTreeSummary struct {
	TreeKey         string `json:"tree_key"`
	DisplayName     string `json:"display_name"`
	ReasoningTreeID string `json:"reasoning_tree_id"`
	Title           string `json:"title"`
	DisplayOrder    int    `json:"display_order"`
	EventCount      int    `json:"event_count"`
	PublishedAt     string `json:"published_at"`
}
type ResearchReasoningTreeDetailResponse struct {
	ThemeID       string                `json:"theme_id"`
	ImpactNodeIDs []string              `json:"impact_node_ids"`
	ReasoningTree ResearchReasoningTree `json:"reasoning_tree"`
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
	PublishedAt               string                            `json:"published_at"`
	EventCount                int                               `json:"event_count"`
	Events                    []ResearchEvent                   `json:"events"`
	Nodes                     []ResearchReasoningTreeNode       `json:"nodes"`
}
