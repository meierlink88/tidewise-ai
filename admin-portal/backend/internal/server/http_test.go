package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/admin-portal/backend/api/admin/v1"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/data"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/service"
)

func TestHealthAndReadiness(t *testing.T) {
	assertServiceHealth(t, newHandler(testConfig(), nil, ""), conf.ServiceName)
}

func TestHandlerPublishesEmbeddedOpenAPIOutsideProduction(t *testing.T) {
	for _, environment := range []conf.Environment{conf.EnvLocal, conf.EnvUAT} {
		cfg := testConfig()
		cfg.App.Env = environment
		response := httptest.NewRecorder()
		newHandler(cfg, nil, "").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
		if response.Code != http.StatusOK || !strings.HasPrefix(response.Body.String(), "openapi: 3.0.4\n") {
			t.Fatalf("%s OpenAPI status=%d", environment, response.Code)
		}
	}
}

func TestHandlerDoesNotPublishOpenAPIInProduction(t *testing.T) {
	cfg := testConfig()
	cfg.App.Env = conf.EnvProd
	response := httptest.NewRecorder()
	newHandler(cfg, nil, "").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("production OpenAPI status=%d, want 404", response.Code)
	}
}

func TestNewHandlerComposesAdminBFFWithOneDataServiceCall(t *testing.T) {
	calls := 0
	client := &biz.FakeDataServiceRepo{ListEventsFunc: func(context.Context, biz.EventListQuery) (biz.EventPage, error) {
		calls++
		return biz.EventPage{Items: []biz.Event{}, Page: 1, PageSize: 50}, nil
	}}
	handler := newHandler(testConfig(), client, "admin-token")
	request := httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/events", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	request.Header.Set(data.RequestIDHeader, "admin-service-test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || calls != 1 {
		t.Fatalf("status/calls=%d/%d body=%s", response.Code, calls, response.Body.String())
	}
	var envelope struct {
		RequestID string `json:"request_id"`
		Result    any    `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.RequestID != "admin-service-test" || envelope.Result == nil {
		t.Fatalf("envelope=%#v err=%v", envelope, err)
	}
}

func TestRuntimeHealthIsDataOnlyAndRetiredRoutesAreAbsent(t *testing.T) {
	handler := newHandler(testConfig(), nil, "admin-token")
	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/runtime-health", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d", unauthorized.Code)
	}

	request := httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/runtime-health", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var envelope struct {
		Result v1.RuntimeHealth `json:"result"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &envelope) != nil ||
		len(envelope.Result.Services) != 1 || envelope.Result.Services[0].Key != "data" {
		t.Fatalf("runtime response=%d %s", response.Code, response.Body.String())
	}

	for _, path := range []string{"/agent-schedules/collector", "/monitoring/summary", "/model-providers", "/connectors"} {
		retired := httptest.NewRecorder()
		retiredRequest := httptest.NewRequest(http.MethodGet, v1.APIPrefix+path, nil)
		retiredRequest.Header.Set("Authorization", "Bearer admin-token")
		handler.ServeHTTP(retired, retiredRequest)
		if retired.Code != http.StatusNotFound {
			t.Fatalf("retired path %s status=%d", path, retired.Code)
		}
	}
}

func assertServiceHealth(t *testing.T, handler http.Handler, serviceName string) {
	t.Helper()
	for _, path := range []string{"/healthz", "/readyz"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), serviceName) {
			t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
}

func testConfig() conf.RuntimeConfig {
	return conf.RuntimeConfig{
		App:           conf.AppConfig{Name: conf.ServiceName, Env: conf.EnvLocal},
		Server:        conf.ServerConfig{Host: "127.0.0.1", Port: 18083, ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 10},
		AllowedOrigin: "http://127.0.0.1:5174",
	}
}

func newHandler(config conf.RuntimeConfig, dataService biz.DataServiceRepo, adminToken string) http.Handler {
	config.AdminToken = adminToken
	useCase := biz.NewService(dataService, biz.WithRuntimeHealthProvider(nil))
	applicationService := service.NewAdminService(useCase)
	return NewHTTPServer(config, applicationService, nil).Server.Handler
}
