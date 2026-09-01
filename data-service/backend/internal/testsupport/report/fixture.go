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
	geoResult := warming()
	macroResult := diverging()
	chainResult := cooling()
	geoConfidence := confidence("高", 0.88)
	macroConfidence := confidence("中", 0.66)
	geoAnchor := reportbiz.Anchor{
		Key: "geo-anchor", DisplayOrder: 1, Name: "地缘锚点", CurrentState: "风险抬升",
		Result: geoResult, Nature: direct(), Reasoning: "公开事实支持风险抬升。",
		TimeWindow: "短期", Confidence: geoConfidence, EvidenceRefs: refs(EvidenceOne),
	}
	macroAnchor := reportbiz.Anchor{
		Key: "macro-anchor", DisplayOrder: 1, Name: "宏观锚点", CurrentState: "路径分化",
		Result: macroResult, Nature: hypothesis(), Reasoning: "宏观路径仍然分化。",
		TimeWindow: "中期", Confidence: macroConfidence, EvidenceRefs: refs(EvidenceTwo),
	}
	chain := industryChain(1)
	geoLayer := reportbiz.Layer{
		Key: "geopolitics", DisplayOrder: 1, Title: "地缘政治", Conclusion: "地缘风险升温。",
		Result: geoResult, Confidence: geoConfidence, TimeWindow: "短期",
		Anchors: []reportbiz.Anchor{geoAnchor},
		ReasoningSteps: []reportbiz.ReasoningStep{{
			Key: "geo-step", DisplayOrder: 1, Input: "地缘事实", Mechanism: "风险传导",
			Output: "风险溢价上升", Type: "causal", Confidence: geoConfidence, EvidenceRefs: refs(EvidenceOne),
		}},
		RelatedAnchorKeys: []string{"geo-anchor"}, RelatedChainKeys: []string{"chain-01"},
		DownwardTransmission: reportbiz.DownwardTransmission{
			Summary: "风险向产业链传导。",
			PublishedPaths: []reportbiz.TransmissionPath{{
				Key: "geo-path", DisplayOrder: 1, SourceConclusion: "地缘风险升温。",
				TargetRefs: []reportbiz.TransmissionTarget{
					{Ref: reportbiz.TargetReference{Type: reportbiz.TargetIndustryChain, Key: "chain-01"}, Label: "链级目标", Result: chainResult},
					{Ref: reportbiz.TargetReference{Type: reportbiz.TargetIndustryChainNode, Key: "node-01"}, Label: "节点目标", Result: chainResult},
				},
				Logic: "风险影响供应。", RelationNature: "推理关系", EvidenceRole: "传导依据",
				Confidence: geoConfidence, Status: "published", EvidenceRefs: refs(EvidenceOne),
			}},
			CandidateMechanisms: []reportbiz.CandidateMechanism{{
				Key: "geo-candidate", DisplayOrder: 1, Mechanism: "替代路径待验证",
				EvidenceGap: stringPointer("缺少后续事实"), Confidence: macroConfidence, EvidenceRefs: refs(EvidenceTwo),
			}},
			BoundaryNotes: []string{},
		},
		Uncertainty:  reportbiz.LayerUncertainty{Checkpoints: []reportbiz.Checkpoint{}},
		EvidenceRefs: refs(EvidenceTwo),
	}
	macroLayer := reportbiz.Layer{
		Key: "macroeconomics", DisplayOrder: 2, Title: "宏观经济", Conclusion: "宏观路径分化。",
		Result: macroResult, Confidence: macroConfidence, TimeWindow: "中期",
		Anchors: []reportbiz.Anchor{macroAnchor},
		ReasoningSteps: []reportbiz.ReasoningStep{{
			Key: "macro-step", DisplayOrder: 1, Input: "宏观事实", Mechanism: "需求传导",
			Output: "需求路径分化", Type: "causal", Confidence: macroConfidence, EvidenceRefs: refs(EvidenceTwo),
		}},
		RelatedAnchorKeys: []string{"macro-anchor"}, RelatedChainKeys: []string{"chain-01"},
		DownwardTransmission: reportbiz.DownwardTransmission{
			Summary: "宏观条件向产业链传导。",
			PublishedPaths: []reportbiz.TransmissionPath{{
				Key: "macro-path", DisplayOrder: 1, SourceConclusion: "宏观路径分化。",
				TargetRefs: []reportbiz.TransmissionTarget{{
					Ref: reportbiz.TargetReference{Type: reportbiz.TargetAnchor, Key: "macro-anchor"}, Label: "宏观锚点", Result: macroResult,
				}},
				Logic: "条件影响需求。", RelationNature: "推理关系", EvidenceRole: "传导依据",
				Confidence: macroConfidence, Status: "published", EvidenceRefs: refs(EvidenceTwo),
			}},
			CandidateMechanisms: []reportbiz.CandidateMechanism{{
				Key: "macro-candidate", DisplayOrder: 1, Mechanism: "需求路径待验证",
				EvidenceGap: nil, Confidence: macroConfidence, EvidenceRefs: refs(EvidenceOne),
			}},
			BoundaryNotes: []string{},
		},
		Uncertainty:  reportbiz.LayerUncertainty{Checkpoints: []reportbiz.Checkpoint{}},
		EvidenceRefs: refs(EvidenceOne),
	}
	return reportbiz.Content{
		ReportType: "investment_reasoning", Title: "每日推理报告", Status: "published", Simulation: false,
		GeneratedAt: time.Date(2026, 9, 1, 8, 30, 0, 0, time.FixedZone("CST", 8*60*60)), Timezone: "Asia/Shanghai",
		PublishedLayers: []string{"geopolitics", "macroeconomics", "industry_chain"},
		Statistics: reportbiz.Statistics{
			AdaptiveInclusionThreshold: 0.6, AdaptiveContinuationThreshold: 0.5,
			GeopoliticAnchorCount: 1, MacroeconomicAnchorCount: 1, IndustryChainCount: 1,
		},
		ReportCards: []reportbiz.ReportCard{
			card("geo-card", reportbiz.CardGeopolitics, 1, reportbiz.TargetReference{Type: reportbiz.TargetLayer, Key: "geopolitics"}, geoLayer.Title, geoLayer.Conclusion, geoLayer.Result, geoLayer.Confidence, geoLayer.TimeWindow, geoAnchor, EvidenceOne),
			card("macro-card", reportbiz.CardMacroeconomics, 2, reportbiz.TargetReference{Type: reportbiz.TargetLayer, Key: "macroeconomics"}, macroLayer.Title, macroLayer.Conclusion, macroLayer.Result, macroLayer.Confidence, macroLayer.TimeWindow, macroAnchor, EvidenceTwo),
			chainCard(chain, 3),
		},
		Geopolitics: geoLayer, Macroeconomics: macroLayer,
		IndustryChains: []reportbiz.IndustryChain{chain},
		Company:        reportbiz.CompanyBoundary{Key: "company", DisplayOrder: 4, Title: "公司", Published: false, Boundary: "本版本不发布公司层。"},
	}
}

