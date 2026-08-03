package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrResearchThemeNotFound          = errors.New("research Theme not found")
	ErrResearchReasoningTreesNotFound = errors.New("research reasoning trees not found")
	ErrResearchReasoningTreeNotFound  = errors.New("research reasoning tree not found")
	ErrResearchDataUnavailable        = errors.New("research data unavailable")
)

type ResearchReasoningTreeListResponse struct {
	Theme          ResearchThemeItem
	ReasoningTrees []ResearchReasoningTreeSummaryDTO
}
type ResearchReasoningTreeSummaryDTO struct {
	TreeKey, DisplayName                                             string
	ReasoningTreeID, IndustryChainEntityID, IndustryChainName, Title string
	DisplayOrder, EventCount                                         int
	PublishedAt                                                      string
}
type ResearchReasoningTreeDetailResponse struct {
	ThemeID       string
	ImpactNodeIDs []string
	ReasoningTree ResearchReasoningTreeDTO
}
type ResearchCheckpointDTO struct{ Type, Summary string }
type ResearchGraphEdgeDTO struct{ ID, RelationType, ReviewStatus, Status string }
type ResearchSignalDTO struct {
	SignalKey                                                      string
	VariableName, Direction                                        *string
	VariableSignalKey, SignalRole, SignalDirection, DisplaySummary string
	DisplayOrder                                                   int
}
type ResearchReasoningTreeNodeDTO struct {
	NodeKey, DisplayName                                         string
	ID, ChainNodeEntityID, Name, ImpactDirection, ImpactStrength string
	Position                                                     int
	StateSummary, ImpactSummary, ReasoningBasisSummary           *string
	EvidenceGapSummary                                           *string
	IncomingIndustryChainGraphEdgeID, IncomingTransmissionTitle  *string
	IncomingTransmissionMechanism, IncomingConditionSummary      *string
	IncomingGraphEdge                                            *ResearchGraphEdgeDTO
	Signals                                                      []ResearchSignalDTO
	PrimarySignal                                                ResearchSignalDTO
	SignalDisplaySummary                                         string
}
type ResearchReasoningTreeDTO struct {
	TreeKey, DisplayName                                               string
	ReasoningTreeID, ThemeID, IndustryChainEntityID, IndustryChainName string
	Title, OneLineConclusion, ImpactDirection, ImpactStrength          string
	DisplayOrder, EventCount                                           int
	FactSummary, TransmissionSummary, ImpactSummary                    *string
	ConclusionBoundarySummary, SupportSummary, CounterSummary          *string
	InvalidationConditions                                             []string
	Checkpoints                                                        []ResearchCheckpointDTO
	PublishedAt                                                        string
	Events                                                             []ResearchEventDTO
	Nodes                                                              []ResearchReasoningTreeNodeDTO
}

func (s *ResearchService) ListReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeListResponse, error) {
	themeID = strings.TrimSpace(themeID)
	if !researchUUIDPattern.MatchString(themeID) {
		return ResearchReasoningTreeListResponse{}, fmt.Errorf("%w: theme id must be a UUID", ErrInvalidResearchRequest)
	}
	if s == nil || s.repo == nil {
		return ResearchReasoningTreeListResponse{}, ErrResearchDataUnavailable
	}
	value, err := s.repo.ListResearchThemeReasoningTrees(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeListResponse{}, normalizeReasoningTreeRepoError(err)
	}
	trees := make([]ResearchReasoningTreeSummaryDTO, 0, len(value.ReasoningTrees))
	for _, tree := range value.ReasoningTrees {
		trees = append(trees, ResearchReasoningTreeSummaryDTO{
			TreeKey: tree.TreeKey, DisplayName: tree.DisplayName,
			ReasoningTreeID: tree.ReasoningTreeID, IndustryChainEntityID: tree.IndustryChainEntityID,
			IndustryChainName: tree.IndustryChainName, Title: tree.Title, DisplayOrder: tree.DisplayOrder,
			EventCount: tree.EventCount, PublishedAt: formatTime(tree.PublishedAt),
		})
	}
	return ResearchReasoningTreeListResponse{Theme: themeItemDTO(value.Theme), ReasoningTrees: trees}, nil
}

