package research

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const researchAnalysisFixtureDirectory = "../../../../../../testdata/research-analysis-context-v1"

type researchAnalysisContextFixtureEnvelope struct {
	RequestID string                  `json:"request_id"`
	Result    ResearchAnalysisContext `json:"result"`
}

type researchGraphFixtureEnvelope struct {
	RequestID string                    `json:"request_id"`
	Result    ResearchGraphSearchResult `json:"result"`
}

type researchResourceLimitFixtureEnvelope struct {
	Error struct {
		Code    string                       `json:"code"`
		Message string                       `json:"message"`
		Details ResearchResourceLimitDetails `json:"details"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

type researchAnalysisContextInconsistentFixtureEnvelope struct {
	Error struct {
		Code    string                                     `json:"code"`
		Message string                                     `json:"message"`
		Details ResearchAnalysisContextInconsistentDetails `json:"details"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

type researchAnalysisFixtureConsumerState struct {
	eventIDs       map[string]struct{}
	mergedEntities map[string]ResearchAnalysisEntity
	nextCursor     string
	pagesRead      int
}

func newResearchAnalysisFixtureConsumerState() *researchAnalysisFixtureConsumerState {
	return &researchAnalysisFixtureConsumerState{
		eventIDs:       map[string]struct{}{},
		mergedEntities: map[string]ResearchAnalysisEntity{},
	}
}

func (s *researchAnalysisFixtureConsumerState) mergePage(page ResearchAnalysisContext) {
	for _, bundle := range page.EventSemanticBundles {
		s.eventIDs[bundle.Event.ID] = struct{}{}
	}
	for _, entity := range page.Dictionaries.Entities {
		s.mergedEntities[entity.EntityID] = entity
	}
	s.nextCursor = page.NextCursor
	s.pagesRead++
}

func (s *researchAnalysisFixtureConsumerState) mergeGraph(result ResearchGraphSearchResult) {
	for _, entity := range result.Entities {
		s.mergedEntities[entity.EntityID] = entity
	}
}

func (s *researchAnalysisFixtureConsumerState) restartFromFirstPage() {
	clear(s.eventIDs)
	clear(s.mergedEntities)
	s.nextCursor = ""
	s.pagesRead = 0
}

func TestResearchAnalysisContextV1SharedFixturesDecodeAndCloseReferences(t *testing.T) {
	var contextFixture researchAnalysisContextFixtureEnvelope
	decodeResearchFixture(
		t,
		"analysis-context-page.json",
		&contextFixture,
	)
	result := contextFixture.Result
	if result.ContractVersion != "research-analysis-context.v1" ||
		result.TBoxContractVersion == "" ||
		!fixtureFingerprint(result.EventPageFingerprint) ||
		!fixtureFingerprint(result.ReferenceClosureFingerprint) {
		t.Fatalf("Analysis Context fixture metadata = %#v", result)
	}
	if !result.HasMore || result.NextCursor == "" {
		t.Fatalf("first Analysis Context fixture must continue: %#v", result)
	}
	entityIDs := map[string]struct{}{}
	for _, entity := range result.Dictionaries.Entities {
		entityIDs[entity.EntityID] = struct{}{}
	}
	relationIDs := map[string]struct{}{}
	for _, relation := range result.Dictionaries.EntityRelations {
		relationIDs[relation.EntityRelationID] = struct{}{}
		assertFixtureEntityExists(t, entityIDs, relation.FromEntityID)
		assertFixtureEntityExists(t, entityIDs, relation.ToEntityID)
	}
	for _, bundle := range result.EventSemanticBundles {
		for _, link := range bundle.EntityLinks {
			assertFixtureEntityExists(t, entityIDs, link.EntityID)
		}
		for _, signal := range bundle.VariableSignals {
			assertFixtureEntityExists(t, entityIDs, signal.SubjectEntityID)
			for _, impact := range signal.DirectImpacts {
				assertFixtureEntityExists(t, entityIDs, impact.TargetEntityID)
				if impact.EntityRelationID == nil {
					t.Fatal("fixture Direct Impact must resolve a formal EntityRelation")
				}
				if _, exists := relationIDs[*impact.EntityRelationID]; !exists {
					t.Fatalf("fixture Direct Impact has unresolved relation %q", *impact.EntityRelationID)
				}
			}
		}
	}

	var graphRequest ResearchGraphSearchRequest
	decodeResearchFixture(t, "research-graph-search-request.json", &graphRequest)
	if graphRequest.MaxDepth != 1 ||
		len(graphRequest.SeedEntityIDs) != 1 ||
		len(graphRequest.RelationFilters) != 1 {
		t.Fatalf("Research Graph request fixture = %#v", graphRequest)
	}

	var graphFixture researchGraphFixtureEnvelope
	decodeResearchFixture(t, "research-graph-search-result.json", &graphFixture)
	if graphFixture.Result.ContractVersion != "research-graph-search.v1" ||
		!fixtureFingerprint(graphFixture.Result.QueryFingerprint) ||
		!fixtureFingerprint(graphFixture.Result.GraphFingerprint) {
		t.Fatalf("Research Graph result fixture = %#v", graphFixture.Result)
	}

	var secondPage researchAnalysisContextFixtureEnvelope
	decodeResearchFixture(t, "analysis-context-page-2.json", &secondPage)
	if secondPage.Result.HasMore ||
		secondPage.Result.NextCursor != "" ||
		len(secondPage.Result.EventSemanticBundles) != 1 {
		t.Fatalf("second Analysis Context fixture = %#v", secondPage.Result)
	}
	consumer := newResearchAnalysisFixtureConsumerState()
	for _, page := range []ResearchAnalysisContext{result, secondPage.Result} {
		consumer.mergePage(page)
	}
	consumer.mergeGraph(graphFixture.Result)
	if len(consumer.eventIDs) != 2 ||
		len(consumer.mergedEntities) != 4 ||
		consumer.pagesRead != 2 {
		t.Fatalf(
			"merged fixture events=%v entities=%v pages=%d",
			consumer.eventIDs,
			consumer.mergedEntities,
			consumer.pagesRead,
		)
	}

	var resourceLimit researchResourceLimitFixtureEnvelope
	decodeResearchFixture(t, "resource-limit-error.json", &resourceLimit)
	if resourceLimit.Error.Code != "RESEARCH_GRAPH_RESOURCE_LIMIT" ||
		resourceLimit.Error.Details.Component != "research_graph_nodes" ||
		resourceLimit.Error.Details.ActualRows != nil ||
		resourceLimit.Error.Details.MaxRows == nil ||
		resourceLimit.Error.Details.RetryGuidance == "" {
		t.Fatalf("resource limit fixture = %#v", resourceLimit)
	}

	var inconsistent researchAnalysisContextInconsistentFixtureEnvelope
	decodeResearchFixture(t, "inconsistent-error.json", &inconsistent)
	if inconsistent.Error.Code != "RESEARCH_ANALYSIS_CONTEXT_INCONSISTENT" ||
		inconsistent.Error.Details.RetryGuidance != "restart_from_first_page" {
		t.Fatalf("inconsistent fixture = %#v", inconsistent)
	}
	interruptedConsumer := newResearchAnalysisFixtureConsumerState()
	interruptedConsumer.mergePage(result)
	interruptedConsumer.mergeGraph(graphFixture.Result)
	if interruptedConsumer.nextCursor == "" ||
		len(interruptedConsumer.eventIDs) == 0 ||
		len(interruptedConsumer.mergedEntities) == 0 {
		t.Fatalf("fixture must accumulate continuation state before 409: %#v", interruptedConsumer)
	}
	interruptedConsumer.restartFromFirstPage()
	if len(interruptedConsumer.eventIDs) != 0 ||
		len(interruptedConsumer.mergedEntities) != 0 ||
		interruptedConsumer.nextCursor != "" ||
		interruptedConsumer.pagesRead != 0 {
		t.Fatalf("restart-from-first-page left consumer state = %#v", interruptedConsumer)
	}
}

func decodeResearchFixture(t *testing.T, name string, target any) {
	t.Helper()
	file, err := os.Open(filepath.Join(researchAnalysisFixtureDirectory, name))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
}

func assertFixtureEntityExists(
	t *testing.T,
	entityIDs map[string]struct{},
	entityID string,
) {
	t.Helper()
	if _, exists := entityIDs[entityID]; !exists {
		t.Fatalf("fixture contains unresolved entity ID %q", entityID)
	}
}

func fixtureFingerprint(value string) bool {
	return regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(value)
}
