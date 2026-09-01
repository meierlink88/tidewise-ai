package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestReportHTTPBindingsUseVersionedRoutesAndSafeErrors(t *testing.T) {
	stub := &reportAPIStub{}
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), stub)

	for _, test := range []struct {
		path      string
		operation string
	}{
		{path: "/api/miniapp/v1/reports/home", operation: api.OperationGetHome},
		{path: "/api/miniapp/v1/reports/" + reportTestID + "/layers/geopolitics", operation: api.OperationGetLayer},
		{path: "/api/miniapp/v1/reports/" + reportTestID + "/industry-chains/chain-21", operation: api.OperationGetChain},
		{path: "/api/miniapp/v1/reports/" + reportTestID + "/evidences?scope_type=report_card&scope_key=geo-card", operation: api.OperationListEvidences},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		request.Header.Set(requestIDHeader, "miniapp-report-request")
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status/body = %d/%s", test.path, response.Code, response.Body.String())
		}
		if response.Header().Get(requestIDHeader) != "miniapp-report-request" {
			t.Fatalf("GET %s request ID = %q", test.path, response.Header().Get(requestIDHeader))
		}
		if !strings.Contains(response.Body.String(), `"request_id":"miniapp-report-request"`) {
			t.Fatalf("GET %s does not use Miniapp envelope: %s", test.path, response.Body.String())
		}
	}
	if stub.layerRequest == nil || stub.layerRequest.ReportID != reportTestID || stub.layerRequest.LayerKey != biz.LayerGeopolitics {
		t.Fatalf("layer request = %#v", stub.layerRequest)
	}
	if stub.chainRequest == nil || stub.chainRequest.ChainKey != "chain-21" {
		t.Fatalf("chain request = %#v", stub.chainRequest)
	}
	if stub.evidenceRequest == nil || stub.evidenceRequest.ScopeType != biz.ScopeReportCard || stub.evidenceRequest.ScopeKey != "geo-card" {
		t.Fatalf("evidence request = %#v", stub.evidenceRequest)
	}

	for _, path := range []string{
		"/api/miniapp/v1/reports/home?unknown=value",
		"/api/miniapp/v1/reports/" + reportTestID + "/evidences?scope_type=report_card&scope_type=layer&scope_key=geo-card",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_REQUEST"`) {
			t.Fatalf("GET %s status/body = %d/%s", path, response.Code, response.Body.String())
		}
	}
}

