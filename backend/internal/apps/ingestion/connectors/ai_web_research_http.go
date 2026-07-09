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

func postSearchJSON(ctx context.Context, client *http.Client, provider string, baseURL string, path string, credential string, body any, target any) error {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return fmt.Errorf("%s base url is required", provider)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode %s request: %w", provider, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build %s request: %w", provider, err)
	}
	request.Header.Set("Content-Type", "application/json")
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}

	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("call %s: %w", provider, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s status %d", provider, response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read %s response: %w", provider, err)
	}
	if err := json.Unmarshal(content, target); err != nil {
		return fmt.Errorf("decode %s response: %w", provider, err)
	}
	return nil
}
