package research

import (
	"context"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const HeavyExecutionBudget = 15 * time.Second

func callWithBudget[T any](ctx kratoshttp.Context, operation string, budget time.Duration, request any, invoke func(context.Context) (*v1.Response[T], error)) error {
	return v1.Call(ctx, operation, request, func(callContext context.Context) (*v1.Response[T], error) {
		deadlineContext, cancel := context.WithTimeout(callContext, budget)
		defer cancel()
		return invoke(deadlineContext)
	})
}

func searchResearchGraphHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[ResearchGraphSearchRequest](ctx)
		if err != nil {
			return err
		}
		return callWithBudget(
			ctx,
			OperationSearchResearchGraph,
			HeavyExecutionBudget,
			request,
			func(callContext context.Context) (*v1.Response[ResearchGraphSearchResult], error) {
				return application.SearchResearchGraph(callContext, request)
			},
		)
	}
}

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	if server == nil || application == nil {
		return
	}
	router := server.Route("/api/data/v1")
	router.POST("/research-graph:search", searchResearchGraphHandler(application))
}
