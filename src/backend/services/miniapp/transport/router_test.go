package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
)

func TestHealthz(t *testing.T) {
	router := NewRouter(testConfig())

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}

	var body HealthResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Status != "ok" || body.Service != "tidewise-api" || body.Environment != runtimeconfig.EnvLocal {
		t.Fatalf("unexpected health response: %+v", body)
	}
}

func TestReadyz(t *testing.T) {
	router := NewRouter(testConfig())

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}

	var body ReadyResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Status != "ready" || body.Checks["config"] != "ok" {
		t.Fatalf("unexpected ready response: %+v", body)
	}
}

func testConfig() runtimeconfig.AppConfig {
	return runtimeconfig.AppConfig{Name: "tidewise-api", Env: runtimeconfig.EnvLocal}
}
