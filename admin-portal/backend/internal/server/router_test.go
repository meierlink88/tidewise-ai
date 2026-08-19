package server

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

	v1 "github.com/meierlink88/tidewise-ai/admin-portal/backend/api/admin/v1"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/data"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/service"
)

func TestHealthAndReadyEndpointsDoNotRequireAdminToken(t *testing.T) {
	router := NewRouter(testConfig(), biz.NewService(nil, nil), "secret")

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
				Status      string            `json:"status"`
				Service     string            `json:"service"`
				Environment conf.Environment  `json:"environment"`
				Checks      map[string]string `json:"checks"`
			}
			decodeJSON(t, response, &body)
			if body.Status != test.wantStatus || body.Service != "adminportal" || body.Environment != conf.EnvLocal {
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
			router := NewRouter(testConfig(), biz.NewService(countingClient(&calls), nil), test.token)
			request := httptest.NewRequest(http.MethodGet, "/api/admin/v1/events", nil)
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
	router := NewRouter(testConfig(), biz.NewService(countingClient(&calls), nil), "secret")
	for _, test := range []struct {
		method string
		path   string
		body   any
	}{
		{method: http.MethodGet, path: "/api/admin/v1/scheduler/config"},
		{method: http.MethodPut, path: "/api/admin/v1/scheduler/config", body: map[string]any{"invalid": true}},
		{method: http.MethodGet, path: "/api/admin/v1/scheduler/runs?limit=invalid"},
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

func TestEventsAPIUsesOneDataCallAndPreservesFiltersAndPublicShape(t *testing.T) {
	eventTime := testTime()
	announcedAt := eventTime.Add(30 * time.Minute)
	what := "维持利率不变"
	calls := 0
	var gotQuery biz.EventListQuery
	client := &biz.FakeDataServiceRepo{ListEventsFunc: func(ctx context.Context, query biz.EventListQuery) (biz.EventPage, error) {
		calls++
		gotQuery = query
		if data.RequestIDFromContext(ctx) != "admin-request-event" {
			t.Fatalf("request id = %q", data.RequestIDFromContext(ctx))
		}
		return biz.EventPage{Items: []biz.Event{{
			ID: "EVT00000000-0000-5000-8000-000000000001", Title: "美联储维持利率不变", Summary: "摘要",
			Semantic: biz.EventSemantic{What: &what}, Modality: biz.EventModalityFact,
			OccurredAt: &eventTime, AnnouncedAt: &announcedAt, Status: biz.EventLifecycleActive,
		}}, Total: 1, Page: 1, PageSize: 50}, nil
	}}
	router := NewRouter(testConfig(), biz.NewService(client, nil), "secret")
	path := "/api/admin/v1/events?title=美联储&modality=FACT&status=ACTIVE&occurred_from=2026-07-09T00:00:00Z&occurred_to=2026-07-10T00:00:00Z&announced_from=2026-07-09T00:00:00Z&announced_to=2026-07-10T00:00:00Z"
	response := performJSONRequest(t, router, http.MethodGet, path, nil, "secret", "admin-request-event")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if calls != 1 || gotQuery.Title != "美联储" || gotQuery.Modality != biz.EventModalityFact || gotQuery.Status != biz.EventLifecycleActive || gotQuery.OccurredFrom == nil || gotQuery.OccurredTo == nil || gotQuery.AnnouncedFrom == nil || gotQuery.AnnouncedTo == nil || gotQuery.Page != 1 || gotQuery.PageSize != 50 {
		t.Fatalf("calls/query = %d/%#v", calls, gotQuery)
	}
	if strings.Contains(response.Body.String(), "fact_payload") {
		t.Fatalf("public response exposes fact_payload: %s", response.Body.String())
	}
	var envelope struct {
		RequestID string               `json:"request_id"`
		Result    v1.EventListResponse `json:"result"`
	}
	decodeJSON(t, response, &envelope)
	body := envelope.Result
	if envelope.RequestID != "admin-request-event" || response.Header().Get(data.RequestIDHeader) != envelope.RequestID {
		t.Fatalf("request IDs = %q/%q", envelope.RequestID, response.Header().Get(data.RequestIDHeader))
	}
	if body.Total != 1 || len(body.Items) != 1 || body.Items[0].Status != "ACTIVE" || body.Items[0].AnnouncedAt == nil || *body.Items[0].AnnouncedAt != announcedAt.Format(time.RFC3339) || body.Items[0].Semantic.What == nil || *body.Items[0].Semantic.What != what {
		t.Fatalf("response = %#v", body)
	}
}

func TestEventsAPIAlwaysEmitsNullableTimeFields(t *testing.T) {
	client := &biz.FakeDataServiceRepo{ListEventsFunc: func(context.Context, biz.EventListQuery) (biz.EventPage, error) {
		return biz.EventPage{Items: []biz.Event{{
			ID: "EVT00000000-0000-5000-8000-000000000001", Title: "无时间事件", Summary: "摘要",
			Modality: biz.EventModalityFact, Status: biz.EventLifecycleActive,
		}}, Total: 1, Page: 1, PageSize: 50}, nil
	}}
	router := NewRouter(testConfig(), biz.NewService(client, nil), "secret")
	response := performJSONRequest(t, router, http.MethodGet, "/api/admin/v1/events", nil, "secret", "admin-null-times")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"occurred_at":null`) || !strings.Contains(response.Body.String(), `"announced_at":null`) {
		t.Fatalf("nullable time fields are not explicit: %s", response.Body.String())
	}
}

func TestCollectionCenterEvidenceAndSourceAPIsExposeConfirmedReadModels(t *testing.T) {
	now := testTime()
	client := &biz.FakeDataServiceRepo{
		ListEvidencesFunc: func(context.Context, biz.EvidenceListQuery) (biz.EvidencePage, error) {
			return biz.EvidencePage{Items: []biz.Evidence{{ID: "EVD00000000-0000-5000-8000-000000000001", RawEvidenceID: "RAW00000000-0000-5000-8000-000000000001", Summary: "summary", Categories: []biz.EvidenceCategory{{ID: "EVC00000000-0000-5000-8000-000000000001", Code: "EVENT_BRIEF", Name: "Event brief", Description: "description"}}, SourceName: "Official", SourceLevel: "L1_OFFICIAL", CollectedAt: now}}, Total: 1, Page: 1, PageSize: 50}, nil
		},
		GetRawEvidenceDocumentFunc: func(context.Context, string) (biz.RawEvidenceDocument, error) {
			return biz.RawEvidenceDocument{RawText: "/raw-evidence/documents/2026/08/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md"}, nil
		},
		ListEvidenceCategoriesFunc: func(context.Context) ([]biz.EvidenceCategory, error) {
			return []biz.EvidenceCategory{{ID: "EVC00000000-0000-5000-8000-000000000001", Code: "EVENT_BRIEF", Name: "Event brief", Description: "description"}}, nil
		},
		ListSourcesFunc: func(context.Context) ([]biz.Source, error) {
			return []biz.Source{{ID: "SRC00000000-0000-5000-8000-000000000001", Code: "official", Name: "Official", OwnershipType: "fixed", ChannelType: "api", Enabled: true, Priority: 1, DefaultSourceLevel: "L1_OFFICIAL", UpdatedAt: now}}, nil
		},
	}
	router := NewRouter(testConfig(), biz.NewService(client, biz.WithRawEvidencePublicBaseURL("https://tideai.tripwise.cn")), "secret")
	for _, path := range []string{"/api/admin/v1/evidences?is_split=false", "/api/admin/v1/raw-evidences/RAW00000000-0000-5000-8000-000000000001/collection-document", "/api/admin/v1/evidence-categories", "/api/admin/v1/sources?query=official"} {
		response := performJSONRequest(t, router, http.MethodGet, path, nil, "secret", "collection-request")
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "adapter_key") || strings.Contains(response.Body.String(), "endpoint") || strings.Contains(response.Body.String(), "app_key") {
			t.Fatalf("GET %s leaked Source internals: %s", path, response.Body.String())
		}
	}
}

func TestCollectionCenterRejectsInvalidFiltersBeforeDataCalls(t *testing.T) {
	calls := 0
	client := &biz.FakeDataServiceRepo{
		ListEvidencesFunc: func(context.Context, biz.EvidenceListQuery) (biz.EvidencePage, error) {
			calls++
			return biz.EvidencePage{}, nil
		},
		ListSourcesFunc: func(context.Context) ([]biz.Source, error) { calls++; return nil, nil },
	}
	router := NewRouter(testConfig(), biz.NewService(client), "secret")
	for _, path := range []string{"/api/admin/v1/evidences?category_id=invalid", "/api/admin/v1/evidences?is_split=yes", "/api/admin/v1/raw-evidences/invalid/collection-document", "/api/admin/v1/raw-evidences/RAW00000000-0000-5000-8000-000000000001/collection-document?unexpected=true", "/api/admin/v1/sources?priority=8", "/api/admin/v1/sources?page_size=101"} {
		response := performJSONRequest(t, router, http.MethodGet, path, nil, "secret", "invalid-filter")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
	if calls != 0 {
		t.Fatalf("Data calls = %d, want 0", calls)
	}
}

func TestCollectionCenterMapsKnownDataTimeoutToServiceUnavailable(t *testing.T) {
	client := &biz.FakeDataServiceRepo{ListSourcesFunc: func(context.Context) ([]biz.Source, error) {
		return nil, context.DeadlineExceeded
	}}
	router := NewRouter(testConfig(), biz.NewService(client), "secret")
	response := performJSONRequest(t, router, http.MethodGet, "/api/admin/v1/sources", nil, "secret", "source-timeout")
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "DATA_SERVICE_UNAVAILABLE") {
		t.Fatalf("timeout response=%d %s", response.Code, response.Body.String())
	}
}

func TestUnexpectedDataErrorReturnsGeneric500WithoutLeak(t *testing.T) {
	client := &biz.FakeDataServiceRepo{ListEventsFunc: func(context.Context, biz.EventListQuery) (biz.EventPage, error) {
		return biz.EventPage{}, errors.New("postgres connection secret-internal-detail")
	}}
	router := NewRouter(testConfig(), biz.NewService(client, nil), "secret")
	response := performJSONRequest(t, router, http.MethodGet, "/api/admin/v1/events", nil, "secret", "")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if strings.Contains(response.Body.String(), "postgres") || strings.Contains(response.Body.String(), "secret-internal-detail") {
		t.Fatalf("internal error leaked: %s", response.Body.String())
	}
}

func TestPanicReturnsStructuredErrorWithRequestID(t *testing.T) {
	client := &biz.FakeDataServiceRepo{
		ListEventsFunc: func(context.Context, biz.EventListQuery) (biz.EventPage, error) {
			panic("sensitive upstream failure")
		},
	}
	router := NewRouter(testConfig(), biz.NewService(client, nil), "secret")
	response := performJSONRequest(
		t,
		router,
		http.MethodGet,
		"/api/admin/v1/events",
		nil,
		"secret",
		"admin-panic-request",
	)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	var body errorEnvelope
	decodeJSON(t, response, &body)
	if body.RequestID != "admin-panic-request" || response.Header().Get(requestIDHeader) != body.RequestID {
		t.Fatalf("request IDs = %q/%q", body.RequestID, response.Header().Get(requestIDHeader))
	}
	if body.Error.Code != "INTERNAL_ERROR" || body.Error.Message != "internal server error" || body.Error.Details == nil {
		t.Fatalf("error = %#v", body.Error)
	}
	if strings.Contains(response.Body.String(), "sensitive upstream failure") {
		t.Fatalf("panic detail leaked: %s", response.Body.String())
	}
}

func TestAdminCORSAllowsOnlyConfiguredOriginAndHandlesPreflightBeforeAuth(t *testing.T) {
	router := NewRouter(testConfig(), biz.NewService(countingClient(new(int)), nil), "secret", "http://uat.example.test:9014")

	preflight := httptest.NewRequest(http.MethodOptions, "/api/admin/v1/events", nil)
	preflight.Header.Set("Origin", "http://uat.example.test:9014")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	allowed := httptest.NewRecorder()
	router.ServeHTTP(allowed, preflight)
	if allowed.Code != http.StatusNoContent || allowed.Header().Get("Access-Control-Allow-Origin") != "http://uat.example.test:9014" {
		t.Fatalf("allowed preflight = status %d, origin %q", allowed.Code, allowed.Header().Get("Access-Control-Allow-Origin"))
	}

	deniedRequest := httptest.NewRequest(http.MethodGet, "/api/admin/v1/events", nil)
	deniedRequest.Header.Set("Origin", "http://attacker.example.test")
	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, deniedRequest)
	if denied.Code != http.StatusForbidden || denied.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("denied origin = status %d, allow-origin %q", denied.Code, denied.Header().Get("Access-Control-Allow-Origin"))
	}
}

func countingClient(calls *int) *biz.FakeDataServiceRepo {
	return &biz.FakeDataServiceRepo{
		ListEventsFunc: func(context.Context, biz.EventListQuery) (biz.EventPage, error) {
			*calls++
			return biz.EventPage{}, nil
		},
	}
}

func testTime() time.Time {
	return time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
}

func NewRouter(config conf.RuntimeConfig, useCase *biz.Service, adminToken string, allowedOrigins ...string) http.Handler {
	config.AdminToken = adminToken
	if len(allowedOrigins) > 0 {
		config.AllowedOrigin = allowedOrigins[0]
	}
	applicationService := service.NewAdminService(useCase)
	return NewHTTPServer(config, applicationService, nil).Server.Handler
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
		request.Header.Set(data.RequestIDHeader, requestID)
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
