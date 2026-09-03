package report

import (
	"fmt"
	"time"

	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
)

const (
	EvidenceOne = "EVD11111111-1111-4111-8111-111111111111"
	EvidenceTwo = "EVD22222222-2222-4222-8222-222222222222"
	ReportOne   = "RPT11111111-1111-4111-8111-111111111111"
)

func Report() reportbiz.Report {
	report := IndustryOnlyReport()
	report.Geopolitics = geopoliticalLayer()
	report.Macroeconomics = macroeconomicLayer()
	return report
}

func IndustryOnlyReport() reportbiz.Report {
	return reportbiz.Report{
		ReportType:     coded("investment_reasoning", "投研推理报告"),
		GeneratedAt:    time.Date(2026, 9, 3, 10, 30, 0, 0, time.FixedZone("CST", 8*60*60)),
		Timezone:       "Asia/Shanghai",
		IndustryChains: []reportbiz.IndustryChain{industryChain(1)},
	}
}

func ReportWithManyChains(count int) reportbiz.Report {
	report := IndustryOnlyReport()
	report.IndustryChains = make([]reportbiz.IndustryChain, count)
	for index := range report.IndustryChains {
		report.IndustryChains[index] = industryChain(index + 1)
	}
	return report
}

func FrozenScaleBaselineReport() reportbiz.Report {
	report := Report()
	evidenceIDs := FrozenScaleBaselineEvidenceIDs()
	report.IndustryChains = make([]reportbiz.IndustryChain, 54)
	nodeIndex := 0
	for chainIndex := range report.IndustryChains {
		chain := industryChain(chainIndex + 1)
		chain.EvidenceRefs = []reportbiz.EvidenceReference{evidenceRef(evidenceIDs[chainIndex%len(evidenceIDs)], "summary_support", "核心依据")}
		nodeCount := 3
		if chainIndex >= 49 {
			nodeCount = 2
		}
		chain.Nodes = make([]reportbiz.IndustryChainNode, nodeCount)
		chain.Edges = make([]reportbiz.IndustryChainEdge, 0, nodeCount-1)
		for position := 0; position < nodeCount; position++ {
			key := fmt.Sprintf("node-%02d-%02d", chainIndex+1, position+1)
			refs := []reportbiz.EvidenceReference{
				evidenceRef(evidenceIDs[nodeIndex%len(evidenceIDs)], "direct_support", "直接依据"),
			}
			if nodeIndex < 51 {
				refs = append(refs, evidenceRef(evidenceIDs[(nodeIndex+1)%len(evidenceIDs)], "direct_support", "直接依据"))
			}
			chain.Nodes[position] = directNode(key, fmt.Sprintf("产业节点 %02d-%02d", chainIndex+1, position+1), refs)
			if position > 0 {
				chain.Edges = append(chain.Edges, reportbiz.IndustryChainEdge{
					FromNodeLocalKey: chain.Nodes[position-1].LocalKey,
					ToNodeLocalKey:   key,
					RelationLabel:    "需求传导",
				})
			}
			nodeIndex++
		}
		report.IndustryChains[chainIndex] = chain
	}
	return report
}

func FrozenScaleBaselineEvidenceIDs() []string {
	result := []string{EvidenceOne, EvidenceTwo}
	for index := 3; index <= 43; index++ {
		result = append(result, fmt.Sprintf("EVD%08d-0000-5000-8000-%012d", index, index))
	}
	return result
}

func Content() reportbiz.Report                        { return Report() }
func IndustryOnlyContent() reportbiz.Report            { return IndustryOnlyReport() }
func ContentWithManyChains(count int) reportbiz.Report { return ReportWithManyChains(count) }

