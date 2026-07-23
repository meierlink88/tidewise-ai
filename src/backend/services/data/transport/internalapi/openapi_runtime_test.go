package internalapi

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	dataapi "github.com/meierlink88/tidewise-ai/backend/services/data/api"
	publicationdomain "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventpublication"
	researchanchordomain "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchanchorimport"
	researchthemedomain "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchthemeimport"
	eventpublicationapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/eventpublication"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/research"
	researchanchorimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchanchorimport"
	researchthemeimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchthemeimport"
	"gopkg.in/yaml.v3"
)

type adminRawDocumentPageDTO struct {
	Items    []adminRawDocument `json:"items"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type adminEventPageDTO struct {
	Items    []adminEvent `json:"items"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

func TestRuntimeBusinessRoutesMatchOpenAPI(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(dataapi.Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	openAPIRoutes := openAPIBusinessRoutes(t, document)
	runtimeRoutes := map[string]struct{}{}
	for _, route := range (Dependencies{}).businessRoutes() {
		runtimeRoutes[strings.ToUpper(route.method)+" "+route.path] = struct{}{}
	}
	assertRouteSetsEqual(t, runtimeRoutes, openAPIRoutes)
}

func TestResponseDTOFieldsMatchOpenAPI(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(dataapi.Document(), &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	for schemaName, dto := range map[string]any{
		"ResearchThemeCollection":            research.ResearchThemePage{},
		"ResearchThemeSummary":               research.ResearchTheme{},
		"ResearchThemeDetail":                research.ResearchThemeDetail{},
		"ResearchThemeChainNode":             research.ResearchThemeChainNode{},
		"ResearchIndex":                      research.ResearchIndex{},
		"ResearchEvent":                      research.ResearchEvent{},
		"ResearchReasoningTreeList":          research.ResearchReasoningTreeList{},
		"ResearchReasoningTreeSummary":       research.ResearchReasoningTreeSummary{},
		"ResearchReasoningTreeDetail":        research.ResearchReasoningTreeDetail{},
		"ResearchReasoningTree":              research.ResearchReasoningTree{},
		"ResearchReasoningTreeChainNode":     research.ResearchReasoningTreeChainNode{},
		"ResearchReasoningTreeEvent":         research.ResearchReasoningTreeEvent{},
		"ResearchReasoningTreePathNode":      research.ResearchReasoningTreePathNode{},
		"AdminRawDocumentPage":               adminRawDocumentPageDTO{},
		"AdminRawDocument":                   adminRawDocument{},
		"AdminEventPage":                     adminEventPageDTO{},
		"AdminEvent":                         adminEvent{},
		"EventPublicationRequest":            publicationdomain.Publication{},
		"EventPublicationProvenance":         publicationdomain.Provenance{},
		"EventPublicationCollectorExecution": publicationdomain.CollectorExecution{},
		"EventPublicationRawDocument":        publicationdomain.RawDocument{},
		"EventPublicationEvent":              publicationdomain.Event{},
		"EventPublicationEvidence":           publicationdomain.Evidence{},
		"EventPublicationTag":                publicationdomain.Tag{},
		"EventPublicationReview":             publicationdomain.Review{},
		"EventPublicationResult":             eventpublicationapp.Result{},
		"EventPublicationEventResult":        eventpublicationapp.EventResult{},
		"EventPublicationRawDocumentResult":  eventpublicationapp.RawDocumentResult{},
		"EventPublicationCounts":             eventpublicationapp.Counts{},
		"ResearchThemeImportRequest":         researchthemedomain.Batch{},
		"ResearchThemeImportItem":            researchthemedomain.Theme{},
		"ResearchThemeImportChainNode":       researchthemedomain.ChainNode{},
		"ResearchThemeImportEvent":           researchthemedomain.Event{},
		"ResearchThemeImportCounts":          researchthemeimportapp.Counts{},
		"ResearchThemeImportResult":          researchthemeimportapp.Result{},
		"ResearchAnchorImportRequest":        researchanchordomain.Publication{},
		"ResearchAnchorImportItem":           researchanchordomain.Anchor{},
		"ResearchAnchorImportEvent":          researchanchordomain.Event{},
		"ResearchAnchorImportPathNode":       researchanchordomain.PathNode{},
		"ResearchAnchorImportCounts":         researchanchorimportapp.Counts{},
		"ResearchAnchorImportResult":         researchanchorimportapp.Result{},
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
