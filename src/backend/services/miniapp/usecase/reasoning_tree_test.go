package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
)

func TestResearchServiceListsReasoningTreeTabsFromSharedFixtureWithOneDataCall(t *testing.T) {
	t.Parallel()
	dataResult, expected := reasoningTreeFixtureResult[dataclient.ResearchReasoningTreeList](t, "01-reasoning-tree-list-result.json")
	calls := 0
	client := &dataclient.Fake{ListResearchThemeReasoningTreesFunc: func(_ context.Context, themeID string) (dataclient.ResearchReasoningTreeList, error) {
		calls++
		if themeID != "11111111-1111-4111-8111-111111111111" {
			t.Fatalf("theme ID = %q", themeID)
		}
		return dataResult, nil
	}}

	result, err := NewResearchService(client).ListReasoningTrees(context.Background(), "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("Data calls = %d, want 1", calls)
	}
	assertJSONEquivalent(t, expected, result)
}

func TestResearchServiceGetsReasoningTreeFromSharedFixtureWithOneDataCall(t *testing.T) {
	t.Parallel()
	dataResult, expected := reasoningTreeFixtureResult[dataclient.ResearchReasoningTreeDetail](t, "02-reasoning-tree-with-contradiction-result.json")
	calls := 0
	client := &dataclient.Fake{GetResearchThemeReasoningTreeFunc: func(_ context.Context, themeID, anchorID string) (dataclient.ResearchReasoningTreeDetail, error) {
		calls++
		if themeID != "11111111-1111-4111-8111-111111111111" || anchorID != "534d83be-774b-51d9-ad00-cdee4ba91799" {
			t.Fatalf("theme/anchor IDs = %q/%q", themeID, anchorID)
		}
		return dataResult, nil
	}}

	result, err := NewResearchService(client).GetReasoningTree(
		context.Background(),
		"11111111-1111-4111-8111-111111111111",
		"534d83be-774b-51d9-ad00-cdee4ba91799",
	)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("Data calls = %d, want 1", calls)
	}
	assertJSONEquivalent(t, expected, result)
}

