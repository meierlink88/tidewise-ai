package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TavilySearchAdapter struct {
	Client  *http.Client
	BaseURL string
}

func (a TavilySearchAdapter) Search(ctx context.Context, request SearchRequest) (SearchResponse, error) {
	baseURL := strings.TrimRight(a.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.tavily.com"
	}
	body := map[string]any{
		"query":       request.Query,
		"max_results": request.MaxResults,
	}
	for _, key := range []string{"topic", "search_depth", "time_range", "include_domains", "exclude_domains", "include_raw_content"} {
		if value, ok := request.Options[key]; ok {
			body[key] = value
		}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("encode tavily request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/search", bytes.NewReader(data))
	if err != nil {
		return SearchResponse{}, fmt.Errorf("build tavily request: %w", err)
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
		return SearchResponse{}, fmt.Errorf("call tavily: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return SearchResponse{}, fmt.Errorf("tavily status %d", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("read tavily response: %w", err)
	}

	var payload tavilySearchResponse
	if err := json.Unmarshal(content, &payload); err != nil {
		return SearchResponse{}, fmt.Errorf("decode tavily response: %w", err)
	}
	results := make([]SearchResultCandidate, 0, len(payload.Results))
	for index, item := range payload.Results {
		results = append(results, SearchResultCandidate{
			Provider:    "tavily",
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Snippet:     strings.TrimSpace(item.Content),
			RawContent:  strings.TrimSpace(item.RawContent),
			PublishedAt: parseProviderTime(item.PublishedDate),
			Rank:        index + 1,
			Score:       item.Score,
			Raw: map[string]any{
				"published_date": item.PublishedDate,
			},
		})
	}
	return SearchResponse{
		Results: results,
		Raw: map[string]any{
			"query":      payload.Query,
			"request_id": payload.RequestID,
		},
	}, nil
}

type tavilySearchResponse struct {
	Query     string               `json:"query"`
	RequestID string               `json:"request_id"`
	Results   []tavilySearchResult `json:"results"`
}

type tavilySearchResult struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`
	RawContent    string  `json:"raw_content"`
	Score         float64 `json:"score"`
	PublishedDate string  `json:"published_date"`
}

func parseProviderTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}
