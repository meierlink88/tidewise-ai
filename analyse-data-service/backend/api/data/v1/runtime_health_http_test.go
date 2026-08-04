package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

type runtimeHealthHTTPStub struct {
	testDataHTTPServer
}

func (runtimeHealthHTTPStub) GetRuntimeHealth(context.Context, *RuntimeHealthRequest) (*Response[RuntimeHealth], error) {
	return &Response[RuntimeHealth]{Status: http.StatusOK, Result: RuntimeHealth{
		CheckedAt: "2026-08-04T10:00:00Z",
		Services: []RuntimeHealthService{
			{Key: "data", DisplayName: "Data Service", Status: "ready", CheckedAt: "2026-08-04T10:00:00Z", LatencyMS: int64Pointer(4)},
			{Key: "neo4j", DisplayName: "Neo4j", Status: "degraded", CheckedAt: "2026-08-04T10:00:00Z", ReasonCode: "authentication_failed"},
		},
	}}, nil
}

func TestRuntimeHealthRoutePublishesSanitizedDependencyStatus(t *testing.T) {
	server := kratoshttp.NewServer()
	RegisterDataHTTPServer(server, runtimeHealthHTTPStub{})

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, APIPrefix+"/runtime-health", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var body RuntimeHealth
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Services) != 2 || body.Services[0].Key != "data" || body.Services[1].Key != "neo4j" {
		t.Fatalf("services = %#v", body.Services)
	}
	if body.Services[1].ReasonCode != "authentication_failed" {
		t.Fatalf("Neo4j reason_code = %q", body.Services[1].ReasonCode)
	}
	if string(response.Body.Bytes()) == "" || containsUnsafeRuntimeDetail(response.Body.String()) {
		t.Fatalf("runtime health leaked unsafe dependency detail: %s", response.Body.String())
	}
}

func int64Pointer(value int64) *int64 { return &value }

func containsUnsafeRuntimeDetail(value string) bool {
	for _, unsafe := range []string{"password", "bolt://", "query", "stack"} {
		if strings.Contains(strings.ToLower(value), unsafe) {
			return true
		}
	}
	return false
}
