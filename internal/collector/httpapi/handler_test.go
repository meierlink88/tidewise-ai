package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	collectorapp "github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/application"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/httpapi"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/testsupport"
)

func TestCollectorRunCompletesThroughHTTPWithSevenDirectConnectors(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	isolatedURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "httpapi_success_test")
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
	if err := store.FailStaleExecutions(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	var callsMu sync.Mutex
	calls := map[string]int{}
	recordCall := func(path string) {
		callsMu.Lock()
		calls[path]++
		callsMu.Unlock()
	}
	providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		recordCall(request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/chat/completions":
			_, _ = writer.Write([]byte(`{"id":"chat-1","object":"chat.completion","created":1,"model":"deepseek-chat","choices":[{"index":0,"message":{"role":"assistant","content":"{\"queries\":[\"中国半导体政策\",\"芯片供应链价格\"],\"combined_query\":\"中国半导体政策 芯片供应链价格\",\"time_window_hours\":168}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":10,"total_tokens":20}}`))
		case "/parallel":
			_, _ = writer.Write([]byte(`{"results":[{"url":"https://example.com/parallel","title":"Parallel 结果","excerpts":["Parallel 直接片段"]}]}`))
		case "/tavily":
			_, _ = writer.Write([]byte(`{"results":[{"url":"https://example.com/tavily","title":"Tavily 结果","content":"Tavily 直接片段"}]}`))
		case "/bocha":
			_, _ = writer.Write([]byte(`{"data":{"webPages":{"value":[{"name":"Bocha 结果","url":"https://example.com/bocha","summary":"Bocha 直接摘要","siteName":"example.com"}]}}}`))
		case "/cls":
			_, _ = fmt.Fprintf(writer, `{"data":{"roll_data":[{"id":101,"ctime":%d,"title":"财联社结果","content":"财联社直接内容"}]}}`, now.Unix())
		case "/eastmoney-fast":
			_, _ = writer.Write([]byte(`{"data":{"fastNewsList":[{"code":"202607220001","title":"东方财富快讯","summary":"东方财富直接摘要"}]}}`))
		case "/eastmoney-stock":
			_, _ = writer.Write([]byte(`callback({"result":{"cmsArticleWebOld":[{"code":"EM001","title":"东方财富个股新闻","url":"https://example.com/eastmoney-stock","content":"东方财富个股直接片段","mediaName":"证券时报"}]}})`))
		case "/stcn":
			_, _ = fmt.Fprintf(writer, `{"state":1,"data":[{"id":"4022599","url":"/article/detail/4022599.html","title":"证券时报快讯","source":"人民财讯","time":%d,"content":"证券时报直接内容"}]}`, now.UnixMilli())
		default:
			http.NotFound(writer, request)
		}
	}))
	defer providers.Close()

	configs := []collector.ProviderConfig{
		{Key: "deepseek", BaseURL: providers.URL, Model: "deepseek-chat", APIKey: "deepseek-test-key"},
		{Key: "parallel_search", BaseURL: providers.URL + "/parallel", APIKey: "parallel-test-key"},
		{Key: "tavily", BaseURL: providers.URL + "/tavily", APIKey: "tavily-test-key"},
		{Key: "bocha", BaseURL: providers.URL + "/bocha", APIKey: "bocha-test-key"},
		{Key: "cls_telegraph", BaseURL: providers.URL + "/cls"},
		{Key: "eastmoney_fastnews", BaseURL: providers.URL + "/eastmoney-fast"},
		{Key: "eastmoney_stock_news", BaseURL: providers.URL + "/eastmoney-stock"},
		{Key: "stcn_quicknews", BaseURL: providers.URL + "/stcn"},
	}
	for _, config := range configs {
		if err := store.UpsertProviderConfig(ctx, config); err != nil {
			t.Fatal(err)
		}
	}

	application, err := collectorapp.New(store, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.NewHandler(application, "service-test-token"))
	defer server.Close()

	body := []byte(`{"prompt":"采集最近一周中国半导体产业链的政策、供需和价格变化。"}`)
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/internal/agent-run/v1/collector/runs", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer service-test-token")
	request.Header.Set("Idempotency-Key", fmt.Sprintf("http-e2e-%d", time.Now().UnixNano()))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("POST status = %d", response.StatusCode)
	}
	var created struct {
		Schema      string `json:"schema"`
		ExecutionID string `json:"execution_id"`
		Status      string `json:"status"`
		StatusURL   string `json:"status_url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Schema != "collector_run.v1" || created.ExecutionID == "" || created.Status != "queued" {
		t.Fatalf("created response = %#v", created)
	}

	var observed struct {
		Schema      string `json:"schema"`
		ExecutionID string `json:"execution_id"`
		Status      string `json:"status"`
		Invocations []struct {
			ConnectorKey string `json:"connector_key"`
			Status       string `json:"status"`
		} `json:"invocations"`
		CandidateCounts map[string]int    `json:"candidate_counts"`
		Artifacts       map[string]string `json:"artifacts"`
	}
	deadline := time.Now().Add(8 * time.Second)
	for {
		statusRequest, _ := http.NewRequest(http.MethodGet, server.URL+created.StatusURL, nil)
		statusRequest.Header.Set("Authorization", "Bearer service-test-token")
		statusResponse, requestErr := http.DefaultClient.Do(statusRequest)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		observed = struct {
			Schema      string `json:"schema"`
			ExecutionID string `json:"execution_id"`
			Status      string `json:"status"`
			Invocations []struct {
				ConnectorKey string `json:"connector_key"`
				Status       string `json:"status"`
			} `json:"invocations"`
			CandidateCounts map[string]int    `json:"candidate_counts"`
			Artifacts       map[string]string `json:"artifacts"`
		}{}
		decodeErr := json.NewDecoder(statusResponse.Body).Decode(&observed)
		statusResponse.Body.Close()
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if observed.Status == "succeeded" || observed.Status == "succeeded_no_change" || observed.Status == "partially_succeeded" || observed.Status == "failed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("execution did not finish: %#v", observed)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if observed.Status != "succeeded" || len(observed.Invocations) != 7 || observed.CandidateCounts["raw_results"] != 7 || observed.CandidateCounts["results_terminal"] != 7 || observed.CandidateCounts["results_pending"] != 0 || observed.CandidateCounts["accepted"] < 1 {
		t.Fatalf("completed response = %#v", observed)
	}
	if _, err := os.Stat(observed.Artifacts["manifest"]); err != nil {
		t.Fatalf("manifest is not readable: %v", err)
	}
	if filepath.Base(observed.Artifacts["manifest"]) != "manifest.json" {
		t.Fatalf("manifest path = %q", observed.Artifacts["manifest"])
	}
	callsMu.Lock()
	defer callsMu.Unlock()
	for _, path := range []string{"/chat/completions", "/parallel", "/tavily", "/bocha", "/cls", "/eastmoney-fast", "/eastmoney-stock", "/stcn"} {
		if calls[path] != 1 {
			t.Fatalf("calls[%s] = %d, want 1; all=%#v", path, calls[path], calls)
		}
	}
}
