package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
	api "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1/report"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
	dataapi "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
	reportdata "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data/report"
	reportservice "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/service/report"
)

const reportTestID = "RPT11111111-1111-4111-8111-111111111111"
const reportScopeToken = "RPE11111111-1111-4111-8111-111111111111"

func TestReportHTTPBindingsUseVersionedRoutes(t *testing.T) {
	stub := &reportAPIStub{}
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), stub)
	paths := []string{
		"/api/miniapp/v1/reports/home",
		"/api/miniapp/v1/reports/" + reportTestID + "/layers/geopolitics",
		"/api/miniapp/v1/reports/" + reportTestID + "/industry-chains?limit=20&cursor=next",
		"/api/miniapp/v1/reports/" + reportTestID + "/industry-chains/chain-21",
		"/api/miniapp/v1/reports/" + reportTestID + "/evidences?scope_token=" + reportScopeToken,
	}
	for _, path := range paths {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set(requestIDHeader, "miniapp-report-request")
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"request_id":"miniapp-report-request"`) {
			t.Fatalf("GET %s status/body=%d/%s", path, response.Code, response.Body.String())
		}
	}
	if stub.chainListRequest == nil || stub.chainListRequest.Cursor != "next" || stub.evidenceRequest == nil || stub.evidenceRequest.ScopeToken != reportScopeToken {
		t.Fatalf("chain=%#v evidence=%#v", stub.chainListRequest, stub.evidenceRequest)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/reports/"+reportTestID+"/evidences?scope_token=a&scope_token=b", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status/body=%d/%s", response.Code, response.Body.String())
	}
}

func TestReportHomeAndEvidenceTraverseRealDataHTTP(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer miniapp-data-token" {
			t.Fatalf("authorization=%q", request.Header.Get("Authorization"))
		}
		switch request.URL.Path {
		case dataapi.DataAPIPrefix + "/reports":
			writeDownstreamResult(t, writer, map[string]any{"items": []any{dataSummary()}, "next_cursor": nil})
		case dataapi.DataAPIPrefix + "/reports/" + reportTestID + "/home":
			writeDownstreamResult(t, writer, map[string]any{"report": dataSummary(), "geopolitics": nil, "macroeconomics": nil})
		case dataapi.DataAPIPrefix + "/reports/" + reportTestID + "/industry-chains":
			writeDownstreamResult(t, writer, map[string]any{"items": []any{dataChainSummary()}, "next_cursor": nil})
		case dataapi.DataAPIPrefix + "/reports/" + reportTestID + "/evidences":
			if request.URL.Query().Get("scope_token") != reportScopeToken {
				t.Fatalf("query=%v", request.URL.Query())
			}
			writeDownstreamResult(t, writer, map[string]any{"report_id": reportTestID, "scope_token": reportScopeToken, "items": []any{map[string]any{"published_at": nil, "summary": "证据摘要", "keywords": []string{"关键词"}}}})
		default:
			t.Fatalf("unexpected Data request %s", request.URL.RequestURI())
		}
	}))
	defer downstream.Close()
	client, err := dataapi.NewHTTPClient(dataapi.HTTPConfig{BaseURL: downstream.URL, ServiceToken: "miniapp-data-token", Timeout: time.Second, MaxReadAttempts: 1, HTTPClient: downstream.Client()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	repository, err := reportdata.NewRepository(client)
	if err != nil {
		t.Fatal(err)
	}
	useCase := biz.NewUseCaseWithClock(repository, func() time.Time { return time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC) })
	application, err := reportservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	router := NewHTTPServer(testRuntimeConfig(), slog.New(slog.NewJSONHandler(io.Discard, nil)), application)
	home := httptest.NewRecorder()
	router.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/reports/home", nil))
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), `"local_key":"chain-01"`) {
		t.Fatalf("home=%d/%s", home.Code, home.Body.String())
	}
	if strings.Contains(home.Body.String(), "evidence_id") {
		t.Fatalf("home leaked Evidence ID: %s", home.Body.String())
	}
	evidence := httptest.NewRecorder()
	router.ServeHTTP(evidence, httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/reports/"+reportTestID+"/evidences?scope_token="+reportScopeToken, nil))
	if evidence.Code != http.StatusOK || !strings.Contains(evidence.Body.String(), "证据摘要") {
		t.Fatalf("evidence=%d/%s", evidence.Code, evidence.Body.String())
	}
}

type reportAPIStub struct {
	chainListRequest *api.IndustryChainListRequest
	evidenceRequest  *api.EvidenceRequest
}

func (*reportAPIStub) GetHome(context.Context, *api.HomeRequest) (*api.HomeResponse, error) {
	return &api.HomeResponse{Selection: api.Selection{Mode: "today", Date: "2026-09-02", Timezone: "Asia/Shanghai"}, Reports: []api.HomeReport{}}, nil
}
func (s *reportAPIStub) ListIndustryChains(_ context.Context, request *api.IndustryChainListRequest) (*api.CardCollection, error) {
	s.chainListRequest = request
	if request.HasUnknownQuery {
		return nil, v1.ErrInvalidRequest
	}
	return &api.CardCollection{Items: []api.Card{}}, nil
}
func (*reportAPIStub) GetLayer(context.Context, *api.LayerRequest) (*api.LayerDetail, error) {
	return &api.LayerDetail{RelatedIndustryChains: []api.RelatedIndustryChain{}}, nil
}
func (*reportAPIStub) GetIndustryChain(context.Context, *api.IndustryChainRequest) (*api.IndustryChainDetail, error) {
	return &api.IndustryChainDetail{}, nil
}
func (s *reportAPIStub) ListEvidences(_ context.Context, request *api.EvidenceRequest) (*api.EvidenceCollection, error) {
	s.evidenceRequest = request
	if request.HasUnknownQuery {
		return nil, v1.ErrInvalidRequest
	}
	return &api.EvidenceCollection{ReportID: request.ReportID, ScopeToken: request.ScopeToken, Items: []api.EvidenceItem{}}, nil
}

func dataSummary() map[string]any {
	return map[string]any{"id": reportTestID, "publisher_report_id": "publisher", "generated_at": "2026-09-02T04:00:00Z", "has_geopolitics": false, "has_macroeconomics": false, "industry_chain_count": 1, "published_at": "2026-09-02T04:01:00Z"}
}
func dataChainSummary() map[string]any {
	return map[string]any{"local_key": "chain-01", "name": "运输产业链", "conclusion": "运输成本升温", "status": "已发布", "result": map[string]any{"code": "warming", "label": "升温"}, "confidence": map[string]any{"code": "medium", "label": "中", "score": nil}, "time_window": map[string]any{"code": "short", "label": "短期"}, "impact_items": []any{}, "evidence_scope_token": reportScopeToken}
}
func writeDownstreamResult(t *testing.T, writer http.ResponseWriter, result any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(map[string]any{"request_id": "data-request", "result": result}); err != nil {
		t.Fatal(err)
	}
}
