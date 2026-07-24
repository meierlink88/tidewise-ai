package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/admin"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/httpapi"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/scheduling"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/testsupport"
	"github.com/jonboulle/clockwork"
)

func TestAdminConfigurationAPIManagesOnlyRegisteredRedactedTargets(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	isolatedURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "admin_configuration_http_test")
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
	service, err := admin.New(postgres.New(database), admin.Registry{
		ModelProviderKeys: []string{"deepseek"},
		ConnectorKeys:     []string{"tavily", "bocha"},
	}, "dev")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.NewAdminHandler(service, "admin-test-token"))
	defer server.Close()

	unauthorized, err := server.Client().Get(server.URL + "/api/admin/v1/model-providers")
	if err != nil {
		t.Fatal(err)
	}
	unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", unauthorized.StatusCode)
	}

	models := adminRequest(t, server, http.MethodGet, "/api/admin/v1/model-providers", nil)
	if models.status != http.StatusOK ||
		!strings.Contains(models.body, `"provider_key":"deepseek"`) ||
		!strings.Contains(models.body, `"configured":false`) {
		t.Fatalf("unconfigured models status=%d body=%s", models.status, models.body)
	}

	secret := "deepseek-admin-secret-9023"
	updated := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/model-providers/deepseek", map[string]any{
		"base_url": "https://api.deepseek.test/v1",
		"model":    "deepseek-chat",
		"api_key":  secret,
	})
	if updated.status != http.StatusOK ||
		!strings.Contains(updated.body, `"configured":true`) ||
		!strings.Contains(updated.body, `"masked_key":"***9023"`) ||
		strings.Contains(updated.body, secret) {
		t.Fatalf("updated model status=%d body=%s", updated.status, updated.body)
	}
	read := adminRequest(t, server, http.MethodGet, "/api/admin/v1/model-providers/deepseek", nil)
	if read.status != http.StatusOK || strings.Contains(read.body, secret) {
		t.Fatalf("read model status=%d body=%s", read.status, read.body)
	}
	shortSecret := "k123"
	shortKey := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/model-providers/deepseek", map[string]any{
		"api_key": shortSecret,
	})
	if shortKey.status != http.StatusOK || strings.Contains(shortKey.body, shortSecret) {
		t.Fatalf("short Model Provider Key status=%d body=%s", shortKey.status, shortKey.body)
	}
	blankKey := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/model-providers/deepseek", map[string]any{
		"api_key": "   ",
	})
	if blankKey.status != http.StatusBadRequest {
		t.Fatalf("blank Model Provider Key status=%d body=%s", blankKey.status, blankKey.body)
	}

	unknown := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/model-providers/unknown", map[string]any{
		"base_url": "https://unknown.test",
		"model":    "unknown",
		"api_key":  "unknown-secret",
	})
	if unknown.status != http.StatusNotFound {
		t.Fatalf("unknown model status=%d body=%s", unknown.status, unknown.body)
	}

	for _, baseURL := range []string{
		"http://api.tavily.test/search",
		"https://url-user:url-secret@api.tavily.test/search",
	} {
		invalidURL := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/connectors/tavily", map[string]any{
			"base_url": baseURL,
		})
		if invalidURL.status != http.StatusBadRequest ||
			strings.Contains(invalidURL.body, "url-user") ||
			strings.Contains(invalidURL.body, "url-secret") {
			t.Fatalf("invalid Connector URL %q status=%d body=%s", baseURL, invalidURL.status, invalidURL.body)
		}
	}
	loopback := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/connectors/tavily", map[string]any{
		"base_url": "http://127.0.0.1:9081/search",
	})
	if loopback.status != http.StatusOK {
		t.Fatalf("dev loopback Connector URL status=%d body=%s", loopback.status, loopback.body)
	}

	connectorSecret := "tavily-admin-secret-7788"
	connector := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/connectors/tavily", map[string]any{
		"base_url": "https://api.tavily.test/search",
		"api_key":  connectorSecret,
	})
	if connector.status != http.StatusOK ||
		!strings.Contains(connector.body, `"configured":true`) ||
		!strings.Contains(connector.body, `"key_configured":true`) ||
		!strings.Contains(connector.body, `"masked_key":"***7788"`) ||
		strings.Contains(connector.body, connectorSecret) {
		t.Fatalf("configured Connector status=%d body=%s", connector.status, connector.body)
	}
	cleared := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/connectors/tavily", map[string]any{
		"api_key": "",
	})
	if cleared.status != http.StatusOK ||
		!strings.Contains(cleared.body, `"key_configured":false`) ||
		strings.Contains(cleared.body, `"masked_key"`) {
		t.Fatalf("cleared Connector status=%d body=%s", cleared.status, cleared.body)
	}

	for _, request := range []struct {
		path string
	}{
		{path: "/api/admin/v1/model-providers/deepseek"},
		{path: "/api/admin/v1/connectors/tavily"},
	} {
		empty := adminRequest(t, server, http.MethodPatch, request.path, map[string]any{})
		if empty.status != http.StatusBadRequest {
			t.Fatalf("empty PATCH %s status=%d body=%s", request.path, empty.status, empty.body)
		}
	}
}

