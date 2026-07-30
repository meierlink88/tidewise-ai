package researchthemeimport

import "testing"

func TestCanonicalHashIsStableForValidatedV1ThemeBatch(t *testing.T) {
	batch := Batch{
		AnalysisBatchID: "analysis-20260728",
		AnalysisAsOf:    "2026-07-28T08:00:00Z",
		WindowStart:     "2026-07-27T00:00:00Z",
		WindowEnd:       "2026-07-28T00:00:00Z",
		Themes: []Theme{{
			ThemeKey:                  "theme:optical-demand",
			Title:                     "高速光模块需求验证",
			OneLineConclusion:         "端口计划上调可能增强需求",
			ConclusionDirection:       "positive",
			ImpactStrength:            "medium",
			TransmissionStage:         "validation",
			InvestmentGuidanceAction:  "focus",
			InvestmentGuidanceSummary: "关注采购订单",
			TimeHorizonCategory:       "medium_term",
			Impacts: []Impact{{
				ChainNodeEntityID:           "11111111-1111-4111-8111-111111111111",
				RelationRole:                "beneficiary",
				ImpactDirection:             "positive",
				PrimarySignalDisplaySummary: "模块需求：可能增加",
				DisplayOrder:                1,
			}},
			Events: []Event{},
		}},
	}
	first, err := CanonicalHash(batch)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CanonicalHash(batch)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || len(first) != 64 {
		t.Fatalf("CanonicalHash() = %q and %q", first, second)
	}
}
