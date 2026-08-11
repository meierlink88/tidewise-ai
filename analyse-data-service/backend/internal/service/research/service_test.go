package research

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	researchapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/research"
	researchbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
)

type useCaseStub struct {
	analysis researchbiz.AnalysisContextResult
	err      error
}

func (s useCaseStub) Publish(context.Context, string, researchbiz.Aggregate) (researchbiz.Result, error) {
	return researchbiz.Result{}, s.err
}
func (s useCaseStub) PublishSnapshot(context.Context, string, researchbiz.SnapshotAggregate) (researchbiz.Result, error) {
	return researchbiz.Result{}, s.err
}
func (s useCaseStub) ListThemes(context.Context, researchbiz.ResearchListRequest) (researchbiz.ResearchThemePage, error) {
	return researchbiz.ResearchThemePage{}, s.err
}
func (s useCaseStub) GetTheme(context.Context, string, researchbiz.ResearchDetailRequest) (researchbiz.ResearchThemeDetail, error) {
	return researchbiz.ResearchThemeDetail{}, s.err
}
func (s useCaseStub) ListReasoningTrees(context.Context, string) (researchbiz.ResearchReasoningTreeList, error) {
	return researchbiz.ResearchReasoningTreeList{}, s.err
}
func (s useCaseStub) GetReasoningTree(context.Context, string, string) (researchbiz.ResearchReasoningTreeDetail, error) {
	return researchbiz.ResearchReasoningTreeDetail{}, s.err
}
func (s useCaseStub) List(context.Context, researchbiz.AnalysisContextRequest) (researchbiz.AnalysisContextResult, error) {
	return s.analysis, s.err
}
func (s useCaseStub) Search(context.Context, researchbiz.GraphSearchRequest) (researchbiz.GraphSearchResult, error) {
	return researchbiz.GraphSearchResult{}, s.err
}

func TestServiceRequiresTheUnifiedResearchUseCase(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) error = nil")
	}
}

