package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

type runtimeHealthHTTPStub struct{ AgentRunHTTPServer }

func (runtimeHealthHTTPStub) GetRuntimeHealth(context.Context, *RuntimeHealthRequest) (*RuntimeHealth, error) {
	return &RuntimeHealth{CheckedAt: "2026-08-04T10:00:00Z", Services: []RuntimeHealthService{
		{Key: "agentrun", DisplayName: "AgentRun", Status: "ready", CheckedAt: "2026-08-04T10:00:00Z"},
		{Key: "qdrant", DisplayName: "Qdrant", Status: "degraded", CheckedAt: "2026-08-04T10:00:00Z", ReasonCode: "collection_unhealthy"},
	}}, nil
}

func TestRuntimeHealthRoutePublishesSanitizedDependencyStatus(t *testing.T) {
	server := kratoshttp.NewServer()
	RegisterAgentRunHTTPServer(server, runtimeHealthHTTPStub{})
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, AdminAPIPrefix+"/runtime-health", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	var result RuntimeHealth
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result.Services) != 2 || result.Services[0].Key != "agentrun" || result.Services[1].Key != "qdrant" {
		t.Fatalf("services = %#v", result.Services)
	}
}
