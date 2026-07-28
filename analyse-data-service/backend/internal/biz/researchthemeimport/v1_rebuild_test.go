package researchthemeimport

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	reasoningtreeimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
)

func TestV1ThemeContractAcceptsImpactsWithoutEvents(t *testing.T) {
	batch := Batch{
		AnalysisBatchID: "analysis-20260728",
		AnalysisAsOf:    "2026-07-28T08:00:00Z",
		WindowStart:     "2026-07-27T00:00:00Z",
		WindowEnd:       "2026-07-28T00:00:00Z",
		Themes: []Theme{{
			ThemeKey:                  "theme:optical-demand",
			Title:                     "高速光模块需求验证",
			OneLineConclusion:         "端口计划上调可能增强高速光模块需求预期",
			ConclusionDirection:       "positive",
			ImpactStrength:            "medium",
			TransmissionStage:         "validation",
			InvestmentGuidanceAction:  "focus",
			InvestmentGuidanceSummary: "关注采购订单与排产",
			TimeHorizonCategory:       "medium_term",
			Impacts: []Impact{{
				ChainNodeEntityID: "11111111-1111-4111-8111-111111111111",
				RelationRole:      "beneficiary",
				ImpactDirection:   "positive",
				DisplayOrder:      1,
			}},
		}},
	}

	window, err := batch.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if window.AnalysisAsOf.IsZero() {
		t.Fatal("analysis_as_of was not parsed")
	}
}

func TestV1ThemeStrictDecoderRejectsRemovedFields(t *testing.T) {
	payload := `{
	  "analysis_batch_id":"analysis-20260728",
	  "analysis_as_of":"2026-07-28T08:00:00Z",
	  "window_start":"2026-07-27T00:00:00Z",
	  "window_end":"2026-07-28T00:00:00Z",
	  "themes":[{
	    "theme_key":"theme:optical-demand",
	    "name":"removed",
	    "title":"高速光模块需求验证",
	    "one_line_conclusion":"端口计划上调可能增强需求",
	    "conclusion_direction":"positive",
	    "impact_strength":"medium",
	    "transmission_stage":"validation",
	    "investment_guidance_action":"focus",
	    "investment_guidance_summary":"关注采购订单",
	    "time_horizon_category":"medium_term",
	    "impacts":[{"chain_node_entity_id":"11111111-1111-4111-8111-111111111111","relation_role":"beneficiary","impact_direction":"positive","display_order":1}],
	    "events":[]
	  }]
	}`
	if _, err := DecodeStrict(bytes.NewBufferString(payload)); err == nil {
		t.Fatal("DecodeStrict() accepted removed name field")
	}
}

