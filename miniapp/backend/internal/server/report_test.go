package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"

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
	if stub.evidenceRequest == nil || stub.evidenceRequest.ScopeToken != reportScopeToken {
		t.Fatalf("evidence=%#v", stub.evidenceRequest)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/reports/"+reportTestID+"/evidences?scope_token=a&scope_token=b", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status/body=%d/%s", response.Code, response.Body.String())
	}
}

type reportAPIStub struct {
	evidenceRequest *api.EvidenceRequest
}

func (*reportAPIStub) GetHome(context.Context, *api.HomeRequest) (*api.HomeResponse, error) {
	return &api.HomeResponse{Selection: api.Selection{Mode: "today", Date: "2026-09-02", Timezone: "Asia/Shanghai"}, Reports: []api.HomeReport{}}, nil
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
	return map[string]any{"local_key": "chain-01", "name": "运输产业链", "conclusion": "运输成本升温", "result": map[string]any{"code": "warming", "label": "升温"}, "confidence": map[string]any{"code": "medium", "label": "中"}, "time_window": map[string]any{"code": "short", "label": "短期"}, "impact_items": []any{}, "evidence_scope_token": reportScopeToken}
}
func writeDownstreamResult(t *testing.T, writer http.ResponseWriter, result any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(map[string]any{"request_id": "data-request", "result": result}); err != nil {
		t.Fatal(err)
	}
}

func (*reportAPIStub) ListAnalyses(context.Context, *api.AnalysisQuery) (*api.AnalysisPage, error) {
	return nil, nil
}

func (*reportAPIStub) GetAnalysis(context.Context, *api.AnalysisQuery) (*api.NormalizedDetailProjection, error) {
	return nil, nil
}

func (*reportAPIStub) GetAnalysisChain(context.Context, *api.AnalysisQuery) (*api.NormalizedChain, error) {
	return nil, nil
}

