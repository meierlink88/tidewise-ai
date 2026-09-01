package research

import (
	"context"
	"net/http"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	researchapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/research"
)

type Service struct{}

func (Service) SearchResearchGraph(context.Context, *researchapi.ResearchGraphSearchRequest) (*v1.Response[researchapi.ResearchGraphSearchResult], error) {
	return response[researchapi.ResearchGraphSearchResult](), nil
}

func response[T any]() *v1.Response[T] {
	return &v1.Response[T]{Status: http.StatusNoContent}
}

var _ researchapi.Service = Service{}
