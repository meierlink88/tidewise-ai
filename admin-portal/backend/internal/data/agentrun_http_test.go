package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

func TestAgentRunHTTPClientDecodesExecutionSuccessEnvelope(t *testing.T) {
	requests := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests <- request.Clone(request.Context())
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"request_id": "agentrun-test-1",
			"result": {
				"items": [{
					"execution_id": "execution-test-1",
					"agent_key": "collector",
					"agent_version": "collector.v1",
					"trigger_source": "api",
					"status": "succeeded",
					"created_at": "2026-07-26T01:02:03Z",
					"triggered_at": "2026-07-26T01:02:03Z"
				}],
				"page": 1,
				"page_size": 20,
				"total_items": 1,
				"total_pages": 1
			}
		}`))
	}))
	defer server.Close()

	client, err := NewAgentRunHTTPClient(AgentRunHTTPConfig{
		BaseURL:         server.URL,
		ServiceToken:    "agentrun-admin-token",
		Timeout:         time.Second,
		MaxReadAttempts: 1,
		HTTPClient:      server.Client(),
	})
	if err != nil {
		t.Fatalf("create AgentRun HTTP client: %v", err)
	}
	defer client.Close()

	page, callErr := client.ListAgentExecutions(context.Background(), biz.AgentExecutionQuery{
		AgentKey: "collector",
		Page:     1,
		PageSize: 20,
	})
	request := <-requests

	if request.Method != http.MethodGet {
		t.Errorf("request method = %q, want GET", request.Method)
	}
	if request.URL.Path != "/api/admin/v1/agent-executions" {
		t.Errorf("request path = %q, want /api/admin/v1/agent-executions", request.URL.Path)
	}
	if got := request.URL.Query().Get("agent_key"); got != "collector" {
		t.Errorf("agent_key = %q, want collector", got)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer agentrun-admin-token" {
		t.Errorf("Authorization = %q, want Bearer agentrun-admin-token", got)
	}
	if callErr != nil {
		t.Fatalf("list AgentRun executions: %v", callErr)
	}
	if page.Page != 1 || page.PageSize != 20 || page.TotalItems != 1 || page.TotalPages != 1 {
		t.Fatalf("execution page metadata = %+v, want page 1 with one item", page)
	}
	if len(page.Items) != 1 {
		t.Fatalf("execution count = %d, want 1", len(page.Items))
	}
	if got := page.Items[0]; got.ID != "execution-test-1" || got.AgentKey != "collector" ||
		got.AgentVersion != "collector.v1" || got.Status != "succeeded" {
		t.Errorf("execution = %+v, want decoded collector execution", got)
	}
}

func TestAgentRunHTTPClientDecodesSafeAgentStatusList(t *testing.T) {
	requests := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests <- request.Clone(request.Context())
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"request_id": "agentrun-status-test",
			"result": {
				"items": [{
					"agent_key": "event-semantic-enricher",
					"display_name": "Event Semantic Enricher",
					"current_version": "event-semantic-enricher.v1",
					"is_working": true,
					"current_execution_status": "running",
					"updated_at": "2026-07-29T08:30:00Z"
				}]
			}
		}`))
	}))
	defer server.Close()

	client, err := NewAgentRunHTTPClient(AgentRunHTTPConfig{
		BaseURL: server.URL, ServiceToken: "agentrun-admin-token",
		Timeout: time.Second, MaxReadAttempts: 1, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	statuses, callErr := client.ListAgentStatuses(context.Background())
	request := <-requests
	if callErr != nil {
		t.Fatal(callErr)
	}
	if request.URL.Path != "/api/admin/v1/agent-statuses" {
		t.Fatalf("path = %q", request.URL.Path)
	}
	if len(statuses) != 1 || statuses[0].AgentKey != "event-semantic-enricher" ||
		!statuses[0].IsWorking || statuses[0].CurrentExecutionStatus != "running" {
		t.Fatalf("statuses = %#v", statuses)
	}
}

func TestAgentRunHTTPClientRejectsInconsistentAgentStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"request_id": "agentrun-status-invalid",
			"result": {"items": [{
				"agent_key": "collector",
				"display_name": "Collector",
				"current_version": "collector.v1",
				"is_working": false,
				"current_execution_status": "running",
				"updated_at": "2026-07-29T08:30:00Z"
			}]}
		}`))
	}))
	defer server.Close()
	client, err := NewAgentRunHTTPClient(AgentRunHTTPConfig{
		BaseURL: server.URL, ServiceToken: "token", Timeout: time.Second,
		MaxReadAttempts: 1, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if _, callErr := client.ListAgentStatuses(context.Background()); callErr == nil {
		t.Fatal("expected inconsistent status to be rejected")
	}
}

func TestAgentRunHTTPClientAcceptsRegisteredButUnconfiguredTargets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/admin/v1/model-providers":
			_, _ = response.Write([]byte(`{
				"request_id": "agentrun-test-models",
				"result": {
					"items": [{
						"provider_key": "deepseek",
						"base_url": "",
						"model": "",
						"configured": false,
						"key_configured": false
					}]
				}
			}`))
		case "/api/admin/v1/connectors":
			_, _ = response.Write([]byte(`{
				"request_id": "agentrun-test-connectors",
				"result": {
					"items": [{
						"connector_key": "tavily",
						"base_url": "",
						"configured": false,
						"key_configured": false
					}]
				}
			}`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	client, err := NewAgentRunHTTPClient(AgentRunHTTPConfig{
		BaseURL:         server.URL,
		ServiceToken:    "agentrun-admin-token",
		Timeout:         time.Second,
		MaxReadAttempts: 1,
		HTTPClient:      server.Client(),
	})
	if err != nil {
		t.Fatalf("create AgentRun HTTP client: %v", err)
	}
	defer client.Close()

	models, modelErr := client.ListModelProviders(context.Background())
	if modelErr != nil {
		t.Fatalf("list unconfigured Model Providers: %v", modelErr)
	}
	if len(models) != 1 || models[0].ProviderKey != "deepseek" || models[0].Configured ||
		models[0].BaseURL != "" {
		t.Fatalf("Model Providers = %+v, want registered unconfigured deepseek", models)
	}

	connectors, connectorErr := client.ListConnectors(context.Background())
	if connectorErr != nil {
		t.Fatalf("list unconfigured Connectors: %v", connectorErr)
	}
	if len(connectors) != 1 || connectors[0].ConnectorKey != "tavily" || connectors[0].Configured ||
		connectors[0].BaseURL != "" {
		t.Fatalf("Connectors = %+v, want registered unconfigured tavily", connectors)
	}
}
