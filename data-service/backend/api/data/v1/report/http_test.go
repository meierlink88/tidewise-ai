package report

import (
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	reportfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/report"
)

func TestPublicationFixtureMatchesStrictContract(t *testing.T) {
	payload, err := os.ReadFile("testdata/report-publication.json")
	if err != nil {
		t.Fatal(err)
	}
	var request PublicationRequest
	if err := v1.DecodeStrictJSON(payload, publicationShape(), &request); err != nil {
		t.Fatalf("fixture error=%v path=%s", err, v1.StrictJSONErrorPath(err))
	}
	if request.PublisherReportID == "" || request.Report.Geopolitics == nil ||
		request.Report.Macroeconomics != nil || len(request.Report.IndustryChains) != 1 {
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
