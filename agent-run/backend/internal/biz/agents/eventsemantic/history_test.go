package eventsemantic

import (
	"testing"
	"time"
)

func TestHistoricalManifestRequiresDisjointValidEventIdentities(t *testing.T) {
	valid := "11111111-1111-4111-8111-111111111111"
	invalid := "22222222-2222-4222-8222-222222222222"
	manifest := HistoricalManifest{
		Version:         HistoricalManifestVersion,
		GeneratedAt:     time.Now(),
		ValidEventIDs:   []string{valid},
		InvalidEventIDs: []string{invalid},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("valid manifest: %v", err)
	}

	for name, candidate := range map[string]HistoricalManifest{
		"unknown version": {
			Version: "other", GeneratedAt: time.Now(), ValidEventIDs: []string{valid},
		},
		"missing generation time": {
			Version: HistoricalManifestVersion, ValidEventIDs: []string{valid},
		},
		"invalid uuid": {
			Version: HistoricalManifestVersion, GeneratedAt: time.Now(),
			InvalidEventIDs: []string{"not-an-id"},
		},
		"overlap": {
			Version: HistoricalManifestVersion, GeneratedAt: time.Now(),
			ValidEventIDs: []string{valid}, InvalidEventIDs: []string{valid},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := candidate.Validate(); err == nil {
				t.Fatalf("manifest %#v was accepted", candidate)
			}
		})
	}
}
