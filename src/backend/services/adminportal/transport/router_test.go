package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/usecase"
)

func TestHealthAndReadyEndpointsDoNotRequireAdminToken(t *testing.T) {
	router := NewRouter(testConfig(), usecase.NewService(nil), "secret")

	for _, test := range []struct {
		path       string
		wantStatus string
	}{
		{path: "/healthz", wantStatus: "ok"},
		{path: "/readyz", wantStatus: "ready"},
	} {
		t.Run(test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))

			if response.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
			}
			var body struct {
				Status      string                    `json:"status"`
				Service     string                    `json:"service"`
				Environment runtimeconfig.Environment `json:"environment"`
				Checks      map[string]string         `json:"checks"`
			}
			decodeJSON(t, response, &body)
			if body.Status != test.wantStatus || body.Service != "adminportal" || body.Environment != runtimeconfig.EnvLocal {
				t.Fatalf("response = %#v", body)
			}
			if test.path == "/readyz" && body.Checks["config"] != "ok" {
				t.Fatalf("ready checks = %v", body.Checks)
			}
		})
	}
}

func TestAdminTokenMiddlewareRejectsMissingWrongAndUnconfiguredTokenWithoutDataCalls(t *testing.T) {
	for _, test := range []struct {
		name         string
		token        string
		header       string
		wantHTTPCode int
	}{
		{name: "missing", token: "secret", wantHTTPCode: http.StatusUnauthorized},
		{name: "wrong", token: "secret", header: "Bearer wrong", wantHTTPCode: http.StatusUnauthorized},
		{name: "unconfigured", header: "Bearer secret", wantHTTPCode: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			router := NewRouter(testConfig(), usecase.NewService(countingClient(&calls)), test.token)
			request := httptest.NewRequest(http.MethodGet, "/admin/raw-documents", nil)
			if test.header != "" {
				request.Header.Set("Authorization", test.header)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantHTTPCode || calls != 0 {
				t.Fatalf("status/calls = %d/%d, want %d/0", response.Code, calls, test.wantHTTPCode)
			}
		})
	}
}

func TestRetiredSchedulerEndpointsAreAbsent(t *testing.T) {
	calls := 0
	router := NewRouter(testConfig(), usecase.NewService(countingClient(&calls)), "secret")
	for _, test := range []struct {
		method string
		path   string
		body   any
	}{
		{method: http.MethodGet, path: "/admin/scheduler/config"},
		{method: http.MethodPut, path: "/admin/scheduler/config", body: map[string]any{"invalid": true}},
		{method: http.MethodGet, path: "/admin/scheduler/runs?limit=invalid"},
	} {
		response := performJSONRequest(t, router, test.method, test.path, test.body, "secret", "")
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, want 404, body=%s", test.method, test.path, response.Code, response.Body.String())
		}
	}
	if calls != 0 {
		t.Fatalf("Data Service calls = %d, want 0", calls)
	}
}

func TestRawDocumentsAPIUsesOneDataCallAndPreservesPublicShape(t *testing.T) {
	collectedAt := testTime()
	publishedAt := collectedAt.Add(-time.Hour)
	calls := 0
	var gotQuery dataclient.RawDocumentListQuery
	var gotRequestID string
	client := &dataclient.Fake{ListRawDocumentsFunc: func(ctx context.Context, query dataclient.RawDocumentListQuery) (dataclient.RawDocumentPage, error) {
		calls++
		gotQuery = query
		gotRequestID = dataclient.RequestIDFromContext(ctx)
		return dataclient.RawDocumentPage{
			Items: []dataclient.RawDocument{{
				ID: "raw-1", SourceID: "source-1", IngestChannel: "rss_feed", SourceType: "news",
				SourceName: "示例来源", SourceURL: "https://example.com/rss.xml", Title: "央行公布金融数据",
				ContentText: "正文", ContentLevel: "full", PublishedAt: &publishedAt, CollectedAt: collectedAt,
				IngestStatus: dataclient.IngestStatusCollected,
			}},
			Total: 1, Page: 2, PageSize: 25,
		}, nil
	}}
	router := NewRouter(testConfig(), usecase.NewService(client), "secret")

	response := performJSONRequest(t, router, http.MethodGet, "/admin/raw-documents?title=央行&page=2&page_size=25", nil, "secret", "admin-request-raw")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if calls != 1 || gotQuery.Title != "央行" || gotQuery.Page != 2 || gotQuery.PageSize != 25 || gotRequestID != "admin-request-raw" {
		t.Fatalf("calls/query/request id = %d/%#v/%q", calls, gotQuery, gotRequestID)
	}
	if strings.Contains(response.Body.String(), "content_level") {
		t.Fatalf("public response exposes Data-only content_level: %s", response.Body.String())
	}
	var body rawDocumentListResponse
	decodeJSON(t, response, &body)
	if body.Total != 1 || body.Page != 2 || body.PageSize != 25 || len(body.Items) != 1 || body.Items[0].Title != "央行公布金融数据" || body.Items[0].PublishedAt != publishedAt.Format(time.RFC3339) {
		t.Fatalf("response = %#v", body)
	}
}