func geopoliticalLayer() *reportbiz.Layer {
	return &reportbiz.Layer{
		LocalKey: "geopolitics", Title: "地缘政治面", Conclusion: "海湾安全风险上升。",
		Result: coded(reportbiz.ResultWarming, "升温"), TimeWindow: window("short_medium", "短期–中期"),
		Confidence: confidence("medium", "中"),
		AffectedAnchors: []reportbiz.Anchor{{
			LocalKey: "geo-anchor-01", Name: "伊朗—美国及海湾安全对抗", CurrentState: "军事与航运风险上升。",
			Result:           coded(reportbiz.ResultWarming, "升温"),
			ConclusionBasis:  coded(reportbiz.BasisDirectEvidence, "直接证据"),
			ValidationStatus: coded(reportbiz.ValidationConfirmed, "已确认"),
			Reasoning:        "冲突和航运受阻共同推高地区风险。",
			TimeWindow:       window("short_medium", "短期–中期"), Confidence: confidence("medium", "中"),
			EvidenceRefs: []reportbiz.EvidenceReference{evidenceRef(EvidenceOne, "direct_support", "直接依据")},
		}},
		ReasoningSteps: []reportbiz.ReasoningStep{{
			LocalKey: "reasoning-01", Input: "冲突和运输受阻。", Mechanism: "关键航道风险上升。",
			Output: "地区风险升温。", Confidence: confidence("medium", "中"),
			EvidenceRefs: []reportbiz.EvidenceReference{evidenceRef(EvidenceOne, "reasoning_support", "推导依据")},
		}},
		Uncertainty: reportbiz.LayerUncertainty{
			Counterevidence: text("替代供应可能缓冲冲击。"), EvidenceGap: text("仍需价格数据验证。"),
			Boundary: text("影响取决于冲突持续时间。"), ReversalCondition: text("若运输恢复则影响减弱。"),
		},
		EvidenceRefs: []reportbiz.EvidenceReference{evidenceRef(EvidenceOne, "summary_support", "核心依据")},
		DownwardTransmission: reportbiz.DownwardTransmission{
			ToMacroeconomics: transmissionGroup("风险继续向下传导。", transmission(
				"geo-to-macro-01", "海湾安全风险上升", "macro_anchor", "宏观经济锚点",
				"macro-anchor-01", "能源供应与进口安全", reportbiz.ResultCooling, "降温",
			)),
			ToIndustryChains: transmissionGroup("风险继续向下传导。", transmission(
				"geo-to-chain-01", "海湾运输风险上升", "industry_chain", "产业链",
				"chain-01", "产业链 01", reportbiz.ResultWarming, "升温",
			)),
		},
	}
}

func macroeconomicLayer() *reportbiz.Layer {
	return &reportbiz.Layer{
		LocalKey: "macroeconomics", Title: "宏观经济面", Conclusion: "能源输入成本承压。",
		Result: coded(reportbiz.ResultDiverging, "分化"), TimeWindow: window("medium", "中期"),
		Confidence: confidence("medium", "中"),
		AffectedAnchors: []reportbiz.Anchor{{
			LocalKey: "macro-anchor-01", Name: "能源供应与进口安全", CurrentState: "进口供应稳定性下降。",
			Result:           coded(reportbiz.ResultCooling, "降温"),
			ConclusionBasis:  coded(reportbiz.BasisReasoningHypothesis, "推理假设"),
			ValidationStatus: coded(reportbiz.ValidationPending, "待验证"),
			Reasoning:        "运输风险可能增加进口成本。",
			TimeWindow:       window("medium", "中期"), Confidence: confidence("medium", "中"),
			EvidenceRefs: []reportbiz.EvidenceReference{},
		}},
		ReasoningSteps: []reportbiz.ReasoningStep{},
		Uncertainty: reportbiz.LayerUncertainty{
			Counterevidence: text("替代供应可能缓冲冲击。"), EvidenceGap: text("仍需价格数据验证。"),
			Boundary: text("影响取决于冲突持续时间。"), ReversalCondition: text("若运输恢复则影响减弱。"),
		},
		EvidenceRefs: []reportbiz.EvidenceReference{},
		DownwardTransmission: reportbiz.DownwardTransmission{
			ToIndustryChains: transmissionGroup("成本压力向产业链传导。", transmission(
				"macro-to-chain-01", "海湾运输风险上升", "industry_chain", "产业链",
				"chain-01", "产业链 01", reportbiz.ResultWarming, "升温",
			)),
		},
	}
}

