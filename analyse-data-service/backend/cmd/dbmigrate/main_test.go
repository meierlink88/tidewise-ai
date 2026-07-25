package main

import (
	"strings"
	"testing"
)

func TestParseCLIOptionsAcceptsAuditableTargetVersion(t *testing.T) {
	options, err := parseCLIOptions([]string{"-apply", "-target-version", "15"})
	if err != nil {
		t.Fatal(err)
	}
	if !options.AutoApply || options.TargetVersion != "15" {
		t.Fatalf("options = %+v", options)
	}
}

func TestParseCLIOptionsRejectsTargetWithoutApply(t *testing.T) {
	_, err := parseCLIOptions([]string{"-target-version", "15"})
	if err == nil || !strings.Contains(err.Error(), "requires -apply") {
		t.Fatalf("error = %v", err)
	}
}
