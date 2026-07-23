package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	collectorapp "github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/application"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/httpapi"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/testsupport"
)

type failingPublicationCommitStore struct {
	*postgres.Store
	attempts  atomic.Int32
	exhausted chan struct{}
	once      sync.Once
}

func (s *failingPublicationCommitStore) CommitPreparedPublication(
	ctx context.Context,
	reference agentrun.PublicationReference,
	completion agentrun.ExecutionCompletion,
) error {
	if s.attempts.Add(1) <= 3 {
		if s.attempts.Load() == 3 {
			s.once.Do(func() { close(s.exhausted) })
		}
		return errors.New("injected persistent commit failure")
	}
	return s.Store.CommitPreparedPublication(ctx, reference, completion)
}

type runSnapshot struct {
	ExecutionID          string `json:"execution_id"`
	Status               string `json:"status"`
	ErrorCode            string `json:"error_code"`
	StopReason           string `json:"stop_reason"`
	BlockedByExecutionID string `json:"blocked_by_execution_id"`
	Invocations          []struct {
		Status       string `json:"status"`
		ErrorCode    string `json:"error_code"`
		ErrorSummary string `json:"error_summary"`
	} `json:"invocations"`
	Artifacts       map[string]string `json:"artifacts"`
	CandidateCounts map[string]int    `json:"candidate_counts"`
}

