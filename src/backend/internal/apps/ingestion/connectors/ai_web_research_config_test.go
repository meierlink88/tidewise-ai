package connectors

import (
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestParseAIWebResearchConfigReadsSearchToolsAndLLMSettings(t *testing.T) {
	config, err := ParseAIWebResearchConfig(aiWebResearchFixtureSource(aiWebResearchFixtureConfig()))
	if err != nil {
		t.Fatalf("ParseAIWebResearchConfig() error = %v", err)
	}

	if config.WebSearchPlan.Mode != "parallel" {
		t.Fatalf("Mode = %q, want parallel", config.WebSearchPlan.Mode)
	}
	if got, want := len(config.WebSearchPlan.Tools), 2; got != want {
		t.Fatalf("tools = %d, want %d", got, want)
	}
	if config.WebSearchPlan.Tools[0].Provider != "tavily" {
		t.Fatalf("first provider = %q, want tavily", config.WebSearchPlan.Tools[0].Provider)
	}
	if config.WebSearchPlan.Tools[0].BaseURL != "https://proxy.example.com/tavily" {
		t.Fatalf("first base url = %q, want configured tavily proxy", config.WebSearchPlan.Tools[0].BaseURL)
	}
	if config.WebSearchPlan.Tools[1].Provider != "bocha_web_search" {
		t.Fatalf("second provider = %q, want bocha_web_search", config.WebSearchPlan.Tools[1].Provider)
	}
	if config.WebSearchPlan.Tools[1].BaseURL != "https://proxy.example.com/bocha" {
		t.Fatalf("second base url = %q, want configured bocha proxy", config.WebSearchPlan.Tools[1].BaseURL)
	}
	if config.CredentialRefs["llm"] != "env:DEEPSEEK_API_KEY" {
		t.Fatalf("LLM credential ref = %q, want env ref", config.CredentialRefs["llm"])
	}
	if config.LLMProvider != "deepseek" || config.Model != "deepseek-v4-pro" {
		t.Fatalf("LLM = %s/%s, want deepseek/deepseek-v4-pro", config.LLMProvider, config.Model)
	}
	if config.PromptRef != "ingestion/ai_web_research/cn-finance-daily.v1.md" {
		t.Fatalf("PromptRef = %q, want repo prompt ref", config.PromptRef)
	}
	if config.PromptVariables["language"] != "zh-CN" {
		t.Fatalf("PromptVariables[language] = %v, want zh-CN", config.PromptVariables["language"])
	}
	if got, want := len(config.TrustedDomains), 2; got != want {
		t.Fatalf("TrustedDomains len = %d, want %d", got, want)
	}
}

func TestParseAIWebResearchConfigRejectsMissingRequiredFields(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(map[string]any)
		wantError string
	}{
		{
			name: "web search plan",
			mutate: func(config map[string]any) {
				delete(config, "web_search_plan")
			},
			wantError: "web_search_plan is required",
		},
		{
			name: "tool credential ref",
			mutate: func(config map[string]any) {
				plan := config["web_search_plan"].(map[string]any)
				tool := plan["tools"].([]any)[0].(map[string]any)
				delete(tool, "credential_ref")
			},
			wantError: "tool credential_ref is required",
		},
		{
			name: "llm provider",
			mutate: func(config map[string]any) {
				delete(config, "llm_provider")
			},
			wantError: "llm_provider is required",
		},
		{
			name: "model",
			mutate: func(config map[string]any) {
				delete(config, "model")
			},
			wantError: "model is required",
		},
		{
			name: "prompt ref",
			mutate: func(config map[string]any) {
				delete(config, "prompt_ref")
			},
			wantError: "prompt_ref is required",
		},
		{
			name: "output schema",
			mutate: func(config map[string]any) {
				delete(config, "output_schema")
			},
			wantError: "output_schema is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := aiWebResearchFixtureConfig()
			tc.mutate(config)

			_, err := ParseAIWebResearchConfig(aiWebResearchFixtureSource(config))
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ParseAIWebResearchConfig() error = %v, want %q", err, tc.wantError)
			}
		})
	}
}

