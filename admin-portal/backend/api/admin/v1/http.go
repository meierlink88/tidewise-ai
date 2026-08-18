package v1

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

func RegisterAdminHTTPServer(server *kratoshttp.Server, service AdminHTTPServer) {
	router := server.Route(APIPrefix)
	router.GET("/raw-documents", listRawDocumentsHandler(service))
	router.GET("/events", listEventsHandler(service))
	router.GET("/runtime-health", getRuntimeHealthHandler(service))
}

func getRuntimeHealthHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &EmptyRequest{}
		return call(ctx, OperationGetRuntimeHealth, request, func(callContext context.Context) (any, error) {
			return service.GetRuntimeHealth(callContext, request)
		})
	}
}

func listRawDocumentsHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		page, pageSize, err := parsePage(ctx, 50)
		if err != nil {
			return err
		}
		request := &ListRawDocumentsRequest{
			Title: ctx.Query().Get("title"), SourceRef: ctx.Query().Get("source_ref"),
			Page: page, PageSize: pageSize,
		}
		return call(ctx, OperationListRawDocuments, request, func(callContext context.Context) (any, error) {
			return service.ListRawDocuments(callContext, request)
		})
	}
}

func listEventsHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		page, pageSize, err := parsePage(ctx, 50)
		if err != nil {
			return err
		}
		request := &ListEventsRequest{
			Title: ctx.Query().Get("title"), EventStatus: ctx.Query().Get("event_status"),
			FactStatus: ctx.Query().Get("fact_status"), EventTimeFrom: ctx.Query().Get("event_time_from"),
			EventTimeTo: ctx.Query().Get("event_time_to"), FirstSeenFrom: ctx.Query().Get("first_seen_from"),
			FirstSeenTo: ctx.Query().Get("first_seen_to"), Page: page, PageSize: pageSize,
		}
		return call(ctx, OperationListEvents, request, func(callContext context.Context) (any, error) {
			return service.ListEvents(callContext, request)
		})
	}
}

func call(
	ctx kratoshttp.Context,
	operation string,
	request any,
	invoke func(context.Context) (any, error),
) error {
	kratoshttp.SetOperation(ctx, operation)
	handler := ctx.Middleware(func(callContext context.Context, _ any) (any, error) {
		return invoke(callContext)
	})
	response, err := handler(ctx, request)
	if err != nil {
		return err
	}
	return ctx.Result(http.StatusOK, response)
}

func parsePage(ctx kratoshttp.Context, defaultPageSize int) (int, int, error) {
	page, err := parsePositiveInt(ctx.Query().Get("page"), 1, "page must be positive")
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := parsePositiveInt(ctx.Query().Get("page_size"), defaultPageSize, "page_size must be positive")
	if err != nil {
		return 0, 0, err
	}
	return page, pageSize, nil
}

func parsePositiveInt(value string, fallback int, message string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", message)
	}
	return parsed, nil
}
