package materialize

import (
	"context"
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
	request := collector.Request{RunID: "run-1", CollectedAt: collectedAt, TimeWindowHours: 48}
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
	for _, path := range []string{result.Index, result.Summary} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing output %s: %v", path, err)
		}
	}
}
