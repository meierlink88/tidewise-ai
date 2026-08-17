package concept

import (
	"context"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const ExecutionBudget = 5 * time.Second

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.POST("/entities/concepts", createHandler(application))
	router.GET("/entities/concepts", listHandler(application))
	router.GET("/entities/concepts/{concept_id}", getHandler(application))
	router.PUT("/entities/concepts/{concept_id}", updateHandler(application))
}

func createHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[CreateRequest](ctx)
		if err != nil {
			return err
		}
		return call(ctx, OperationCreate, request, func(callContext context.Context) (*v1.Response[Concept], error) {
			return application.Create(callContext, request)
		})
	}
}

func listHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListRequest{}
		return call(ctx, OperationList, request, func(callContext context.Context) (*v1.Response[ConceptList], error) {
			return application.List(callContext, request)
		})
	}
}

func getHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetRequest{ConceptID: ctx.Vars().Get("concept_id")}
		return call(ctx, OperationGet, request, func(callContext context.Context) (*v1.Response[Concept], error) {
			return application.Get(callContext, request)
		})
	}
}

func updateHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[UpdateRequest](ctx)
		if err != nil {
			return err
		}
		request.ConceptID = ctx.Vars().Get("concept_id")
		return call(ctx, OperationUpdate, request, func(callContext context.Context) (*v1.Response[Concept], error) {
			return application.Update(callContext, request)
		})
	}
}

func call[T any](ctx kratoshttp.Context, operation string, request any, invoke func(context.Context) (*v1.Response[T], error)) error {
	return v1.Call(ctx, operation, request, func(callContext context.Context) (*v1.Response[T], error) {
		deadlineContext, cancel := context.WithTimeout(callContext, ExecutionBudget)
		defer cancel()
		return invoke(deadlineContext)
	})
}