func TestInvalidRawPaginationReturns400WithoutDataCall(t *testing.T) {
	calls := 0
	router := NewRouter(testConfig(), usecase.NewService(countingClient(&calls)), "secret")
	response := performJSONRequest(t, router, http.MethodGet, "/admin/raw-documents?page=0", nil, "secret", "")
	if response.Code != http.StatusBadRequest || calls != 0 {
		t.Fatalf("status/calls = %d/%d, want 400/0", response.Code, calls)
	}
}

func TestEventsAPIUsesOneDataCallAndPreservesFiltersAndPublicShape(t *testing.T) {
	eventTime := testTime()
	firstSeenAt := eventTime.Add(30 * time.Minute)
	knowableAt := firstSeenAt.Add(time.Minute)
	primarySourceID := "source-1"
	calls := 0
	var gotQuery dataclient.EventListQuery
	client := &dataclient.Fake{ListEventsFunc: func(ctx context.Context, query dataclient.EventListQuery) (dataclient.EventPage, error) {
		calls++
		gotQuery = query
		if dataclient.RequestIDFromContext(ctx) != "admin-request-event" {
			t.Fatalf("request id = %q", dataclient.RequestIDFromContext(ctx))
		}
		return dataclient.EventPage{Items: []dataclient.Event{{
			ID: "event-1", Title: "美联储维持利率不变", Summary: "摘要", EventTime: &eventTime,
			FirstSeenAt: firstSeenAt, KnowableAt: &knowableAt, EventStatus: dataclient.EventStatusConfirmed,
			FactStatus: dataclient.FactStatusVerified, DedupeKey: "fed-rate-hold", PrimarySourceID: &primarySourceID,
		}}, Total: 1, Page: 1, PageSize: 50}, nil
	}}
	router := NewRouter(testConfig(), usecase.NewService(client), "secret")
	path := "/admin/events?title=美联储&event_status=confirmed&fact_status=verified&event_time_from=2026-07-09T00:00:00Z&event_time_to=2026-07-10T00:00:00Z&first_seen_from=2026-07-09T00:00:00Z&first_seen_to=2026-07-10T00:00:00Z"
	response := performJSONRequest(t, router, http.MethodGet, path, nil, "secret", "admin-request-event")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if calls != 1 || gotQuery.Title != "美联储" || gotQuery.EventStatus != dataclient.EventStatusConfirmed || gotQuery.FactStatus != dataclient.FactStatusVerified || gotQuery.EventTimeFrom == nil || gotQuery.EventTimeTo == nil || gotQuery.FirstSeenFrom == nil || gotQuery.FirstSeenTo == nil || gotQuery.Page != 1 || gotQuery.PageSize != 50 {
		t.Fatalf("calls/query = %d/%#v", calls, gotQuery)
	}
	if strings.Contains(response.Body.String(), "fact_payload") {
		t.Fatalf("public response exposes fact_payload: %s", response.Body.String())
	}
	var body eventListResponse
	decodeJSON(t, response, &body)
	if body.Total != 1 || len(body.Items) != 1 || body.Items[0].ID != "event-1" || body.Items[0].PrimarySourceID != primarySourceID || body.Items[0].KnowableAt != knowableAt.Format(time.RFC3339) {
		t.Fatalf("response = %#v", body)
	}
}