func TestParseAIWebResearchConfigReadsStaticSearchPlanWithoutLLMSettings(t *testing.T) {
	config := map[string]any{
		"kind":             "llm_web_research",
		"collection_mode":  "search_results",
		"search_plan_mode": "static_query_plan",
		"search_queries": []any{
			map[string]any{
				"query":       "近24小时 中国 财经 政策 A股 港股 产业影响",
				"region":      "china",
				"topic":       "china_policy_market",
				"providers":   []any{"bocha_web_search"},
				"max_results": 30,
				"options": map[string]any{
					"freshness": "oneDay",
				},
			},
			map[string]any{
				"query":       "past 24 hours global macro geopolitical market impact stocks",
				"region":      "global",
				"topic":       "global_macro_market",
				"providers":   []any{"tavily"},
				"max_results": 20,
			},
		},
		"web_search_plan": map[string]any{
			"mode": "parallel",
			"tools": []any{
				map[string]any{"provider": "tavily", "credential_ref": "env:TAVILY_API_KEY", "max_results": 10},
				map[string]any{"provider": "bocha_web_search", "credential_ref": "env:BOCHA_API_KEY", "max_results": 10},
			},
		},
		"max_results":        50,
		"output_schema":      map[string]any{"type": "llm_research_items.v1"},
		"source_preferences": map[string]any{"region": "mixed"},
		"trusted_domains":    []any{"pbc.gov.cn", "reuters.com"},
	}

	parsed, err := ParseAIWebResearchConfig(aiWebResearchFixtureSource(config))
	if err != nil {
		t.Fatalf("ParseAIWebResearchConfig() error = %v", err)
	}
	if parsed.CollectionMode != "search_results" {
		t.Fatalf("CollectionMode = %q, want search_results", parsed.CollectionMode)
	}
	if parsed.SearchPlanMode != "static_query_plan" {
		t.Fatalf("SearchPlanMode = %q, want static_query_plan", parsed.SearchPlanMode)
	}
	if got, want := len(parsed.SearchQueries), 2; got != want {
		t.Fatalf("SearchQueries len = %d, want %d", got, want)
	}
	if parsed.SearchQueries[0].Providers[0] != "bocha_web_search" {
		t.Fatalf("first query provider = %q, want bocha_web_search", parsed.SearchQueries[0].Providers[0])
	}
	if parsed.SearchQueries[0].Options["freshness"] != "oneDay" {
		t.Fatalf("first query freshness = %v, want oneDay", parsed.SearchQueries[0].Options["freshness"])
	}
	if parsed.LLMProvider != "" || parsed.Model != "" {
		t.Fatalf("static search mode should not require LLM settings, got provider=%q model=%q", parsed.LLMProvider, parsed.Model)
	}
}

func TestParseAIWebResearchConfigReadsLLMQueryPlanSettings(t *testing.T) {
	config := aiWebResearchFixtureConfig()
	config["collection_mode"] = "search_results"
	config["search_plan_mode"] = "llm_query_plan"
	config["prompt_ref"] = "ingestion/ai_web_research/search-plan.v1.md"
	config["prompt_version"] = "v1"
	config["model"] = "qwen3.7-plus"
	config["credential_refs"] = map[string]any{
		"planner": "env:QWEN_API_KEY",
	}
	delete(config, "search_queries")
	config["prompt_variables"] = map[string]any{
		"language":          "zh-CN",
		"time_window":       "24h",
		"max_queries":       6,
		"china_ratio":       0.5,
		"allowed_providers": []any{"tavily", "bocha_web_search"},
	}

	parsed, err := ParseAIWebResearchConfig(aiWebResearchFixtureSource(config))
	if err != nil {
		t.Fatalf("ParseAIWebResearchConfig() error = %v", err)
	}
	if parsed.CollectionMode != "search_results" {
		t.Fatalf("CollectionMode = %q, want search_results", parsed.CollectionMode)
	}
	if parsed.SearchPlanMode != "llm_query_plan" {
		t.Fatalf("SearchPlanMode = %q, want llm_query_plan", parsed.SearchPlanMode)
	}
	if parsed.CredentialRefs["planner"] != "env:QWEN_API_KEY" {
		t.Fatalf("planner credential ref = %q, want env:QWEN_API_KEY", parsed.CredentialRefs["planner"])
	}
	if parsed.PromptRef != "ingestion/ai_web_research/search-plan.v1.md" {
		t.Fatalf("PromptRef = %q, want search plan prompt ref", parsed.PromptRef)
	}
	if parsed.PromptVariables["max_queries"] != 6 {
		t.Fatalf("PromptVariables[max_queries] = %v, want 6", parsed.PromptVariables["max_queries"])
	}
	if len(parsed.SearchQueries) != 0 {
		t.Fatalf("SearchQueries len = %d, want 0 for llm_query_plan source config", len(parsed.SearchQueries))
	}
}

