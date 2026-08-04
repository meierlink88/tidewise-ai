package qdranthealth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/runtimehealth"
)

const (
	qdrantHealthIntegrationOptIn = "TIDEWISE_QDRANT_HEALTH_INTEGRATION_TEST"
	qdrantIntegrationVersion     = "1.15.5"
)

func TestProbeAgainstFixedQdrantContainer(t *testing.T) {
	baseURL, apiKey := qdrantHealthIntegrationConfig(t)
	client := &http.Client{Timeout: 5 * time.Second}
	assertQdrantVersion(t, client, baseURL, apiKey)
	collections := []string{entitySemanticCollection, variableDefinitionSemanticCollection}
	for _, collection := range collections {
		assertQdrantCollectionAbsent(t, client, baseURL, apiKey, collection)
		createQdrantCollection(t, client, baseURL, apiKey, collection)
		collection := collection
		t.Cleanup(func() { deleteQdrantCollection(t, client, baseURL, apiKey, collection, true) })
	}

	t.Run("both collections green", func(t *testing.T) {
		probe, err := New(Config{BaseURL: baseURL, APIKey: apiKey, Timeout: 5 * time.Second, MaxResponseBytes: 4096})
		if err != nil {
			t.Fatal(err)
		}
		result := probe.Check(context.Background())
		if result.Status != runtimehealth.StatusReady || result.ReasonCode != runtimehealth.ReasonNone {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("missing collection", func(t *testing.T) {
		deleteQdrantCollection(t, client, baseURL, apiKey, variableDefinitionSemanticCollection, false)
		probe, err := New(Config{BaseURL: baseURL, APIKey: apiKey, Timeout: 5 * time.Second, MaxResponseBytes: 4096})
		if err != nil {
			t.Fatal(err)
		}
		result := probe.Check(context.Background())
		if result.Status != runtimehealth.StatusDegraded || result.ReasonCode != runtimehealth.ReasonCollectionUnhealthy {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		probe, err := New(Config{BaseURL: baseURL, APIKey: apiKey, Timeout: time.Nanosecond, MaxResponseBytes: 4096})
		if err != nil {
			t.Fatal(err)
		}
		result := probe.Check(context.Background())
		if result.Status != runtimehealth.StatusDown || result.ReasonCode != runtimehealth.ReasonTimeout {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		probe, err := New(Config{BaseURL: baseURL, APIKey: apiKey, Timeout: 5 * time.Second, MaxResponseBytes: 4096})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := probe.Check(ctx)
		if result.Status != runtimehealth.StatusDown || result.ReasonCode != runtimehealth.ReasonTimeout {
			t.Fatalf("result = %#v", result)
		}
	})
}

func qdrantHealthIntegrationConfig(t *testing.T) (string, string) {
	t.Helper()
	if os.Getenv(qdrantHealthIntegrationOptIn) != "1" {
		t.Skip("set TIDEWISE_QDRANT_HEALTH_INTEGRATION_TEST=1 to run against an empty disposable Qdrant v1.15.5 container")
	}
	baseURL := strings.TrimRight(os.Getenv("TIDEWISE_TEST_QDRANT_URL"), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || !isQdrantLoopbackHost(parsed.Hostname()) {
		t.Fatalf("health integration test requires a loopback Qdrant container URL, got %q", baseURL)
	}
	return baseURL, os.Getenv("TIDEWISE_TEST_QDRANT_API_KEY")
}

func assertQdrantVersion(t *testing.T, client *http.Client, baseURL, apiKey string) {
	t.Helper()
	response := qdrantIntegrationRequest(t, client, http.MethodGet, baseURL+"/", apiKey, nil)
	defer response.Body.Close()
	var info struct {
		Version string `json:"version"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&info) != nil || info.Version != qdrantIntegrationVersion {
		t.Fatalf("Qdrant integration container must be version %s (status=%d version=%q)", qdrantIntegrationVersion, response.StatusCode, info.Version)
	}
}

func assertQdrantCollectionAbsent(t *testing.T, client *http.Client, baseURL, apiKey, collection string) {
	t.Helper()
	response := qdrantIntegrationRequest(t, client, http.MethodGet, baseURL+"/collections/"+url.PathEscape(collection), apiKey, nil)
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("integration test requires an empty disposable Qdrant container; collection %q status=%d", collection, response.StatusCode)
	}
}

func createQdrantCollection(t *testing.T, client *http.Client, baseURL, apiKey, collection string) {
	t.Helper()
	body := bytes.NewBufferString(`{"vectors":{"size":4,"distance":"Cosine"}}`)
	response := qdrantIntegrationRequest(t, client, http.MethodPut, baseURL+"/collections/"+url.PathEscape(collection), apiKey, body)
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		content, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		t.Fatalf("create collection %q: status=%d body=%q", collection, response.StatusCode, content)
	}
}

func deleteQdrantCollection(t *testing.T, client *http.Client, baseURL, apiKey, collection string, allowMissing bool) {
	t.Helper()
	response := qdrantIntegrationRequest(t, client, http.MethodDelete, baseURL+"/collections/"+url.PathEscape(collection), apiKey, nil)
	defer response.Body.Close()
	if allowMissing && response.StatusCode == http.StatusNotFound {
		return
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Errorf("delete collection %q: status=%d", collection, response.StatusCode)
	}
}

func qdrantIntegrationRequest(t *testing.T, client *http.Client, method, endpoint, apiKey string, body io.Reader) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), method, endpoint, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		request.Header.Set("api-key", apiKey)
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(fmt.Errorf("Qdrant integration request failed: %w", err))
	}
	return response
}

func isQdrantLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
