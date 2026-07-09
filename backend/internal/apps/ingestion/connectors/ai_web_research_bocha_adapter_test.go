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

func TestBochaSearchAdapterMapsRequestAndResponse(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/web-search" {
			t.Fatalf("path = %q, want /v1/web-search", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer bocha-key" {
			t.Fatalf("Authorization = %q, want bearer key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"_type": "SearchResponse",
			"webPages": {
				"value": [
					{
						"name": "央行发布公开市场操作公告",
						"url": "https://www.pbc.gov.cn/example.html",
						"siteName": "中国人民银行",
						"snippet": "央行公告摘要",
						"summary": "央行公告结构化摘要",
						"datePublished": "2026-07-09T09:00:00Z"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	response, err := BochaSearchAdapter{
		Client:  server.Client(),
		BaseURL: server.URL,
	}.Search(context.Background(), SearchRequest{
		Query:      "中国财经新闻",
		MaxResults: 8,
		Credential: "bocha-key",
		Options: map[string]any{
			"freshness": "oneDay",
			"summary":   true,
			"count":     8,
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if captured["query"] != "中国财经新闻" {
		t.Fatalf("query = %v, want 中国财经新闻", captured["query"])
	}
	if captured["freshness"] != "oneDay" || captured["summary"] != true || captured["count"] != float64(8) {
		t.Fatalf("request = %+v, want bocha options", captured)
	}
	if got, want := len(response.Results), 1; got != want {
		t.Fatalf("results = %d, want %d", got, want)
	}
	result := response.Results[0]
	if result.Provider != "bocha_web_search" {
		t.Fatalf("Provider = %q, want bocha_web_search", result.Provider)
	}
	if result.Title != "央行发布公开市场操作公告" || result.SourceName != "中国人民银行" {
		t.Fatalf("result = %+v, want title/source", result)
	}
	if result.Snippet != "央行公告摘要" || result.RawContent != "央行公告结构化摘要" {
		t.Fatalf("snippet/raw = %q/%q, want snippet/summary", result.Snippet, result.RawContent)
	}
	if !result.PublishedAt.Equal(time.Date(2026, 7, 9, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("PublishedAt = %v, want parsed date", result.PublishedAt)
	}
}

func TestBochaSearchAdapterReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "quota exceeded", http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, err := BochaSearchAdapter{Client: server.Client(), BaseURL: server.URL}.Search(context.Background(), SearchRequest{
		Query:      "财经",
		MaxResults: 1,
		Credential: "bad-key",
	})
	if err == nil || !strings.Contains(err.Error(), "bocha status 429") {
		t.Fatalf("Search() error = %v, want status error", err)
	}
}

func TestBochaSearchAdapterUsesRequestBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/web-search" {
			t.Fatalf("path = %q, want /v1/web-search", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"webPages":{"value":[]}}`))
	}))
	defer server.Close()

	_, err := BochaSearchAdapter{Client: server.Client(), BaseURL: "https://wrong.example.com"}.Search(context.Background(), SearchRequest{
		BaseURL:    server.URL,
		Query:      "财经",
		MaxResults: 1,
		Credential: "bocha-key",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
}