func TestSourceCatalogsAPIUsesOneDataCallWithoutExposingParser(t *testing.T) {
	calls := 0
	var gotQuery dataclient.SourceCatalogListQuery
	client := &dataclient.Fake{ListSourceCatalogsFunc: func(ctx context.Context, query dataclient.SourceCatalogListQuery) (dataclient.SourceCatalogCollection, error) {
		calls++
		gotQuery = query
		if dataclient.RequestIDFromContext(ctx) != "admin-request-source" {
			t.Fatalf("request id = %q", dataclient.RequestIDFromContext(ctx))
		}
		return dataclient.SourceCatalogCollection{Items: []dataclient.SourceCatalog{{
			ID: "source-2", IngestChannel: "rss_feed", ProviderKey: "rss", ConnectorKey: "rss",
			ParserKey: "rss_item", SourceType: "news", SourceName: "暂停 RSS", SourceURL: "https://example.com/rss.xml",
			Status: dataclient.SourceStatusInactive,
		}}}, nil
	}}
	router := NewRouter(testConfig(), usecase.NewService(client), "secret")

	response := performJSONRequest(t, router, http.MethodGet, "/admin/source-catalogs?status=inactive", nil, "secret", "admin-request-source")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if calls != 1 || gotQuery.Status != dataclient.SourceStatusInactive {
		t.Fatalf("calls/query = %d/%#v", calls, gotQuery)
	}
	if strings.Contains(response.Body.String(), "parser_key") || strings.Contains(response.Body.String(), "rss_item") {
		t.Fatalf("public response exposes parser fields: %s", response.Body.String())
	}
	var body sourceCatalogListResponse
	decodeJSON(t, response, &body)
	if len(body.Items) != 1 || body.Items[0].ID != "source-2" || body.Items[0].Status != "inactive" {
		t.Fatalf("response = %#v", body)
	}
}

func TestUnexpectedDataErrorReturnsGeneric500WithoutLeak(t *testing.T) {
	client := &dataclient.Fake{ListRawDocumentsFunc: func(context.Context, dataclient.RawDocumentListQuery) (dataclient.RawDocumentPage, error) {
		return dataclient.RawDocumentPage{}, errors.New("postgres connection secret-internal-detail")
	}}
	router := NewRouter(testConfig(), usecase.NewService(client), "secret")
	response := performJSONRequest(t, router, http.MethodGet, "/admin/raw-documents", nil, "secret", "")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if strings.Contains(response.Body.String(), "postgres") || strings.Contains(response.Body.String(), "secret-internal-detail") {
		t.Fatalf("internal error leaked: %s", response.Body.String())
	}
}

func TestAdminCORSAllowsOnlyConfiguredOriginAndHandlesPreflightBeforeAuth(t *testing.T) {
	router := NewRouter(testConfig(), usecase.NewService(countingClient(new(int))), "secret", "http://uat.example.test:9014")

	preflight := httptest.NewRequest(http.MethodOptions, "/admin/source-catalogs", nil)
	preflight.Header.Set("Origin", "http://uat.example.test:9014")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	allowed := httptest.NewRecorder()
	router.ServeHTTP(allowed, preflight)
	if allowed.Code != http.StatusNoContent || allowed.Header().Get("Access-Control-Allow-Origin") != "http://uat.example.test:9014" {
		t.Fatalf("allowed preflight = status %d, origin %q", allowed.Code, allowed.Header().Get("Access-Control-Allow-Origin"))
	}

	deniedRequest := httptest.NewRequest(http.MethodGet, "/admin/source-catalogs", nil)
	deniedRequest.Header.Set("Origin", "http://attacker.example.test")
	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, deniedRequest)
	if denied.Code != http.StatusForbidden || denied.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("denied origin = status %d, allow-origin %q", denied.Code, denied.Header().Get("Access-Control-Allow-Origin"))
	}
}

func countingClient(calls *int) *dataclient.Fake {
	return &dataclient.Fake{
		ListRawDocumentsFunc: func(context.Context, dataclient.RawDocumentListQuery) (dataclient.RawDocumentPage, error) {
			*calls++
			return dataclient.RawDocumentPage{}, nil
		},
		ListEventsFunc: func(context.Context, dataclient.EventListQuery) (dataclient.EventPage, error) {
			*calls++
			return dataclient.EventPage{}, nil
		},
		ListSourceCatalogsFunc: func(context.Context, dataclient.SourceCatalogListQuery) (dataclient.SourceCatalogCollection, error) {
			*calls++
			return dataclient.SourceCatalogCollection{}, nil
		},
	}
}

func testConfig() runtimeconfig.AppConfig {
	return runtimeconfig.AppConfig{Name: "adminportal", Env: runtimeconfig.EnvLocal}
}

func testTime() time.Time {
	return time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
}

func performJSONRequest(t *testing.T, handler http.Handler, method string, path string, body any, token string, requestID string) *httptest.ResponseRecorder {
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
	if requestID != "" {
		request.Header.Set(dataclient.RequestIDHeader, requestID)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, response.Body.String())
	}
}
