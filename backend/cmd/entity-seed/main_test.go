package main

import "testing"

func TestValidateCommandOptionsRejectsRetiredApplyScopes(t *testing.T) {
	for _, retired := range []string{
		"industry-chain-master",
		"industry-chain-membership",
		"industry-chain-topology",
		"industry-chain-physical-constraint",
		"industry-chain-sector-mapping",
	} {
		if _, err := validateCommandOptions(commandOptions{applyScope: retired}); err == nil {
			t.Fatalf("validateCommandOptions(%q) error = nil", retired)
		}
	}
}
