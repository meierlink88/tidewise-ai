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
