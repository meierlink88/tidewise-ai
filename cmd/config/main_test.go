package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector"
	collectorapp "github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector/usecase"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform/admin"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/conf"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/artifacts"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/connectors"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/modelprovider/deepseek"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/postgres"
	serverlayer "github.com/guanchaojia/tidewise-ai-agentrun/internal/server"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/service"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/testsupport"
)

type readyComponent struct{}

func (readyComponent) Ready(context.Context) error { return nil }

type memoryConfigurationStore struct {
	models     map[string]agentrun.ModelProviderConfig
	connectors map[string]agentrun.ConnectorConfig
}

func (s *memoryConfigurationStore) UpsertModelProviderConfig(_ context.Context, config agentrun.ModelProviderConfig) error {
	if s.models == nil {
		s.models = make(map[string]agentrun.ModelProviderConfig)
	}
	s.models[config.ProviderKey] = config
	return nil
}

func (s *memoryConfigurationStore) LoadModelProviderConfigs(context.Context) (map[string]agentrun.ModelProviderConfig, error) {
	return s.models, nil
}

func (s *memoryConfigurationStore) ListModelProviderConfigViews(context.Context) ([]agentrun.ModelProviderConfigView, error) {
	var views []agentrun.ModelProviderConfigView
	for _, config := range s.models {
		views = append(views, agentrun.ModelProviderConfigView{
			ProviderKey:   config.ProviderKey,
			BaseURL:       config.BaseURL,
			Model:         config.Model,
			KeyConfigured: config.APIKey != "",
		})
	}
	return views, nil
}

func (s *memoryConfigurationStore) UpsertConnectorConfig(_ context.Context, config agentrun.ConnectorConfig) error {
	if s.connectors == nil {
		s.connectors = make(map[string]agentrun.ConnectorConfig)
	}
	s.connectors[config.ConnectorKey] = config
	return nil
}

func (s *memoryConfigurationStore) LoadConnectorConfigs(context.Context) (map[string]agentrun.ConnectorConfig, error) {
	return s.connectors, nil
}

func (s *memoryConfigurationStore) ListConnectorConfigViews(context.Context) ([]agentrun.ConnectorConfigView, error) {
	var views []agentrun.ConnectorConfigView
	for _, config := range s.connectors {
		views = append(views, agentrun.ConnectorConfigView{
			ConnectorKey:  config.ConnectorKey,
			BaseURL:       config.BaseURL,
			KeyConfigured: config.APIKey != "",
		})
	}
	return views, nil
}

func TestRunConfigurationCommandSeparatesModelAndConnectorResources(t *testing.T) {
	store := &memoryConfigurationStore{}
	var output bytes.Buffer

	err := runConfigurationCommand(
		context.Background(),
		store,
		"dev",
		[]string{"model", "set", "--provider", "deepseek", "--base-url", "https://deepseek.test", "--model", "deepseek-chat", "--api-key-stdin"},
		strings.NewReader("model-secret\n"),
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}
	model := store.models["deepseek"]
	if model.Model != "deepseek-chat" || model.APIKey != "model-secret" {
		t.Fatalf("Model Provider Configuration = %#v", model)
	}
	if !strings.Contains(output.String(), "next Agent Execution") ||
		strings.Contains(output.String(), "restart AgentRun") ||
		strings.Contains(output.String(), "model-secret") {
		t.Fatalf("model set output = %q", output.String())
	}

	output.Reset()
	err = runConfigurationCommand(
		context.Background(),
		store,
		"dev",
		[]string{"connector", "set", "--connector", "tavily", "--base-url", "https://tavily.test"},
		strings.NewReader(""),
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}
	connector := store.connectors["tavily"]
	if connector.BaseURL != "https://tavily.test" || connector.APIKey != "" {
		t.Fatalf("Connector Configuration = %#v", connector)
	}
	if !strings.Contains(output.String(), "next Agent Execution") ||
		strings.Contains(output.String(), "restart AgentRun") {
		t.Fatalf("connector set output = %q", output.String())
	}
}

