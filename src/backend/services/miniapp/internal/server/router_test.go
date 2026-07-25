package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apihttp"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	usecase "github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/conf"
	dataclient "github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/data"
	appservice "github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/service"
)

func TestHealthz(t *testing.T) {
	router := NewHTTPServer(testRuntimeConfig(), nil, testLogger())

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
	router := NewHTTPServer(testRuntimeConfig(), nil, testLogger())

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

func TestPanicReturnsStructuredErrorWithRequestID(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	client := &dataclient.Fake{
		ListResearchThemesFunc: func(context.Context, dataclient.ResearchListQuery) (dataclient.ResearchThemePage, error) {
			panic("sensitive upstream failure")
		},
	}
	router := NewHTTPServer(
		testRuntimeConfig(),
		appservice.NewResearchService(usecase.NewResearchService(client)),
		logger,
	)
	request := httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/research/themes?cursor=sensitive-request-value", nil)
	request.Header.Set(apihttp.RequestIDHeader, "miniapp-panic-request")
	request.Header.Set("Authorization", "Bearer sensitive-token-value")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	var body apihttp.ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.RequestID != "miniapp-panic-request" || response.Header().Get(apihttp.RequestIDHeader) != body.RequestID {
		t.Fatalf("request IDs = %q/%q", body.RequestID, response.Header().Get(apihttp.RequestIDHeader))
	}
	if body.Error.Code != "INTERNAL_ERROR" || body.Error.Message != "internal server error" || body.Error.Details == nil {
		t.Fatalf("error = %#v", body.Error)
	}
	if strings.Contains(response.Body.String(), "sensitive upstream failure") {
		t.Fatalf("panic detail leaked: %s", response.Body.String())
	}
	for _, secret := range []string{"sensitive upstream failure", "sensitive-request-value", "sensitive-token-value", "runtime/debug"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("recovery log leaked %q: %s", secret, logs.String())
		}
	}
	if !strings.Contains(logs.String(), `"operation":"miniapp.research.listThemes"`) ||
		!strings.Contains(logs.String(), `"status":500`) {
		t.Fatalf("sanitized panic access log is incomplete: %s", logs.String())
	}
}

func TestServerDoesNotImposeRequestContextDeadline(t *testing.T) {
	var hasDeadline bool
	client := &dataclient.Fake{
		ListResearchThemesFunc: func(ctx context.Context, _ dataclient.ResearchListQuery) (dataclient.ResearchThemePage, error) {
			_, hasDeadline = ctx.Deadline()
			return dataclient.ResearchThemePage{}, nil
		},
	}
	response := httptest.NewRecorder()
	researchTestRouter(usecase.NewResearchService(client)).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/research/themes", nil),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if hasDeadline {
		t.Fatal("Kratos server added a request context deadline; Biz/Data must own the request budget")
	}
}

func TestOuterFilterRecoversPanicAndWritesSanitizedAccessLog(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := observabilityFilter(
		runtimeconfig.AppConfig{Name: "miniapp", Env: runtimeconfig.EnvLocal},
		logger,
	)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret binding failure")
	}))
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(apihttp.RequestIDHeader, "outer-panic-request")
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
		environment runtimeconfig.Environment
		wantStatus  int
	}{
		{environment: runtimeconfig.EnvLocal, wantStatus: http.StatusOK},
		{environment: runtimeconfig.EnvUAT, wantStatus: http.StatusOK},
		{environment: runtimeconfig.EnvProd, wantStatus: http.StatusNotFound},
	} {
		config := testRuntimeConfig()
		config.App.Env = test.environment
		response := httptest.NewRecorder()
		NewHTTPServer(config, nil, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
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
	request.Header.Set(apihttp.RequestIDHeader, "miniapp-docs-request")
	response := httptest.NewRecorder()

	NewHTTPServer(testRuntimeConfig(), nil, logger).ServeHTTP(response, request)

	if response.Code != http.StatusTemporaryRedirect || response.Header().Get(apihttp.RequestIDHeader) != "miniapp-docs-request" {
		t.Fatalf("docs status/request ID = %d/%q", response.Code, response.Header().Get(apihttp.RequestIDHeader))
	}
	if !strings.Contains(logs.String(), `"operation":"miniapp.docs"`) ||
		!strings.Contains(logs.String(), `"request_id":"miniapp-docs-request"`) ||
		!strings.Contains(logs.String(), `"status":307`) {
		t.Fatalf("docs access log is incomplete: %s", logs.String())
	}
}

func TestLegacyMiniappRoutesRemainRemoved(t *testing.T) {
	for _, path := range []string{
		"/api/v1/miniapp/research/themes",
		"/api/miniapp/v1/research/anchors",
	} {
		response := httptest.NewRecorder()
		NewHTTPServer(testRuntimeConfig(), nil, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}

func testRuntimeConfig() conf.RuntimeConfig {
	return conf.RuntimeConfig{
		App: runtimeconfig.AppConfig{Name: "tidewise-api", Env: runtimeconfig.EnvLocal},
		Server: runtimeconfig.ServerConfig{
			Host: "127.0.0.1", Port: 18082, ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 10,
		},
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