func TestCollectorHTTPAuthenticationIdempotencyAndActiveRunConflict(t *testing.T) {
	store := openHTTPTestStore(t)
	releasePlanner := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(releasePlanner) })
	var plannerCalls atomic.Int32
	providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/chat/completions" {
			plannerCalls.Add(1)
			<-releasePlanner
			_, _ = writer.Write([]byte(`{"id":"chat-blocked","object":"chat.completion","created":1,"model":"deepseek-chat","choices":[{"index":0,"message":{"role":"assistant","content":"{\"queries\":[\"中国市场\"],\"combined_query\":\"中国市场\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
			return
		}
		switch request.URL.Path {
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
		}
	}))
	defer providers.Close()
	configureHTTPTestProviders(t, store, providers.URL)
	application, err := collectorapp.New(store, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
	defer server.Close()

	unauthorized, _ := http.NewRequest(http.MethodGet, server.URL+"/internal/agent-run/v1/collector/runs/00000000-0000-0000-0000-000000000000", nil)
	unauthorizedResponse, err := http.DefaultClient.Do(unauthorized)
	if err != nil {
		t.Fatal(err)
	}
	unauthorizedResponse.Body.Close()
	if unauthorizedResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorizedResponse.StatusCode)
	}

	key := "idempotency-" + fmt.Sprint(time.Now().UnixNano())
	first := postHTTPTestRun(t, server.URL, key, "采集中国市场资讯")
	replayed := postHTTPTestRun(t, server.URL, key, "采集中国市场资讯")
	if replayed.ExecutionID != first.ExecutionID {
		t.Fatalf("replayed ID = %q, want %q", replayed.ExecutionID, first.ExecutionID)
	}
	productionApplication, err := collectorapp.New(store, t.TempDir(), collectorapp.WithEnvironment("production"))
	if err != nil {
		t.Fatal(err)
	}
	productionServer := httptest.NewServer(httpapi.NewHandler(productionApplication, "service-test-token"))
	defer productionServer.Close()
	status, replayWhileNotReady := rawPostHTTPTestRun(t, productionServer.URL, key, "采集中国市场资讯")
	if status != http.StatusAccepted || replayWhileNotReady["execution_id"] != first.ExecutionID {
		t.Fatalf("not-ready replay status=%d body=%#v", status, replayWhileNotReady)
	}

	status, conflict := rawPostHTTPTestRun(t, server.URL, key, "不同采集意图")
	if status != http.StatusConflict || conflict["error_code"] != "idempotency_conflict" {
		t.Fatalf("idempotency conflict status=%d body=%#v", status, conflict)
	}
	status, conflict = rawPostHTTPTestRun(t, server.URL, key+"-other", "另一轮采集")
	if status != http.StatusConflict || conflict["error_code"] != "active_execution_exists" ||
		conflict["active_execution_id"] != first.ExecutionID || conflict["skipped_execution_id"] == "" {
		t.Fatalf("active conflict status=%d body=%#v", status, conflict)
	}
	skippedID, ok := conflict["skipped_execution_id"].(string)
	if !ok {
		t.Fatalf("skipped execution ID = %#v", conflict["skipped_execution_id"])
	}
	skipped := getHTTPTestRun(t, server.URL, skippedID)
	if skipped.Status != "skipped" || skipped.StopReason != "skipped_previous_run_active" ||
		skipped.BlockedByExecutionID != first.ExecutionID || len(skipped.Invocations) != 7 {
		t.Fatalf("skipped execution = %#v", skipped)
	}
	if _, err := os.Stat(skipped.Artifacts["manifest"]); err != nil {
		t.Fatalf("skipped audit manifest: %v", err)
	}
	for _, invocation := range skipped.Invocations {
		if invocation.Status != "not_invoked" || invocation.ErrorCode != "not_invoked" {
			t.Fatalf("skipped invocation = %#v", invocation)
		}
	}
	status, replayedConflict := rawPostHTTPTestRun(t, server.URL, key+"-other", "另一轮采集")
	if status != http.StatusConflict || replayedConflict["skipped_execution_id"] != skippedID ||
		replayedConflict["active_execution_id"] != first.ExecutionID {
		t.Fatalf("replayed active conflict status=%d body=%#v", status, replayedConflict)
	}
	for deadline := time.Now().Add(time.Second); plannerCalls.Load() == 0 && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if plannerCalls.Load() != 1 {
		t.Fatalf("Planner calls = %d, want 1", plannerCalls.Load())
	}
	if running := getHTTPTestRun(t, server.URL, first.ExecutionID); running.Status != "planning" {
		t.Fatalf("blocked Planner status = %q, want planning", running.Status)
	}
	releaseOnce.Do(func() { close(releasePlanner) })
	finished := waitHTTPTestRun(t, server.URL, first.ExecutionID)
	if finished.Status != "succeeded_no_change" || len(finished.Invocations) != 7 || finished.CandidateCounts["results_pending"] != 0 {
		t.Fatalf("finished run = %#v", finished)
	}
	if _, err := os.Stat(finished.Artifacts["manifest"]); err != nil {
		t.Fatalf("no-change manifest: %v", err)
	}
	encoded, _ := json.Marshal(finished)
	if strings.Contains(string(encoded), "采集中国市场资讯") {
		t.Fatalf("no-change response leaked Prompt: %s", encoded)
	}
}

func TestCollectorHTTPReadinessAndPromptValidation(t *testing.T) {
	store := openHTTPTestStore(t)
	providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("invalid request reached Provider: %s", request.URL.Path)
	}))
	defer providers.Close()
	configureHTTPTestProviders(t, store, providers.URL)

	localApplication, err := collectorapp.New(store, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	localServer := httptest.NewServer(httpapi.NewHandler(localApplication, "service-test-token"))
	defer localServer.Close()

	healthResponse, err := http.Get(localServer.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	healthResponse.Body.Close()
	if healthResponse.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", healthResponse.StatusCode)
	}

	tests := []struct {
		name           string
		body           string
		idempotencyKey string
		wantStatus     int
		wantCode       string
	}{
		{name: "missing idempotency key", body: `{"prompt":"采集资讯"}`, wantStatus: 400, wantCode: "idempotency_key_required"},
		{name: "blank prompt", body: `{"prompt":"  \n "}`, idempotencyKey: "blank", wantStatus: 400, wantCode: "prompt_required"},
		{name: "unknown field", body: `{"prompt":"采集资讯","connector":"tavily"}`, idempotencyKey: "unknown", wantStatus: 400, wantCode: "invalid_request"},
		{name: "trailing content", body: `{"prompt":"采集资讯"} true`, idempotencyKey: "trailing", wantStatus: 400, wantCode: "invalid_request"},
		{name: "over 64 KiB", body: `{"prompt":"` + strings.Repeat("a", 64*1024+1) + `"}`, idempotencyKey: "large", wantStatus: 413, wantCode: "prompt_too_large"},
		{name: "request far over limit", body: `{"prompt":"` + strings.Repeat("a", 80*1024) + `"}`, idempotencyKey: "very-large", wantStatus: 413, wantCode: "prompt_too_large"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodPost, localServer.URL+"/internal/agent-run/v1/collector/runs", strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer service-test-token")
			request.Header.Set("Content-Type", "application/json")
			if test.idempotencyKey != "" {
				request.Header.Set("Idempotency-Key", test.idempotencyKey)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			var payload map[string]any
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.wantStatus || payload["error_code"] != test.wantCode {
				t.Fatalf("status=%d body=%#v", response.StatusCode, payload)
			}
		})
	}

	productionApplication, err := collectorapp.New(store, t.TempDir(), collectorapp.WithEnvironment("production"))
	if err != nil {
		t.Fatal(err)
	}
	productionServer := httptest.NewServer(httpapi.NewHandler(productionApplication, "service-test-token"))
	defer productionServer.Close()
	readyResponse, err := http.Get(productionServer.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	readyResponse.Body.Close()
	if readyResponse.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("production readiness status = %d", readyResponse.StatusCode)
	}
	status, payload := rawPostHTTPTestRun(t, productionServer.URL, "production", "采集资讯")
	if status != http.StatusServiceUnavailable || payload["error_code"] != "configuration_not_ready" {
		t.Fatalf("production POST status=%d body=%#v", status, payload)
	}
}

func TestPlannerFailureIsFailClosedAndRedacted(t *testing.T) {
	store := openHTTPTestStore(t)
	var connectorCalls atomic.Int32
	providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/chat/completions" {
			_, _ = writer.Write([]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"secret malformed planner response"},"finish_reason":"stop"}]}`))
			return
		}
		connectorCalls.Add(1)
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer providers.Close()
	configureHTTPTestProviders(t, store, providers.URL)
	application, err := collectorapp.New(store, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
	defer server.Close()

	created := postHTTPTestRun(t, server.URL, "planner-failure-"+fmt.Sprint(time.Now().UnixNano()), "secret business collection prompt")
	observed := waitHTTPTestRun(t, server.URL, created.ExecutionID)
	if observed.Status != "failed" || observed.ErrorCode != "planning_failed" || observed.StopReason != "agent_or_tool_limit" {
		t.Fatalf("run = %#v", observed)
	}
	if connectorCalls.Load() != 0 || len(observed.Invocations) != 7 {
		t.Fatalf("connector calls=%d invocations=%#v", connectorCalls.Load(), observed.Invocations)
	}
	for _, invocation := range observed.Invocations {
		if invocation.Status != "not_invoked" || invocation.ErrorCode != "not_invoked" || invocation.ErrorSummary != "Connector was not invoked because query planning failed" {
			t.Fatalf("invocation = %#v", invocation)
		}
	}
	if _, err := os.Stat(observed.Artifacts["manifest"]); err != nil {
		t.Fatalf("failed planning audit manifest: %v", err)
	}
	encoded, _ := json.Marshal(observed)
	for _, secret := range []string{"secret business collection prompt", "secret malformed planner response"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("GET response leaked %q: %s", secret, encoded)
		}
	}
}

