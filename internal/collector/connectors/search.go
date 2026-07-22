package connectors

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
)

type ParallelSearch struct {
	APIKey   string
	Endpoint string
	Client   HTTPClient
}

func (c ParallelSearch) Name() string { return "parallel_search" }

func (c ParallelSearch) Collect(ctx context.Context, request collector.Request) ([]collector.Candidate, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("Parallel API key is missing")
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "https://api.parallel.ai/v1/search"
	}
	client := c.Client
	if client == nil {
		client = defaultClient()
	}
	body := map[string]any{
		"objective":       request.Prompt,
		"search_queries":  request.SearchQueries,
		"max_chars_total": 50000,
	}
	var response struct {
		Results []struct {
			URL         string   `json:"url"`
			Title       string   `json:"title"`
			PublishDate string   `json:"publish_date"`
			Excerpts    []string `json:"excerpts"`
		} `json:"results"`
	}
	if err := postJSON(ctx, client, endpoint, map[string]string{"x-api-key": c.APIKey}, body, &response); err != nil {
		return nil, err
	}
	results := make([]collector.Candidate, 0, min(request.CandidateLimit, len(response.Results)))
	for _, item := range response.Results {
		if len(results) >= request.CandidateLimit {
			break
		}
		content := strings.TrimSpace(strings.Join(item.Excerpts, "\n\n"))
		results = appendCandidate(results, collector.Candidate{
			Title: item.Title, URL: item.URL, PublishedAtHint: item.PublishDate,
			SourceName: host(item.URL), Content: content,
			ContentLevel: levelFor(content, collector.LevelSnippet), SourceType: "news",
		})
	}
	return results, nil
}

type Tavily struct {
	APIKey     string
	Endpoint   string
	Client     HTTPClient
	Topic      string
	MaxResults int
}

func (c Tavily) Name() string { return "tavily" }

func (c Tavily) Collect(ctx context.Context, request collector.Request) ([]collector.Candidate, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("Tavily API key is missing")
	}
	if !validTavilyTopic(c.Topic) {
		return nil, fmt.Errorf("Tavily Topic must be general, news, or finance")
	}
	if c.MaxResults < 1 || c.MaxResults > 20 {
		return nil, fmt.Errorf("Tavily MaxResults must be between 1 and 20")
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "https://api.tavily.com/search"
	}
	client := c.Client
	if client == nil {
		client = defaultClient()
	}
	query := request.CombinedQuery
	start := request.CollectedAt.Add(-time.Duration(request.TimeWindowHours) * time.Hour).Format("2006-01-02")
	maxResults := min(request.CandidateLimit, c.MaxResults)
	var response struct {
		Results []struct {
			Title         string `json:"title"`
			URL           string `json:"url"`
			Content       string `json:"content"`
			PublishedDate string `json:"published_date"`
		} `json:"results"`
	}
	body := map[string]any{
		"query": query, "topic": c.Topic, "search_depth": "advanced",
		"chunks_per_source": 3, "max_results": maxResults,
		"include_answer": false,
		"start_date":     start, "end_date": request.CollectedAt.Format("2006-01-02"),
	}
	if err := postJSON(ctx, client, endpoint, map[string]string{"Authorization": "Bearer " + c.APIKey}, body, &response); err != nil {
		return nil, err
	}
	results := make([]collector.Candidate, 0, min(maxResults, len(response.Results)))
	for _, item := range response.Results {
		if len(results) >= maxResults {
			break
		}
		content := strings.TrimSpace(item.Content)
		results = appendCandidate(results, collector.Candidate{
			Title: item.Title, URL: item.URL, PublishedAtHint: item.PublishedDate,
			SourceName: host(item.URL), Content: content,
			ContentLevel: levelFor(content, collector.LevelSnippet), SourceType: "news",
		})
	}
	return results, nil
}

func validTavilyTopic(topic string) bool {
	switch topic {
	case "general", "news", "finance":
		return true
	default:
		return false
	}
}

type Bocha struct {
	APIKey   string
	Endpoint string
	Client   HTTPClient
}

func (c Bocha) Name() string { return "bocha" }

func (c Bocha) Collect(ctx context.Context, request collector.Request) ([]collector.Candidate, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("Bocha API key is missing")
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "https://api.bochaai.com/v1/web-search"
	}
	client := c.Client
	if client == nil {
		client = defaultClient()
	}
	var response struct {
		Data struct {
			WebPages struct {
				Value []struct {
					Name, URL, Summary, Snippet, DatePublished, SiteName string
				} `json:"value"`
			} `json:"webPages"`
		} `json:"data"`
	}
	body := map[string]any{
		"query":     request.CombinedQuery,
		"freshness": bochaFreshness(request.TimeWindowHours),
		"summary":   true, "count": request.CandidateLimit,
	}
	if err := postJSON(ctx, client, endpoint, map[string]string{"Authorization": "Bearer " + c.APIKey}, body, &response); err != nil {
		return nil, err
	}
	limit := min(request.CandidateLimit, 10)
	results := make([]collector.Candidate, 0, min(limit, len(response.Data.WebPages.Value)))
	for _, item := range response.Data.WebPages.Value {
		if len(results) >= limit {
			break
		}
		content := strings.TrimSpace(item.Summary)
		level := collector.LevelSummary
		if content == "" {
			content = strings.TrimSpace(item.Snippet)
			level = levelFor(content, collector.LevelSnippet)
		}
		results = appendCandidate(results, collector.Candidate{
			Title: item.Name, URL: item.URL, PublishedAtHint: item.DatePublished,
			SourceName: item.SiteName, Content: content, ContentLevel: level, SourceType: "news",
		})
	}
	return results, nil
}

func appendCandidate(results []collector.Candidate, item collector.Candidate) []collector.Candidate {
	item.Title = strings.TrimSpace(item.Title)
	item.URL = strings.TrimSpace(item.URL)
	if strings.TrimSpace(item.Content) == "" {
		item.Content = item.Title
		item.ContentLevel = collector.LevelTitle
	}
	item.ContentOrigin = collector.ContentOrigin
	return append(results, item)
}

func levelFor(content string, populated collector.ContentLevel) collector.ContentLevel {
	if strings.TrimSpace(content) == "" {
		return collector.LevelTitle
	}
	return populated
}

func host(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func bochaFreshness(hours int) string {
	switch {
	case hours <= 24:
		return "oneDay"
	case hours <= 7*24:
		return "oneWeek"
	case hours <= 30*24:
		return "oneMonth"
	default:
		return "oneYear"
	}
}
