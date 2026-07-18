package dataclient

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIContractMatchesAdminTypedClient(t *testing.T) {
	document := loadOpenAPI(t)
	for _, contract := range []struct {
		path        string
		operationID string
		response    string
		parameters  []string
	}{
		{
			AdminRawDocumentsPath,
			"listAdminRawDocuments",
			"AdminRawDocumentPageEnvelope",
			[]string{"Page", "PageSize", "RequestID", "ingest_status", "source_id", "title"},
		},
		{
			AdminEventsPath,
			"listAdminEvents",
			"AdminEventPageEnvelope",
			[]string{"Page", "PageSize", "RequestID", "event_status", "event_time_from", "event_time_to", "fact_status", "first_seen_from", "first_seen_to", "title"},
		},
		{
			AdminSourceCatalogsPath,
			"listAdminSourceCatalogs",
			"AdminSourceCatalogCollectionEnvelope",
			[]string{"RequestID", "SourceStatus"},
		},
	} {
		operation := openAPIOperation(t, document, contract.path)
		assertOpenAPIString(t, operation, "operationId", contract.operationID)
		assertOpenAPIString(t, operation, "x-retry-policy", "safe-get")
		if got := operationParameterNames(t, operation); !reflect.DeepEqual(got, contract.parameters) {
			t.Fatalf("parameters for %s = %v, want %v", contract.path, got, contract.parameters)
		}
		response := openAPIObject(t, openAPIObject(t, operation["responses"], "responses")["200"], "200 response")
		content := openAPIObject(t, response["content"], "200 content")
		media := openAPIObject(t, content["application/json"], "application/json response")
		assertOpenAPIString(t, openAPIObject(t, media["schema"], "response schema"), "$ref", "#/components/schemas/"+contract.response)
	}

	for schemaName, dataType := range map[string]reflect.Type{
		"AdminRawDocumentPage":         reflect.TypeOf(RawDocumentPage{}),
		"AdminRawDocument":             reflect.TypeOf(RawDocument{}),
		"AdminEventPage":               reflect.TypeOf(EventPage{}),
		"AdminEvent":                   reflect.TypeOf(Event{}),
		"AdminSourceCatalogCollection": reflect.TypeOf(SourceCatalogCollection{}),
		"AdminSourceCatalog":           reflect.TypeOf(SourceCatalog{}),
	} {
		assertDTOJSONFieldsMatchSchema(t, document, schemaName, dataType)
	}

	assertComponentEnum(t, document, "IngestStatus", []string{"collected", "duplicate", "failed", "pending_extract"})
	assertComponentEnum(t, document, "EventStatus", []string{"candidate", "confirmed", "rejected"})
	assertComponentEnum(t, document, "FactStatus", []string{"disputed", "unverified", "verified"})
	assertComponentEnum(t, document, "SourceStatus", []string{"active", "disabled", "inactive"})
}

func loadOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	path := filepath.Join("..", "..", "data", "api", "openapi.yaml")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(payload, &document); err != nil {
		t.Fatalf("decode OpenAPI contract: %v", err)
	}
	return document
}

func openAPIOperation(t *testing.T, document map[string]any, path string) map[string]any {
	t.Helper()
	paths := openAPIObject(t, document["paths"], "paths")
	pathItem := openAPIObject(t, paths[path], "path "+path)
	return openAPIObject(t, pathItem["get"], "GET "+path)
}

func operationParameterNames(t *testing.T, operation map[string]any) []string {
	t.Helper()
	raw, ok := operation["parameters"].([]any)
	if !ok {
		t.Fatalf("parameters = %T, want array", operation["parameters"])
	}
	names := make([]string, 0, len(raw))
	for _, value := range raw {
		parameter := openAPIObject(t, value, "parameter")
		if reference, ok := parameter["$ref"].(string); ok {
			names = append(names, reference[strings.LastIndex(reference, "/")+1:])
			continue
		}
		name, ok := parameter["name"].(string)
		if !ok {
			t.Fatalf("parameter has no name or $ref: %v", parameter)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func assertDTOJSONFieldsMatchSchema(t *testing.T, document map[string]any, schemaName string, dataType reflect.Type) {
	t.Helper()
	components := openAPIObject(t, document["components"], "components")
	schemas := openAPIObject(t, components["schemas"], "schemas")
	schema := openAPIObject(t, schemas[schemaName], "schema "+schemaName)
	properties := openAPIObject(t, schema["properties"], "properties for "+schemaName)
	want := make([]string, 0, len(properties))
	for name := range properties {
		want = append(want, name)
	}
	sort.Strings(want)

	got := make([]string, 0, dataType.NumField())
	for index := 0; index < dataType.NumField(); index++ {
		tag := strings.Split(dataType.Field(index).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			got = append(got, tag)
		}
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s JSON fields = %v, OpenAPI properties = %v", dataType.Name(), got, want)
	}
}

func assertComponentEnum(t *testing.T, document map[string]any, schemaName string, want []string) {
	t.Helper()
	components := openAPIObject(t, document["components"], "components")
	schemas := openAPIObject(t, components["schemas"], "schemas")
	schema := openAPIObject(t, schemas[schemaName], "schema "+schemaName)
	raw, ok := schema["enum"].([]any)
	if !ok {
		t.Fatalf("%s enum = %T, want array", schemaName, schema["enum"])
	}
	got := make([]string, 0, len(raw))
	for _, value := range raw {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("%s enum value = %T, want string", schemaName, value)
		}
		got = append(got, text)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s enum = %v, want %v", schemaName, got, want)
	}
}

func openAPIObject(t *testing.T, value any, label string) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s = %T, want object", label, value)
	}
	return object
}

func assertOpenAPIString(t *testing.T, object map[string]any, key string, want string) {
	t.Helper()
	if got, ok := object[key].(string); !ok || got != want {
		t.Fatalf("%s = %v, want %q", key, object[key], want)
	}
}
