package v1

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIContractFreezesAdminRoutesSecurityAndEnvelopes(t *testing.T) {
	document := parseAdminDocument(t)
	if document["openapi"] != "3.0.4" {
		t.Fatalf("openapi = %v, want 3.0.4", document["openapi"])
	}
	servers := adminArray(t, document["servers"], "servers")
	if len(servers) != 1 || adminObject(t, servers[0], "server")["url"] != "/" {
		t.Fatalf("servers = %#v, want relative root", servers)
	}
	security := adminArray(t, document["security"], "security")
	if len(security) != 1 {
		t.Fatalf("global security = %#v", security)
	}
	if _, exists := adminObject(t, security[0], "security[0]")["AdminBearer"]; !exists {
		t.Fatalf("global security = %#v, want AdminBearer", security)
	}

	paths := adminObject(t, document["paths"], "paths")
	type operationExpectation struct {
		method      string
		operationID string
		envelope    string
		statuses    []string
	}
	want := map[string][]operationExpectation{
		"/healthz": {
			{method: "get", operationID: "getAdminPortalHealth"},
		},
		"/readyz": {
			{method: "get", operationID: "getAdminPortalReadiness"},
		},
		"/api/admin/v1/events": {
			{method: "get", operationID: "listAdminPortalEvents", envelope: "EventPageEnvelope", statuses: []string{"400", "401", "403", "500", "503"}},
		},
		"/api/admin/v1/evidences": {
			{method: "get", operationID: "listAdminPortalEvidences", envelope: "EvidencePageEnvelope", statuses: []string{"400", "401", "403", "500", "503"}},
		},
		"/api/admin/v1/raw-evidences/{raw_evidence_id}/collection-document": {
			{method: "get", operationID: "getAdminPortalCollectionDocument", envelope: "CollectionDocumentEnvelope", statuses: []string{"400", "401", "403", "404", "500", "503"}},
		},
		"/api/admin/v1/evidence-categories": {
			{method: "get", operationID: "listAdminPortalEvidenceCategories", envelope: "EvidenceCategoryListEnvelope", statuses: []string{"400", "401", "403", "500", "503"}},
		},
		"/api/admin/v1/sources": {
			{method: "get", operationID: "listAdminPortalSources", envelope: "SourcePageEnvelope", statuses: []string{"400", "401", "403", "500", "503"}},
		},
		"/api/admin/v1/runtime-health": {
			{method: "get", operationID: "getAdminPortalRuntimeHealth", envelope: "RuntimeHealthEnvelope", statuses: []string{"401", "403", "500"}},
		},
	}
	if len(paths) != len(want) {
		t.Fatalf("path count = %d, want %d", len(paths), len(want))
	}
	seen := map[string]bool{}
	for path, expectations := range want {
		pathItem := adminObject(t, paths[path], "path "+path)
		for _, expected := range expectations {
			operation := adminObject(t, pathItem[expected.method], strings.ToUpper(expected.method)+" "+path)
			operationID, _ := operation["operationId"].(string)
			if operationID != expected.operationID || seen[operationID] {
				t.Fatalf("%s %s operationId = %q, duplicate=%v", strings.ToUpper(expected.method), path, operationID, seen[operationID])
			}
			seen[operationID] = true
			if path == "/healthz" || path == "/readyz" {
				override := adminArray(t, operation["security"], "operation security")
				if len(override) != 0 {
					t.Fatalf("GET %s must disable global auth", path)
				}
				continue
			}
			responses := adminObject(t, operation["responses"], "responses")
			schema := adminResponseSchema(t, document, responses["200"])
			if schema["$ref"] != "#/components/schemas/"+expected.envelope {
				t.Fatalf("%s %s success schema = %v", strings.ToUpper(expected.method), path, schema["$ref"])
			}
			for _, status := range expected.statuses {
				if _, exists := responses[status]; !exists {
					t.Fatalf("%s %s missing %s response", strings.ToUpper(expected.method), path, status)
				}
			}
		}
	}

	schemes := adminObject(t, adminObject(t, document["components"], "components")["securitySchemes"], "securitySchemes")
	bearer := adminObject(t, schemes["AdminBearer"], "AdminBearer")
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Fatalf("AdminBearer = %#v", bearer)
	}
	assertAdminRequired(t, adminSchema(t, document, "ErrorEnvelope"), "error", "request_id")
	assertAdminRequired(t, adminSchema(t, document, "ErrorDetail"), "code", "message", "details")
	assertAdminNoDanglingLocalReferences(t, document)
}

