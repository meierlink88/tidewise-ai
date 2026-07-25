package service

import (
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/backend/services/miniapp/api/miniapp/v1"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
)

var _ v1.ResearchHTTPServer = (*ResearchService)(nil)

type ResearchService struct {
	research *biz.ResearchService
}

func NewResearchService(research *biz.ResearchService) *ResearchService {
	return &ResearchService{research: research}
}

func (s *ResearchService) ListResearchThemes(ctx context.Context, request *v1.ListResearchThemesRequest) (*v1.ResearchThemeListResponse, error) {
	if s == nil || s.research == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.research.ListThemes(ctx, biz.ResearchListRequest{
		WindowHours: request.WindowHours,
		Limit:       request.Limit,
		Cursor:      request.Cursor,
	})
	if err != nil {
		return nil, mapBizError(err)
	}
	return themeListResponse(result), nil
}

func (s *ResearchService) GetResearchTheme(ctx context.Context, request *v1.GetResearchThemeRequest) (*v1.ResearchThemeDetailResponse, error) {
	if s == nil || s.research == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.research.GetTheme(ctx, request.ThemeID, biz.ResearchDetailRequest{WindowHours: request.WindowHours})
	if err != nil {
		return nil, mapBizError(err)
	}
	return themeDetailResponse(result), nil
}

func (s *ResearchService) ListResearchThemeReasoningTrees(ctx context.Context, request *v1.ListResearchThemeReasoningTreesRequest) (*v1.ResearchReasoningTreeListResponse, error) {
	if s == nil || s.research == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.research.ListReasoningTrees(ctx, request.ThemeID)
	if err != nil {
		return nil, mapBizError(err)
	}
	trees := make([]v1.ResearchReasoningTreeSummary, 0, len(result.ReasoningTrees))
	for _, tree := range result.ReasoningTrees {
		trees = append(trees, v1.ResearchReasoningTreeSummary{
			AnchorID: tree.AnchorID,
			CenterChainNode: v1.ResearchReasoningTreeChainNode{
				ID: tree.CenterChainNode.ID, Name: tree.CenterChainNode.Name,
			},
		})
	}
	return &v1.ResearchReasoningTreeListResponse{
		Theme:          themeItem(result.Theme),
		ReasoningTrees: trees,
	}, nil
}

func (s *ResearchService) GetResearchThemeReasoningTree(ctx context.Context, request *v1.GetResearchThemeReasoningTreeRequest) (*v1.ResearchReasoningTreeDetailResponse, error) {
	if s == nil || s.research == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.research.GetReasoningTree(ctx, request.ThemeID, request.AnchorID)
	if err != nil {
		return nil, mapBizError(err)
	}
	return &v1.ResearchReasoningTreeDetailResponse{
		ThemeID:       result.ThemeID,
		ReasoningTree: reasoningTree(result.ReasoningTree),
	}, nil
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

func themeListResponse(value biz.ResearchThemeListResponse) *v1.ResearchThemeListResponse {
	items := make([]v1.ResearchThemeItem, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, themeItem(item))
	}
	return &v1.ResearchThemeListResponse{
		WindowStart: value.WindowStart,
		WindowEnd:   value.WindowEnd,
		AsOf:        value.AsOf,
		ThemeCount:  value.ThemeCount,
		EventCount:  value.EventCount,
		Items:       items,
		NextCursor:  value.NextCursor,
	}
}

func themeDetailResponse(value biz.ResearchThemeDetailResponse) *v1.ResearchThemeDetailResponse {
	events := make([]v1.ResearchEvent, 0, len(value.Events))
	for _, event := range value.Events {
		events = append(events, v1.ResearchEvent{
			EventID:        event.EventID,
			Title:          event.Title,
			Summary:        event.Summary,
			EventTime:      event.EventTime,
			EvidenceRole:   event.EvidenceRole,
			SupportedClaim: event.SupportedClaim,
		})
	}
	return &v1.ResearchThemeDetailResponse{
		ResearchThemeItem: themeItem(value.ResearchThemeItem),
		Events:            events,
	}
}

func themeItem(value biz.ResearchThemeItem) v1.ResearchThemeItem {
	nodes := make([]v1.ResearchChainNode, 0, len(value.AffectedChainNodes))
	for _, node := range value.AffectedChainNodes {
		nodes = append(nodes, v1.ResearchChainNode{
			ID: node.ID, Name: node.Name, RelationRole: node.RelationRole, ImpactSummary: node.Summary,
		})
	}
	indices := make([]v1.ResearchIndex, 0, len(value.RelatedIndices))
	for _, index := range value.RelatedIndices {
		indices = append(indices, v1.ResearchIndex{
			ID: index.ID, Name: index.Name, ImpactDirection: index.ImpactDirection, ImpactSummary: index.Summary,
		})
	}
	return v1.ResearchThemeItem{
		ID:                        value.ID,
		Name:                      value.Name,
		OneLineConclusion:         value.OneLineConclusion,
		ImpactLevel:               value.ImpactLevel,
		TransmissionPath:          value.TransmissionPath,
		TradingDirection:          value.TradingDirection,
		TransmissionStage:         value.TransmissionStage,
		NextCheckpoint:            value.NextCheckpoint,
		MarketConfirmationSummary: value.MarketConfirmationSummary,
		PublishedAt:               value.PublishedAt,
		AffectedChainNodes:        nodes,
		RelatedIndices:            indices,
		SupportingEventCount:      value.SupportingEventCount,
		ContradictingEventCount:   value.ContradictingEventCount,
	}
}

func reasoningTree(value biz.ResearchReasoningTreeDTO) v1.ResearchReasoningTree {
	events := make([]v1.ResearchReasoningTreeEvent, 0, len(value.Events))
	for _, event := range value.Events {
		events = append(events, v1.ResearchReasoningTreeEvent{
			EventID:         event.EventID,
			Title:           event.Title,
			Summary:         event.Summary,
			EventTime:       event.EventTime,
			EvidenceRole:    event.EvidenceRole,
			EvidenceSummary: event.EvidenceSummary,
		})
	}
	pathNodes := make([]v1.ResearchReasoningTreePathNode, 0, len(value.PathNodes))
	for _, node := range value.PathNodes {
		pathNodes = append(pathNodes, v1.ResearchReasoningTreePathNode{
			ChainNodeID:                   node.ChainNodeID,
			Name:                          node.Name,
			ChangeDirection:               node.ChangeDirection,
			ChangeSummary:                 node.ChangeSummary,
			ImpactSummary:                 node.ImpactSummary,
			IncomingTransmissionMechanism: node.IncomingTransmissionMechanism,
		})
	}
	return v1.ResearchReasoningTree{
		AnchorID: value.AnchorID,
		CenterChainNode: v1.ResearchReasoningTreeChainNode{
			ID: value.CenterChainNode.ID, Name: value.CenterChainNode.Name,
		},
		OneLineConclusion:   value.OneLineConclusion,
		FactSummary:         value.FactSummary,
		NetDirectionSummary: value.NetDirectionSummary,
		SupportSummary:      value.SupportSummary,
		CounterSummary:      value.CounterSummary,
		TradingDirection:    value.TradingDirection,
		NextCheckpoint:      value.NextCheckpoint,
		EventCount:          value.EventCount,
		Events:              events,
		PathNodes:           pathNodes,
	}
}
