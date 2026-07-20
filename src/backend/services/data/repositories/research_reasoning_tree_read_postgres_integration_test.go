package repositories_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
	anchorapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchanchorimport"
)

func TestResearchReasoningTreeReadPostgresIntegration(t *testing.T) {
	db := openResearchAnchorImportDatabase(t)
	prepareResearchAnchorImportSchema(t, db)
	publication := readResearchAnchorPublication(t)
	seedResearchAnchorPrerequisites(t, db, publication)
	repository := repositories.NewPostgresRepository(db)
	homeFilter := repositories.ResearchThemeListFilter{
		WindowStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		AsOf:        time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
		Limit:       20,
	}
	homeBefore, err := repository.ListResearchThemes(context.Background(), homeFilter)
	if err != nil {
		t.Fatal(err)
	}

	imported, err := anchorapp.NewService(repositories.NewPostgresRepository(db)).Import(
		context.Background(), "service:ai-research-analyst", publication,
	)
	if err != nil {
		t.Fatal(err)
	}
	homeAfter, err := repository.ListResearchThemes(context.Background(), homeFilter)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(homeBefore, homeAfter) {
		t.Fatalf("home Theme list changed after Anchor publication\nbefore: %#v\nafter:  %#v", homeBefore, homeAfter)
	}

	list, err := repository.ListResearchThemeReasoningTrees(context.Background(), publication.ThemeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.ReasoningTrees) != 2 {
		t.Fatalf("reasoning tree count = %d, want 2", len(list.ReasoningTrees))
	}
	if list.ReasoningTrees[0].CenterChainNodeName != "先进封装" || list.ReasoningTrees[1].CenterChainNodeName != "光模块" {
		t.Fatalf("reasoning tree order = %#v", list.ReasoningTrees)
	}
	if !list.Theme.PublishedAt.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Fatalf("historical theme published_at = %s", list.Theme.PublishedAt)
	}

	opticalAnchorID := imported.AnchorIDsByCenterChainNodeID["22222222-2222-4222-8222-222222222222"]
	optical, err := repository.GetResearchThemeReasoningTree(context.Background(), publication.ThemeID, opticalAnchorID)
	if err != nil {
		t.Fatal(err)
	}
	if len(optical.ReasoningTree.Events) != 2 || optical.ReasoningTree.Events[1].EventTime != nil {
		t.Fatalf("NULL event ordering = %#v", optical.ReasoningTree.Events)
	}
	if len(optical.ReasoningTree.PathNodes) != 2 || optical.ReasoningTree.PathNodes[0].Position != 1 || optical.ReasoningTree.PathNodes[1].Position != 2 {
		t.Fatalf("path ordering = %#v", optical.ReasoningTree.PathNodes)
	}

	packagingAnchorID := imported.AnchorIDsByCenterChainNodeID["33333333-3333-4333-8333-333333333333"]
	packaging, err := repository.GetResearchThemeReasoningTree(context.Background(), publication.ThemeID, packagingAnchorID)
	if err != nil {
		t.Fatal(err)
	}
	if len(packaging.ReasoningTree.Events) != 2 || packaging.ReasoningTree.Events[0].EventID >= packaging.ReasoningTree.Events[1].EventID {
		t.Fatalf("same-time UUID ordering = %#v", packaging.ReasoningTree.Events)
	}
}
