package data

import (
	"time"

	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
)

type wireResearchThemePage struct {
	WindowStart time.Time           `json:"window_start"`
	WindowEnd   time.Time           `json:"window_end"`
	AsOf        time.Time           `json:"as_of"`
	ThemeCount  int                 `json:"theme_count"`
	EventCount  int                 `json:"event_count"`
	Items       []wireResearchTheme `json:"items"`
	NextCursor  *string             `json:"next_cursor"`
}
type wireResearchTheme struct {
	ID                        string                    `json:"id"`
	AnalysisBatchID           string                    `json:"analysis_batch_id"`
	Title                     string                    `json:"title"`
	OneLineConclusion         string                    `json:"one_line_conclusion"`
	ConclusionDirection       string                    `json:"conclusion_direction"`
	ImpactStrength            string                    `json:"impact_strength"`
	AttentionLevel            *string                   `json:"attention_level"`
	ConclusionStatus          *string                   `json:"conclusion_status"`
	TransmissionStage         string                    `json:"transmission_stage"`
	InvestmentGuidanceAction  string                    `json:"investment_guidance_action"`
	InvestmentGuidanceSummary string                    `json:"investment_guidance_summary"`
	TimeHorizonCategory       string                    `json:"time_horizon_category"`
	TimeHorizonSummary        *string                   `json:"time_horizon_summary"`
	TransmissionSummary       *string                   `json:"transmission_summary"`
	CheckpointSummary         *string                   `json:"checkpoint_summary"`
	RiskSummary               *string                   `json:"risk_summary"`
	AnalysisAsOf              time.Time                 `json:"analysis_as_of"`
	WindowStart               time.Time                 `json:"window_start"`
	WindowEnd                 time.Time                 `json:"window_end"`
	PublishedAt               time.Time                 `json:"published_at"`
	Impacts                   []wireResearchThemeImpact `json:"impacts"`
	EvidenceEventCount        int                       `json:"evidence_event_count"`
	ReasoningTreeCount        int                       `json:"reasoning_tree_count"`
}
type wireResearchThemeImpact struct {
	ChainNodeEntityID           string  `json:"chain_node_entity_id"`
	Name                        string  `json:"name"`
	RelationRole                string  `json:"relation_role"`
	ImpactDirection             string  `json:"impact_direction"`
	ImpactSummary               *string `json:"impact_summary"`
	PrimarySignalDisplaySummary *string `json:"primary_signal_display_summary"`
	DisplayOrder                int     `json:"display_order"`
}
type wireResearchThemeDetail struct {
	Theme  wireResearchTheme   `json:"theme"`
	Events []wireResearchEvent `json:"events"`
}
type wireResearchEvent struct {
	EventID        string     `json:"event_id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim *string    `json:"supported_claim"`
	DisplayOrder   int        `json:"display_order"`
}
type wireResearchReasoningTreeSummary struct {
	ReasoningTreeID       string    `json:"reasoning_tree_id"`
	IndustryChainEntityID string    `json:"industry_chain_entity_id"`
	IndustryChainName     string    `json:"industry_chain_name"`
	Title                 string    `json:"title"`
	DisplayOrder          int       `json:"display_order"`
	EventCount            int       `json:"event_count"`
	PublishedAt           time.Time `json:"published_at"`
}
type wireResearchReasoningTreeList struct {
	Theme          wireResearchTheme                  `json:"theme"`
	ReasoningTrees []wireResearchReasoningTreeSummary `json:"reasoning_trees"`
}
type wireResearchCheckpoint struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}
type wireResearchGraphEdge struct {
	ID           string `json:"id"`
	RelationType string `json:"relation_type"`
	ReviewStatus string `json:"review_status"`
	Status       string `json:"status"`
}
type wireResearchSignal struct {
	VariableSignalKey string `json:"variable_signal_key"`
	SignalRole        string `json:"signal_role"`
	SignalDirection   string `json:"signal_direction"`
	DisplaySummary    string `json:"display_summary"`
	DisplayOrder      int    `json:"display_order"`
}
type wireResearchReasoningTreeNode struct {
	ID                               string                 `json:"id"`
	Position                         int                    `json:"position"`
	ChainNodeEntityID                string                 `json:"chain_node_entity_id"`
	Name                             string                 `json:"name"`
	StateSummary                     *string                `json:"state_summary"`
	ImpactDirection                  string                 `json:"impact_direction"`
	ImpactStrength                   string                 `json:"impact_strength"`
	ImpactSummary                    *string                `json:"impact_summary"`
	ReasoningBasisSummary            *string                `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string                `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string                `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string                `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string                `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string                `json:"incoming_condition_summary"`
	IncomingGraphEdge                *wireResearchGraphEdge `json:"incoming_graph_edge"`
	Signals                          []wireResearchSignal   `json:"signals"`
	PrimarySignal                    wireResearchSignal     `json:"primary_signal"`
	SignalDisplaySummary             string                 `json:"signal_display_summary"`
}
type wireResearchReasoningTree struct {
	ReasoningTreeID           string                          `json:"reasoning_tree_id"`
	ThemeID                   string                          `json:"theme_id"`
	IndustryChainEntityID     string                          `json:"industry_chain_entity_id"`
	IndustryChainName         string                          `json:"industry_chain_name"`
	Title                     string                          `json:"title"`
	DisplayOrder              int                             `json:"display_order"`
	OneLineConclusion         string                          `json:"one_line_conclusion"`
	FactSummary               *string                         `json:"fact_summary"`
	TransmissionSummary       *string                         `json:"transmission_summary"`
	ImpactDirection           string                          `json:"impact_direction"`
	ImpactStrength            string                          `json:"impact_strength"`
	ImpactSummary             *string                         `json:"impact_summary"`
	ConclusionBoundarySummary *string                         `json:"conclusion_boundary_summary"`
	SupportSummary            *string                         `json:"support_summary"`
	CounterSummary            *string                         `json:"counter_summary"`
	InvalidationConditions    []string                        `json:"invalidation_conditions"`
	Checkpoints               []wireResearchCheckpoint        `json:"checkpoints"`
	PublishedAt               time.Time                       `json:"published_at"`
	EventCount                int                             `json:"event_count"`
	Events                    []wireResearchEvent             `json:"events"`
	Nodes                     []wireResearchReasoningTreeNode `json:"nodes"`
}
type wireResearchReasoningTreeDetail struct {
	ThemeID       string                    `json:"theme_id"`
	ImpactNodeIDs []string                  `json:"impact_node_ids"`
	ReasoningTree wireResearchReasoningTree `json:"reasoning_tree"`
}

