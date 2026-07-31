package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoricalApplyRequiresExplicitEnvironmentAndPreChangeExport(t *testing.T) {
	for _, arguments := range [][]string{
		{"-apply", "-manifest", "audit.json"},
		{"-apply", "-manifest", "audit.json", "-allow-env", "uat"},
		{"-dry-run", "-manifest", "audit.json", "-export", "before.json"},
	} {
		if _, err := parseOptions(arguments); err == nil {
			t.Fatalf("arguments %#v were accepted", arguments)
		}
	}
	options, err := parseOptions([]string{
		"-apply", "-manifest", "audit.json",
		"-allow-env", "uat", "-export", "before.json",
	})
	if err != nil || !options.Apply {
		t.Fatalf("options = %#v err = %v", options, err)
	}
}

func TestPreChangeExportNeverOverwritesReviewedEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "before.json")
	if err := os.WriteFile(path, []byte("reviewed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusiveJSON(path, map[string]int{"changed": 1}); err == nil {
		t.Fatal("existing pre-change export was overwritten")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "reviewed" {
		t.Fatalf("payload = %q", payload)
	}
}
