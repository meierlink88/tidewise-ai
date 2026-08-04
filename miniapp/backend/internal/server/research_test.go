package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	usecase "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
	dataclient "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
	appservice "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/service"
)

func TestResearchRoutesPreserveNonEmptyPublicThemeGoldenAndRequestID(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	calls := 0
	client := &usecase.Fake{ListResearchThemesFunc: func(ctx context.Context, query usecase.ResearchListQuery) (usecase.ResearchThemePage, error) {
		calls++
		if dataclient.RequestIDFromContext(ctx) != "miniapp-request-1" {
			t.Fatalf("request ID = %q", dataclient.RequestIDFromContext(ctx))
		}
		return usecase.ResearchThemePage{
			WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, AsOf: now, ThemeCount: 1, EventCount: 1,
			Items: []usecase.ResearchTheme{{
				ID: "11111111-1111-4111-8111-111111111111", Title: "主题", OneLineConclusion: "结论",
				ConclusionDirection: "positive", ImpactStrength: "medium", TransmissionStage: "identification",
				InvestmentGuidanceAction: "focus", InvestmentGuidanceSummary: "风险偏好可能回升",
				TimeHorizonCategory: "short_term", AnalysisAsOf: now, WindowStart: now.Add(-24 * time.Hour),
				WindowEnd: now, PublishedAt: now, Impacts: []usecase.ResearchThemeImpact{},
			}},
		}, nil
	}}
	router := researchTestRouter(usecase.NewResearchService(client))
	request := httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/research/themes?window_hours=24&limit=20", nil)
	request.Header.Set(dataclient.RequestIDHeader, "miniapp-request-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["request_id"] != "miniapp-request-1" || response.Header().Get(dataclient.RequestIDHeader) != "miniapp-request-1" {
		t.Fatalf("request IDs = %#v/%q", body["request_id"], response.Header().Get(dataclient.RequestIDHeader))
	}
	result, ok := body["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", body["result"])
	}
	items, ok := result["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v", result["items"])
	}
	item := items[0].(map[string]any)
	if item["impact_strength"] != "medium" || item["investment_guidance_summary"] != "风险偏好可能回升" || item["transmission_stage"] != "identification" {
		t.Fatalf("item = %#v", item)
	}
	if item["impacts"] == nil || calls != 1 {
		t.Fatalf("collections/calls = %#v/%d", item["impacts"], calls)
	}
}

func TestResearchRoutesDoNotExposeLegacyStandaloneAnchorAPI(t *testing.T) {
	router := researchTestRouter(usecase.NewResearchService(&usecase.Fake{}))
	for _, path := range []string{
		"/api/miniapp/v1/research/anchors",
		"/api/miniapp/v1/research/anchors/11111111-1111-4111-8111-111111111111",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want 404", path, response.Code)
		}
	}
}

func TestResearchRoutesPreserve400404And500WithoutUpstreamLeak(t *testing.T) {
	t.Run("invalid request", func(t *testing.T) {
		calls := 0
		client := &usecase.Fake{ListResearchThemesFunc: func(context.Context, usecase.ResearchListQuery) (usecase.ResearchThemePage, error) {
			calls++
			return usecase.ResearchThemePage{}, nil
		}}
		response := serveResearch(t, usecase.NewResearchService(client), "/api/miniapp/v1/research/themes?limit=51")
		if response.Code != http.StatusBadRequest || calls != 0 {
			t.Fatalf("status/calls = %d/%d", response.Code, calls)
		}
	})

	for _, test := range []struct {
		name       string
		upstream   error
		wantStatus int
		wantCode   string
	}{
		{name: "not found", upstream: usecase.ErrResearchNotFound, wantStatus: 404, wantCode: "RESEARCH_RESULT_NOT_FOUND"},
		{name: "internal", upstream: errors.New("postgres password=do-not-leak"), wantStatus: 500, wantCode: "RESEARCH_DATA_UNAVAILABLE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := &usecase.Fake{GetResearchThemeFunc: func(context.Context, string) (usecase.ResearchThemeDetail, error) {
				calls++
				return usecase.ResearchThemeDetail{}, test.upstream
			}}
			response := serveResearch(t, usecase.NewResearchService(client), "/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111")
			if response.Code != test.wantStatus || calls != 1 {
				t.Fatalf("status/body/calls = %d/%q/%d", response.Code, response.Body.String(), calls)
			}
			assertReasoningError(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func researchTestRouter(service *usecase.ResearchService) http.Handler {
	return researchTestServer(service)
}

func researchTestServer(useCase *usecase.ResearchService) *kratoshttp.Server {
	return NewHTTPServer(testRuntimeConfig(), appservice.NewResearchService(useCase), testLogger())
}

func serveResearch(t *testing.T, service *usecase.ResearchService, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	researchTestRouter(service).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}
