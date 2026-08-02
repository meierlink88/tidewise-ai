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

func TestProhibitedJSONKeysAuditsNestedV3BoundaryViolations(t *testing.T) {
	counts := prohibitedJSONKeys([]byte(`{
		"entity_links": [],
		"nested": {"direct_impacts": [], "theme": {}, "reasoning_trees": [], "candidate_type": "direct_impact"}
	}`))
	if counts.directImpact != 2 || counts.theme != 1 || counts.reasonTree != 1 {
		t.Fatalf("prohibited counts = %#v", counts)
	}
	clean := prohibitedJSONKeys([]byte(`{"entity_links":[],"variable_signals":[]}`))
	if clean != (prohibitedCounts{}) {
		t.Fatalf("clean V3 payload counts = %#v", clean)
	}
}

func TestFinalizeCandidateMetricsPreservesPerCandidateDecisions(t *testing.T) {
	report := runReport{EntityRejectionReasons: map[string]int{}, SignalRejectionReasons: map[string]int{}}
	finalizeCandidateMetrics(&report, eventsemantic.SubmissionResult{
		EntityLinks:     []eventsemantic.CandidateDecision{{CandidateKey: "entity", Status: "accepted"}},
		VariableSignals: []eventsemantic.CandidateDecision{{CandidateKey: "signal", Status: "rejected", ReasonCode: "upstream_rejected"}},
	}, map[string]int{"signal": 1})
	if len(report.EntityDecisions) != 1 || report.EntityDecisions[0].CandidateKey != "entity" ||
		len(report.SignalDecisions) != 1 || report.SignalDecisions[0].ReasonCode != "upstream_rejected" {
		t.Fatalf("decisions were not preserved: entities=%#v signals=%#v", report.EntityDecisions, report.SignalDecisions)
	}
}
