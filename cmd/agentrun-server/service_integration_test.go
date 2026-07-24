package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/admin"
	agentrunhttp "github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/httpapi"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/scheduling"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	collectorapp "github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/application"
	collectorhttp "github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/httpapi"
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
	defer cleanup()
	database, err := postgres.Open(ctx, isolatedURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	store := postgres.New(database)

	var callsMu sync.Mutex
	calls := make(map[string]int)
	providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		callsMu.Lock()
		calls[request.URL.Path]++
		callsMu.Unlock()
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/chat/completions":
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
	collectorApplication, err := collectorapp.New(store, artifactRoot, collectorapp.WithEnvironment("dev"))
	if err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 7, 24, 9, 59, 0, 0, location)
	clock := clockwork.NewFakeClockAt(start)
	scheduleService, err := scheduling.New(
		store,
		location,
		map[string]scheduling.AgentRunner{
			collectorAgentKey: collectorScheduleRunner{collector: collectorApplication},
		},
		scheduling.WithClock(clock),
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
	server := httptest.NewServer(newRootHandler(
		collectorhttp.NewHandler(collectorApplication, "service-test-token"),
		agentrunhttp.NewAdminHandler(adminService, "admin-test-token"),
	))
	defer server.Close()

	adminJSONRequest(t, server, http.MethodPatch, "/api/admin/v1/model-providers/deepseek", map[string]any{
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
		adminJSONRequest(t, server, http.MethodPatch, "/api/admin/v1/connectors/"+config.key, map[string]any{
			"base_url": providers.URL + config.path,
			"api_key":  config.apiKey,
		}, http.StatusOK)
	}

	scheduleBody := adminJSONRequest(t, server, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
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
		body := adminJSONRequest(t, server, http.MethodGet,
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
		server.URL+"/api/v1/collector/runs/"+item.ID,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	collectorRequest.Header.Set("Authorization", "Bearer service-test-token")
	collectorResponse, err := server.Client().Do(collectorRequest)
	if err != nil {
		t.Fatal(err)
	}
	collectorResponse.Body.Close()
	if collectorResponse.StatusCode != http.StatusOK {
		t.Fatalf("Collector status route = %d", collectorResponse.StatusCode)
	}
	callsMu.Lock()
	defer callsMu.Unlock()
	for _, path := range []string{
		"/chat/completions", "/parallel", "/tavily", "/bocha",
		"/cls", "/eastmoney-fast", "/eastmoney-stock", "/stcn",
	} {
		if calls[path] != 1 {
			t.Fatalf("Provider calls[%s]=%d, all=%#v", path, calls[path], calls)
		}
	}
}

func adminJSONRequest(
	t *testing.T,
	server *httptest.Server,
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
	response, err := server.Client().Do(request)
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
	return decoded.Bytes()
}