func TestServiceMapsAnalysisContextWithoutOwningDomainPolicy(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	service, err := NewService(useCaseStub{analysis: researchbiz.AnalysisContextResult{
		ContractVersion:     researchbiz.AnalysisContextContractVersion,
		TBoxContractVersion: researchbiz.AnalysisContextTBoxContractVersion,
		TemporalSemantics:   researchbiz.AnalysisContextTemporalSemantics,
		TemporalLimitation:  researchbiz.AnalysisContextTemporalLimitation,
		AnalysisAsOf:        "2026-08-12T00:00:00Z",
		EventSemanticBundles: []researchbiz.EventSemanticBundle{{
			Event: researchbiz.Event{ID: "11111111-1111-4111-8111-111111111111", Title: "Event", FirstSeenAt: now, KnowledgeAvailableAt: now},
			Evidence: []researchbiz.Evidence{{
				EvidenceID: "22222222-2222-4222-8222-222222222222", PublishedAt: &now,
				FirstSeenAt: now, KnowledgeAvailableAt: now, AcceptedAt: now,
			}},
		}},
		Dictionaries: researchbiz.Dictionaries{Entities: []researchbiz.Entity{{
			EntityID: "33333333-3333-4333-8333-333333333333", EntityType: "company", Name: "Company", CanonicalName: "Company", Status: "active",
		}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.ListResearchAnalysisContext(context.Background(), analysisRequest())
	if err != nil {
		t.Fatal(err)
	}
	if response.Result.ContractVersion != researchbiz.AnalysisContextContractVersion ||
		len(response.Result.EventSemanticBundles) != 1 ||
		response.Result.EventSemanticBundles[0].Evidence[0].PublishedAt == nil ||
		len(response.Result.Dictionaries.Entities) != 1 {
		t.Fatalf("result = %#v", response.Result)
	}
}

func TestServiceKeepsStableAnalysisContextFailureClassification(t *testing.T) {
	service, err := NewService(useCaseStub{err: researchbiz.ErrHistoricalSemanticsUnavailable})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ListResearchAnalysisContext(context.Background(), analysisRequest())
	var public *v1.PublicError
	if err == nil || errors.Is(err, researchbiz.ErrHistoricalSemanticsUnavailable) ||
		!errors.As(err, &public) || public.Status != v1.StatusUnprocessableEntity ||
		public.Code != "RESEARCH_ANALYSIS_CONTEXT_HISTORY_UNAVAILABLE" {
		t.Fatalf("public error = %v", err)
	}
}

func TestServiceKeepsStableAnalysisContextResourceLimitClassification(t *testing.T) {
	actual := int64(50_001)
	maximum := int64(50_000)
	service, err := NewService(useCaseStub{err: &researchbiz.ResearchResourceLimitError{
		Reason: "reference closure exceeds row budget", Component: "reference_closure",
		ActualRows: &actual, MaxRows: &maximum, RetryGuidance: "reduce_page_size",
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ListResearchAnalysisContext(context.Background(), analysisRequest())
	var public *v1.PublicError
	if !errors.As(err, &public) || public.Status != v1.StatusTooManyRequests ||
		public.Code != "RESEARCH_ANALYSIS_CONTEXT_RESOURCE_LIMIT" {
		t.Fatalf("public resource limit = %#v", err)
	}
}

func TestServiceKeepsStablePublicationReadAndGraphErrorContracts(t *testing.T) {
	actual := int64(11)
	maximum := int64(10)
	tests := []struct {
		name       string
		domainErr  error
		invoke     func(*Service) error
		wantStatus int
		wantCode   string
	}{
		{
			name: "publication reference", domainErr: &researchbiz.ReferenceError{Path: "theme.events[0]", Reference: "missing", Message: "unavailable"},
			invoke: func(service *Service) error {
				_, err := service.PublishResearchTheme(context.Background(), &researchapi.ResearchThemeImportRequest{})
				return err
			},
			wantStatus: v1.StatusUnprocessableEntity, wantCode: "RESEARCH_THEME_REFERENCE_INVALID",
		},
		{
			name: "publication conflict", domainErr: researchbiz.ErrPayloadConflict,
			invoke: func(service *Service) error {
				_, err := service.PublishResearchTheme(context.Background(), &researchapi.ResearchThemeImportRequest{})
				return err
			},
			wantStatus: v1.StatusConflict, wantCode: "RESEARCH_THEME_PAYLOAD_CONFLICT",
		},
		{
			name: "read not found", domainErr: researchbiz.ErrNotFound,
			invoke: func(service *Service) error {
				_, err := service.GetResearchTheme(context.Background(), &researchapi.GetResearchThemeRequest{ThemeID: "missing"})
				return err
			},
			wantStatus: v1.StatusNotFound, wantCode: "NOT_FOUND",
		},
		{
			name: "reason tree invariant", domainErr: researchbiz.ErrReasoningTreeInvariantViolation,
			invoke: func(service *Service) error {
				_, err := service.GetResearchReasoningTree(context.Background(), &researchapi.ReasoningTreeDetailRequest{})
				return err
			},
			wantStatus: v1.StatusInternalServerError, wantCode: "RESEARCH_REASONING_TREE_INVARIANT_VIOLATION",
		},
		{
			name: "graph validation", domainErr: &researchbiz.GraphValidationError{Reason: "invalid graph query"},
			invoke: func(service *Service) error {
				_, err := service.SearchResearchGraph(context.Background(), &researchapi.ResearchGraphSearchRequest{})
				return err
			},
			wantStatus: v1.StatusBadRequest, wantCode: "RESEARCH_GRAPH_INVALID",
		},
		{
			name: "graph resource", domainErr: &researchbiz.ResearchResourceLimitError{
				Reason: "too many nodes", Component: "research_graph_nodes", ActualRows: &actual, MaxRows: &maximum,
			},
			invoke: func(service *Service) error {
				_, err := service.SearchResearchGraph(context.Background(), &researchapi.ResearchGraphSearchRequest{})
				return err
			},
			wantStatus: v1.StatusTooManyRequests, wantCode: "RESEARCH_GRAPH_RESOURCE_LIMIT",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewService(useCaseStub{err: test.domainErr})
			if err != nil {
				t.Fatal(err)
			}
			err = test.invoke(service)
			var public *v1.PublicError
			if !errors.As(err, &public) || public.Status != test.wantStatus || public.Code != test.wantCode {
				t.Fatalf("public error = %#v, want status=%d code=%s", err, test.wantStatus, test.wantCode)
			}
		})
	}
}

func analysisRequest() *researchapi.ResearchAnalysisContextRequest {
	return &researchapi.ResearchAnalysisContextRequest{
		DiscoveryWindowStart: "2026-08-11T00:00:00Z",
		DiscoveryWindowEnd:   "2026-08-12T00:00:00Z",
		AnalysisAsOf:         "2026-08-12T00:00:00Z",
		PageSize:             20,
	}
}
