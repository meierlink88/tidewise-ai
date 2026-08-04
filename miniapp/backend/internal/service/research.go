package service

import (
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
)

var _ v1.ResearchHTTPServer = (*ResearchService)(nil)

type ResearchService struct{ research *biz.ResearchService }

func NewResearchService(research *biz.ResearchService) *ResearchService {
	return &ResearchService{research: research}
}

func (s *ResearchService) ListResearchThemes(ctx context.Context, request *v1.ListResearchThemesRequest) (*v1.ResearchThemeListResponse, error) {
	if s == nil || s.research == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	value, err := s.research.ListThemes(ctx, biz.ResearchListRequest{Period: request.Period, WindowHours: request.WindowHours, Limit: request.Limit, Cursor: request.Cursor})
	if err != nil {
		return nil, mapBizError(err)
	}
	items := make([]v1.ResearchThemeItem, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, themeItem(item))
	}
	return &v1.ResearchThemeListResponse{WindowStart: value.WindowStart, WindowEnd: value.WindowEnd, AsOf: value.AsOf, ThemeCount: value.ThemeCount, EventCount: value.EventCount, Items: items, NextCursor: value.NextCursor}, nil
}
func (s *ResearchService) GetResearchTheme(ctx context.Context, request *v1.GetResearchThemeRequest) (*v1.ResearchThemeDetailResponse, error) {
	if s == nil || s.research == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	value, err := s.research.GetTheme(ctx, request.ThemeID, biz.ResearchDetailRequest{WindowHours: request.WindowHours})
	if err != nil {
		return nil, mapBizError(err)
	}
	return &v1.ResearchThemeDetailResponse{ResearchThemeItem: themeItem(value.ResearchThemeItem), Events: events(value.Events)}, nil
}
func (s *ResearchService) ListResearchThemeReasoningTrees(ctx context.Context, request *v1.ListResearchThemeReasoningTreesRequest) (*v1.ResearchReasoningTreeListResponse, error) {
	if s == nil || s.research == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	value, err := s.research.ListReasoningTrees(ctx, request.ThemeID)
	if err != nil {
		return nil, mapBizError(err)
	}
	trees := make([]v1.ResearchReasoningTreeSummary, 0, len(value.ReasoningTrees))
	for _, tree := range value.ReasoningTrees {
		trees = append(trees, v1.ResearchReasoningTreeSummary{TreeKey: tree.TreeKey, DisplayName: tree.DisplayName, ReasoningTreeID: tree.ReasoningTreeID, IndustryChainEntityID: tree.IndustryChainEntityID, IndustryChainName: tree.IndustryChainName, Title: tree.Title, DisplayOrder: tree.DisplayOrder, EventCount: tree.EventCount, PublishedAt: tree.PublishedAt})
	}
	return &v1.ResearchReasoningTreeListResponse{Theme: themeItem(value.Theme), ReasoningTrees: trees}, nil
}
func (s *ResearchService) GetResearchThemeReasoningTree(ctx context.Context, request *v1.GetResearchThemeReasoningTreeRequest) (*v1.ResearchReasoningTreeDetailResponse, error) {
	if s == nil || s.research == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	value, err := s.research.GetReasoningTree(ctx, request.ThemeID, request.ReasoningTreeID)
	if err != nil {
		return nil, mapBizError(err)
	}
	return &v1.ResearchReasoningTreeDetailResponse{ThemeID: value.ThemeID, ImpactNodeIDs: value.ImpactNodeIDs, ReasoningTree: reasoningTree(value.ReasoningTree)}, nil
}

