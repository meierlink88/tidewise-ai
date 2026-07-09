package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAICompatibleNormalizerMapsRequestAndResponse(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer llm-key" {
			t.Fatalf("Authorization = %q, want bearer key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-test",
			"model": "deepseek-v4-pro",
			"choices": [
				{
					"message": {
						"content": "{\"items\":[{\"title\":\"全球央行新闻\",\"content_text\":\"央行政策摘要\",\"source_name\":\"Example\",\"source_url\":\"https://example.com/news\",\"content_origin\":\"search_snippet\"}],\"meta\":{\"actual_results\":1}}"
					}
				}
			],
			"usage": {"prompt_tokens": 100, "completion_tokens": 50}
		}`))
	}))
	defer server.Close()

	normalizer := OpenAICompatibleNormalizer{Client: server.Client(), BaseURL: server.URL}
	response, err := normalizer.Normalize(context.Background(), LLMNormalizeRequest{
		Model:      "deepseek-v4-pro",
		Credential: "llm-key",
		Prompt:     "请整理搜索结果",
		SearchResults: []SearchResultCandidate{
			{Provider: "tavily", Title: "全球央行新闻", URL: "https://example.com/news", Snippet: "央行政策摘要"},
		},
	})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if captured["model"] != "deepseek-v4-pro" {
		t.Fatalf("model = %v, want deepseek-v4-pro", captured["model"])
	}
	messages, ok := captured["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %v, want system and user messages", captured["messages"])
	}
	if response.Content["items"] == nil {
		t.Fatalf("Content = %+v, want items", response.Content)
	}
	if response.Raw["model"] != "deepseek-v4-pro" {
		t.Fatalf("raw model = %v, want deepseek-v4-pro", response.Raw["model"])
	}
}

func TestOpenAICompatibleNormalizerRejectsNonJSONContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
	}))
	defer server.Close()

	_, err := (OpenAICompatibleNormalizer{Client: server.Client(), BaseURL: server.URL}).Normalize(context.Background(), LLMNormalizeRequest{
		Model:      "deepseek-v4-pro",
		Credential: "llm-key",
		Prompt:     "prompt",
	})
	if err == nil || !strings.Contains(err.Error(), "decode llm content") {
		t.Fatalf("Normalize() error = %v, want decode content error", err)
	}
}

func TestOpenAICompatibleNormalizerReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := (OpenAICompatibleNormalizer{Client: server.Client(), BaseURL: server.URL}).Normalize(context.Background(), LLMNormalizeRequest{
		Model:      "deepseek-v4-pro",
		Credential: "bad-key",
		Prompt:     "prompt",
	})
	if err == nil || !strings.Contains(err.Error(), "llm status 401") {
		t.Fatalf("Normalize() error = %v, want status error", err)
	}
}
