package seed

import "testing"

func TestParseApplyScopeAllowsOnlyUnifiedProductionSeed(t *testing.T) {
	if _, err := ParseApplyScope(""); err != nil {
		t.Fatalf("ParseApplyScope(empty) error = %v", err)
	}
	for _, retired := range []string{
		"industry-chain-master",
		"industry-chain-membership",
		"industry-chain-topology",
		"industry-chain-physical-constraint",
		"industry-chain-sector-mapping",
	} {
		if _, err := ParseApplyScope(retired); err == nil {
			t.Fatalf("ParseApplyScope(%q) error = nil", retired)
		}
	}
}
