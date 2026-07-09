package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type BochaSearchAdapter struct {
	Client  *http.Client
	BaseURL string
}

func (a BochaSearchAdapter) Search(ctx context.Context, request SearchRequest) (SearchResponse, error) {
	baseURL := strings.TrimRight(a.BaseURL, "/")
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
	data, err := json.Marshal(body)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("encode bocha request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/web-search", bytes.NewReader(data))
	if err != nil {
		return SearchResponse{}, fmt.Errorf("build bocha request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if request.Credential != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+request.Credential)
	}

	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("call bocha: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return SearchResponse{}, fmt.Errorf("bocha status %d", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("read bocha response: %w", err)
	}

	var payload bochaSearchResponse
	if err := json.Unmarshal(content, &payload); err != nil {
		return SearchResponse{}, fmt.Errorf("decode bocha response: %w", err)
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
