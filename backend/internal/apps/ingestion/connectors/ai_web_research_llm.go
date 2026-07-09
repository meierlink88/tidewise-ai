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

type LLMNormalizeRequest struct {
	Model         string
	Credential    string
	Prompt        string
	SearchResults []SearchResultCandidate
}

type LLMNormalizeResponse struct {
	Content map[string]any
	Raw     map[string]any
	Usage   map[string]any
}

type OpenAICompatibleNormalizer struct {
	Client  *http.Client
	BaseURL string
}

func (n OpenAICompatibleNormalizer) Normalize(ctx context.Context, request LLMNormalizeRequest) (LLMNormalizeResponse, error) {
	baseURL := strings.TrimRight(n.BaseURL, "/")
	if baseURL == "" {
		return LLMNormalizeResponse{}, fmt.Errorf("llm base url is required")
	}
	searchPayload, err := json.Marshal(request.SearchResults)
	if err != nil {
		return LLMNormalizeResponse{}, fmt.Errorf("encode search results: %w", err)
	}
	body := map[string]any{
		"model": request.Model,
		"messages": []map[string]string{
			{"role": "system", "content": request.Prompt},
			{"role": "user", "content": string(searchPayload)},
		},
		"response_format": map[string]string{"type": "json_object"},
		"stream":          false,
		"thinking":        map[string]string{"type": "disabled"},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return LLMNormalizeResponse{}, fmt.Errorf("encode llm request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return LLMNormalizeResponse{}, fmt.Errorf("build llm request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if request.Credential != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+request.Credential)
	}

	client := n.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return LLMNormalizeResponse{}, fmt.Errorf("call llm: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return LLMNormalizeResponse{}, fmt.Errorf("llm status %d", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return LLMNormalizeResponse{}, fmt.Errorf("read llm response: %w", err)
	}

	var payload openAIChatCompletionResponse
	if err := json.Unmarshal(content, &payload); err != nil {
		return LLMNormalizeResponse{}, fmt.Errorf("decode llm response: %w", err)
	}
	if len(payload.Choices) == 0 {
		return LLMNormalizeResponse{}, fmt.Errorf("llm response choices are empty")
	}
	var structured map[string]any
	if err := json.Unmarshal([]byte(payload.Choices[0].Message.Content), &structured); err != nil {
		return LLMNormalizeResponse{}, fmt.Errorf("decode llm content: %w", err)
	}
	return LLMNormalizeResponse{
		Content: structured,
		Raw: map[string]any{
			"id":    payload.ID,
			"model": payload.Model,
		},
		Usage: payload.Usage,
	}, nil
}

type openAIChatCompletionResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   map[string]any `json:"usage"`
}

type openAIChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}
