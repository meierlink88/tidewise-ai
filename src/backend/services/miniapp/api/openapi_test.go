package api

import (
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIContractFreezesMiniappRoutesAndEnvelopes(t *testing.T) {
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
	want := map[string]struct {
		operationID string
		envelope    string
	}{
		"/healthz":                        {operationID: "getMiniappHealth"},
		"/readyz":                         {operationID: "getMiniappReadiness"},
		"/api/miniapp/v1/research/themes": {operationID: "listMiniappResearchThemes", envelope: "ResearchThemePageEnvelope"},
		"/api/miniapp/v1/research/themes/{theme_id}":                             {operationID: "getMiniappResearchTheme", envelope: "ResearchThemeDetailEnvelope"},
		"/api/miniapp/v1/research/themes/{theme_id}/reasoning-trees":             {operationID: "listMiniappResearchReasoningTrees", envelope: "ReasoningTreeListEnvelope"},
		"/api/miniapp/v1/research/themes/{theme_id}/reasoning-trees/{anchor_id}": {operationID: "getMiniappResearchReasoningTree", envelope: "ReasoningTreeDetailEnvelope"},
	}
	if len(paths) != len(want) {
		t.Fatalf("paths = %v, want %v", sortedKeys(paths), sortedKeysFromContract(want))
	}
	seenOperations := map[string]bool{}
	for path, expected := range want {
		operation := object(t, object(t, paths[path], "path "+path)["get"], "GET "+path)
		operationID, _ := operation["operationId"].(string)
		if operationID != expected.operationID || seenOperations[operationID] {
			t.Fatalf("GET %s operationId = %q, duplicate=%v", path, operationID, seenOperations[operationID])
		}
		seenOperations[operationID] = true
		if expected.envelope == "" {
			continue
		}
		responses := object(t, operation["responses"], "responses")
		schema := responseSchema(t, document, responses["200"])
		if schema["$ref"] != "#/components/schemas/"+expected.envelope {
			t.Fatalf("GET %s success schema = %v", path, schema["$ref"])
		}
		statuses := []string{"400", "404", "500"}
		if strings.Contains(path, "/reasoning-trees") {
			statuses = append(statuses, "502")
		}
		for _, status := range statuses {
			response, exists := responses[status]
			if !exists {
				t.Fatalf("GET %s is missing response %s", path, status)
			}
			if ref := object(t, response, status)["$ref"]; ref != "#/components/responses/"+errorResponseName(status) {
				t.Fatalf("GET %s response %s = %v", path, status, ref)
			}
		}
	}

	errorEnvelope := schema(t, document, "ErrorEnvelope")
	assertRequired(t, errorEnvelope, "error", "request_id")
	errorDetail := schema(t, document, "ErrorDetail")
	assertRequired(t, errorDetail, "code", "message", "details")
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

func responseSchema(t *testing.T, document map[string]any, value any) map[string]any {
	t.Helper()
	response := object(t, value, "response")
	if ref, ok := response["$ref"].(string); ok && strings.HasPrefix(ref, "#/components/responses/") {
		name := strings.TrimPrefix(ref, "#/components/responses/")
		components := object(t, document["components"], "components")
		response = object(t, object(t, components["responses"], "responses")[name], "response "+name)
	}
	content := object(t, response["content"], "response content")
	media := object(t, content["application/json"], "application/json")
	return object(t, media["schema"], "response schema")
}

func errorResponseName(status string) string {
	return map[string]string{"400": "BadRequest", "404": "NotFound", "500": "InternalError", "502": "BadGateway"}[status]
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

func sortedKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysFromContract(value map[string]struct {
	operationID string
	envelope    string
}) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
