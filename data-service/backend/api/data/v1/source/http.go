package source

import (
	"context"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const ExecutionBudget = 3 * time.Second

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.GET("/sources", completeListHandler(application))
	router.POST("/sources", createHandler(application))
	router.PUT("/sources/{source_id}", updateHandler(application))
	router.DELETE("/sources/{source_id}", deleteHandler(application))
	router.GET("/source-snapshot", snapshotHandler(application))
}

func completeListHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if ctx.Request().URL.RawQuery != "" {
			return v1.NewPublicError(v1.StatusBadRequest, "INVALID_REQUEST", "Source list does not accept query parameters", nil)
		}
		return call(ctx, OperationList, nil, func(callContext context.Context) (*v1.Response[SourceList], error) {
			return application.List(callContext)
		})
	}
}

func createHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[CreateRequest](ctx)
		if err != nil {
			return err
		}
		return call(ctx, OperationCreate, request, func(callContext context.Context) (*v1.Response[Source], error) {
			return application.Create(callContext, request)
		})
	}
}

func updateHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[UpdateRequest](ctx)
		if err != nil {
			return err
		}
		request.SourceID = ctx.Vars().Get("source_id")
		return call(ctx, OperationUpdate, request, func(callContext context.Context) (*v1.Response[Source], error) {
			return application.Update(callContext, request)
		})
	}
}

func deleteHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &DeleteRequest{SourceID: ctx.Vars().Get("source_id")}
		return call(ctx, OperationDelete, request, func(callContext context.Context) (*v1.Response[DeleteResult], error) {
			return application.Delete(callContext, request)
		})
	}
}

func snapshotHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if ctx.Request().URL.RawQuery != "" {
			return v1.NewPublicError(v1.StatusBadRequest, "INVALID_REQUEST", "Source snapshot does not accept query parameters", nil)
		}
		return call(ctx, OperationSnapshot, nil, func(callContext context.Context) (*v1.Response[SourceSnapshot], error) {
			return application.Snapshot(callContext)
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
