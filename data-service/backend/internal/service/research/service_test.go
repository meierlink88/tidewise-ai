package research

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	researchapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/research"
	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
)

type useCaseStub struct{ err error }

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
func (s useCaseStub) Search(context.Context, researchbiz.GraphSearchRequest) (researchbiz.GraphSearchResult, error) {
	return researchbiz.GraphSearchResult{}, s.err
}

func TestServiceRequiresTheUnifiedResearchUseCase(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) error = nil")
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
