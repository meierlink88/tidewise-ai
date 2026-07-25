package research

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type fakeRepository struct {
	themePage       ThemeStorePage
	themeDetail     ThemeDetailRecord
	reasoningTrees  ReasoningTreeListRecord
	reasoningTree   ReasoningTreeDetailRecord
	err             error
	themeFilter     ThemeListFilter
	reasoningTheme  string
	reasoningAnchor string
}

func (f *fakeRepository) ListResearchThemes(_ context.Context, filter ThemeListFilter) (ThemeStorePage, error) {
	f.themeFilter = filter
	return f.themePage, f.err
}

func (f *fakeRepository) GetResearchTheme(context.Context, string, DetailFilter) (ThemeDetailRecord, error) {
	return f.themeDetail, f.err
}

func (f *fakeRepository) ListResearchThemeReasoningTrees(_ context.Context, themeID string) (ReasoningTreeListRecord, error) {
	f.reasoningTheme = themeID
	return f.reasoningTrees, f.err
}

func (f *fakeRepository) GetResearchThemeReasoningTree(_ context.Context, themeID, anchorID string) (ReasoningTreeDetailRecord, error) {
	f.reasoningTheme = themeID
	f.reasoningAnchor = anchorID
	return f.reasoningTree, f.err
}

func TestServiceKeepsStableThemeCursorAndAuthoritativeFields(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	repository := &fakeRepository{themePage: ThemeStorePage{
		AsOf: now, WindowStart: now.Add(-24 * time.Hour), WindowEnd: now,
		ThemeCount: 1, EventCount: 2, HasMore: true,
		Items: []ThemeSummaryRecord{{
			ID: "11111111-1111-4111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
			ImpactLevel: model.ImpactLevelFocus, TransmissionPath: "供给传导", TradingDirection: "关注成本回落后的利润修复",
			TransmissionStage: model.TransmissionStageDiffusion, NextCheckpoint: "下次数据", MarketConfirmationSummary: "市场信号混合",
			PublishedAt:          now,
			ChainNodes:           []ChainNodeRecord{{ID: "22222222-2222-4222-8222-222222222222", Name: "节点", RelationRole: "driver", Summary: "主题影响"}},
			Indices:              []IndexRecord{{ID: "33333333-3333-4333-8333-333333333333", Name: "指数", ImpactDirection: "mixed", Summary: "指数影响"}},
			SupportingEventCount: 1, ContradictingEventCount: 1,
		}},
	}}
	service := NewService(repository, func() time.Time { return now })

	page, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor == nil {
		t.Fatal("next cursor is nil")
	}
	if page.Items[0].ImpactLevel != model.ImpactLevelFocus || page.Items[0].TransmissionStage != model.TransmissionStageDiffusion {
		t.Fatalf("authoritative theme enums drifted: %#v", page.Items[0])
	}
	if page.Items[0].TradingDirection != "关注成本回落后的利润修复" {
		t.Fatalf("trading_direction = %q", page.Items[0].TradingDirection)
	}
	if got := page.Items[0].AffectedChainNodes[0].ImpactSummary; got != "主题影响" {
		t.Fatalf("theme impact_summary = %q", got)
	}
	cursor, err := decodeResearchCursor(*page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.Kind != "themes" || cursor.Rank != 2 || cursor.ID != page.Items[0].ID || !cursor.AsOf.Equal(now) {
		t.Fatalf("cursor = %#v", cursor)
	}

	if _, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20, Cursor: *page.NextCursor}); err != nil {
		t.Fatal(err)
	}
	if repository.themeFilter.CursorRank != 2 || repository.themeFilter.CursorID != page.Items[0].ID || !repository.themeFilter.AsOf.Equal(now) {
		t.Fatalf("cursor repository filter = %#v", repository.themeFilter)
	}
}

