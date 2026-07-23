package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/usecase"
)

func TestResearchRoutesPreserveNonEmptyPublicThemeGoldenAndRequestID(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	calls := 0
	client := &dataclient.Fake{ListResearchThemesFunc: func(ctx context.Context, query dataclient.ResearchListQuery) (dataclient.ResearchThemePage, error) {
		calls++
		if dataclient.RequestIDFromContext(ctx) != "miniapp-request-1" {
			t.Fatalf("request ID = %q", dataclient.RequestIDFromContext(ctx))
		}
		return dataclient.ResearchThemePage{
			WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, AsOf: now, ThemeCount: 1, EventCount: 1,
			Items: []dataclient.ResearchTheme{{
				ID: "11111111-1111-4111-8111-111111111111", Name: "主题", OneLineConclusion: "结论",
				ImpactLevel: dataclient.ImpactLevelHigh, TransmissionPath: "政策到产业链", TradingDirection: "风险偏好可能回升",
				TransmissionStage: dataclient.TransmissionStageIdentification, NextCheckpoint: "下周数据", PublishedAt: now,
				AffectedChainNodes: []dataclient.ResearchThemeChainNode{}, RelatedIndices: []dataclient.ResearchIndex{},
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
	if item["impact_level"] != "high" || item["trading_direction"] != "风险偏好可能回升" || item["transmission_stage"] != "identification" {
		t.Fatalf("item = %#v", item)
	}
	if item["affected_chain_nodes"] == nil || item["related_indices"] == nil || calls != 1 {
		t.Fatalf("collections/calls = %#v/%#v/%d", item["affected_chain_nodes"], item["related_indices"], calls)
	}
}

func TestResearchRoutesDoNotExposeLegacyStandaloneAnchorAPI(t *testing.T) {
	router := researchTestRouter(usecase.NewResearchService(&dataclient.Fake{}))
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
		client := &dataclient.Fake{ListResearchThemesFunc: func(context.Context, dataclient.ResearchListQuery) (dataclient.ResearchThemePage, error) {
			calls++
			return dataclient.ResearchThemePage{}, nil
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
		{name: "not found", upstream: &dataclient.Error{Kind: dataclient.ErrorKindClient, StatusCode: 404}, wantStatus: 404, wantCode: "RESEARCH_RESULT_NOT_FOUND"},
		{name: "internal", upstream: errors.New("postgres password=do-not-leak"), wantStatus: 500, wantCode: "RESEARCH_DATA_UNAVAILABLE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := &dataclient.Fake{GetResearchThemeFunc: func(context.Context, string, dataclient.ResearchDetailQuery) (dataclient.ResearchThemeDetail, error) {
				calls++
				return dataclient.ResearchThemeDetail{}, test.upstream
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
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterResearchRoutes(router.Group("/api/miniapp/v1"), service)
	return router
}

func serveResearch(t *testing.T, service *usecase.ResearchService, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	researchTestRouter(service).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}
