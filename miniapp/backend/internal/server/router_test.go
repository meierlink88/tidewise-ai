package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/conf"
)

func TestHealthz(t *testing.T) {
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), nil)

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

	if body.Status != "ok" || body.Service != "tidewise-api" || body.Environment != conf.EnvLocal {
		t.Fatalf("unexpected health response: %+v", body)
	}
}

func TestReadyz(t *testing.T) {
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), nil)

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

func TestHealthRoutesRunKratosMiddlewareWithStableOperation(t *testing.T) {
	var logs bytes.Buffer
	router := NewHTTPServer(testRuntimeConfig(), slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})), nil)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(logs.String(), `"msg":"business request completed"`) ||
		!strings.Contains(logs.String(), `"operation":"miniapp.health"`) {
		t.Fatalf("health middleware log is incomplete: %s", logs.String())
	}
}

func TestOuterFilterRecoversPanicAndWritesSanitizedAccessLog(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := observabilityFilter(
		conf.AppConfig{Name: "miniapp", Env: conf.EnvLocal},
		logger,
	)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret binding failure")
	}))
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(requestIDHeader, "outer-panic-request")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), "secret binding failure") || strings.Contains(logs.String(), "secret binding failure") {
		t.Fatal("panic details leaked into response or access log")
	}
	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry); err != nil {
		t.Fatalf("decode access log: %v\n%s", err, logs.String())
	}
	for key, want := range map[string]any{
		"service":     "miniapp",
		"environment": "local",
		"operation":   "miniapp.health",
		"request_id":  "outer-panic-request",
		"status":      float64(http.StatusInternalServerError),
	} {
		if entry[key] != want {
			t.Fatalf("access log %s = %#v, want %#v", key, entry[key], want)
		}
	}
	if duration, ok := entry["duration_ms"].(float64); !ok || duration < 0 {
		t.Fatalf("access log duration_ms = %#v", entry["duration_ms"])
	}
}

func TestOpenAPIDocumentationVisibilityFollowsEnvironment(t *testing.T) {
	for _, test := range []struct {
		environment conf.Environment
		wantStatus  int
	}{
		{environment: conf.EnvLocal, wantStatus: http.StatusOK},
		{environment: conf.EnvUAT, wantStatus: http.StatusOK},
		{environment: conf.EnvProd, wantStatus: http.StatusNotFound},
	} {
		config := testRuntimeConfig()
		config.App.Env = test.environment
		response := httptest.NewRecorder()
		NewHTTPServer(config, testLogger(), nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
		if response.Code != test.wantStatus {
			t.Fatalf("%s OpenAPI status = %d, want %d", test.environment, response.Code, test.wantStatus)
		}
		if test.wantStatus == http.StatusOK && !strings.HasPrefix(response.Body.String(), "openapi: 3.0.4\n") {
			t.Fatalf("%s OpenAPI document does not declare 3.0.4", test.environment)
		}
	}
}

func TestDocumentationRoutesUseObservabilityFilter(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	request := httptest.NewRequest(http.MethodGet, "/docs", nil)
	request.Header.Set(requestIDHeader, "miniapp-docs-request")
	response := httptest.NewRecorder()

	NewHTTPServer(testRuntimeConfig(), logger, nil).ServeHTTP(response, request)

	if response.Code != http.StatusTemporaryRedirect || response.Header().Get(requestIDHeader) != "miniapp-docs-request" {
		t.Fatalf("docs status/request ID = %d/%q", response.Code, response.Header().Get(requestIDHeader))
	}
	if !strings.Contains(logs.String(), `"operation":"miniapp.docs"`) ||
		!strings.Contains(logs.String(), `"request_id":"miniapp-docs-request"`) ||
		!strings.Contains(logs.String(), `"status":307`) {
		t.Fatalf("docs access log is incomplete: %s", logs.String())
	}
}

func TestRetiredResearchRoutesReturnNotFound(t *testing.T) {
	for _, path := range []string{
		"/api/v1/miniapp/research/themes",
		"/api/miniapp/v1/research/anchors",
		"/api/miniapp/v1/research/themes",
		"/api/miniapp/v1/research/themes/theme-id",
		"/api/miniapp/v1/research/themes/theme-id/reasoning-trees",
		"/api/miniapp/v1/research/themes/theme-id/reasoning-trees/tree-id",
	} {
		response := httptest.NewRecorder()
		NewHTTPServer(testRuntimeConfig(), testLogger(), nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}

func testRuntimeConfig() conf.RuntimeConfig {
	return conf.RuntimeConfig{
		App: conf.AppConfig{Name: "tidewise-api", Env: conf.EnvLocal},
		Server: conf.ServerConfig{
			Host: "127.0.0.1", Port: 18082, ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 10,
		},
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
