package materialization

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

func TestDeterministicContractMatchesCrossLanguageGolden(t *testing.T) {
	var golden struct {
		URL              string `json:"url"`
		CanonicalURL     string `json:"canonical_url"`
		BodyInput        string `json:"body_input"`
		NormalizedBody   string `json:"normalized_body"`
		PublishedAtInput string `json:"published_at_input"`
		PublishedAtUTC   string `json:"published_at_utc"`
		ContentSHA256    string `json:"content_sha256"`
		DocumentID       string `json:"document_id"`
		SimHash64        string `json:"simhash64"`
	}
	payload, err := os.ReadFile("testdata/deterministic_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &golden); err != nil {
		t.Fatal(err)
	}
	if got := CanonicalURL(golden.URL); got != golden.CanonicalURL {
		t.Fatalf("canonical URL = %q, want %q", got, golden.CanonicalURL)
	}
	body := NormalizeBody(golden.BodyInput)
	if body != golden.NormalizedBody {
		t.Fatalf("normalized body = %q, want %q", body, golden.NormalizedBody)
	}
	if got := ParseTime(golden.PublishedAtInput); got.IsZero() || got.Format(time.RFC3339) != golden.PublishedAtUTC {
		t.Fatalf("published_at = %v, want %s", got, golden.PublishedAtUTC)
	}
	contentHash := Hash(body)
	if contentHash != golden.ContentSHA256 {
		t.Fatalf("content SHA-256 = %q, want %q", contentHash, golden.ContentSHA256)
	}
	if got := "sha256:" + Hash(golden.CanonicalURL+"\n"+contentHash); got != golden.DocumentID {
		t.Fatalf("document ID = %q, want %q", got, golden.DocumentID)
	}
	if got := SimHash(body); got != golden.SimHash64 {
		t.Fatalf("SimHash64 = %q, want %q", got, golden.SimHash64)
	}
}

func TestMergeUsesUnicodeRichnessAndStableConnectorOrder(t *testing.T) {
	rows := []collector.Candidate{
		{Connector: "tavily", URL: "https://example.com/item", Title: "标题", Content: "a b      ", ContentLevel: collector.LevelSnippet},
		{Connector: "parallel_search", URL: "https://example.com/item", Title: "标题", Content: "abc", ContentLevel: collector.LevelSnippet},
	}
	for _, input := range [][]collector.Candidate{rows, {rows[1], rows[0]}} {
		merged := Merge(input)
		if len(merged) != 1 || merged[0].PrimaryConnector != "parallel_search" || merged[0].Content != "abc" {
			t.Fatalf("merged = %#v", merged)
		}
	}
}

