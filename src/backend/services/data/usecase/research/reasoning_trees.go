package research

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

var (
	ErrThemeNotFound                   = errors.New("research theme not found")
	ErrReasoningTreesNotFound          = errors.New("research reasoning trees not found")
	ErrReasoningTreeNotFound           = errors.New("research reasoning tree not found")
	ErrReasoningTreeInvariantViolation = errors.New("research reasoning tree invariant violation")
)

type ResearchReasoningTreeChainNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResearchReasoningTreeSummary struct {
	AnchorID        string                         `json:"anchor_id"`
	CenterChainNode ResearchReasoningTreeChainNode `json:"center_chain_node"`
}

type ResearchReasoningTreeList struct {
	Theme          ResearchTheme                  `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type ResearchReasoningTreeEvent struct {
	EventID         string     `json:"event_id"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	EventTime       *time.Time `json:"event_time,omitempty"`
	EvidenceRole    string     `json:"evidence_role"`
	EvidenceSummary string     `json:"evidence_summary"`
}

type ResearchReasoningTreePathNode struct {
	ChainNodeID                   string  `json:"chain_node_id"`
	Name                          string  `json:"name"`
	ChangeDirection               string  `json:"change_direction"`
	ChangeSummary                 string  `json:"change_summary"`
	ImpactSummary                 string  `json:"impact_summary"`
	IncomingTransmissionMechanism *string `json:"incoming_transmission_mechanism"`
}

type ResearchReasoningTree struct {
	AnchorID            string                          `json:"anchor_id"`
	CenterChainNode     ResearchReasoningTreeChainNode  `json:"center_chain_node"`
	OneLineConclusion   string                          `json:"one_line_conclusion"`
	FactSummary         string                          `json:"fact_summary"`
	NetDirectionSummary string                          `json:"net_direction_summary"`
	TradingDirection    string                          `json:"trading_direction"`
	NextCheckpoint      string                          `json:"next_checkpoint"`
	EventCount          int                             `json:"event_count"`
	Events              []ResearchReasoningTreeEvent    `json:"events"`
	PathNodes           []ResearchReasoningTreePathNode `json:"path_nodes"`
}

type ResearchReasoningTreeDetail struct {
	ThemeID       string                `json:"theme_id"`
	ReasoningTree ResearchReasoningTree `json:"reasoning_tree"`
}

func (s *Service) ListReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeList, error) {
	themeID = strings.ToLower(strings.TrimSpace(themeID))
	if !researchUUIDPattern.MatchString(themeID) {
		return ResearchReasoningTreeList{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidRequest)
	}
	result, err := s.repository.ListResearchThemeReasoningTrees(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeList{}, mapReasoningTreeRepositoryError(err)
	}
	trees := make([]ResearchReasoningTreeSummary, 0, len(result.ReasoningTrees))
	for _, tree := range result.ReasoningTrees {
		trees = append(trees, ResearchReasoningTreeSummary{
			AnchorID:        tree.AnchorID,
			CenterChainNode: ResearchReasoningTreeChainNode{ID: tree.CenterChainNodeID, Name: tree.CenterChainNodeName},
		})
	}
	return ResearchReasoningTreeList{Theme: themeDTO(result.Theme), ReasoningTrees: trees}, nil
}

func (s *Service) GetReasoningTree(ctx context.Context, themeID, anchorID string) (ResearchReasoningTreeDetail, error) {
	themeID = strings.ToLower(strings.TrimSpace(themeID))
	anchorID = strings.ToLower(strings.TrimSpace(anchorID))
	if !researchUUIDPattern.MatchString(themeID) {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidRequest)
	}
	if !researchUUIDPattern.MatchString(anchorID) {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("%w: anchor id must be a UUID", ErrInvalidRequest)
	}
	result, err := s.repository.GetResearchThemeReasoningTree(ctx, themeID, anchorID)
	if err != nil {
		return ResearchReasoningTreeDetail{}, mapReasoningTreeRepositoryError(err)
	}
	return ResearchReasoningTreeDetail{ThemeID: result.ThemeID, ReasoningTree: reasoningTreeDTO(result.ReasoningTree)}, nil
}

func reasoningTreeDTO(value repositories.ResearchReasoningTree) ResearchReasoningTree {
	events := make([]ResearchReasoningTreeEvent, 0, len(value.Events))
	for _, event := range value.Events {
		var eventTime *time.Time
		if event.EventTime != nil {
			formatted := event.EventTime.UTC()
			eventTime = &formatted
		}
		events = append(events, ResearchReasoningTreeEvent{
			EventID: event.EventID, Title: event.Title, Summary: event.Summary, EventTime: eventTime,
			EvidenceRole: event.EvidenceRole, EvidenceSummary: event.EvidenceSummary,
		})
	}
	pathNodes := make([]ResearchReasoningTreePathNode, 0, len(value.PathNodes))
	for _, node := range value.PathNodes {
		pathNodes = append(pathNodes, ResearchReasoningTreePathNode{
			ChainNodeID: node.ChainNodeID, Name: node.Name, ChangeDirection: node.ChangeDirection,
			ChangeSummary: node.ChangeSummary, ImpactSummary: node.ImpactSummary,
			IncomingTransmissionMechanism: node.IncomingTransmissionMechanism,
		})
	}
	return ResearchReasoningTree{
		AnchorID:          value.AnchorID,
		CenterChainNode:   ResearchReasoningTreeChainNode{ID: value.CenterChainNodeID, Name: value.CenterChainNodeName},
		OneLineConclusion: value.OneLineConclusion, FactSummary: value.FactSummary,
		NetDirectionSummary: value.NetDirectionSummary, TradingDirection: value.TradingDirection,
		NextCheckpoint: value.NextCheckpoint, EventCount: len(events), Events: events, PathNodes: pathNodes,
	}
}

func mapReasoningTreeRepositoryError(err error) error {
	switch {
	case errors.Is(err, repositories.ErrResearchThemeNotFound):
		return ErrThemeNotFound
	case errors.Is(err, repositories.ErrResearchReasoningTreesNotFound):
		return ErrReasoningTreesNotFound
	case errors.Is(err, repositories.ErrResearchReasoningTreeNotFound):
		return ErrReasoningTreeNotFound
	case errors.Is(err, repositories.ErrResearchReasoningTreeInvariant):
		return ErrReasoningTreeInvariantViolation
	default:
		return fmt.Errorf("%w: %v", ErrRepository, err)
	}
}
