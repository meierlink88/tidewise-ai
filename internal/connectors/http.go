package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

func defaultClient() HTTPClient {
	return &http.Client{Timeout: 45 * time.Second}
}

func postJSON(ctx context.Context, client HTTPClient, endpoint string, headers map[string]string, body any, output any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("request failed: status=%d body=%s", response.StatusCode, string(body))
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 20<<20)).Decode(output); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