func TestCollectorAcceptsEscapedPromptAtDecodedLimit(t *testing.T) {
	store := openHTTPTestStore(t)
	providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/chat/completions" {
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"invalid"}}]}`))
			return
		}
		writeEmptyConnectorResponse(writer, request.URL.Path)
	}))
	defer providers.Close()
	configureHTTPTestProviders(t, store, providers.URL)
	application, err := collectorapp.New(store, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
	defer server.Close()
	body := `{"prompt":"` + strings.Repeat(`\u0061`, 64*1024) + `"}`
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/internal/agent-run/v1/collector/runs", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer service-test-token")
	request.Header.Set("Idempotency-Key", "escaped-limit-"+fmt.Sprint(time.Now().UnixNano()))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("escaped 64 KiB prompt status=%d body=%s", response.StatusCode, payload)
	}
	var created struct {
		ExecutionID string `json:"execution_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if observed := waitHTTPTestRun(t, server.URL, created.ExecutionID); observed.Status != "failed" {
		t.Fatalf("escaped Prompt run = %#v", observed)
	}
}

func TestExecutionTimeoutAndArtifactFailure(t *testing.T) {
	store := openHTTPTestStore(t)

	t.Run("execution timeout", func(t *testing.T) {
		var connectorCalls atomic.Int32
		releaseProvider := make(chan struct{})
		var releaseOnce sync.Once
		providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/chat/completions" {
				<-releaseProvider
				return
			}
			connectorCalls.Add(1)
		}))
		defer func() {
			releaseOnce.Do(func() { close(releaseProvider) })
			providers.Close()
		}()
		configureHTTPTestProviders(t, store, providers.URL)
		application, err := collectorapp.New(store, t.TempDir(), collectorapp.WithExecutionTimeout(100*time.Millisecond))
		if err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
		defer server.Close()
		created := postHTTPTestRun(t, server.URL, "timeout-"+fmt.Sprint(time.Now().UnixNano()), "采集超时测试")
		observed := waitHTTPTestRun(t, server.URL, created.ExecutionID)
		if observed.Status != "failed" || observed.ErrorCode != "execution_timeout" ||
			observed.StopReason != "agent_or_tool_limit" || connectorCalls.Load() != 0 {
			t.Fatalf("timeout run=%#v connector_calls=%d", observed, connectorCalls.Load())
		}
		if _, err := os.Stat(observed.Artifacts["manifest"]); err != nil {
			t.Fatalf("timeout audit manifest: %v", err)
		}
		for _, invocation := range observed.Invocations {
			if invocation.Status != "not_invoked" || invocation.ErrorCode != "not_invoked" {
				t.Fatalf("timeout invocation=%#v", invocation)
			}
		}
		releaseOnce.Do(func() { close(releaseProvider) })
	})

	t.Run("Artifact failure", func(t *testing.T) {
		plannerStarted := make(chan struct{})
		releasePlanner := make(chan struct{})
		var startedOnce, releaseOnce sync.Once
		providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			if request.URL.Path == "/chat/completions" {
				startedOnce.Do(func() { close(plannerStarted) })
				<-releasePlanner
				_, _ = writer.Write([]byte(`{"id":"chat-artifact","object":"chat.completion","choices":[{"message":{"role":"assistant","content":"{\"queries\":[\"市场\"],\"combined_query\":\"市场\"}"}}]}`))
				return
			}
			writeEmptyConnectorResponse(writer, request.URL.Path)
		}))
		defer func() {
			releaseOnce.Do(func() { close(releasePlanner) })
			providers.Close()
		}()
		configureHTTPTestProviders(t, store, providers.URL)
		artifactRoot := t.TempDir()
		application, err := collectorapp.New(store, artifactRoot)
		if err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
		defer server.Close()
		created := postHTTPTestRun(t, server.URL, "artifact-failure-"+fmt.Sprint(time.Now().UnixNano()), "采集 Artifact 失败测试")
		select {
		case <-plannerStarted:
		case <-time.After(time.Second):
			t.Fatal("Planner did not start")
		}
		if err := os.RemoveAll(artifactRoot); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(artifactRoot, []byte("not a directory"), 0o644); err != nil {
			t.Fatal(err)
		}
		releaseOnce.Do(func() { close(releasePlanner) })
		observed := waitHTTPTestRun(t, server.URL, created.ExecutionID)
		if observed.Status != "failed" || observed.ErrorCode != "artifact_failed" || len(observed.Artifacts) != 0 || len(observed.Invocations) != 7 {
			t.Fatalf("Artifact failure run=%#v", observed)
		}
	})
}

