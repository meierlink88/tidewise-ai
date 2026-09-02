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

func TestPublicationV2SmokeFixtureMatchesStrictContract(t *testing.T) {
	payload, err := os.ReadFile("testdata/report-publication.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var request PublicationRequest
	if err := v1.DecodeStrictJSON(payload, publicationV2Shape(), &request); err != nil {
		t.Fatalf("smoke fixture error=%v path=%s", err, v1.StrictJSONErrorPath(err))
	}
	if request.ContractVersion != ContractVersion || request.PublisherReportID == "" ||
		request.Content.Geopolitics == nil || request.Content.Macroeconomics == nil || len(request.Content.IndustryChains) != 1 {
		t.Fatalf("smoke fixture=%#v", request)
	}
}

func TestPublicationV2ShapeAcceptsOptionalUpperSectionsAndRejectsPageFields(t *testing.T) {
	for _, content := range []any{reportfixture.IndustryOnlyContent(), reportfixture.ContentWithManyChains(54)} {
		payload, err := json.Marshal(map[string]any{"contract_version": ContractVersion, "publisher_report_id": "publisher-report", "content": content})
		if err != nil {
			t.Fatal(err)
		}
		var request PublicationRequest
		if err := v1.DecodeStrictJSON(payload, publicationV2Shape(), &request); err != nil {
			t.Fatalf("valid v2 error=%v path=%s", err, v1.StrictJSONErrorPath(err))
		}
		if request.PublisherReportID != "publisher-report" || len(request.Content.IndustryChains) == 0 {
			t.Fatalf("request=%#v", request)
		}
	}
	payload, err := json.Marshal(map[string]any{"contract_version": ContractVersion, "publisher_report_id": "publisher-report", "content": reportfixture.IndustryOnlyContent()})
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		t.Fatal(err)
	}
	root["content"].(map[string]any)["report_cards"] = []any{}
	changed, _ := json.Marshal(root)
	var request PublicationRequest
	err = v1.DecodeStrictJSON(changed, publicationV2Shape(), &request)
	if err == nil || !strings.Contains(v1.StrictJSONErrorPath(err), "content.report_cards") {
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
		{query: url.Values{"scope_type": {"anchor"}, "scope_key": {"geo-anchor"}}, required: []string{"scope_type", "scope_key"}, want: true},
	} {
		if got := validQueryValues(test.query, test.required, test.optional); got != test.want {
			t.Fatalf("query=%v got=%t want=%t", test.query, got, test.want)
		}
	}
}
