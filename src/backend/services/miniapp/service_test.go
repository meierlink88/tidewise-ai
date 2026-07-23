package miniapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	miniappconfig "github.com/meierlink88/tidewise-ai/backend/services/miniapp/config"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
)

func TestHealthAndReadiness(t *testing.T) {
	assertServiceHealth(t, NewHandler(testConfig()), ServiceName)
}

func TestHandlerPublishesEmbeddedOpenAPIOutsideProduction(t *testing.T) {
	for _, environment := range []runtimeconfig.Environment{runtimeconfig.EnvLocal, runtimeconfig.EnvUAT} {
		cfg := testConfig()
		cfg.App.Env = environment
		response := httptest.NewRecorder()
		NewHandler(cfg).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s openapi status = %d, want %d", environment, response.Code, http.StatusOK)
		}
		if !strings.HasPrefix(response.Body.String(), "openapi: 3.0.4\n") {
			t.Fatalf("%s openapi document does not declare 3.0.4", environment)
		}
	}
}

func TestHandlerDoesNotPublishOpenAPIInProduction(t *testing.T) {
	cfg := testConfig()
	cfg.App.Env = runtimeconfig.EnvProd

	response := httptest.NewRecorder()
	NewHandler(cfg).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("production openapi status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestNewServerPreservesCompatibilityHandler(t *testing.T) {
	legacy := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	server := NewServer(testConfig(), legacy)
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/research/home", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("compatibility status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestNewHandlerComposesResearchBFFWithOneDataServiceCall(t *testing.T) {
	calls := 0
	client := &dataclient.Fake{ListResearchThemesFunc: func(context.Context, dataclient.ResearchListQuery) (dataclient.ResearchThemePage, error) {
		calls++
		return dataclient.ResearchThemePage{Items: []dataclient.ResearchTheme{}}, nil
	}}
	handler := NewHandler(testConfig(), client)
	assertServiceHealth(t, handler, ServiceName)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/research/themes", nil)
	request.Header.Set(dataclient.RequestIDHeader, "miniapp-service-test")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || calls != 1 {
		t.Fatalf("status/calls = %d/%d, body=%s", response.Code, calls, response.Body.String())
	}
	var envelope struct {
		RequestID string `json:"request_id"`
		Result    any    `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.RequestID != "miniapp-service-test" || envelope.Result == nil {
		t.Fatalf("business envelope = %#v, err=%v", envelope, err)
	}

	legacy := httptest.NewRecorder()
	handler.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/api/v1/miniapp/research/themes", nil))
	if legacy.Code != http.StatusNotFound {
		t.Fatalf("legacy path status = %d, want %d", legacy.Code, http.StatusNotFound)
	}
}

func assertServiceHealth(t *testing.T, handler http.Handler, service string) {
	t.Helper()
	for _, test := range []struct {
		path       string
		wantStatus string
	}{
		{path: "/healthz", wantStatus: "ok"},
		{path: "/readyz", wantStatus: "ready"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", test.path, response.Code, http.StatusOK)
		}
		var body struct {
			Status      string                    `json:"status"`
			Service     string                    `json:"service"`
			Environment runtimeconfig.Environment `json:"environment"`
			Checks      map[string]string         `json:"checks"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", test.path, err)
		}
		if body.Status != test.wantStatus || body.Service != service || body.Environment != runtimeconfig.EnvLocal {
			t.Fatalf("%s response = %+v", test.path, body)
		}
		if test.path == "/readyz" && body.Checks["config"] != "ok" {
			t.Fatalf("%s checks = %v, want config=ok", test.path, body.Checks)
		}
	}
}

func testConfig() miniappconfig.RuntimeConfig {
	return miniappconfig.RuntimeConfig{
		App: runtimeconfig.AppConfig{Env: runtimeconfig.EnvLocal},
		Server: runtimeconfig.ServerConfig{
			Host:                "127.0.0.1",
			Port:                18082,
			ReadTimeoutSeconds:  5,
			WriteTimeoutSeconds: 10,
		},
	}
}
