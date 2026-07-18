package researchseed

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

const DefaultManifestPath = "data/research_themes/local_homepage.json"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type Manifest struct {
	AnalysisBatchID string  `json:"analysis_batch_id"`
	Themes          []Theme `json:"themes"`
}

type Theme struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	OneLineConclusion  string                   `json:"one_line_conclusion"`
	ImpactLevel        domain.ImpactLevel       `json:"impact_level"`
	TransmissionPath   string                   `json:"transmission_path"`
	TradingDirection   string                   `json:"trading_direction"`
	TransmissionStage  domain.TransmissionStage `json:"transmission_stage"`
	NextCheckpoint     string                   `json:"next_checkpoint"`
	IndexImpactSummary string                   `json:"index_impact_summary"`
	ChainNodes         []ChainNodeReference     `json:"chain_nodes"`
	Events             []EventReference         `json:"events"`
}

type ChainNodeReference struct {
	Name          string                      `json:"name"`
	RelationRole  domain.ResearchRelationRole `json:"relation_role"`
	ImpactSummary string                      `json:"impact_summary"`
}

type EventReference struct {
	ID             string                      `json:"id"`
	EvidenceRole   domain.ResearchEvidenceRole `json:"evidence_role"`
	SupportedClaim string                      `json:"supported_claim"`
}

func LoadFile(path string) (Manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read research theme manifest: %w", err)
	}
	var manifest Manifest
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode research theme manifest: %w", err)
	}
	return manifest, nil
}

func (m Manifest) Validate() error {
	if strings.TrimSpace(m.AnalysisBatchID) == "" {
		return fmt.Errorf("analysis_batch_id is required")
	}
	if len(m.Themes) == 0 {
		return fmt.Errorf("at least one research theme is required")
	}
	seenThemeIDs := make(map[string]struct{}, len(m.Themes))
	for index, theme := range m.Themes {
		if _, exists := seenThemeIDs[theme.ID]; exists {
			return fmt.Errorf("theme[%d] has duplicate id %q", index, theme.ID)
		}
		seenThemeIDs[theme.ID] = struct{}{}
		if err := theme.Validate(m.AnalysisBatchID); err != nil {
			return fmt.Errorf("theme[%d]: %w", index, err)
		}
	}
	return nil
}

func (t Theme) Validate(batchID string) error {
	if !uuidPattern.MatchString(t.ID) {
		return fmt.Errorf("id must be a UUID")
	}
	researchTheme := domain.ResearchTheme{
		ID: t.ID, AnalysisBatchID: batchID, Name: t.Name, OneLineConclusion: t.OneLineConclusion,
		ImpactLevel: t.ImpactLevel, TransmissionPath: t.TransmissionPath, TradingDirection: t.TradingDirection,
		TransmissionStage: t.TransmissionStage, NextCheckpoint: t.NextCheckpoint, IndexImpactSummary: t.IndexImpactSummary,
	}
	if err := researchTheme.Validate(); err != nil {
		return err
	}
	if len(t.ChainNodes) == 0 || len(t.Events) == 0 {
		return fmt.Errorf("chain_nodes and events must not be empty")
	}
	seenNodes := make(map[string]struct{}, len(t.ChainNodes))
	for _, node := range t.ChainNodes {
		name := strings.TrimSpace(node.Name)
		if name == "" || strings.TrimSpace(node.ImpactSummary) == "" || !validRelationRole(node.RelationRole) {
			return fmt.Errorf("invalid chain node reference %q", node.Name)
		}
		if _, exists := seenNodes[name]; exists {
			return fmt.Errorf("duplicate chain node %q", name)
		}
		seenNodes[name] = struct{}{}
	}
	seenEvents := make(map[string]struct{}, len(t.Events))
	for _, event := range t.Events {
		if !uuidPattern.MatchString(event.ID) || strings.TrimSpace(event.SupportedClaim) == "" || !validEvidenceRole(event.EvidenceRole) {
			return fmt.Errorf("invalid event reference %q", event.ID)
		}
		if _, exists := seenEvents[event.ID]; exists {
			return fmt.Errorf("duplicate event %q", event.ID)
		}
		seenEvents[event.ID] = struct{}{}
	}
	return nil
}

func validRelationRole(value domain.ResearchRelationRole) bool {
	switch value {
	case domain.ResearchRelationDriver, domain.ResearchRelationBeneficiary, domain.ResearchRelationConstraint, domain.ResearchRelationExposure:
		return true
	default:
		return false
	}
}

func validEvidenceRole(value domain.ResearchEvidenceRole) bool {
	switch value {
	case domain.ResearchEvidenceDriver, domain.ResearchEvidenceSupporting, domain.ResearchEvidenceContradicting, domain.ResearchEvidenceContext:
		return true
	default:
		return false
	}
}