// The same synthetic wire fixture is consumed by the frontend parser and all BFF layers.
func TestNormalizedReportHTTPTraversesDataAndPreservesProjection(t *testing.T) {
	for _, version := range []string{"v4", "v5", "v5-industry"} {
		t.Run(version, func(t *testing.T) { testNormalizedHTTP(t, version) })
	}
}
func testNormalizedHTTP(t *testing.T, version string) {
	document, err := openapi3.NewLoader().LoadFromFile("../../api/miniapp/v1/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	filename := "normalized.json"
	if strings.HasPrefix(version, "v5") {
		filename = "normalized-v5.json"
	}
	raw, err := os.ReadFile("../../../frontend/src/mocks/reports/" + filename)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Groups  []map[string]any `json:"groups"`
		Details map[string]any   `json:"details"`
		Chains  map[string]any   `json:"chains"`
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if version == "v5-industry" {
		// Exercise the same projection shape under the independent industry route.
		fixture.Groups[2]["kind"] = "industry_chain_analyses"
		for key, value := range fixture.Details {
			if strings.HasPrefix(key, "concept_analyses/") {
				fixture.Details[strings.Replace(key, "concept_analyses/", "industry_chain_analyses/", 1)] = value
				delete(fixture.Details, key)
			}
		}
		for key, value := range fixture.Chains {
			if strings.HasPrefix(key, "concept_analyses/") {
				fixture.Chains[strings.Replace(key, "concept_analyses/", "industry_chain_analyses/", 1)] = value
				delete(fixture.Chains, key)
			}
		}
		fixture.Groups = append(fixture.Groups[:2], map[string]any{"kind": "concept_analyses", "items": []any{}, "next_cursor": nil}, fixture.Groups[2])
		version = "v5"
	} else if version == "v5" {
		fixture.Groups = append(fixture.Groups, map[string]any{"kind": "industry_chain_analyses", "items": []any{}, "next_cursor": nil})
	}
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := dataapi.DataAPIPrefix + "/reports"
		if r.URL.Path == prefix {
			s := dataSummary()
			s["schema_version"] = "report-publication/" + version
			s["analysis_window"] = map[string]string{"start": "2026-09-01T00:00:00Z", "end": "2026-09-02T00:00:00Z"}
			writeDownstreamResult(t, w, map[string]any{"items": []any{s}, "next_cursor": nil})
			return
		}
		key := strings.TrimPrefix(r.URL.Path, prefix+"/"+reportTestID+"/analyses/")
		for _, g := range fixture.Groups {
			if key == g["kind"] {
				if r.URL.Query().Get("limit") != "20" {
					t.Errorf("page limit not forwarded: %s", r.URL.RequestURI())
				}
				if cursor := r.URL.Query().Get("cursor"); cursor != "" && cursor != "scoped-next" {
					t.Errorf("cursor changed: %s", cursor)
				}
				writeDownstreamResult(t, w, map[string]any{"items": g["items"], "next_cursor": g["next_cursor"]})
				return
			}
		}
		if v, ok := fixture.Details[key]; ok {
			writeDownstreamResult(t, w, v)
			return
		}
		key = strings.Replace(key, "/industry-chains/", "/", 1)
		if v, ok := fixture.Chains[key]; ok {
			writeDownstreamResult(t, w, v)
			return
		}
		t.Errorf("unexpected request %s", r.URL.RequestURI())
		http.NotFound(w, r)
	}))
	defer downstream.Close()
	client, err := dataapi.NewHTTPClient(dataapi.HTTPConfig{BaseURL: downstream.URL, ServiceToken: "test-token", Timeout: time.Second, MaxReadAttempts: 1, HTTPClient: downstream.Client()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	repository, err := reportdata.NewRepository(client)
	if err != nil {
		t.Fatal(err)
	}
	app, err := reportservice.NewService(biz.NewUseCase(repository))
	if err != nil {
		t.Fatal(err)
	}
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), app)
	read := func(path string) any {
		t.Helper()
		r := httptest.NewRecorder()
		router.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != 200 {
			t.Fatalf("%s: %d %s", path, r.Code, r.Body.String())
		}
		var envelope map[string]any
		if err := json.Unmarshal(r.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(r.Body.String(), "evidence_ids") {
			t.Fatal("Evidence IDs leaked")
		}
		result := envelope["result"]
		schema := "NormalizedDetailProjection"
		if strings.HasSuffix(path, "/home") {
			schema = "HomeResponse"
		} else if strings.Contains(path, "?limit=") {
			schema = "AnalysisPage"
		} else if strings.Contains(path, "/industry-chains/") {
			schema = "NormalizedReadChain"
		}
		if err := document.Components.Schemas[schema].Value.VisitJSON(result); err != nil {
			t.Fatalf("%s: %v", schema, err)
		}
		return result
	}
	home := read("/api/miniapp/v1/reports/home").(map[string]any)
	reports := home["reports"].([]any)
	groups := reports[0].(map[string]any)["analysis_groups"]
	wantGroups := make([]any, len(fixture.Groups))
	for i, g := range fixture.Groups {
		wantGroups[i] = g
	}
	if !reflect.DeepEqual(groups, wantGroups) {
		t.Fatal("home did not preserve group projections and counts")
	}
	for _, group := range fixture.Groups {
		kind := group["kind"].(string)
		page := read("/api/miniapp/v1/reports/" + reportTestID + "/analyses/" + kind + "?limit=20&cursor=scoped-next").(map[string]any)
		if !reflect.DeepEqual(page["items"], group["items"]) {
			t.Fatalf("page mismatch %s", kind)
		}
	}
	for key, want := range fixture.Details {
		if got := read("/api/miniapp/v1/reports/" + reportTestID + "/analyses/" + key); !reflect.DeepEqual(got, want) {
			t.Fatalf("detail mismatch %s", key)
		}
	}
	for key, want := range fixture.Chains {
		parts := strings.Split(key, "/")
		path := strings.Join(parts[:2], "/") + "/industry-chains/" + parts[2]
		if got := read("/api/miniapp/v1/reports/" + reportTestID + "/analyses/" + path); !reflect.DeepEqual(got, want) {
			t.Fatalf("chain mismatch %s", key)
		}
	}
	for _, suffix := range []string{"?limit=2&limit=3", "?limit=101", "?unknown=1"} {
		r := httptest.NewRecorder()
		router.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/reports/"+reportTestID+"/analyses/geopolitical_stories"+suffix, nil))
		if r.Code != 400 {
			t.Fatalf("%s: %d", suffix, r.Code)
		}
	}
}

func TestRetiredReportRoutesReturnNotFound(t *testing.T) {
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), &reportAPIStub{})
	for _, suffix := range []string{"/industry-chains", "/layers/geopolitics", "/industry-chains/chain-01"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/reports/"+reportTestID+suffix, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d", suffix, response.Code)
		}
	}
}
