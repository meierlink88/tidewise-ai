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
		Objective: "目标", SearchQueries: []string{"全球政策", "资本市场"},
		CandidateLimit: 10, CollectedAt: time.Now().UTC(), TimeWindowHours: 48,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Content != "片段一\n\n片段二" || results[0].ContentLevel != collector.LevelSnippet {
		t.Fatalf("unexpected results: %+v", results)
	}
}
