package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderResponsesDoNotExposeCredentialsInRawMetadata(t *testing.T) {
	tavilyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"request_id":"safe","results":[]}`))
	}))
	defer tavilyServer.Close()

	tavily, err := TavilySearchAdapter{Client: tavilyServer.Client(), BaseURL: tavilyServer.URL}.Search(context.Background(), SearchRequest{
		Query:      "news",
		Credential: "secret-tavily-key",
		MaxResults: 1,
	})
	if err != nil {
		t.Fatalf("Tavily Search() error = %v", err)
	}
	assertNoSecretInMap(t, tavily.Raw, "secret-tavily-key")

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "safe",
			"choices": []any{map[string]any{"message": map[string]any{"content": `{"items":[]}`}}},
		})
	}))
	defer llmServer.Close()

	llm, err := OpenAICompatibleNormalizer{Client: llmServer.Client(), BaseURL: llmServer.URL}.Normalize(context.Background(), LLMNormalizeRequest{
		Model:      "deepseek-v4-pro",
		Credential: "secret-llm-key",
		Prompt:     "prompt",
	})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	assertNoSecretInMap(t, llm.Raw, "secret-llm-key")
	assertNoSecretInMap(t, llm.Content, "secret-llm-key")
}

func assertNoSecretInMap(t *testing.T, value map[string]any, secret string) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatalf("metadata leaked secret %q: %s", secret, string(data))
	}
}