func themeItem(value biz.ResearchThemeItem) v1.ResearchThemeItem {
	impacts := make([]v1.ResearchThemeImpact, 0, len(value.Impacts))
	for _, impact := range value.Impacts {
		impacts = append(impacts, v1.ResearchThemeImpact{NodeKey: impact.NodeKey, DisplayName: impact.DisplayName, ChainNodeEntityID: impact.ChainNodeEntityID, Name: impact.Name, RelationRole: impact.RelationRole, ImpactDirection: impact.ImpactDirection, ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder})
	}
	return v1.ResearchThemeItem{ID: value.ID, AnalysisBatchID: value.AnalysisBatchID, Title: value.Title, OneLineConclusion: value.OneLineConclusion, ConclusionDirection: value.ConclusionDirection, ImpactStrength: value.ImpactStrength, AttentionLevel: value.AttentionLevel, ConclusionStatus: value.ConclusionStatus, TransmissionStage: value.TransmissionStage, InvestmentGuidanceAction: value.InvestmentGuidanceAction, InvestmentGuidanceSummary: value.InvestmentGuidanceSummary, TimeHorizonCategory: value.TimeHorizonCategory, TimeHorizonSummary: value.TimeHorizonSummary, TransmissionSummary: value.TransmissionSummary, CheckpointSummary: value.CheckpointSummary, RiskSummary: value.RiskSummary, AnalysisAsOf: value.AnalysisAsOf, WindowStart: value.WindowStart, WindowEnd: value.WindowEnd, PublishedAt: value.PublishedAt, Impacts: impacts, EvidenceEventCount: value.EvidenceEventCount, ReasoningTreeCount: value.ReasoningTreeCount}
}
func events(values []biz.ResearchEventDTO) []v1.ResearchEvent {
	result := make([]v1.ResearchEvent, 0, len(values))
	for _, e := range values {
		result = append(result, v1.ResearchEvent{EventID: e.EventID, EvidenceIDs: append([]string(nil), e.EvidenceIDs...), Title: e.Title, Summary: e.Summary, EventTime: e.EventTime, EvidenceRole: e.EvidenceRole, SupportedClaim: e.SupportedClaim, DisplayOrder: e.DisplayOrder})
	}
	return result
}
func signal(value biz.ResearchSignalDTO) v1.ResearchReasoningTreeSignal {
	return v1.ResearchReasoningTreeSignal{SignalKey: value.SignalKey, VariableName: value.VariableName, Direction: value.Direction, VariableSignalKey: value.VariableSignalKey, SignalRole: value.SignalRole, SignalDirection: value.SignalDirection, DisplaySummary: value.DisplaySummary, DisplayOrder: value.DisplayOrder}
}
func reasoningTree(value biz.ResearchReasoningTreeDTO) v1.ResearchReasoningTree {
	checkpoints := make([]v1.ResearchReasoningTreeCheckpoint, 0, len(value.Checkpoints))
	for _, c := range value.Checkpoints {
		checkpoints = append(checkpoints, v1.ResearchReasoningTreeCheckpoint{Type: c.Type, Summary: c.Summary})
	}
	nodes := make([]v1.ResearchReasoningTreeNode, 0, len(value.Nodes))
	for _, n := range value.Nodes {
		signals := make([]v1.ResearchReasoningTreeSignal, 0, len(n.Signals))
		for _, s := range n.Signals {
			signals = append(signals, signal(s))
		}
		var edge *v1.ResearchReasoningTreeGraphEdge
		if n.IncomingGraphEdge != nil {
			edge = &v1.ResearchReasoningTreeGraphEdge{ID: n.IncomingGraphEdge.ID, RelationType: n.IncomingGraphEdge.RelationType, ReviewStatus: n.IncomingGraphEdge.ReviewStatus, Status: n.IncomingGraphEdge.Status}
		}
		nodes = append(nodes, v1.ResearchReasoningTreeNode{NodeKey: n.NodeKey, DisplayName: n.DisplayName, ID: n.ID, Position: n.Position, ChainNodeEntityID: n.ChainNodeEntityID, Name: n.Name, StateSummary: n.StateSummary, ImpactDirection: n.ImpactDirection, ImpactStrength: n.ImpactStrength, ImpactSummary: n.ImpactSummary, ReasoningBasisSummary: n.ReasoningBasisSummary, EvidenceGapSummary: n.EvidenceGapSummary, IncomingIndustryChainGraphEdgeID: n.IncomingIndustryChainGraphEdgeID, IncomingTransmissionTitle: n.IncomingTransmissionTitle, IncomingTransmissionMechanism: n.IncomingTransmissionMechanism, IncomingConditionSummary: n.IncomingConditionSummary, IncomingGraphEdge: edge, Signals: signals, PrimarySignal: signal(n.PrimarySignal), SignalDisplaySummary: n.SignalDisplaySummary})
	}
	return v1.ResearchReasoningTree{TreeKey: value.TreeKey, DisplayName: value.DisplayName, ReasoningTreeID: value.ReasoningTreeID, ThemeID: value.ThemeID, IndustryChainEntityID: value.IndustryChainEntityID, IndustryChainName: value.IndustryChainName, Title: value.Title, DisplayOrder: value.DisplayOrder, OneLineConclusion: value.OneLineConclusion, FactSummary: value.FactSummary, TransmissionSummary: value.TransmissionSummary, ImpactDirection: value.ImpactDirection, ImpactStrength: value.ImpactStrength, ImpactSummary: value.ImpactSummary, ConclusionBoundarySummary: value.ConclusionBoundarySummary, SupportSummary: value.SupportSummary, CounterSummary: value.CounterSummary, InvalidationConditions: value.InvalidationConditions, Checkpoints: checkpoints, PublishedAt: value.PublishedAt, EventCount: value.EventCount, Events: events(value.Events), Nodes: nodes}
}
func mapBizError(err error) error {
	switch {
	case errors.Is(err, biz.ErrInvalidResearchRequest):
		return v1.ErrInvalidRequest
	case errors.Is(err, biz.ErrResearchNotFound):
		return v1.ErrResearchResultNotFound
	case errors.Is(err, biz.ErrResearchThemeNotFound):
		return v1.ErrResearchThemeNotFound
	case errors.Is(err, biz.ErrResearchReasoningTreesNotFound):
		return v1.ErrResearchReasoningTreesNotFound
	case errors.Is(err, biz.ErrResearchReasoningTreeNotFound):
		return v1.ErrResearchReasoningTreeNotFound
	case errors.Is(err, biz.ErrResearchDataService):
		return v1.ErrResearchDataFailure
	case errors.Is(err, biz.ErrResearchDataUnavailable):
		return v1.ErrResearchDataUnavailable
	default:
		return err
	}
}
