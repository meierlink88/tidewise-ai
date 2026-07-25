package data

import (
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
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
	ID                        string                       `json:"id"`
	Name                      string                       `json:"name"`
	OneLineConclusion         string                       `json:"one_line_conclusion"`
	ImpactLevel               string                       `json:"impact_level"`
	TransmissionPath          string                       `json:"transmission_path"`
	TradingDirection          string                       `json:"trading_direction"`
	TransmissionStage         string                       `json:"transmission_stage"`
	NextCheckpoint            string                       `json:"next_checkpoint"`
	MarketConfirmationSummary string                       `json:"market_confirmation_summary"`
	PublishedAt               time.Time                    `json:"published_at"`
	AffectedChainNodes        []wireResearchThemeChainNode `json:"affected_chain_nodes"`
	RelatedIndices            []wireResearchIndex          `json:"related_indices"`
	SupportingEventCount      int                          `json:"supporting_event_count"`
	ContradictingEventCount   int                          `json:"contradicting_event_count"`
}

type wireResearchThemeDetail struct {
	Theme  wireResearchTheme   `json:"theme"`
	Events []wireResearchEvent `json:"events"`
}

type wireResearchThemeChainNode struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RelationRole  string `json:"relation_role"`
	ImpactSummary string `json:"impact_summary"`
}

type wireResearchIndex struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImpactDirection string `json:"impact_direction"`
	ImpactSummary   string `json:"impact_summary"`
}

type wireResearchEvent struct {
	EventID        string     `json:"event_id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	EventTime      *time.Time `json:"event_time,omitempty"`
	EvidenceRole   string     `json:"evidence_role"`
	SupportedClaim string     `json:"supported_claim"`
}

type wireResearchReasoningTreeChainNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type wireResearchReasoningTreeSummary struct {
	AnchorID        string                             `json:"anchor_id"`
	CenterChainNode wireResearchReasoningTreeChainNode `json:"center_chain_node"`
}

type wireResearchReasoningTreeList struct {
	Theme          wireResearchTheme                  `json:"theme"`
	ReasoningTrees []wireResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type wireResearchReasoningTreeEvent struct {
	EventID         string     `json:"event_id"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	EventTime       *time.Time `json:"event_time,omitempty"`
	EvidenceRole    string     `json:"evidence_role"`
	EvidenceSummary string     `json:"evidence_summary"`
}

type wireResearchReasoningTreePathNode struct {
	ChainNodeID                   string  `json:"chain_node_id"`
	Name                          string  `json:"name"`
	ChangeDirection               string  `json:"change_direction"`
	ChangeSummary                 string  `json:"change_summary"`
	ImpactSummary                 string  `json:"impact_summary"`
	IncomingTransmissionMechanism *string `json:"incoming_transmission_mechanism"`
}

type wireResearchReasoningTree struct {
	AnchorID            string                              `json:"anchor_id"`
	CenterChainNode     wireResearchReasoningTreeChainNode  `json:"center_chain_node"`
	OneLineConclusion   string                              `json:"one_line_conclusion"`
	FactSummary         string                              `json:"fact_summary"`
	NetDirectionSummary string                              `json:"net_direction_summary"`
	SupportSummary      string                              `json:"support_summary"`
	CounterSummary      *string                             `json:"counter_summary"`
	TradingDirection    string                              `json:"trading_direction"`
	NextCheckpoint      string                              `json:"next_checkpoint"`
	EventCount          int                                 `json:"event_count"`
	Events              []wireResearchReasoningTreeEvent    `json:"events"`
	PathNodes           []wireResearchReasoningTreePathNode `json:"path_nodes"`
}

type wireResearchReasoningTreeDetail struct {
	ThemeID       string                    `json:"theme_id"`
	ReasoningTree wireResearchReasoningTree `json:"reasoning_tree"`
}

func (value wireResearchThemePage) toBiz() biz.ResearchThemePage {
	return biz.ResearchThemePage{
		WindowStart: value.WindowStart,
		WindowEnd:   value.WindowEnd,
		AsOf:        value.AsOf,
		ThemeCount:  value.ThemeCount,
		EventCount:  value.EventCount,
		Items:       mapSlice(value.Items, wireResearchTheme.toBiz),
		NextCursor:  value.NextCursor,
	}
}

