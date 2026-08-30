package company

import (
	"context"
	"strings"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const ExecutionBudget = 5 * time.Second

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.GET("/entities/companies", listHandler(application))
}

func listHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		query := ctx.Query()
		for key, values := range query {
			if (key != "page_size" && key != "cursor") || len(values) != 1 ||
				((key == "page_size" || key == "cursor") && strings.TrimSpace(values[0]) == "") {
				return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "Company projection query is invalid", nil)
			}
		}
		request := &ListRequest{PageSize: query.Get("page_size"), Cursor: query.Get("cursor")}
		return v1.Call(ctx, OperationList, request, func(callContext context.Context) (*v1.Response[CompanyProjectionPage], error) {
			deadlineContext, cancel := context.WithTimeout(callContext, ExecutionBudget)
			defer cancel()
			return application.List(deadlineContext, request)
		})
	}
}
