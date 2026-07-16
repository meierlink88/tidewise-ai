package adminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestHealthAndReadyEndpointsDoNotRequireAdminToken(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	router := NewRouter(repo, "secret")

	for _, item := range []struct {
		path      string
		wantField string
		wantValue string
	}{
		{path: "/healthz", wantField: "status", wantValue: "ok"},
		{path: "/readyz", wantField: "status", wantValue: "ready"},
	} {
		t.Run(item.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, item.path, nil)
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
			}

			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body[item.wantField] != item.wantValue {
				t.Fatalf("%s = %q, want %q", item.wantField, body[item.wantField], item.wantValue)
			}
		})
	}
}

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

func TestRawDocumentsAPIListsPagedDocumentsWithTitleSearch(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	collectedAt := testTime()
	for _, doc := range []domain.RawDocument{
		testRawDocument("raw-1", "央行公布金融数据", collectedAt.Add(2*time.Minute)),
		testRawDocument("raw-2", "美联储维持利率不变", collectedAt.Add(time.Minute)),
		testRawDocument("raw-3", "央行开展逆回购操作", collectedAt),
	} {
		if _, err := repo.UpsertRawDocument(context.Background(), doc); err != nil {
			t.Fatalf("UpsertRawDocument(%s) error = %v", doc.ID, err)
		}
	}
	router := NewRouter(repo, "secret")

	response := performJSONRequest(t, router, http.MethodGet, "/admin/raw-documents?title=央行&page=1", nil, "secret")
	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200, body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Items []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			CollectedAt string `json:"collected_at"`
		} `json:"items"`
		Total    int `json:"total"`
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode raw documents response: %v", err)
	}
	if payload.Total != 2 || payload.Page != 1 || payload.PageSize != 50 {
		t.Fatalf("pagination = total %d page %d page_size %d, want 2/1/50", payload.Total, payload.Page, payload.PageSize)
	}
	if got := documentTitles(payload.Items); !strings.Contains(strings.Join(got, ","), "央行公布金融数据") || !strings.Contains(strings.Join(got, ","), "央行开展逆回购操作") {
		t.Fatalf("raw document titles = %v, want only matching titles", got)
	}
	if payload.Items[0].Title != "央行公布金融数据" {
		t.Fatalf("first title = %q, want newest matching document first", payload.Items[0].Title)
	}
}

func TestRawDocumentsAPIRequiresAdminToken(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	router := NewRouter(repo, "secret")

	request := httptest.NewRequest(http.MethodGet, "/admin/raw-documents", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %d, want 401", response.Code)
	}
}

func TestEventsAPIListsFilteredEvents(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	eventTime := testTime()
	firstSeenAt := eventTime.Add(30 * time.Minute)
	for _, event := range []domain.Event{
		{
			ID:          "event-1",
			Title:       "美联储维持利率不变",
			Summary:     "FOMC 会议维持联邦基金利率目标区间不变。",
			EventTime:   &eventTime,
			FirstSeenAt: firstSeenAt,
			EventStatus: domain.EventStatusConfirmed,
			FactStatus:  domain.FactStatusVerified,
			DedupeKey:   "fed-rate-hold",
			FactPayload: domain.FactPayload{"policy_rate": map[string]any{"value": 3.5}},
		},
		{
			ID:          "event-2",
			Title:       "欧洲央行释放政策信号",
			Summary:     "欧洲央行官员发表讲话。",
			EventTime:   ptrTime(eventTime.Add(-48 * time.Hour)),
			FirstSeenAt: firstSeenAt.Add(-47 * time.Hour),
			EventStatus: domain.EventStatusCandidate,
			FactStatus:  domain.FactStatusUnverified,
			DedupeKey:   "ecb-policy-signal",
			FactPayload: domain.FactPayload{},
		},
	} {
		if err := repo.SeedEvent(context.Background(), event); err != nil {
			t.Fatalf("SeedEvent(%s) error = %v", event.ID, err)
		}
	}
	router := NewRouter(repo, "secret")

	path := "/admin/events?title=美联储&event_status=confirmed&fact_status=verified&event_time_from=2026-07-09T00:00:00Z&event_time_to=2026-07-10T00:00:00Z&first_seen_from=2026-07-09T00:00:00Z&first_seen_to=2026-07-10T00:00:00Z"
	response := performJSONRequest(t, router, http.MethodGet, path, nil, "secret")
	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200, body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "fact_payload") {
		t.Fatalf("response body unexpectedly exposes fact_payload: %s", response.Body.String())
	}
	var payload struct {
		Items []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			EventStatus string `json:"event_status"`
			FactStatus  string `json:"fact_status"`
		} `json:"items"`
		Total    int `json:"total"`
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode events response: %v", err)
	}
	if payload.Total != 1 || payload.Page != 1 || payload.PageSize != 50 {
		t.Fatalf("pagination = total %d page %d page_size %d, want 1/1/50", payload.Total, payload.Page, payload.PageSize)
	}
	if len(payload.Items) != 1 || payload.Items[0].ID != "event-1" {
		t.Fatalf("event items = %+v, want only event-1", payload.Items)
	}
}

