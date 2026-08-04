package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/agent-run/backend/api/agentrun/v1"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/conf"
)

func TestKratosHTTPServerAppliesAuthAndTidewiseEnvelope(t *testing.T) {
	t.Parallel()

	api := stubAPI{
		create: func(_ context.Context, request *v1.CreateCollectorSubmissionRequest) (*v1.CollectorSubmissionResult, error) {
			return &v1.CollectorSubmissionResult{
				Schema: "collector_run.v1", AgentKey: "collector",
				ExecutionID: "11111111-1111-4111-8111-111111111111",
				Status:      "queued", PromptSHA256: string(make([]byte, 64)),
				PromptBytes: len(request.Prompt),
			}, nil
		},
	}
	server := newTestServer(t, api)

	unauthorized := request(t, server, http.MethodPost, v1.CollectorRunsPath, `{"prompt":"x","unknown":true}`, nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d body=%s", unauthorized.Code, unauthorized.Body.String())
	}

	headers := map[string]string{
		"Authorization":   "Bearer service-token",
		"Idempotency-Key": "request-1",
		RequestIDHeader:   "request-id-1",
	}
	invalid := request(t, server, http.MethodPost, v1.CollectorRunsPath, `{"prompt":"x","unknown":true}`, headers)
	assertErrorEnvelope(t, invalid, http.StatusBadRequest, "INVALID_REQUEST", "request-id-1")

	response := request(t, server, http.MethodPost, v1.CollectorRunsPath, `{"prompt":"采集资讯"}`, headers)
	if response.Code != http.StatusAccepted {
		t.Fatalf("create status = %d body=%s", response.Code, response.Body.String())
	}
	var envelope struct {
		RequestID string `json:"request_id"`
		Result    struct {
			ExecutionID string `json:"execution_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.RequestID != "request-id-1" ||
		response.Header().Get(RequestIDHeader) != envelope.RequestID ||
		envelope.Result.ExecutionID == "" {
		t.Fatalf("success envelope = %#v header=%q", envelope, response.Header().Get(RequestIDHeader))
	}
	unsafe := request(t, server, http.MethodGet, "/does-not-exist", "", map[string]string{
		RequestIDHeader: "unsafe request id",
	})
	if got := unsafe.Header().Get(RequestIDHeader); !strings.HasPrefix(got, "agentrun-") {
		t.Fatalf("unsafe request ID was preserved: %q", got)
	}
}

func TestEventSemanticReanalysisRouteIsAuthenticatedAndReturnsWorkItem(t *testing.T) {
	t.Parallel()
	api := stubAPI{
		reanalyze: func(
			_ context.Context,
			request *v1.CreateEventSemanticReanalysisRequest,
		) (*v1.EventSemanticWorkItem, error) {
			return &v1.EventSemanticWorkItem{
				WorkItemID: "11111111-1111-4111-8111-111111111111",
				EventID:    request.EventID, SupersedesSubmissionID: request.SupersedesSubmissionID,
				Status: "pending", CreatedAt: time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC),
			}, nil
		},
	}
	server := newTestServer(t, api)
	body := `{"event_id":"22222222-2222-4222-8222-222222222222",` +
		`"supersedes_submission_id":"33333333-3333-4333-8333-333333333333",` +
		`"reason":"ontology_upgrade"}`
	unauthorized := request(
		t, server, http.MethodPost, v1.EventSemanticReanalysisPath, body, nil,
	)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.Code)
	}
	response := request(t, server, http.MethodPost, v1.EventSemanticReanalysisPath, body, map[string]string{
		"Authorization":   "Bearer service-token",
		"Idempotency-Key": "semantic-reanalysis-1",
		RequestIDHeader:   "semantic-request-1",
	})
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	var envelope struct {
		RequestID string                   `json:"request_id"`
		Result    v1.EventSemanticWorkItem `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.RequestID != "semantic-request-1" || envelope.Result.Status != "pending" {
		t.Fatalf("response = %#v", envelope)
	}
}

func TestKratosHTTPServerOwnsNotFoundMethodPanicHealthAndDocs(t *testing.T) {
	t.Parallel()

	api := stubAPI{
		create: func(context.Context, *v1.CreateCollectorSubmissionRequest) (*v1.CollectorSubmissionResult, error) {
			panic("must not leak")
		},
		get: func(context.Context, *v1.GetCollectorSubmissionRequest) (*v1.CollectorSubmissionResult, error) {
			return nil, v1.NewPublicError(http.StatusNotFound, "EXECUTION_NOT_FOUND", "Agent Execution was not found", nil)
		},
	}
	server := newTestServer(t, api)

	health := request(t, server, http.MethodGet, "/healthz", "", nil)
	if health.Code != http.StatusOK || bytes.Contains(health.Body.Bytes(), []byte(`"result"`)) {
		t.Fatalf("health response = %d %s", health.Code, health.Body.String())
	}
	docs := request(t, server, http.MethodGet, "/openapi.yaml", "", nil)
	if docs.Code != http.StatusOK || !bytes.Contains(docs.Body.Bytes(), []byte("openapi: 3.0.4")) {
		t.Fatalf("OpenAPI response = %d %s", docs.Code, docs.Body.String())
	}
	swagger := request(t, server, http.MethodGet, "/docs/", "", nil)
	if swagger.Code != http.StatusOK || !bytes.Contains(swagger.Body.Bytes(), []byte("Tidewise AI AgentRun API")) {
		t.Fatalf("Swagger response = %d %s", swagger.Code, swagger.Body.String())
	}
	notFound := request(t, server, http.MethodGet, "/does-not-exist", "", map[string]string{RequestIDHeader: "request-404"})
	assertErrorEnvelope(t, notFound, http.StatusNotFound, "NOT_FOUND", "request-404")
	method := request(t, server, http.MethodDelete, v1.CollectorRunsPath, "", map[string]string{RequestIDHeader: "request-405"})
	assertErrorEnvelope(t, method, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "request-405")
	panicResponse := request(t, server, http.MethodPost, v1.CollectorRunsPath, `{"prompt":"panic"}`, map[string]string{
		"Authorization": "Bearer service-token", "Idempotency-Key": "panic-1", RequestIDHeader: "request-panic",
	})
	assertErrorEnvelope(t, panicResponse, http.StatusInternalServerError, "INTERNAL_ERROR", "request-panic")
	if bytes.Contains(panicResponse.Body.Bytes(), []byte("must not leak")) {
		t.Fatalf("panic detail leaked: %s", panicResponse.Body.String())
	}
}

func TestKratosRuntimeRoutesMatchOpenAPI(t *testing.T) {
	t.Parallel()

	config := testConfig()
	runtime := NewHTTPServer(config, stubAPI{}, stubAPI{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	actual := make(map[string]struct{})
	if err := runtime.WalkRoute(func(route kratoshttp.RouteInfo) error {
		actual[route.Method+" "+route.Path] = struct{}{}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	document, err := openapi3.NewLoader().LoadFromData(v1.Document())
	if err != nil {
		t.Fatal(err)
	}
	expected := make(map[string]struct{})
	for path, item := range document.Paths.Map() {
		for method := range item.Operations() {
			expected[strings.ToUpper(method)+" "+path] = struct{}{}
		}
	}
	if _, exists := actual["GET /docs/{asset:.*}"]; exists {
		delete(actual, "GET /docs/{asset:.*}")
		actual["GET /docs/"] = struct{}{}
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("runtime routes = %#v, OpenAPI routes = %#v", actual, expected)
	}
}

func TestOperationalAndDocumentationRoutesExecuteKratosMiddleware(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	config := testConfig()
	api := stubAPI{
		create: func(context.Context, *v1.CreateCollectorSubmissionRequest) (*v1.CollectorSubmissionResult, error) {
			panic("must not leak")
		},
		get: func(context.Context, *v1.GetCollectorSubmissionRequest) (*v1.CollectorSubmissionResult, error) {
			return nil, v1.NewPublicError(http.StatusNotFound, "EXECUTION_NOT_FOUND", "Agent Execution was not found", nil)
		},
	}
	runtime := &kratosTestServer{handler: NewHTTPServer(config, api, api, logger)}
	for _, path := range []string{"/healthz", "/readyz", "/openapi.yaml", "/docs", "/docs/"} {
		response := request(t, runtime, http.MethodGet, path, "", nil)
		if response.Code < 200 || response.Code >= 400 {
			t.Fatalf("%s status = %d", path, response.Code)
		}
	}
	for _, operation := range []string{operationHealth, operationReady, operationOpenAPI, operationDocs} {
		if !strings.Contains(logs.String(), operation) {
			t.Fatalf("Kratos access middleware did not observe %s: %s", operation, logs.String())
		}
	}
	request(t, runtime, http.MethodGet, "/missing", "", nil)
	request(t, runtime, http.MethodGet, "/api/v1/collector/runs/11111111-1111-4111-8111-111111111111", "", map[string]string{
		"Authorization": "Bearer service-token",
	})
	request(t, runtime, http.MethodDelete, v1.CollectorRunsPath, "", nil)
	request(t, runtime, http.MethodPost, v1.CollectorRunsPath, `{"prompt":"panic"}`, map[string]string{
		"Authorization": "Bearer service-token", "Idempotency-Key": "panic-access-log",
	})
	for _, operation := range []string{"HTTP_NOT_FOUND", "HTTP_METHOD_NOT_ALLOWED", "HTTP_PANIC"} {
		if !strings.Contains(logs.String(), operation) {
			t.Fatalf("boundary access log did not observe %s: %s", operation, logs.String())
		}
	}
	if count := strings.Count(logs.String(), "HTTP_NOT_FOUND"); count != 1 {
		t.Fatalf("business 404 produced a duplicate boundary access log: count=%d logs=%s", count, logs.String())
	}
}

func newTestServer(t *testing.T, api stubAPI) *kratosTestServer {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := testConfig()
	return &kratosTestServer{handler: NewHTTPServer(config, api, api, logger)}
}

func testConfig() conf.Config {
	return conf.Config{
		App:     conf.AppConfig{Name: conf.ServiceName, Env: conf.EnvDev},
		Server:  conf.ServerConfig{Host: "127.0.0.1", Port: conf.ServicePort},
		Secrets: conf.SecretConfig{ServiceToken: "service-token"},
	}
}

type kratosTestServer struct {
	handler http.Handler
}

func request(
	t *testing.T,
	server *kratosTestServer,
	method string,
	path string,
	body string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()
	input := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		input.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		input.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, input)
	return response
}

func assertErrorEnvelope(t *testing.T, response *httptest.ResponseRecorder, status int, code, requestID string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d want %d body=%s", response.Code, status, response.Body.String())
	}
	var envelope struct {
		RequestID string `json:"request_id"`
		Error     struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.RequestID != requestID || response.Header().Get(RequestIDHeader) != requestID ||
		envelope.Error.Code != code || envelope.Error.Message == "" || envelope.Error.Details == nil {
		t.Fatalf("error envelope = %#v header=%q", envelope, response.Header().Get(RequestIDHeader))
	}
}

type stubAPI struct {
	create    func(context.Context, *v1.CreateCollectorSubmissionRequest) (*v1.CollectorSubmissionResult, error)
	get       func(context.Context, *v1.GetCollectorSubmissionRequest) (*v1.CollectorSubmissionResult, error)
	reanalyze func(context.Context, *v1.CreateEventSemanticReanalysisRequest) (*v1.EventSemanticWorkItem, error)
	readyErr  error
}

func (s stubAPI) Ready(context.Context) error { return s.readyErr }

func (s stubAPI) CreateCollectorRun(ctx context.Context, request *v1.CreateCollectorSubmissionRequest) (*v1.CollectorSubmissionResult, error) {
	if s.create != nil {
		return s.create(ctx, request)
	}
	return &v1.CollectorSubmissionResult{}, nil
}

func (s stubAPI) GetCollectorRun(ctx context.Context, request *v1.GetCollectorSubmissionRequest) (*v1.CollectorSubmissionResult, error) {
	if s.get != nil {
		return s.get(ctx, request)
	}
	return &v1.CollectorSubmissionResult{}, nil
}

func (s stubAPI) CreateEventSemanticReanalysis(
	ctx context.Context,
	request *v1.CreateEventSemanticReanalysisRequest,
) (*v1.EventSemanticWorkItem, error) {
	if s.reanalyze != nil {
		return s.reanalyze(ctx, request)
	}
	return &v1.EventSemanticWorkItem{}, nil
}

func (stubAPI) ListModelProviders(context.Context, *v1.ListModelProvidersRequest) (*v1.ModelProviderList, error) {
	return &v1.ModelProviderList{}, nil
}

func (stubAPI) GetModelProvider(context.Context, *v1.GetModelProviderRequest) (*v1.ModelProviderConfiguration, error) {
	return &v1.ModelProviderConfiguration{}, nil
}

func (stubAPI) PatchModelProvider(context.Context, *v1.PatchModelProviderRequest) (*v1.ModelProviderConfiguration, error) {
	return &v1.ModelProviderConfiguration{}, nil
}

func (stubAPI) ListConnectors(context.Context, *v1.ListConnectorsRequest) (*v1.ConnectorList, error) {
	return &v1.ConnectorList{}, nil
}

func (stubAPI) GetConnector(context.Context, *v1.GetConnectorRequest) (*v1.ConnectorConfiguration, error) {
	return &v1.ConnectorConfiguration{}, nil
}

func (stubAPI) PatchConnector(context.Context, *v1.PatchConnectorRequest) (*v1.ConnectorConfiguration, error) {
	return &v1.ConnectorConfiguration{}, nil
}

func (stubAPI) ListAgentSchedules(context.Context, *v1.ListAgentSchedulesRequest) (*v1.AgentScheduleList, error) {
	return &v1.AgentScheduleList{}, nil
}

func (stubAPI) GetAgentSchedule(context.Context, *v1.GetAgentScheduleRequest) (*v1.AgentSchedule, error) {
	return &v1.AgentSchedule{}, nil
}

func (stubAPI) PutAgentSchedule(context.Context, *v1.PutAgentScheduleRequest) (*v1.AgentSchedule, error) {
	return &v1.AgentSchedule{}, nil
}

func (stubAPI) PatchAgentSchedule(context.Context, *v1.PatchAgentScheduleRequest) (*v1.AgentSchedule, error) {
	return &v1.AgentSchedule{}, nil
}

func (stubAPI) ListAgentExecutions(context.Context, *v1.ListAgentExecutionsRequest) (*v1.AgentExecutionPage, error) {
	return &v1.AgentExecutionPage{}, nil
}

func (stubAPI) ListAgentStatuses(context.Context, *v1.ListAgentStatusesRequest) (*v1.AgentStatusList, error) {
	return &v1.AgentStatusList{}, nil
}

func (stubAPI) GetMonitoringSummary(context.Context, *v1.MonitoringSummaryRequest) (*v1.MonitoringSummary, error) {
	return &v1.MonitoringSummary{}, nil
}

func (stubAPI) ListCollectorMonitoring(context.Context, *v1.MonitoringListRequest) (*v1.CollectorMonitoringPage, error) {
	return &v1.CollectorMonitoringPage{}, nil
}

func (stubAPI) ListArtifactMonitoring(context.Context, *v1.MonitoringListRequest) (*v1.ArtifactMonitoringPage, error) {
	return &v1.ArtifactMonitoringPage{}, nil
}

func (stubAPI) ListSemanticMonitoring(context.Context, *v1.MonitoringListRequest) (*v1.SemanticMonitoringPage, error) {
	return &v1.SemanticMonitoringPage{}, nil
}
func (stubAPI) GetRuntimeHealth(context.Context, *v1.RuntimeHealthRequest) (*v1.RuntimeHealth, error) {
	return &v1.RuntimeHealth{}, nil
}
