package connectors

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type SearchAdapter interface {
	Search(context.Context, SearchRequest) (SearchResponse, error)
}

type SearchRequest struct {
	Query          string
	MaxResults     int
	Credential     string
	Options        map[string]any
	TrustedDomains []string
}

type SearchResponse struct {
	Results []SearchResultCandidate
	Usage   map[string]any
	Raw     map[string]any
}

type SearchResultCandidate struct {
	Provider    string
	Title       string
	URL         string
	Snippet     string
	RawContent  string
	SourceName  string
	PublishedAt time.Time
	Rank        int
	Score       float64
	Raw         map[string]any
}

type SearchPlanResult struct {
	Results         []SearchResultCandidate
	ProviderReports []SearchProviderReport
}

type SearchProviderReport struct {
	Provider    string
	Status      string
	ResultCount int
	Error       string
}

type SearchPlanExecutor struct {
	Adapters           map[string]SearchAdapter
	TrustedDomains     []string
	CredentialResolver CredentialResolver
}

func (e SearchPlanExecutor) Search(ctx context.Context, plan WebSearchPlan, request SearchRequest) (SearchPlanResult, error) {
	var result SearchPlanResult
	for _, tool := range plan.Tools {
		adapter, ok := e.Adapters[tool.Provider]
		if !ok {
			return SearchPlanResult{}, fmt.Errorf("search adapter %q is not registered", tool.Provider)
		}
		toolRequest := request
		toolRequest.MaxResults = firstPositive(tool.MaxResults, request.MaxResults)
		toolRequest.Options = tool.Options
		if e.CredentialResolver != nil {
			credential, err := e.CredentialResolver.Resolve(tool.CredentialRef)
			if err != nil {
				result.ProviderReports = append(result.ProviderReports, SearchProviderReport{
					Provider: tool.Provider,
					Status:   "failed",
					Error:    err.Error(),
				})
				continue
			}
			toolRequest.Credential = credential
		}
		response, err := adapter.Search(ctx, toolRequest)
		report := SearchProviderReport{Provider: tool.Provider}
		if err != nil {
			report.Status = "failed"
			report.Error = err.Error()
			result.ProviderReports = append(result.ProviderReports, report)
			continue
		}
		report.Status = "succeeded"
		report.ResultCount = len(response.Results)
		result.ProviderReports = append(result.ProviderReports, report)
		result.Results = append(result.Results, response.Results...)
		if plan.Mode == "fallback" && len(response.Results) > 0 {
			break
		}
	}
	result.Results = e.rankAndLimit(result.Results, request.MaxResults)
	return result, nil
}

func (e SearchPlanExecutor) rankAndLimit(results []SearchResultCandidate, maxResults int) []SearchResultCandidate {
	deduped := make([]SearchResultCandidate, 0, len(results))
	seen := map[string]struct{}{}
	for _, item := range results {
		key := normalizedResultKey(item)
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(item.Title + "|" + item.Snippet))
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, item)
	}
	sort.SliceStable(deduped, func(i, j int) bool {
		return trustedDomainRank(deduped[i].URL, e.TrustedDomains) > trustedDomainRank(deduped[j].URL, e.TrustedDomains)
	})
	if maxResults > 0 && len(deduped) > maxResults {
		return deduped[:maxResults]
	}
	return deduped
}

func normalizedResultKey(item SearchResultCandidate) string {
	parsed, err := url.Parse(strings.TrimSpace(item.URL))
	if err != nil || parsed.Host == "" {
		return ""
	}
	parsed.Fragment = ""
	return strings.ToLower(parsed.String())
}

func trustedDomainRank(rawURL string, trustedDomains []string) int {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}
	host := strings.ToLower(parsed.Hostname())
	for _, domain := range trustedDomains {
		trusted := strings.ToLower(strings.TrimSpace(domain))
		if trusted != "" && (host == trusted || strings.HasSuffix(host, "."+trusted)) {
			return 1
		}
	}
	return 0
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
