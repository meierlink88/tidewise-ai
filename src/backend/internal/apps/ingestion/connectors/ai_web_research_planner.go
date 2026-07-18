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

type SearchPlanRequest struct {
	Model      string
	Credential string
	Prompt     string
	Variables  map[string]any
}

type SearchPlanResponse struct {
	Plan  SearchQueryPlan
	Raw   map[string]any
	Usage map[string]any
}

type SearchQueryPlan struct {
	Queries []PlannedSearchQuery `json:"queries"`
}

type PlannedSearchQuery struct {
	Query      string   `json:"query"`
	Region     string   `json:"region"`
	Topic      string   `json:"topic"`
	Providers  []string `json:"providers"`
	MaxResults int      `json:"max_results"`
	Reason     string   `json:"reason"`
}

type OpenAICompatibleSearchPlanner struct {
	Client  *http.Client
	BaseURL string
}

func (p OpenAICompatibleSearchPlanner) Plan(ctx context.Context, request SearchPlanRequest) (SearchPlanResponse, error) {
	baseURL := strings.TrimRight(p.BaseURL, "/")
	if baseURL == "" {
		return SearchPlanResponse{}, fmt.Errorf("llm base url is required")
	}
	variablePayload, err := json.Marshal(request.Variables)
	if err != nil {
		return SearchPlanResponse{}, fmt.Errorf("encode planner variables: %w", err)
	}
	body := map[string]any{
		"model": request.Model,
		"messages": []map[string]string{
			{"role": "system", "content": request.Prompt},
			{"role": "user", "content": string(variablePayload)},
		},
		"response_format": map[string]string{"type": "json_object"},
		"stream":          false,
		"thinking":        map[string]string{"type": "disabled"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return SearchPlanResponse{}, fmt.Errorf("encode llm planner request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return SearchPlanResponse{}, fmt.Errorf("build llm planner request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if request.Credential != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+request.Credential)
	}

	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return SearchPlanResponse{}, fmt.Errorf("call llm planner: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return SearchPlanResponse{}, fmt.Errorf("llm planner status %d", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return SearchPlanResponse{}, fmt.Errorf("read llm planner response: %w", err)
	}

	var payload openAIChatCompletionResponse
	if err := json.Unmarshal(content, &payload); err != nil {
		return SearchPlanResponse{}, fmt.Errorf("decode llm planner response: %w", err)
	}
	if len(payload.Choices) == 0 {
		return SearchPlanResponse{}, fmt.Errorf("llm planner response choices are empty")
	}
	var rawPlan map[string]any
	if err := json.Unmarshal([]byte(payload.Choices[0].Message.Content), &rawPlan); err != nil {
		return SearchPlanResponse{}, fmt.Errorf("decode llm query plan: %w", err)
	}
	if err := rejectForbiddenQueryPlanFields(rawPlan); err != nil {
		return SearchPlanResponse{}, err
	}
	var plan SearchQueryPlan
	if err := json.Unmarshal([]byte(payload.Choices[0].Message.Content), &plan); err != nil {
		return SearchPlanResponse{}, fmt.Errorf("decode llm query plan: %w", err)
	}
	return SearchPlanResponse{
		Plan: plan,
		Raw: map[string]any{
			"id":    payload.ID,
			"model": payload.Model,
		},
		Usage: payload.Usage,
	}, nil
}

func DecodeAndValidateSearchQueryPlan(data []byte, config AIWebResearchConfig) ([]SearchQueryConfig, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode search query plan: %w", err)
	}
	if err := rejectForbiddenQueryPlanFields(raw); err != nil {
		return nil, err
	}
	var plan SearchQueryPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("decode search query plan: %w", err)
	}
	return ValidateSearchQueryPlan(plan, config)
}

func ValidateSearchQueryPlan(plan SearchQueryPlan, config AIWebResearchConfig) ([]SearchQueryConfig, error) {
	if len(plan.Queries) == 0 {
		return nil, fmt.Errorf("queries are required")
	}
	maxQueries := connectorPositiveInt(config.PromptVariables["max_queries"])
	if maxQueries <= 0 {
		return nil, fmt.Errorf("prompt_variables.max_queries must be positive")
	}
	if len(plan.Queries) > maxQueries {
		return nil, fmt.Errorf("queries exceed max_queries")
	}
	allowedProviders := webSearchProviderSet(config.WebSearchPlan)
	queries := make([]SearchQueryConfig, 0, len(plan.Queries))
	for _, planned := range plan.Queries {
		if strings.TrimSpace(planned.Query) == "" {
			return nil, fmt.Errorf("query is required")
		}
		if planned.MaxResults <= 0 {
			return nil, fmt.Errorf("query max_results must be positive")
		}
		if planned.MaxResults > config.MaxResults {
			return nil, fmt.Errorf("query max_results exceeds limit")
		}
		for _, provider := range planned.Providers {
			if _, ok := allowedProviders[provider]; !ok {
				return nil, fmt.Errorf("provider %q is not allowed", provider)
			}
		}
		queries = append(queries, SearchQueryConfig{
			Query:      planned.Query,
			Region:     planned.Region,
			Topic:      planned.Topic,
			Providers:  planned.Providers,
			MaxResults: planned.MaxResults,
		})
	}
	return queries, nil
}

func rejectForbiddenQueryPlanFields(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if forbiddenQueryPlanFields[key] {
				return fmt.Errorf("forbidden query plan field %q", key)
			}
			if err := rejectForbiddenQueryPlanFields(item); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range typed {
			if err := rejectForbiddenQueryPlanFields(item); err != nil {
				return err
			}
		}
	}
	return nil
}

var forbiddenQueryPlanFields = map[string]bool{
	"items":            true,
	"title":            true,
	"content_text":     true,
	"source_url":       true,
	"event":            true,
	"events":           true,
	"tag":              true,
	"tags":             true,
	"entity":           true,
	"entities":         true,
	"entity_relation":  true,
	"entity_relations": true,
	"raw_document":     true,
	"raw_documents":    true,
}

func webSearchProviderSet(plan WebSearchPlan) map[string]struct{} {
	providers := make(map[string]struct{}, len(plan.Tools))
	for _, tool := range plan.Tools {
		providers[tool.Provider] = struct{}{}
	}
	return providers
}
