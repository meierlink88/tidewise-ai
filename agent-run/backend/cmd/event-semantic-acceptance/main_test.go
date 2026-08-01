package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

func TestReadEventIDsFreezesUniqueSample(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.csv")
	if err := os.WriteFile(path, []byte("id,first_seen_at\n11111111-1111-4111-8111-111111111111,2026-08-01\n22222222-2222-4222-8222-222222222222,2026-08-01\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ids, err := readEventIDs(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("ids = %#v", ids)
	}
}

func TestLatestSubmissionIgnoresSupersededHistory(t *testing.T) {
	got := latestSubmissionID([]eventsemantic.SubmissionResult{
		{SubmissionID: "old", Status: "superseded", CreatedAt: "2026-08-01T01:00:00Z"},
		{SubmissionID: "current", Status: "accepted", CreatedAt: "2026-08-01T02:00:00Z"},
	})
	if got != "current" {
		t.Fatalf("latest = %q", got)
	}
}
