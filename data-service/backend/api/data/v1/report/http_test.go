package report

import (
	"bytes"
	"context"
	"encoding/json"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	reportfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/report"
)

func TestAgentOSPublicationFixtureMatchesStrictContract(t *testing.T) {
	payload, err := os.ReadFile("testdata/investment-report-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	var request PublicationRequest
	if err := v1.DecodeStrictJSON(payload, publicationShape(), &request); err != nil {
		t.Fatalf("AgentOS fixture error=%v path=%s", err, v1.StrictJSONErrorPath(err))
	}
	if request.PublisherReportID != "agentos-investment-run-001" ||
		request.Report.ReportType.Code != "investment_reasoning" ||
		request.Report.Timezone != "Asia/Shanghai" || len(request.Report.IndustryChains) != 1 {
		t.Fatalf("fixture=%#v", request)
	}
}

func TestPublicationShapeAcceptsOptionalUpperSectionsAndRejectsRetiredFields(t *testing.T) {
	for _, report := range []any{reportfixture.IndustryOnlyReport(), reportfixture.FrozenScaleBaselineReport()} {
		payload, err := json.Marshal(map[string]any{"publisher_report_id": "publisher-report", "report": report})
		if err != nil {
			t.Fatal(err)
		}
		var request PublicationRequest
		if err := v1.DecodeStrictJSON(payload, publicationShape(), &request); err != nil {
			t.Fatalf("valid publication error=%v path=%s", err, v1.StrictJSONErrorPath(err))
		}
		if request.PublisherReportID != "publisher-report" || len(request.Report.IndustryChains) == 0 {
			t.Fatalf("request=%#v", request)
		}
	}
	payload, err := json.Marshal(map[string]any{"publisher_report_id": "publisher-report", "report": reportfixture.IndustryOnlyReport()})
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		t.Fatal(err)
	}
	root["report"].(map[string]any)["analysis_window"] = map[string]any{}
	changed, _ := json.Marshal(root)
	var request PublicationRequest
	err = v1.DecodeStrictJSON(changed, publicationShape(), &request)
	if err == nil || !strings.Contains(v1.StrictJSONErrorPath(err), "report.analysis_window") {
		t.Fatalf("error=%v path=%s", err, v1.StrictJSONErrorPath(err))
	}
}

func TestReportQueriesRejectDuplicatesAndUnknowns(t *testing.T) {
	for _, test := range []struct {
		query              url.Values
		required, optional []string
		want               bool
	}{
		{query: url.Values{}, optional: []string{"limit", "cursor"}, want: true},
		{query: url.Values{"limit": {"20"}}, optional: []string{"limit", "cursor"}, want: true},
		{query: url.Values{"limit": {"20", "30"}}, optional: []string{"limit", "cursor"}},
		{query: url.Values{"page": {"1"}}, optional: []string{"limit", "cursor"}},
		{query: url.Values{"scope_token": {"RPE11111111-1111-4111-8111-111111111111"}}, required: []string{"scope_token"}, want: true},
	} {
		if got := validQueryValues(test.query, test.required, test.optional); got != test.want {
			t.Fatalf("query=%v got=%t want=%t", test.query, got, test.want)
		}
	}
}

