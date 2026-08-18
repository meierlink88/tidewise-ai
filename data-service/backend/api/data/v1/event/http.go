package event

import (
	"context"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	server.Route(v1.APIPrefix).GET("/events", listHandler(application))
}

func listHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		for name := range ctx.Request().URL.Query() {
			if !allowedListQueryParameter(name) {
				return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "unsupported Event query parameter", nil)
			}
		}
		query := ctx.Query()
		request := &ListRequest{
			Title: query.Get("title"), Modality: query.Get("modality"), Status: query.Get("status"),
			OccurredFrom: query.Get("occurred_from"), OccurredTo: query.Get("occurred_to"),
			AnnouncedFrom: query.Get("announced_from"), AnnouncedTo: query.Get("announced_to"),
			Page: query.Get("page"), PageSize: query.Get("page_size"),
		}
		return v1.Call(ctx, OperationListAdminEvents, request, func(callContext context.Context) (*v1.Response[Page], error) {
			return application.ListEvents(callContext, request)
		})
	}
}

func allowedListQueryParameter(name string) bool {
	switch name {
	case "title", "modality", "status", "occurred_from", "occurred_to", "announced_from", "announced_to", "page", "page_size":
		return true
	default:
		return false
	}
}
