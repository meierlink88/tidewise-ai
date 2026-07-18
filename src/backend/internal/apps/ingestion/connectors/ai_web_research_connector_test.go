package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestAIWebResearchConnectorRegistryRunsFixtureSource(t *testing.T) {
	searchAdapters := map[string]SearchAdapter{
		"tavily": fakeCredentialSearchAdapter{
			wantCredential: "tavily-secret",
			results: []SearchResultCandidate{{
				Provider:    "tavily",
				Title:       "央行政策新闻",
				URL:         "https://pbc.gov.cn/news",
				Snippet:     "央行政策新闻摘要",
				SourceName:  "中国人民银行",
				PublishedAt: time.Date(2026, 7, 9, 8, 0, 0, 0, time.UTC),
			}},
		},
		"bocha_web_search": fakeCredentialSearchAdapter{
			wantCredential: "bocha-secret",
			results: []SearchResultCandidate{{
				Provider:   "bocha_web_search",
				Title:      "财政政策新闻",
				URL:        "https://mof.gov.cn/news",
				Snippet:    "财政政策新闻摘要",
				SourceName: "财政部",
			}},
		},
	}
	normalizer := &fakeLLMNormalizer{
		content: map[string]any{
			"items": []any{
				map[string]any{
					"title":          "央行释放政策信号",
					"content_text":   "央行释放政策信号，市场关注流动性变化。",
					"source_name":    "中国人民银行",
					"source_url":     "https://pbc.gov.cn/news",
					"published_at":   "2026-07-09T08:00:00Z",
					"language":       "zh-CN",
					"content_origin": "search_snippet",
				},
			},
		},
	}

	registry := core.NewRegistry()
	RegisterAIWebResearchConnectors(registry, AIWebResearchRegistryOptions{
		SearchAdapters: searchAdapters,
		Normalizer:     normalizer,
		PromptLoader:   fakePromptLoader{text: "把搜索结果结构化"},
		CredentialResolver: fakeCredentialResolver{values: map[string]string{
			"env:TAVILY_API_KEY":   "tavily-secret",
			"env:BOCHA_API_KEY":    "bocha-secret",
			"env:DEEPSEEK_API_KEY": "llm-secret",
		}},
	})

	source := aiWebResearchConnectorFixtureSource(domain.SourceCatalogStatusActive)
	connector, err := registry.Connector("llm_web_research")
	if err != nil {
		t.Fatalf("Connector() error = %v", err)
	}
	parser, err := registry.Parser("llm_research_items")
	if err != nil {
		t.Fatalf("Parser() error = %v", err)
	}

	response, err := connector.Fetch(context.Background(), source, core.Credential{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if response.ContentType != "application/json" {
		t.Fatalf("ContentType = %q, want application/json", response.ContentType)
	}
	if normalizer.lastCredential != "llm-secret" {
		t.Fatalf("llm credential = %q, want llm-secret", normalizer.lastCredential)
	}
	if got, want := len(normalizer.lastSearchResults), 2; got != want {
		t.Fatalf("search results = %d, want %d", got, want)
	}

	docs, err := parser.Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("docs len = %d, want 1", len(docs))
	}
	if docs[0].Title != "央行释放政策信号" {
		t.Fatalf("Title = %q, want normalized item title", docs[0].Title)
	}
}

func TestAIWebResearchConnectorFailsWhenAllSearchToolsHaveMissingCredentials(t *testing.T) {
	registry := core.NewRegistry()
	RegisterAIWebResearchConnectors(registry, AIWebResearchRegistryOptions{
		SearchAdapters: map[string]SearchAdapter{
			"tavily":           fakeCredentialSearchAdapter{results: []SearchResultCandidate{{Title: "新闻", URL: "https://example.com/news"}}},
			"bocha_web_search": fakeCredentialSearchAdapter{results: []SearchResultCandidate{{Title: "新闻", URL: "https://example.com/bocha"}}},
		},
		Normalizer:         &fakeLLMNormalizer{},
		PromptLoader:       fakePromptLoader{text: "prompt"},
		CredentialResolver: fakeCredentialResolver{values: map[string]string{"env:DEEPSEEK_API_KEY": "llm-secret"}},
	})
	connector, err := registry.Connector("llm_web_research")
	if err != nil {
		t.Fatalf("Connector() error = %v", err)
	}

	_, err = connector.Fetch(context.Background(), aiWebResearchConnectorFixtureSource(domain.SourceCatalogStatusActive), core.Credential{})
	if err == nil || !strings.Contains(err.Error(), "web search returned no results") {
		t.Fatalf("Fetch() error = %v, want no search results", err)
	}
}

func TestAIWebResearchConnectorStaticSearchPlanMapsSearchResultsWithoutLLM(t *testing.T) {
	tavily := &queryCapturingSearchAdapter{
		provider:       "tavily",
		wantCredential: "tavily-secret",
		results: []SearchResultCandidate{{
			Provider:    "tavily",
			Title:       "Fed signals policy path",
			URL:         "https://www.reuters.com/markets/fed-policy",
			Snippet:     "Federal Reserve officials discussed the policy path.",
			RawContent:  "Federal Reserve officials discussed the policy path and market expectations.",
			PublishedAt: time.Date(2026, 7, 9, 7, 0, 0, 0, time.UTC),
		}},
	}
	bocha := &queryCapturingSearchAdapter{
		provider:       "bocha_web_search",
		wantCredential: "bocha-secret",
		results: []SearchResultCandidate{{
			Provider:   "bocha_web_search",
			Title:      "中国财政政策释放信号",
			URL:        "https://www.mof.gov.cn/news/policy",
			Snippet:    "财政政策释放信号，市场关注产业影响。",
			SourceName: "财政部",
		}},
	}
	normalizer := &failingLLMNormalizer{t: t}

	registry := core.NewRegistry()
	RegisterAIWebResearchConnectors(registry, AIWebResearchRegistryOptions{
		SearchAdapters: map[string]SearchAdapter{
			"tavily":           tavily,
			"bocha_web_search": bocha,
		},
		Normalizer:   normalizer,
		PromptLoader: fakePromptLoader{err: errors.New("prompt loader must not be called")},
		CredentialResolver: fakeCredentialResolver{values: map[string]string{
			"env:TAVILY_API_KEY": "tavily-secret",
			"env:BOCHA_API_KEY":  "bocha-secret",
		}},
	})
	connector, err := registry.Connector("llm_web_research")
	if err != nil {
		t.Fatalf("Connector() error = %v", err)
	}
	parser, err := registry.Parser("llm_research_items")
	if err != nil {
		t.Fatalf("Parser() error = %v", err)
	}

	source := aiWebResearchStaticSearchFixtureSource(domain.SourceCatalogStatusActive)
	response, err := connector.Fetch(context.Background(), source, core.Credential{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if normalizer.called {
		t.Fatalf("LLM normalizer must not be called in static search mode")
	}
	if got, want := len(bocha.queries), 1; got != want {
		t.Fatalf("bocha queries = %d, want %d", got, want)
	}
	if bocha.queries[0] != "近24小时 中国 财经 政策 A股 港股 产业影响" {
		t.Fatalf("bocha query = %q, want configured China query", bocha.queries[0])
	}
	if got, want := len(tavily.queries), 1; got != want {
		t.Fatalf("tavily queries = %d, want %d", got, want)
	}
	if tavily.queries[0] != "past 24 hours global macro market impact stocks" {
		t.Fatalf("tavily query = %q, want configured global query", tavily.queries[0])
	}

	docs, err := parser.Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got, want := len(docs), 2; got != want {
		t.Fatalf("docs len = %d, want %d", got, want)
	}
	if docs[0].Title != "中国财政政策释放信号" {
		t.Fatalf("first title = %q, want trusted domain China result first", docs[0].Title)
	}
	if docs[0].ContentText != "财政政策释放信号，市场关注产业影响。" {
		t.Fatalf("first content = %q, want provider snippet", docs[0].ContentText)
	}
	if docs[1].ContentText != "Federal Reserve officials discussed the policy path and market expectations." {
		t.Fatalf("second content = %q, want raw content", docs[1].ContentText)
	}
}

func TestAIWebResearchConnectorLLMQueryPlanUsesPlannerAndMapsSearchResultsWithoutNormalizer(t *testing.T) {
	tavily := &queryCapturingSearchAdapter{
		provider:       "tavily",
		wantCredential: "tavily-secret",
		results: []SearchResultCandidate{{
			Provider:   "tavily",
			Title:      "Global central banks adjust policy",
			URL:        "https://www.reuters.com/markets/global-central-banks",
			Snippet:    "Central banks adjusted policy expectations.",
			SourceName: "Reuters",
		}},
	}
	bocha := &queryCapturingSearchAdapter{
		provider:       "bocha_web_search",
		wantCredential: "bocha-secret",
		results: []SearchResultCandidate{{
			Provider:   "bocha_web_search",
			Title:      "中国央行政策信号",
			URL:        "https://www.pbc.gov.cn/news/policy",
			Snippet:    "央行政策信号影响市场流动性预期。",
			SourceName: "中国人民银行",
		}},
	}
	planner := &fakeSearchPlanner{
		plan: SearchQueryPlan{Queries: []PlannedSearchQuery{
			{
				Query:      "近24小时 中国 央行 财政政策 A股 港股 产业影响",
				Region:     "china",
				Topic:      "china_policy_market",
				Providers:  []string{"bocha_web_search"},
				MaxResults: 2,
			},
			{
				Query:      "past 24 hours global central bank fiscal policy stock market impact",
				Region:     "global",
				Topic:      "global_macro_market",
				Providers:  []string{"tavily"},
				MaxResults: 2,
			},
		}},
	}

	registry := core.NewRegistry()
	RegisterAIWebResearchConnectors(registry, AIWebResearchRegistryOptions{
		SearchAdapters: map[string]SearchAdapter{
			"tavily":           tavily,
			"bocha_web_search": bocha,
		},
		SearchPlanner: planner,
		Normalizer:    &failingLLMNormalizer{t: t},
		PromptLoader:  fakePromptLoader{text: "把采集意图转换为查询计划"},
		CredentialResolver: fakeCredentialResolver{values: map[string]string{
			"env:TAVILY_API_KEY": "tavily-secret",
			"env:BOCHA_API_KEY":  "bocha-secret",
			"env:QWEN_API_KEY":   "planner-secret",
		}},
	})
	connector, err := registry.Connector("llm_web_research")
	if err != nil {
		t.Fatalf("Connector() error = %v", err)
	}
	parser, err := registry.Parser("llm_research_items")
	if err != nil {
		t.Fatalf("Parser() error = %v", err)
	}

	source := aiWebResearchLLMQueryPlanFixtureSource(domain.SourceCatalogStatusActive)
	response, err := connector.Fetch(context.Background(), source, core.Credential{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if planner.lastCredential != "planner-secret" {
		t.Fatalf("planner credential = %q, want planner-secret", planner.lastCredential)
	}
	if got, want := len(bocha.queries), 1; got != want {
		t.Fatalf("bocha queries = %d, want %d", got, want)
	}
	if bocha.queries[0] != "近24小时 中国 央行 财政政策 A股 港股 产业影响" {
		t.Fatalf("bocha query = %q, want planner China query", bocha.queries[0])
	}
	if got, want := len(tavily.queries), 1; got != want {
		t.Fatalf("tavily queries = %d, want %d", got, want)
	}

	docs, err := parser.Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got, want := len(docs), 2; got != want {
		t.Fatalf("docs len = %d, want %d", got, want)
	}
	if docs[0].Title != "中国央行政策信号" {
		t.Fatalf("first title = %q, want trusted China result first", docs[0].Title)
	}
}

func aiWebResearchConnectorFixtureSource(status domain.SourceCatalogStatus) domain.SourceCatalog {
	return domain.SourceCatalog{
		ID:            "tidewise:ai-web-research:test",
		IngestChannel: "ai_web_research",
		ProviderKey:   "llm_web_research",
		ConnectorKey:  "llm_web_research",
		ParserKey:     "llm_research_items",
		SourceType:    "news",
		SourceName:    "AI Web Research Test",
		SourceURL:     "tidewise://ai-web-research/test",
		TopicHint:     "近24小时中国财经新闻",
		SourceConfig: map[string]any{
			"web_search_plan": map[string]any{
				"mode": "parallel",
				"tools": []any{
					map[string]any{"provider": "tavily", "credential_ref": "env:TAVILY_API_KEY", "max_results": 2},
					map[string]any{"provider": "bocha_web_search", "credential_ref": "env:BOCHA_API_KEY", "max_results": 2},
				},
			},
			"credential_refs": map[string]any{"llm": "env:DEEPSEEK_API_KEY"},
			"llm_provider":    "deepseek",
			"api_base_url":    "https://api.deepseek.com",
			"api_protocol":    "openai_compatible",
			"model":           "deepseek-v4-pro",
			"prompt_ref":      "ingestion/ai_web_research/test.v1.md",
			"prompt_version":  "v1",
			"prompt_variables": map[string]any{
				"language": "zh-CN",
			},
			"max_results":        4,
			"output_schema":      map[string]any{"type": "llm_research_items.v1"},
			"source_preferences": map[string]any{"region": "china"},
			"trusted_domains":    []any{"pbc.gov.cn"},
		},
		RateLimitPolicy: map[string]any{"requests_per_minute": 6},
		Status:          status,
	}
}

func aiWebResearchStaticSearchFixtureSource(status domain.SourceCatalogStatus) domain.SourceCatalog {
	return domain.SourceCatalog{
		ID:            "tidewise:ai-web-research:static-test",
		IngestChannel: "ai_web_research",
		ProviderKey:   "llm_web_research",
		ConnectorKey:  "llm_web_research",
		ParserKey:     "llm_research_items",
		SourceType:    "news",
		SourceName:    "AI Web Research Static Test",
		SourceURL:     "tidewise://ai-web-research/static-test",
		TopicHint:     "近24小时全球政经财经热点",
		SourceConfig: map[string]any{
			"collection_mode":  "search_results",
			"search_plan_mode": "static_query_plan",
			"search_queries": []any{
				map[string]any{
					"query":       "近24小时 中国 财经 政策 A股 港股 产业影响",
					"providers":   []any{"bocha_web_search"},
					"max_results": 2,
				},
				map[string]any{
					"query":       "past 24 hours global macro market impact stocks",
					"providers":   []any{"tavily"},
					"max_results": 2,
				},
			},
			"web_search_plan": map[string]any{
				"mode": "parallel",
				"tools": []any{
					map[string]any{"provider": "tavily", "credential_ref": "env:TAVILY_API_KEY", "max_results": 2},
					map[string]any{"provider": "bocha_web_search", "credential_ref": "env:BOCHA_API_KEY", "max_results": 2},
				},
			},
			"max_results":        4,
			"output_schema":      map[string]any{"type": "llm_research_items.v1"},
			"source_preferences": map[string]any{"region": "mixed"},
			"trusted_domains":    []any{"mof.gov.cn", "reuters.com"},
		},
		Status: status,
	}
}

func aiWebResearchLLMQueryPlanFixtureSource(status domain.SourceCatalogStatus) domain.SourceCatalog {
	source := aiWebResearchStaticSearchFixtureSource(status)
	source.ID = "tidewise:ai-web-research:llm-query-plan-test"
	source.SourceName = "AI Web Research LLM Query Plan Test"
	source.SourceConfig["search_plan_mode"] = "llm_query_plan"
	delete(source.SourceConfig, "search_queries")
	source.SourceConfig["credential_refs"] = map[string]any{"planner": "env:QWEN_API_KEY"}
	source.SourceConfig["llm_provider"] = "qwen"
	source.SourceConfig["api_base_url"] = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	source.SourceConfig["api_protocol"] = "openai_compatible"
	source.SourceConfig["model"] = "qwen3.7-plus"
	source.SourceConfig["prompt_ref"] = "ingestion/ai_web_research/search-plan.v1.md"
	source.SourceConfig["prompt_version"] = "v1"
	source.SourceConfig["prompt_variables"] = map[string]any{
		"language":    "zh-CN",
		"time_window": "24h",
		"max_queries": 4,
	}
	source.SourceConfig["trusted_domains"] = []any{"pbc.gov.cn", "reuters.com"}
	return source
}

type fakePromptLoader struct {
	text string
	err  error
}

func (l fakePromptLoader) LoadPrompt(string, string, map[string]any) (string, error) {
	if l.err != nil {
		return "", l.err
	}
	return l.text, nil
}

type fakeCredentialResolver struct {
	values map[string]string
}

func (r fakeCredentialResolver) Resolve(ref string) (string, error) {
	value := r.values[ref]
	if value == "" {
		return "", errFakeMissingCredential
	}
	return value, nil
}

var errFakeMissingCredential = &fakeCredentialError{}

type fakeCredentialError struct{}

func (*fakeCredentialError) Error() string {
	return "credential missing"
}

type fakeCredentialSearchAdapter struct {
	wantCredential string
	results        []SearchResultCandidate
}

type queryCapturingSearchAdapter struct {
	provider       string
	wantCredential string
	results        []SearchResultCandidate
	queries        []string
}

func (a *queryCapturingSearchAdapter) Search(_ context.Context, request SearchRequest) (SearchResponse, error) {
	if a.wantCredential != "" && request.Credential != a.wantCredential {
		return SearchResponse{}, errFakeMissingCredential
	}
	a.queries = append(a.queries, request.Query)
	results := make([]SearchResultCandidate, len(a.results))
	copy(results, a.results)
	for index := range results {
		results[index].Provider = a.provider
	}
	return SearchResponse{Results: results}, nil
}

type failingLLMNormalizer struct {
	t      *testing.T
	called bool
}

func (n *failingLLMNormalizer) Normalize(context.Context, LLMNormalizeRequest) (LLMNormalizeResponse, error) {
	n.called = true
	n.t.Fatalf("LLM normalizer must not be called")
	return LLMNormalizeResponse{}, nil
}

func (a fakeCredentialSearchAdapter) Search(_ context.Context, request SearchRequest) (SearchResponse, error) {
	if a.wantCredential != "" && request.Credential != a.wantCredential {
		return SearchResponse{}, errFakeMissingCredential
	}
	return SearchResponse{Results: a.results}, nil
}

type fakeLLMNormalizer struct {
	content           map[string]any
	lastCredential    string
	lastSearchResults []SearchResultCandidate
}

type fakeSearchPlanner struct {
	plan           SearchQueryPlan
	lastCredential string
}

func (p *fakeSearchPlanner) Plan(_ context.Context, request SearchPlanRequest) (SearchPlanResponse, error) {
	p.lastCredential = request.Credential
	return SearchPlanResponse{Plan: p.plan}, nil
}

func (n *fakeLLMNormalizer) Normalize(_ context.Context, request LLMNormalizeRequest) (LLMNormalizeResponse, error) {
	n.lastCredential = request.Credential
	n.lastSearchResults = append([]SearchResultCandidate(nil), request.SearchResults...)
	if n.content == nil {
		n.content = map[string]any{"items": []any{}}
	}
	content, err := json.Marshal(n.content)
	if err != nil {
		return LLMNormalizeResponse{}, err
	}
	var parsed map[string]any
	if err := json.Unmarshal(content, &parsed); err != nil {
		return LLMNormalizeResponse{}, err
	}
	return LLMNormalizeResponse{Content: parsed}, nil
}