func TestRunConfigurationCommandListsEachResourceAndChecksCombinedReadiness(t *testing.T) {
	store := &memoryConfigurationStore{
		models: map[string]agentrun.ModelProviderConfig{
			"deepseek": {
				ProviderKey: "deepseek",
				BaseURL:     "https://deepseek.test",
				Model:       "deepseek-chat",
				APIKey:      "model-secret",
			},
		},
		connectors: map[string]agentrun.ConnectorConfig{},
	}
	for _, key := range []string{
		"parallel_search", "tavily", "bocha", "cls_telegraph",
		"eastmoney_fastnews", "eastmoney_stock_news", "stcn_quicknews",
	} {
		store.connectors[key] = agentrun.ConnectorConfig{
			ConnectorKey: key,
			BaseURL:      "https://" + key + ".test",
		}
	}

	var output bytes.Buffer
	if err := runConfigurationCommand(context.Background(), store, "dev", []string{"model", "list"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"provider_key": "deepseek"`) || strings.Contains(output.String(), "model-secret") {
		t.Fatalf("model list output = %q", output.String())
	}

	output.Reset()
	if err := runConfigurationCommand(context.Background(), store, "dev", []string{"connector", "list"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"connector_key"`) || strings.Contains(output.String(), `"model"`) {
		t.Fatalf("connector list output = %q", output.String())
	}

	output.Reset()
	if err := runConfigurationCommand(context.Background(), store, "dev", []string{"check"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "Collector configuration is ready\n" {
		t.Fatalf("check output = %q", output.String())
	}
}

func TestHistoricalConfigurationUpgradeRunsThroughCLIReadinessAndCollectorHTTP(t *testing.T) {
	baseDatabaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if baseDatabaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	databaseURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, baseDatabaseURL, "config_cli_upgrade_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	providers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/chat/completions":
			_, _ = writer.Write([]byte(`{"id":"chat-upgrade","object":"chat.completion","created":1,"model":"deepseek-chat","choices":[{"index":0,"message":{"role":"assistant","content":"{\"queries\":[\"中国市场\"],\"combined_query\":\"中国市场\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
		case "/parallel", "/tavily":
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

	if _, err := database.Exec(ctx, `
		CREATE TABLE schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatal(err)
	}
	migrationRoot := filepath.Join("..", "..", "internal", "data", "postgres", "migrations")
	for _, name := range []string{
		"001_agent_registry.sql",
		"002_executions.sql",
		"003_provider_configs.sql",
		"004_collector_audit_publication.sql",
	} {
		payload, err := os.ReadFile(filepath.Join(migrationRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(ctx, string(payload)); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, "migrations/"+name); err != nil {
			t.Fatal(err)
		}
	}
	oldConfigurations := [][]string{
		{"deepseek", providers.URL, "deepseek-chat", "deepseek-test-key"},
		{"parallel_search", providers.URL + "/parallel", "", "parallel-test-key"},
		{"tavily", providers.URL + "/tavily", "", "tavily-test-key"},
		{"bocha", providers.URL + "/bocha", "", "bocha-test-key"},
		{"cls_telegraph", providers.URL + "/cls", "", ""},
		{"eastmoney_fastnews", providers.URL + "/eastmoney-fast", "", ""},
		{"eastmoney_stock_news", providers.URL + "/eastmoney-stock", "", ""},
		{"stcn_quicknews", providers.URL + "/stcn", "", ""},
	}
	for _, config := range oldConfigurations {
		if _, err := database.Exec(ctx, `
			INSERT INTO provider_configs (provider_key, base_url, model, api_key)
			VALUES ($1, $2, $3, $4)
		`, config[0], config[1], config[2], config[3]); err != nil {
			t.Fatal(err)
		}
	}
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatalf("upgrade historical configuration: %v", err)
	}

	store := postgres.New(database)
	var output bytes.Buffer
	if err := runConfigurationCommand(ctx, store, "dev", []string{"model", "list"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"provider_key": "deepseek"`) ||
		strings.Contains(output.String(), "deepseek-test-key") {
		t.Fatalf("upgraded model list = %q", output.String())
	}
	output.Reset()
	if err := runConfigurationCommand(ctx, store, "dev", []string{"connector", "list"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"connector_key": "tavily"`) ||
		strings.Contains(output.String(), "tavily-test-key") {
		t.Fatalf("upgraded Connector list = %q", output.String())
	}
	output.Reset()
	if err := runConfigurationCommand(ctx, store, "dev", []string{"check"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}

	artifactStore := artifacts.Store{Root: t.TempDir(), Publications: store, Now: time.Now}
	application, err := collectorapp.New(
		store,
		deepseek.Factory{},
		connectors.Factory{},
		artifactStore,
		collectorapp.WithEnvironment("dev"),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = application.Shutdown(shutdownContext)
	})
	adminService, err := admin.New(store, admin.Registry{
		ModelProviderKeys: []string{collector.ModelProviderDeepSeek},
		ConnectorKeys:     collector.ConnectorKeys(),
	}, "dev")
	if err != nil {
		t.Fatal(err)
	}
	apiService, err := service.NewAgentRunService(application, adminService, readyComponent{})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := serverlayer.NewHTTPServer(conf.Config{
		App:     conf.AppConfig{Name: conf.ServiceName, Env: conf.EnvDev},
		Server:  conf.ServerConfig{Host: "127.0.0.1", Port: conf.ServicePort},
		Secrets: conf.SecretConfig{ServiceToken: "service-test-token", AdminToken: "admin-test-token"},
	}, apiService, apiService, slog.New(slog.NewTextHandler(io.Discard, nil)))
	runtimeServer := httptest.NewServer(httpServer)
	defer runtimeServer.Close()
	ready, err := http.Get(runtimeServer.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	ready.Body.Close()
	if ready.StatusCode != http.StatusOK {
		t.Fatalf("upgraded readiness = %d, want 200", ready.StatusCode)
	}

	body := strings.NewReader(`{"prompt":"采集中国市场资讯"}`)
	request, _ := http.NewRequest(http.MethodPost, runtimeServer.URL+"/api/v1/collector/runs", body)
	request.Header.Set("Authorization", "Bearer service-test-token")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", fmt.Sprintf("upgrade-e2e-%d", time.Now().UnixNano()))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		Result struct {
			ExecutionID string `json:"execution_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusAccepted || created.Result.ExecutionID == "" {
		t.Fatalf("upgraded POST status=%d execution_id=%q", response.StatusCode, created.Result.ExecutionID)
	}

	deadline := time.Now().Add(8 * time.Second)
	for {
		statusRequest, _ := http.NewRequest(
			http.MethodGet,
			runtimeServer.URL+"/api/v1/collector/runs/"+created.Result.ExecutionID,
			nil,
		)
		statusRequest.Header.Set("Authorization", "Bearer service-test-token")
		statusResponse, err := http.DefaultClient.Do(statusRequest)
		if err != nil {
			t.Fatal(err)
		}
		var snapshot struct {
			Result struct {
				Status      string `json:"status"`
				Invocations []struct {
					Status string `json:"status"`
				} `json:"invocations"`
			} `json:"result"`
		}
		if err := json.NewDecoder(statusResponse.Body).Decode(&snapshot); err != nil {
			statusResponse.Body.Close()
			t.Fatal(err)
		}
		statusResponse.Body.Close()
		if snapshot.Result.Status == "succeeded_no_change" {
			if len(snapshot.Result.Invocations) != 7 {
				t.Fatalf("upgraded run Invocations = %d, want 7", len(snapshot.Result.Invocations))
			}
			for _, invocation := range snapshot.Result.Invocations {
				if invocation.Status != "completed" {
					t.Fatalf("upgraded run Invocation = %#v", invocation)
				}
			}
			break
		}
		if snapshot.Result.Status == "failed" || snapshot.Result.Status == "partially_succeeded" || snapshot.Result.Status == "succeeded" {
			t.Fatalf("upgraded run terminal status = %q", snapshot.Result.Status)
		}
		if time.Now().After(deadline) {
			t.Fatalf("upgraded run did not finish: %q", snapshot.Result.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