func ContentWithManyChains(count int) reportbiz.Content {
	content := Content()
	content.IndustryChains = make([]reportbiz.IndustryChain, count)
	for index := range content.IndustryChains {
		content.IndustryChains[index] = industryChain(index + 1)
	}
	content.Statistics.IndustryChainCount = count
	content.ReportCards = content.ReportCards[:2]
	if count > 0 {
		content.ReportCards = append(content.ReportCards, chainCard(content.IndustryChains[0], 3))
	}
	if count > 1 {
		content.ReportCards = append(content.ReportCards, chainCard(content.IndustryChains[count-1], 4))
	}
	return content
}

func industryChain(order int) reportbiz.IndustryChain {
	key := fmt.Sprintf("chain-%02d", order)
	nodeKey := fmt.Sprintf("node-%02d", order)
	result := cooling()
	confidence := confidence("中", 0.72)
	return reportbiz.IndustryChain{
		Key: key, ClaimKey: fmt.Sprintf("claim-%02d", order), DisplayOrder: order,
		Name: fmt.Sprintf("产业链 %02d", order), Conclusion: "产业链压力降温。", Status: "published",
		Result: result, Confidence: confidence, TimeWindow: "长期", PathSummary: nil,
		AcceptedHypothesisSummary: nil, EvidenceRefs: refs(EvidenceOne),
		Nodes: []reportbiz.IndustryChainNode{{
			Key: nodeKey, DisplayOrder: 1, Name: fmt.Sprintf("产业节点 %02d", order), Impact: "供应压力缓解",
			Result: result, Nature: hypothesis(), Reasoning: "供应条件改善。", TimeWindow: "长期",
			Confidence: confidence, EvidenceRefs: []reportbiz.EvidenceReference{
				{EvidenceID: EvidenceOne, Role: "节点依据一", DisplayOrder: 1},
				{EvidenceID: EvidenceTwo, Role: "节点依据二", DisplayOrder: 2},
			},
		}},
		Edges:       []reportbiz.IndustryChainEdge{},
		Uncertainty: reportbiz.ChainUncertainty{Checkpoints: []reportbiz.Checkpoint{}},
	}
}

