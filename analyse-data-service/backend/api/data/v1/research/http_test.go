package research

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

type httpStub struct {
	analysisContextRequest *ResearchAnalysisContextRequest
	graphRequest           *ResearchGraphSearchRequest
	analysisDeadline       time.Time
	themeReadDeadline      time.Time
}

func TestHTTPRegistersTheCompleteExistingResearchContract(t *testing.T) {
	application := &httpStub{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, application)
	routes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, v1.APIPrefix + "/research-theme-imports", `{}`},
		{http.MethodGet, v1.APIPrefix + "/research/themes", ""},
		{http.MethodGet, v1.APIPrefix + "/research/themes/theme-id", ""},
		{http.MethodGet, v1.APIPrefix + "/research/themes/theme-id/reasoning-trees", ""},
		{http.MethodGet, v1.APIPrefix + "/research/themes/theme-id/reasoning-trees/tree-id", ""},
		{http.MethodGet, v1.APIPrefix + "/research-analysis-context", ""},
		{http.MethodPost, v1.APIPrefix + "/research-graph:search", `{}`},
	}
	for _, route := range routes {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(route.method, route.path, strings.NewReader(route.body))
		server.ServeHTTP(recorder, request)
		if recorder.Code == http.StatusNotFound {
			t.Errorf("%s %s is not registered", route.method, route.path)
		}
	}
}

func (*httpStub) PublishResearchTheme(context.Context, *ResearchThemeImportRequest) (*v1.Response[ResearchThemeImportResult], error) {
	return nil, nil
}

func (s *httpStub) ListResearchThemes(ctx context.Context, _ *ListResearchThemesRequest) (*v1.Response[ResearchThemePage], error) {
	s.themeReadDeadline, _ = ctx.Deadline()
	return &v1.Response[ResearchThemePage]{Status: v1.StatusOK}, nil
}

func (*httpStub) GetResearchTheme(context.Context, *GetResearchThemeRequest) (*v1.Response[ResearchThemeDetail], error) {
	return nil, nil
}

func (*httpStub) ListResearchReasoningTrees(context.Context, *ReasoningTreeListRequest) (*v1.Response[ResearchReasoningTreeList], error) {
	return nil, nil
}

func (*httpStub) GetResearchReasoningTree(context.Context, *ReasoningTreeDetailRequest) (*v1.Response[ResearchReasoningTreeDetail], error) {
	return nil, nil
}

func (s *httpStub) ListResearchAnalysisContext(ctx context.Context, request *ResearchAnalysisContextRequest) (*v1.Response[ResearchAnalysisContext], error) {
	s.analysisContextRequest = request
	s.analysisDeadline, _ = ctx.Deadline()
	return &v1.Response[ResearchAnalysisContext]{Status: v1.StatusOK, Result: ResearchAnalysisContext{
		ContractVersion: "research-analysis-context.v1",
		AnalysisAsOf:    request.AnalysisAsOf,
	}}, nil
}

func (s *httpStub) SearchResearchGraph(_ context.Context, request *ResearchGraphSearchRequest) (*v1.Response[ResearchGraphSearchResult], error) {
	s.graphRequest = request
	return &v1.Response[ResearchGraphSearchResult]{Status: v1.StatusOK, Result: ResearchGraphSearchResult{
		ContractVersion: "research-graph-search.v1", AnalysisAsOf: request.AnalysisAsOf,
		Entities: []ResearchAnalysisEntity{}, RelationDefinitions: []ResearchAnalysisRelationDefinition{},
		EntityRelations: []ResearchAnalysisEntityRelation{}, IndustryChains: []ResearchAnalysisIndustryChain{},
		IndustryChainMemberships: []ResearchAnalysisIndustryChainMembership{},
		IndustryChainGraphEdges:  []ResearchAnalysisIndustryChainGraphEdge{},
	}}, nil
}

func TestAnalysisContextHTTPBindsTheExistingResearchContract(t *testing.T) {
	application := &httpStub{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, application)
	request := httptest.NewRequest(
		http.MethodGet,
		v1.APIPrefix+"/research-analysis-context"+
			"?discovery_window_start=2026-07-28T00%3A00%3A00Z"+
			"&discovery_window_end=2026-07-29T00%3A00%3A00Z"+
			"&analysis_as_of=2026-07-29T00%3A00%3A00Z&page_size=20",
		nil,
	)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || application.analysisContextRequest == nil ||
		application.analysisContextRequest.PageSize != 20 ||
		application.analysisContextRequest.DiscoveryWindowStart != "2026-07-28T00:00:00Z" ||
		application.analysisDeadline.IsZero() || time.Until(application.analysisDeadline) > HeavyExecutionBudget {
		t.Fatalf("status=%d request=%#v body=%s", recorder.Code, application.analysisContextRequest, recorder.Body)
	}
}

func TestResearchThemeReadHTTPAppliesFiveSecondBudget(t *testing.T) {
	application := &httpStub{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, application)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/research/themes?window_hours=24&limit=20", nil))
	remaining := time.Until(application.themeReadDeadline)
	if recorder.Code != http.StatusOK || application.themeReadDeadline.IsZero() || remaining <= 0 || remaining > ReadExecutionBudget {
		t.Fatalf("status=%d read deadline remaining=%s", recorder.Code, remaining)
	}
}

