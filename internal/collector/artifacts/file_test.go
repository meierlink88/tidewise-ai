package artifacts

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
)

func TestMaterializeMergesOnceAndProducesMarkdownTSVAndSummary(t *testing.T) {
	root := t.TempDir()
	collectedAt := time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC)
	runs := map[string]collector.ConnectorRun{
		"parallel_search": {Connector: "parallel_search", Results: []collector.Candidate{{
			Connector: "parallel_search", Title: "政策", URL: "https://example.com/a?utm_source=x",
			Content: "片段", ContentLevel: collector.LevelSnippet, PublishedAtHint: "2026-07-17",
		}}},
		"tavily": {Connector: "tavily", Results: []collector.Candidate{
			{Connector: "tavily", Title: "政策", URL: "https://example.com/a", Content: "完整正文", ContentLevel: collector.LevelFullText, PublishedAtHint: "2026-07-17"},
			{Connector: "tavily", Title: "旧闻", URL: "https://example.com/old", Content: "旧内容", ContentLevel: collector.LevelSnippet, PublishedAtHint: "2026-07-01"},
		}},
	}
	request := collector.Request{RunID: "run-1", Prompt: "采集政策资讯", CollectedAt: collectedAt, TimeWindowHours: 48}
	result, err := (File{Root: root, NearDuplicateRadius: 3}).Materialize(context.Background(), request, runs)
	if err != nil {
		t.Fatal(err)
	}
	if result.Stats.RawResults != 3 || result.Stats.MergedResults != 2 || result.Stats.Accepted != 1 || result.Stats.OutOfWindow != 1 || result.Stats.ResultsPending != 0 {
		t.Fatalf("unexpected stats: %+v", result.Stats)
	}
	matches, err := filepath.Glob(filepath.Join(root, "documents", "*", "*", "*", "*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("documents: %v, err=%v", matches, err)
	}
	document, _ := os.ReadFile(matches[0])
	text := string(document)
	if !strings.Contains(text, "完整正文") || !strings.Contains(text, `primary_connector: "tavily"`) || !strings.Contains(text, `- "parallel_search"`) {
		t.Fatalf("unexpected document:\n%s", text)
	}
	for _, path := range []string{result.Index, result.Summary, result.Candidates, result.Manifest} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing output %s: %v", path, err)
		}
	}
	if result.Summary != filepath.Join(root, "runs", "run-1", "summary.md") || result.Candidates != filepath.Join(root, "runs", "run-1", "candidates.jsonl") || result.Manifest != filepath.Join(root, "runs", "run-1", "manifest.json") {
		t.Fatalf("unexpected run artifact paths: %#v", result)
	}
	manifestBytes, err := os.ReadFile(result.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Schema         string            `json:"schema"`
		ExecutionID    string            `json:"execution_id"`
		PromptSHA256   string            `json:"prompt_sha256"`
		PromptBytes    int               `json:"prompt_bytes"`
		ResultsPending int               `json:"results_pending"`
		Artifacts      map[string]string `json:"artifacts"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.Schema != "collector_artifact_manifest.v1" || manifest.ExecutionID != "run-1" || manifest.PromptSHA256 == "" || manifest.PromptBytes != len([]byte(request.Prompt)) || manifest.ResultsPending != 0 {
		t.Fatalf("manifest = %#v", manifest)
	}
	ledger, err := os.ReadFile(result.Candidates)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(ledger)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0]+lines[1], `"disposition":"accepted"`) || !strings.Contains(lines[0]+lines[1], `"disposition":"out_of_window"`) {
		t.Fatalf("candidate ledger = %s", ledger)
	}
	if !strings.Contains(lines[0]+lines[1], `"reason":"accepted"`) || !strings.Contains(lines[0]+lines[1], `"reason":"published_at_outside_time_window"`) {
		t.Fatalf("candidate ledger reasons = %s", ledger)
	}
	if strings.Contains(string(ledger), "旧内容") {
		t.Fatalf("rejected Candidate body leaked into ledger: %s", ledger)
	}
}

func TestManifestAndSummaryContainCompleteBatchAudit(t *testing.T) {
	root := t.TempDir()
	collectedAt := time.Date(2026, 7, 22, 8, 30, 0, 0, time.UTC)
	result, err := (File{Root: root, NearDuplicateRadius: 3}).Materialize(context.Background(), collector.Request{
		RunID: "audit", Prompt: "collect", CollectedAt: collectedAt, TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{
		"parallel_search": {
			Connector: "parallel_search",
			Results: []collector.Candidate{{
				Connector: "parallel_search", Title: "Policy", URL: "https://example.com/policy",
				Content: "direct result", ContentLevel: collector.LevelFullText,
			}},
		},
		"tavily": {
			Connector: "tavily", ErrorCode: "connector_failed", ErrorSummary: "Connector request failed",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(result.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		WindowStart         string `json:"window_start"`
		WindowEnd           string `json:"window_end"`
		ConnectorsAttempted int    `json:"connectors_attempted"`
		ConnectorsCompleted int    `json:"connectors_completed"`
		ConnectorsFailed    int    `json:"connectors_failed"`
		ConnectorOutcomes   []struct {
			Connector   string `json:"connector"`
			Status      string `json:"status"`
			ResultCount int    `json:"result_count"`
			ErrorCode   string `json:"error_code"`
		} `json:"connector_outcomes"`
		CandidateCounts map[string]int      `json:"candidate_counts"`
		ContentLevels   map[string]int      `json:"content_levels"`
		Artifacts       map[string]string   `json:"artifacts"`
		Accepted        []map[string]string `json:"accepted"`
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.WindowStart != "2026-07-20T08:30:00Z" || manifest.WindowEnd != "2026-07-22T08:30:00Z" {
		t.Fatalf("window = %s .. %s", manifest.WindowStart, manifest.WindowEnd)
	}
	if manifest.ConnectorsAttempted != 2 || manifest.ConnectorsCompleted != 1 || manifest.ConnectorsFailed != 1 {
		t.Fatalf("connector totals = attempted:%d completed:%d failed:%d", manifest.ConnectorsAttempted, manifest.ConnectorsCompleted, manifest.ConnectorsFailed)
	}
	if len(manifest.ConnectorOutcomes) != 2 ||
		manifest.ConnectorOutcomes[0].Connector != "parallel_search" ||
		manifest.ConnectorOutcomes[0].Status != "completed" ||
		manifest.ConnectorOutcomes[0].ResultCount != 1 ||
		manifest.ConnectorOutcomes[1].Connector != "tavily" ||
		manifest.ConnectorOutcomes[1].Status != "failed" ||
		manifest.ConnectorOutcomes[1].ErrorCode != "connector_failed" {
		t.Fatalf("connector outcomes = %#v", manifest.ConnectorOutcomes)
	}
	if manifest.CandidateCounts["merged_results"] != manifest.CandidateCounts["results_terminal"]+manifest.CandidateCounts["results_pending"] {
		t.Fatalf("Candidate conservation = %#v", manifest.CandidateCounts)
	}
	if manifest.ContentLevels["full_text"] != 1 ||
		manifest.ContentLevels["summary"] != 0 ||
		manifest.ContentLevels["snippet"] != 0 ||
		manifest.ContentLevels["title_only"] != 0 {
		t.Fatalf("content levels = %#v", manifest.ContentLevels)
	}
	for _, key := range []string{"documents", "index", "candidates", "summary", "manifest"} {
		if manifest.Artifacts[key] == "" {
			t.Fatalf("missing Artifact path %q: %#v", key, manifest.Artifacts)
		}
	}
	if len(manifest.Accepted) != 1 || manifest.Accepted[0]["path"] == "" || len(manifest.Accepted[0]["sha256"]) != 64 {
		t.Fatalf("accepted audit = %#v", manifest.Accepted)
	}
	summary, err := os.ReadFile(result.Summary)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"window_start: 2026-07-20T08:30:00Z",
		"window_end: 2026-07-22T08:30:00Z",
		"connectors_attempted: 2",
		"connectors_completed: 1",
		"connectors_failed: 1",
		"full_text: 1",
		"manifest: " + result.Manifest,
		manifest.Accepted[0]["path"] + " sha256:" + manifest.Accepted[0]["sha256"],
		"parallel_search: completed (1)",
		"tavily: failed (0), connector_failed: Connector request failed",
	} {
		if !strings.Contains(string(summary), fragment) {
			t.Fatalf("summary missing %q:\n%s", fragment, summary)
		}
	}
}

func TestMaterializeHonorsCanceledExecutionBeforePublishing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (File{Root: root, NearDuplicateRadius: 3}).Materialize(ctx, collector.Request{
		RunID: "canceled", Prompt: "prompt", CollectedAt: time.Now().UTC(), TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{"tavily": {Results: []collector.Candidate{{
		Connector: "tavily", Title: "title", URL: "https://example.com/item", Content: "content", ContentLevel: collector.LevelSnippet,
	}}}})
	if err != context.Canceled {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "runs", "canceled")); !os.IsNotExist(statErr) {
		t.Fatalf("canceled materialization published a run: %v", statErr)
	}
}

func TestSimHash64MatchesCodexBLAKE2bContract(t *testing.T) {
	text := "The ministry announced new semiconductor export controls on Tuesday. The rules cover advanced chips and manufacturing equipment."
	if got := simhash(text); got != "5ebccb6498055f7e" {
		t.Fatalf("simhash = %s, want 5ebccb6498055f7e", got)
	}
}

func TestNormalizeBodyMatchesCrossLanguageContract(t *testing.T) {
	input := "e\u0301\r\nline with spaces  \r\n\r\n\r\nnext\n"
	if got := normalizeBody(input); got != "é\nline with spaces\n\nnext" {
		t.Fatalf("normalized body = %q", got)
	}
}

func TestDeterministicContractMatchesCodexPythonGolden(t *testing.T) {
	var golden struct {
		URL            string `json:"url"`
		CanonicalURL   string `json:"canonical_url"`
		BodyInput      string `json:"body_input"`
		NormalizedBody string `json:"normalized_body"`
		ContentSHA256  string `json:"content_sha256"`
		DocumentID     string `json:"document_id"`
		SimHash64      string `json:"simhash64"`
	}
	payload, err := os.ReadFile("testdata/deterministic_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &golden); err != nil {
		t.Fatal(err)
	}
	if got := canonicalURL(golden.URL); got != golden.CanonicalURL {
		t.Fatalf("canonical URL = %q, want %q", got, golden.CanonicalURL)
	}
	body := normalizeBody(golden.BodyInput)
	if body != golden.NormalizedBody {
		t.Fatalf("normalized body = %q, want %q", body, golden.NormalizedBody)
	}
	contentHash := hash(body)
	if contentHash != golden.ContentSHA256 {
		t.Fatalf("content SHA-256 = %q, want %q", contentHash, golden.ContentSHA256)
	}
	if got := "sha256:" + hash(golden.CanonicalURL+"\n"+contentHash); got != golden.DocumentID {
		t.Fatalf("document ID = %q, want %q", got, golden.DocumentID)
	}
	if got := simhash(body); got != golden.SimHash64 {
		t.Fatalf("SimHash64 = %q, want %q", got, golden.SimHash64)
	}
}

func TestCanonicalURLSuppliesRootPath(t *testing.T) {
	if got := canonicalURL("https://EXAMPLE.com"); got != "https://example.com/" {
		t.Fatalf("canonical URL = %q, want root path", got)
	}
}

func TestDeterministicMergeUsesUnicodeRichnessAndStableConnectorOrder(t *testing.T) {
	rows := []collector.Candidate{
		{Connector: "tavily", URL: "https://example.com/item", Title: "标题", Content: "a b      ", ContentLevel: collector.LevelSnippet},
		{Connector: "parallel_search", URL: "https://example.com/item", Title: "标题", Content: "abc", ContentLevel: collector.LevelSnippet},
	}
	for _, input := range [][]collector.Candidate{rows, {rows[1], rows[0]}} {
		merged := merge(input)
		if len(merged) != 1 || merged[0].PrimaryConnector != "parallel_search" || merged[0].Content != "abc" {
			t.Fatalf("merged = %#v", merged)
		}
	}
}

func TestTimeParserAcceptsCommonPythonISOForms(t *testing.T) {
	for input, want := range map[string]string{
		"2026-07-22 12:03:44":       "2026-07-22T12:03:44Z",
		"2026-07-22T12:03:44Z":      "2026-07-22T12:03:44Z",
		"2026-07-22T20:03:44+08:00": "2026-07-22T12:03:44Z",
		"2026-07-22":                "2026-07-22T00:00:00Z",
	} {
		if got := parseTime(input); got.IsZero() || got.Format(time.RFC3339) != want {
			t.Fatalf("parseTime(%q) = %v, want %s", input, got, want)
		}
	}
}

func TestMaterializeClassifiesKnownExactAndNearDuplicatesAcrossRuns(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	materializer := File{Root: root, NearDuplicateRadius: 3}
	baseTitle := "Policy"
	baseContent := "The ministry announced new semiconductor export controls on Tuesday. The rules cover advanced chips and manufacturing equipment."

	materializeOneRun := func(runID, rawURL, title, content string) collector.Stats {
		t.Helper()
		result, err := materializer.Materialize(context.Background(), collector.Request{
			RunID: runID, Prompt: "collect policy", CollectedAt: now, TimeWindowHours: 48,
		}, map[string]collector.ConnectorRun{"tavily": {
			Connector: "tavily",
			Results:   []collector.Candidate{{Connector: "tavily", Title: title, URL: rawURL, Content: content, ContentLevel: collector.LevelSnippet}},
		}})
		if err != nil {
			t.Fatal(err)
		}
		return result.Stats
	}

	if stats := materializeOneRun("accepted", "https://example.com/item?utm_source=test", baseTitle, baseContent); stats.Accepted != 1 {
		t.Fatalf("accepted stats = %#v", stats)
	}
	if stats := materializeOneRun("known-url", "https://EXAMPLE.com/item", "Changed title", "Changed content"); stats.KnownURL != 1 {
		t.Fatalf("known URL stats = %#v", stats)
	}
	if stats := materializeOneRun("exact", "https://example.com/exact", baseTitle, baseContent); stats.ExactDuplicate != 1 {
		t.Fatalf("exact duplicate stats = %#v", stats)
	}
	nearContent := baseContent + " today."
	if stats := materializeOneRun("near", "https://example.com/near", baseTitle, nearContent); stats.NearDuplicate != 1 {
		t.Fatalf("near duplicate stats = %#v", stats)
	}
}

func TestMaterializeRebuildsMissingIndexFromAcceptedMarkdown(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	materializer := File{Root: root, NearDuplicateRadius: 3}
	run := func(runID, content string) (*collector.Result, error) {
		return materializer.Materialize(context.Background(), collector.Request{
			RunID: runID, Prompt: "collect", CollectedAt: now, TimeWindowHours: 48,
		}, map[string]collector.ConnectorRun{"tavily": {
			Connector: "tavily",
			Results: []collector.Candidate{{
				Connector: "tavily", Title: "Policy", URL: "https://example.com/policy",
				Content: content, ContentLevel: collector.LevelSnippet,
			}},
		}})
	}
	first, err := run("first", "original direct result")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(first.Index); err != nil {
		t.Fatal(err)
	}

	second, err := run("second", "changed direct result")
	if err != nil {
		t.Fatal(err)
	}
	if second.Stats.KnownURL != 1 || second.Stats.Accepted != 0 {
		t.Fatalf("stats = %#v", second.Stats)
	}
	index, err := os.ReadFile(second.Index)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(string(index)), "\n"); len(lines) != 2 {
		t.Fatalf("rebuilt index = %s", index)
	}
}

func TestVerifyIndexDetectsStaleCacheAndRebuildRepairsIt(t *testing.T) {
	root := t.TempDir()
	result, err := (File{Root: root, NearDuplicateRadius: 3}).Materialize(context.Background(), collector.Request{
		RunID: "verify", Prompt: "collect", CollectedAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC), TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{"tavily": {
		Results: []collector.Candidate{{
			Connector: "tavily", Title: "Policy", URL: "https://example.com/policy",
			Content: "direct result", ContentLevel: collector.LevelSnippet,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(result.Index)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(index)), "\n")
	columns := strings.Split(lines[1], "\t")
	columns[3] = strings.Repeat("0", 64)
	lines[1] = strings.Join(columns, "\t")
	if err := os.WriteFile(result.Index, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := VerifyIndex(root); err == nil || !strings.Contains(err.Error(), "content hash") {
		t.Fatalf("VerifyIndex error = %v", err)
	}
	rebuilt, err := RebuildIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if rebuilt.Documents != 1 || rebuilt.Records != 1 {
		t.Fatalf("rebuild report = %#v", rebuilt)
	}
	verified, err := VerifyIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Documents != 1 || verified.Records != 1 {
		t.Fatalf("verify report = %#v", verified)
	}
}

func TestVerifyIndexDetectsMissingAndExtraRows(t *testing.T) {
	root := t.TempDir()
	result, err := (File{Root: root, NearDuplicateRadius: 3}).Materialize(context.Background(), collector.Request{
		RunID: "coverage", Prompt: "collect",
		CollectedAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC), TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{"tavily": {
		Results: []collector.Candidate{{
			Connector: "tavily", Title: "Policy", URL: "https://example.com/policy",
			Content: "direct result", ContentLevel: collector.LevelSnippet,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	originalIndex, err := os.ReadFile(result.Index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.Index, []byte(indexHeader), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyIndex(root); err == nil || !strings.Contains(err.Error(), "missing document path") {
		t.Fatalf("missing-row VerifyIndex error = %v", err)
	}

	extraPath := filepath.Join(root, "documents", "extra.txt")
	if err := os.WriteFile(extraPath, []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	extraRecord := strings.Join([]string{
		"sha256:" + strings.Repeat("a", 64), "",
		strings.Repeat("b", 64), strings.Repeat("c", 64),
		strings.Repeat("d", 16), extraPath,
	}, "\t") + "\n"
	if err := os.WriteFile(result.Index, append(originalIndex, []byte(extraRecord)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyIndex(root); err == nil || !strings.Contains(err.Error(), "extra document path") {
		t.Fatalf("extra-row VerifyIndex error = %v", err)
	}
}

func TestRebuildIndexFailureLeavesPreviousCacheUntouched(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "indexes", "dedup-index.tsv")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatal(err)
	}
	previous := []byte(indexHeader)
	if err := os.WriteFile(indexPath, previous, 0o644); err != nil {
		t.Fatal(err)
	}
	invalidDocument := filepath.Join(root, "documents", "invalid.md")
	if err := os.MkdirAll(filepath.Dir(invalidDocument), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidDocument, []byte("not accepted Markdown"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildIndex(root); err == nil {
		t.Fatal("invalid accepted Markdown did not fail rebuild")
	}
	after, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(previous) {
		t.Fatalf("failed rebuild changed index:\n%s", after)
	}
}

func TestRebuildIndexRejectsDuplicateAcceptedDocumentIdentity(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	result, err := (File{Root: root, NearDuplicateRadius: 3}).Materialize(context.Background(), collector.Request{
		RunID: "duplicate-source", Prompt: "collect", CollectedAt: now, TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{"tavily": {
		Results: []collector.Candidate{{
			Connector: "tavily", Title: "Policy", URL: "https://example.com/policy",
			Content: "direct result", ContentLevel: collector.LevelSnippet,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	previous, err := os.ReadFile(result.Index)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := filepath.Join(root, "documents", "duplicate.md")
	payload, err := os.ReadFile(result.AcceptedDocuments[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(duplicate, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildIndex(root); err == nil || !strings.Contains(err.Error(), "duplicate document ID") {
		t.Fatalf("RebuildIndex error = %v", err)
	}
	after, err := os.ReadFile(result.Index)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(previous) {
		t.Fatal("duplicate-identity rebuild changed the previous cache")
	}
}

func TestLoadIndexRejectsMalformedSchemaAndDuplicateIdentity(t *testing.T) {
	t.Parallel()

	validRecord := strings.Join([]string{
		"sha256:" + strings.Repeat("1", 64),
		"2026-07-22T08:00:00Z",
		strings.Repeat("2", 64),
		strings.Repeat("3", 64),
		strings.Repeat("4", 16),
		"/tmp/document.md",
	}, "\t")
	tests := map[string]string{
		"wrong header":     strings.Replace(indexHeader, "document_id", "id", 1) + validRecord + "\n",
		"bad document ID":  indexHeader + strings.Replace(validRecord, "sha256:"+strings.Repeat("1", 64), "document-1", 1) + "\n",
		"bad URL hash":     indexHeader + strings.Replace(validRecord, strings.Repeat("2", 64), "not-a-hash", 1) + "\n",
		"bad content hash": indexHeader + strings.Replace(validRecord, strings.Repeat("3", 64), "not-a-hash", 1) + "\n",
		"bad SimHash":      indexHeader + strings.Replace(validRecord, strings.Repeat("4", 16), "not-a-hash", 1) + "\n",
		"missing path":     indexHeader + strings.TrimSuffix(validRecord, "/tmp/document.md") + "\n",
		"duplicate document ID": indexHeader + validRecord + "\n" +
			strings.Replace(validRecord, "/tmp/document.md", "/tmp/other.md", 1) + "\n",
		"duplicate path": indexHeader + validRecord + "\n" +
			strings.Replace(validRecord, "sha256:"+strings.Repeat("1", 64), "sha256:"+strings.Repeat("5", 64), 1) + "\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "dedup-index.tsv")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := loadIndex(path); err == nil {
				t.Fatalf("loadIndex accepted:\n%s", content)
			}
		})
	}
}

func TestMaterializeUsesHealthyIndexWithoutReadingAcceptedMarkdown(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	materializer := File{Root: root, NearDuplicateRadius: 3}
	first, err := materializer.Materialize(context.Background(), collector.Request{
		RunID: "first", Prompt: "collect", CollectedAt: now, TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{"tavily": {
		Results: []collector.Candidate{{
			Connector: "tavily", Title: "Policy", URL: "https://example.com/policy",
			Content: "direct result", ContentLevel: collector.LevelSnippet,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.AcceptedDocuments) != 1 {
		t.Fatalf("accepted documents = %#v", first.AcceptedDocuments)
	}
	if err := os.WriteFile(first.AcceptedDocuments[0], []byte("deliberately unreadable as accepted Markdown"), 0o644); err != nil {
		t.Fatal(err)
	}

	second, err := materializer.Materialize(context.Background(), collector.Request{
		RunID: "second", Prompt: "collect", CollectedAt: now, TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{"tavily": {
		Results: []collector.Candidate{{
			Connector: "tavily", Title: "Changed", URL: "https://example.com/policy",
			Content: "changed direct result", ContentLevel: collector.LevelSnippet,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if second.Stats.KnownURL != 1 || second.Stats.Accepted != 0 {
		t.Fatalf("stats = %#v", second.Stats)
	}
	if _, err := VerifyIndex(root); err == nil || !strings.Contains(err.Error(), "invalid accepted Markdown") {
		t.Fatalf("VerifyIndex error = %v", err)
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

func TestPublishArtifactsRollsBackDocumentsAndIndexWhenManifestCannotPublish(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "indexes", "dedup-index.tsv")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatal(err)
	}
	previousIndex := []byte(indexHeader + "old\t\turl\tcontent\t0000000000000000\told.md\n")
	if err := os.WriteFile(indexPath, previousIndex, 0o644); err != nil {
		t.Fatal(err)
	}
	stageRoot := t.TempDir()
	staged := collector.Result{
		Candidates: filepath.Join(stageRoot, "candidates.jsonl"),
		Summary:    filepath.Join(stageRoot, "summary.md"),
		Manifest:   filepath.Join(stageRoot, "missing-manifest.json"),
		Index:      filepath.Join(stageRoot, "dedup-index.tsv"),
	}
	for path, content := range map[string]string{
		staged.Candidates: "{}\n", staged.Summary: "summary", staged.Index: indexHeader + "new\t\turl2\tcontent2\t0000000000000001\tnew.md\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runRoot := filepath.Join(root, "runs", "run-rollback")
	final := collector.Result{
		Candidates: filepath.Join(runRoot, "candidates.jsonl"),
		Summary:    filepath.Join(runRoot, "summary.md"), Manifest: filepath.Join(runRoot, "manifest.json"),
	}
	stagedDocument := filepath.Join(stageRoot, "document.md")
	finalDocument := filepath.Join(root, "documents", "2026", "07", "22", "document.md")
	if err := os.WriteFile(stagedDocument, []byte("document"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publishArtifacts(context.Background(), staged, final, map[string]string{finalDocument: stagedDocument}, indexPath, runRoot); err == nil {
		t.Fatal("expected missing staged manifest to fail publication")
	}
	if _, err := os.Stat(finalDocument); !os.IsNotExist(err) {
		t.Fatalf("document survived rollback: %v", err)
	}
	if _, err := os.Stat(runRoot); !os.IsNotExist(err) {
		t.Fatalf("run directory survived rollback: %v", err)
	}
	gotIndex, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotIndex) != string(previousIndex) {
		t.Fatalf("index was not restored:\n%s", gotIndex)
	}
}

func TestManifestAndSummaryContainOnlySafeConnectorFailure(t *testing.T) {
	root := t.TempDir()
	request := collector.Request{RunID: "safe-failure", Prompt: "secret prompt", CollectedAt: time.Now().UTC(), TimeWindowHours: 48}
	result, err := (File{Root: root, NearDuplicateRadius: 3}).Materialize(context.Background(), request, map[string]collector.ConnectorRun{
		"tavily": {Connector: "tavily", ErrorCode: "connector_failed", ErrorSummary: "Connector request failed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(result.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := os.ReadFile(result.Summary)
	if err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{string(manifest), string(summary)} {
		if !strings.Contains(content, "connector_failed") || !strings.Contains(content, "Connector request failed") {
			t.Fatalf("safe failure missing: %s", content)
		}
		if strings.Contains(content, request.Prompt) {
			t.Fatalf("prompt leaked: %s", content)
		}
	}
}
