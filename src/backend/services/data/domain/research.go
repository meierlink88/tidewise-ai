package domain

import (
	"fmt"
	"strings"
	"time"
)

type ImpactLevel string

const (
	ImpactLevelHigh  ImpactLevel = "high"
	ImpactLevelFocus ImpactLevel = "focus"
	ImpactLevelWatch ImpactLevel = "watch"
)

type TransmissionStage string

const (
	TransmissionStageIdentification TransmissionStage = "identification"
	TransmissionStageValidation     TransmissionStage = "validation"
	TransmissionStageDiffusion      TransmissionStage = "diffusion"
	TransmissionStageDampening      TransmissionStage = "dampening"
)

type ResearchRelationRole string

const (
	ResearchRelationDriver      ResearchRelationRole = "driver"
	ResearchRelationBeneficiary ResearchRelationRole = "beneficiary"
	ResearchRelationConstraint  ResearchRelationRole = "constraint"
	ResearchRelationExposure    ResearchRelationRole = "exposure"
)

type ResearchImportance string

const (
	ResearchImportancePrimary    ResearchImportance = "primary"
	ResearchImportanceSecondary  ResearchImportance = "secondary"
	ResearchImportanceContextual ResearchImportance = "contextual"
)

type ResearchEvidenceRole string

const (
	ResearchEvidenceDriver        ResearchEvidenceRole = "driver"
	ResearchEvidenceSupporting    ResearchEvidenceRole = "supporting"
	ResearchEvidenceContradicting ResearchEvidenceRole = "contradicting"
	ResearchEvidenceContext       ResearchEvidenceRole = "context"
)

type ResearchImpactDirection string

const (
	ResearchImpactPositive ResearchImpactDirection = "positive"
	ResearchImpactNegative ResearchImpactDirection = "negative"
	ResearchImpactMixed    ResearchImpactDirection = "mixed"
	ResearchImpactNeutral  ResearchImpactDirection = "neutral"
)

type AnchorType string

const (
	AnchorTypePolicy          AnchorType = "policy"
	AnchorTypeSupply          AnchorType = "supply"
	AnchorTypeDemand          AnchorType = "demand"
	AnchorTypeTechnology      AnchorType = "technology"
	AnchorTypeCost            AnchorType = "cost"
	AnchorTypeGeopolitics     AnchorType = "geopolitics"
	AnchorTypeMarketStructure AnchorType = "market_structure"
)

type ResearchTheme struct {
	ID                 string
	AnalysisBatchID    string
	Name               string
	OneLineConclusion  string
	ImpactLevel        ImpactLevel
	TransmissionPath   string
	TradingDirection   string
	TransmissionStage  TransmissionStage
	NextCheckpoint     string
	IndexImpactSummary string
	WindowStart        *time.Time
	WindowEnd          *time.Time
	PublishedAt        *time.Time
}

type ResearchAnchor struct {
	ID                string
	AnalysisBatchID   string
	AnchorType        AnchorType
	Name              string
	OneLineConclusion string
	Importance        ResearchImportance
	TransmissionPath  string
	TradingDirection  string
	PublishedAt       *time.Time
}

func (r ResearchTheme) Validate() error {
	if err := validateResearchIdentity(r.ID, r.AnalysisBatchID); err != nil {
		return err
	}
	for field, value := range map[string]string{
		"name": r.Name, "one_line_conclusion": r.OneLineConclusion, "transmission_path": r.TransmissionPath,
		"trading_direction": r.TradingDirection, "next_checkpoint": r.NextCheckpoint, "index_impact_summary": r.IndexImpactSummary,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if !validResearchValue(r.ImpactLevel, ImpactLevelHigh, ImpactLevelFocus, ImpactLevelWatch) {
		return fmt.Errorf("unsupported impact level %q", r.ImpactLevel)
	}
	if !validResearchValue(r.TransmissionStage, TransmissionStageIdentification, TransmissionStageValidation, TransmissionStageDiffusion, TransmissionStageDampening) {
		return fmt.Errorf("unsupported transmission stage %q", r.TransmissionStage)
	}
	if (r.WindowStart == nil) != (r.WindowEnd == nil) {
		return fmt.Errorf("window start and end must be provided together")
	}
	if r.WindowStart != nil && r.WindowEnd.Before(*r.WindowStart) {
		return fmt.Errorf("window end must be greater than or equal to window start")
	}
	return nil
}

func (r ResearchAnchor) Validate() error {
	if err := validateResearchIdentity(r.ID, r.AnalysisBatchID); err != nil {
		return err
	}
	for field, value := range map[string]string{"name": r.Name, "one_line_conclusion": r.OneLineConclusion, "transmission_path": r.TransmissionPath, "trading_direction": r.TradingDirection} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if !validResearchValue(r.AnchorType, AnchorTypePolicy, AnchorTypeSupply, AnchorTypeDemand, AnchorTypeTechnology, AnchorTypeCost, AnchorTypeGeopolitics, AnchorTypeMarketStructure) {
		return fmt.Errorf("unsupported anchor type %q", r.AnchorType)
	}
	if !validResearchValue(r.Importance, ResearchImportancePrimary, ResearchImportanceSecondary, ResearchImportanceContextual) {
		return fmt.Errorf("unsupported importance %q", r.Importance)
	}
	return nil
}

func validateResearchIdentity(id, batch string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("research id is required")
	}
	if strings.TrimSpace(batch) == "" {
		return fmt.Errorf("analysis batch id is required")
	}
	return nil
}

func validResearchValue[T ~string](value T, allowed ...T) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
