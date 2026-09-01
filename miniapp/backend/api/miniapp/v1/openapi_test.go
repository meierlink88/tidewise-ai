package v1

import (
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIContractExposesOperationsAndReportRoutes(t *testing.T) {
	document := parseDocument(t)
	if document["openapi"] != "3.0.4" {
		t.Fatalf("openapi = %v, want 3.0.4", document["openapi"])
	}
	servers := array(t, document["servers"], "servers")
	if len(servers) != 1 || object(t, servers[0], "server")["url"] != "/" {
		t.Fatalf("servers = %#v, want relative root", servers)
	}
	if _, exists := document["security"]; exists {
		t.Fatal("Miniapp V1 must not declare an authentication scheme")
	}

	paths := object(t, document["paths"], "paths")
	want := map[string]string{
		"/healthz":                     "getMiniappHealth",
		"/readyz":                      "getMiniappReadiness",
		"/api/miniapp/v1/reports/home": "getReportHome",
		"/api/miniapp/v1/reports/{report_id}/layers/{layer_key}":          "getReportLayer",
		"/api/miniapp/v1/reports/{report_id}/industry-chains/{chain_key}": "getReportIndustryChain",
		"/api/miniapp/v1/reports/{report_id}/evidences":                   "listReportEvidences",
	}
	if len(paths) != len(want) {
		t.Fatalf("paths = %v, want %v", sortedKeys(paths), sortedKeys(want))
	}
	seenOperations := map[string]bool{}
	for path, expectedOperation := range want {
		operation := object(t, object(t, paths[path], "path "+path)["get"], "GET "+path)
		operationID, _ := operation["operationId"].(string)
		if operationID != expectedOperation || seenOperations[operationID] {
			t.Fatalf("GET %s operationId = %q, duplicate=%v", path, operationID, seenOperations[operationID])
		}
		seenOperations[operationID] = true
		responses := object(t, operation["responses"], "responses")
		response := object(t, responses["200"], "200 response")
		content := object(t, response["content"], "response content")
		media := object(t, content["application/json"], "application/json")
		responseSchema := object(t, media["schema"], "response schema")
		wantSchema := map[string]string{
			"/healthz":                     "#/components/schemas/HealthResponse",
			"/readyz":                      "#/components/schemas/ReadinessResponse",
			"/api/miniapp/v1/reports/home": "#/components/schemas/HomeEnvelope",
			"/api/miniapp/v1/reports/{report_id}/layers/{layer_key}":          "#/components/schemas/LayerEnvelope",
			"/api/miniapp/v1/reports/{report_id}/industry-chains/{chain_key}": "#/components/schemas/IndustryChainEnvelope",
			"/api/miniapp/v1/reports/{report_id}/evidences":                   "#/components/schemas/EvidenceEnvelope",
		}[path]
		if responseSchema["$ref"] != wantSchema {
			t.Fatalf("GET %s schema = %v, want %s", path, responseSchema["$ref"], wantSchema)
		}
	}

	assertRequired(t, schema(t, document, "HealthResponse"), "status", "service", "environment")
	assertRequired(t, schema(t, document, "ReadinessResponse"), "status", "service", "environment", "checks")
	assertRequired(t, schema(t, document, "HomeResponse"), "selection", "reports")
	assertRequired(t, schema(t, document, "Summary"), "id", "title", "generated_at", "published_at")
	summaryProperties := object(t, schema(t, document, "Summary")["properties"], "Summary properties")
	if got, want := sortedKeys(summaryProperties), []string{"generated_at", "id", "published_at", "title"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Summary properties = %v, want %v", got, want)
	}
	assertRequired(t, schema(t, document, "Card"), "key", "kind", "display_order", "detail_ref", "impact_items", "has_evidence")
	assertRequired(t, schema(t, document, "Layer"), "reasoning_steps", "downward_transmission", "uncertainty", "scope", "has_evidence")
	assertRequired(t, schema(t, document, "IndustryChain"), "nodes", "edges", "uncertainty", "scope", "has_evidence")
	assertRequired(t, schema(t, document, "EvidenceCollection"), "report_id", "scope", "items")
	evidenceProperties := object(t, schema(t, document, "EvidenceItem")["properties"], "EvidenceItem properties")
	if got, want := sortedKeys(evidenceProperties), []string{"keywords", "published_at", "summary"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("EvidenceItem properties = %v, want %v", got, want)
	}
	for _, retired := range []string{"/api/miniapp/v1/research/themes", "/api/miniapp/v1/reasoning-trees", "/api/miniapp/v1/events"} {
		if _, exists := paths[retired]; exists {
			t.Fatalf("retired path %q remains in OpenAPI", retired)
		}
	}
	assertNoDanglingLocalReferences(t, document)
}

func parseDocument(t *testing.T) map[string]any {
	t.Helper()
	var document map[string]any
	if err := yaml.Unmarshal(Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	return document
}

func schema(t *testing.T, document map[string]any, name string) map[string]any {
	t.Helper()
	components := object(t, document["components"], "components")
	return object(t, object(t, components["schemas"], "schemas")[name], "schema "+name)
}

func assertRequired(t *testing.T, value map[string]any, names ...string) {
	t.Helper()
	required := array(t, value["required"], "required")
	got := map[string]bool{}
	for _, item := range required {
		name, ok := item.(string)
		if !ok {
			t.Fatalf("required item = %#v", item)
		}
		got[name] = true
	}
	for _, name := range names {
		if !got[name] {
			t.Fatalf("required = %v, missing %q", required, name)
		}
	}
}

func assertNoDanglingLocalReferences(t *testing.T, document map[string]any) {
	t.Helper()
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if ref, ok := typed["$ref"].(string); ok && strings.HasPrefix(ref, "#/components/") {
				assertLocalReferenceResolves(t, document, ref)
			}
			for _, child := range typed {
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(document)
}

func assertLocalReferenceResolves(t *testing.T, document map[string]any, ref string) {
	t.Helper()
	parts := strings.Split(strings.TrimPrefix(ref, "#/components/"), "/")
	if len(parts) != 2 {
		t.Fatalf("unsupported local reference %q", ref)
	}
	components := object(t, document["components"], "components")
	section := object(t, components[parts[0]], "components."+parts[0])
	if _, exists := section[parts[1]]; !exists {
		t.Fatalf("local reference %q does not resolve", ref)
	}
}

func object(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", name, value)
	}
	return result
}

func array(t *testing.T, value any, name string) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", name, value)
	}
	return result
}

func sortedKeys[T any](value map[string]T) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
