package research

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	themePage      ThemeStorePage
	themeDetail    ThemeDetailRecord
	reasoningTrees ReasoningTreeListRecord
	reasoningTree  ReasoningTreeDetailRecord
	err            error
	themeFilter    ThemeListFilter
	themeID        string
	treeID         string
}

func (f *fakeRepository) ListResearchThemes(_ context.Context, filter ThemeListFilter) (ThemeStorePage, error) {
	f.themeFilter = filter
	return f.themePage, f.err
}
func (f *fakeRepository) GetResearchTheme(context.Context, string, DetailFilter) (ThemeDetailRecord, error) {
	return f.themeDetail, f.err
}
func (f *fakeRepository) ListResearchThemeReasoningTrees(_ context.Context, themeID string) (ReasoningTreeListRecord, error) {
	f.themeID = themeID
	return f.reasoningTrees, f.err
}
func (f *fakeRepository) GetResearchThemeReasoningTree(_ context.Context, themeID, treeID string) (ReasoningTreeDetailRecord, error) {
	f.themeID, f.treeID = themeID, treeID
	return f.reasoningTree, f.err
}

func TestServiceUsesPublishedAtCursorForThemeOrdering(t *testing.T) {
	now := time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC)
	repository := &fakeRepository{themePage: ThemeStorePage{
		AsOf: now, WindowStart: now.Add(-24 * time.Hour), WindowEnd: now,
		ThemeCount: 1, EventCount: 2, HasMore: true,
		Items: []ThemeSummaryRecord{{
			ID: "11111111-1111-4111-8111-111111111111", AnalysisBatchID: "batch",
			Title: "Theme", OneLineConclusion: "结论", ConclusionDirection: "positive",
			ImpactStrength: "medium", TransmissionStage: "validation",
			InvestmentGuidanceAction: "focus", InvestmentGuidanceSummary: "关注订单",
			TimeHorizonCategory: "short_term", AnalysisAsOf: now, WindowStart: now.Add(-time.Hour),
			WindowEnd: now, PublishedAt: now, Impacts: []ThemeImpactRecord{},
		}},
	}}
	service := NewService(repository, func() time.Time { return now })

	page, err := service.ListThemes(context.Background(), ResearchListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor == nil {
		t.Fatal("next cursor is nil")
	}
	cursor, err := decodeResearchCursor(*page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.Kind != "themes" || cursor.ID != page.Items[0].ID || !cursor.PublishedAt.Equal(now) {
		t.Fatalf("cursor = %#v", cursor)
	}
}

func TestServiceMapsReasoningTreeSignalsWithoutChoosingImpactPriority(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	treeID := "22222222-2222-4222-8222-222222222222"
	nodeID := "33333333-3333-4333-8333-333333333333"
	repository := &fakeRepository{reasoningTree: ReasoningTreeDetailRecord{
		ThemeID:       themeID,
		ImpactNodeIDs: []string{nodeID},
		ReasoningTree: ReasoningTreeRecord{
			ReasoningTreeID: treeID, ThemeID: themeID,
			IndustryChainEntityID: "44444444-4444-4444-8444-444444444444",
			IndustryChainName:     "产业链", Title: "Tree", DisplayOrder: 1,
			OneLineConclusion: "结论", ImpactDirection: "positive", ImpactStrength: "medium",
			PublishedAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
			Nodes: []ReasoningTreeNodeRecord{{
				ID: nodeID, Position: 1, ChainNodeEntityID: nodeID, Name: "节点",
				ImpactDirection: "positive", ImpactStrength: "medium",
				Signals: []SignalRecord{
					{VariableSignalKey: "primary", SignalRole: "primary", SignalDirection: "increase", DisplaySummary: "主信号", DisplayOrder: 1},
					{VariableSignalKey: "support", SignalRole: "supporting", SignalDirection: "uncertain", DisplaySummary: "支持信号", DisplayOrder: 2},
				},
			}},
		},
	}}
	service := NewService(repository, time.Now)

	detail, err := service.GetReasoningTree(context.Background(), themeID, treeID)
	if err != nil {
		t.Fatal(err)
	}
	node := detail.ReasoningTree.Nodes[0]
	if node.PrimarySignal.VariableSignalKey != "primary" || node.SignalDisplaySummary != "支持信号" {
		t.Fatalf("node signal projection = %#v", node)
	}
	if len(detail.ImpactNodeIDs) != 1 || detail.ImpactNodeIDs[0] != nodeID {
		t.Fatalf("impact IDs = %#v", detail.ImpactNodeIDs)
	}
}

func TestServiceKeepsStableReasoningTreeErrors(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	treeID := "22222222-2222-4222-8222-222222222222"
	for _, test := range []struct {
		repositoryError error
		want            error
	}{
		{ErrResearchThemeNotFound, ErrThemeNotFound},
		{ErrResearchReasoningTreesNotFound, ErrReasoningTreesNotFound},
		{ErrResearchReasoningTreeNotFound, ErrReasoningTreeNotFound},
		{ErrResearchReasoningTreeInvariant, ErrReasoningTreeInvariantViolation},
		{errors.New("database unavailable"), ErrRepository},
	} {
		service := NewService(&fakeRepository{err: test.repositoryError}, time.Now)
		_, err := service.GetReasoningTree(context.Background(), themeID, treeID)
		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	}
}

var _ Repository = (*fakeRepository)(nil)
