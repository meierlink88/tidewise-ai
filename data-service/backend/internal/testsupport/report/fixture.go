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
	report.Geopolitics = layer("GEO-01", "地缘政治", "地缘风险升温。", "geo-anchor", EvidenceOne, "chain-01")
	report.Macroeconomics = layer("MAC-01", "宏观经济", "宏观路径分化。", "macro-anchor", EvidenceTwo, "chain-01")
	return report
}

func IndustryOnlyReport() reportbiz.Report {
	return reportbiz.Report{
		GeneratedAt:    time.Date(2026, 9, 2, 8, 30, 0, 0, time.FixedZone("CST", 8*60*60)),
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

// FrozenScaleBaselineReport preserves the cardinalities of the reviewed
// 2026-09-02 presentation report without coupling tests to AgentOS files.
// It intentionally uses compact deterministic copy so HTTP, persistence and
// pagination tests exercise the production-sized contract rather than the
// report's prose.
func FrozenScaleBaselineReport() reportbiz.Report {
	report := Report()
	evidenceIDs := FrozenScaleBaselineEvidenceIDs()
	report.IndustryChains = make([]reportbiz.IndustryChain, 54)
	impactIndex := 0
	for chainIndex := range report.IndustryChains {
		chain := industryChain(chainIndex + 1)
		chain.Summary.EvidenceIDs = []string{evidenceIDs[chainIndex%len(evidenceIDs)]}
		impactCount := 3
		if chainIndex >= 49 {
			impactCount = 2
		}
		chain.Summary.Graph.Nodes = make([]reportbiz.IndustryChainTopologyNode, impactCount)
		chain.Detail.AffectedNodes = make([]reportbiz.IndustryChainNode, impactCount)
		for nodeIndex := 0; nodeIndex < impactCount; nodeIndex++ {
			nodeKey := fmt.Sprintf("topology-node-%02d-%02d", chainIndex+1, nodeIndex+1)
			impactKey := fmt.Sprintf("impact-%02d-%02d", chainIndex+1, nodeIndex+1)
			name := fmt.Sprintf("产业节点 %02d-%02d", chainIndex+1, nodeIndex+1)
			ids := []string{evidenceIDs[impactIndex%len(evidenceIDs)]}
			if impactIndex < 50 {
				ids = append(ids, evidenceIDs[(impactIndex+1)%len(evidenceIDs)])
			}
			chain.Summary.Graph.Nodes[nodeIndex] = reportbiz.IndustryChainTopologyNode{LocalKey: nodeKey, Name: name}
			chain.Detail.AffectedNodes[nodeIndex] = reportbiz.IndustryChainNode{
				LocalKey: impactKey, NodeLocalKey: nodeKey, Name: name, Impact: "需求 UP",
				Result: result(reportbiz.ResultWarming, "升温"), ConclusionBasis: label(reportbiz.BasisDirectEvidence, "直接证据"),
				TransmissionLogic: "目标节点具有直接 Signal。", TimeWindow: window("medium", "中期"),
				Confidence: confidence("medium", "中", 0.72), EvidenceIDs: ids,
			}
			impactIndex++
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

// These aliases keep test call sites concise while the production contract uses
// the domain name Report rather than the retired content envelope.
func Content() reportbiz.Report                        { return Report() }
func IndustryOnlyContent() reportbiz.Report            { return IndustryOnlyReport() }
func ContentWithManyChains(count int) reportbiz.Report { return ReportWithManyChains(count) }

func layer(transmissionKey, title, conclusion, anchorKey, evidenceID, chainKey string) *reportbiz.Layer {
	return &reportbiz.Layer{
		Title: title,
		Summary: reportbiz.LayerSummary{
			Conclusion: conclusion,
			Result:     result(reportbiz.ResultWarming, "升温"),
			Confidence: confidence("high", "高", 0.88),
			TimeWindow: window("immediate_medium", "即时–中期"),
			DownwardTransmission: []reportbiz.Transmission{{
				LocalKey:         transmissionKey,
				SourceConclusion: conclusion,
				Targets: []reportbiz.TransmissionTarget{{
					Type: "industry_chain", LocalKey: chainKey, Name: "产业链 01",
					Result: result(reportbiz.ResultDiverging, "分化"),
				}},
				TransmissionLogic: "风险通过供需与成本向下传导。",
				TransmissionKind:  coded(reportbiz.TransmissionCrossLayer, "跨层推理"),
				Confidence:        confidence("high", "高", 0.88),
				Status:            coded("published", "已发布"),
			}},
			Uncertainty: reportbiz.LayerUncertainty{
				Counterevidence:   text("替代供应可能削弱该结论。"),
				EvidenceGap:       text("仍需目标节点经营数据验证。"),
				Boundary:          text("不扩展到缺少直接 Signal 的目标。"),
				ReversalCondition: text("若关键 Signal 失效，则下调结论。"),
			},
			EvidenceIDs: []string{evidenceID},
		},
		Detail: reportbiz.LayerAnalysis{
			AffectedAnchors: []reportbiz.Anchor{{
				LocalKey: anchorKey, Name: title + "影响锚点", CurrentState: "风险指标 UP",
				Result: result(reportbiz.ResultWarming, "升温"), ConclusionBasis: label(reportbiz.BasisDirectEvidence, "直接证据"),
				TransmissionLogic: "公开事实直接支持该锚点结论。", TimeWindow: window("short", "短期"),
				Confidence: confidence("high", "高", 0.88), EvidenceIDs: []string{evidenceID},
			}},
			ReasoningSteps: []reportbiz.ReasoningStep{},
		},
	}
}

func industryChain(order int) reportbiz.IndustryChain {
	chainKey := fmt.Sprintf("chain-%02d", order)
	topologyKey := fmt.Sprintf("topology-node-%02d", order)
	impactKey := fmt.Sprintf("impact-%02d", order)
	return reportbiz.IndustryChain{
		LocalKey: chainKey,
		Name:     fmt.Sprintf("产业链 %02d", order),
		Summary: reportbiz.ChainSummary{
			Conclusion: "产业链聚合结果升温。", Status: "已形成可解释的动态传导假设",
			Result: result(reportbiz.ResultWarming, "升温"), Confidence: confidence("medium", "中", 0.72),
			TimeWindow: window("medium_long", "中期–长期"), Path: "输入→节点→产出",
			Graph: reportbiz.IndustryChainGraph{
				Nodes: []reportbiz.IndustryChainTopologyNode{{LocalKey: topologyKey, Name: fmt.Sprintf("产业节点 %02d", order)}},
				Edges: []reportbiz.IndustryChainEdge{},
			},
			CounterevidenceAndGap: "反证与缺口仍需经营数据验证。",
			StopCondition:         "若后续 Signal 失效或方向反转，停止该链结论。",
			EvidenceIDs:           []string{EvidenceOne},
		},
		Detail: reportbiz.IndustryChainAnalysis{AffectedNodes: []reportbiz.IndustryChainNode{{
			LocalKey: impactKey, NodeLocalKey: topologyKey, Name: fmt.Sprintf("产业节点 %02d", order), Impact: "需求 UP",
			Result: result(reportbiz.ResultWarming, "升温"), ConclusionBasis: label(reportbiz.BasisDirectEvidence, "直接证据"),
			TransmissionLogic: "目标节点具有直接 Signal。", TimeWindow: window("medium", "中期"),
			Confidence: confidence("medium", "中", 0.72), EvidenceIDs: []string{EvidenceTwo},
		}}},
	}
}

func result(code, label string) reportbiz.CodedLabel { return coded(code, label) }
func coded(code, label string) reportbiz.CodedLabel {
	return reportbiz.CodedLabel{Code: code, Label: label}
}
func label(code, value string) *reportbiz.CodedLabel { item := coded(code, value); return &item }
func confidence(code, label string, score float64) reportbiz.Confidence {
	return reportbiz.Confidence{Code: code, Label: label, Score: &score}
}
func window(code, label string) reportbiz.TimeWindow {
	return reportbiz.TimeWindow{Code: code, Label: label}
}
func text(value string) *string { return &value }
