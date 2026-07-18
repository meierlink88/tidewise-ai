package domain

import (
	"strings"
	"testing"
	"time"
)

func TestResearchThemeValidate(t *testing.T) {
	start := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	base := ResearchTheme{
		ID: "theme-1", AnalysisBatchID: "batch-1", Name: "算力基建",
		OneLineConclusion: "需求持续，瓶颈向互联传导。", ImpactLevel: ImpactLevelHigh,
		TransmissionPath: "需求 -> 互联", TradingDirection: "关注供给约束",
		TransmissionStage: TransmissionStageIdentification, NextCheckpoint: "看订单",
		IndexImpactSummary: "指数影响偏正面", WindowStart: &start, WindowEnd: &end,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	for name, mutate := range map[string]func(*ResearchTheme){
		"blank required text": func(v *ResearchTheme) { v.Name = " " },
		"unsupported impact":  func(v *ResearchTheme) { v.ImpactLevel = "critical" },
		"half window":         func(v *ResearchTheme) { v.WindowEnd = nil },
		"reversed window":     func(v *ResearchTheme) { v.WindowEnd = &start; v.WindowStart = &end },
	} {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if err := value.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want validation error")
			}
		})
	}
}

func TestResearchThemeValidateAcceptsConclusionTransmissionStagesOnly(t *testing.T) {
	base := ResearchTheme{
		ID: "theme-1", AnalysisBatchID: "batch-1", Name: "算力基建",
		OneLineConclusion: "需求持续，瓶颈向互联传导。", ImpactLevel: ImpactLevelHigh,
		TransmissionPath: "需求 -> 互联", TradingDirection: "关注供给约束",
		NextCheckpoint: "尚未显现", IndexImpactSummary: "指数影响偏正面",
	}

	for _, stage := range []TransmissionStage{
		TransmissionStageIdentification,
		TransmissionStageValidation,
		TransmissionStageDiffusion,
		TransmissionStageDampening,
	} {
		value := base
		value.TransmissionStage = stage
		if err := value.Validate(); err != nil {
			t.Fatalf("Validate() stage %q error = %v", stage, err)
		}
	}

	for _, legacyStage := range []TransmissionStage{"upstream", "midstream", "downstream", "infrastructure", "service"} {
		value := base
		value.TransmissionStage = legacyStage
		if err := value.Validate(); err == nil {
			t.Fatalf("Validate() legacy stage %q error = nil", legacyStage)
		}
	}
}

func TestResearchAnchorValidateRejectsUnsupportedValues(t *testing.T) {
	anchor := ResearchAnchor{
		ID: "anchor-1", AnalysisBatchID: "batch-1", AnchorType: AnchorTypePolicy,
		Name: "政策锚点", OneLineConclusion: "政策方向稳定。", Importance: ResearchImportancePrimary,
		TransmissionPath: "政策 -> 需求", TradingDirection: "观察政策兑现",
	}
	if err := anchor.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	anchor.AnchorType = "unsupported"
	if err := anchor.Validate(); err == nil || !strings.Contains(err.Error(), "anchor type") {
		t.Fatalf("Validate() error = %v, want anchor type validation error", err)
	}
}
