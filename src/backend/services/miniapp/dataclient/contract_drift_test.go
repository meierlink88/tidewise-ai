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

func TestOpenAPIContractMatchesMiniappTypedClient(t *testing.T) {
	document := loadOpenAPI(t)
	for _, contract := range []struct {
		path        string
		operationID string
		response    string
		parameters  []string
	}{
		{ResearchThemesPath, "listResearchThemes", "ResearchThemeListEnvelope", []string{"Cursor", "Limit", "RequestID", "WindowHours"}},
		{ResearchThemesPath + "/{theme_id}", "getResearchTheme", "ResearchThemeDetailEnvelope", []string{"RequestID", "WindowHours"}},
		{ResearchThemesPath + "/{theme_id}/reasoning-trees", "listResearchThemeReasoningTrees", "ResearchReasoningTreeListEnvelope", []string{"RequestID"}},
		{ResearchThemesPath + "/{theme_id}/reasoning-trees/{anchor_id}", "getResearchThemeReasoningTree", "ResearchReasoningTreeDetailEnvelope", []string{"RequestID"}},
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
		"ResearchThemeCollection":        reflect.TypeOf(ResearchThemePage{}),
		"ResearchThemeSummary":           reflect.TypeOf(ResearchTheme{}),
		"ResearchThemeDetail":            reflect.TypeOf(ResearchThemeDetail{}),
		"ResearchThemeChainNode":         reflect.TypeOf(ResearchThemeChainNode{}),
		"ResearchIndex":                  reflect.TypeOf(ResearchIndex{}),
		"ResearchEvent":                  reflect.TypeOf(ResearchEvent{}),
		"ResearchReasoningTreeChainNode": reflect.TypeOf(ResearchReasoningTreeChainNode{}),
		"ResearchReasoningTreeSummary":   reflect.TypeOf(ResearchReasoningTreeSummary{}),
		"ResearchReasoningTreeList":      reflect.TypeOf(ResearchReasoningTreeList{}),
		"ResearchReasoningTreeEvent":     reflect.TypeOf(ResearchReasoningTreeEvent{}),
		"ResearchReasoningTreePathNode":  reflect.TypeOf(ResearchReasoningTreePathNode{}),
		"ResearchReasoningTree":          reflect.TypeOf(ResearchReasoningTree{}),
		"ResearchReasoningTreeDetail":    reflect.TypeOf(ResearchReasoningTreeDetail{}),
	} {
		assertDTOJSONFieldsMatchSchema(t, document, schemaName, dataType)
	}

	assertSchemaEnum(t, document, "ResearchThemeSummary", "impact_level", []string{"focus", "high", "watch"})
	assertSchemaHasNoEnum(t, document, "ResearchThemeSummary", "trading_direction")
	assertSchemaEnum(t, document, "ResearchThemeSummary", "transmission_stage", []string{"dampening", "diffusion", "identification", "validation"})
	assertSchemaEnum(t, document, "ResearchIndex", "impact_direction", []string{"mixed", "negative", "neutral", "positive"})
	assertSchemaEnum(t, document, "ResearchEvent", "evidence_role", []string{"context", "contradicting", "driver", "supporting"})
	assertArrayItemSchema(t, document, "ResearchThemeSummary", "affected_chain_nodes", "ResearchThemeChainNode")
	assertSchemaEnum(t, document, "ResearchReasoningTreeEvent", "evidence_role", []string{"context", "contradicting", "driver", "supporting"})
	assertSchemaEnum(t, document, "ResearchReasoningTreePathNode", "change_direction", []string{"decrease", "increase", "mixed", "unchanged", "uncertain"})
	assertArrayItemSchema(t, document, "ResearchReasoningTreeList", "reasoning_trees", "ResearchReasoningTreeSummary")
	assertArrayItemSchema(t, document, "ResearchReasoningTree", "events", "ResearchReasoningTreeEvent")
	assertArrayItemSchema(t, document, "ResearchReasoningTree", "path_nodes", "ResearchReasoningTreePathNode")
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

func assertSchemaEnum(t *testing.T, document map[string]any, schemaName string, propertyName string, want []string) {
	t.Helper()
	components := openAPIObject(t, document["components"], "components")
	schemas := openAPIObject(t, components["schemas"], "schemas")
	schema := openAPIObject(t, schemas[schemaName], "schema "+schemaName)
	properties := openAPIObject(t, schema["properties"], "properties for "+schemaName)
	property := openAPIObject(t, properties[propertyName], "property "+propertyName)
	raw, ok := property["enum"].([]any)
	if !ok {
		t.Fatalf("%s.%s enum = %T, want array", schemaName, propertyName, property["enum"])
	}
	got := make([]string, 0, len(raw))
	for _, value := range raw {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("%s.%s enum value = %T, want string", schemaName, propertyName, value)
		}
		got = append(got, text)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s.%s enum = %v, want %v", schemaName, propertyName, got, want)
	}
}

func assertSchemaHasNoEnum(t *testing.T, document map[string]any, schemaName string, propertyName string) {
	t.Helper()
	components := openAPIObject(t, document["components"], "components")
	schemas := openAPIObject(t, components["schemas"], "schemas")
	schema := openAPIObject(t, schemas[schemaName], "schema "+schemaName)
	properties := openAPIObject(t, schema["properties"], "properties for "+schemaName)
	property := openAPIObject(t, properties[propertyName], "property "+propertyName)
	if _, ok := property["enum"]; ok {
		t.Fatalf("%s.%s must remain a natural-language string without enum", schemaName, propertyName)
	}
}

func assertArrayItemSchema(t *testing.T, document map[string]any, schemaName string, propertyName string, itemSchema string) {
	t.Helper()
	components := openAPIObject(t, document["components"], "components")
	schemas := openAPIObject(t, components["schemas"], "schemas")
	schema := openAPIObject(t, schemas[schemaName], "schema "+schemaName)
	properties := openAPIObject(t, schema["properties"], "properties for "+schemaName)
	property := openAPIObject(t, properties[propertyName], "property "+propertyName)
	items := openAPIObject(t, property["items"], "items for "+schemaName+"."+propertyName)
	assertOpenAPIString(t, items, "$ref", "#/components/schemas/"+itemSchema)
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