func TestAnalysisContextHTTPRejectsMissingUnknownAndRepeatedQueryParameters(t *testing.T) {
	valid := v1.APIPrefix + "/research-analysis-context?discovery_window_start=2026-07-28T00%3A00%3A00Z&discovery_window_end=2026-07-29T00%3A00%3A00Z&analysis_as_of=2026-07-29T00%3A00%3A00Z&page_size=20"
	for _, target := range []string{
		v1.APIPrefix + "/research-analysis-context?analysis_as_of=2026-07-29T00%3A00%3A00Z&page_size=20",
		valid + "&invented=true",
		valid + "&page_size=30",
	} {
		application := &httpStub{}
		server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(publicErrorStatusEncoder(t)))
		RegisterHTTPServer(server, application)
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != http.StatusBadRequest || application.analysisContextRequest != nil {
			t.Fatalf("target=%s status=%d request=%#v", target, recorder.Code, application.analysisContextRequest)
		}
	}
}

func TestResearchGraphHTTPBindsStrictRequestAndRejectsUnknownOrOversizedBodies(t *testing.T) {
	valid := `{"analysis_as_of":"2026-07-30T00:00:00Z","seed_entity_ids":["11111111-1111-4111-8111-111111111111"],"relation_filters":[{"relation_type":"produces","direction":"outgoing"}],"max_depth":2,"node_budget":100,"edge_budget":200}`
	t.Run("valid", func(t *testing.T) {
		application := &httpStub{}
		server := kratoshttp.NewServer()
		RegisterHTTPServer(server, application)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/research-graph:search", strings.NewReader(valid))
		request.Header.Set("Content-Type", "application/json")
		server.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK || application.graphRequest == nil || application.graphRequest.MaxDepth != 2 {
			t.Fatalf("status=%d request=%#v body=%s", recorder.Code, application.graphRequest, recorder.Body.String())
		}
	})
	for _, test := range []struct {
		name, payload string
		want          int
	}{
		{name: "unknown", payload: `{"analysis_as_of":"2026-07-30T00:00:00Z","invented":true}`, want: http.StatusBadRequest},
		{name: "oversized", payload: string(bytes.Repeat([]byte("x"), v1.MaxRequestBodySize+1)), want: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			application := &httpStub{}
			server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(publicErrorStatusEncoder(t)))
			RegisterHTTPServer(server, application)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/research-graph:search", strings.NewReader(test.payload))
			request.Header.Set("Content-Type", "application/json")
			server.ServeHTTP(recorder, request)
			if recorder.Code != test.want || application.graphRequest != nil {
				t.Fatalf("status=%d request=%#v", recorder.Code, application.graphRequest)
			}
		})
	}
}

func publicErrorStatusEncoder(t *testing.T) func(http.ResponseWriter, *http.Request, error) {
	t.Helper()
	return func(response http.ResponseWriter, _ *http.Request, err error) {
		public, ok := err.(*v1.PublicError)
		if !ok {
			t.Fatalf("error = %T %v", err, err)
		}
		response.WriteHeader(public.Status)
	}
}

func TestResearchThemeBindingRejectsDuplicateAndUnknownFieldsWithPath(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
		path    string
	}{
		{
			name:    "duplicate nested field",
			payload: `{"analysis_batch_id":"batch","analysis_as_of":"as-of","discovery_window_start":"start","discovery_window_end":"end","theme":{"theme_key":"theme-1","theme_key":"theme-2"},"reasoning_trees":[]}`,
			path:    "theme.theme_key",
		},
		{
			name:    "unknown nested field",
			payload: `{"analysis_batch_id":"batch","analysis_as_of":"as-of","discovery_window_start":"start","discovery_window_end":"end","theme":{"theme_key":"theme-1","unexpected":true},"reasoning_trees":[]}`,
			path:    "theme.unexpected",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeResearchThemeImport([]byte(test.payload))
			publicError, ok := err.(*v1.PublicError)
			if !ok {
				t.Fatalf("error = %T %v, want *v1.PublicError", err, err)
			}
			details, ok := publicError.Details.(map[string]any)
			if !ok || details["path"] != test.path {
				t.Fatalf("details = %#v, want path %q", publicError.Details, test.path)
			}
		})
	}
}

func TestResearchThemeBindingAcceptsPreparedUATAnalystSnapshotFixture(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "..", "testdata", "research-theme-analyst-snapshot-v3", "01-uat-at01-prepared-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	request, err := decodeResearchThemeImport(payload)
	if err != nil || request.Snapshot == nil || len(request.Snapshot.ReasoningTrees) != 2 {
		t.Fatalf("decode prepared UAT fixture = %#v, %v", request, err)
	}
}

