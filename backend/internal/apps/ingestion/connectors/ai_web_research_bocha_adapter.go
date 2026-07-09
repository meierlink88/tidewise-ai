package connectors

import (
	"context"
	"net/http"
	"strings"
)

type BochaSearchAdapter struct {
	Client  *http.Client
	BaseURL string
}

func (a BochaSearchAdapter) Search(ctx context.Context, request SearchRequest) (SearchResponse, error) {
	baseURL := strings.TrimRight(firstNonEmpty(request.BaseURL, a.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.bochaai.com"
	}
	body := map[string]any{
		"query": request.Query,
		"count": firstPositive(request.MaxResults, connectorPositiveInt(request.Options["count"])),
	}
	for _, key := range []string{"freshness", "summary"} {
		if value, ok := request.Options[key]; ok {
			body[key] = value
		}
	}
	var payload bochaSearchResponse
	if err := postSearchJSON(ctx, a.Client, "bocha", baseURL, "/v1/web-search", request.Credential, body, &payload); err != nil {
		return SearchResponse{}, err
	}
	results := make([]SearchResultCandidate, 0, len(payload.WebPages.Value))
	for index, item := range payload.WebPages.Value {
		results = append(results, SearchResultCandidate{
			Provider:    "bocha_web_search",
			Title:       strings.TrimSpace(item.Name),
			URL:         strings.TrimSpace(item.URL),
			Snippet:     strings.TrimSpace(item.Snippet),
			RawContent:  strings.TrimSpace(item.Summary),
			SourceName:  strings.TrimSpace(item.SiteName),
			PublishedAt: parseProviderTime(item.DatePublished),
			Rank:        index + 1,
			Raw: map[string]any{
				"datePublished": item.DatePublished,
				"siteName":      item.SiteName,
			},
		})
	}
	return SearchResponse{
		Results: results,
		Raw: map[string]any{
			"type": payload.Type,
		},
	}, nil
}

type bochaSearchResponse struct {
	Type     string `json:"_type"`
	WebPages struct {
		Value []bochaWebPage `json:"value"`
	} `json:"webPages"`
}

type bochaWebPage struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	SiteName      string `json:"siteName"`
	Snippet       string `json:"snippet"`
	Summary       string `json:"summary"`
	DatePublished string `json:"datePublished"`
}
