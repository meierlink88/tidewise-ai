package researchthemeimport

import "testing"

func TestCanonicalHashMatchesFrozenV1Fixture(t *testing.T) {
	batch := Batch{
		AnalysisBatchID: "batch-1",
		WindowStart:     "2026-07-15T00:00:00Z",
		WindowEnd:       "2026-07-18T00:00:00Z",
		Themes: []Theme{{
			ThemeKey:                  "theme:ai",
			Name:                      "AI <扩产> & 验证",
			OneLineConclusion:         "结论",
			ImpactLevel:               "high",
			TransmissionPath:          "采购 → 扩产",
			TradingDirection:          "研究设备",
			TransmissionStage:         "validation",
			NextCheckpoint:            "跟踪订单",
			MarketConfirmationSummary: "没有可归属的市场观测",
			ChainNodes: []ChainNode{{
				ChainNodeID:   "11111111-1111-4111-8111-111111111111",
				RelationRole:  "driver",
				ImpactSummary: "需求驱动",
			}},
			Events: []Event{{
				EventID:        "22222222-2222-4222-8222-222222222222",
				EvidenceRole:   "driver",
				SupportedClaim: "支持判断",
			}},
		}},
	}

	got, err := CanonicalHash(batch)
	if err != nil {
		t.Fatalf("CanonicalHash() error = %v", err)
	}
	const want = "e30f2bae458a940233e593a7cf78a9964ad0396b54949fde3219ba31bb7adc8e"
	if got != want {
		t.Fatalf("CanonicalHash() = %q, want %q", got, want)
	}
}

func TestCanonicalHashPreservesArrayOrder(t *testing.T) {
	left := validBatch()
	second := left.Themes[0].Events[0]
	second.EventID = "33333333-3333-4333-8333-333333333333"
	second.EvidenceRole = "supporting"
	left.Themes[0].Events = append(left.Themes[0].Events, second)
	right := left
	right.Themes = append([]Theme(nil), left.Themes...)
	right.Themes[0].Events = []Event{second, left.Themes[0].Events[0]}

	leftHash, err := CanonicalHash(left)
	if err != nil {
		t.Fatal(err)
	}
	rightHash, err := CanonicalHash(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftHash == rightHash {
		t.Fatal("CanonicalHash() ignored array order")
	}
}
