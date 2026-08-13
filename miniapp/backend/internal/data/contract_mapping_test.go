package data

import (
	"testing"
)

func TestSnapshotLocalKeysSurviveMiniappDataMapping(t *testing.T) {
	variableName := "采购数量"
	direction := "increase"
	wire := wireResearchReasoningTreeDetail{
		ThemeID: "70000000-0000-4000-8000-000000000001",
		ReasoningTree: wireResearchReasoningTree{
			TreeKey: "tree:high-speed-optical-module", DisplayName: "高速光模块产业链",
			ReasoningTreeID: "70000000-0000-4000-8000-000000000002",
			Nodes: []wireResearchReasoningTreeNode{{
				NodeKey: "node:optical-module", DisplayName: "高速光模块",
				Signals: []wireResearchSignal{{
					SignalKey: "signal:purchase-volume", VariableName: &variableName,
					Direction: &direction, DisplaySummary: "采购数量增加",
				}},
			}},
		},
	}

	mapped := wire.toBiz()
	if mapped.ReasoningTree.TreeKey != "tree:high-speed-optical-module" || mapped.ReasoningTree.DisplayName != "高速光模块产业链" ||
		mapped.ReasoningTree.Nodes[0].NodeKey != "node:optical-module" ||
		mapped.ReasoningTree.Nodes[0].DisplayName != "高速光模块" ||
		mapped.ReasoningTree.Nodes[0].Signals[0].SignalKey != "signal:purchase-volume" ||
		mapped.ReasoningTree.Nodes[0].Signals[0].DisplaySummary != "采购数量增加" {
		t.Fatalf("Miniapp mapping lost snapshot fields: %#v", mapped)
	}
}
