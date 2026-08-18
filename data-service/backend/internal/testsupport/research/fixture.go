package research

import (
	"context"
	"net/http"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	researchapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/research"
)

type Service struct{}

func (Service) PublishResearchTheme(context.Context, *researchapi.ResearchThemeImportRequest) (*v1.Response[researchapi.ResearchThemeImportResult], error) {
	return response[researchapi.ResearchThemeImportResult](), nil
}

func (Service) ListResearchThemes(context.Context, *researchapi.ListResearchThemesRequest) (*v1.Response[researchapi.ResearchThemePage], error) {
	return response[researchapi.ResearchThemePage](), nil
}

func (Service) GetResearchTheme(context.Context, *researchapi.GetResearchThemeRequest) (*v1.Response[researchapi.ResearchThemeDetail], error) {
	return response[researchapi.ResearchThemeDetail](), nil
}

func (Service) ListResearchReasoningTrees(context.Context, *researchapi.ReasoningTreeListRequest) (*v1.Response[researchapi.ResearchReasoningTreeList], error) {
	return response[researchapi.ResearchReasoningTreeList](), nil
}

func (Service) GetResearchReasoningTree(context.Context, *researchapi.ReasoningTreeDetailRequest) (*v1.Response[researchapi.ResearchReasoningTreeDetail], error) {
	return response[researchapi.ResearchReasoningTreeDetail](), nil
}

func (Service) SearchResearchGraph(context.Context, *researchapi.ResearchGraphSearchRequest) (*v1.Response[researchapi.ResearchGraphSearchResult], error) {
	return response[researchapi.ResearchGraphSearchResult](), nil
}

func response[T any]() *v1.Response[T] {
	return &v1.Response[T]{Status: http.StatusNoContent}
}

var _ researchapi.Service = Service{}