func (s *ResearchService) GetReasoningTree(ctx context.Context, themeID, treeID string) (ResearchReasoningTreeDetailResponse, error) {
	themeID, treeID = strings.TrimSpace(themeID), strings.TrimSpace(treeID)
	if !researchUUIDPattern.MatchString(themeID) || !researchUUIDPattern.MatchString(treeID) {
		return ResearchReasoningTreeDetailResponse{}, fmt.Errorf("%w: resource id must be a UUID", ErrInvalidResearchRequest)
	}
	if s == nil || s.repo == nil {
		return ResearchReasoningTreeDetailResponse{}, ErrResearchDataUnavailable
	}
	value, err := s.repo.GetResearchThemeReasoningTree(ctx, themeID, treeID)
	if err != nil {
		return ResearchReasoningTreeDetailResponse{}, normalizeReasoningTreeRepoError(err)
	}
	return ResearchReasoningTreeDetailResponse{
		ThemeID: value.ThemeID, ImpactNodeIDs: append([]string(nil), value.ImpactNodeIDs...),
		ReasoningTree: reasoningTreeDTO(value.ReasoningTree),
	}, nil
}

func reasoningTreeDTO(value ResearchReasoningTree) ResearchReasoningTreeDTO {
	checkpoints := make([]ResearchCheckpointDTO, 0, len(value.Checkpoints))
	for _, checkpoint := range value.Checkpoints {
		checkpoints = append(checkpoints, ResearchCheckpointDTO{Type: checkpoint.Type, Summary: checkpoint.Summary})
	}
	nodes := make([]ResearchReasoningTreeNodeDTO, 0, len(value.Nodes))
	for _, node := range value.Nodes {
		signals := make([]ResearchSignalDTO, 0, len(node.Signals))
		for _, signal := range node.Signals {
			signals = append(signals, signalDTO(signal))
		}
		var edge *ResearchGraphEdgeDTO
		if node.IncomingGraphEdge != nil {
			edge = &ResearchGraphEdgeDTO{ID: node.IncomingGraphEdge.ID, RelationType: node.IncomingGraphEdge.RelationType, ReviewStatus: node.IncomingGraphEdge.ReviewStatus, Status: node.IncomingGraphEdge.Status}
		}
		nodes = append(nodes, ResearchReasoningTreeNodeDTO{
			NodeKey: node.NodeKey, DisplayName: node.DisplayName,
			ID: node.ID, Position: node.Position, ChainNodeEntityID: node.ChainNodeEntityID, Name: node.Name,
			StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection, ImpactStrength: node.ImpactStrength,
			ImpactSummary: node.ImpactSummary, ReasoningBasisSummary: node.ReasoningBasisSummary,
			EvidenceGapSummary: node.EvidenceGapSummary, IncomingIndustryChainGraphEdgeID: node.IncomingIndustryChainGraphEdgeID,
			IncomingTransmissionTitle: node.IncomingTransmissionTitle, IncomingTransmissionMechanism: node.IncomingTransmissionMechanism,
			IncomingConditionSummary: node.IncomingConditionSummary, IncomingGraphEdge: edge, Signals: signals,
			PrimarySignal: signalDTO(node.PrimarySignal), SignalDisplaySummary: node.SignalDisplaySummary,
		})
	}
	return ResearchReasoningTreeDTO{
		TreeKey: value.TreeKey, DisplayName: value.DisplayName,
		ReasoningTreeID: value.ReasoningTreeID, ThemeID: value.ThemeID,
		IndustryChainEntityID: value.IndustryChainEntityID, IndustryChainName: value.IndustryChainName,
		Title: value.Title, DisplayOrder: value.DisplayOrder, OneLineConclusion: value.OneLineConclusion,
		FactSummary: value.FactSummary, TransmissionSummary: value.TransmissionSummary,
		ImpactDirection: value.ImpactDirection, ImpactStrength: value.ImpactStrength, ImpactSummary: value.ImpactSummary,
		ConclusionBoundarySummary: value.ConclusionBoundarySummary, SupportSummary: value.SupportSummary,
		CounterSummary: value.CounterSummary, InvalidationConditions: append([]string(nil), value.InvalidationConditions...),
		Checkpoints: checkpoints, PublishedAt: formatTime(value.PublishedAt), EventCount: value.EventCount,
		Events: eventDTOs(value.Events), Nodes: nodes,
	}
}
func signalDTO(value ResearchSignal) ResearchSignalDTO {
	return ResearchSignalDTO{SignalKey: value.SignalKey, VariableName: value.VariableName, Direction: value.Direction, VariableSignalKey: value.VariableSignalKey, SignalRole: value.SignalRole, SignalDirection: value.SignalDirection, DisplaySummary: value.DisplaySummary, DisplayOrder: value.DisplayOrder}
}
func normalizeReasoningTreeRepoError(err error) error {
	switch {
	case errors.Is(err, ErrResearchThemeNotFound):
		return ErrResearchThemeNotFound
	case errors.Is(err, ErrResearchReasoningTreesNotFound):
		return ErrResearchReasoningTreesNotFound
	case errors.Is(err, ErrResearchReasoningTreeNotFound):
		return ErrResearchReasoningTreeNotFound
	default:
		return ErrResearchDataUnavailable
	}
}
