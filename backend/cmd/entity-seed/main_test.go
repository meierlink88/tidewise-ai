package main

import (
	"testing"

	entityseed "github.com/meierlink88/tidewise-ai/backend/internal/apps/entityfoundation/seed"
)

func TestValidateCommandOptionsAcceptsExplicitMasterScope(t *testing.T) {
	scope, err := validateCommandOptions(commandOptions{applyScope: "industry-chain-master"})
	if err != nil {
		t.Fatalf("validateCommandOptions() error = %v", err)
	}
	if scope != entityseed.ApplyScopeIndustryChainMaster {
		t.Fatalf("scope = %q", scope)
	}
}

func TestValidateCommandOptionsAcceptsExplicitMembershipScope(t *testing.T) {
	scope, err := validateCommandOptions(commandOptions{applyScope: "industry-chain-membership"})
	if err != nil {
		t.Fatalf("validateCommandOptions() error = %v", err)
	}
	if scope != entityseed.ApplyScopeIndustryChainMembership {
		t.Fatalf("scope = %q", scope)
	}
}

func TestValidateCommandOptionsAcceptsExplicitTopologyScope(t *testing.T) {
	scope, err := validateCommandOptions(commandOptions{applyScope: "industry-chain-topology"})
	if err != nil {
		t.Fatalf("validateCommandOptions() error = %v", err)
	}
	if scope != entityseed.ApplyScopeIndustryChainTopology {
		t.Fatalf("scope = %q", scope)
	}
}

func TestValidateCommandOptionsRejectsUnknownAndConflictingScopes(t *testing.T) {
	tests := []commandOptions{
		{applyScope: "membership-only"},
		{applyScope: "industry-chain-master", applySectorConvergence: true},
		{applyScope: "industry-chain-master", applySectorConvergenceCorrection: true},
		{applySectorConvergence: true, applySectorConvergenceCorrection: true},
	}
	for _, options := range tests {
		if _, err := validateCommandOptions(options); err == nil {
			t.Fatalf("validateCommandOptions(%+v) error = nil", options)
		}
	}
}
