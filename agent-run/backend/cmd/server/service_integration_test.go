package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	kratos "github.com/go-kratos/kratos/v3"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/guanchaojia/tidewise-ai-agentrun/api/agentrun/v1"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector"
	collectorapp "github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector/usecase"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform/admin"
	bizschedule "github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform/scheduling"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/conf"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/artifacts"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/connectors"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/modelprovider/deepseek"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/postgres"
	scheduler "github.com/guanchaojia/tidewise-ai-agentrun/internal/data/scheduler"
	serverlayer "github.com/guanchaojia/tidewise-ai-agentrun/internal/server"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/service"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/testsupport"
	"github.com/jonboulle/clockwork"
)

func TestAdminScheduleTriggersCollectorAndListsExecution(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	isolatedURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "scheduled_collector_service_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	database, err := postgres.Open(ctx, isolatedURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	store := postgres.New(database)

	var callsMu sync.Mutex
	calls := make(map[string]int)
	var blockModel atomic.Bool
	blockedModelStarted := make(chan struct{}, 1)
	blockedModelRelease := make(chan struct{})
	providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		callsMu.Lock()
		calls[request.URL.Path]++
		callsMu.Unlock()
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/chat/completions":
			if blockModel.Load() {
				select {
				case blockedModelStarted <- struct{}{}:
				default:
				}
				select {
				case <-request.Context().Done():
				case <-blockedModelRelease:
				}
				return
			}
			_, _ = writer.Write([]byte(`{"id":"scheduled-chat","object":"chat.completion","created":1,"model":"deepseek-chat","choices":[{"index":0,"message":{"role":"assistant","content":"{\"queries\":[\"中国市场\"],\"combined_query\":\"中国市场\",\"time_window_hours\":48}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
		case "/parallel":
			_, _ = writer.Write([]byte(`{"results":[]}`))
		case "/tavily":
			_, _ = writer.Write([]byte(`{"results":[]}`))
		case "/bocha":
			_, _ = writer.Write([]byte(`{"data":{"webPages":{"value":[]}}}`))
		case "/cls":
			_, _ = writer.Write([]byte(`{"data":{"roll_data":[]}}`))
		case "/eastmoney-fast":
			_, _ = writer.Write([]byte(`{"data":{"fastNewsList":[]}}`))
		case "/eastmoney-stock":
			_, _ = writer.Write([]byte(`callback({"result":{"cmsArticleWebOld":[]}})`))
		case "/stcn":
			_, _ = writer.Write([]byte(`{"state":1,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer providers.Close()

	artifactRoot := t.TempDir()
	artifactStore := artifacts.Store{Root: artifactRoot, Publications: store, Now: time.Now}
	collectorApplication, err := collectorapp.New(
		store,
		deepseek.Factory{},
		connectors.Factory{},
		artifactStore,
		collectorapp.WithEnvironment("dev"),
	)
	if err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 7, 24, 9, 59, 0, 0, location)
	clock := clockwork.NewFakeClockAt(start)
	scheduleRunners := map[string]bizschedule.AgentRunner{
		collector.AgentKey: collector.NewScheduleRunner(collectorApplication),
	}
	scheduleRuntime, err := scheduler.NewRuntime(
		store,
		location,
		scheduleRunners,
		scheduler.WithClock(clock),
	)
	if err != nil {
		t.Fatal(err)
	}
	scheduleService, err := bizschedule.New(
		store,
		scheduleRunners,
		scheduleRuntime,
		bizschedule.WithNow(clock.Now),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = scheduleService.Shutdown() })
	if err := scheduleService.Start(ctx); err != nil {
		t.Fatal(err)
	}
	adminService, err := admin.New(
		store,
		admin.Registry{
			ModelProviderKeys: []string{collector.ModelProviderDeepSeek},
			ConnectorKeys:     collector.ConnectorKeys(),
		},
		"dev",
		admin.WithScheduleManager(scheduleService),
	)
	if err != nil {
		t.Fatal(err)
	}
	apiService, err := service.NewAgentRunService(collectorApplication, adminService, scheduleService)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := serverlayer.NewHTTPServer(conf.Config{
		App:      conf.AppConfig{Name: conf.ServiceName, Env: conf.EnvDev},
		Server:   conf.ServerConfig{Host: "127.0.0.1", Port: 0},
		Artifact: conf.ArtifactConfig{Root: artifactRoot},
		Secrets: conf.SecretConfig{
			ServiceToken: "service-test-token",
			AdminToken:   "admin-test-token",
		},
	}, apiService, apiService, slog.New(slog.NewTextHandler(io.Discard, nil)))
	httpServer.Route("/").GET("/__test/panic", func(ctx kratoshttp.Context) error {
		kratoshttp.SetOperation(ctx, "/agentrun.v1.Operations/Health")
		handler := ctx.Middleware(func(context.Context, any) (any, error) {
			panic("black-box panic detail")
		})
		_, err := handler(ctx, nil)
		return err
	})
	assertBlackBoxOpenAPIParity(t, httpServer)
	appStarted := make(chan struct{})
	application := kratos.New(
		kratos.Name(conf.ServiceName),
		kratos.Version(serviceVersion),
		kratos.Logger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		kratos.Server(httpServer),
		kratos.StopTimeout(time.Second),
		kratos.AfterStart(func(context.Context) error {
			close(appStarted)
			return nil
		}),
		kratos.BeforeStop(func(context.Context) error {
			return scheduleService.Shutdown()
		}),
		kratos.AfterStop(func(context.Context) error {
			stopContext, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			collectorApplication.BeginShutdown()
			collectorErr := collectorApplication.Wait(stopContext)
			databaseErr := closeWithin(stopContext, database.Close)
			return errors.Join(collectorErr, databaseErr)
		}),
	)
	runResult := make(chan error, 1)
	go func() { runResult <- application.Run() }()
	select {
	case <-appStarted:
	case runErr := <-runResult:
		t.Fatalf("Kratos App exited during startup: %v", runErr)
	case <-time.After(3 * time.Second):
		t.Fatal("Kratos App did not finish startup")
	}
	var serviceURL string
	var lastEndpoint string
	var lastRequestErr error
	startupDeadline := time.Now().Add(3 * time.Second)
	for {
		select {
		case runErr := <-runResult:
			t.Fatalf("Kratos App exited during startup: %v", runErr)
		default:
		}
		endpoint, endpointErr := httpServer.Endpoint()
		if endpointErr == nil && endpoint.Port() != "" && endpoint.Port() != "0" {
			candidateURL := endpoint.String()
			lastEndpoint = candidateURL
			response, requestErr := http.Get(candidateURL + "/healthz")
			lastRequestErr = requestErr
			if requestErr == nil {
				response.Body.Close()
				if response.StatusCode == http.StatusOK {
					serviceURL = candidateURL
					break
				}
			}
		}
		if time.Now().After(startupDeadline) {
			t.Fatalf("Kratos App did not expose an HTTP endpoint: endpoint=%q endpoint_err=%v request_err=%v",
				lastEndpoint, endpointErr, lastRequestErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	testServer := liveHTTPServer{URL: serviceURL, Client: http.DefaultClient}
	var stopOnce sync.Once
	stopApplication := func() {
		stopOnce.Do(func() {
			if err := application.Stop(); err != nil {
				t.Errorf("stop Kratos App: %v", err)
			}
			select {
			case err := <-runResult:
				if err != nil {
					t.Errorf("run Kratos App: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Error("Kratos App did not stop within the bounded deadline")
			}
		})
	}
	t.Cleanup(stopApplication)

	unauthorizedRequest, _ := http.NewRequest(
		http.MethodPost,
		testServer.URL+v1.CollectorRunsPath,
		strings.NewReader(`{"prompt":"unauthorized"}`),
	)
	unauthorizedRequest.Header.Set("Content-Type", "application/json")
	unauthorizedRequest.Header.Set("Idempotency-Key", "black-box-unauthorized")
	assertLiveErrorResponse(t, testServer, unauthorizedRequest, http.StatusUnauthorized, "UNAUTHORIZED")
	adminOnCollectorRequest, _ := http.NewRequest(
		http.MethodPost,
		testServer.URL+v1.CollectorRunsPath,
		strings.NewReader(`{"prompt":"wrong token"}`),
	)
	adminOnCollectorRequest.Header.Set("Authorization", "Bearer admin-test-token")
	adminOnCollectorRequest.Header.Set("Content-Type", "application/json")
	adminOnCollectorRequest.Header.Set("Idempotency-Key", "black-box-admin-on-collector")
	assertLiveErrorResponse(t, testServer, adminOnCollectorRequest, http.StatusUnauthorized, "UNAUTHORIZED")
	serviceOnAdminRequest, _ := http.NewRequest(
		http.MethodGet,
		testServer.URL+"/api/admin/v1/model-providers",
		nil,
	)
	serviceOnAdminRequest.Header.Set("Authorization", "Bearer service-test-token")
	assertLiveErrorResponse(t, testServer, serviceOnAdminRequest, http.StatusUnauthorized, "UNAUTHORIZED")

	strictRequest, _ := http.NewRequest(
		http.MethodPost,
		testServer.URL+v1.CollectorRunsPath,
		strings.NewReader(`{"prompt":"strict","unknown":true}`),
	)
	strictRequest.Header.Set("Authorization", "Bearer service-test-token")
	strictRequest.Header.Set("Content-Type", "application/json")
	strictRequest.Header.Set("Idempotency-Key", "black-box-strict")
	assertLiveErrorResponse(t, testServer, strictRequest, http.StatusBadRequest, "INVALID_REQUEST")

	notFoundRequest, _ := http.NewRequest(http.MethodGet, testServer.URL+"/does-not-exist", nil)
	assertLiveErrorResponse(t, testServer, notFoundRequest, http.StatusNotFound, "NOT_FOUND")
	methodRequest, _ := http.NewRequest(http.MethodDelete, testServer.URL+v1.CollectorRunsPath, nil)
	assertLiveErrorResponse(t, testServer, methodRequest, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
	panicRequest, _ := http.NewRequest(http.MethodGet, testServer.URL+"/__test/panic", nil)
	assertLiveErrorResponse(t, testServer, panicRequest, http.StatusInternalServerError, "INTERNAL_ERROR")

	adminJSONRequest(t, testServer, http.MethodPatch, "/api/admin/v1/model-providers/deepseek", map[string]any{
		"base_url": providers.URL,
		"model":    "deepseek-chat",
		"api_key":  "scheduled-model-secret",
	}, http.StatusOK)
	for _, config := range []struct {
		key    string
		path   string
		apiKey string
	}{
		{key: collector.ConnectorParallelSearch, path: "/parallel", apiKey: "parallel-secret"},
		{key: collector.ConnectorTavily, path: "/tavily", apiKey: "tavily-secret"},
		{key: collector.ConnectorBocha, path: "/bocha", apiKey: "bocha-secret"},
		{key: collector.ConnectorCLSTelegraph, path: "/cls"},
		{key: collector.ConnectorEastmoneyFastNews, path: "/eastmoney-fast"},
		{key: collector.ConnectorEastmoneyStock, path: "/eastmoney-stock"},
		{key: collector.ConnectorSTCNQuickNews, path: "/stcn"},
	} {
		adminJSONRequest(t, testServer, http.MethodPatch, "/api/admin/v1/connectors/"+config.key, map[string]any{
			"base_url": providers.URL + config.path,
			"api_key":  config.apiKey,
		}, http.StatusOK)
	}

	scheduleBody := adminJSONRequest(t, testServer, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"agent_version":   "collector.v1",
		"schedule_type":   "cron",
		"cron_expression": "0 10 * * *",
		"input":           map[string]any{"prompt": "采集最近两小时中国市场资讯"},
		"enabled":         true,
	}, http.StatusOK)
	var schedule agentrun.AgentSchedule
	if err := json.Unmarshal(scheduleBody, &schedule); err != nil {
		t.Fatal(err)
	}
	if err := clock.BlockUntilContext(ctx, 1); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute)

	var page agentrun.ExecutionPage
	deadline := time.Now().Add(8 * time.Second)
	for {
		body := adminJSONRequest(t, testServer, http.MethodGet,
			"/api/admin/v1/agent-executions?agent_key=collector&page=1&page_size=20&sort_order=desc",
			nil, http.StatusOK)
		if bytes.Contains(body, []byte("采集最近两小时中国市场资讯")) ||
			bytes.Contains(body, []byte("scheduled-model-secret")) {
			t.Fatalf("Execution list exposed private payload: %s", body)
		}
		page = agentrun.ExecutionPage{}
		if err := json.Unmarshal(body, &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Items) > 0 {
			switch page.Items[0].Status {
			case agentrun.StatusSucceeded, agentrun.StatusSucceededNoChange,
				agentrun.StatusPartiallySucceeded, agentrun.StatusFailed:
				goto completed
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("scheduled Collector Execution did not finish: %#v", page)
		}
		time.Sleep(20 * time.Millisecond)
	}

completed:
	item := page.Items[0]
	if page.TotalItems != 1 || item.AgentKey != "collector" ||
		item.TriggerSource != agentrun.TriggerSchedule ||
		item.ScheduleID != schedule.ID ||
		!item.TriggeredAt.Equal(start.Add(time.Minute)) ||
		item.Status != agentrun.StatusSucceededNoChange {
		t.Fatalf("scheduled Collector Execution = %#v page=%#v", item, page)
	}
	collectorRequest, err := http.NewRequest(
		http.MethodGet,
		testServer.URL+"/api/v1/collector/runs/"+item.ID,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	collectorRequest.Header.Set("Authorization", "Bearer service-test-token")
	collectorResponse, err := testServer.Client.Do(collectorRequest)
	if err != nil {
		t.Fatal(err)
	}
	collectorResponse.Body.Close()
	if collectorResponse.StatusCode != http.StatusOK {
		t.Fatalf("Collector status route = %d", collectorResponse.StatusCode)
	}

	createRequest, err := http.NewRequest(
		http.MethodPost,
		testServer.URL+"/api/v1/collector/runs",
		strings.NewReader(`{"prompt":"采集 API 触发的中国市场资讯"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	createRequest.Header.Set("Authorization", "Bearer service-test-token")
	createRequest.Header.Set("Idempotency-Key", "kratos-black-box-api-run")
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse, err := testServer.Client.Do(createRequest)
	if err != nil {
		t.Fatal(err)
	}
	var createdEnvelope struct {
		Result struct {
			ExecutionID string `json:"execution_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createdEnvelope); err != nil {
		createResponse.Body.Close()
		t.Fatal(err)
	}
	createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusAccepted || createdEnvelope.Result.ExecutionID == "" {
		t.Fatalf("API Collector create status=%d result=%#v", createResponse.StatusCode, createdEnvelope.Result)
	}
	idempotentRequest, _ := http.NewRequest(
		http.MethodPost,
		testServer.URL+v1.CollectorRunsPath,
		strings.NewReader(`{"prompt":"采集 API 触发的中国市场资讯"}`),
	)
	idempotentRequest.Header.Set("Authorization", "Bearer service-test-token")
	idempotentRequest.Header.Set("Idempotency-Key", "kratos-black-box-api-run")
	idempotentRequest.Header.Set("Content-Type", "application/json")
	idempotentResponse, err := testServer.Client.Do(idempotentRequest)
	if err != nil {
		t.Fatal(err)
	}
	var idempotentEnvelope struct {
		Result struct {
			ExecutionID string `json:"execution_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(idempotentResponse.Body).Decode(&idempotentEnvelope); err != nil {
		idempotentResponse.Body.Close()
		t.Fatal(err)
	}
	idempotentResponse.Body.Close()
	if idempotentResponse.StatusCode != http.StatusAccepted ||
		idempotentEnvelope.Result.ExecutionID != createdEnvelope.Result.ExecutionID {
		t.Fatalf("idempotent create status=%d result=%#v", idempotentResponse.StatusCode, idempotentEnvelope.Result)
	}
	conflictRequest, _ := http.NewRequest(
		http.MethodPost,
		testServer.URL+v1.CollectorRunsPath,
		strings.NewReader(`{"prompt":"different prompt"}`),
	)
	conflictRequest.Header.Set("Authorization", "Bearer service-test-token")
	conflictRequest.Header.Set("Idempotency-Key", "kratos-black-box-api-run")
	conflictRequest.Header.Set("Content-Type", "application/json")
	assertLiveErrorResponse(t, testServer, conflictRequest, http.StatusConflict, "IDEMPOTENCY_CONFLICT")
	apiDeadline := time.Now().Add(8 * time.Second)
	var apiCandidateCounts map[string]int
	var apiArtifacts map[string]string
	for {
		statusRequest, _ := http.NewRequest(
			http.MethodGet,
			testServer.URL+"/api/v1/collector/runs/"+createdEnvelope.Result.ExecutionID,
			nil,
		)
		statusRequest.Header.Set("Authorization", "Bearer service-test-token")
		statusResponse, requestErr := testServer.Client.Do(statusRequest)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		var statusEnvelope struct {
			Result struct {
				Status          string            `json:"status"`
				CandidateCounts map[string]int    `json:"candidate_counts"`
				Artifacts       map[string]string `json:"artifacts"`
				Invocations     []struct {
					Status string `json:"status"`
				} `json:"invocations"`
			} `json:"result"`
		}
		if err := json.NewDecoder(statusResponse.Body).Decode(&statusEnvelope); err != nil {
			statusResponse.Body.Close()
			t.Fatal(err)
		}
		statusResponse.Body.Close()
		if statusEnvelope.Result.Status == string(agentrun.StatusSucceededNoChange) {
			apiCandidateCounts = statusEnvelope.Result.CandidateCounts
			apiArtifacts = statusEnvelope.Result.Artifacts
			if len(statusEnvelope.Result.Invocations) != 7 {
				t.Fatalf("API Connector Invocations = %d, want 7", len(statusEnvelope.Result.Invocations))
			}
			for _, invocation := range statusEnvelope.Result.Invocations {
				if invocation.Status != string(agentrun.InvocationCompleted) {
					t.Fatalf("API Connector Invocation = %#v", invocation)
				}
			}
			break
		}
		if time.Now().After(apiDeadline) {
			t.Fatalf("API Collector Execution did not finish: %q", statusEnvelope.Result.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if apiCandidateCounts["merged_results"] !=
		apiCandidateCounts["results_terminal"]+apiCandidateCounts["results_pending"] ||
		apiCandidateCounts["results_pending"] != 0 {
		t.Fatalf("API Candidate conservation = %#v", apiCandidateCounts)
	}
	for _, key := range []string{"manifest", "summary", "candidates", "index"} {
		path := apiArtifacts[key]
		if path == "" {
			t.Fatalf("API Artifact %s is missing: %#v", key, apiArtifacts)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("API Artifact %s is not published at %s: %v", key, path, err)
		}
	}

	callsMu.Lock()
	for _, path := range []string{
		"/chat/completions", "/parallel", "/tavily", "/bocha",
		"/cls", "/eastmoney-fast", "/eastmoney-stock", "/stcn",
	} {
		if calls[path] != 2 {
			t.Fatalf("Provider calls[%s]=%d, all=%#v", path, calls[path], calls)
		}
	}
	callsMu.Unlock()

	blockModel.Store(true)
	blockedRequest, _ := http.NewRequest(
		http.MethodPost,
		testServer.URL+"/api/v1/collector/runs",
		strings.NewReader(`{"prompt":"验证 Kratos 停机取消活跃 Execution"}`),
	)
	blockedRequest.Header.Set("Authorization", "Bearer service-test-token")
	blockedRequest.Header.Set("Idempotency-Key", "kratos-shutdown-active-run")
	blockedRequest.Header.Set("Content-Type", "application/json")
	blockedResponse, err := testServer.Client.Do(blockedRequest)
	if err != nil {
		t.Fatal(err)
	}
	var blockedEnvelope struct {
		Result struct {
			ExecutionID string `json:"execution_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(blockedResponse.Body).Decode(&blockedEnvelope); err != nil {
		blockedResponse.Body.Close()
		t.Fatal(err)
	}
	blockedResponse.Body.Close()
	if blockedResponse.StatusCode != http.StatusAccepted || blockedEnvelope.Result.ExecutionID == "" {
		t.Fatalf("active shutdown run status = %d", blockedResponse.StatusCode)
	}
	select {
	case <-blockedModelStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("active Execution did not enter the blocking Model call")
	}
	go func() {
		time.Sleep(2 * time.Second)
		close(blockedModelRelease)
	}()
	shutdownStarted := time.Now()
	stopApplication()
	if time.Since(shutdownStarted) >= 2*time.Second {
		t.Fatal("Kratos App shutdown exceeded its bounded deadline")
	}
	clock.Advance(24 * time.Hour)
	time.Sleep(50 * time.Millisecond)
	callsMu.Lock()
	defer callsMu.Unlock()
	for path, count := range calls {
		want := 2
		if path == "/chat/completions" {
			want = 3
		}
		if count != want {
			t.Fatalf("Provider calls continued after Kratos shutdown: %s=%d", path, count)
		}
	}
	if err := database.Ping(context.Background()); err == nil {
		t.Fatal("PostgreSQL pool remained usable after Kratos shutdown")
	}
	verificationDatabase, err := postgres.Open(context.Background(), isolatedURL)
	if err != nil {
		t.Fatal(err)
	}
	defer verificationDatabase.Close()
	canceledExecution, err := postgres.New(verificationDatabase).GetExecution(
		context.Background(),
		blockedEnvelope.Result.ExecutionID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if canceledExecution.Status != agentrun.StatusFailed ||
		canceledExecution.ErrorCode != "service_stopping" ||
		canceledExecution.StopReason != "agent_or_tool_limit" {
		t.Fatalf("shutdown Execution terminal state = %#v", canceledExecution)
	}
	for _, invocation := range canceledExecution.Invocations {
		if invocation.Status == agentrun.InvocationPending || invocation.Status == agentrun.InvocationRunning {
			t.Fatalf("shutdown left non-terminal Invocation: %#v", invocation)
		}
	}
}

func adminJSONRequest(
	t *testing.T,
	server liveHTTPServer,
	method string,
	path string,
	payload any,
	wantStatus int,
) []byte {
	t.Helper()
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
	}
	request, err := http.NewRequest(method, server.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer admin-test-token")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := server.Client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var decoded bytes.Buffer
	if _, err := decoded.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("%s %s status=%d body=%s", method, path, response.StatusCode, decoded.Bytes())
	}
	for _, secret := range []string{
		"scheduled-model-secret", "parallel-secret", "tavily-secret", "bocha-secret",
	} {
		if strings.Contains(decoded.String(), secret) {
			t.Fatalf("%s %s exposed %s: %s", method, path, secret, decoded.Bytes())
		}
	}
	if wantStatus >= 200 && wantStatus < 300 {
		var envelope struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(decoded.Bytes(), &envelope); err != nil || len(envelope.Result) == 0 {
			t.Fatalf("%s %s returned invalid success envelope: %s", method, path, decoded.Bytes())
		}
		return envelope.Result
	}
	return decoded.Bytes()
}

type liveHTTPServer struct {
	URL    string
	Client *http.Client
}

func assertBlackBoxOpenAPIParity(t *testing.T, runtime *kratoshttp.Server) {
	t.Helper()
	actual := make(map[string]struct{})
	if err := runtime.WalkRoute(func(route kratoshttp.RouteInfo) error {
		if route.Path == "/__test/panic" {
			return nil
		}
		path := route.Path
		if path == "/docs/{asset:.*}" {
			path = "/docs/"
		}
		actual[route.Method+" "+path] = struct{}{}
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
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("black-box runtime routes = %#v, OpenAPI routes = %#v", actual, expected)
	}
}

func assertLiveErrorResponse(
	t *testing.T,
	server liveHTTPServer,
	request *http.Request,
	wantStatus int,
	wantCode string,
) {
	t.Helper()
	response, err := server.Client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var envelope struct {
		RequestID string `json:"request_id"`
		Error     struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != wantStatus || envelope.Error.Code != wantCode ||
		envelope.RequestID == "" || envelope.Error.Message == "" || envelope.Error.Details == nil ||
		response.Header.Get(serverlayer.RequestIDHeader) != envelope.RequestID {
		t.Fatalf("error response status=%d envelope=%#v headers=%v", response.StatusCode, envelope, response.Header)
	}
}
