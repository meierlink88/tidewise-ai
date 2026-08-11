package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
	researchanalysiscontextapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
	eventsemanticdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/eventsemantic"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
	eventsemanticfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/eventsemantic"
)

func TestPostgresResearchAnalysisContextSelectsEventsBeforeReferenceClosure(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO events (
		    id, title, summary, first_seen_at, knowable_at,
		    event_status, fact_status, dedupe_key
		) VALUES (
		    '10000000-0000-4000-8000-000000000001',
		    'Reference closure capacity fixture',
		    'One eligible Event must remain readable when unrelated master data exceeds the old budget.',
		    '2026-07-28T08:00:00Z',
		    '2026-07-28T08:00:00Z',
		    'confirmed',
		    'verified',
		    'research-analysis-context:capacity:1'
		), (
		    '10000000-0000-4000-8000-000000000002',
		    'Reference closure pagination fixture',
		    'Every eligible Event must remain available through cursor pagination.',
		    '2026-07-28T09:00:00Z',
		    '2026-07-28T09:00:00Z',
		    'confirmed',
		    'verified',
		    'research-analysis-context:capacity:2'
		);

		INSERT INTO entity_nodes (
		    id, entity_key, entity_type, layer_code, name, canonical_name,
		    aliases, status, created_at, updated_at
		)
		SELECT
		    md5('analysis-context-unrelated-' || series)::uuid,
		    'company:analysis-context-unrelated-' || series,
		    'company',
		    'company',
		    'Unrelated Company ' || series,
		    'unrelated company ' || series,
		    '{}',
		    'active',
		    '2026-07-28T07:00:00Z',
		    '2026-07-28T07:00:00Z'
		FROM generate_series(1, 50001) series
	`); err != nil {
		t.Fatal(err)
	}

	handler := dataServiceTestHandler(Dependencies{
		ResearchAnalysisContext: researchanalysiscontextapp.NewService(
			postgres.NewResearchAnalysisContextStore(db),
		),
	}, map[string]v1.Principal{
		"read-token": {
			Identity: "codex-analyst",
			Scopes:   []string{ScopeResearchRead},
		},
	}, "request-analysis-context-capacity")
	request := httptest.NewRequest(
		http.MethodGet,
		Namespace+"/research-analysis-context"+
			"?discovery_window_start=2026-07-28T00%3A00%3A00Z"+
			"&discovery_window_end=2026-07-29T00%3A00%3A00Z"+
			"&analysis_as_of=2026-07-29T00%3A00%3A00Z"+
			"&page_size=1",
		nil,
	)
	request.Header.Set("Authorization", "Bearer read-token")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body)
	}
	var response struct {
		Result v1.ResearchAnalysisContext `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Result.EventSemanticBundles) != 1 ||
		!response.Result.HasMore ||
		response.Result.NextCursor == "" ||
		response.Result.EventSemanticBundles[0].Event.ID != "10000000-0000-4000-8000-000000000001" {
		t.Fatalf("result = %#v", response.Result)
	}
	if len(response.Result.Dictionaries.Entities) != 0 {
		t.Fatalf(
			"reference closure contains %d unrelated entities",
			len(response.Result.Dictionaries.Entities),
		)
	}

	nextRequest := httptest.NewRequest(
		http.MethodGet,
		Namespace+"/research-analysis-context"+
			"?discovery_window_start=2026-07-28T00%3A00%3A00Z"+
			"&discovery_window_end=2026-07-29T00%3A00%3A00Z"+
			"&analysis_as_of=2026-07-29T00%3A00%3A00Z"+
			"&page_size=1"+
			"&cursor="+url.QueryEscape(response.Result.NextCursor),
		nil,
	)
	nextRequest.Header.Set("Authorization", "Bearer read-token")
	nextRecorder := httptest.NewRecorder()
	handler.ServeHTTP(nextRecorder, nextRequest)
	if nextRecorder.Code != http.StatusOK {
		t.Fatalf("next status=%d body=%s", nextRecorder.Code, nextRecorder.Body)
	}
	var nextResponse struct {
		Result v1.ResearchAnalysisContext `json:"result"`
	}
	if err := json.Unmarshal(nextRecorder.Body.Bytes(), &nextResponse); err != nil {
		t.Fatal(err)
	}
	if len(nextResponse.Result.EventSemanticBundles) != 1 ||
		nextResponse.Result.HasMore ||
		nextResponse.Result.NextCursor != "" ||
		nextResponse.Result.EventSemanticBundles[0].Event.ID != "10000000-0000-4000-8000-000000000002" {
		t.Fatalf("next result = %#v", nextResponse.Result)
	}
}

