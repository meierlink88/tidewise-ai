package biz

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResearchServiceMapsReasoningTreeNodesAndSignals(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	treeID := "22222222-2222-4222-8222-222222222222"
	calls := 0
	repo := &Fake{GetResearchThemeReasoningTreeFunc: func(_ context.Context, gotTheme, gotTree string) (ResearchReasoningTreeDetail, error) {
		calls++
		if gotTheme != themeID || gotTree != treeID {
			t.Fatalf("ids=%s/%s", gotTheme, gotTree)
		}
		return ResearchReasoningTreeDetail{ThemeID: themeID, ImpactNodeIDs: []string{"node"}, ReasoningTree: ResearchReasoningTree{ReasoningTreeID: treeID, ThemeID: themeID, Title: "高速光模块", PublishedAt: time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC), Nodes: []ResearchReasoningTreeNode{{ID: "node", Name: "DSP 芯片", PrimarySignal: ResearchSignal{VariableSignalKey: "dsp-demand", SignalRole: "primary", DisplayOrder: 1}, SignalDisplaySummary: "渗透率待确认"}}}}, nil
	}}
	result, err := NewResearchService(repo).GetReasoningTree(context.Background(), themeID, treeID)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || result.ImpactNodeIDs[0] != "node" || result.ReasoningTree.Nodes[0].PrimarySignal.VariableSignalKey != "dsp-demand" {
		t.Fatalf("result=%#v", result)
	}
}

func TestResearchServiceAllowsThemeWithoutTreesAndMapsErrors(t *testing.T) {
	themeID := "11111111-1111-4111-8111-111111111111"
	service := NewResearchService(&Fake{ListResearchThemeReasoningTreesFunc: func(context.Context, string) (ResearchReasoningTreeList, error) {
		return ResearchReasoningTreeList{Theme: ResearchTheme{ID: themeID}, ReasoningTrees: []ResearchReasoningTreeSummary{}}, nil
	}})
	result, err := service.ListReasoningTrees(context.Background(), themeID)
	if err != nil || len(result.ReasoningTrees) != 0 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	service = NewResearchService(&Fake{ListResearchThemeReasoningTreesFunc: func(context.Context, string) (ResearchReasoningTreeList, error) {
		return ResearchReasoningTreeList{}, ErrResearchThemeNotFound
	}})
	if _, err := service.ListReasoningTrees(context.Background(), themeID); !errors.Is(err, ErrResearchThemeNotFound) {
		t.Fatalf("error=%v", err)
	}
}
