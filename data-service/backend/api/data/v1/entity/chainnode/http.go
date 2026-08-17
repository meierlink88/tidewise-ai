package chainnode

import (
	"context"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const ExecutionBudget = 5 * time.Second

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.POST("/entities/chain-nodes", createHandler(application))
	router.GET("/entities/chain-nodes", listHandler(application))
	router.GET("/entities/chain-nodes/{chain_node_id}", getHandler(application))
	router.PUT("/entities/chain-nodes/{chain_node_id}", updateHandler(application))
}

func createHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[CreateRequest](ctx)
		if err != nil {
			return err
		}
		return call(ctx, OperationCreate, request, func(callContext context.Context) (*v1.Response[ChainNode], error) {
			return application.Create(callContext, request)
		})
	}
}

func listHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		query := ctx.Query()
		request := &ListRequest{PageSize: query.Get("page_size"), Cursor: query.Get("cursor")}
		return call(ctx, OperationList, request, func(callContext context.Context) (*v1.Response[ChainNodeList], error) {
			return application.List(callContext, request)
		})
	}
}

func getHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetRequest{ChainNodeID: ctx.Vars().Get("chain_node_id")}
		return call(ctx, OperationGet, request, func(callContext context.Context) (*v1.Response[ChainNode], error) {
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
		request.ChainNodeID = ctx.Vars().Get("chain_node_id")
		return call(ctx, OperationUpdate, request, func(callContext context.Context) (*v1.Response[ChainNode], error) {
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
