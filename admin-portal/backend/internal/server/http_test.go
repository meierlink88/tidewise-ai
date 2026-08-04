package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/admin-portal/backend/api/admin/v1"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/data"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/service"
)

func TestHealthAndReadiness(t *testing.T) {
	assertServiceHealth(t, newHandler(testConfig(), nil, nil, ""), conf.ServiceName)
}

func TestHandlerPublishesEmbeddedOpenAPIOutsideProduction(t *testing.T) {
	for _, environment := range []conf.Environment{conf.EnvLocal, conf.EnvUAT} {
		cfg := testConfig()
		cfg.App.Env = environment
		response := httptest.NewRecorder()
		newHandler(cfg, nil, nil, "").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
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
	cfg.App.Env = conf.EnvProd

	response := httptest.NewRecorder()
	newHandler(cfg, nil, nil, "").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("production openapi status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestNewHandlerComposesAdminBFFWithOneDataServiceCall(t *testing.T) {
	calls := 0
	client := &biz.FakeDataServiceRepo{ListRawDocumentsFunc: func(context.Context, biz.RawDocumentListQuery) (biz.RawDocumentPage, error) {
		calls++
		return biz.RawDocumentPage{Items: []biz.RawDocument{}, Page: 1, PageSize: 50}, nil
	}}
	handler := newHandler(testConfig(), client, nil, "admin-token")
	assertServiceHealth(t, handler, conf.ServiceName)

	request := httptest.NewRequest(http.MethodGet, "/api/admin/v1/raw-documents", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	request.Header.Set(data.RequestIDHeader, "admin-service-test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || calls != 1 {
		t.Fatalf("status/calls = %d/%d, body=%s", response.Code, calls, response.Body.String())
	}
	var envelope struct {
		RequestID string `json:"request_id"`
		Result    any    `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.RequestID != "admin-service-test" || envelope.Result == nil {
		t.Fatalf("business envelope = %#v, err=%v", envelope, err)
	}

	legacy := httptest.NewRecorder()
	handler.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/admin/raw-documents", nil))
	if legacy.Code != http.StatusNotFound {
		t.Fatalf("legacy path status = %d, want %d", legacy.Code, http.StatusNotFound)
	}
}

func TestRuntimeHealthRequiresAdminAuthAndReturnsSafePartialHTTP200(t *testing.T) {
	handler := newHandler(testConfig(), nil, nil, "browser-admin-token")

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/admin/v1/runtime-health", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", unauthorized.Code)
	}

	response := performAdminRequest(t, handler, http.MethodGet, "/api/admin/v1/runtime-health", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("partial runtime health status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Result v1.RuntimeHealth `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	want := []string{"data", "agentrun", "qdrant", "neo4j"}
	if envelope.Result.Status != "degraded" || len(envelope.Result.Services) != len(want) {
		t.Fatalf("runtime health = %#v", envelope.Result)
	}
	for index, key := range want {
		item := envelope.Result.Services[index]
		if item.Key != key || item.Status != "unknown" || item.ReasonCode != "not_ready" {
			t.Fatalf("runtime service[%d] = %#v", index, item)
		}
	}
	if strings.Contains(strings.ToLower(response.Body.String()), "token") || strings.Contains(strings.ToLower(response.Body.String()), "http://") {
		t.Fatalf("runtime response leaked internal detail: %s", response.Body.String())
	}
}

func TestNewHandlerProxiesCollectorScheduleWithServiceIdentity(t *testing.T) {
	var gotAuthorization string
	var gotRequestID string
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotRequestID = request.Header.Get("X-Request-ID")
		if request.Method != http.MethodGet || request.URL.Path != "/api/admin/v1/agent-schedules/collector" {
			t.Fatalf("upstream request = %s %s", request.Method, request.URL.Path)
		}
		response.Header().Set("Content-Type", "application/json")
		writeAgentRunSuccess(response, `{
			"schedule_id":"11111111-1111-4111-8111-111111111111",
			"agent_key":"collector",
			"agent_version":"collector.v1",
			"schedule_type":"daily",
			"daily_times":["08:30","12:30","18:30"],
			"input":{"prompt":"采集全球政经事实"},
			"enabled":true,
			"last_triggered_at":"2026-07-24T04:30:00Z",
			"next_run_at":"2026-07-24T10:30:00Z",
			"created_at":"2026-07-20T01:00:00Z",
			"updated_at":"2026-07-24T04:30:00Z"
		}`)
	}))
	defer upstream.Close()

	agentClient, err := data.NewAgentRunHTTPClient(data.AgentRunHTTPConfig{
		BaseURL: upstream.URL, ServiceToken: "agentrun-service-token", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandler(testConfig(), nil, agentClient, "browser-admin-token")
	request := httptest.NewRequest(http.MethodGet, "/api/admin/v1/agent-schedules/collector", nil)
	request.Header.Set("Authorization", "Bearer browser-admin-token")
	request.Header.Set("X-Request-ID", "admin-schedule-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if gotAuthorization != "Bearer agentrun-service-token" || gotRequestID != "admin-schedule-request" {
		t.Fatalf("upstream identity/request id = %q/%q", gotAuthorization, gotRequestID)
	}
	if strings.Contains(response.Body.String(), "agentrun-service-token") || strings.Contains(response.Body.String(), "browser-admin-token") {
		t.Fatalf("response leaked token: %s", response.Body.String())
	}
	var envelope struct {
		RequestID string           `json:"request_id"`
		Result    v1.AgentSchedule `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.RequestID != "admin-schedule-request" || envelope.Result.AgentKey != "collector" || !envelope.Result.Enabled || len(envelope.Result.DailyTimes) != 3 {
		t.Fatalf("response = %#v", envelope)
	}
}

func TestAgentRunProxyRetriesOneTransientReadFailureButNeverRetriesWrites(t *testing.T) {
	readCalls := 0
	writeCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodGet:
			readCalls++
			if readCalls == 1 {
				response.WriteHeader(http.StatusBadGateway)
				_, _ = response.Write([]byte(`{"error_code":"temporary_failure"}`))
				return
			}
			writeAgentRunSuccess(response, `{
				"schedule_id":"11111111-1111-4111-8111-111111111111",
				"agent_key":"collector",
				"agent_version":"collector.v1",
				"schedule_type":"daily",
				"daily_times":["08:30"],
				"input":{"prompt":"采集事实"},
				"enabled":false,
				"created_at":"2026-07-20T01:00:00Z",
				"updated_at":"2026-07-24T04:30:00Z"
			}`)
		case http.MethodPatch:
			writeCalls++
			response.WriteHeader(http.StatusBadGateway)
			_, _ = response.Write([]byte(`{"error_code":"temporary_failure"}`))
		default:
			t.Fatalf("unexpected upstream request: %s", request.Method)
		}
	}))
	defer upstream.Close()

	agentClient, err := data.NewAgentRunHTTPClient(data.AgentRunHTTPConfig{
		BaseURL: upstream.URL, ServiceToken: "agentrun-service-token", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandler(testConfig(), nil, agentClient, "browser-admin-token")
	readResponse := performAdminRequest(t, handler, http.MethodGet, "/api/admin/v1/agent-schedules/collector", nil)
	if readResponse.Code != http.StatusOK || readCalls != 2 {
		t.Fatalf("read response/calls = %d/%d, body=%s", readResponse.Code, readCalls, readResponse.Body.String())
	}

	writeResponse := performAdminRequest(t, handler, http.MethodPatch, "/api/admin/v1/agent-schedules/collector", map[string]any{"enabled": true})
	if writeResponse.Code != http.StatusServiceUnavailable || writeCalls != 1 {
		t.Fatalf("write response/calls = %d/%d, body=%s", writeResponse.Code, writeCalls, writeResponse.Body.String())
	}
}

func TestAdminScheduleSavePreservesEnabledAndStopUsesIndependentPatch(t *testing.T) {
	var upstreamPatches []map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodGet:
			writeAgentRunSuccess(response, `{
				"schedule_id":"11111111-1111-4111-8111-111111111111",
				"agent_key":"collector",
				"agent_version":"collector.v1",
				"schedule_type":"daily",
				"daily_times":["08:30"],
				"input":{"prompt":"旧 Prompt"},
				"enabled":true,
				"created_at":"2026-07-20T01:00:00Z",
				"updated_at":"2026-07-24T04:30:00Z"
			}`)
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			upstreamPatches = append(upstreamPatches, body)
			scheduleType := "cron"
			enabled := false
			if value, exists := body["enabled"]; exists {
				enabled, _ = value.(bool)
				scheduleType = "cron"
			}
			writeAgentRunSuccess(response, `{
				"schedule_id":"11111111-1111-4111-8111-111111111111",
				"agent_key":"collector",
				"agent_version":"collector.v1",
				"schedule_type":"`+scheduleType+`",
				"cron_expression":"0 * * * *",
				"input":{"prompt":"每小时采集一次"},
				"enabled":`+strconv.FormatBool(enabled)+`,
				"created_at":"2026-07-20T01:00:00Z",
				"updated_at":"2026-07-24T05:00:00Z"
			}`)
		default:
			t.Fatalf("unexpected upstream request: %s %s", request.Method, request.URL.Path)
		}
	}))
	defer upstream.Close()

	agentClient, err := data.NewAgentRunHTTPClient(data.AgentRunHTTPConfig{
		BaseURL: upstream.URL, ServiceToken: "agentrun-service-token", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandler(testConfig(), nil, agentClient, "browser-admin-token")
	save := performAdminRequest(t, handler, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"agent_version":   "collector.v1",
		"schedule_type":   "cron",
		"cron_expression": "0 * * * *",
		"input":           map[string]any{"prompt": "每小时采集一次"},
	})
	if save.Code != http.StatusOK {
		t.Fatalf("save status = %d, body=%s", save.Code, save.Body.String())
	}
	if !strings.Contains(save.Body.String(), `"enabled":false`) {
		t.Fatalf("save returned stale enabled state: %s", save.Body.String())
	}
	if len(upstreamPatches) != 1 || upstreamPatches[0]["schedule_type"] != "cron" {
		t.Fatalf("save patches = %#v", upstreamPatches)
	}
	if _, changedEnabled := upstreamPatches[0]["enabled"]; changedEnabled {
		t.Fatalf("save changed enabled state: %#v", upstreamPatches[0])
	}

	stop := performAdminRequest(t, handler, http.MethodPatch, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"enabled": false,
	})
	if stop.Code != http.StatusOK {
		t.Fatalf("stop status = %d, body=%s", stop.Code, stop.Body.String())
	}
	if len(upstreamPatches) != 2 || len(upstreamPatches[1]) != 1 || upstreamPatches[1]["enabled"] != false {
		t.Fatalf("stop patch = %#v", upstreamPatches)
	}
}

func TestAdminScheduleFirstSaveCreatesDisabledSchedule(t *testing.T) {
	var putBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodGet:
			response.WriteHeader(http.StatusNotFound)
			_, _ = response.Write([]byte(`{"error_code":"schedule_not_found","message":"not found"}`))
		case http.MethodPut:
			if err := json.NewDecoder(request.Body).Decode(&putBody); err != nil {
				t.Fatal(err)
			}
			writeAgentRunSuccess(response, `{
				"schedule_id":"11111111-1111-4111-8111-111111111111",
				"agent_key":"collector",
				"agent_version":"collector.v1",
				"schedule_type":"daily",
				"daily_times":["08:30"],
				"input":{"prompt":"首次配置"},
				"enabled":false,
				"created_at":"2026-07-24T05:00:00Z",
				"updated_at":"2026-07-24T05:00:00Z"
			}`)
		default:
			t.Fatalf("unexpected upstream request: %s", request.Method)
		}
	}))
	defer upstream.Close()
	agentClient, err := data.NewAgentRunHTTPClient(data.AgentRunHTTPConfig{
		BaseURL: upstream.URL, ServiceToken: "agentrun-service-token", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandler(testConfig(), nil, agentClient, "browser-admin-token")
	response := performAdminRequest(t, handler, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"agent_version": "collector.v1",
		"schedule_type": "daily",
		"daily_times":   []string{"08:30"},
		"input":         map[string]any{"prompt": "首次配置"},
	})
	if response.Code != http.StatusOK || putBody["enabled"] != false {
		t.Fatalf("response/put = %d %s / %#v", response.Code, response.Body.String(), putBody)
	}
}

func TestAdminAgentRunErrorsAreMappedWithoutUpstreamMessageLeak(t *testing.T) {
	for _, test := range []struct {
		upstreamStatus int
		wantStatus     int
		wantCode       string
	}{
		{upstreamStatus: http.StatusBadRequest, wantStatus: http.StatusBadRequest, wantCode: "AGENTRUN_INVALID_REQUEST"},
		{upstreamStatus: http.StatusUnauthorized, wantStatus: http.StatusServiceUnavailable, wantCode: "AGENTRUN_AUTHENTICATION_FAILED"},
		{upstreamStatus: http.StatusForbidden, wantStatus: http.StatusServiceUnavailable, wantCode: "AGENTRUN_AUTHENTICATION_FAILED"},
		{upstreamStatus: http.StatusNotFound, wantStatus: http.StatusNotFound, wantCode: "AGENTRUN_NOT_FOUND"},
		{upstreamStatus: http.StatusConflict, wantStatus: http.StatusConflict, wantCode: "AGENTRUN_CONFLICT"},
		{upstreamStatus: http.StatusInternalServerError, wantStatus: http.StatusServiceUnavailable, wantCode: "AGENTRUN_UNAVAILABLE"},
	} {
		t.Run(strconv.Itoa(test.upstreamStatus), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.Header().Set("Content-Type", "application/json")
				response.WriteHeader(test.upstreamStatus)
				_, _ = response.Write([]byte(`{"error_code":"upstream_code","message":"secret upstream detail"}`))
			}))
			defer upstream.Close()
			agentClient, err := data.NewAgentRunHTTPClient(data.AgentRunHTTPConfig{
				BaseURL: upstream.URL, ServiceToken: "agentrun-service-token", Timeout: time.Second,
			})
			if err != nil {
				t.Fatal(err)
			}
			handler := newHandler(testConfig(), nil, agentClient, "browser-admin-token")
			response := performAdminRequest(t, handler, http.MethodGet, "/api/admin/v1/agent-schedules/collector", nil)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"`+test.wantCode+`"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "secret upstream detail") || strings.Contains(response.Body.String(), "upstream_code") {
				t.Fatalf("response leaked upstream detail: %s", response.Body.String())
			}
		})
	}
}

func TestAdminConfigurationListsAndUpdatesRegisteredTargetsWithoutCredentialLeaks(t *testing.T) {
	var modelPatch map[string]any
	var connectorPatch map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.Method + " " + request.URL.Path {
		case "GET /api/admin/v1/model-providers":
			writeAgentRunSuccess(response, `{"items":[{
				"provider_key":"deepseek",
				"base_url":"https://api.deepseek.com",
				"model":"deepseek-chat",
				"configured":true,
				"key_configured":true,
				"masked_key":"***9023"
			}]}`)
		case "GET /api/admin/v1/model-providers/deepseek":
			writeAgentRunSuccess(response, `{
				"provider_key":"deepseek",
				"base_url":"https://api.deepseek.com",
				"model":"deepseek-chat",
				"configured":true,
				"key_configured":true,
				"masked_key":"***9023"
			}`)
		case "PATCH /api/admin/v1/model-providers/deepseek":
			if err := json.NewDecoder(request.Body).Decode(&modelPatch); err != nil {
				t.Fatal(err)
			}
			writeAgentRunSuccess(response, `{
				"provider_key":"deepseek",
				"base_url":"https://api.deepseek.com/v1",
				"model":"deepseek-chat",
				"configured":true,
				"key_configured":true,
				"masked_key":"***1234"
			}`)
		case "GET /api/admin/v1/connectors":
			writeAgentRunSuccess(response, `{"items":[{
				"connector_key":"tavily",
				"base_url":"https://api.tavily.com",
				"configured":true,
				"key_configured":true,
				"masked_key":"***7788"
			}]}`)
		case "GET /api/admin/v1/connectors/tavily":
			writeAgentRunSuccess(response, `{
				"connector_key":"tavily",
				"base_url":"https://api.tavily.com",
				"configured":true,
				"key_configured":true,
				"masked_key":"***7788"
			}`)
		case "PATCH /api/admin/v1/connectors/tavily":
			if err := json.NewDecoder(request.Body).Decode(&connectorPatch); err != nil {
				t.Fatal(err)
			}
			writeAgentRunSuccess(response, `{
				"connector_key":"tavily",
				"base_url":"https://api.tavily.com",
				"configured":true,
				"key_configured":false
			}`)
		default:
			t.Fatalf("unexpected upstream request: %s %s", request.Method, request.URL.Path)
		}
	}))
	defer upstream.Close()
	agentClient, err := data.NewAgentRunHTTPClient(data.AgentRunHTTPConfig{
		BaseURL: upstream.URL, ServiceToken: "agentrun-service-token", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandler(testConfig(), nil, agentClient, "browser-admin-token")

	for _, path := range []string{
		"/api/admin/v1/model-providers",
		"/api/admin/v1/model-providers/deepseek",
		"/api/admin/v1/connectors",
		"/api/admin/v1/connectors/tavily",
	} {
		response := performAdminRequest(t, handler, http.MethodGet, path, nil)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"configured":true`) {
			t.Fatalf("%s response = %d %s", path, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "agentrun-service-token") {
			t.Fatalf("%s leaked service token", path)
		}
	}
	modelResponse := performAdminRequest(t, handler, http.MethodPatch, "/api/admin/v1/model-providers/deepseek", map[string]any{
		"base_url": "https://api.deepseek.com/v1",
		"api_key":  "new-model-key-1234",
	})
	if modelResponse.Code != http.StatusOK || modelPatch["api_key"] != "new-model-key-1234" {
		t.Fatalf("model response/patch = %d %s / %#v", modelResponse.Code, modelResponse.Body.String(), modelPatch)
	}
	connectorResponse := performAdminRequest(t, handler, http.MethodPatch, "/api/admin/v1/connectors/tavily", map[string]any{
		"api_key": "",
	})
	if connectorResponse.Code != http.StatusOK || connectorPatch["api_key"] != "" {
		t.Fatalf("connector response/patch = %d %s / %#v", connectorResponse.Code, connectorResponse.Body.String(), connectorPatch)
	}
}

func writeAgentRunSuccess(response http.ResponseWriter, result string) {
	_, _ = response.Write([]byte(`{"request_id":"agentrun-test","result":` + result + `}`))
}

func performAdminRequest(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, path, nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		request = httptest.NewRequest(method, path, strings.NewReader(string(payload)))
	}
	request.Header.Set("Authorization", "Bearer browser-admin-token")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
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
			Status      string            `json:"status"`
			Service     string            `json:"service"`
			Environment conf.Environment  `json:"environment"`
			Checks      map[string]string `json:"checks"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", test.path, err)
		}
		if body.Status != test.wantStatus || body.Service != service || body.Environment != conf.EnvLocal {
			t.Fatalf("%s response = %+v", test.path, body)
		}
		if test.path == "/readyz" && body.Checks["config"] != "ok" {
			t.Fatalf("%s checks = %v, want config=ok", test.path, body.Checks)
		}
	}
}

func testConfig() conf.RuntimeConfig {
	return conf.RuntimeConfig{
		App: conf.AppConfig{Name: conf.ServiceName, Env: conf.EnvLocal},
		Server: conf.ServerConfig{
			Host:                "127.0.0.1",
			Port:                18083,
			ReadTimeoutSeconds:  5,
			WriteTimeoutSeconds: 10,
		},
		AllowedOrigin: "http://127.0.0.1:5174",
	}
}

func newHandler(
	config conf.RuntimeConfig,
	dataService biz.DataServiceRepo,
	agentRun biz.AgentRunRepo,
	adminToken string,
) http.Handler {
	config.AdminToken = adminToken
	useCase := biz.NewService(dataService, agentRun)
	applicationService := service.NewAdminService(useCase)
	return NewHTTPServer(config, applicationService, nil).Server.Handler
}
