package connectors

import (
	"context"
	"encoding/json"
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