func TestPostgresResearchAnalysisContextReturnsCompleteReferencedSemantics(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	semanticStore, err := eventsemanticdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	semanticService, err := eventsemantic.NewUseCase(semanticStore)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	lease, err := semanticService.CreateContextLease(ctx, eventsemantic.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: "analysis-context-semantic-execution",
		WorkerID: "analysis-context-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	submissionRequest := eventsemanticfixture.Submission(
		lease.ID,
		"analysis-context-semantic-execution",
		"",
	)
	submissionRequest.EntityLinks[0].EntityRole = "actor"
	submission, err := semanticService.CreateSubmission(ctx, submissionRequest)
	if err != nil {
		t.Fatal(err)
	}
	if submission.Status != eventsemantic.StatusPendingReview {
		t.Fatalf("submission = %#v", submission)
	}
	review, err := semanticService.SubmitReview(ctx, eventsemantic.ReviewSubmission{
		SubmissionID:         submission.SubmissionID,
		ReviewerExecutionKey: "analysis-context-semantic-execution:reviewer",
		PromptHash:           strings.Repeat("b", 64),
		Model:                "fixture-reviewer",
		Items:                eventsemanticfixture.ReviewItems("pass"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if review.Status != eventsemantic.StatusAccepted {
		t.Fatalf("review status = %q", review.Status)
	}

	handler := dataServiceTestHandler(Dependencies{
		ResearchAnalysisContext: researchanalysiscontextapp.NewService(
			postgres.NewResearchAnalysisContextStore(db),
		),
	}, map[string]v1.Principal{
		"read-token": {
			Identity: "codex-analyst",
			Scopes:   []string{ScopeResearchRead},
		},
	}, "request-analysis-context-lineage")
	request := httptest.NewRequest(
		http.MethodGet,
		Namespace+"/research-analysis-context"+
			"?discovery_window_start=2026-07-28T00%3A00%3A00Z"+
			"&discovery_window_end=2026-07-29T00%3A00%3A00Z"+
			"&analysis_as_of=2030-07-29T00%3A00%3A00Z"+
			"&page_size=1",
		nil,
	)
	request.Header.Set("Authorization", "Bearer read-token")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body)
	}
	var response struct {
		Result v1.ResearchAnalysisContext `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	result := response.Result
	if len(result.EventSemanticBundles) != 1 ||
		len(result.EventSemanticBundles[0].Evidence) != 1 ||
		len(result.EventSemanticBundles[0].EntityLinks) != 1 ||
		len(result.EventSemanticBundles[0].VariableSignals) != 1 ||
		len(result.EventSemanticBundles[0].VariableSignals[0].Measurements) != 1 ||
		len(result.EventSemanticBundles[0].VariableSignals[0].DirectImpacts) != 0 {
		t.Fatalf("incomplete Event semantics = %#v", result.EventSemanticBundles)
	}
	if got := len(result.Dictionaries.Entities); got != 1 {
		t.Fatalf("entities = %d, want the Event-linked Entity", got)
	}
	if got := len(result.Dictionaries.EntityRelations); got != 0 {
		t.Fatalf("entity relations = %d, want no Event Semantic transmission relation", got)
	}
	if got := len(result.Dictionaries.VariableDefinitions); got != 1 {
		t.Fatalf("variable definitions = %d, want the Signal definition", got)
	}
	if got := len(result.Dictionaries.DirectTransmissionRules); got != 0 {
		t.Fatalf("rules = %d, want no Event Semantic transmission rule", got)
	}
	if got := len(result.Dictionaries.AcceptancePolicies); got != 1 {
		t.Fatalf("acceptance policies = %d, want submission policy", got)
	}
}