func TestEvaluateOwnsTerminalCandidateDecisions(t *testing.T) {
	now := time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC)
	candidate := collector.Candidate{
		Connector: "tavily", Title: "政策", URL: "HTTPS://Example.com:443/a?utm_source=x",
		Content: "政策正文", ContentLevel: collector.LevelFullText, PublishedAtHint: "2026-07-25 07:30:00",
	}
	accepted := Evaluate(candidate, now, now.Add(-2*time.Hour), nil, 3)
	if accepted.Disposition != collector.DispositionAccepted ||
		accepted.Candidate.URL != "https://example.com/a" ||
		accepted.DocumentID == "" || accepted.ContentHash == "" || accepted.SimHash == "" {
		t.Fatalf("accepted decision = %#v", accepted)
	}
	for name, test := range map[string]struct {
		candidate collector.Candidate
		records   []ExistingRecord
		want      collector.CandidateDisposition
	}{
		"invalid": {candidate: collector.Candidate{}, want: collector.DispositionInvalidResult},
		"old": {
			candidate: collector.Candidate{
				Title: "旧闻", URL: "https://example.com/old", Content: "旧内容",
				ContentLevel: collector.LevelSnippet, PublishedAtHint: "2026-07-20",
			},
			want: collector.DispositionOutOfWindow,
		},
		"known URL": {
			candidate: candidate,
			records:   []ExistingRecord{{URLHash: accepted.URLHash}},
			want:      collector.DispositionKnownURL,
		},
		"exact": {
			candidate: candidate,
			records:   []ExistingRecord{{ContentHash: accepted.ContentHash}},
			want:      collector.DispositionExactDuplicate,
		},
		"near": {
			candidate: candidate,
			records:   []ExistingRecord{{SimHash: accepted.SimHash}},
			want:      collector.DispositionNearDuplicate,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := Evaluate(test.candidate, now, now.Add(-2*time.Hour), test.records, 3).Disposition; got != test.want {
				t.Fatalf("disposition = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPrimitiveContracts(t *testing.T) {
	if got := SimHash("The ministry announced new semiconductor export controls on Tuesday. The rules cover advanced chips and manufacturing equipment."); got != "5ebccb6498055f7e" {
		t.Fatalf("SimHash = %s", got)
	}
	if got := NormalizeBody("e\u0301\r\nline with spaces  \r\n\r\n\r\nnext\n"); got != "é\nline with spaces\n\nnext" {
		t.Fatalf("normalized body = %q", got)
	}
	if got := CanonicalURL("https://EXAMPLE.com"); got != "https://example.com/" {
		t.Fatalf("canonical URL = %q", got)
	}
	for input, want := range map[string]string{
		"2026-07-22 12:03:44":       "2026-07-22T12:03:44Z",
		"2026-07-22T12:03:44Z":      "2026-07-22T12:03:44Z",
		"2026-07-22T20:03:44+08:00": "2026-07-22T12:03:44Z",
		"2026-07-22":                "2026-07-22T00:00:00Z",
	} {
		if got := ParseTime(input); got.IsZero() || got.Format(time.RFC3339) != want {
			t.Fatalf("ParseTime(%q) = %v, want %s", input, got, want)
		}
	}
}

func TestBuildRunAuditOwnsTerminalStatusAndConservationCounts(t *testing.T) {
	runs := make(map[string]collector.ConnectorRun)
	stats := collector.Stats{
		RawResults: 4, MergedResults: 3, ResultsTerminal: 3,
		Accepted: 1, KnownURL: 1, ExactDuplicate: 1,
		ConnectorErrors: make(map[string]string),
		ContentLevels: map[collector.ContentLevel]int{
			collector.LevelFullText: 1,
			collector.LevelSnippet:  2,
		},
	}
	for position, key := range collector.ConnectorKeys() {
		run := collector.ConnectorRun{Connector: key}
		if position == 0 {
			run.Results = []collector.Candidate{{Title: "result"}}
		}
		if position == 1 {
			run.ErrorCode = "provider_unavailable"
			run.ErrorSummary = "Connector request failed"
			stats.ConnectorErrors[key] = run.ErrorCode + ": " + run.ErrorSummary
		}
		runs[key] = run
	}

	audit := BuildRunAudit(collector.Result{Stats: stats}, runs)
	if audit.ExecutionStatus != agentrun.StatusPartiallySucceeded {
		t.Fatalf("execution status = %q", audit.ExecutionStatus)
	}
	if audit.ConnectorsAttempted != 7 || audit.ConnectorsCompleted != 6 || audit.ConnectorsFailed != 1 {
		t.Fatalf("connector audit = %#v", audit)
	}
	if audit.CandidateCounts["merged_results"] != 3 ||
		audit.CandidateCounts["results_terminal"] != 3 ||
		audit.CandidateCounts["results_pending"] != 0 {
		t.Fatalf("candidate counts = %#v", audit.CandidateCounts)
	}
	if audit.ContentLevels[string(collector.LevelFullText)] != 1 ||
		audit.ContentLevels[string(collector.LevelSnippet)] != 2 {
		t.Fatalf("content levels = %#v", audit.ContentLevels)
	}
}

func TestBuildRunAuditMapsAllConnectorFailures(t *testing.T) {
	runs := make(map[string]collector.ConnectorRun)
	errorsByConnector := make(map[string]string)
	for _, key := range collector.ConnectorKeys() {
		runs[key] = collector.ConnectorRun{
			Connector: key, ErrorCode: "provider_unavailable", ErrorSummary: "Connector request failed",
		}
		errorsByConnector[key] = "provider_unavailable: Connector request failed"
	}

	audit := BuildRunAudit(collector.Result{
		Stats: collector.Stats{
			ConnectorErrors: errorsByConnector,
			ContentLevels:   make(map[collector.ContentLevel]int),
		},
	}, runs)
	if audit.ExecutionStatus != agentrun.StatusFailed ||
		audit.ErrorCode != "all_connectors_failed" ||
		audit.ConnectorsFailed != len(collector.ConnectorKeys()) {
		t.Fatalf("audit = %#v", audit)
	}
}

func TestOrderedRunNamesAreStableForKnownAndUnknownConnectors(t *testing.T) {
	runs := map[string]collector.ConnectorRun{
		"z_custom":        {},
		"tavily":          {},
		"a_custom":        {},
		"parallel_search": {},
	}
	got := orderedRunNames(runs)
	want := []string{"parallel_search", "tavily", "a_custom", "z_custom"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ordered run names = %#v, want %#v", got, want)
	}
}

func TestProcessorOwnsSequentialCandidateDecisionsAndConservation(t *testing.T) {
	now := time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC)
	runs := map[string]collector.ConnectorRun{
		"tavily": {
			Results: []collector.Candidate{
				{
					Connector: "tavily", Title: "政策", URL: "https://example.com/policy",
					Content: "政策正文", ContentLevel: collector.LevelFullText,
				},
				{
					Connector: "tavily", Title: "政策", URL: "https://example.com/policy-copy",
					Content: "政策正文", ContentLevel: collector.LevelFullText,
				},
			},
		},
	}
	processor := NewProcessor(runs, nil, DefaultNearDuplicateRadius, collector.Request{
		CollectedAt: now, TimeWindowHours: 1,
	})
	first, err := processor.Decide(processor.Merged[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := processor.Decide(processor.Merged[1])
	if err != nil {
		t.Fatal(err)
	}
	if first.Disposition != collector.DispositionAccepted ||
		second.Disposition != collector.DispositionExactDuplicate {
		t.Fatalf("decisions = %q, %q", first.Disposition, second.Disposition)
	}
	if _, err := processor.Finish(); err != nil {
		t.Fatal(err)
	}
	if processor.Stats.ResultsTerminal != 2 || processor.Stats.ResultsPending != 0 {
		t.Fatalf("stats = %#v", processor.Stats)
	}
}

func TestWindowForOwnsCollectorTimeBoundary(t *testing.T) {
	end := time.Date(2026, 7, 25, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	window := WindowFor(collector.Request{CollectedAt: end, TimeWindowHours: 48})
	if !window.End.Equal(end.UTC()) ||
		!window.Start.Equal(end.UTC().Add(-48*time.Hour)) ||
		window.Hours != 48 {
		t.Fatalf("window = %#v", window)
	}
}
