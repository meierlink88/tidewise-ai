package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTavilySearchAdapterMapsRequestAndResponse(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("path = %q, want /search", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"query": "global macro news",
			"request_id": "tvly-123",
			"results": [
				{
					"title": "Central banks signal policy divergence",
					"url": "https://example.com/central-banks",
					"content": "Search snippet",
					"raw_content": "Full article content",
					"score": 0.92,
					"published_date": "2026-07-09T08:30:00Z"
				}
			]
		}`))
	}))
	defer server.Close()

	response, err := TavilySearchAdapter{
		Client:  server.Client(),
		BaseURL: server.URL,
	}.Search(context.Background(), SearchRequest{
		Query:      "global macro news",
		MaxResults: 5,
		Credential: "test-key",
		Options: map[string]any{
			"topic":               "news",
			"search_depth":        "advanced",
			"time_range":          "day",
			"include_domains":     []any{"reuters.com"},
			"exclude_domains":     []any{"spam.example"},
			"include_raw_content": true,
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if captured["query"] != "global macro news" {
		t.Fatalf("query = %v, want global macro news", captured["query"])
	}
	if captured["topic"] != "news" || captured["search_depth"] != "advanced" || captured["time_range"] != "day" {
		t.Fatalf("request options = %+v, want tavily options", captured)
	}
	if captured["max_results"] != float64(5) {
		t.Fatalf("max_results = %v, want 5", captured["max_results"])
	}
	if got, want := len(response.Results), 1; got != want {
		t.Fatalf("results = %d, want %d", got, want)
	}
	result := response.Results[0]
	if result.Provider != "tavily" {
		t.Fatalf("Provider = %q, want tavily", result.Provider)
	}
	if result.Title != "Central banks signal policy divergence" || result.URL != "https://example.com/central-banks" {
		t.Fatalf("result = %+v, want title/url", result)
	}
	if result.Snippet != "Search snippet" || result.RawContent != "Full article content" {
		t.Fatalf("content = %q raw = %q, want snippet/raw", result.Snippet, result.RawContent)
	}
	if !result.PublishedAt.Equal(time.Date(2026, 7, 9, 8, 30, 0, 0, time.UTC)) {
		t.Fatalf("PublishedAt = %v, want parsed date", result.PublishedAt)
	}
	if response.Raw["request_id"] != "tvly-123" {
		t.Fatalf("request_id = %v, want tvly-123", response.Raw["request_id"])
	}
}

func TestTavilySearchAdapterReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := TavilySearchAdapter{Client: server.Client(), BaseURL: server.URL}.Search(context.Background(), SearchRequest{
		Query:      "news",
		MaxResults: 1,
		Credential: "bad-key",
	})
	if err == nil || !strings.Contains(err.Error(), "tavily status 401") {
		t.Fatalf("Search() error = %v, want status error", err)
	}
}

func TestTavilySearchAdapterUsesRequestBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("path = %q, want /search", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	_, err := TavilySearchAdapter{Client: server.Client(), BaseURL: "https://wrong.example.com"}.Search(context.Background(), SearchRequest{
		BaseURL:    server.URL,
		Query:      "news",
		MaxResults: 1,
		Credential: "test-key",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
}
