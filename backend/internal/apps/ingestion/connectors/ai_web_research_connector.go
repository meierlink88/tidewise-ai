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
