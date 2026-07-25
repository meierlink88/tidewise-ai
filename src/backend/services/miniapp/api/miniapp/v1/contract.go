package v1

import "context"

const (
	APIPrefix = "/api/miniapp/v1"

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
	WindowHours int
	Limit       int
	Cursor      string
}

type GetResearchThemeRequest struct {
	ThemeID     string
	WindowHours int
}

type ListResearchThemeReasoningTreesRequest struct {
	ThemeID string
}

type GetResearchThemeReasoningTreeRequest struct {
	ThemeID  string
	AnchorID string
}

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
	ID                        string              `json:"id"`
	Name                      string              `json:"name"`
	OneLineConclusion         string              `json:"one_line_conclusion"`
	ImpactLevel               string              `json:"impact_level"`
	TransmissionPath          string              `json:"transmission_path"`
	TradingDirection          string              `json:"trading_direction"`
	TransmissionStage         string              `json:"transmission_stage"`
	NextCheckpoint            string              `json:"next_checkpoint"`
	MarketConfirmationSummary string              `json:"market_confirmation_summary"`
	PublishedAt               string              `json:"published_at"`
	AffectedChainNodes        []ResearchChainNode `json:"affected_chain_nodes"`
	RelatedIndices            []ResearchIndex     `json:"related_indices"`
	SupportingEventCount      int                 `json:"supporting_event_count"`
	ContradictingEventCount   int                 `json:"contradicting_event_count"`
}

type ResearchThemeDetailResponse struct {
	ResearchThemeItem
	Events []ResearchEvent `json:"events"`
}

type ResearchChainNode struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RelationRole  string `json:"relation_role"`
	ImpactSummary string `json:"impact_summary"`
}

type ResearchIndex struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImpactDirection string `json:"impact_direction"`
	ImpactSummary   string `json:"impact_summary"`
}

type ResearchEvent struct {
	EventID        string  `json:"event_id"`
	Title          string  `json:"title"`
	Summary        string  `json:"summary"`
	EventTime      *string `json:"event_time,omitempty"`
	EvidenceRole   string  `json:"evidence_role"`
	SupportedClaim string  `json:"supported_claim"`
}

type ResearchReasoningTreeListResponse struct {
	Theme          ResearchThemeItem              `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type ResearchReasoningTreeSummary struct {
	AnchorID        string                         `json:"anchor_id"`
	CenterChainNode ResearchReasoningTreeChainNode `json:"center_chain_node"`
}

type ResearchReasoningTreeChainNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResearchReasoningTreeDetailResponse struct {
	ThemeID       string                `json:"theme_id"`
	ReasoningTree ResearchReasoningTree `json:"reasoning_tree"`
}

type ResearchReasoningTree struct {
	AnchorID            string                          `json:"anchor_id"`
	CenterChainNode     ResearchReasoningTreeChainNode  `json:"center_chain_node"`
	OneLineConclusion   string                          `json:"one_line_conclusion"`
	FactSummary         string                          `json:"fact_summary"`
	NetDirectionSummary string                          `json:"net_direction_summary"`
	SupportSummary      string                          `json:"support_summary"`
	CounterSummary      *string                         `json:"counter_summary"`
	TradingDirection    string                          `json:"trading_direction"`
	NextCheckpoint      string                          `json:"next_checkpoint"`
	EventCount          int                             `json:"event_count"`
	Events              []ResearchReasoningTreeEvent    `json:"events"`
	PathNodes           []ResearchReasoningTreePathNode `json:"path_nodes"`
}

type ResearchReasoningTreeEvent struct {
	EventID         string  `json:"event_id"`
	Title           string  `json:"title"`
	Summary         string  `json:"summary"`
	EventTime       *string `json:"event_time"`
	EvidenceRole    string  `json:"evidence_role"`
	EvidenceSummary string  `json:"evidence_summary"`
}

type ResearchReasoningTreePathNode struct {
	ChainNodeID                   string  `json:"chain_node_id"`
	Name                          string  `json:"name"`
	ChangeDirection               string  `json:"change_direction"`
	ChangeSummary                 string  `json:"change_summary"`
	ImpactSummary                 string  `json:"impact_summary"`
	IncomingTransmissionMechanism *string `json:"incoming_transmission_mechanism"`
}
