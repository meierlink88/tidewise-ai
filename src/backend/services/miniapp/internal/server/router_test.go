package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apihttp"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	usecase "github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/conf"
	dataclient "github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/data"
)

func TestHealthz(t *testing.T) {
	router := NewHTTPServer(testRuntimeConfig(), nil)

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
	router := NewHTTPServer(testRuntimeConfig(), nil)

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
	client := &dataclient.Fake{
		ListResearchThemesFunc: func(context.Context, dataclient.ResearchListQuery) (dataclient.ResearchThemePage, error) {
			panic("sensitive upstream failure")
		},
	}
	router := researchTestRouter(usecase.NewResearchService(client))
	request := httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/research/themes", nil)
	request.Header.Set(apihttp.RequestIDHeader, "miniapp-panic-request")
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
		NewHTTPServer(config, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
		if response.Code != test.wantStatus {
			t.Fatalf("%s OpenAPI status = %d, want %d", test.environment, response.Code, test.wantStatus)
		}
		if test.wantStatus == http.StatusOK && !strings.HasPrefix(response.Body.String(), "openapi: 3.0.4\n") {
			t.Fatalf("%s OpenAPI document does not declare 3.0.4", test.environment)
		}
	}
}

func TestLegacyMiniappRoutesRemainRemoved(t *testing.T) {
	for _, path := range []string{
		"/api/v1/miniapp/research/themes",
		"/api/miniapp/v1/research/anchors",
	} {
		response := httptest.NewRecorder()
		NewHTTPServer(testRuntimeConfig(), nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
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