func transmissionGroup(summary string, paths ...reportbiz.Transmission) *reportbiz.TransmissionGroup {
	return &reportbiz.TransmissionGroup{Summary: summary, Paths: paths}
}

func transmission(key, source, targetCode, targetLabel, targetKey, targetName, resultCode, resultLabel string) reportbiz.Transmission {
	return reportbiz.Transmission{
		LocalKey: key, SourceConclusion: source,
		Targets: []reportbiz.TransmissionTarget{{
			TargetType: coded(targetCode, targetLabel), TargetLocalKey: targetKey,
			TargetName: targetName, Result: coded(resultCode, resultLabel),
		}},
		TransmissionLogic: "供应风险提高资源保障需求。",
		TransmissionKind:  coded(reportbiz.TransmissionCrossLayer, "跨层推理"),
		Confidence:        confidence("medium", "中"), Status: coded("established", "已形成传导"),
	}
}

func industryChain(order int) reportbiz.IndustryChain {
	key := fmt.Sprintf("chain-%02d", order)
	nodeKey := fmt.Sprintf("node-%02d", order)
	return reportbiz.IndustryChain{
		LocalKey: key, Name: fmt.Sprintf("产业链 %02d", order), Conclusion: "产业链聚合结果升温。",
		Result: coded(reportbiz.ResultWarming, "升温"), TimeWindow: window("medium_long", "中期–长期"),
		Confidence: confidence("medium", "中"), PathSummary: text("输入→节点→产出"),
		AcceptedHypothesisSummary: text("设备需求可能随后改善。"),
		Nodes: []reportbiz.IndustryChainNode{
			directNode(nodeKey, fmt.Sprintf("产业节点 %02d", order), []reportbiz.EvidenceReference{
				evidenceRef(EvidenceTwo, "direct_support", "直接依据"),
			}),
		},
		Edges: []reportbiz.IndustryChainEdge{},
		Uncertainty: reportbiz.ChainUncertainty{
			CounterevidenceAndGap: text("反证与缺口仍需经营数据验证。"),
			StopCondition:         text("若后续 Signal 失效或方向反转，停止该链结论。"),
		},
		EvidenceRefs: []reportbiz.EvidenceReference{evidenceRef(EvidenceOne, "summary_support", "核心依据")},
	}
}

func directNode(key, name string, refs []reportbiz.EvidenceReference) reportbiz.IndustryChainNode {
	return reportbiz.IndustryChainNode{
		LocalKey: key, Name: name, Impact: "需求 UP",
		Result:           coded(reportbiz.ResultWarming, "升温"),
		ConclusionBasis:  coded(reportbiz.BasisDirectEvidence, "直接证据"),
		ValidationStatus: coded(reportbiz.ValidationConfirmed, "已确认"),
		Reasoning:        "目标节点具有直接 Signal。", TimeWindow: window("medium", "中期"),
		Confidence: confidence("medium", "中"), EvidenceRefs: refs,
	}
}

func coded(code, label string) reportbiz.CodedLabel {
	return reportbiz.CodedLabel{Code: code, Label: label}
}
func confidence(code, label string) reportbiz.Confidence {
	return reportbiz.Confidence{Code: code, Label: label}
}
func window(code, label string) reportbiz.TimeWindow {
	return reportbiz.TimeWindow{Code: code, Label: label}
}
func evidenceRef(id, code, label string) reportbiz.EvidenceReference {
	return reportbiz.EvidenceReference{EvidenceID: id, Role: coded(code, label)}
}
func text(value string) *string { return &value }
