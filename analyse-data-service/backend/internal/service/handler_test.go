package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func TestResearchThemeImportMapsV1Contract(t *testing.T) {
	importer := &fakeThemeImporter{result: researchthemeimport.Result{
		ReceiptID: "receipt", AnalysisBatchID: "batch", ThemeIDsByKey: map[string]string{"ai": "theme"},
		Counts: researchthemeimport.Counts{Themes: 1, Impacts: 2, EventAssociations: 0, Receipts: 1},
	}}
	handler := NewDataService(Dependencies{ResearchThemeImports: importer})
	attention := "high"
	response, err := handler.ImportResearchThemes(v1.WithPrincipal(context.Background(), v1.Principal{Identity: "analyst"}), &v1.ResearchThemeImportRequest{
		AnalysisBatchID: "batch", AnalysisAsOf: "2026-07-28T00:00:00Z",
		WindowStart: "2026-07-27T00:00:00Z", WindowEnd: "2026-07-28T00:00:00Z",
		Themes: []v1.ResearchThemeImportItem{{
			ThemeKey: "ai", Title: "AI infrastructure", OneLineConclusion: "Demand rises",
			ConclusionDirection: "positive", ImpactStrength: "medium", AttentionLevel: &attention,
			TransmissionStage: "validation", InvestmentGuidanceAction: "focus",
			InvestmentGuidanceSummary: "Watch orders", TimeHorizonCategory: "short_term",
			Impacts: []v1.ResearchThemeImportImpact{{
				ChainNodeEntityID: "11111111-1111-4111-8111-111111111111",
				RelationRole:      "beneficiary", ImpactDirection: "positive", DisplayOrder: 1,
			}},
		}},
	})
	if err != nil {
		t.Fatalf("ImportResearchThemes() error = %v", err)
	}
	if importer.publisher != "analyst" || importer.batch.AnalysisAsOf != "2026-07-28T00:00:00Z" {
		t.Fatalf("import call = publisher %q batch %#v", importer.publisher, importer.batch)
	}
	if response.Result.Counts.Impacts != 2 || response.Result.ThemeIDsByKey["ai"] != "theme" {
		t.Fatalf("response = %#v", response.Result)
	}
}