type adminRecordingRunner struct {
	triggers chan scheduling.Trigger
}

func (r *adminRecordingRunner) ValidateInput(context.Context, string, json.RawMessage) error {
	return nil
}

func (r *adminRecordingRunner) ConfigurationReady(context.Context, string) error {
	return nil
}

func (r *adminRecordingRunner) Trigger(_ context.Context, trigger scheduling.Trigger) error {
	r.triggers <- trigger
	return nil
}

func TestAdminScheduleAPIActivatesReplacesAndDisablesSchedule(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	isolatedURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "admin_schedule_http_test")
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
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 7, 24, 9, 59, 0, 0, location)
	clock := clockwork.NewFakeClockAt(start)
	runner := &adminRecordingRunner{triggers: make(chan scheduling.Trigger, 2)}
	scheduleService, err := scheduling.New(
		postgres.New(database),
		location,
		map[string]scheduling.AgentRunner{"collector": runner},
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
		postgres.New(database),
		admin.Registry{},
		"dev",
		admin.WithScheduleManager(scheduleService),
	)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.NewAdminHandler(adminService, "admin-test-token"))
	defer server.Close()

	emptyList := adminRequest(t, server, http.MethodGet, "/api/admin/v1/agent-schedules", nil)
	if emptyList.status != http.StatusOK || !strings.Contains(emptyList.body, `"items":[]`) {
		t.Fatalf("empty Schedule list status=%d body=%s", emptyList.status, emptyList.body)
	}
	const inputEnvelopeBytes = len(`{"value":""}`)
	exactInput := json.RawMessage(`{"value":"` + strings.Repeat("x", 64*1024-inputEnvelopeBytes) + `"}`)
	exactBoundary := adminRequest(t, server, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"agent_version":   "collector.v1",
		"schedule_type":   "cron",
		"cron_expression": "0 10 * * *",
		"input":           exactInput,
		"enabled":         false,
	})
	if exactBoundary.status != http.StatusOK {
		t.Fatalf("64 KiB Schedule Input status=%d body=%s", exactBoundary.status, exactBoundary.body)
	}
	tooLargeInput := append(json.RawMessage(nil), exactInput[:len(exactInput)-2]...)
	tooLargeInput = append(tooLargeInput, 'x', '"', '}')
	tooLarge := adminRequest(t, server, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"agent_version":   "collector.v1",
		"schedule_type":   "cron",
		"cron_expression": "0 10 * * *",
		"input":           tooLargeInput,
		"enabled":         false,
	})
	if tooLarge.status != http.StatusBadRequest {
		t.Fatalf("oversized Schedule Input status=%d body=%s", tooLarge.status, tooLarge.body)
	}
	invalidDailyTime := adminRequest(t, server, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"agent_version": "collector.v1",
		"schedule_type": "daily",
		"daily_times":   []string{"9:00"},
		"input":         map[string]any{"prompt": "采集上午资讯"},
		"enabled":       false,
	})
	if invalidDailyTime.status != http.StatusBadRequest {
		t.Fatalf("invalid Daily time status=%d body=%s", invalidDailyTime.status, invalidDailyTime.body)
	}

	missingEnabled := adminRequest(t, server, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"agent_version":   "collector.v1",
		"schedule_type":   "cron",
		"cron_expression": "0 10 * * *",
		"input":           map[string]any{"prompt": "采集上午资讯"},
	})
	if missingEnabled.status != http.StatusBadRequest {
		t.Fatalf("missing enabled status=%d body=%s", missingEnabled.status, missingEnabled.body)
	}

	created := adminRequest(t, server, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"agent_version":   "collector.v1",
		"schedule_type":   "cron",
		"cron_expression": "0 10 * * *",
		"input":           map[string]any{"prompt": "采集上午资讯"},
		"enabled":         true,
	})
	if created.status != http.StatusOK ||
		!strings.Contains(created.body, `"agent_key":"collector"`) ||
		!strings.Contains(created.body, `"next_run_at"`) {
		t.Fatalf("created Schedule status=%d body=%s", created.status, created.body)
	}
	if err := clock.BlockUntilContext(ctx, 1); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute)
	select {
	case trigger := <-runner.triggers:
		if trigger.AgentKey != "collector" || !trigger.TriggeredAt.Equal(start.Add(time.Minute)) {
			t.Fatalf("scheduled trigger = %#v", trigger)
		}
	case <-time.After(time.Second):
		t.Fatal("Admin-created Schedule did not trigger")
	}

	replaced := adminRequest(t, server, http.MethodPut, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"agent_version": "collector.v1",
		"schedule_type": "daily",
		"daily_times":   []string{"10:02", "10:01", "10:01"},
		"input":         map[string]any{"prompt": "采集固定时点资讯"},
		"enabled":       true,
	})
	if replaced.status != http.StatusOK ||
		!strings.Contains(replaced.body, `"schedule_type":"daily"`) ||
		!strings.Contains(replaced.body, `"daily_times":["10:01","10:02"]`) {
		t.Fatalf("replaced Schedule status=%d body=%s", replaced.status, replaced.body)
	}
	if err := clock.BlockUntilContext(ctx, 1); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute)
	select {
	case trigger := <-runner.triggers:
		if string(trigger.InputPayload) != `{"prompt": "采集固定时点资讯"}` ||
			!trigger.TriggeredAt.Equal(start.Add(2*time.Minute)) {
			t.Fatalf("daily trigger = %#v", trigger)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement Daily Schedule did not trigger")
	}

	disabled := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/agent-schedules/collector", map[string]any{
		"enabled": false,
	})
	if disabled.status != http.StatusOK || !strings.Contains(disabled.body, `"enabled":false`) {
		t.Fatalf("disabled Schedule status=%d body=%s", disabled.status, disabled.body)
	}
	emptyPatch := adminRequest(t, server, http.MethodPatch, "/api/admin/v1/agent-schedules/collector", map[string]any{})
	if emptyPatch.status != http.StatusBadRequest {
		t.Fatalf("empty Schedule PATCH status=%d body=%s", emptyPatch.status, emptyPatch.body)
	}
	clock.Advance(24 * time.Hour)
	select {
	case trigger := <-runner.triggers:
		t.Fatalf("disabled Schedule triggered: %#v", trigger)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestAdminExecutionListPaginatesMetadataWithoutPayloads(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	isolatedURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "admin_execution_list_http_test")
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
	firstTime := time.Date(2026, 7, 24, 1, 0, 0, 0, time.UTC)
	first, _, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "execution-list-first", Prompt: "first-private-prompt",
		CreatedAt: firstTime, AgentVersion: "collector.v1", InvocationKeys: []string{"tavily"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.FailExecutionAndIncompleteInvocations(ctx, agentrun.ExecutionFailure{
		ExecutionID: first.ID, ErrorCode: "test_failure", ErrorSummary: "Safe failure",
		StopReason: "agent_or_tool_limit", NotInvokedSummary: "Not invoked",
		CompletedAt: firstTime.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	second, _, err := store.CreateExecution(ctx, agentrun.CreateExecutionInput{
		IdempotencyKey: "execution-list-second", Prompt: "second-private-prompt",
		CreatedAt: firstTime.Add(time.Hour), AgentVersion: "collector.v1", InvocationKeys: []string{"tavily"},
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := admin.New(store, admin.Registry{}, "dev")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.NewAdminHandler(service, "admin-test-token"))
	defer server.Close()

	descending := adminRequest(t, server, http.MethodGet,
		"/api/admin/v1/agent-executions?agent_key=collector&page=1&page_size=1&sort_order=desc", nil)
	if descending.status != http.StatusOK ||
		!strings.Contains(descending.body, second.ID) ||
		strings.Contains(descending.body, first.ID) ||
		!strings.Contains(descending.body, `"total_items":2`) ||
		!strings.Contains(descending.body, `"total_pages":2`) {
		t.Fatalf("descending executions status=%d body=%s", descending.status, descending.body)
	}
	for _, forbidden := range []string{
		"first-private-prompt", "second-private-prompt",
		"prompt_sha256", "input", "artifacts", "candidate_counts", "invocations",
	} {
		if strings.Contains(descending.body, forbidden) {
			t.Fatalf("execution list exposed %q: %s", forbidden, descending.body)
		}
	}
	ascending := adminRequest(t, server, http.MethodGet,
		"/api/admin/v1/agent-executions?agent_key=collector&page=1&page_size=1&sort_order=asc", nil)
	if ascending.status != http.StatusOK || !strings.Contains(ascending.body, first.ID) {
		t.Fatalf("ascending executions status=%d body=%s", ascending.status, ascending.body)
	}
	invalid := adminRequest(t, server, http.MethodGet,
		"/api/admin/v1/agent-executions?page=0&page_size=101&sort_order=newest", nil)
	if invalid.status != http.StatusBadRequest {
		t.Fatalf("invalid pagination status=%d body=%s", invalid.status, invalid.body)
	}
}

type adminResponse struct {
	status int
	body   string
}

func adminRequest(t *testing.T, server *httptest.Server, method, path string, payload any) adminResponse {
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
	return adminResponse{status: response.StatusCode, body: decoded.String()}
}
