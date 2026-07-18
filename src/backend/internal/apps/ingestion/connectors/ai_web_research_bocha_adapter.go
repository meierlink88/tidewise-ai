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
	data := payload.Data
	if len(data.WebPages.Value) == 0 && len(payload.WebPages.Value) > 0 {
		data.Type = payload.Type
		data.WebPages = payload.WebPages
	}
	results := make([]SearchResultCandidate, 0, len(data.WebPages.Value))
	for index, item := range data.WebPages.Value {
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
			"code":   payload.Code,
			"log_id": payload.LogID,
			"type":   firstNonEmpty(data.Type, payload.Type),
		},
	}, nil
}

type bochaSearchResponse struct {
	Code     int             `json:"code"`
	LogID    string          `json:"log_id"`
	Message  string          `json:"message"`
	Msg      string          `json:"msg"`
	Type     string          `json:"_type"`
	WebPages bochaWebPages   `json:"webPages"`
	Data     bochaSearchData `json:"data"`
}

type bochaSearchData struct {
	Type     string        `json:"_type"`
	WebPages bochaWebPages `json:"webPages"`
}

type bochaWebPages struct {
	Value []bochaWebPage `json:"value"`
}

type bochaWebPage struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	SiteName      string `json:"siteName"`
	Snippet       string `json:"snippet"`
	Summary       string `json:"summary"`
	DatePublished string `json:"datePublished"`
}