func TestResearchServicePreservesReasoningTreeWithoutContradictionOrQuantification(t *testing.T) {
	t.Parallel()
	dataResult, expected := reasoningTreeFixtureResult[dataclient.ResearchReasoningTreeDetail](t, "03-reasoning-tree-without-contradiction-unquantified-result.json")
	client := &dataclient.Fake{GetResearchThemeReasoningTreeFunc: func(context.Context, string, string) (dataclient.ResearchReasoningTreeDetail, error) {
		return dataResult, nil
	}}

	result, err := NewResearchService(client).GetReasoningTree(
		context.Background(),
		"11111111-1111-4111-8111-111111111111",
		"5c18fc57-6bd8-5612-9a24-01a4e928b761",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEquivalent(t, expected, result)
}

func TestResearchServicePreservesNullReasoningTreeEventTime(t *testing.T) {
	t.Parallel()
	dataResult, _ := reasoningTreeFixtureResult[dataclient.ResearchReasoningTreeDetail](t, "02-reasoning-tree-with-contradiction-result.json")
	dataResult.ReasoningTree.Events[0].EventTime = nil
	client := &dataclient.Fake{GetResearchThemeReasoningTreeFunc: func(context.Context, string, string) (dataclient.ResearchReasoningTreeDetail, error) {
		return dataResult, nil
	}}

	result, err := NewResearchService(client).GetReasoningTree(
		context.Background(),
		"11111111-1111-4111-8111-111111111111",
		"534d83be-774b-51d9-ad00-cdee4ba91799",
	)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	events := raw["reasoning_tree"].(map[string]any)["events"].([]any)
	event, ok := events[0].(map[string]any)
	if !ok || event["event_time"] != nil {
		t.Fatalf("event_time was not preserved as null: %s", payload)
	}
}

func TestResearchServiceRejectsInvalidReasoningTreeInputBeforeCallingDataService(t *testing.T) {
	t.Parallel()
	calls := 0
	client := &dataclient.Fake{
		ListResearchThemeReasoningTreesFunc: func(context.Context, string) (dataclient.ResearchReasoningTreeList, error) {
			calls++
			return dataclient.ResearchReasoningTreeList{}, nil
		},
		GetResearchThemeReasoningTreeFunc: func(context.Context, string, string) (dataclient.ResearchReasoningTreeDetail, error) {
			calls++
			return dataclient.ResearchReasoningTreeDetail{}, nil
		},
	}
	service := NewResearchService(client)

	if _, err := service.ListReasoningTrees(context.Background(), "11111111-1111-4111-8111-11111111111A"); !errors.Is(err, ErrInvalidResearchRequest) {
		t.Fatalf("ListReasoningTrees() error = %v, want invalid request", err)
	}
	if _, err := service.GetReasoningTree(context.Background(), "11111111-1111-4111-8111-111111111111", "anchor-1"); !errors.Is(err, ErrInvalidResearchRequest) {
		t.Fatalf("GetReasoningTree() error = %v, want invalid request", err)
	}
	if calls != 0 {
		t.Fatalf("Data Service calls = %d, want 0", calls)
	}
}

func TestResearchServiceRejectsUnknownReasoningTreeEnums(t *testing.T) {
	t.Parallel()

	t.Run("list Theme enum", func(t *testing.T) {
		dataResult, _ := reasoningTreeFixtureResult[dataclient.ResearchReasoningTreeList](t, "01-reasoning-tree-list-result.json")
		dataResult.Theme.ImpactLevel = dataclient.ImpactLevel("unexpected")
		client := &dataclient.Fake{ListResearchThemeReasoningTreesFunc: func(context.Context, string) (dataclient.ResearchReasoningTreeList, error) {
			return dataResult, nil
		}}
		_, err := NewResearchService(client).ListReasoningTrees(context.Background(), "11111111-1111-4111-8111-111111111111")
		if !errors.Is(err, ErrResearchDataUnavailable) {
			t.Fatalf("error = %v, want data unavailable", err)
		}
	})

	t.Run("detail path enum", func(t *testing.T) {
		dataResult, _ := reasoningTreeFixtureResult[dataclient.ResearchReasoningTreeDetail](t, "02-reasoning-tree-with-contradiction-result.json")
		dataResult.ReasoningTree.PathNodes[0].ChangeDirection = dataclient.ChangeDirection("unexpected")
		client := &dataclient.Fake{GetResearchThemeReasoningTreeFunc: func(context.Context, string, string) (dataclient.ResearchReasoningTreeDetail, error) {
			return dataResult, nil
		}}
		_, err := NewResearchService(client).GetReasoningTree(
			context.Background(),
			"11111111-1111-4111-8111-111111111111",
			"534d83be-774b-51d9-ad00-cdee4ba91799",
		)
		if !errors.Is(err, ErrResearchDataUnavailable) {
			t.Fatalf("error = %v, want data unavailable", err)
		}
	})
}

func TestResearchServiceMapsReasoningTreeDataErrorsToStablePublicErrors(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		code string
		want error
	}{
		{name: "Theme missing", code: "RESEARCH_THEME_NOT_FOUND", want: ErrResearchThemeNotFound},
		{name: "trees missing", code: "RESEARCH_REASONING_TREES_NOT_FOUND", want: ErrResearchReasoningTreesNotFound},
		{name: "tree missing", code: "RESEARCH_REASONING_TREE_NOT_FOUND", want: ErrResearchReasoningTreeNotFound},
		{name: "unknown not found", code: "UNEXPECTED_NOT_FOUND", want: ErrResearchDataUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &dataclient.Fake{GetResearchThemeReasoningTreeFunc: func(context.Context, string, string) (dataclient.ResearchReasoningTreeDetail, error) {
				return dataclient.ResearchReasoningTreeDetail{}, &dataclient.Error{
					Kind: dataclient.ErrorKindClient, StatusCode: 404, Code: test.code, RequestID: "must-not-leak",
				}
			}}
			_, err := NewResearchService(client).GetReasoningTree(
				context.Background(),
				"11111111-1111-4111-8111-111111111111",
				"534d83be-774b-51d9-ad00-cdee4ba91799",
			)
			if !errors.Is(err, test.want) || err.Error() != test.want.Error() {
				t.Fatalf("error = %v, want stable %v", err, test.want)
			}
		})
	}
}

func reasoningTreeFixtureResult[T any](t *testing.T, name string) (T, any) {
	t.Helper()
	var zero T
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "reasoning-tree-v1", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("decode fixture envelope %s: %v", name, err)
	}
	if err := json.Unmarshal(envelope.Result, &zero); err != nil {
		t.Fatalf("decode typed fixture result %s: %v", name, err)
	}
	var expected any
	if err := json.Unmarshal(envelope.Result, &expected); err != nil {
		t.Fatalf("decode expected fixture result %s: %v", name, err)
	}
	return zero, expected
}

func assertJSONEquivalent(t *testing.T, want any, got any) {
	t.Helper()
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("encode result: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !reflect.DeepEqual(decoded, want) {
		t.Fatalf("result = %#v, want %#v", decoded, want)
	}
}