func TestServiceReturnsThemeReasoningTreeTabsAndOneCompleteTree(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	anchorID := "22222222-2222-4222-8222-222222222222"
	eventTime := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	theme := ThemeSummaryRecord{
		ID: themeID, Name: "AI算力扩产与半导体", OneLineConclusion: "AI算力扩产继续传导",
		ImpactLevel: model.ImpactLevelHigh, TransmissionPath: "资本开支 → 芯片 → 光模块",
		TradingDirection: "优先研究光模块", TransmissionStage: model.TransmissionStageDiffusion,
		NextCheckpoint: "跟踪订单", MarketConfirmationSummary: "订单证据偏强", PublishedAt: eventTime,
		ChainNodes: []ChainNodeRecord{{ID: "33333333-3333-4333-8333-333333333333", Name: "光模块", RelationRole: "beneficiary", Summary: "受益"}},
		Indices:    []IndexRecord{}, SupportingEventCount: 1, ContradictingEventCount: 1,
	}
	repository := &fakeRepository{
		reasoningTrees: ReasoningTreeListRecord{
			Theme: theme,
			ReasoningTrees: []ReasoningTreeSummaryRecord{
				{AnchorID: anchorID, CenterChainNodeID: "33333333-3333-4333-8333-333333333333", CenterChainNodeName: "光模块"},
				{AnchorID: "44444444-4444-4444-8444-444444444444", CenterChainNodeID: "55555555-5555-4555-8555-555555555555", CenterChainNodeName: "先进封装"},
			},
		},
		reasoningTree: ReasoningTreeDetailRecord{
			ThemeID: themeID,
			ReasoningTree: ReasoningTreeRecord{
				AnchorID: anchorID, CenterChainNodeID: "33333333-3333-4333-8333-333333333333", CenterChainNodeName: "光模块",
				OneLineConclusion: "需求偏强", FactSummary: "资本开支增加", NetDirectionSummary: "偏正",
				TradingDirection: "研究订单", NextCheckpoint: "观察交付",
				Events: []ReasoningTreeEventRecord{{
					EventID: "66666666-6666-4666-8666-666666666666", Title: "资本开支上调", Summary: "投入增加",
					EventTime: &eventTime, EvidenceRole: "driver", EvidenceSummary: "构成直接驱动",
				}},
				PathNodes: []ReasoningTreePathNodeRecord{
					{Position: 1, ChainNodeID: "77777777-7777-4777-8777-777777777777", Name: "AI芯片", ChangeDirection: "increase", ChangeSummary: "采购增加", ImpactSummary: "扩大部署基础"},
					{Position: 2, ChainNodeID: "33333333-3333-4333-8333-333333333333", Name: "光模块", ChangeDirection: "mixed", ChangeSummary: "需求上升", ImpactSummary: "订单机会增加", IncomingTransmissionMechanism: stringPointer("集群扩容提高互联需求")},
				},
			},
		},
	}
	service := NewService(repository, time.Now)

	list, err := service.ListReasoningTrees(context.Background(), themeID)
	if err != nil {
		t.Fatal(err)
	}
	if repository.reasoningTheme != themeID || list.Theme.ID != themeID || len(list.ReasoningTrees) != 2 {
		t.Fatalf("list = %#v repository theme=%q", list, repository.reasoningTheme)
	}
	if list.ReasoningTrees[0].CenterChainNode.Name != "光模块" || list.Theme.RelatedIndices == nil {
		t.Fatalf("list mapping = %#v", list)
	}

	detail, err := service.GetReasoningTree(context.Background(), themeID, anchorID)
	if err != nil {
		t.Fatal(err)
	}
	if repository.reasoningAnchor != anchorID || detail.ThemeID != themeID || detail.ReasoningTree.EventCount != 1 {
		t.Fatalf("detail = %#v repository anchor=%q", detail, repository.reasoningAnchor)
	}
	if detail.ReasoningTree.Events[0].EvidenceSummary != "构成直接驱动" || detail.ReasoningTree.PathNodes[1].IncomingTransmissionMechanism == nil {
		t.Fatalf("detail mapping = %#v", detail)
	}
}

func TestServiceClassifiesReasoningTreeErrorsAndRejectsInvalidIDs(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	anchorID := "22222222-2222-4222-8222-222222222222"
	for _, test := range []struct {
		name       string
		repository error
		want       error
	}{
		{name: "theme missing", repository: ErrResearchThemeNotFound, want: ErrThemeNotFound},
		{name: "publication missing", repository: ErrResearchReasoningTreesNotFound, want: ErrReasoningTreesNotFound},
		{name: "tree missing", repository: ErrResearchReasoningTreeNotFound, want: ErrReasoningTreeNotFound},
		{name: "invariant", repository: ErrResearchReasoningTreeInvariant, want: ErrReasoningTreeInvariantViolation},
		{name: "repository", repository: errors.New("database unavailable"), want: ErrRepository},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(&fakeRepository{err: test.repository}, time.Now)
			_, err := service.GetReasoningTree(context.Background(), themeID, anchorID)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}

	service := NewService(&fakeRepository{}, time.Now)
	if _, err := service.ListReasoningTrees(context.Background(), "not-a-uuid"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid theme ID error = %v", err)
	}
	if _, err := service.GetReasoningTree(context.Background(), themeID, "not-a-uuid"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid anchor ID error = %v", err)
	}
}

func TestServiceNormalizesReasoningTreeUUIDsBeforeRepositoryLookup(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	anchorID := "22222222-2222-4222-8222-222222222222"
	repository := &fakeRepository{reasoningTree: ReasoningTreeDetailRecord{ThemeID: themeID}}
	service := NewService(repository, time.Now)

	if _, err := service.GetReasoningTree(context.Background(), strings.ToUpper(themeID), strings.ToUpper(anchorID)); err != nil {
		t.Fatal(err)
	}
	if repository.reasoningTheme != themeID || repository.reasoningAnchor != anchorID {
		t.Fatalf("repository IDs = %q %q", repository.reasoningTheme, repository.reasoningAnchor)
	}
}

func stringPointer(value string) *string { return &value }

func TestServiceValidatesRequestsAndClassifiesRepositoryErrors(t *testing.T) {
	service := NewService(&fakeRepository{}, time.Now)
	if _, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 169, Limit: 20}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid window error = %v", err)
	}
	if _, err := service.GetTheme(context.Background(), "not-a-uuid", ResearchDetailRequest{WindowHours: 24}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid UUID error = %v", err)
	}

	service = NewService(&fakeRepository{err: ErrResearchNotFound}, time.Now)
	if _, err := service.GetTheme(context.Background(), "11111111-1111-4111-8111-111111111111", ResearchDetailRequest{WindowHours: 24}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("not found error = %v", err)
	}
	service = NewService(&fakeRepository{err: errors.New("database unavailable")}, time.Now)
	if _, err := service.ListThemes(context.Background(), ResearchListRequest{WindowHours: 24, Limit: 20}); !errors.Is(err, ErrRepository) {
		t.Fatalf("repository error = %v", err)
	}
}

var _ Repository = (*fakeRepository)(nil)
