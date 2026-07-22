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