func TestOpenAPIContractPreservesRetainedDataListSchemas(t *testing.T) {
	document := parseAdminDocument(t)
	evidenceOperation := adminObject(t, adminObject(t, adminObject(t, document["paths"], "paths")["/api/admin/v1/evidences"], "Evidence path")["get"], "Evidence operation")
	foundSourceID := false
	for _, value := range adminArray(t, evidenceOperation["parameters"], "Evidence parameters") {
		parameter := adminObject(t, value, "Evidence parameter")
		if parameter["name"] != "source_id" {
			continue
		}
		foundSourceID = true
		if parameter["x-trimmed-max-length"] != 32 {
			t.Fatalf("source_id x-trimmed-max-length = %v, want 32", parameter["x-trimmed-max-length"])
		}
		parameterSchema := adminObject(t, parameter["schema"], "source_id schema")
		if _, exists := parameterSchema["maxLength"]; exists {
			t.Fatal("source_id query maxLength must apply after trimming, not to the raw query value")
		}
	}
	if !foundSourceID {
		t.Fatal("Evidence parameters do not contain source_id")
	}
	components := adminObject(t, document["components"], "components")
	parameters := adminObject(t, components["parameters"], "parameters")
	pageSchema := adminObject(t, adminObject(t, parameters["Page"], "Page")["schema"], "Page.schema")
	if pageSchema["maximum"] != 1000000 {
		t.Fatalf("Page schema = %#v, want maximum 1000000", pageSchema)
	}
	pageSizeSchema := adminObject(t, adminObject(t, parameters["PageSize"], "PageSize")["schema"], "PageSize.schema")
	if pageSizeSchema["default"] != 50 || pageSizeSchema["maximum"] != 100 {
		t.Fatalf("PageSize schema = %#v, want retained default 50 and maximum 100", pageSizeSchema)
	}

	for name, want := range map[string][]string{
		"EventModality":        {"FACT", "PLAN", "SPEC"},
		"EventLifecycleStatus": {"ACTIVE", "DEPRECATED", "ARCHIVED"},
	} {
		schema := adminSchema(t, document, name)
		values := adminArray(t, schema["enum"], name+".enum")
		if len(values) != len(want) {
			t.Fatalf("%s enum = %v, want %v", name, values, want)
		}
		for index, expected := range want {
			if values[index] != expected {
				t.Fatalf("%s enum = %v, want %v", name, values, want)
			}
		}
	}

	event := adminSchema(t, document, "Event")
	if event["additionalProperties"] != false {
		t.Fatalf("Event additionalProperties = %v, want false", event["additionalProperties"])
	}
	assertAdminRequired(t, event, "id", "title", "summary", "semantic", "modality", "occurred_at", "announced_at", "status")
	assertAdminRequired(t, adminSchema(t, document, "EventSemantic"), "actors", "action", "objects", "stage", "jurisdictions", "effective_at", "time_precision")
	assertAdminRequired(t, adminSchema(t, document, "Evidence"), "id", "raw_evidence_id", "title", "summary", "semantic", "categories", "source_id", "source_name", "source_level", "source_url", "is_original", "quoted_source_name", "keywords", "is_split", "published_at", "collected_at")
	assertAdminRequired(t, adminSchema(t, document, "CollectionDocument"), "available", "url")
	source := adminSchema(t, document, "Source")
	assertAdminRequired(t, source, "id", "code", "name", "ownership_type", "channel_type", "enabled", "priority", "default_source_level", "updated_at")
	properties := adminObject(t, source["properties"], "Source.properties")
	for _, forbidden := range []string{"adapter_key", "endpoint", "app_key", "config", "timeout_seconds", "max_results"} {
		if _, exists := properties[forbidden]; exists {
			t.Fatalf("Admin Source exposes %q", forbidden)
		}
	}
}

func parseAdminDocument(t *testing.T) map[string]any {
	t.Helper()
	var document map[string]any
	if err := yaml.Unmarshal(Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	return document
}

func adminResponseSchema(t *testing.T, document map[string]any, value any) map[string]any {
	t.Helper()
	response := adminObject(t, value, "response")
	if ref, ok := response["$ref"].(string); ok && strings.HasPrefix(ref, "#/components/responses/") {
		name := strings.TrimPrefix(ref, "#/components/responses/")
		components := adminObject(t, document["components"], "components")
		response = adminObject(t, adminObject(t, components["responses"], "responses")[name], "response "+name)
	}
	content := adminObject(t, response["content"], "content")
	media := adminObject(t, content["application/json"], "application/json")
	return adminObject(t, media["schema"], "schema")
}

func adminSchema(t *testing.T, document map[string]any, name string) map[string]any {
	t.Helper()
	components := adminObject(t, document["components"], "components")
	return adminObject(t, adminObject(t, components["schemas"], "schemas")[name], "schema "+name)
}

func assertAdminRequired(t *testing.T, value map[string]any, names ...string) {
	t.Helper()
	required := adminArray(t, value["required"], "required")
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

func assertAdminNoDanglingLocalReferences(t *testing.T, document map[string]any) {
	t.Helper()
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if ref, ok := typed["$ref"].(string); ok && strings.HasPrefix(ref, "#/components/") {
				assertAdminLocalReferenceResolves(t, document, ref)
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

func assertAdminLocalReferenceResolves(t *testing.T, document map[string]any, ref string) {
	t.Helper()
	parts := strings.Split(strings.TrimPrefix(ref, "#/components/"), "/")
	if len(parts) != 2 {
		t.Fatalf("unsupported local reference %q", ref)
	}
	components := adminObject(t, document["components"], "components")
	section := adminObject(t, components[parts[0]], "components."+parts[0])
	if _, exists := section[parts[1]]; !exists {
		t.Fatalf("local reference %q does not resolve", ref)
	}
}

func adminObject(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", name, value)
	}
	return result
}

func adminArray(t *testing.T, value any, name string) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", name, value)
	}
	return result
}
