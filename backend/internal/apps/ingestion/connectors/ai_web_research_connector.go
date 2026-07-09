package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
	"github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/parsers"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/promptstore"
)

type AIWebResearchRegistryOptions struct {
	Client             *http.Client
	PromptRoot         string
	SearchAdapters     map[string]SearchAdapter
	SearchPlanner      SearchPlanner
	Normalizer         LLMNormalizer
	PromptLoader       AIWebResearchPromptLoader
	CredentialResolver CredentialResolver
}

type AIWebResearchPromptLoader interface {
	LoadPrompt(ref string, version string, variables map[string]any) (string, error)
}

type CredentialResolver interface {
	Resolve(string) (string, error)
}

type LLMNormalizer interface {
	Normalize(context.Context, LLMNormalizeRequest) (LLMNormalizeResponse, error)
}

type SearchPlanner interface {
	Plan(context.Context, SearchPlanRequest) (SearchPlanResponse, error)
}

type PromptStoreLoader struct {
	Root string
}

func (l PromptStoreLoader) LoadPrompt(ref string, version string, variables map[string]any) (string, error) {
	prompt, err := promptstore.Loader{Root: l.Root}.Load(ref, version, variables)
	if err != nil {
		return "", err
	}
	return prompt.Text, nil
}

type LLMWebResearchConnector struct {
	Client             *http.Client
	PromptRoot         string
	SearchExecutor     SearchPlanExecutor
	SearchPlanner      SearchPlanner
	Normalizer         LLMNormalizer
	PromptLoader       AIWebResearchPromptLoader
	CredentialResolver CredentialResolver
}

func RegisterAIWebResearchConnectors(registry *coreingestion.Registry, options AIWebResearchRegistryOptions) {
	registry.RegisterConnector("llm_web_research", LLMWebResearchConnector{
		Client: options.Client,
		SearchExecutor: SearchPlanExecutor{
			Adapters:           defaultSearchAdapters(options.Client, options.SearchAdapters),
			CredentialResolver: options.CredentialResolver,
		},
		SearchPlanner:      options.SearchPlanner,
		Normalizer:         options.Normalizer,
		PromptLoader:       firstPromptLoader(options.PromptLoader, PromptStoreLoader{Root: options.PromptRoot}),
		PromptRoot:         options.PromptRoot,
		CredentialResolver: firstCredentialResolver(options.CredentialResolver, coreingestion.EnvCredentialResolver{}),
	})
	registry.RegisterParser("llm_research_items", parsers.LLMResearchItemsParser{})
}

func (c LLMWebResearchConnector) Fetch(ctx context.Context, source domain.SourceCatalog, _ coreingestion.Credential) (coreingestion.RawResponse, error) {
	config, err := ParseAIWebResearchConfig(source)
	if err != nil {
		return coreingestion.RawResponse{}, err
	}
	if config.CollectionMode == "search_results" && config.SearchPlanMode == "static_query_plan" {
		return c.fetchStaticSearchResults(ctx, source, config)
	}
	if config.CollectionMode == "search_results" && config.SearchPlanMode == "llm_query_plan" {
		return c.fetchLLMQueryPlanSearchResults(ctx, source, config)
	}

	prompt, err := c.promptLoader().LoadPrompt(config.PromptRef, config.PromptVersion, config.PromptVariables)
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("load ai web research prompt: %w", err)
	}

	executor := c.SearchExecutor
	if executor.Adapters == nil {
		executor.Adapters = defaultSearchAdapters(c.Client, nil)
	}
	executor.TrustedDomains = config.TrustedDomains
	executor.CredentialResolver = c.credentialResolver()
	searchResult, err := executor.Search(ctx, config.WebSearchPlan, SearchRequest{
		Query:          firstNonEmpty(source.TopicHint, source.SourceName),
		MaxResults:     config.MaxResults,
		TrustedDomains: config.TrustedDomains,
	})
	if err != nil {
		return coreingestion.RawResponse{}, err
	}
	if len(searchResult.Results) == 0 {
		return coreingestion.RawResponse{}, fmt.Errorf("web search returned no results")
	}

	llmCredential, err := c.credentialResolver().Resolve(config.CredentialRefs["llm"])
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("resolve llm credential: %w", err)
	}
	normalizer, err := c.normalizer(config)
	if err != nil {
		return coreingestion.RawResponse{}, err
	}
	normalized, err := normalizer.Normalize(ctx, LLMNormalizeRequest{
		Model:         config.Model,
		Credential:    llmCredential,
		Prompt:        prompt,
		SearchResults: searchResult.Results,
	})
	if err != nil {
		return coreingestion.RawResponse{}, err
	}

	content, err := json.Marshal(normalized.Content)
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("encode normalized llm content: %w", err)
	}
	return coreingestion.RawResponse{
		ContentType: "application/json",
		Content:     content,
		CollectedAt: time.Now(),
	}, nil
}

