package main

import (
	"strings"
	"testing"
)

func TestParseCLIOptions(t *testing.T) {
	options, err := parseCLIOptions([]string{"-apply"})
	if err != nil {
		t.Fatal(err)
	}
	if !options.apply {
		t.Fatalf("unexpected options: %+v", options)
	}

	options, err = parseCLIOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	if options.apply {
		t.Fatalf("check-only unexpectedly applies: %+v", options)
	}
}

func TestValidateApplyConfirmation(t *testing.T) {
	if err := validateApplyConfirmation(cliOptions{}, func(string) string { return "" }); err != nil {
		t.Fatalf("check-only validation failed: %v", err)
	}
	if err := validateApplyConfirmation(cliOptions{apply: true}, func(string) string { return "" }); err == nil || !strings.Contains(err.Error(), "confirmation") {
		t.Fatalf("missing confirmation error = %v", err)
	}
	if err := validateApplyConfirmation(cliOptions{apply: true}, func(string) string { return preV74EvidenceRecoveryConfirmation }); err != nil {
		t.Fatalf("valid confirmation failed: %v", err)
	}
}
