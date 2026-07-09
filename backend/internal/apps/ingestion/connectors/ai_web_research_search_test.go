package connectors

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSearchPlanExecutorRunsFakeAdaptersAndReportsProviderErrors(t *testing.T) {
	executor := SearchPlanExecutor{
		Adapters: map[string]SearchAdapter{
			"tavily": fakeSearchAdapter{
				results: []SearchResultCandidate{
					{
						Provider:    "tavily",
						Title:       "全球央行政策变化",
						URL:         "https://example.com/global-central-bank",
						Snippet:     "全球央行政策摘要",
						SourceName:  "Example Global",
						PublishedAt: time.Date(2026, 7, 9, 8, 0, 0, 0, time.UTC),
					},
				},
			},
			"bocha_web_search": fakeSearchAdapter{err: errors.New("provider timeout")},
		},
	}

	result, err := executor.Search(context.Background(), WebSearchPlan{
		Mode: "parallel",
		Tools: []WebSearchToolConfig{
			{Provider: "tavily", CredentialRef: "env:TAVILY_API_KEY", MaxResults: 5},
			{Provider: "bocha_web_search", CredentialRef: "env:BOCHA_API_KEY", MaxResults: 5},
		},
	}, SearchRequest{Query: "global macro news", MaxResults: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got, want := len(result.Results), 1; got != want {
		t.Fatalf("results = %d, want %d", got, want)
	}
	if result.Results[0].Provider != "tavily" {
		t.Fatalf("result provider = %q, want tavily", result.Results[0].Provider)
	}
	if got, want := len(result.ProviderReports), 2; got != want {
		t.Fatalf("provider reports = %d, want %d", got, want)
	}
	if result.ProviderReports[1].Status != "failed" || !strings.Contains(result.ProviderReports[1].Error, "provider timeout") {
		t.Fatalf("failed provider report = %+v, want timeout failure", result.ProviderReports[1])
	}
}

func TestSearchPlanExecutorFailsUnknownProvider(t *testing.T) {
	executor := SearchPlanExecutor{Adapters: map[string]SearchAdapter{}}

	_, err := executor.Search(context.Background(), WebSearchPlan{
		Mode:  "parallel",
		Tools: []WebSearchToolConfig{{Provider: "missing", CredentialRef: "env:MISSING", MaxResults: 5}},
	}, SearchRequest{Query: "news"})
	if err == nil || !strings.Contains(err.Error(), "search adapter") {
		t.Fatalf("Search() error = %v, want missing adapter", err)
	}
}

func TestSearchPlanExecutorDedupesRanksTrustedDomainsAndLimitsResults(t *testing.T) {
	executor := SearchPlanExecutor{
		Adapters: map[string]SearchAdapter{
			"tavily": fakeSearchAdapter{
				results: []SearchResultCandidate{
					{Provider: "tavily", Title: "重复新闻", URL: "https://example.com/duplicate", Snippet: "A"},
					{Provider: "tavily", Title: "可信来源新闻", URL: "https://pbc.gov.cn/news", Snippet: "B"},
				},
			},
			"bocha_web_search": fakeSearchAdapter{
				results: []SearchResultCandidate{
					{Provider: "bocha_web_search", Title: "重复新闻", URL: "https://example.com/duplicate", Snippet: "A2"},
					{Provider: "bocha_web_search", Title: "普通来源新闻", URL: "https://media.example/news", Snippet: "C"},
				},
			},
		},
		TrustedDomains: []string{"pbc.gov.cn"},
	}

	result, err := executor.Search(context.Background(), WebSearchPlan{
		Mode: "parallel",
		Tools: []WebSearchToolConfig{
			{Provider: "tavily", CredentialRef: "env:TAVILY_API_KEY", MaxResults: 5},
			{Provider: "bocha_web_search", CredentialRef: "env:BOCHA_API_KEY", MaxResults: 5},
		},
	}, SearchRequest{Query: "news", MaxResults: 2})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got, want := len(result.Results), 2; got != want {
		t.Fatalf("results = %d, want %d", got, want)
	}
	if result.Results[0].URL != "https://pbc.gov.cn/news" {
		t.Fatalf("first URL = %q, want trusted domain result first", result.Results[0].URL)
	}
	for _, item := range result.Results {
		if item.URL == "https://media.example/news" {
			t.Fatalf("untrusted overflow result must be truncated: %+v", item)
		}
	}
}

func TestSearchPlanExecutorFallbackSkipsLaterToolsAfterSuccess(t *testing.T) {
	second := &countingSearchAdapter{}
	executor := SearchPlanExecutor{
		Adapters: map[string]SearchAdapter{
			"tavily":           fakeSearchAdapter{results: []SearchResultCandidate{{Provider: "tavily", Title: "新闻", URL: "https://example.com/news"}}},
			"bocha_web_search": second,
		},
	}

	result, err := executor.Search(context.Background(), WebSearchPlan{
		Mode: "fallback",
		Tools: []WebSearchToolConfig{
			{Provider: "tavily", CredentialRef: "env:TAVILY_API_KEY", MaxResults: 5},
			{Provider: "bocha_web_search", CredentialRef: "env:BOCHA_API_KEY", MaxResults: 5},
		},
	}, SearchRequest{Query: "news", MaxResults: 5})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got, want := len(result.Results), 1; got != want {
		t.Fatalf("results = %d, want %d", got, want)
	}
	if second.calls != 0 {
		t.Fatalf("fallback second adapter calls = %d, want 0", second.calls)
	}
}

type fakeSearchAdapter struct {
	results []SearchResultCandidate
	err     error
}

func (a fakeSearchAdapter) Search(context.Context, SearchRequest) (SearchResponse, error) {
	if a.err != nil {
		return SearchResponse{}, a.err
	}
	return SearchResponse{Results: a.results}, nil
}

type countingSearchAdapter struct {
	calls int
}

func (a *countingSearchAdapter) Search(context.Context, SearchRequest) (SearchResponse, error) {
	a.calls++
	return SearchResponse{Results: []SearchResultCandidate{{Provider: "bocha_web_search", Title: "新闻", URL: "https://example.com/bocha"}}}, nil
}