func chainCard(chain reportbiz.IndustryChain, order int) reportbiz.ReportCard {
	node := chain.Nodes[0]
	return reportbiz.ReportCard{
		Key: fmt.Sprintf("%s-card", chain.Key), Kind: reportbiz.CardIndustryChain, DisplayOrder: order,
		DetailRef: reportbiz.TargetReference{Type: reportbiz.TargetIndustryChain, Key: chain.Key},
		Title:     chain.Name, Subtitle: "产业链", Conclusion: chain.Conclusion, Result: chain.Result,
		Confidence: chain.Confidence, TimeWindow: chain.TimeWindow,
		ImpactItems: []reportbiz.ImpactItem{{
			Ref: reportbiz.TargetReference{Type: reportbiz.TargetIndustryChainNode, Key: node.Key}, Name: node.Name,
			Result: node.Result, Confidence: node.Confidence, TimeWindow: node.TimeWindow,
		}},
		EvidenceRefs: refs(EvidenceOne),
	}
}

func card(key string, kind reportbiz.CardKind, order int, detail reportbiz.TargetReference, title, conclusion string, result reportbiz.Result, confidence reportbiz.Confidence, window string, anchor reportbiz.Anchor, evidenceID string) reportbiz.ReportCard {
	return reportbiz.ReportCard{
		Key: key, Kind: kind, DisplayOrder: order, DetailRef: detail, Title: title, Subtitle: "推理层",
		Conclusion: conclusion, Result: result, Confidence: confidence, TimeWindow: window,
		ImpactItems: []reportbiz.ImpactItem{{
			Ref: reportbiz.TargetReference{Type: reportbiz.TargetAnchor, Key: anchor.Key}, Name: anchor.Name,
			Result: anchor.Result, Confidence: anchor.Confidence, TimeWindow: anchor.TimeWindow,
		}},
		EvidenceRefs: refs(evidenceID),
	}
}

func refs(evidenceID string) []reportbiz.EvidenceReference {
	return []reportbiz.EvidenceReference{{EvidenceID: evidenceID, Role: "直接依据", DisplayOrder: 1}}
}

func confidence(label string, score float64) reportbiz.Confidence {
	return reportbiz.Confidence{Label: label, Score: &score}
}

func warming() reportbiz.Result {
	return reportbiz.Result{Code: reportbiz.ResultWarming, Label: "升温"}
}
func cooling() reportbiz.Result {
	return reportbiz.Result{Code: reportbiz.ResultCooling, Label: "降温"}
}
func diverging() reportbiz.Result {
	return reportbiz.Result{Code: reportbiz.ResultDiverging, Label: "分化"}
}
func direct() reportbiz.Nature {
	return reportbiz.Nature{Code: reportbiz.NatureDirectEvidence, Label: "直接证据"}
}
func hypothesis() reportbiz.Nature {
	return reportbiz.Nature{Code: reportbiz.NatureReasoningHypothesis, Label: "推理假设"}
}

func stringPointer(value string) *string { return &value }