func TestParseAIWebResearchConfigRejectsInvalidLLMQueryPlanSettings(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(map[string]any)
		wantError string
	}{
		{
			name: "missing planner credential ref",
			mutate: func(config map[string]any) {
				config["credential_refs"] = map[string]any{
					"llm": "env:DEEPSEEK_API_KEY",
				}
			},
			wantError: "credential_refs.planner is required",
		},
		{
			name: "missing max queries",
			mutate: func(config map[string]any) {
				config["prompt_variables"] = map[string]any{
					"language":    "zh-CN",
					"time_window": "24h",
				}
			},
			wantError: "prompt_variables.max_queries must be positive",
		},
		{
			name: "missing api base url",
			mutate: func(config map[string]any) {
				delete(config, "api_base_url")
			},
			wantError: "api_base_url is required",
		},
		{
			name: "missing prompt ref",
			mutate: func(config map[string]any) {
				delete(config, "prompt_ref")
			},
			wantError: "prompt_ref is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := aiWebResearchFixtureConfig()
			config["collection_mode"] = "search_results"
			config["search_plan_mode"] = "llm_query_plan"
			config["credential_refs"] = map[string]any{
				"planner": "env:QWEN_API_KEY",
			}
			config["prompt_ref"] = "ingestion/ai_web_research/search-plan.v1.md"
			config["prompt_variables"] = map[string]any{
				"language":    "zh-CN",
				"time_window": "24h",
				"max_queries": 6,
			}
			delete(config, "search_queries")
			tc.mutate(config)

			_, err := ParseAIWebResearchConfig(aiWebResearchFixtureSource(config))
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ParseAIWebResearchConfig() error = %v, want %q", err, tc.wantError)
			}
		})
	}
}

func aiWebResearchFixtureSource(config map[string]any) domain.SourceCatalog {
	return domain.SourceCatalog{
		ID:            "tidewise:ai-web-research:cn-finance-daily",
		IngestChannel: "ai_web_research",
		ProviderKey:   "llm_web_research",
		ConnectorKey:  "llm_web_research",
		ParserKey:     "llm_research_items",
		SourceType:    "news",
		SourceName:    "AI Web Research 中文财经日度采集",
		SourceConfig:  config,
	}
}

func aiWebResearchFixtureConfig() map[string]any {
	return map[string]any{
		"kind": "llm_web_research",
		"web_search_plan": map[string]any{
			"mode": "parallel",
			"tools": []any{
				map[string]any{
					"provider":       "tavily",
					"base_url":       "https://proxy.example.com/tavily",
					"credential_ref": "env:TAVILY_API_KEY",
					"max_results":    10,
					"options": map[string]any{
						"topic":               "news",
						"search_depth":        "advanced",
						"include_raw_content": true,
					},
				},
				map[string]any{
					"provider":       "bocha_web_search",
					"base_url":       "https://proxy.example.com/bocha",
					"credential_ref": "env:BOCHA_API_KEY",
					"max_results":    10,
					"options": map[string]any{
						"freshness": "oneDay",
						"summary":   true,
						"count":     10,
					},
				},
			},
		},
		"credential_refs": map[string]any{
			"llm": "env:DEEPSEEK_API_KEY",
		},
		"llm_provider":   "deepseek",
		"api_base_url":   "https://api.deepseek.com",
		"api_protocol":   "openai_compatible",
		"model":          "deepseek-v4-pro",
		"prompt_ref":     "ingestion/ai_web_research/cn-finance-daily.v1.md",
		"prompt_version": "v1",
		"prompt_variables": map[string]any{
			"language":    "zh-CN",
			"time_window": "24h",
		},
		"max_results": 20,
		"output_schema": map[string]any{
			"type": "llm_research_items.v1",
		},
		"source_preferences": map[string]any{
			"region": "china_finance",
		},
		"trusted_domains": []any{"pbc.gov.cn", "sse.com.cn"},
	}
}