func (value wireResearchTheme) toBiz() biz.ResearchTheme {
	return biz.ResearchTheme{
		ID:                        value.ID,
		Name:                      value.Name,
		OneLineConclusion:         value.OneLineConclusion,
		ImpactLevel:               biz.ImpactLevel(value.ImpactLevel),
		TransmissionPath:          value.TransmissionPath,
		TradingDirection:          value.TradingDirection,
		TransmissionStage:         biz.TransmissionStage(value.TransmissionStage),
		NextCheckpoint:            value.NextCheckpoint,
		MarketConfirmationSummary: value.MarketConfirmationSummary,
		PublishedAt:               value.PublishedAt,
		AffectedChainNodes:        mapSlice(value.AffectedChainNodes, wireResearchThemeChainNode.toBiz),
		RelatedIndices:            mapSlice(value.RelatedIndices, wireResearchIndex.toBiz),
		SupportingEventCount:      value.SupportingEventCount,
		ContradictingEventCount:   value.ContradictingEventCount,
	}
}

func (value wireResearchThemeDetail) toBiz() biz.ResearchThemeDetail {
	return biz.ResearchThemeDetail{
		Theme:  value.Theme.toBiz(),
		Events: mapSlice(value.Events, wireResearchEvent.toBiz),
	}
}

func (value wireResearchThemeChainNode) toBiz() biz.ResearchThemeChainNode {
	return biz.ResearchThemeChainNode{
		ID: value.ID, Name: value.Name, RelationRole: value.RelationRole, ImpactSummary: value.ImpactSummary,
	}
}

func (value wireResearchIndex) toBiz() biz.ResearchIndex {
	return biz.ResearchIndex{
		ID: value.ID, Name: value.Name, ImpactDirection: biz.ImpactDirection(value.ImpactDirection),
		ImpactSummary: value.ImpactSummary,
	}
}

func (value wireResearchEvent) toBiz() biz.ResearchEvent {
	return biz.ResearchEvent{
		EventID: value.EventID, Title: value.Title, Summary: value.Summary, EventTime: value.EventTime,
		EvidenceRole: biz.EvidenceRole(value.EvidenceRole), SupportedClaim: value.SupportedClaim,
	}
}

func (value wireResearchReasoningTreeChainNode) toBiz() biz.ResearchReasoningTreeChainNode {
	return biz.ResearchReasoningTreeChainNode{ID: value.ID, Name: value.Name}
}

func (value wireResearchReasoningTreeSummary) toBiz() biz.ResearchReasoningTreeSummary {
	return biz.ResearchReasoningTreeSummary{
		AnchorID: value.AnchorID, CenterChainNode: value.CenterChainNode.toBiz(),
	}
}

func (value wireResearchReasoningTreeList) toBiz() biz.ResearchReasoningTreeList {
	return biz.ResearchReasoningTreeList{
		Theme: value.Theme.toBiz(), ReasoningTrees: mapSlice(value.ReasoningTrees, wireResearchReasoningTreeSummary.toBiz),
	}
}

func (value wireResearchReasoningTreeEvent) toBiz() biz.ResearchReasoningTreeEvent {
	return biz.ResearchReasoningTreeEvent{
		EventID: value.EventID, Title: value.Title, Summary: value.Summary, EventTime: value.EventTime,
		EvidenceRole: biz.EvidenceRole(value.EvidenceRole), EvidenceSummary: value.EvidenceSummary,
	}
}

func (value wireResearchReasoningTreePathNode) toBiz() biz.ResearchReasoningTreePathNode {
	return biz.ResearchReasoningTreePathNode{
		ChainNodeID: value.ChainNodeID, Name: value.Name,
		ChangeDirection: biz.ChangeDirection(value.ChangeDirection),
		ChangeSummary:   value.ChangeSummary, ImpactSummary: value.ImpactSummary,
		IncomingTransmissionMechanism: value.IncomingTransmissionMechanism,
	}
}

func (value wireResearchReasoningTree) toBiz() biz.ResearchReasoningTree {
	return biz.ResearchReasoningTree{
		AnchorID: value.AnchorID, CenterChainNode: value.CenterChainNode.toBiz(),
		OneLineConclusion: value.OneLineConclusion, FactSummary: value.FactSummary,
		NetDirectionSummary: value.NetDirectionSummary, SupportSummary: value.SupportSummary,
		CounterSummary: value.CounterSummary, TradingDirection: value.TradingDirection,
		NextCheckpoint: value.NextCheckpoint, EventCount: value.EventCount,
		Events:    mapSlice(value.Events, wireResearchReasoningTreeEvent.toBiz),
		PathNodes: mapSlice(value.PathNodes, wireResearchReasoningTreePathNode.toBiz),
	}
}

func (value wireResearchReasoningTreeDetail) toBiz() biz.ResearchReasoningTreeDetail {
	return biz.ResearchReasoningTreeDetail{
		ThemeID: value.ThemeID, ReasoningTree: value.ReasoningTree.toBiz(),
	}
}

func mapSlice[From any, To any](values []From, convert func(From) To) []To {
	result := make([]To, 0, len(values))
	for _, value := range values {
		result = append(result, convert(value))
	}
	return result
}
