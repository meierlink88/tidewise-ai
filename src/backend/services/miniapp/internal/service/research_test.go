package service

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/backend/services/miniapp/api/miniapp/v1"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
)

func TestResearchServiceMapsBizReadModelToPublicContract(t *testing.T) {
	publishedAt := time.Date(2026, 7, 17, 1, 2, 3, 0, time.UTC)
	repository := &biz.Fake{
		ListResearchThemesFunc: func(context.Context, biz.ResearchListQuery) (biz.ResearchThemePage, error) {
			return biz.ResearchThemePage{
				AsOf: publishedAt,
				Items: []biz.ResearchTheme{{
					ID: "11111111-1111-4111-8111-111111111111", Name: "主题",
					OneLineConclusion: "结论", ImpactLevel: biz.ImpactLevelHigh,
					TransmissionPath: "事件到产业", TradingDirection: "关注扩产受益环节",
					TransmissionStage: biz.TransmissionStageDiffusion,
					NextCheckpoint:    "验证订单", MarketConfirmationSummary: "市场尚未验证",
					PublishedAt: publishedAt,
					AffectedChainNodes: []biz.ResearchThemeChainNode{{
						ID: "22222222-2222-4222-8222-222222222222", Name: "节点",
						RelationRole: "driver", ImpactSummary: "需求增加",
					}},
					RelatedIndices: []biz.ResearchIndex{},
				}},
			}, nil
		},
	}
	subject := NewResearchService(biz.NewResearchService(repository))

	result, err := subject.ListResearchThemes(context.Background(), &v1.ListResearchThemesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].ImpactLevel != "high" ||
		result.Items[0].AffectedChainNodes[0].ImpactSummary != "需求增加" ||
		result.Items[0].RelatedIndices == nil {
		t.Fatalf("result = %#v", result)
	}
}

func TestResearchServiceMapsBizErrorsToAPIContract(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "invalid request", err: biz.ErrInvalidResearchRequest, want: v1.ErrInvalidRequest},
		{name: "result not found", err: biz.ErrResearchNotFound, want: v1.ErrResearchResultNotFound},
		{name: "theme not found", err: biz.ErrResearchThemeNotFound, want: v1.ErrResearchThemeNotFound},
		{name: "trees not found", err: biz.ErrResearchReasoningTreesNotFound, want: v1.ErrResearchReasoningTreesNotFound},
		{name: "tree not found", err: biz.ErrResearchReasoningTreeNotFound, want: v1.ErrResearchReasoningTreeNotFound},
		{name: "data failure", err: biz.ErrResearchDataService, want: v1.ErrResearchDataFailure},
		{name: "data unavailable", err: biz.ErrResearchDataUnavailable, want: v1.ErrResearchDataUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := mapBizError(test.err); !errors.Is(got, test.want) {
				t.Fatalf("mapBizError(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}