func TestSharedV1FixtureKeepsPublicationAndReadLineageDeterministic(t *testing.T) {
	fixtureDirectory := filepath.Join("..", "..", "..", "..", "..", "testdata", "reasoning-tree-v1")
	themeFile, err := os.Open(filepath.Join(fixtureDirectory, "00-theme-import-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer themeFile.Close()
	themeBatch, err := DecodeStrict(themeFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := themeBatch.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(themeBatch.Themes) != 1 {
		t.Fatalf("shared fixture Theme count = %d, want 1", len(themeBatch.Themes))
	}
	theme := themeBatch.Themes[0]
	themeID := ThemeID(themeBatch.AnalysisBatchID, theme.ThemeKey)

	treeFile, err := os.Open(filepath.Join(fixtureDirectory, "00-reasoning-tree-import-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer treeFile.Close()
	treePublication, err := reasoningtreeimport.DecodeStrict(treeFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := treePublication.Validate(); err != nil {
		t.Fatal(err)
	}
	if treePublication.ThemeID != themeID {
		t.Fatalf("Tree publication Theme ID = %s, want deterministic %s", treePublication.ThemeID, themeID)
	}

	var list struct {
		Result struct {
			Theme struct {
				ID                  string `json:"id"`
				AnalysisBatchID     string `json:"analysis_batch_id"`
				Title               string `json:"title"`
				OneLineConclusion   string `json:"one_line_conclusion"`
				ConclusionDirection string `json:"conclusion_direction"`
				ImpactStrength      string `json:"impact_strength"`
				TransmissionStage   string `json:"transmission_stage"`
				TransmissionSummary string `json:"transmission_summary"`
				PublishedAt         string `json:"published_at"`
				Impacts             []struct {
					ChainNodeEntityID string `json:"chain_node_entity_id"`
					RelationRole      string `json:"relation_role"`
					ImpactDirection   string `json:"impact_direction"`
					DisplayOrder      int    `json:"display_order"`
				} `json:"impacts"`
			} `json:"theme"`
			ReasoningTrees []struct {
				ReasoningTreeID       string `json:"reasoning_tree_id"`
				IndustryChainEntityID string `json:"industry_chain_entity_id"`
				Title                 string `json:"title"`
				PublishedAt           string `json:"published_at"`
				DisplayOrder          int    `json:"display_order"`
			} `json:"reasoning_trees"`
		} `json:"result"`
	}
	decodeFixtureJSON(t, filepath.Join(fixtureDirectory, "01-reasoning-tree-list-result.json"), &list)
	if list.Result.Theme.ID != themeID ||
		list.Result.Theme.AnalysisBatchID != themeBatch.AnalysisBatchID ||
		list.Result.Theme.Title != theme.Title ||
		list.Result.Theme.OneLineConclusion != theme.OneLineConclusion ||
		list.Result.Theme.ConclusionDirection != theme.ConclusionDirection ||
		list.Result.Theme.ImpactStrength != theme.ImpactStrength ||
		list.Result.Theme.TransmissionStage != theme.TransmissionStage ||
		list.Result.Theme.TransmissionSummary != stringValue(theme.TransmissionSummary) {
		t.Fatalf("Theme read fixture drifted from its publication: %#v", list.Result.Theme)
	}
	if list.Result.Theme.PublishedAt == "" {
		t.Fatal("Theme read fixture has no server publication time")
	}
	if len(list.Result.Theme.Impacts) != len(theme.Impacts) {
		t.Fatalf("Theme read Impact count = %d, want %d", len(list.Result.Theme.Impacts), len(theme.Impacts))
	}
	for index, impact := range theme.Impacts {
		got := list.Result.Theme.Impacts[index]
		if got.ChainNodeEntityID != impact.ChainNodeEntityID ||
			got.RelationRole != impact.RelationRole ||
			got.ImpactDirection != impact.ImpactDirection ||
			got.DisplayOrder != impact.DisplayOrder {
			t.Fatalf("Theme read Impact %d = %#v, want publication %#v", index, got, impact)
		}
	}
	if len(list.Result.ReasoningTrees) != len(treePublication.ReasoningTrees) {
		t.Fatalf("Tree list count = %d, want %d", len(list.Result.ReasoningTrees), len(treePublication.ReasoningTrees))
	}
	var treeReceiptPublishedAt string
	for index, tree := range treePublication.ReasoningTrees {
		got := list.Result.ReasoningTrees[index]
		expectedID := reasoningtreeimport.ReasoningTreeID(themeID, tree.IndustryChainEntityID)
		if got.ReasoningTreeID != expectedID ||
			got.IndustryChainEntityID != tree.IndustryChainEntityID ||
			got.Title != tree.Title || got.DisplayOrder != tree.DisplayOrder {
			t.Fatalf("Tree read %d = %#v, want deterministic publication identity", index, got)
		}
		if index == 0 {
			treeReceiptPublishedAt = got.PublishedAt
		} else if got.PublishedAt != treeReceiptPublishedAt {
			t.Fatalf("Tree %d published_at = %s, want shared receipt time %s",
				index, got.PublishedAt, treeReceiptPublishedAt)
		}
	}

	var detail struct {
		Result struct {
			ThemeID       string   `json:"theme_id"`
			ImpactNodeIDs []string `json:"impact_node_ids"`
			ReasoningTree struct {
				ReasoningTreeID       string `json:"reasoning_tree_id"`
				ThemeID               string `json:"theme_id"`
				IndustryChainEntityID string `json:"industry_chain_entity_id"`
				Title                 string `json:"title"`
				OneLineConclusion     string `json:"one_line_conclusion"`
				TransmissionSummary   string `json:"transmission_summary"`
				PublishedAt           string `json:"published_at"`
				Nodes                 []struct {
					ID                string `json:"id"`
					ChainNodeEntityID string `json:"chain_node_entity_id"`
					Position          int    `json:"position"`
				} `json:"nodes"`
			} `json:"reasoning_tree"`
		} `json:"result"`
	}
	decodeFixtureJSON(t, filepath.Join(fixtureDirectory, "02-reasoning-tree-with-contradiction-result.json"), &detail)
	firstTree := treePublication.ReasoningTrees[0]
	firstTreeID := reasoningtreeimport.ReasoningTreeID(themeID, firstTree.IndustryChainEntityID)
	if detail.Result.ThemeID != themeID ||
		detail.Result.ReasoningTree.ThemeID != themeID ||
		detail.Result.ReasoningTree.ReasoningTreeID != firstTreeID ||
		detail.Result.ReasoningTree.IndustryChainEntityID != firstTree.IndustryChainEntityID ||
		detail.Result.ReasoningTree.Title != firstTree.Title ||
		detail.Result.ReasoningTree.OneLineConclusion != firstTree.OneLineConclusion ||
		detail.Result.ReasoningTree.TransmissionSummary != stringValue(firstTree.TransmissionSummary) ||
		detail.Result.ReasoningTree.PublishedAt != treeReceiptPublishedAt {
		t.Fatalf("Tree detail fixture drifted from its publication: %#v", detail.Result.ReasoningTree)
	}
	if len(detail.Result.ReasoningTree.Nodes) != len(firstTree.Nodes) {
		t.Fatalf("Tree detail Node count = %d, want %d",
			len(detail.Result.ReasoningTree.Nodes), len(firstTree.Nodes))
	}
	for index, node := range firstTree.Nodes {
		got := detail.Result.ReasoningTree.Nodes[index]
		expectedID := reasoningtreeimport.ReasoningTreeNodeID(firstTreeID, node.Position, node.ChainNodeEntityID)
		if got.ID != expectedID || got.Position != node.Position || got.ChainNodeEntityID != node.ChainNodeEntityID {
			t.Fatalf("Tree detail Node %d = %#v, want deterministic ID %s", index, got, expectedID)
		}
	}
}

func decodeFixtureJSON(t *testing.T, path string, target any) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