func TestResearchThemeBindingAcceptsIsolatedAnalystSnapshotAndRejectsFormalIDs(t *testing.T) {
	request := ResearchThemeSnapshotImportRequest{
		PublicationMode: "analyst_snapshot", AnalysisBatchID: "batch-snapshot",
		AnalysisAsOf: "2026-08-03T11:00:00Z", DiscoveryWindowStart: "2026-08-03T03:00:00Z",
		DiscoveryWindowEnd: "2026-08-03T07:00:00Z",
		Theme: ResearchThemeSnapshotItem{
			ThemeKey: "theme:snapshot", Title: "Theme", OneLineConclusion: "Conclusion",
			ConclusionDirection: "uncertain", ImpactStrength: "unknown", TransmissionStage: "validation",
			InvestmentGuidanceAction: "observe", InvestmentGuidanceSummary: "Observe",
			TimeHorizonCategory: "medium_term",
			Impacts:             []ResearchThemeSnapshotImpact{{NodeKey: "node:a", DisplayName: "Focus name", RelationRole: "driver", ImpactDirection: "uncertain", DisplayOrder: 1}},
			Events:              []ResearchThemeSnapshotEvent{{EventID: "11111111-1111-4111-8111-111111111111", EvidenceRole: "driver"}},
		},
		ReasoningTrees: []ResearchReasoningTreeSnapshotImportItem{{
			TreeKey: "tree:a", DisplayName: "Analysis path", Title: "Tree", DisplayOrder: 1,
			OneLineConclusion: "Tree conclusion", ImpactDirection: "uncertain", ImpactStrength: "unknown",
			InvalidationConditions: []string{}, Checkpoints: []ResearchReasoningTreeImportCheckpoint{},
			Events: []ResearchReasoningTreeSnapshotEvent{{EventID: "11111111-1111-4111-8111-111111111111", EvidenceRole: "driver", DisplayOrder: 1}},
			Nodes: []ResearchReasoningTreeSnapshotNode{{
				NodeKey: "node:a", DisplayName: "Detailed node name", Position: 1,
				ImpactDirection: "uncertain", ImpactStrength: "unknown",
				Signals: []ResearchReasoningTreeSnapshotSignal{{SignalKey: "signal:a", DisplaySummary: "完成流片", Role: "primary", DisplayOrder: 1}},
			}},
		}},
	}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeResearchThemeImport(payload)
	if err != nil || decoded.Snapshot == nil {
		t.Fatalf("decode analyst_snapshot = %#v, %v", decoded, err)
	}

	formalID := `,"chain_node_entity_id":"33333333-3333-4333-8333-333333333333"`
	mixed := strings.Replace(string(payload), `"node_key":"node:a"`, `"node_key":"node:a"`+formalID, 1)
	if _, err := decodeResearchThemeImport([]byte(mixed)); err == nil {
		t.Fatal("analyst_snapshot carrying formal ontology ID was accepted")
	}
}

func TestResearchThemeBindingRequiresAtomicTreeAndLineageShape(t *testing.T) {
	payload := `{"analysis_batch_id":"batch","analysis_as_of":"2026-07-29T00:00:00Z","discovery_window_start":"2026-07-28T00:00:00Z","discovery_window_end":"2026-07-29T00:00:00Z","theme":{},"reasoning_trees":[{"industry_chain_entity_id":"22222222-2222-4222-8222-222222222222"}]}`
	_, err := decodeResearchThemeImport([]byte(payload))
	publicError, ok := err.(*v1.PublicError)
	if !ok {
		t.Fatalf("error = %T %v, want *v1.PublicError", err, err)
	}
	details, ok := publicError.Details.(map[string]any)
	if !ok || details["path"] != "reasoning_trees[0].title" {
		t.Fatalf("details = %#v", publicError.Details)
	}
}

func TestResearchThemeBindingAcceptsAtomicAggregateShape(t *testing.T) {
	payload := `{
		"analysis_batch_id":"batch","analysis_as_of":"2026-07-02T00:00:00Z",
		"discovery_window_start":"2026-07-01T00:00:00Z","discovery_window_end":"2026-07-02T00:00:00Z",
		"theme":{},"reasoning_trees":[{
			"industry_chain_entity_id":"22222222-2222-4222-8222-222222222222",
			"title":"tree","display_order":1,"one_line_conclusion":"conclusion",
			"fact_summary":null,"transmission_summary":null,"impact_direction":"positive",
			"impact_strength":"medium","impact_summary":null,"conclusion_boundary_summary":null,
			"support_summary":null,"counter_summary":null,"invalidation_conditions":[],
			"checkpoints":[],"events":[],"nodes":[{
				"position":1,"chain_node_entity_id":"33333333-3333-4333-8333-333333333333",
				"state_summary":null,"impact_direction":"positive","impact_strength":"medium",
				"impact_summary":null,"reasoning_basis_summary":null,"evidence_gap_summary":null,
				"incoming_industry_chain_graph_edge_id":null,"incoming_transmission_title":null,
				"incoming_transmission_mechanism":null,"incoming_condition_summary":null,
				"incoming_lineage":null,"signals":[]
			}]
		}]
	}`
	if _, err := decodeResearchThemeImport([]byte(payload)); err != nil {
		t.Fatalf("decode atomic aggregate: %#v", err)
	}
}