func TestStoryConceptStrictPublication(t *testing.T) {
	payload, err := os.ReadFile("testdata/story-concept-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	var request PublicationRequest
	if err := v1.DecodeStrictJSON(payload, analysisPublicationShape(), &request); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(map[string]any){
		func(r map[string]any) { delete(r, "macroeconomic_stories") },
		func(r map[string]any) { r["macroeconomic_stories"] = nil },
		func(r map[string]any) { r["industry_chains"] = []any{} },
		func(r map[string]any) { r["unexpected"] = true },
	} {
		var root map[string]any
		_ = json.Unmarshal(payload, &root)
		change(root["report"].(map[string]any))
		raw, _ := json.Marshal(root)
		if v1.DecodeStrictJSON(raw, analysisPublicationShape(), &request) == nil {
			t.Fatal("accepted invalid shape")
		}
	}
}

func TestPublicationFixturesMatchOpenAPI(t *testing.T) {
	document, err := openapi3.NewLoader().LoadFromFile("../openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"investment-report-publication-request.json", "story-concept-publication-request.json", "story-chain-publication-request.json", "normalized-publication-request.json", "signal-publication-request.json"} {
		payload, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal(payload, &value); err != nil {
			t.Fatal(err)
		}
		if err := document.Components.Schemas["ReportPublicationRequest"].Value.VisitJSON(value); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestImpactAssessmentStrictPublication(t *testing.T) {
	payload, err := os.ReadFile("testdata/story-chain-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	var request PublicationRequest
	if err := v1.DecodeStrictJSON(payload, analysisPublicationShape(), &request); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(map[string]any){
		func(s map[string]any) { s["impact_assessment"] = nil },
		func(s map[string]any) { delete(s["impact_assessment"].(map[string]any), "level") },
		func(s map[string]any) { s["impact_assessment"].(map[string]any)["rationale"] = 5 },
		func(s map[string]any) { s["impact_assessment"].(map[string]any)["event_ids"] = []any{} },
	} {
		var root map[string]any
		_ = json.Unmarshal(payload, &root)
		summary := root["report"].(map[string]any)["geopolitical_stories"].([]any)[0].(map[string]any)["summary"].(map[string]any)
		change(summary)
		wire, _ := json.Marshal(root)
		if err := v1.DecodeStrictJSON(wire, analysisPublicationShape(), &request); err == nil {
			t.Fatal("invalid assessment wire accepted")
		}
	}
}

func TestNormalizedPublicationStrictShape(t *testing.T) {
	b, err := os.ReadFile("testdata/normalized-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	var r PublicationRequest
	if err := v1.DecodeStrictJSON(b, normalizedPublicationShape(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Report.V4 == nil {
		t.Fatal("version dispatch lost report")
	}
}

// Provider schema and Miniapp consumer use the same synthetic v5 read fixture.
func TestSignalReadFixtureMatchesProviderContract(t *testing.T) {
	document, err := openapi3.NewLoader().LoadFromFile("../openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile("../../../../../../miniapp/frontend/src/mocks/reports/normalized-v5.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Groups []struct {
			Items []any `json:"items"`
		} `json:"groups"`
		Details map[string]any `json:"details"`
		Chains  map[string]any `json:"chains"`
	}
	if err = json.Unmarshal(payload, &fixture); err != nil {
		t.Fatal(err)
	}
	check := func(name string, value any) {
		t.Helper()
		if err := document.Components.Schemas[name].Value.VisitJSON(value); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for _, g := range fixture.Groups {
		for _, item := range g.Items {
			check("SignalSummaryProjection", item)
		}
	}
	for _, d := range fixture.Details {
		check("SignalDetailProjection", d)
	}
	for _, c := range fixture.Chains {
		check("SignalChainRead", c)
	}
}

// Exercise version dispatch at the actual HTTP boundary, not only the Go decoder.
func TestUnifiedPublicationHTTPDispatch(t *testing.T) {
	payload, err := os.ReadFile("testdata/unified-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	app := &unifiedPublicationService{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, app)
	send := func(body []byte) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/data/v1/report-publications", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		return rec
	}
	rec := send(payload)
	if app.calls != 1 || rec.Code >= 400 {
		t.Fatalf("calls=%d status=%d body=%s", app.calls, rec.Code, rec.Body.String())
	}
	for _, mutate := range []func(map[string]any){
		func(r map[string]any) { r["unexpected"] = true },
		func(r map[string]any) {
			r["geopolitical_stories"].([]any)[0].(map[string]any)["detail"].(map[string]any)["industry_chains"] = []any{}
		},
		func(r map[string]any) {
			r["geopolitical_stories"].([]any)[0].(map[string]any)["detail"].(map[string]any)["reasonings"].([]any)[0].(map[string]any)["affected_assets"].([]any)[0].(map[string]any)["assessment"].(map[string]any)["weight_delta_pp"] = "4"
		},
	} {
		var root map[string]any
		if err := json.Unmarshal(payload, &root); err != nil {
			t.Fatal(err)
		}
		mutate(root["report"].(map[string]any))
		changed, _ := json.Marshal(root)
		rec = send(changed)
		if rec.Code < 400 || app.calls != 1 {
			t.Fatalf("invalid reached application: calls=%d status=%d", app.calls, rec.Code)
		}
	}
}

type unifiedPublicationService struct {
	Service
	calls int
}

func (s *unifiedPublicationService) PublishReport(_ context.Context, r *PublicationRequest) (*v1.Response[PublicationResult], error) {
	s.calls++
	return &v1.Response[PublicationResult]{Status: v1.StatusCreated}, nil
}

func TestUnifiedGeopoliticalNullableFieldsDecode(t *testing.T) {
	payload, err := os.ReadFile("testdata/unified-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err = json.Unmarshal(payload, &root); err != nil {
		t.Fatal(err)
	}
	unit := root["report"].(map[string]any)["geopolitical_stories"].([]any)[0].(map[string]any)
	assessment := unit["detail"].(map[string]any)["reasonings"].([]any)[0].(map[string]any)["assessment"].(map[string]any)
	for _, key := range []string{"confidence", "forecast_window", "follow_up"} {
		assessment[key] = nil
	}
	body, _ := json.Marshal(root)
	app := &unifiedPublicationService{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, app)
	req := httptest.NewRequest("POST", "/api/data/v1/report-publications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code >= 400 || app.calls != 1 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUnifiedNonGeopoliticalMetadataRemainsRequired(t *testing.T) {
	payload, err := os.ReadFile("testdata/unified-publication-request.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"macroeconomic_stories", "concept_analyses", "industry_chain_analyses"} {
		for _, field := range []string{"confidence", "forecast_window", "follow_up"} {
			var root map[string]any
			_ = json.Unmarshal(payload, &root)
			report := root["report"].(map[string]any)
			units := report["geopolitical_stories"].([]any)
			report[kind] = units
			report["geopolitical_stories"] = []any{}
			a := units[0].(map[string]any)["detail"].(map[string]any)["reasonings"].([]any)[0].(map[string]any)["assessment"].(map[string]any)
			delete(a, field)
			body, _ := json.Marshal(root)
			var req PublicationRequest
			if v1.DecodeStrictJSON(body, requiredShape(map[string]*v1.StrictJSONShape{"publisher_report_id": v1.StrictJSONString(), "report": unifiedReportShape()}), &req) == nil {
				t.Fatalf("%s accepted missing %s", kind, field)
			}
		}
	}
}
