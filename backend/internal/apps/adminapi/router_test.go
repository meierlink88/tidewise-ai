package adminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestAdminTokenMiddlewareRejectsMissingWrongAndUnconfiguredToken(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)

	for _, item := range []struct {
		name         string
		token        string
		header       string
		wantHTTPCode int
	}{
		{name: "missing", token: "secret", header: "", wantHTTPCode: http.StatusUnauthorized},
		{name: "wrong", token: "secret", header: "Bearer wrong", wantHTTPCode: http.StatusUnauthorized},
		{name: "unconfigured", token: "", header: "Bearer secret", wantHTTPCode: http.StatusServiceUnavailable},
	} {
		t.Run(item.name, func(t *testing.T) {
			router := NewRouter(repo, item.token)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/admin/scheduler/config", nil)
			if item.header != "" {
				request.Header.Set("Authorization", item.header)
			}
			router.ServeHTTP(response, request)
			if response.Code != item.wantHTTPCode {
				t.Fatalf("status code = %d, want %d", response.Code, item.wantHTTPCode)
			}
		})
	}
}

func TestSchedulerConfigAPI(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	router := NewRouter(repo, "secret")
	body := schedulerConfigRequest{
		Enabled:         true,
		Mode:            string(domain.SchedulerModeFixedTimes),
		IntervalMinutes: 0,
		FixedTimes:      []string{"09:00", "12:00", "15:00", "18:00", "21:00"},
		Concurrency:     2,
		BatchSize:       20,
		TimeoutSeconds:  180,
		SourceFilter: schedulerSourceFilterRequest{
			ProviderKey:   "llm_web_research",
			IngestChannel: "ai_web_research",
			SourceType:    "news",
		},
		Timezone: "Asia/Shanghai",
	}

	response := performJSONRequest(t, router, http.MethodPut, "/admin/scheduler/config", body, "secret")
	if response.Code != http.StatusOK {
		t.Fatalf("PUT status code = %d, want 200, body=%s", response.Code, response.Body.String())
	}

	response = performJSONRequest(t, router, http.MethodGet, "/admin/scheduler/config", nil, "secret")
	if response.Code != http.StatusOK {
		t.Fatalf("GET status code = %d, want 200", response.Code)
	}
	var config schedulerConfigResponse
	if err := json.Unmarshal(response.Body.Bytes(), &config); err != nil {
		t.Fatalf("decode config response: %v", err)
	}
	if !config.Enabled || config.Mode != string(domain.SchedulerModeFixedTimes) {
		t.Fatalf("config response = %+v", config)
	}
	if config.SourceFilter.ProviderKey != "llm_web_research" {
		t.Fatalf("ProviderKey = %q", config.SourceFilter.ProviderKey)
	}
}

func TestSchedulerConfigAPIRejectsInvalidPayload(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	router := NewRouter(repo, "secret")
	body := schedulerConfigRequest{
		Enabled:        true,
		Mode:           string(domain.SchedulerModeFixedTimes),
		FixedTimes:     []string{"09:00", "09:00"},
		Concurrency:    1,
		BatchSize:      10,
		TimeoutSeconds: 180,
		Timezone:       "Asia/Shanghai",
	}

	response := performJSONRequest(t, router, http.MethodPut, "/admin/scheduler/config", body, "secret")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want 400", response.Code)
	}
}

func TestSchedulerRunsAPI(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	started := testTime()
	if _, err := repo.CreateIngestionRun(context.Background(), domain.IngestionRun{
		ID:          "run-1",
		TriggerType: domain.SchedulerTriggerManualOnce,
		Status:      domain.SchedulerRunStatusRunning,
		StartedAt:   started,
	}); err != nil {
		t.Fatalf("CreateIngestionRun() error = %v", err)
	}
	run := domain.IngestionRun{
		ID:               "run-1",
		TriggerType:      domain.SchedulerTriggerManualOnce,
		Status:           domain.SchedulerRunStatusSucceeded,
		StartedAt:        started,
		TotalSources:     1,
		SucceededSources: 1,
	}
	if err := repo.CompleteIngestionRun(context.Background(), run); err != nil {
		t.Fatalf("CompleteIngestionRun() error = %v", err)
	}
	router := NewRouter(repo, "secret")

	response := performJSONRequest(t, router, http.MethodGet, "/admin/scheduler/runs?limit=5", nil, "secret")
	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", response.Code)
	}
	var runs []schedulerRunResponse
	if err := json.Unmarshal(response.Body.Bytes(), &runs); err != nil {
		t.Fatalf("decode runs response: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != string(domain.SchedulerRunStatusSucceeded) {
		t.Fatalf("runs response = %+v", runs)
	}
}

func testTime() time.Time {
	return time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
}

func performJSONRequest(t *testing.T, handler http.Handler, method string, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var content bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&content).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	request := httptest.NewRequest(method, path, &content)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
