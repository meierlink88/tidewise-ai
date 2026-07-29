package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
)

func TestResearchThemeImportMapsV1Contract(t *testing.T) {
	importer := &fakeThemeImporter{result: researchpublication.Result{
		ReceiptID: "receipt", AnalysisBatchID: "batch", ThemeID: "theme",
		ReasoningTreeIDsByIndustryChainEntityID: map[string]string{"chain": "tree"},
		Counts:                                  researchpublication.Counts{Themes: 1, Impacts: 2, ReasoningTrees: 1, Nodes: 2, Receipts: 2},
	}}
	handler := NewDataService(Dependencies{ResearchThemeImports: importer})
	attention := "high"
	signalID := "44444444-4444-4444-8444-444444444444"
	submissionID := "55555555-5555-4555-8555-555555555555"
	evidenceID := "66666666-6666-4666-8666-666666666666"
	evidenceHash := strings.Repeat("a", 64)
	response, err := handler.PublishResearchTheme(v1.WithPrincipal(context.Background(), v1.Principal{Identity: "analyst"}), &v1.ResearchThemeImportRequest{
		AnalysisBatchID: "batch", AnalysisAsOf: "2026-07-28T00:00:00Z",
		DiscoveryWindowStart: "2026-07-27T00:00:00Z", DiscoveryWindowEnd: "2026-07-28T00:00:00Z",
		Theme: v1.ResearchThemeImportItem{
			ThemeKey: "ai", Title: "AI infrastructure", OneLineConclusion: "Demand rises",
			ConclusionDirection: "positive", ImpactStrength: "medium", AttentionLevel: &attention,
			TransmissionStage: "validation", InvestmentGuidanceAction: "focus",
			InvestmentGuidanceSummary: "Watch orders", TimeHorizonCategory: "short_term",
			Impacts: []v1.ResearchThemeImportImpact{{
				ChainNodeEntityID: "11111111-1111-4111-8111-111111111111",
				RelationRole:      "beneficiary", ImpactDirection: "positive", DisplayOrder: 1,
			}},
		},
		ReasoningTrees: []v1.ResearchReasoningTreeImportItem{{
			IndustryChainEntityID: "22222222-2222-4222-8222-222222222222",
			Title:                 "Optical module", DisplayOrder: 1, OneLineConclusion: "Demand rises",
			ImpactDirection: "positive", ImpactStrength: "medium",
			Nodes: []v1.ResearchReasoningTreeImportNode{{
				Position: 1, ChainNodeEntityID: "33333333-3333-4333-8333-333333333333",
				ImpactDirection: "positive", ImpactStrength: "medium",
				Signals: []v1.ResearchReasoningTreeImportSignal{{
					VariableSignalKey: "market_demand", SignalRole: "primary",
					SignalDirection: "increase", DisplaySummary: "Demand rises", DisplayOrder: 1,
					Lineage: v1.ResearchReasoningTreeSignalLineage{
						SourceKind: "formal_signal", VariableSignalID: &signalID,
						SemanticSubmissionID: &submissionID, EvidenceID: &evidenceID,
						EvidenceHash: &evidenceHash,
					},
				}},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("PublishResearchTheme() error = %v", err)
	}
	if importer.publisher != "analyst" || importer.aggregate.AnalysisAsOf != "2026-07-28T00:00:00Z" {
		t.Fatalf("import call = publisher %q aggregate %#v", importer.publisher, importer.aggregate)
	}
	if response.Result.Counts.Impacts != 2 || response.Result.ThemeID != "theme" ||
		importer.aggregate.ReasoningTrees[0].Nodes[0].Signals[0].Lineage.SourceKind != "formal_signal" {
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

type fakeThemeImporter struct {
	result    researchpublication.Result
	aggregate researchpublication.Aggregate
	publisher string
}

func (f *fakeThemeImporter) Publish(_ context.Context, publisher string, aggregate researchpublication.Aggregate) (researchpublication.Result, error) {
	f.publisher, f.aggregate = publisher, aggregate
	return f.result, nil
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
