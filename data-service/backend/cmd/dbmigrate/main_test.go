package main

import (
	"strings"
	"testing"
)

func TestParseCLIOptionsRequiresBoundedApplyForEmptySchemaRebuild(t *testing.T) {
	options, err := parseCLIOptions([]string{"-apply", "-target-version", "58", "-rebuild-empty-schema"})
	if err != nil {
		t.Fatal(err)
	}
	if !options.AutoApply || options.TargetVersion != "58" || !options.RebuildEmptySchema {
		t.Fatalf("unexpected rebuild options: %+v", options)
	}

	for _, arguments := range [][]string{
		{"-rebuild-empty-schema"},
		{"-apply", "-rebuild-empty-schema"},
		{"-apply", "-target-version", "57", "-rebuild-empty-schema"},
	} {
		if _, err := parseCLIOptions(arguments); err == nil {
			t.Fatalf("parseCLIOptions(%v) unexpectedly allowed an unbounded rebuild", arguments)
		}
	}
}

func TestValidateEmptySchemaRebuildConfirmation(t *testing.T) {
	options, err := parseCLIOptions([]string{"-apply", "-target-version", "58", "-rebuild-empty-schema"})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateEmptySchemaRebuildConfirmation(options, func(string) string { return "" }); err == nil || !strings.Contains(err.Error(), "confirmation") {
		t.Fatalf("missing confirmation error = %v", err)
	}
	if err := validateEmptySchemaRebuildConfirmation(options, func(string) string { return "issue-266-data-only" }); err != nil {
		t.Fatalf("valid confirmation failed: %v", err)
	}
}