func TestReportHomeAndEvidenceTraverseRealDataHTTPWithoutLeakingInternalFields(t *testing.T) {
	var mutex sync.Mutex
	seenPaths := make([]string, 0, 3)
	downstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer miniapp-data-token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		mutex.Lock()
		seenPaths = append(seenPaths, request.URL.RequestURI())
		mutex.Unlock()
		switch request.URL.Path {
		case dataapi.DataAPIPrefix + "/reports":
			if request.URL.Query().Get("published_from") != "2026-08-31T16:00:00Z" ||
				request.URL.Query().Get("published_to") != "2026-09-01T16:00:00Z" ||
				request.URL.Query().Get("limit") != "100" {
				t.Fatalf("list query = %v", request.URL.Query())
			}
			writeDownstreamResult(t, writer, map[string]any{"items": []any{dataSummary()}, "next_cursor": nil})
		case dataapi.DataAPIPrefix + "/reports/" + reportTestID + "/home":
			writeDownstreamResult(t, writer, dataHome())
		case dataapi.DataAPIPrefix + "/reports/" + reportTestID + "/evidences":
			if request.URL.Query().Get("scope_type") != "report_card" || request.URL.Query().Get("scope_key") != "geo-card" {
				t.Fatalf("evidence query = %v", request.URL.Query())
			}
			writeDownstreamResult(t, writer, map[string]any{
				"report_id": reportTestID, "scope_type": "report_card", "scope_key": "geo-card",
				"items": []any{
					map[string]any{"evidence_id": "EVD11111111-1111-4111-8111-111111111111", "role": "supports", "display_order": 1,
						"published_at": "2026-09-01T03:00:00Z", "summary": "第一条证据", "keywords": []string{"海湾"}},
					map[string]any{"evidence_id": "EVD22222222-2222-4222-8222-222222222222", "role": "context", "display_order": 2,
						"published_at": nil, "summary": "第二条证据", "keywords": []string{"运输"}},
				},
			})
		default:
			t.Fatalf("unexpected Data request %s", request.URL.RequestURI())
		}
	}))
	defer downstream.Close()
	client, err := dataapi.NewHTTPClient(dataapi.HTTPConfig{BaseURL: downstream.URL,
		ServiceToken: "miniapp-data-token", Timeout: time.Second, MaxReadAttempts: 1, HTTPClient: downstream.Client()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	repository, err := reportdata.NewRepository(client)
	if err != nil {
		t.Fatal(err)
	}
	useCase := biz.NewUseCaseWithClock(repository, func() time.Time {
		return time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	})
	application, err := reportservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	router := NewHTTPServer(testRuntimeConfig(), slog.New(slog.NewJSONHandler(io.Discard, nil)), application)

	homeResponse := httptest.NewRecorder()
	router.ServeHTTP(homeResponse, httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/reports/home", nil))
	if homeResponse.Code != http.StatusOK {
		t.Fatalf("home status/body = %d/%s", homeResponse.Code, homeResponse.Body.String())
	}
	var homeEnvelope struct {
		Result api.HomeResponse `json:"result"`
	}
	if err := json.Unmarshal(homeResponse.Body.Bytes(), &homeEnvelope); err != nil {
		t.Fatal(err)
	}
	if len(homeEnvelope.Result.Reports) != 1 || len(homeEnvelope.Result.Reports[0].Cards) != 2 ||
		homeEnvelope.Result.Reports[0].Cards[0].Key != "geo-card" {
		t.Fatalf("home result = %#v", homeEnvelope.Result)
	}
	for _, forbidden := range []string{"source_report_id", "evidence_count", "evidence_id"} {
		if strings.Contains(homeResponse.Body.String(), forbidden) {
			t.Fatalf("home response leaked %q: %s", forbidden, homeResponse.Body.String())
		}
	}

	evidenceResponse := httptest.NewRecorder()
	router.ServeHTTP(evidenceResponse, httptest.NewRequest(http.MethodGet,
		"/api/miniapp/v1/reports/"+reportTestID+"/evidences?scope_type=report_card&scope_key=geo-card", nil))
	if evidenceResponse.Code != http.StatusOK {
		t.Fatalf("evidence status/body = %d/%s", evidenceResponse.Code, evidenceResponse.Body.String())
	}
	var evidenceEnvelope struct {
		Result api.EvidenceCollection `json:"result"`
	}
	if err := json.Unmarshal(evidenceResponse.Body.Bytes(), &evidenceEnvelope); err != nil {
		t.Fatal(err)
	}
	if len(evidenceEnvelope.Result.Items) != 2 || evidenceEnvelope.Result.Items[0].Summary != "第一条证据" ||
		evidenceEnvelope.Result.Scope != (api.Scope{Type: "report_card", Key: "geo-card"}) {
		t.Fatalf("evidence result = %#v", evidenceEnvelope.Result)
	}
	for _, forbidden := range []string{"evidence_id", "display_order", `"role"`, "evidence_count", "source_type", "event_id"} {
		if strings.Contains(evidenceResponse.Body.String(), forbidden) {
			t.Fatalf("evidence response leaked %q: %s", forbidden, evidenceResponse.Body.String())
		}
	}
	mutex.Lock()
	defer mutex.Unlock()
	if len(seenPaths) != 3 {
		t.Fatalf("Data calls = %v", seenPaths)
	}
}

type reportAPIStub struct {
	layerRequest    *api.LayerRequest
	chainRequest    *api.IndustryChainRequest
	evidenceRequest *api.EvidenceRequest
}

func (s *reportAPIStub) GetHome(context.Context, *api.HomeRequest) (*api.HomeResponse, error) {
	return &api.HomeResponse{Selection: api.Selection{Mode: "today", Date: "2026-09-01", Timezone: "Asia/Shanghai"},
		Reports: []api.HomeReport{}}, nil
}

func (s *reportAPIStub) GetLayer(_ context.Context, request *api.LayerRequest) (*api.LayerDetail, error) {
	s.layerRequest = request
	return &api.LayerDetail{RelatedIndustryChains: []api.IndustryChainSummary{}}, nil
}

func (s *reportAPIStub) GetIndustryChain(_ context.Context, request *api.IndustryChainRequest) (*api.IndustryChainDetail, error) {
	s.chainRequest = request
	return &api.IndustryChainDetail{}, nil
}

func (s *reportAPIStub) ListEvidences(_ context.Context, request *api.EvidenceRequest) (*api.EvidenceCollection, error) {
	s.evidenceRequest = request
	if request.HasUnknownQuery {
		return nil, v1.ErrInvalidRequest
	}
	return &api.EvidenceCollection{ReportID: request.ReportID,
		Scope: api.Scope{Type: request.ScopeType, Key: request.ScopeKey}, Items: []api.EvidenceItem{}}, nil
}

func dataSummary() map[string]any {
	return map[string]any{
		"id": reportTestID, "source_report_id": "source-report-1", "report_type": "investment_reasoning",
		"title": "传导推理报告", "status": "published", "simulation": false,
		"generated_at": "2026-09-01T12:00:00+08:00", "timezone": "Asia/Shanghai",
		"published_layers": []string{"geopolitics", "macroeconomics", "industry_chain"},
		"statistics": map[string]any{
			"event_count": 0, "ordinary_fact_count": 0, "signal_fact_count": 0,
			"transmission_hypothesis_count": 0, "remaining_topology_pending_count": 0,
			"adaptive_inclusion_threshold": 0.6, "adaptive_continuation_threshold": 0.5,
			"adaptive_hard_max_hops": 0, "adaptive_observed_max_hops": 0,
			"adaptive_stopped_by_confidence": 0, "adaptive_stopped_by_no_unvisited_neighbor": 0,
			"adaptive_rejected_below_inclusion": 0, "geopolitic_anchor_count": 1,
			"macroeconomic_anchor_count": 1, "signaled_chain_node_count": 0,
			"industry_chain_count": 0, "unmapped_chain_node_count": 0,
		}, "published_at": "2026-09-01T04:01:00Z",
	}
}

func dataHome() map[string]any {
	return map[string]any{"report": dataSummary(), "report_cards": []any{
		map[string]any{
			"key": "geo-card", "kind": "geopolitics", "display_order": 1,
			"detail_ref": map[string]any{"type": "layer", "key": "geopolitics"},
			"title":      "地缘政治", "subtitle": "风险主线", "conclusion": "海湾风险升温。",
			"result":     map[string]any{"code": "warming", "label": "升温"},
			"confidence": map[string]any{"label": "中", "score": nil}, "time_window": "短期",
			"impact_items": []any{map[string]any{
				"ref": map[string]any{"type": "anchor", "key": "anchor-g1"}, "name": "海湾安全",
				"result":     map[string]any{"code": "warming", "label": "升温"},
				"confidence": map[string]any{"label": "中", "score": nil}, "time_window": "短期", "evidence_count": 0,
			}}, "evidence_count": 1,
		},
		map[string]any{
			"key": "macro-card", "kind": "macroeconomics", "display_order": 2,
			"detail_ref": map[string]any{"type": "layer", "key": "macroeconomics"},
			"title":      "宏观经济", "subtitle": "需求主线", "conclusion": "宏观路径分化。",
			"result":     map[string]any{"code": "diverging", "label": "分化"},
			"confidence": map[string]any{"label": "中", "score": nil}, "time_window": "中期",
			"impact_items": []any{map[string]any{
				"ref": map[string]any{"type": "anchor", "key": "anchor-m1"}, "name": "需求路径",
				"result":     map[string]any{"code": "diverging", "label": "分化"},
				"confidence": map[string]any{"label": "中", "score": nil}, "time_window": "中期", "evidence_count": 1,
			}}, "evidence_count": 1,
		},
	}, "company": map[string]any{"key": "company", "display_order": 4, "title": "企业",
		"published": false, "boundary": "本次推理尚未进入企业层。"}}
}

func writeDownstreamResult(t *testing.T, writer http.ResponseWriter, result any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(map[string]any{"request_id": "data-report-request", "result": result}); err != nil {
		t.Fatal(err)
	}
}