func TestSourceCatalogsAPIListsStatusWithoutParser(t *testing.T) {
	repo := repositories.NewInMemoryRepository([]domain.SourceCatalog{
		{
			ID:            "source-1",
			IngestChannel: "ai_web_research",
			ProviderKey:   "llm_web_research",
			ConnectorKey:  "ai_web_research",
			ParserKey:     "internal-parser",
			SourceType:    "news",
			SourceName:    "AI 全球政经搜索",
			SourceURL:     "https://example.com/ai",
			Status:        domain.SourceCatalogStatusActive,
		},
		{
			ID:            "source-2",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			ConnectorKey:  "rss",
			ParserKey:     "rss_item",
			SourceType:    "news",
			SourceName:    "暂停 RSS",
			SourceURL:     "https://example.com/rss.xml",
			Status:        domain.SourceCatalogStatusInactive,
		},
	})
	router := NewRouter(repo, "secret")

	response := performJSONRequest(t, router, http.MethodGet, "/admin/source-catalogs?status=inactive", nil, "secret")
	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200, body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "parser_key") || strings.Contains(body, "rss_item") {
		t.Fatalf("source catalog response must not expose parser fields, body=%s", body)
	}
	var payload struct {
		Items []struct {
			ID            string `json:"id"`
			ProviderKey   string `json:"provider_key"`
			IngestChannel string `json:"ingest_channel"`
			SourceType    string `json:"source_type"`
			SourceName    string `json:"source_name"`
			SourceURL     string `json:"source_url"`
			Status        string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode source catalogs response: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].ID != "source-2" {
		t.Fatalf("source items = %+v, want only inactive source-2", payload.Items)
	}
}

func TestSchedulerRunsAPILimitsToRecent50AndReturnsSourceCounts(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	base := testTime()
	for index := 0; index < 55; index++ {
		started := base.Add(time.Duration(index) * time.Minute)
		finished := started.Add(30 * time.Second)
		run := domain.IngestionRun{
			ID:               "run-" + strconv.Itoa(index),
			TriggerType:      domain.SchedulerTriggerInterval,
			Status:           domain.SchedulerRunStatusSucceeded,
			StartedAt:        started,
			FinishedAt:       &finished,
			TotalSources:     3,
			SucceededSources: 2,
			FailedSources:    1,
			SkippedSources:   0,
		}
		if _, err := repo.CreateIngestionRun(context.Background(), run); err != nil {
			t.Fatalf("CreateIngestionRun(%s) error = %v", run.ID, err)
		}
	}
	router := NewRouter(repo, "secret")

	response := performJSONRequest(t, router, http.MethodGet, "/admin/scheduler/runs?limit=50", nil, "secret")
	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", response.Code)
	}
	var runs []schedulerRunResponse
	if err := json.Unmarshal(response.Body.Bytes(), &runs); err != nil {
		t.Fatalf("decode runs response: %v", err)
	}
	if len(runs) != 50 {
		t.Fatalf("runs length = %d, want 50", len(runs))
	}
	if runs[0].ID != "run-54" || runs[49].ID != "run-5" {
		t.Fatalf("run order = first %q last %q, want run-54/run-5", runs[0].ID, runs[49].ID)
	}
	if runs[0].TotalSources != 3 || runs[0].SucceededSources != 2 || runs[0].FailedSources != 1 || runs[0].SkippedSources != 0 {
		t.Fatalf("run source counts = %+v, want 3/2/1/0", runs[0])
	}
}

func testTime() time.Time {
	return time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func testRawDocument(id string, title string, collectedAt time.Time) domain.RawDocument {
	return domain.RawDocument{
		ID:            id,
		SourceID:      "source-1",
		IngestChannel: "rss_feed",
		SourceType:    "news",
		SourceName:    "示例来源",
		SourceURL:     "https://example.com/rss.xml",
		Title:         title,
		ContentText:   title + "正文",
		ContentHash:   id + "-hash",
		CollectedAt:   collectedAt,
		IngestStatus:  domain.IngestStatusCollected,
	}
}

func documentTitles(items []struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	CollectedAt string `json:"collected_at"`
}) []string {
	titles := make([]string, 0, len(items))
	for _, item := range items {
		titles = append(titles, item.Title)
	}
	return titles
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
