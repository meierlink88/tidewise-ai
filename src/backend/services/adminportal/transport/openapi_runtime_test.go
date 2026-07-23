package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	adminapi "github.com/meierlink88/tidewise-ai/backend/services/adminportal/api"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/usecase"
	"gopkg.in/yaml.v3"
)

func TestRuntimeRoutesMatchOpenAPI(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(adminapi.Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	openAPIRoutes := adminOpenAPIRoutes(t, document)
	runtimeRoutes := map[string]struct{}{}
	router := NewRouter(testConfig(), usecase.NewService(&dataclient.Fake{}), "test-token")
	for _, route := range router.Routes() {
		if route.Method == "OPTIONS" {
			continue
		}
		runtimeRoutes[route.Method+" "+route.Path] = struct{}{}
	}
	assertAdminRouteSetsEqual(t, runtimeRoutes, openAPIRoutes)
}

func TestResponseDTOFieldsMatchOpenAPI(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(adminapi.Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	for schemaName, dto := range map[string]any{
		"RawDocumentPage": rawDocumentListResponse{},
		"RawDocument":     rawDocumentResponse{},
		"EventPage":       eventListResponse{},
		"Event":           eventResponse{},
	} {
		assertAdminSchemaFields(t, document, schemaName, dto)
	}
}

func TestOperationalResponseFieldsMatchOpenAPI(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(adminapi.Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	router := NewRouter(testConfig(), usecase.NewService(&dataclient.Fake{}), "test-token")
	for path, schemaName := range map[string]string{
		"/healthz": "HealthResponse",
		"/readyz":  "ReadinessResponse",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		assertAdminJSONFields(t, document, schemaName, body)
	}
}

func adminOpenAPIRoutes(t *testing.T, document map[string]any) map[string]struct{} {
	t.Helper()
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI paths = %#v", document["paths"])
	}
	routes := map[string]struct{}{}
	for path, value := range paths {
		pathItem, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI path %s = %#v", path, value)
		}
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			if _, exists := pathItem[method]; exists {
				routes[strings.ToUpper(method)+" "+path] = struct{}{}
			}
		}
	}
	return routes
}

func assertAdminRouteSetsEqual(t *testing.T, runtimeRoutes, openAPIRoutes map[string]struct{}) {
	t.Helper()
	for route := range runtimeRoutes {
		if _, exists := openAPIRoutes[route]; !exists {
			t.Errorf("runtime route %q is missing from OpenAPI", route)
		}
	}
	for route := range openAPIRoutes {
		if _, exists := runtimeRoutes[route]; !exists {
			t.Errorf("OpenAPI route %q is missing from runtime", route)
		}
	}
}

func assertAdminSchemaFields(t *testing.T, document map[string]any, schemaName string, dto any) {
	t.Helper()
	components := document["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	schema := schemas[schemaName].(map[string]any)
	properties := schema["properties"].(map[string]any)
	want := adminJSONFieldNames(reflect.TypeOf(dto))
	got := make([]string, 0, len(properties))
	for name := range properties {
		got = append(got, name)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s fields = %v, DTO json fields = %v", schemaName, got, want)
	}
}

func assertAdminJSONFields(t *testing.T, document map[string]any, schemaName string, body map[string]any) {
	t.Helper()
	components := document["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	schema := schemas[schemaName].(map[string]any)
	properties := schema["properties"].(map[string]any)
	got := make([]string, 0, len(body))
	for name := range body {
		got = append(got, name)
	}
	want := make([]string, 0, len(properties))
	for name := range properties {
		want = append(want, name)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s fields = %v, response JSON fields = %v", schemaName, want, got)
	}
}

func adminJSONFieldNames(value reflect.Type) []string {
	names := make([]string, 0, value.NumField())
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		tag := field.Tag.Get("json")
		if field.Anonymous && tag == "" {
			names = append(names, adminJSONFieldNames(field.Type)...)
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