func (c LLMWebResearchConnector) fetchStaticSearchResults(ctx context.Context, source domain.SourceCatalog, config AIWebResearchConfig) (coreingestion.RawResponse, error) {
	return c.fetchSearchResultsForQueries(ctx, source, config, config.SearchQueries)
}

func (c LLMWebResearchConnector) fetchLLMQueryPlanSearchResults(ctx context.Context, source domain.SourceCatalog, config AIWebResearchConfig) (coreingestion.RawResponse, error) {
	prompt, err := c.promptLoader().LoadPrompt(config.PromptRef, config.PromptVersion, config.PromptVariables)
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("load ai web research prompt: %w", err)
	}
	plannerCredential, err := c.credentialResolver().Resolve(config.CredentialRefs["planner"])
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("resolve llm planner credential: %w", err)
	}
	planner, err := c.searchPlanner(config)
	if err != nil {
		return coreingestion.RawResponse{}, err
	}
	planResponse, err := planner.Plan(ctx, SearchPlanRequest{
		Model:      config.Model,
		Credential: plannerCredential,
		Prompt:     prompt,
		Variables:  config.PromptVariables,
	})
	if err != nil {
		return coreingestion.RawResponse{}, err
	}
	searchQueries, err := ValidateSearchQueryPlan(planResponse.Plan, config)
	if err != nil {
		return coreingestion.RawResponse{}, err
	}
	return c.fetchSearchResultsForQueries(ctx, source, config, searchQueries)
}

func (c LLMWebResearchConnector) fetchSearchResultsForQueries(ctx context.Context, source domain.SourceCatalog, config AIWebResearchConfig, queries []SearchQueryConfig) (coreingestion.RawResponse, error) {
	executor := c.SearchExecutor
	if executor.Adapters == nil {
		executor.Adapters = defaultSearchAdapters(c.Client, nil)
	}
	executor.TrustedDomains = config.TrustedDomains
	executor.CredentialResolver = c.credentialResolver()

	var allResults []SearchResultCandidate
	var reports []SearchProviderReport
	for _, query := range queries {
		plan := webSearchPlanForQuery(config.WebSearchPlan, query)
		if len(plan.Tools) == 0 {
			return coreingestion.RawResponse{}, fmt.Errorf("search query %q has no matching providers", query.Query)
		}
		result, err := executor.Search(ctx, plan, SearchRequest{
			Query:          query.Query,
			MaxResults:     firstPositive(query.MaxResults, config.MaxResults),
			TrustedDomains: config.TrustedDomains,
		})
		if err != nil {
			return coreingestion.RawResponse{}, err
		}
		allResults = append(allResults, result.Results...)
		reports = append(reports, result.ProviderReports...)
	}
	allResults = executor.rankAndLimit(allResults, config.MaxResults)
	if len(allResults) == 0 {
		return coreingestion.RawResponse{}, fmt.Errorf("web search returned no results")
	}

	envelope, itemCount := searchResultsItemsEnvelope(source, config, allResults, reports)
	if itemCount == 0 {
		return coreingestion.RawResponse{}, fmt.Errorf("web search returned no mappable results")
	}
	content, err := json.Marshal(envelope)
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("encode search result items: %w", err)
	}
	return coreingestion.RawResponse{
		ContentType: "application/json",
		Content:     content,
		CollectedAt: time.Now(),
	}, nil
}

func webSearchPlanForQuery(plan WebSearchPlan, query SearchQueryConfig) WebSearchPlan {
	if len(query.Providers) == 0 {
		return plan
	}
	allowed := make(map[string]struct{}, len(query.Providers))
	for _, provider := range query.Providers {
		allowed[provider] = struct{}{}
	}
	filtered := make([]WebSearchToolConfig, 0, len(plan.Tools))
	for _, tool := range plan.Tools {
		if _, ok := allowed[tool.Provider]; !ok {
			continue
		}
		tool.MaxResults = firstPositive(query.MaxResults, tool.MaxResults)
		tool.Options = mergeAnyMaps(tool.Options, query.Options)
		filtered = append(filtered, tool)
	}
	return WebSearchPlan{Mode: plan.Mode, Tools: filtered}
}

