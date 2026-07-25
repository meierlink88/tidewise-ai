package data

import (
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
)

func TestWireReasoningTreeConvertsToBizModel(t *testing.T) {
	eventTime := time.Date(2026, 7, 25, 1, 2, 3, 0, time.UTC)
	mechanism := "供给收紧传导至价格"
	counter := "需求回落可能抵消涨价"
	wire := wireResearchReasoningTreeDetail{
		ThemeID: "theme-id",
		ReasoningTree: wireResearchReasoningTree{
			AnchorID:            "anchor-id",
			CenterChainNode:     wireResearchReasoningTreeChainNode{ID: "center-id", Name: "先进封装"},
			OneLineConclusion:   "封装供需继续收紧",
			FactSummary:         "订单与交期同步扩张",
			NetDirectionSummary: "供给偏紧",
			SupportSummary:      "当前证据支持供给收紧",
			CounterSummary:      &counter,
			TradingDirection:    "关注设备与材料",
			NextCheckpoint:      "观察交期",
			EventCount:          1,
			Events: []wireResearchReasoningTreeEvent{{
				EventID: "event-id", Title: "事件", Summary: "摘要", EventTime: &eventTime,
				EvidenceRole: "driver", EvidenceSummary: "驱动判断",
			}},
			PathNodes: []wireResearchReasoningTreePathNode{{
				ChainNodeID: "node-id", Name: "设备", ChangeDirection: "increase",
				ChangeSummary: "需求增加", ImpactSummary: "订单改善",
				IncomingTransmissionMechanism: &mechanism,
			}},
		},
	}

	got := wire.toBiz()

	if got.ThemeID != "theme-id" || got.ReasoningTree.CenterChainNode.Name != "先进封装" {
		t.Fatalf("identity conversion = %#v", got)
	}
	if got.ReasoningTree.Events[0].EvidenceRole != biz.EvidenceRoleDriver || !got.ReasoningTree.Events[0].EventTime.Equal(eventTime) {
		t.Fatalf("event conversion = %#v", got.ReasoningTree.Events)
	}
	if got.ReasoningTree.PathNodes[0].ChangeDirection != biz.ChangeDirectionIncrease ||
		*got.ReasoningTree.PathNodes[0].IncomingTransmissionMechanism != mechanism {
		t.Fatalf("path conversion = %#v", got.ReasoningTree.PathNodes)
	}
}
