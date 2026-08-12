package runtimehealth

import (
	"context"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

func RegisterHTTPServer(server *kratoshttp.Server, service Service) {
	router := server.Route(v1.APIPrefix)
	router.GET("/runtime-health", handler(service))
}

func handler(service Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &Request{}
		return v1.Call(ctx, OperationGet, request, func(callContext context.Context) (*v1.Response[Result], error) {
			return service.GetRuntimeHealth(callContext, request)
		})
	}
}
