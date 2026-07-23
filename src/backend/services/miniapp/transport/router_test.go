package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apihttp"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/usecase"
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

func TestPanicReturnsStructuredErrorWithRequestID(t *testing.T) {
	client := &dataclient.Fake{
		ListResearchThemesFunc: func(context.Context, dataclient.ResearchListQuery) (dataclient.ResearchThemePage, error) {
			panic("sensitive upstream failure")
		},
	}
	router := NewRouter(testConfig(), usecase.NewResearchService(client))
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

func testConfig() runtimeconfig.AppConfig {
	return runtimeconfig.AppConfig{Name: "tidewise-api", Env: runtimeconfig.EnvLocal}
}
