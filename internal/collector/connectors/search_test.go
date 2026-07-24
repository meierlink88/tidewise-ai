package connectors

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestParallelSearchUsesDirectAPIContract(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("x-api-key") != "parallel-key" {
			t.Fatalf("missing API key header")
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		queries, ok := body["search_queries"].([]any)
		if !ok || len(queries) != 2 {
			t.Fatalf("unexpected search_queries: %#v", body["search_queries"])
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"results":[{"url":"https://example.com/a","title":"政策更新","publish_date":"2026-07-17","excerpts":["片段一","片段二"]}]}`)),
		}, nil
	})}

	connector := ParallelSearch{APIKey: "parallel-key", Endpoint: "https://parallel.test/v1/search", Client: client}
	results, err := connector.Collect(context.Background(), collector.Request{
		Prompt: "目标", SearchQueries: []string{"全球政策", "资本市场"},
		CandidateLimit: 10, CollectedAt: time.Now().UTC(), TimeWindowHours: 48,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Content != "片段一\n\n片段二" || results[0].ContentLevel != collector.LevelSnippet {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestAppendCandidatePreservesInvalidDirectResultForTerminalAccounting(t *testing.T) {
	t.Parallel()

	results := appendCandidate(nil, collector.Candidate{Title: "missing URL", Content: "direct content", ContentLevel: collector.LevelSnippet})
	if len(results) != 1 || results[0].Title != "missing URL" || results[0].ContentOrigin != collector.ContentOrigin {
		t.Fatalf("invalid direct result was dropped: %#v", results)
	}
}

func TestTavilyUsesCodexDirectResultContract(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Header.Get("Authorization") != "Bearer tavily-key" {
			t.Fatalf("unexpected authorization header")
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{
			"query":               "A股半导体产业链最新变化",
			"topic":               "news",
			"search_depth":        "advanced",
			"auto_parameters":     false,
			"chunks_per_source":   float64(3),
			"max_results":         float64(10),
			"include_answer":      false,
			"include_raw_content": "markdown",
			"start_date":          "2026-07-16",
			"end_date":            "2026-07-18",
		}
		for key, value := range want {
			if body[key] != value {
				t.Fatalf("body[%q] = %#v, want %#v; body=%#v", key, body[key], value, body)
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"results":[
				{"url":"https://example.com/news","title":"半导体政策","content":" 与查询相关的片段 ","raw_content":"完整网页 Markdown","published_date":"2026-07-17"},
				{"url":"https://example.com/snippet","title":"只有片段","content":"备用片段","raw_content":" "},
				{"url":"https://example.com/title","title":"只有标题","content":"  ","raw_content":""}
			]}`)),
		}, nil
	})}

	connector := Tavily{APIKey: "tavily-key", Endpoint: "https://tavily.test/search", Client: client}
	results, err := connector.Collect(context.Background(), collector.Request{
		SearchQueries: []string{"A股", "半导体"}, CombinedQuery: "A股半导体产业链最新变化", CandidateLimit: 10,
		CollectedAt: time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC), TimeWindowHours: 48,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("Tavily calls = %d, want 1", calls)
	}
	if len(results) != 3 {
		t.Fatalf("results = %+v", results)
	}
	if results[0].Content != "完整网页 Markdown" || results[0].ContentLevel != collector.LevelFullText {
		t.Fatalf("raw content result = %+v", results[0])
	}
	if results[1].Content != "备用片段" || results[1].ContentLevel != collector.LevelSnippet {
		t.Fatalf("snippet fallback result = %+v", results[1])
	}
	if results[2].Content != "只有标题" || results[2].ContentLevel != collector.LevelTitle {
		t.Fatalf("title fallback result = %+v", results[2])
	}
}

func TestTavilyUsesRelativeDayForSubDayWindow(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["time_range"] != "day" {
			t.Fatalf("time_range = %#v, want day; body=%#v", body["time_range"], body)
		}
		if startDate, exists := body["start_date"]; exists {
			t.Fatalf("sub-day request contains start_date = %#v; body=%#v", startDate, body)
		}
		if endDate, exists := body["end_date"]; exists {
			t.Fatalf("sub-day request contains end_date = %#v; body=%#v", endDate, body)
		}
		return jsonResponse(map[string]any{"results": []any{}}), nil
	})}

	_, err := (Tavily{APIKey: "key", Endpoint: "https://tavily.test/search", Client: client}).Collect(
		context.Background(),
		collector.Request{
			CombinedQuery:   "A股产业链最新资讯",
			CandidateLimit:  10,
			CollectedAt:     time.Date(2026, 7, 24, 1, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60)),
			TimeWindowHours: 2,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTavilyCapsDirectResultsAtTen(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(map[string]any{"results": candidateResponseItems(12)}), nil
	})}
	results, err := (Tavily{APIKey: "key", Endpoint: "https://tavily.test/search", Client: client}).Collect(context.Background(), collector.Request{
		CombinedQuery: "query", CandidateLimit: 20, CollectedAt: time.Now().UTC(), TimeWindowHours: 48,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 10 {
		t.Fatalf("results = %d, want 10", len(results))
	}
}

func TestBochaUsesCombinedQueryOnceAndDirectSummary(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["query"] != "中国半导体政策 芯片供应链价格" || body["summary"] != true || body["freshness"] != "oneWeek" || body["count"] != float64(10) {
			t.Fatalf("Bocha body = %#v", body)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"data":{"webPages":{"value":[{"name":"产业新闻","url":"https://example.com/bocha","summary":"直接摘要","snippet":"备用片段","siteName":"example.com"}]}}}`)),
		}, nil
	})}
	results, err := (Bocha{APIKey: "bocha-key", Endpoint: "https://bocha.test/search", Client: client}).Collect(context.Background(), collector.Request{
		SearchQueries: []string{"不应", "机械拼接"}, CombinedQuery: "中国半导体政策 芯片供应链价格",
		CandidateLimit: 10, TimeWindowHours: 168,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(results) != 1 || results[0].Content != "直接摘要" || results[0].ContentLevel != collector.LevelSummary {
		t.Fatalf("calls=%d results=%#v", calls, results)
	}
}

func TestBochaCapsResultsWhenProviderReturnsTooMany(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(map[string]any{"data": map[string]any{"webPages": map[string]any{"value": candidateResponseItems(12)}}}), nil
	})}
	results, err := (Bocha{APIKey: "key", Endpoint: "https://bocha.test/search", Client: client}).Collect(context.Background(), collector.Request{
		CombinedQuery: "query", CandidateLimit: 20, TimeWindowHours: 48,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 10 {
		t.Fatalf("results = %d, want 10", len(results))
	}
}

func candidateResponseItems(count int) []map[string]any {
	items := make([]map[string]any, count)
	for index := range items {
		items[index] = map[string]any{
			"title": "title", "name": "title", "url": "https://example.com/item/" + string(rune('a'+index)),
			"content": "content", "summary": "summary",
		}
	}
	return items
}

func jsonResponse(payload any) *http.Response {
	body, _ := json.Marshal(payload)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}
}
