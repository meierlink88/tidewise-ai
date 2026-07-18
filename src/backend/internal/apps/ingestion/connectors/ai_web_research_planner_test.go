package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAICompatibleSearchPlannerMapsRequestAndResponse(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer planner-key" {
			t.Fatalf("Authorization = %q, want bearer key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-plan",
			"model": "qwen3.7-plus",
			"choices": [
				{
					"message": {
						"content": "{\"queries\":[{\"query\":\"近24小时 中国 央行 财政政策 A股 港股 产业影响\",\"region\":\"china\",\"topic\":\"china_policy_market\",\"providers\":[\"bocha_web_search\"],\"max_results\":20,\"reason\":\"覆盖中国政策变化\"}]}"
					}
				}
			],
			"usage": {"prompt_tokens": 80, "completion_tokens": 40}
		}`))
	}))
	defer server.Close()

	planner := OpenAICompatibleSearchPlanner{Client: server.Client(), BaseURL: server.URL}
	response, err := planner.Plan(context.Background(), SearchPlanRequest{
		Model:      "qwen3.7-plus",
		Credential: "planner-key",
		Prompt:     "你是查询计划生成器",
		Variables: map[string]any{
			"time_window": "24h",
			"max_queries": 6,
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if captured["model"] != "qwen3.7-plus" {
		t.Fatalf("model = %v, want qwen3.7-plus", captured["model"])
	}
	thinking, ok := captured["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking = %v, want disabled thinking config", captured["thinking"])
	}
	if captured["response_format"] == nil {
		t.Fatalf("response_format missing from request")
	}
	messages, ok := captured["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %v, want system and user messages", captured["messages"])
	}
	if got, want := len(response.Plan.Queries), 1; got != want {
		t.Fatalf("queries len = %d, want %d", got, want)
	}
	if response.Plan.Queries[0].Providers[0] != "bocha_web_search" {
		t.Fatalf("provider = %q, want bocha_web_search", response.Plan.Queries[0].Providers[0])
	}
	if response.Raw["model"] != "qwen3.7-plus" {
		t.Fatalf("raw model = %v, want qwen3.7-plus", response.Raw["model"])
	}
}

func TestOpenAICompatibleSearchPlannerRejectsNonJSONContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
	}))
	defer server.Close()

	_, err := (OpenAICompatibleSearchPlanner{Client: server.Client(), BaseURL: server.URL}).Plan(context.Background(), SearchPlanRequest{
		Model:      "qwen3.7-plus",
		Credential: "planner-key",
		Prompt:     "prompt",
	})
	if err == nil || !strings.Contains(err.Error(), "decode llm query plan") {
		t.Fatalf("Plan() error = %v, want decode query plan error", err)
	}
}

func TestDecodeAndValidateSearchQueryPlanRejectsInvalidOutput(t *testing.T) {
	config := aiWebResearchFixtureLLMQueryPlanConfig(t)
	cases := []struct {
		name      string
		content   string
		wantError string
	}{
		{
			name:      "invalid json",
			content:   `not json`,
			wantError: "decode search query plan",
		},
		{
			name:      "missing queries",
			content:   `{}`,
			wantError: "queries are required",
		},
		{
			name:      "unknown provider",
			content:   `{"queries":[{"query":"global macro news","providers":["unknown"],"max_results":10}]}`,
			wantError: "provider \"unknown\" is not allowed",
		},
		{
			name:      "too many queries",
			content:   `{"queries":[{"query":"q1","providers":["tavily"],"max_results":10},{"query":"q2","providers":["tavily"],"max_results":10},{"query":"q3","providers":["tavily"],"max_results":10}]}`,
			wantError: "queries exceed max_queries",
		},
		{
			name:      "max results overflow",
			content:   `{"queries":[{"query":"global macro news","providers":["tavily"],"max_results":99}]}`,
			wantError: "query max_results exceeds limit",
		},
		{
			name:      "forbidden raw document field",
			content:   `{"queries":[{"query":"global macro news","providers":["tavily"],"max_results":10,"source_url":"https://example.com"}]}`,
			wantError: "forbidden query plan field",
		},
		{
			name:      "forbidden items field",
			content:   `{"items":[{"title":"news"}],"queries":[{"query":"global macro news","providers":["tavily"],"max_results":10}]}`,
			wantError: "forbidden query plan field",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeAndValidateSearchQueryPlan([]byte(tc.content), config)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("DecodeAndValidateSearchQueryPlan() error = %v, want %q", err, tc.wantError)
			}
		})
	}
}

func TestDecodeAndValidateSearchQueryPlanReturnsSearchQueries(t *testing.T) {
	config := aiWebResearchFixtureLLMQueryPlanConfig(t)
	queries, err := DecodeAndValidateSearchQueryPlan([]byte(`{
		"queries": [
			{
				"query": "近24小时 中国 财政政策 A股",
				"region": "china",
				"topic": "china_policy_market",
				"providers": ["bocha_web_search"],
				"max_results": 20,
				"reason": "覆盖中国政策变化"
			}
		]
	}`), config)
	if err != nil {
		t.Fatalf("DecodeAndValidateSearchQueryPlan() error = %v", err)
	}
	if got, want := len(queries), 1; got != want {
		t.Fatalf("queries len = %d, want %d", got, want)
	}
	if queries[0].Query != "近24小时 中国 财政政策 A股" {
		t.Fatalf("query = %q, want planned query", queries[0].Query)
	}
	if queries[0].Providers[0] != "bocha_web_search" {
		t.Fatalf("provider = %q, want bocha_web_search", queries[0].Providers[0])
	}
}

func aiWebResearchFixtureLLMQueryPlanConfig(t *testing.T) AIWebResearchConfig {
	t.Helper()
	config := aiWebResearchFixtureConfig()
	config["collection_mode"] = "search_results"
	config["search_plan_mode"] = "llm_query_plan"
	config["credential_refs"] = map[string]any{"planner": "env:QWEN_API_KEY"}
	config["prompt_ref"] = "ingestion/ai_web_research/search-plan.v1.md"
	config["prompt_variables"] = map[string]any{
		"language":    "zh-CN",
		"time_window": "24h",
		"max_queries": 2,
	}
	config["max_results"] = 50
	delete(config, "search_queries")
	parsed, err := ParseAIWebResearchConfig(aiWebResearchFixtureSource(config))
	if err != nil {
		t.Fatalf("ParseAIWebResearchConfig() error = %v", err)
	}
	return parsed
}
