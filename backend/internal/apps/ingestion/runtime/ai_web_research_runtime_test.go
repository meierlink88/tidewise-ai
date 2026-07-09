package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	ingestionconnectors "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/connectors"
	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestRuntimeRunsAIWebResearchSourceThroughRawDocumentWriter(t *testing.T) {
	source := activeAIWebResearchRuntimeSource()
	repo := repositories.NewInMemoryRepository([]domain.SourceCatalog{source})
	registry := coreingestion.NewRegistry()
	normalizer := &runtimeFakeLLMNormalizer{
		content: map[string]any{
			"items": []any{
				map[string]any{
					"title":          "中国财经政策事件",
					"content_text":   "中国财经政策事件摘要。",
					"source_name":    "新华社",
					"citation_text":  "新华社报道称...",
					"content_origin": "llm_generated_summary",
				},
			},
		},
	}
	ingestionconnectors.RegisterAIWebResearchConnectors(registry, ingestionconnectors.AIWebResearchRegistryOptions{
		SearchAdapters: map[string]ingestionconnectors.SearchAdapter{
			"tavily": runtimeFakeSearchAdapter{
				results: []ingestionconnectors.SearchResultCandidate{{Title: "中国财经政策事件", URL: "https://example.com/news"}},
			},
		},
		Normalizer: normalizer,
		PromptLoader: runtimeFakePromptLoader{
			text: "结构化搜索结果",
		},
		CredentialResolver: runtimeNestedCredentialResolver{values: map[string]string{
			"env:TAVILY_API_KEY":   "tavily-secret",
			"env:DEEPSEEK_API_KEY": "llm-secret",
		}},
	})
	limiter := &runtimeLimiter{}
	job := NewIngestionJobWithOptions(
		coreingestion.NewSourceRegistry(repo),
		registry,
		runtimeCredentialResolver{},
		limiter,
		coreingestion.NewRawDocumentWriter(repo),
		IngestionJobOptions{Concurrency: 1},
	)

	report := job.Run(context.Background(), repositories.SourceCatalogFilter{ProviderKey: "llm_web_research"})
	if report.Succeeded != 1 || report.Failed != 0 {
		t.Fatalf("report = %+v, want one success", report)
	}
	count, err := repo.RawDocumentCount(context.Background(), source.ID)
	if err != nil {
		t.Fatalf("RawDocumentCount() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("raw document count = %d, want 1", count)
	}
	if got, want := len(limiter.Calls()), 1; got != want {
		t.Fatalf("limiter calls = %d, want %d", got, want)
	}

	secondReport := job.Run(context.Background(), repositories.SourceCatalogFilter{ProviderKey: "llm_web_research"})
	if secondReport.Succeeded != 1 || secondReport.Failed != 0 {
		t.Fatalf("second report = %+v, want duplicate write to stay successful", secondReport)
	}
	count, err = repo.RawDocumentCount(context.Background(), source.ID)
	if err != nil {
		t.Fatalf("RawDocumentCount() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("raw document count after duplicate run = %d, want 1", count)
	}
}

func TestRuntimeReportsAIWebResearchCredentialFailure(t *testing.T) {
	source := activeAIWebResearchRuntimeSource()
	repo := repositories.NewInMemoryRepository([]domain.SourceCatalog{source})
	registry := coreingestion.NewRegistry()
	ingestionconnectors.RegisterAIWebResearchConnectors(registry, ingestionconnectors.AIWebResearchRegistryOptions{
		SearchAdapters: map[string]ingestionconnectors.SearchAdapter{
			"tavily": runtimeFakeSearchAdapter{results: []ingestionconnectors.SearchResultCandidate{{Title: "新闻", URL: "https://example.com/news"}}},
		},
		Normalizer:         &runtimeFakeLLMNormalizer{},
		PromptLoader:       runtimeFakePromptLoader{text: "结构化搜索结果"},
		CredentialResolver: runtimeNestedCredentialResolver{values: map[string]string{}},
	})
	job := NewIngestionJob(
		coreingestion.NewSourceRegistry(repo),
		registry,
		runtimeCredentialResolver{},
		&runtimeLimiter{},
		coreingestion.NewRawDocumentWriter(repo),
	)

	report := job.Run(context.Background(), repositories.SourceCatalogFilter{ProviderKey: "llm_web_research"})
	if report.Succeeded != 0 || report.Failed != 1 {
		t.Fatalf("report = %+v, want one failure", report)
	}
	if len(report.Errors) != 1 || !strings.Contains(report.Errors[0], "web search returned no results") {
		t.Fatalf("errors = %+v, want AI search credential failure", report.Errors)
	}
}

func activeAIWebResearchRuntimeSource() domain.SourceCatalog {
	return domain.SourceCatalog{
		ID:              "tidewise:ai-web-research:runtime",
		IngestChannel:   "ai_web_research",
		ProviderKey:     "llm_web_research",
		ConnectorKey:    "llm_web_research",
		ParserKey:       "llm_research_items",
		SourceType:      "news",
		SourceName:      "AI Web Research Runtime",
		SourceURL:       "tidewise://ai-web-research/runtime",
		TopicHint:       "近24小时中国财经新闻",
		RateLimitPolicy: map[string]any{"requests_per_minute": 6},
		SourceConfig: map[string]any{
			"web_search_plan": map[string]any{
				"mode": "parallel",
				"tools": []any{
					map[string]any{"provider": "tavily", "credential_ref": "env:TAVILY_API_KEY", "max_results": 2},
				},
			},
			"credential_refs":    map[string]any{"llm": "env:DEEPSEEK_API_KEY"},
			"llm_provider":       "deepseek",
			"api_base_url":       "https://api.deepseek.com",
			"api_protocol":       "openai_compatible",
			"model":              "deepseek-v4-pro",
			"prompt_ref":         "ingestion/ai_web_research/runtime.v1.md",
			"prompt_version":     "v1",
			"prompt_variables":   map[string]any{"language": "zh-CN"},
			"max_results":        2,
			"output_schema":      map[string]any{"type": "llm_research_items.v1"},
			"source_preferences": map[string]any{"region": "china"},
			"trusted_domains":    []any{"xinhuanet.com"},
		},
		Status: domain.SourceCatalogStatusActive,
	}
}

type runtimeFakeSearchAdapter struct {
	results []ingestionconnectors.SearchResultCandidate
}

func (a runtimeFakeSearchAdapter) Search(context.Context, ingestionconnectors.SearchRequest) (ingestionconnectors.SearchResponse, error) {
	return ingestionconnectors.SearchResponse{Results: a.results}, nil
}

type runtimeFakePromptLoader struct {
	text string
}

func (l runtimeFakePromptLoader) LoadPrompt(string, string, map[string]any) (string, error) {
	return l.text, nil
}

type runtimeNestedCredentialResolver struct {
	values map[string]string
}

func (r runtimeNestedCredentialResolver) Resolve(ref string) (string, error) {
	value := r.values[ref]
	if value == "" {
		return "", errRuntimeMissingCredential
	}
	return value, nil
}

var errRuntimeMissingCredential = &runtimeCredentialError{}

type runtimeCredentialError struct{}

func (*runtimeCredentialError) Error() string {
	return "credential missing"
}

type runtimeFakeLLMNormalizer struct {
	content map[string]any
}

func (n *runtimeFakeLLMNormalizer) Normalize(context.Context, ingestionconnectors.LLMNormalizeRequest) (ingestionconnectors.LLMNormalizeResponse, error) {
	if n.content == nil {
		n.content = map[string]any{"items": []any{}}
	}
	data, err := json.Marshal(n.content)
	if err != nil {
		return ingestionconnectors.LLMNormalizeResponse{}, err
	}
	var content map[string]any
	if err := json.Unmarshal(data, &content); err != nil {
		return ingestionconnectors.LLMNormalizeResponse{}, err
	}
	return ingestionconnectors.LLMNormalizeResponse{Content: content}, nil
}
