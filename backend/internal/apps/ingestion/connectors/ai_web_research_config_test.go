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
