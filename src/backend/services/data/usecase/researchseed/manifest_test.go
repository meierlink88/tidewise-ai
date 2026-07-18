package researchseed

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestLocalHomepageManifestPreservesReviewedReportContract(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "research_themes", "local_homepage.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got, want := len(manifest.Themes), 3; got != want {
		t.Fatalf("theme count = %d, want %d", got, want)
	}

	wantNames := []string{"AI算力扩产与半导体", "中东冲突与能源风险", "AI应用商业化与治理"}
	for index, theme := range manifest.Themes {
		if theme.Name != wantNames[index] {
			t.Fatalf("theme[%d].name = %q, want %q", index, theme.Name, wantNames[index])
		}
		if strings.Contains(theme.TransmissionPath, "；") || strings.Contains(theme.TransmissionPath, ";") {
			t.Fatalf("theme[%d] transmission path still contains the report field separator", index)
		}
		if strings.TrimSpace(theme.TradingDirection) == "" || strings.TrimSpace(theme.NextCheckpoint) == "" {
			t.Fatalf("theme[%d] is missing trading direction or checkpoint", index)
		}
		if len(theme.ChainNodes) == 0 || len(theme.Events) == 0 {
			t.Fatalf("theme[%d] must reference existing chain nodes and events", index)
		}
	}

	if manifest.Themes[0].TransmissionStage != domain.TransmissionStageDiffusion {
		t.Fatalf("first theme stage = %q", manifest.Themes[0].TransmissionStage)
	}
	if manifest.Themes[1].TransmissionStage != domain.TransmissionStageValidation || manifest.Themes[2].TransmissionStage != domain.TransmissionStageValidation {
		t.Fatalf("validation themes stages = %q/%q", manifest.Themes[1].TransmissionStage, manifest.Themes[2].TransmissionStage)
	}
}

func TestManifestRejectsDuplicateReferencesAndInvalidTheme(t *testing.T) {
	manifest := Manifest{
		AnalysisBatchID: "batch",
		Themes: []Theme{{
			ID: "11111111-1111-5111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
			ImpactLevel: domain.ImpactLevelHigh, TransmissionPath: "事件 -> 影响",
			TradingDirection: "等待验证", TransmissionStage: domain.TransmissionStageValidation,
			NextCheckpoint: "尚未显现", IndexImpactSummary: "未观察",
			ChainNodes: []ChainNodeReference{{Name: "算力", RelationRole: domain.ResearchRelationDriver, ImpactSummary: "影响"}, {Name: "算力", RelationRole: domain.ResearchRelationDriver, ImpactSummary: "重复"}},
			Events:     []EventReference{{ID: "22222222-2222-5222-8222-222222222222", EvidenceRole: domain.ResearchEvidenceDriver, SupportedClaim: "支持"}},
		}},
	}

	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate chain node") {
		t.Fatalf("Validate() error = %v, want duplicate chain node", err)
	}
}
