package service

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	dataapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"gopkg.in/yaml.v3"
)

func TestResponseDTOFieldsMatchOpenAPI(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(dataapi.Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	for schemaName, dto := range map[string]any{
		"ResearchThemeCollection":            dataapi.ResearchThemePage{},
		"ResearchThemeSummary":               dataapi.ResearchTheme{},
		"ResearchThemeDetail":                dataapi.ResearchThemeDetail{},
		"ResearchThemeChainNode":             dataapi.ResearchThemeChainNode{},
		"ResearchIndex":                      dataapi.ResearchIndex{},
		"ResearchEvent":                      dataapi.ResearchEvent{},
		"ResearchReasoningTreeList":          dataapi.ResearchReasoningTreeList{},
		"ResearchReasoningTreeSummary":       dataapi.ResearchReasoningTreeSummary{},
		"ResearchReasoningTreeDetail":        dataapi.ResearchReasoningTreeDetail{},
		"ResearchReasoningTree":              dataapi.ResearchReasoningTree{},
		"ResearchReasoningTreeChainNode":     dataapi.ResearchReasoningTreeChainNode{},
		"ResearchReasoningTreeEvent":         dataapi.ResearchReasoningTreeEvent{},
		"ResearchReasoningTreePathNode":      dataapi.ResearchReasoningTreePathNode{},
		"AdminRawDocumentPage":               dataapi.AdminRawDocumentPage{},
		"AdminRawDocument":                   dataapi.AdminRawDocument{},
		"AdminEventPage":                     dataapi.AdminEventPage{},
		"AdminEvent":                         dataapi.AdminEvent{},
		"EventPublicationRequest":            dataapi.EventPublicationRequest{},
		"EventPublicationProvenance":         dataapi.EventPublicationProvenance{},
		"EventPublicationCollectorExecution": dataapi.EventPublicationCollectorExecution{},
		"EventPublicationRawDocument":        dataapi.EventPublicationRawDocument{},
		"EventPublicationEvent":              dataapi.EventPublicationEvent{},
		"EventPublicationEvidence":           dataapi.EventPublicationEvidence{},
		"EventPublicationTag":                dataapi.EventPublicationTag{},
		"EventPublicationReview":             dataapi.EventPublicationReview{},
		"EventPublicationResult":             dataapi.EventPublicationResult{},
		"EventPublicationEventResult":        dataapi.EventPublicationEventResult{},
		"EventPublicationRawDocumentResult":  dataapi.EventPublicationRawDocumentResult{},
		"EventPublicationCounts":             dataapi.EventPublicationCounts{},
		"ResearchThemeImportRequest":         dataapi.ResearchThemeImportRequest{},
		"ResearchThemeImportItem":            dataapi.ResearchThemeImportItem{},
		"ResearchThemeImportChainNode":       dataapi.ResearchThemeImportChainNode{},
		"ResearchThemeImportEvent":           dataapi.ResearchThemeImportEvent{},
		"ResearchThemeImportCounts":          dataapi.ResearchThemeImportCounts{},
		"ResearchThemeImportResult":          dataapi.ResearchThemeImportResult{},
		"ResearchAnchorImportRequest":        dataapi.ResearchAnchorImportRequest{},
		"ResearchAnchorImportItem":           dataapi.ResearchAnchorImportItem{},
		"ResearchAnchorImportEvent":          dataapi.ResearchAnchorImportEvent{},
		"ResearchAnchorImportPathNode":       dataapi.ResearchAnchorImportPathNode{},
		"ResearchAnchorImportCounts":         dataapi.ResearchAnchorImportCounts{},
		"ResearchAnchorImportResult":         dataapi.ResearchAnchorImportResult{},
	} {
		assertDataSchemaFields(t, document, schemaName, dto)
	}
}

func openAPIBusinessRoutes(t *testing.T, document map[string]any) map[string]struct{} {
	t.Helper()
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI paths = %#v", document["paths"])
	}
	routes := map[string]struct{}{}
	for path, value := range paths {
		if path == "/healthz" || path == "/readyz" {
			continue
		}
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

func assertRouteSetsEqual(t *testing.T, runtimeRoutes, openAPIRoutes map[string]struct{}) {
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

func assertDataSchemaFields(t *testing.T, document map[string]any, schemaName string, dto any) {
	t.Helper()
	components := document["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	schema := schemas[schemaName].(map[string]any)
	properties := schema["properties"].(map[string]any)
	want := dataJSONFieldNames(reflect.TypeOf(dto))
	got := make([]string, 0, len(properties))
	for name := range properties {
		got = append(got, name)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s fields = %v, DTO json fields = %v", schemaName, got, want)
	}
}

func dataJSONFieldNames(value reflect.Type) []string {
	names := make([]string, 0, value.NumField())
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		tag := field.Tag.Get("json")
		if field.Anonymous && tag == "" {
			names = append(names, dataJSONFieldNames(field.Type)...)
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