func (v wireResearchThemePage) toBiz() biz.ResearchThemePage {
	return biz.ResearchThemePage{WindowStart: v.WindowStart, WindowEnd: v.WindowEnd, AsOf: v.AsOf, ThemeCount: v.ThemeCount, EventCount: v.EventCount, Items: mapSlice(v.Items, wireResearchTheme.toBiz), NextCursor: v.NextCursor}
}
func (v wireResearchTheme) toBiz() biz.ResearchTheme {
	return biz.ResearchTheme{
		ID: v.ID, AnalysisBatchID: v.AnalysisBatchID, Title: v.Title, OneLineConclusion: v.OneLineConclusion,
		ConclusionDirection: v.ConclusionDirection, ImpactStrength: v.ImpactStrength, AttentionLevel: v.AttentionLevel,
		ConclusionStatus: v.ConclusionStatus, TransmissionStage: v.TransmissionStage,
		InvestmentGuidanceAction: v.InvestmentGuidanceAction, InvestmentGuidanceSummary: v.InvestmentGuidanceSummary,
		TimeHorizonCategory: v.TimeHorizonCategory, TimeHorizonSummary: v.TimeHorizonSummary,
		TransmissionSummary: v.TransmissionSummary, CheckpointSummary: v.CheckpointSummary, RiskSummary: v.RiskSummary,
		AnalysisAsOf: v.AnalysisAsOf, WindowStart: v.WindowStart, WindowEnd: v.WindowEnd, PublishedAt: v.PublishedAt,
		Impacts: mapSlice(v.Impacts, wireResearchThemeImpact.toBiz), EvidenceEventCount: v.EvidenceEventCount, ReasoningTreeCount: v.ReasoningTreeCount,
	}
}
func (v wireResearchThemeImpact) toBiz() biz.ResearchThemeImpact {
	return biz.ResearchThemeImpact{ChainNodeEntityID: v.ChainNodeEntityID, Name: v.Name, RelationRole: v.RelationRole, ImpactDirection: v.ImpactDirection, ImpactSummary: v.ImpactSummary, PrimarySignalDisplaySummary: v.PrimarySignalDisplaySummary, DisplayOrder: v.DisplayOrder}
}
func (v wireResearchEvent) toBiz() biz.ResearchEvent {
	return biz.ResearchEvent{EventID: v.EventID, Title: v.Title, Summary: v.Summary, EventTime: v.EventTime, EvidenceRole: v.EvidenceRole, SupportedClaim: v.SupportedClaim, DisplayOrder: v.DisplayOrder}
}
func (v wireResearchThemeDetail) toBiz() biz.ResearchThemeDetail {
	return biz.ResearchThemeDetail{Theme: v.Theme.toBiz(), Events: mapSlice(v.Events, wireResearchEvent.toBiz)}
}
func (v wireResearchReasoningTreeSummary) toBiz() biz.ResearchReasoningTreeSummary {
	return biz.ResearchReasoningTreeSummary{ReasoningTreeID: v.ReasoningTreeID, IndustryChainEntityID: v.IndustryChainEntityID, IndustryChainName: v.IndustryChainName, Title: v.Title, DisplayOrder: v.DisplayOrder, EventCount: v.EventCount, PublishedAt: v.PublishedAt}
}
func (v wireResearchReasoningTreeList) toBiz() biz.ResearchReasoningTreeList {
	return biz.ResearchReasoningTreeList{Theme: v.Theme.toBiz(), ReasoningTrees: mapSlice(v.ReasoningTrees, wireResearchReasoningTreeSummary.toBiz)}
}
func (v wireResearchSignal) toBiz() biz.ResearchSignal {
	return biz.ResearchSignal{VariableSignalKey: v.VariableSignalKey, SignalRole: v.SignalRole, SignalDirection: v.SignalDirection, DisplaySummary: v.DisplaySummary, DisplayOrder: v.DisplayOrder}
}
func (v wireResearchReasoningTreeNode) toBiz() biz.ResearchReasoningTreeNode {
	var edge *biz.ResearchGraphEdge
	if v.IncomingGraphEdge != nil {
		edge = &biz.ResearchGraphEdge{ID: v.IncomingGraphEdge.ID, RelationType: v.IncomingGraphEdge.RelationType, ReviewStatus: v.IncomingGraphEdge.ReviewStatus, Status: v.IncomingGraphEdge.Status}
	}
	return biz.ResearchReasoningTreeNode{ID: v.ID, Position: v.Position, ChainNodeEntityID: v.ChainNodeEntityID, Name: v.Name, StateSummary: v.StateSummary, ImpactDirection: v.ImpactDirection, ImpactStrength: v.ImpactStrength, ImpactSummary: v.ImpactSummary, ReasoningBasisSummary: v.ReasoningBasisSummary, EvidenceGapSummary: v.EvidenceGapSummary, IncomingIndustryChainGraphEdgeID: v.IncomingIndustryChainGraphEdgeID, IncomingTransmissionTitle: v.IncomingTransmissionTitle, IncomingTransmissionMechanism: v.IncomingTransmissionMechanism, IncomingConditionSummary: v.IncomingConditionSummary, IncomingGraphEdge: edge, Signals: mapSlice(v.Signals, wireResearchSignal.toBiz), PrimarySignal: v.PrimarySignal.toBiz(), SignalDisplaySummary: v.SignalDisplaySummary}
}
func (v wireResearchReasoningTree) toBiz() biz.ResearchReasoningTree {
	checkpoints := make([]biz.ResearchCheckpoint, 0, len(v.Checkpoints))
	for _, c := range v.Checkpoints {
		checkpoints = append(checkpoints, biz.ResearchCheckpoint{Type: c.Type, Summary: c.Summary})
	}
	return biz.ResearchReasoningTree{ReasoningTreeID: v.ReasoningTreeID, ThemeID: v.ThemeID, IndustryChainEntityID: v.IndustryChainEntityID, IndustryChainName: v.IndustryChainName, Title: v.Title, DisplayOrder: v.DisplayOrder, OneLineConclusion: v.OneLineConclusion, FactSummary: v.FactSummary, TransmissionSummary: v.TransmissionSummary, ImpactDirection: v.ImpactDirection, ImpactStrength: v.ImpactStrength, ImpactSummary: v.ImpactSummary, ConclusionBoundarySummary: v.ConclusionBoundarySummary, SupportSummary: v.SupportSummary, CounterSummary: v.CounterSummary, InvalidationConditions: v.InvalidationConditions, Checkpoints: checkpoints, PublishedAt: v.PublishedAt, EventCount: v.EventCount, Events: mapSlice(v.Events, wireResearchEvent.toBiz), Nodes: mapSlice(v.Nodes, wireResearchReasoningTreeNode.toBiz)}
}
func (v wireResearchReasoningTreeDetail) toBiz() biz.ResearchReasoningTreeDetail {
	return biz.ResearchReasoningTreeDetail{ThemeID: v.ThemeID, ImpactNodeIDs: v.ImpactNodeIDs, ReasoningTree: v.ReasoningTree.toBiz()}
}

func mapSlice[From any, To any](values []From, convert func(From) To) []To {
	if values == nil {
		return nil
	}
	result := make([]To, 0, len(values))
	for _, value := range values {
		result = append(result, convert(value))
	}
	return result
}
