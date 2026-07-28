package research

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrThemeNotFound                   = errors.New("research Theme not found")
	ErrReasoningTreesNotFound          = errors.New("research Theme has no published reasoning trees")
	ErrReasoningTreeNotFound           = errors.New("research reasoning tree not found")
	ErrReasoningTreeInvariantViolation = errors.New("research reasoning tree invariant violation")
)

type ResearchReasoningTreeList struct {
	Theme          ResearchTheme                  `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummary `json:"reasoning_trees"`
}

type ResearchReasoningTreeSummary struct {
	ReasoningTreeID       string    `json:"reasoning_tree_id"`
	IndustryChainEntityID string    `json:"industry_chain_entity_id"`
	IndustryChainName     string    `json:"industry_chain_name"`
	Title                 string    `json:"title"`
	DisplayOrder          int       `json:"display_order"`
	EventCount            int       `json:"event_count"`
	PublishedAt           time.Time `json:"published_at"`
}

type ResearchCheckpoint struct {
	Type, Summary string
}

type ResearchGraphEdge struct {
	ID, RelationType, ReviewStatus, Status string
}

type ResearchSignal struct {
	VariableSignalKey string `json:"variable_signal_key"`
	SignalRole        string `json:"signal_role"`
	SignalDirection   string `json:"signal_direction"`
	DisplaySummary    string `json:"display_summary"`
	DisplayOrder      int    `json:"display_order"`
}

type ResearchReasoningTreeNode struct {
	ID                               string             `json:"id"`
	Position                         int                `json:"position"`
	ChainNodeEntityID                string             `json:"chain_node_entity_id"`
	Name                             string             `json:"name"`
	StateSummary                     *string            `json:"state_summary"`
	ImpactDirection                  string             `json:"impact_direction"`
	ImpactStrength                   string             `json:"impact_strength"`
	ImpactSummary                    *string            `json:"impact_summary"`
	ReasoningBasisSummary            *string            `json:"reasoning_basis_summary"`
	EvidenceGapSummary               *string            `json:"evidence_gap_summary"`
	IncomingIndustryChainGraphEdgeID *string            `json:"incoming_industry_chain_graph_edge_id"`
	IncomingTransmissionTitle        *string            `json:"incoming_transmission_title"`
	IncomingTransmissionMechanism    *string            `json:"incoming_transmission_mechanism"`
	IncomingConditionSummary         *string            `json:"incoming_condition_summary"`
	IncomingGraphEdge                *ResearchGraphEdge `json:"incoming_graph_edge"`
	Signals                          []ResearchSignal   `json:"signals"`
	PrimarySignal                    ResearchSignal     `json:"primary_signal"`
	SignalDisplaySummary             string             `json:"signal_display_summary"`
}

type ResearchReasoningTree struct {
	ReasoningTreeID, ThemeID, IndustryChainEntityID, IndustryChainName string
	Title, OneLineConclusion, ImpactDirection, ImpactStrength          string
	DisplayOrder, EventCount                                           int
	FactSummary, TransmissionSummary, ImpactSummary                    *string
	ConclusionBoundarySummary, SupportSummary, CounterSummary          *string
	InvalidationConditions                                             []string
	Checkpoints                                                        []ResearchCheckpoint
	PublishedAt                                                        time.Time
	Events                                                             []ResearchEvent
	Nodes                                                              []ResearchReasoningTreeNode
}

type ResearchReasoningTreeDetail struct {
	ThemeID       string
	ImpactNodeIDs []string
	ReasoningTree ResearchReasoningTree
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
	summaries := make([]ResearchReasoningTreeSummary, 0, len(result.ReasoningTrees))
	for _, value := range result.ReasoningTrees {
		summaries = append(summaries, ResearchReasoningTreeSummary{
			ReasoningTreeID:       value.ReasoningTreeID,
			IndustryChainEntityID: value.IndustryChainEntityID,
			IndustryChainName:     value.IndustryChainName, Title: value.Title,
			DisplayOrder: value.DisplayOrder, EventCount: value.EventCount,
			PublishedAt: value.PublishedAt.UTC(),
		})
	}
	return ResearchReasoningTreeList{Theme: themeDTO(result.Theme), ReasoningTrees: summaries}, nil
}

func (s *Service) GetReasoningTree(ctx context.Context, themeID, reasoningTreeID string) (ResearchReasoningTreeDetail, error) {
	themeID = strings.ToLower(strings.TrimSpace(themeID))
	reasoningTreeID = strings.ToLower(strings.TrimSpace(reasoningTreeID))
	if !researchUUIDPattern.MatchString(themeID) {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidRequest)
	}
	if !researchUUIDPattern.MatchString(reasoningTreeID) {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("%w: reasoning tree id must be a UUID", ErrInvalidRequest)
	}
	result, err := s.repository.GetResearchThemeReasoningTree(ctx, themeID, reasoningTreeID)
	if err != nil {
		return ResearchReasoningTreeDetail{}, mapReasoningTreeRepositoryError(err)
	}
	tree := result.ReasoningTree
	nodes := make([]ResearchReasoningTreeNode, 0, len(tree.Nodes))
	for _, node := range tree.Nodes {
		signals := make([]ResearchSignal, 0, len(node.Signals))
		var primary ResearchSignal
		secondary := make([]string, 0, len(node.Signals)-1)
		for _, signal := range node.Signals {
			item := ResearchSignal{
				VariableSignalKey: signal.VariableSignalKey, SignalRole: signal.SignalRole,
				SignalDirection: signal.SignalDirection, DisplaySummary: signal.DisplaySummary,
				DisplayOrder: signal.DisplayOrder,
			}
			signals = append(signals, item)
			if signal.SignalRole == "primary" {
				primary = item
			} else {
				secondary = append(secondary, signal.DisplaySummary)
			}
		}
		var graphEdge *ResearchGraphEdge
		if node.IncomingGraphEdge != nil {
			graphEdge = &ResearchGraphEdge{
				ID: node.IncomingGraphEdge.ID, RelationType: node.IncomingGraphEdge.RelationType,
				ReviewStatus: node.IncomingGraphEdge.ReviewStatus, Status: node.IncomingGraphEdge.Status,
			}
		}
		nodes = append(nodes, ResearchReasoningTreeNode{
			ID: node.ID, Position: node.Position, ChainNodeEntityID: node.ChainNodeEntityID,
			Name: node.Name, StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection,
			ImpactStrength: node.ImpactStrength, ImpactSummary: node.ImpactSummary,
			ReasoningBasisSummary: node.ReasoningBasisSummary, EvidenceGapSummary: node.EvidenceGapSummary,
			IncomingIndustryChainGraphEdgeID: node.IncomingIndustryChainGraphEdgeID,
			IncomingTransmissionTitle:        node.IncomingTransmissionTitle,
			IncomingTransmissionMechanism:    node.IncomingTransmissionMechanism,
			IncomingConditionSummary:         node.IncomingConditionSummary, IncomingGraphEdge: graphEdge,
			Signals: signals, PrimarySignal: primary, SignalDisplaySummary: strings.Join(secondary, " · "),
		})
	}
	checkpoints := make([]ResearchCheckpoint, 0, len(tree.Checkpoints))
	for _, checkpoint := range tree.Checkpoints {
		checkpoints = append(checkpoints, ResearchCheckpoint{Type: checkpoint.Type, Summary: checkpoint.Summary})
	}
	return ResearchReasoningTreeDetail{
		ThemeID: result.ThemeID, ImpactNodeIDs: append([]string(nil), result.ImpactNodeIDs...),
		ReasoningTree: ResearchReasoningTree{
			ReasoningTreeID: tree.ReasoningTreeID, ThemeID: tree.ThemeID,
			IndustryChainEntityID: tree.IndustryChainEntityID, IndustryChainName: tree.IndustryChainName,
			Title: tree.Title, DisplayOrder: tree.DisplayOrder, OneLineConclusion: tree.OneLineConclusion,
			FactSummary: tree.FactSummary, TransmissionSummary: tree.TransmissionSummary,
			ImpactDirection: tree.ImpactDirection, ImpactStrength: tree.ImpactStrength,
			ImpactSummary: tree.ImpactSummary, ConclusionBoundarySummary: tree.ConclusionBoundarySummary,
			SupportSummary: tree.SupportSummary, CounterSummary: tree.CounterSummary,
			InvalidationConditions: append([]string(nil), tree.InvalidationConditions...),
			Checkpoints:            checkpoints, PublishedAt: tree.PublishedAt.UTC(),
			EventCount: tree.EventCount, Events: eventDTOs(tree.Events), Nodes: nodes,
		},
	}, nil
}

func mapReasoningTreeRepositoryError(err error) error {
	switch {
	case errors.Is(err, ErrResearchThemeNotFound):
		return ErrThemeNotFound
	case errors.Is(err, ErrResearchReasoningTreesNotFound):
		return ErrReasoningTreesNotFound
	case errors.Is(err, ErrResearchReasoningTreeNotFound):
		return ErrReasoningTreeNotFound
	case errors.Is(err, ErrResearchReasoningTreeInvariant):
		return ErrReasoningTreeInvariantViolation
	default:
		return fmt.Errorf("%w: %v", ErrRepository, err)
	}
}
