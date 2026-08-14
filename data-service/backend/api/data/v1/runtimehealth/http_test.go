package runtimehealth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

type httpStub struct{}

func (httpStub) GetRuntimeHealth(context.Context, *Request) (*v1.Response[Result], error) {
	return &v1.Response[Result]{Status: http.StatusOK, Result: Result{
		CheckedAt: "2026-08-04T10:00:00Z",
		Services: []ServiceStatus{
			{Key: "data", DisplayName: "Data Service", Status: "ready", CheckedAt: "2026-08-04T10:00:00Z", LatencyMS: int64Pointer(4)},
		},
	}}, nil
}

func TestRuntimeHealthRoutePublishesDataStatus(t *testing.T) {
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, httpStub{})

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/runtime-health", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var body Result
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Services) != 1 || body.Services[0].Key != "data" {
		t.Fatalf("services = %#v", body.Services)
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
