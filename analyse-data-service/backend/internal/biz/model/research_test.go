package model

import "testing"

func TestResearchEnumValuesRemainWireStable(t *testing.T) {
	values := map[string]string{
		"positive":       string(ResearchImpactPositive),
		"medium":         string(ResearchImpactMedium),
		"identification": string(TransmissionStageIdentification),
		"validation":     string(TransmissionStageValidation),
		"diffusion":      string(TransmissionStageDiffusion),
		"dampening":      string(TransmissionStageDampening),
	}
	for want, got := range values {
		if got != want {
			t.Fatalf("enum value = %q, want %q", got, want)
		}
	}
}
