package main

import "testing"

func TestParseSchedulerOptions(t *testing.T) {
	options, err := parseSchedulerOptions([]string{
		"-once",
		"-dry-run",
		"-tick-seconds", "45",
		"-rsshub-base-url", "https://rsshub.example.com",
		"-prompt-root", "data/prompts",
	})
	if err != nil {
		t.Fatalf("parseSchedulerOptions() error = %v", err)
	}

	if !options.once {
		t.Fatal("once = false, want true")
	}
	if !options.dryRun {
		t.Fatal("dryRun = false, want true")
	}
	if options.tickSeconds != 45 {
		t.Fatalf("tickSeconds = %d, want 45", options.tickSeconds)
	}
	if options.rsshubBaseURL != "https://rsshub.example.com" {
		t.Fatalf("rsshubBaseURL = %q", options.rsshubBaseURL)
	}
	if options.promptRoot != "data/prompts" {
		t.Fatalf("promptRoot = %q", options.promptRoot)
	}
}

func TestNormalizeTickSeconds(t *testing.T) {
	if got := normalizeTickSeconds(0, 30); got != 30 {
		t.Fatalf("normalizeTickSeconds() = %d, want config default", got)
	}
	if got := normalizeTickSeconds(15, 30); got != 15 {
		t.Fatalf("normalizeTickSeconds() = %d, want override", got)
	}
	if got := normalizeTickSeconds(-1, 30); got != 30 {
		t.Fatalf("normalizeTickSeconds() = %d, want config default for negative override", got)
	}
}

func TestSchedulerHTTPTimeoutSecondsAllowsLongAIRequests(t *testing.T) {
	if got := schedulerHTTPTimeoutSeconds(10); got != 180 {
		t.Fatalf("schedulerHTTPTimeoutSeconds() = %d, want 180", got)
	}
	if got := schedulerHTTPTimeoutSeconds(240); got != 240 {
		t.Fatalf("schedulerHTTPTimeoutSeconds() = %d, want config value", got)
	}
}