func TestConnectorFailureMatrix(t *testing.T) {
	store := openHTTPTestStore(t)

	t.Run("one failure produces partial success", func(t *testing.T) {
		providers := newMatrixProviderServer(t, false, true)
		defer providers.Close()
		configureHTTPTestProviders(t, store, providers.URL)
		application, err := collectorapp.New(store, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
		defer server.Close()

		created := postHTTPTestRun(t, server.URL, "partial-"+fmt.Sprint(time.Now().UnixNano()), "采集中国市场资讯")
		observed := waitHTTPTestRun(t, server.URL, created.ExecutionID)
		if observed.Status != "partially_succeeded" || len(observed.Invocations) != 7 || observed.CandidateCounts["results_pending"] != 0 || observed.CandidateCounts["accepted"] != 1 {
			t.Fatalf("partial run = %#v", observed)
		}
		failed := 0
		for _, invocation := range observed.Invocations {
			if invocation.Status == "failed" {
				failed++
			}
		}
		if failed != 1 {
			t.Fatalf("failed invocations = %d, want 1", failed)
		}
		if _, err := os.Stat(observed.Artifacts["manifest"]); err != nil {
			t.Fatalf("partial run manifest: %v", err)
		}
	})

	t.Run("one failure with no accepted result remains partial", func(t *testing.T) {
		providers := newMatrixProviderServer(t, false, false)
		defer providers.Close()
		configureHTTPTestProviders(t, store, providers.URL)
		application, err := collectorapp.New(store, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
		defer server.Close()

		prompt := "采集不会产生新文档的资讯"
		created := postHTTPTestRun(t, server.URL, "partial-empty-"+fmt.Sprint(time.Now().UnixNano()), prompt)
		observed := waitHTTPTestRun(t, server.URL, created.ExecutionID)
		if observed.Status != "partially_succeeded" || len(observed.Invocations) != 7 || observed.CandidateCounts["accepted"] != 0 || observed.CandidateCounts["results_pending"] != 0 {
			t.Fatalf("partial empty run = %#v", observed)
		}
		if _, err := os.Stat(observed.Artifacts["manifest"]); err != nil {
			t.Fatalf("partial empty manifest: %v", err)
		}
		encoded, _ := json.Marshal(observed)
		if strings.Contains(string(encoded), prompt) || strings.Contains(string(encoded), "provider-body") {
			t.Fatalf("partial empty response leaked sensitive input: %s", encoded)
		}
	})

	t.Run("all failures fail execution", func(t *testing.T) {
		providers := newMatrixProviderServer(t, true, false)
		defer providers.Close()
		configureHTTPTestProviders(t, store, providers.URL)
		artifactRoot := t.TempDir()
		application, err := collectorapp.New(store, artifactRoot)
		if err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
		defer server.Close()

		created := postHTTPTestRun(t, server.URL, "all-failed-"+fmt.Sprint(time.Now().UnixNano()), "采集中国市场资讯")
		observed := waitHTTPTestRun(t, server.URL, created.ExecutionID)
		if observed.Status != "failed" || observed.ErrorCode != "all_connectors_failed" ||
			observed.StopReason != "completed_with_connector_failures" ||
			observed.CandidateCounts["results_pending"] != 0 {
			t.Fatalf("all-failed run = %#v", observed)
		}
		if _, err := os.Stat(observed.Artifacts["manifest"]); err != nil {
			t.Fatalf("all-failed audit manifest: %v", err)
		}
		for _, invocation := range observed.Invocations {
			if invocation.Status != "failed" {
				t.Fatalf("invocation = %#v", invocation)
			}
		}
	})
}

func TestStartupReconcilesPreparedPublicationBeforeStaleAndExposesArtifacts(t *testing.T) {
	store := openHTTPTestStore(t)
	var providerCalls atomic.Int32
	providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		providerCalls.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/chat/completions" {
			_, _ = writer.Write([]byte(`{"id":"chat-reconcile","object":"chat.completion","created":1,"model":"deepseek-chat","choices":[{"index":0,"message":{"role":"assistant","content":"{\"queries\":[\"中国市场\"],\"combined_query\":\"中国市场\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
			return
		}
		if request.URL.Path == "/parallel" {
			_, _ = writer.Write([]byte(`{"results":[{"url":"https://example.com/recovered","title":"恢复测试","excerpts":["直接片段"]}]}`))
			return
		}
		writeEmptyConnectorResponse(writer, request.URL.Path)
	}))
	defer providers.Close()
	configureHTTPTestProviders(t, store, providers.URL)

	artifactRoot := t.TempDir()
	failingStore := &failingPublicationCommitStore{
		Store: store, exhausted: make(chan struct{}),
	}
	firstApplication, err := collectorapp.New(failingStore, artifactRoot)
	if err != nil {
		t.Fatal(err)
	}
	firstServer := httptest.NewServer(httpapi.NewHandler(firstApplication, "service-test-token"))
	defer firstServer.Close()
	created := postHTTPTestRun(t, firstServer.URL, "restart-reconcile-"+fmt.Sprint(time.Now().UnixNano()), "采集恢复测试")
	select {
	case <-failingStore.exhausted:
	case <-time.After(5 * time.Second):
		t.Fatal("publication commit failure was not reached")
	}
	prepared, err := store.GetExecution(context.Background(), created.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Status != agentrun.StatusMaterializing || len(prepared.Artifacts) != 0 {
		t.Fatalf("prepared Execution = %#v", prepared)
	}
	references, err := store.ListPreparedPublications(context.Background())
	if err != nil || len(references) != 1 {
		t.Fatalf("prepared references = %#v, err=%v", references, err)
	}
	callsBeforeRestart := providerCalls.Load()

	restartedApplication, err := collectorapp.New(store, artifactRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := restartedApplication.ReconcileStartup(context.Background(), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if providerCalls.Load() != callsBeforeRestart {
		t.Fatalf("startup reconciliation called Planner or Connector: before=%d after=%d", callsBeforeRestart, providerCalls.Load())
	}
	restartedServer := httptest.NewServer(httpapi.NewHandler(restartedApplication, "service-test-token"))
	defer restartedServer.Close()
	recovered := getHTTPTestRun(t, restartedServer.URL, created.ExecutionID)
	if recovered.Status != string(agentrun.StatusSucceeded) ||
		recovered.CandidateCounts["results_pending"] != 0 ||
		recovered.Artifacts["manifest"] == "" {
		t.Fatalf("recovered Execution = %#v", recovered)
	}
	if _, err := os.Stat(recovered.Artifacts["manifest"]); err != nil {
		t.Fatalf("recovered manifest: %v", err)
	}
	references, err = store.ListPreparedPublications(context.Background())
	if err != nil || len(references) != 0 {
		t.Fatalf("remaining prepared references = %#v, err=%v", references, err)
	}
}

func TestStartupAuditsUnpreparedAndTerminalFailureWithoutProviderReplay(t *testing.T) {
	t.Run("unprepared active Execution becomes stale failure", func(t *testing.T) {
		store := openHTTPTestStore(t)
		now := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
		execution, _, err := store.CreateExecution(context.Background(), agentrun.CreateExecutionInput{
			IdempotencyKey: "stale-before-prepare", Prompt: "collect",
			CreatedAt: now, AgentVersion: "collector.v1", InvocationKeys: collector.ConnectorKeys(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.SetExecutionStatus(context.Background(), execution.ID, agentrun.StatusPlanning, now); err != nil {
			t.Fatal(err)
		}
		application, err := collectorapp.New(store, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if err := application.ReconcileStartup(context.Background(), now.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
		defer server.Close()
		observed := getHTTPTestRun(t, server.URL, execution.ID)
		if observed.Status != string(agentrun.StatusFailed) ||
			observed.ErrorCode != "process_restarted" ||
			observed.StopReason != "agent_or_tool_limit" ||
			observed.Artifacts["manifest"] == "" {
			t.Fatalf("stale Execution = %#v", observed)
		}
		for _, invocation := range observed.Invocations {
			if invocation.Status != string(agentrun.InvocationNotInvoked) {
				t.Fatalf("stale invocation = %#v", invocation)
			}
		}
	})

	t.Run("terminal database row receives missing audit", func(t *testing.T) {
		store := openHTTPTestStore(t)
		now := time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC)
		execution, _, err := store.CreateExecution(context.Background(), agentrun.CreateExecutionInput{
			IdempotencyKey: "terminal-without-audit", Prompt: "collect",
			CreatedAt: now, AgentVersion: "collector.v1", InvocationKeys: collector.ConnectorKeys(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.FailExecutionAndIncompleteInvocations(context.Background(), agentrun.ExecutionFailure{
			ExecutionID: execution.ID, ErrorCode: "planning_failed", ErrorSummary: "Query planning failed",
			StopReason: "agent_or_tool_limit", NotInvokedSummary: "Connector was not invoked",
			CompletedAt: now.Add(time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
		artifactRoot := t.TempDir()
		application, err := collectorapp.New(store, artifactRoot)
		if err != nil {
			t.Fatal(err)
		}
		if err := application.ReconcileStartup(context.Background(), now.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
		defer server.Close()
		observed := getHTTPTestRun(t, server.URL, execution.ID)
		if observed.Status != string(agentrun.StatusFailed) ||
			observed.ErrorCode != "planning_failed" ||
			observed.Artifacts["manifest"] == "" {
			t.Fatalf("terminal audit recovery = %#v", observed)
		}
		if _, err := os.Stat(observed.Artifacts["manifest"]); err != nil {
			t.Fatalf("recovered terminal audit: %v", err)
		}
	})
}

func newMatrixProviderServer(t *testing.T, failAll, emitCandidate bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/chat/completions" {
			_, _ = writer.Write([]byte(`{"id":"chat-matrix","object":"chat.completion","created":1,"model":"deepseek-chat","choices":[{"index":0,"message":{"role":"assistant","content":"{\"queries\":[\"中国市场\"],\"combined_query\":\"中国市场\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
			return
		}
		if failAll || request.URL.Path == "/tavily" {
			writer.WriteHeader(http.StatusBadGateway)
			_, _ = writer.Write([]byte(`{"secret":"provider-body"}`))
			return
		}
		switch request.URL.Path {
		case "/parallel":
			if emitCandidate {
				_, _ = writer.Write([]byte(`{"results":[{"url":"https://example.com/only-result","title":"唯一结果","excerpts":["直接片段"]}]}`))
			} else {
				_, _ = writer.Write([]byte(`{"results":[]}`))
			}
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
}

func writeEmptyConnectorResponse(writer http.ResponseWriter, path string) {
	switch path {
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
	}
}

func openHTTPTestStore(t *testing.T) *postgres.Store {
	t.Helper()
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	isolatedURL, cleanup, err := testsupport.IsolatedPostgresDatabase(context.Background(), databaseURL, "httpapi_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	database, err := postgres.Open(context.Background(), isolatedURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)
	if err := postgres.Migrate(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	store := postgres.New(database)
	if err := store.FailStaleExecutions(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	return store
}

func configureHTTPTestProviders(t *testing.T, store *postgres.Store, baseURL string) {
	t.Helper()
	configs := []collector.ProviderConfig{
		{Key: "deepseek", BaseURL: baseURL, Model: "deepseek-chat", APIKey: "deepseek-test-key"},
		{Key: "parallel_search", BaseURL: baseURL + "/parallel", APIKey: "parallel-test-key"},
		{Key: "tavily", BaseURL: baseURL + "/tavily", APIKey: "tavily-test-key"},
		{Key: "bocha", BaseURL: baseURL + "/bocha", APIKey: "bocha-test-key"},
		{Key: "cls_telegraph", BaseURL: baseURL + "/cls"},
		{Key: "eastmoney_fastnews", BaseURL: baseURL + "/eastmoney-fast"},
		{Key: "eastmoney_stock_news", BaseURL: baseURL + "/eastmoney-stock"},
		{Key: "stcn_quicknews", BaseURL: baseURL + "/stcn"},
	}
	for _, config := range configs {
		if err := store.UpsertProviderConfig(context.Background(), config); err != nil {
			t.Fatal(err)
		}
	}
}

func postHTTPTestRun(t *testing.T, serverURL, idempotencyKey, prompt string) runSnapshot {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"prompt": prompt})
	request, _ := http.NewRequest(http.MethodPost, serverURL+"/internal/agent-run/v1/collector/runs", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer service-test-token")
	request.Header.Set("Idempotency-Key", idempotencyKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("POST status=%d", response.StatusCode)
	}
	var snapshot runSnapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func rawPostHTTPTestRun(t *testing.T, serverURL, idempotencyKey, prompt string) (int, map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"prompt": prompt})
	request, _ := http.NewRequest(http.MethodPost, serverURL+"/internal/agent-run/v1/collector/runs", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer service-test-token")
	request.Header.Set("Idempotency-Key", idempotencyKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, payload
}

func waitHTTPTestRun(t *testing.T, serverURL, executionID string) runSnapshot {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		snapshot := getHTTPTestRun(t, serverURL, executionID)
		switch snapshot.Status {
		case "succeeded", "succeeded_no_change", "partially_succeeded":
			return snapshot
		case "failed":
			if snapshot.ErrorCode == "artifact_failed" || len(snapshot.Artifacts) > 0 {
				return snapshot
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("execution did not finish: %#v", snapshot)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func getHTTPTestRun(t *testing.T, serverURL, executionID string) runSnapshot {
	t.Helper()
	request, _ := http.NewRequest(http.MethodGet, serverURL+"/internal/agent-run/v1/collector/runs/"+executionID, nil)
	request.Header.Set("Authorization", "Bearer service-test-token")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var snapshot runSnapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}
