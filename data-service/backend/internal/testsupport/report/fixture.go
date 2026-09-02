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

func Content() reportbiz.Content {
	content := IndustryOnlyContent()
	content.Geopolitics = layer("geopolitics", "地缘政治", "C-GEO-01", "地缘风险升温。", "geo-anchor", EvidenceOne)
	content.Macroeconomics = layer("macroeconomics", "宏观经济", "C-MAC-01", "宏观路径分化。", "macro-anchor", EvidenceTwo)
	content.Statistics.GeopoliticAnchorCount = 1
	content.Statistics.MacroeconomicAnchorCount = 1
	content.Geopolitics.Detail.RelatedChainKeys = []string{"chain-01"}
	content.Macroeconomics.Detail.RelatedChainKeys = []string{"chain-01"}
	content.Geopolitics.Summary.Transmissions[0].Targets[0].Ref = &reportbiz.TargetReference{Type: reportbiz.TargetIndustryChain, Key: "chain-01"}
	content.Macroeconomics.Summary.Transmissions[0].Targets[0].Ref = &reportbiz.TargetReference{Type: reportbiz.TargetIndustryChain, Key: "chain-01"}
	return content
}

func IndustryOnlyContent() reportbiz.Content {
	generatedAt := time.Date(2026, 9, 2, 8, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	return reportbiz.Content{
		ReportType:       "investment_reasoning",
		Title:            "每日投资推理报告",
		GenerationStatus: "completed",
		GeneratedAt:      generatedAt,
		AnalysisWindow: reportbiz.AnalysisWindow{
			StartedAt: generatedAt.Add(-24 * time.Hour), EndedAt: generatedAt,
		},
		Timezone: "Asia/Shanghai",
		Provenance: reportbiz.Provenance{Template: reportbiz.TemplateReference{
			Name: "investment-reasoning-report", Version: "presentation-v2", Role: "publication",
		}},
		Statistics:     reportbiz.Statistics{IndustryChainCount: 1, SignaledChainNodeCount: 1},
		IndustryChains: []reportbiz.IndustryChain{industryChain(1)},
	}
}

func ContentWithManyChains(count int) reportbiz.Content {
	content := IndustryOnlyContent()
	content.IndustryChains = make([]reportbiz.IndustryChain, count)
	for index := range content.IndustryChains {
		content.IndustryChains[index] = industryChain(index + 1)
	}
	content.Statistics.IndustryChainCount = count
	content.Statistics.SignaledChainNodeCount = count
	return content
}

func layer(key, title, claimKey, claimText, anchorKey, evidenceID string) *reportbiz.Layer {
	high := confidence(reportbiz.ConfidenceHigh, "高", 0.88)
	counterevidence := "相反事实可能削弱该结论。"
	boundary := "不把缺少直接 Signal 的对象纳入本节结论。"
	reversal := "若关键 Signal 失效或方向反转，则下调本节结论。"
	return &reportbiz.Layer{
		Key: key, Title: title,
		Summary: reportbiz.LayerSummary{
			Claim: reportbiz.Claim{Key: claimKey, Text: claimText},
			Transmissions: []reportbiz.Transmission{{
				Key: key + "-tx-01", DisplayOrder: 1, SourceClaimKey: claimKey, SourceConclusion: claimText,
				Targets: []reportbiz.TransmissionTarget{{Label: "产业链", Results: []reportbiz.NamedResult{{Name: "供需", Result: warming()}}}},
				Logic:   "风险通过供需与成本向下传导。", RelationNature: "cross_layer_inference",
				Confidence: high, Status: "published", EvidenceRefs: roleRefs(evidenceID, reportbiz.EvidenceRoleSupportsTransmission),
			}},
			Uncertainty: reportbiz.LayerUncertainty{Counterevidence: &counterevidence, Boundary: &boundary, ReversalCondition: &reversal, Checkpoints: []reportbiz.Checkpoint{}}, EvidenceRefs: roleRefs(evidenceID, reportbiz.EvidenceRoleSupportsClaim),
		},
		Detail: reportbiz.LayerAnalysis{
			Anchors: []reportbiz.Anchor{{
				Key: anchorKey, DisplayOrder: 1, Name: title + "影响锚点",
				Effects: []reportbiz.Effect{{DisplayOrder: 1, Dimension: "风险", Direction: reportbiz.DirectionUp, Confidence: reportbiz.SignalConfidenceHigh}},
				Result:  warming(), Nature: direct(), Reasoning: "公开事实直接支持该锚点结论。",
				TimeWindow: window(reportbiz.HorizonShort, "短期"), Confidence: high, EvidenceRefs: refs(evidenceID),
			}},
			ReasoningSteps: []reportbiz.ReasoningStep{}, RelatedChainKeys: []string{},
		},
	}
}

func industryChain(order int) reportbiz.IndustryChain {
	chainKey := fmt.Sprintf("chain-%02d", order)
	topologyKey := fmt.Sprintf("topology-node-%02d", order)
	medium := confidence(reportbiz.ConfidenceMedium, "中", 0.72)
	return reportbiz.IndustryChain{
		Key: chainKey, DisplayOrder: order, Name: fmt.Sprintf("产业链 %02d", order),
		Summary: reportbiz.ChainSummary{
			Claim:  reportbiz.Claim{Key: fmt.Sprintf("C-CHAIN-%02d", order), Text: "产业链聚合结果升温。"},
			Status: "completed", Result: warming(), Confidence: medium,
			TimeWindow: window(reportbiz.HorizonMedium, "中期"), Path: "输入→节点→产出", EvidenceRefs: roleRefs(EvidenceOne, reportbiz.EvidenceRoleSupportsClaim),
			Graph: reportbiz.IndustryChainGraph{
				Nodes: []reportbiz.IndustryChainTopologyNode{{Key: topologyKey, DisplayOrder: 1, Name: fmt.Sprintf("产业节点 %02d", order)}},
				Edges: []reportbiz.IndustryChainEdge{},
			},
			Uncertainty: reportbiz.ChainUncertainty{CounterevidenceAndGap: "反证与缺口仍需经营数据验证。", StopCondition: "若后续 Signal 失效或方向反转，停止该链结论。"},
		},
		Detail: reportbiz.IndustryChainAnalysis{
			NodeImpacts: []reportbiz.IndustryChainNode{{
				Key: fmt.Sprintf("impact-%02d", order), DisplayOrder: 1, NodeKey: topologyKey,
				Effects: []reportbiz.Effect{{DisplayOrder: 1, Dimension: "需求", Direction: reportbiz.DirectionUp, Confidence: reportbiz.SignalConfidenceMedium}},
				Result:  warming(), Nature: direct(), Reasoning: "目标节点具有直接 Signal。",
				TimeWindow: window(reportbiz.HorizonMedium, "中期"), Confidence: medium, EvidenceRefs: refs(EvidenceTwo),
			}},
		},
	}
}

func refs(evidenceID string) []reportbiz.EvidenceReference {
	return roleRefs(evidenceID, reportbiz.EvidenceRoleDirectTarget)
}

func roleRefs(evidenceID string, role reportbiz.EvidenceRoleCode) []reportbiz.EvidenceReference {
	return []reportbiz.EvidenceReference{{EvidenceID: evidenceID, Role: role, DisplayOrder: 1}}
}

func confidence(code reportbiz.ConfidenceCode, label string, score float64) reportbiz.Confidence {
	return reportbiz.Confidence{Code: code, Label: label, Score: &score}
}

func window(horizon reportbiz.HorizonCode, label string) reportbiz.TimeWindow {
	return reportbiz.TimeWindow{Horizons: []reportbiz.HorizonCode{horizon}, Label: label}
}

func warming() reportbiz.Result {
	return reportbiz.Result{Code: reportbiz.ResultWarming, Label: "升温"}
}
func direct() reportbiz.Nature {
	return reportbiz.Nature{Code: reportbiz.NatureDirectEvidence, Label: "直接证据"}
}
