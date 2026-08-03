package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPreparedUATSnapshotLocalKeysSurviveMiniappDataMapping(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "research-theme-analyst-snapshot-v3", "01-uat-at01-prepared-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var prepared struct {
		ReasoningTrees []struct {
			TreeKey     string `json:"tree_key"`
			DisplayName string `json:"display_name"`
			Nodes       []struct {
				NodeKey     string `json:"node_key"`
				DisplayName string `json:"display_name"`
				Signals     []struct {
					SignalKey      string  `json:"signal_key"`
					DisplaySummary string  `json:"display_summary"`
					VariableName   *string `json:"variable_name"`
					Direction      *string `json:"direction"`
				} `json:"signals"`
			} `json:"nodes"`
		} `json:"reasoning_trees"`
	}
	if err := json.Unmarshal(payload, &prepared); err != nil {
		t.Fatal(err)
	}
	tree := prepared.ReasoningTrees[0]
	node := tree.Nodes[0]
	signal := node.Signals[0]
	wire := wireResearchReasoningTreeDetail{
		ThemeID: "70000000-0000-4000-8000-000000000001",
		ReasoningTree: wireResearchReasoningTree{
			TreeKey: tree.TreeKey, DisplayName: tree.DisplayName,
			ReasoningTreeID: "70000000-0000-4000-8000-000000000002",
			Nodes: []wireResearchReasoningTreeNode{{
				NodeKey: node.NodeKey, DisplayName: node.DisplayName,
				Signals: []wireResearchSignal{{
					SignalKey: signal.SignalKey, VariableName: signal.VariableName,
					Direction: signal.Direction, DisplaySummary: signal.DisplaySummary,
				}},
			}},
		},
	}

	mapped := wire.toBiz()
	if mapped.ReasoningTree.TreeKey != tree.TreeKey || mapped.ReasoningTree.DisplayName != tree.DisplayName ||
		mapped.ReasoningTree.Nodes[0].NodeKey != node.NodeKey ||
		mapped.ReasoningTree.Nodes[0].DisplayName != node.DisplayName ||
		mapped.ReasoningTree.Nodes[0].Signals[0].SignalKey != signal.SignalKey ||
		mapped.ReasoningTree.Nodes[0].Signals[0].DisplaySummary != signal.DisplaySummary {
		t.Fatalf("Miniapp mapping lost prepared UAT snapshot fields: %#v", mapped)
	}
}
