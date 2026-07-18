package researchseed

import "github.com/meierlink88/tidewise-ai/backend/services/data/domain"

func validTestManifest() Manifest {
	return Manifest{
		AnalysisBatchID: "batch",
		Themes: []Theme{{
			ID: "11111111-1111-5111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
			ImpactLevel: domain.ImpactLevelHigh, TransmissionPath: "事件 -> 影响",
			TradingDirection: "等待验证", TransmissionStage: domain.TransmissionStageValidation,
			NextCheckpoint: "尚未显现", IndexImpactSummary: "未观察",
			ChainNodes: []ChainNodeReference{{Name: "算力", RelationRole: domain.ResearchRelationDriver, ImpactSummary: "影响"}},
			Events:     []EventReference{{ID: "22222222-2222-5222-8222-222222222222", EvidenceRole: domain.ResearchEvidenceDriver, SupportedClaim: "支持"}},
		}},
	}
}