func TestResearchReasoningTreeImportMapsV1Contract(t *testing.T) {
	importer := &fakeTreeImporter{result: researchreasoningtreeimport.Result{
		ReceiptID: "receipt", ThemeID: "11111111-1111-4111-8111-111111111111",
		ReasoningTreeIDsByIndustryChainEntityID: map[string]string{"chain": "tree"},
		Counts:                                  researchreasoningtreeimport.Counts{ReasoningTrees: 1, Nodes: 2, SignalAssociations: 2, Receipts: 1},
	}}
	handler := NewDataService(Dependencies{ResearchReasoningTreeImports: importer})
	response, err := handler.ImportResearchReasoningTrees(context.Background(), &v1.ResearchReasoningTreeImportRequest{
		ThemeID: "11111111-1111-4111-8111-111111111111",
		ReasoningTrees: []v1.ResearchReasoningTreeImportItem{{
			IndustryChainEntityID: "22222222-2222-4222-8222-222222222222",
			Title:                 "Optical module", DisplayOrder: 1, OneLineConclusion: "Demand rises",
			ImpactDirection: "positive", ImpactStrength: "medium",
			Nodes: []v1.ResearchReasoningTreeImportNode{{
				Position: 1, ChainNodeEntityID: "33333333-3333-4333-8333-333333333333",
				ImpactDirection: "positive", ImpactStrength: "medium",
				Signals: []v1.ResearchReasoningTreeImportSignal{{
					VariableSignalKey: "port-plan", SignalRole: "primary",
					SignalDirection: "up", DisplaySummary: "Port plan +80%", DisplayOrder: 1,
				}},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("ImportResearchReasoningTrees() error = %v", err)
	}
	if importer.publication.ReasoningTrees[0].Nodes[0].Signals[0].VariableSignalKey != "port-plan" {
		t.Fatalf("publication = %#v", importer.publication)
	}
	if response.Result.Counts.Nodes != 2 {
		t.Fatalf("response = %#v", response.Result)
	}
}

func TestReasoningTreeDetailExposesImpactIntersectionAndSignals(t *testing.T) {
	treeID := "22222222-2222-4222-8222-222222222222"
	researchService := &fakeResearchService{detail: research.ResearchReasoningTreeDetail{
		ThemeID:       "11111111-1111-4111-8111-111111111111",
		ImpactNodeIDs: []string{"33333333-3333-4333-8333-333333333333"},
		ReasoningTree: research.ResearchReasoningTree{
			ReasoningTreeID: treeID, ThemeID: "11111111-1111-4111-8111-111111111111",
			IndustryChainEntityID: "44444444-4444-4444-8444-444444444444", Title: "Tree",
			Nodes: []research.ResearchReasoningTreeNode{{
				ID: "node", PrimarySignal: research.ResearchSignal{
					VariableSignalKey: "demand", SignalRole: "primary", DisplayOrder: 1,
				},
			}},
		},
	}}
	handler := NewDataService(Dependencies{Research: researchService})
	response, err := handler.GetResearchReasoningTree(context.Background(), &v1.ReasoningTreeDetailRequest{
		ThemeID: "11111111-1111-4111-8111-111111111111", ReasoningTreeID: treeID,
	})
	if err != nil {
		t.Fatalf("GetResearchReasoningTree() error = %v", err)
	}
	if researchService.reasoningTreeID != treeID || response.Result.ReasoningTree.Nodes[0].PrimarySignal.VariableSignalKey != "demand" {
		t.Fatalf("detail = %#v", response.Result)
	}
}

func TestReasoningTreeImportReferenceErrorsRemainActionable(t *testing.T) {
	importer := &fakeTreeImporter{err: &researchreasoningtreeimport.ReferenceError{
		Kind: researchreasoningtreeimport.ReferenceInvalid, IndustryChainEntityID: "chain",
		Path: "reasoning_trees[0].nodes[1]", Reference: "node",
	}}
	_, err := NewDataService(Dependencies{ResearchReasoningTreeImports: importer}).
		ImportResearchReasoningTrees(context.Background(), &v1.ResearchReasoningTreeImportRequest{})
	var public *v1.PublicError
	if !errors.As(err, &public) || public.Status != v1.StatusUnprocessableEntity || public.Code != "RESEARCH_REASONING_TREE_REFERENCE_INVALID" {
		t.Fatalf("error = %#v", err)
	}
}

type fakeThemeImporter struct {
	result    researchthemeimport.Result
	batch     researchthemeimport.Batch
	publisher string
}

func (f *fakeThemeImporter) Import(_ context.Context, publisher string, batch researchthemeimport.Batch) (researchthemeimport.Result, error) {
	f.publisher, f.batch = publisher, batch
	return f.result, nil
}

type fakeTreeImporter struct {
	result      researchreasoningtreeimport.Result
	publication researchreasoningtreeimport.Publication
	err         error
}

func (f *fakeTreeImporter) Import(_ context.Context, _ string, publication researchreasoningtreeimport.Publication) (researchreasoningtreeimport.Result, error) {
	f.publication = publication
	return f.result, f.err
}

type fakeResearchService struct {
	detail          research.ResearchReasoningTreeDetail
	reasoningTreeID string
}

func (f *fakeResearchService) ListThemes(context.Context, research.ResearchListRequest) (research.ResearchThemePage, error) {
	return research.ResearchThemePage{}, nil
}

func (f *fakeResearchService) GetTheme(context.Context, string, research.ResearchDetailRequest) (research.ResearchThemeDetail, error) {
	return research.ResearchThemeDetail{}, nil
}

func (f *fakeResearchService) ListReasoningTrees(context.Context, string) (research.ResearchReasoningTreeList, error) {
	return research.ResearchReasoningTreeList{}, nil
}

func (f *fakeResearchService) GetReasoningTree(_ context.Context, _ string, treeID string) (research.ResearchReasoningTreeDetail, error) {
	f.reasoningTreeID = treeID
	return f.detail, nil
}

func dataServiceTestHandler(dependencies Dependencies, credentials map[string]v1.Principal, generatedRequestID string) http.Handler {
	server := kratoshttp.NewServer(
		kratoshttp.ResponseEncoder(func(response http.ResponseWriter, request *http.Request, result any) error {
			response.Header().Set("Content-Type", "application/json")
			return json.NewEncoder(response).Encode(map[string]any{"request_id": request.Header.Get("X-Request-ID"), "result": result})
		}),
		kratoshttp.ErrorEncoder(func(response http.ResponseWriter, request *http.Request, err error) {
			public := asPublicError(err)
			if public == nil {
				public = v1.NewPublicError(http.StatusInternalServerError, "INTERNAL_ERROR", "internal data service error", nil)
			}
			response.Header().Set("Content-Type", "application/json")
			response.WriteHeader(public.Status)
			_ = json.NewEncoder(response).Encode(map[string]any{"request_id": request.Header.Get("X-Request-ID"), "error": map[string]any{
				"code": public.Code, "message": public.Message, "details": public.Details,
			}})
		}),
	)
	v1.RegisterDataHTTPServer(server, NewDataService(dependencies))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := strings.TrimSpace(request.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 128 {
			requestID = generatedRequestID
			request.Header.Set("X-Request-ID", requestID)
		}
		response.Header().Set("X-Request-ID", requestID)
		scope, protected := testRequiredScope(request.Method, request.URL.Path)
		if !protected {
			server.ServeHTTP(response, request)
			return
		}
		token := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
		principal, ok := credentials[token]
		if !ok {
			writeTestError(response, requestID, http.StatusUnauthorized, "UNAUTHENTICATED", "valid service identity is required")
			return
		}
		if !principal.HasScope(scope) {
			writeTestError(response, requestID, http.StatusForbidden, "FORBIDDEN", "service identity lacks the required scope")
			return
		}
		server.ServeHTTP(response, request.WithContext(v1.WithPrincipal(request.Context(), principal)))
	})
}

func writeTestError(response http.ResponseWriter, requestID string, status int, code, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(map[string]any{"request_id": requestID, "error": map[string]any{
		"code": code, "message": message, "details": map[string]any{},
	}})
}

func testRequiredScope(method, path string) (string, bool) {
	switch {
	case method == http.MethodPost && path == Namespace+"/reviewed-event-imports":
		return ScopeReviewedEventImport, true
	case method == http.MethodGet && path == Namespace+"/event-tags":
		return ScopeEventTagRead, true
	case method == http.MethodPost && (path == Namespace+"/research-theme-imports" ||
		path == Namespace+"/research-reasoning-tree-imports"):
		return ScopeResearchImport, true
	case method == http.MethodGet && (path == Namespace+"/research/themes" ||
		strings.HasPrefix(path, Namespace+"/research/themes/")):
		return ScopeResearchRead, true
	case method == http.MethodGet && (path == Namespace+"/raw-documents" || path == Namespace+"/events"):
		return ScopeAdminRead, true
	default:
		return "", false
	}
}