func mergeAnyMaps(base map[string]any, override map[string]any) map[string]any {
	if len(base) == 0 && len(override) == 0 {
		return map[string]any{}
	}
	merged := make(map[string]any, len(base)+len(override))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range override {
		merged[key] = value
	}
	return merged
}

func searchResultsItemsEnvelope(source domain.SourceCatalog, config AIWebResearchConfig, results []SearchResultCandidate, reports []SearchProviderReport) (map[string]any, int) {
	items := make([]map[string]any, 0, len(results))
	for _, result := range results {
		contentText := firstNonEmpty(result.RawContent, result.Snippet)
		if result.Title == "" || contentText == "" || firstNonEmpty(result.URL, result.SourceName) == "" {
			continue
		}
		contentOrigin := "search_snippet"
		if result.RawContent != "" {
			contentOrigin = "web_content"
		}
		attributionType := "url"
		if result.URL == "" {
			attributionType = "named_source"
		}
		item := map[string]any{
			"title":                   result.Title,
			"content_text":            contentText,
			"source_url":              result.URL,
			"source_name":             firstNonEmpty(result.SourceName, source.SourceName),
			"source_reference":        result.Provider,
			"provider_source_note":    result.Provider,
			"source_attribution_type": attributionType,
			"language":                configLanguage(config),
			"content_origin":          contentOrigin,
			"evidence_excerpt":        firstNonEmpty(result.Snippet, contentText),
			"relevance_reason":        "由 Web Search provider 按采集查询召回，作为后续事件提取的原始材料。",
			"topic_tags":              []string{"web_search", result.Provider},
		}
		if !result.PublishedAt.IsZero() {
			item["published_at"] = result.PublishedAt.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	return map[string]any{
		"items": items,
		"meta": map[string]any{
			"collection_mode":   config.CollectionMode,
			"search_plan_mode":  config.SearchPlanMode,
			"requested_results": config.MaxResults,
			"actual_results":    len(items),
			"provider_reports":  reports,
		},
	}, len(items)
}

func configLanguage(config AIWebResearchConfig) string {
	if value, ok := config.SourcePreferences["language"]; ok {
		if language := jsonStringValue(value); language != "" {
			return language
		}
	}
	if value, ok := config.PromptVariables["language"]; ok {
		if language := jsonStringValue(value); language != "" {
			return language
		}
	}
	return "zh-CN"
}

func (c LLMWebResearchConnector) promptLoader() AIWebResearchPromptLoader {
	if c.PromptLoader != nil {
		return c.PromptLoader
	}
	return PromptStoreLoader{Root: c.PromptRoot}
}

func (c LLMWebResearchConnector) credentialResolver() CredentialResolver {
	return firstCredentialResolver(c.CredentialResolver, coreingestion.EnvCredentialResolver{})
}

func (c LLMWebResearchConnector) normalizer(config AIWebResearchConfig) (LLMNormalizer, error) {
	if c.Normalizer != nil {
		return c.Normalizer, nil
	}
	if config.APIProtocol != "openai_compatible" {
		return nil, fmt.Errorf("unsupported api_protocol %q", config.APIProtocol)
	}
	return OpenAICompatibleNormalizer{Client: c.Client, BaseURL: config.APIBaseURL}, nil
}

func (c LLMWebResearchConnector) searchPlanner(config AIWebResearchConfig) (SearchPlanner, error) {
	if c.SearchPlanner != nil {
		return c.SearchPlanner, nil
	}
	if config.APIProtocol != "openai_compatible" {
		return nil, fmt.Errorf("unsupported api_protocol %q", config.APIProtocol)
	}
	return OpenAICompatibleSearchPlanner{Client: c.Client, BaseURL: config.APIBaseURL}, nil
}

func defaultSearchAdapters(client *http.Client, overrides map[string]SearchAdapter) map[string]SearchAdapter {
	adapters := map[string]SearchAdapter{
		"tavily":           TavilySearchAdapter{Client: client},
		"bocha_web_search": BochaSearchAdapter{Client: client},
	}
	for key, adapter := range overrides {
		adapters[key] = adapter
	}
	return adapters
}

func firstPromptLoader(values ...AIWebResearchPromptLoader) AIWebResearchPromptLoader {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstCredentialResolver(values ...CredentialResolver) CredentialResolver {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return coreingestion.EnvCredentialResolver{}
}
