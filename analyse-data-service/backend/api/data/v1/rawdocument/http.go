package rawdocument

import (
	"context"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.GET("/raw-documents", listHandler(application))
}

func listHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListRequest{
			Title: ctx.Query().Get("title"), SourceRef: ctx.Query().Get("source_ref"),
			IngestStatus: ctx.Query().Get("ingest_status"), Page: ctx.Query().Get("page"), PageSize: ctx.Query().Get("page_size"),
		}
		return v1.Call(ctx, OperationList, request, func(callContext context.Context) (*v1.Response[Page], error) {
			return application.List(callContext, request)
		})
	}
}
