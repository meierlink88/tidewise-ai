package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

func getJSON(ctx context.Context, client HTTPClient, endpoint string, params url.Values, headers map[string]string, output any) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("create request: invalid endpoint")
	}
	query := parsed.Query()
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Accept", "application/json,*/*")
	request.Header.Set("User-Agent", "Mozilla/5.0 AppleWebKit/537.36 Chrome/124.0 Safari/537.36")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("request failed: status=%d", response.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, 20<<20))
	if err != nil {
		return fmt.Errorf("decode response")
	}
	trimmed := strings.TrimSpace(string(payload))
	if !strings.HasPrefix(trimmed, "{") {
		start := strings.Index(trimmed, "(")
		end := strings.LastIndex(trimmed, ")")
		if start < 0 || end <= start {
			return fmt.Errorf("decode response")
		}
		trimmed = trimmed[start+1 : end]
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("decode response")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode response")
	}
	return nil
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
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("Connector request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("request failed: status=%d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 20<<20)).Decode(output); err != nil {
		return errors.New("Connector response was invalid")
	}
	return nil
}
