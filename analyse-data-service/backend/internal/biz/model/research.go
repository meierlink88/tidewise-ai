package model

type ResearchImpactDirection string
type ResearchImpactStrength string
type TransmissionStage string
type ResearchAttentionLevel string
type ResearchConclusionStatus string
type InvestmentGuidanceAction string
type TimeHorizonCategory string
type ResearchRelationRole string
type ResearchEvidenceRole string
type ResearchSignalRole string
type ResearchSignalDirection string

const (
	ResearchImpactPositive  ResearchImpactDirection = "positive"
	ResearchImpactNegative  ResearchImpactDirection = "negative"
	ResearchImpactMixed     ResearchImpactDirection = "mixed"
	ResearchImpactNeutral   ResearchImpactDirection = "neutral"
	ResearchImpactUncertain ResearchImpactDirection = "uncertain"

	ResearchImpactStrong  ResearchImpactStrength = "strong"
	ResearchImpactMedium  ResearchImpactStrength = "medium"
	ResearchImpactWeak    ResearchImpactStrength = "weak"
	ResearchImpactUnknown ResearchImpactStrength = "unknown"

	TransmissionStageIdentification TransmissionStage = "identification"
	TransmissionStageValidation     TransmissionStage = "validation"
	TransmissionStageDiffusion      TransmissionStage = "diffusion"
	TransmissionStageDampening      TransmissionStage = "dampening"
)
