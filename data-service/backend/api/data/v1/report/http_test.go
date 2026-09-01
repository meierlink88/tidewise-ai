package report

import (
	"encoding/json"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	reportfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/report"
)

func TestPublicationShapeAcceptsOnlyFrozenReportPublicationV1(t *testing.T) {
	payload := publicationPayload(t)
	var request PublicationRequest
	if err := v1.DecodeStrictJSON(payload, publicationShape(), &request); err != nil {
		t.Fatalf("DecodeStrictJSON(valid) error = %v", err)
	}
	if request.SourceReportID != "agentos-report-2026-09-01-a" || len(request.Content.ReportCards) != 3 ||
		len(request.Content.Geopolitics.DownwardTransmission.PublishedPaths[0].TargetRefs) != 2 {
		t.Fatalf("decoded request = %#v", request)
	}
	expectedPayload, err := json.Marshal(reportfixture.Content())
	if err != nil {
		t.Fatal(err)
	}
	var expected Content
	if err := json.Unmarshal(expectedPayload, &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(request.Content, expected) {
		t.Fatal("stable HTTP fixture drifted from the reviewed Report test snapshot")
	}

	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
		path   string
	}{
		{name: "old publication key", mutate: func(root map[string]any) {
			root["publication_key"] = "retired"
		}, path: "publication_key"},
		{name: "source id duplicated inside content", mutate: func(root map[string]any) {
			root["content"].(map[string]any)["source_report_id"] = "retired"
		}, path: "content.source_report_id"},
		{name: "missing persisted cards", mutate: func(root map[string]any) {
			delete(root["content"].(map[string]any), "report_cards")
		}, path: "content.report_cards"},
		{name: "text transmission target", mutate: func(root map[string]any) {
			content := root["content"].(map[string]any)
			path := content["geopolitics"].(map[string]any)["downward_transmission"].(map[string]any)["published_paths"].([]any)[0].(map[string]any)
			delete(path, "target_refs")
			path["target"] = "retired text target"
		}, path: "content.geopolitics.downward_transmission.published_paths[0].target"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var root map[string]any
			if err := json.Unmarshal(payload, &root); err != nil {
				t.Fatal(err)
			}
			test.mutate(root)
			changed, err := json.Marshal(root)
			if err != nil {
				t.Fatal(err)
			}
			var request PublicationRequest
			err = v1.DecodeStrictJSON(changed, publicationShape(), &request)
			if err == nil || !strings.Contains(v1.StrictJSONErrorPath(err), test.path) {
				t.Fatalf("DecodeStrictJSON() error=%v path=%q, want %q", err, v1.StrictJSONErrorPath(err), test.path)
			}
		})
	}
}

func TestReportQueriesRejectDuplicatesUnknownsAndMissingEvidenceScope(t *testing.T) {
	for _, test := range []struct {
		name     string
		query    url.Values
		required []string
		optional []string
		want     bool
	}{
		{name: "empty list query", query: url.Values{}, optional: []string{"published_from", "published_to", "limit", "cursor"}, want: true},
		{name: "single list query", query: url.Values{"limit": {"20"}}, optional: []string{"published_from", "published_to", "limit", "cursor"}, want: true},
		{name: "duplicate list query", query: url.Values{"limit": {"20", "30"}}, optional: []string{"published_from", "published_to", "limit", "cursor"}},
		{name: "unknown list query", query: url.Values{"page": {"1"}}, optional: []string{"published_from", "published_to", "limit", "cursor"}},
		{name: "complete evidence scope", query: url.Values{"scope_type": {"anchor"}, "scope_key": {"geo-anchor"}}, required: []string{"scope_type", "scope_key"}, want: true},
		{name: "missing evidence scope", query: url.Values{"scope_type": {"anchor"}}, required: []string{"scope_type", "scope_key"}},
		{name: "duplicate evidence scope", query: url.Values{"scope_type": {"anchor", "layer"}, "scope_key": {"geo-anchor"}}, required: []string{"scope_type", "scope_key"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validQueryValues(test.query, test.required, test.optional); got != test.want {
				t.Fatalf("validQueryValues()=%t want=%t", got, test.want)
			}
		})
	}
}

func publicationPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/report-publication.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	return payload
}
