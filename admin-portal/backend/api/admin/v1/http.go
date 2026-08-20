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
	router.GET("/events", listEventsHandler(service))
	router.GET("/evidences", listEvidencesHandler(service))
	router.GET("/raw-evidences/{raw_evidence_id}/collection-document", getCollectionDocumentHandler(service))
	router.GET("/evidence-categories", listEvidenceCategoriesHandler(service))
	router.GET("/sources", listSourcesHandler(service))
	router.GET("/runtime-health", getRuntimeHealthHandler(service))
}

func getCollectionDocumentHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "collection document does not accept query parameters")
		}
		request := &GetCollectionDocumentRequest{RawEvidenceID: ctx.Vars().Get("raw_evidence_id")}
		return call(ctx, OperationGetCollectionDocument, request, func(callContext context.Context) (any, error) {
			return service.GetCollectionDocument(callContext, request)
		})
	}
}

func listEvidenceCategoriesHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "evidence categories do not accept query parameters")
		}
		request := &EmptyRequest{}
		return call(ctx, OperationListEvidenceCategories, request, func(callContext context.Context) (any, error) {
			return service.ListEvidenceCategories(callContext, request)
		})
	}
}

func listEvidencesHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		for name := range ctx.Request().URL.Query() {
			if !allowedEvidenceQueryParameter(name) {
				return NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "unsupported Evidence query parameter")
			}
		}
		page, pageSize, err := parsePage(ctx, 50)
		if err != nil {
			return err
		}
		query := ctx.Query()
		request := &ListEvidencesRequest{
			Title: query.Get("title"), Summary: query.Get("summary"), CategoryID: query.Get("category_id"),
			SourceID: query.Get("source_id"), SourceName: query.Get("source_name"), SourceLevel: query.Get("source_level"), IsSplit: query.Get("is_split"),
			PublishedFrom: query.Get("published_from"), PublishedTo: query.Get("published_to"),
			CollectedFrom: query.Get("collected_from"), CollectedTo: query.Get("collected_to"), Page: page, PageSize: pageSize,
		}
		return call(ctx, OperationListEvidences, request, func(callContext context.Context) (any, error) {
			return service.ListEvidences(callContext, request)
		})
	}
}

func allowedEvidenceQueryParameter(name string) bool {
	switch name {
	case "title", "summary", "category_id", "source_id", "source_name", "source_level", "is_split", "published_from", "published_to", "collected_from", "collected_to", "page", "page_size":
		return true
	default:
		return false
	}
}

func listSourcesHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		for name := range ctx.Request().URL.Query() {
			if !allowedSourceQueryParameter(name) {
				return NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "unsupported Source query parameter")
			}
		}
		page, pageSize, err := parsePage(ctx, 50)
		if err != nil {
			return err
		}
		query := ctx.Query()
		request := &ListSourcesRequest{
			Query: query.Get("query"), OwnershipType: query.Get("ownership_type"), ChannelType: query.Get("channel_type"),
			Enabled: query.Get("enabled"), Priority: query.Get("priority"), DefaultSourceLevel: query.Get("default_source_level"),
			UpdatedFrom: query.Get("updated_from"), UpdatedTo: query.Get("updated_to"), Page: page, PageSize: pageSize,
		}
		return call(ctx, OperationListSources, request, func(callContext context.Context) (any, error) {
			return service.ListSources(callContext, request)
		})
	}
}

func allowedSourceQueryParameter(name string) bool {
	switch name {
	case "query", "ownership_type", "channel_type", "enabled", "priority", "default_source_level", "updated_from", "updated_to", "page", "page_size":
		return true
	default:
		return false
	}
}

func getRuntimeHealthHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &EmptyRequest{}
		return call(ctx, OperationGetRuntimeHealth, request, func(callContext context.Context) (any, error) {
			return service.GetRuntimeHealth(callContext, request)
		})
	}
}

func listEventsHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		for name := range ctx.Request().URL.Query() {
			if !allowedEventQueryParameter(name) {
				return NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "unsupported Event query parameter")
			}
		}
		page, pageSize, err := parsePage(ctx, 50)
		if err != nil {
			return err
		}
		request := &ListEventsRequest{
			Title: ctx.Query().Get("title"), Modality: ctx.Query().Get("modality"), Status: ctx.Query().Get("status"),
			OccurredFrom: ctx.Query().Get("occurred_from"), OccurredTo: ctx.Query().Get("occurred_to"),
			AnnouncedFrom: ctx.Query().Get("announced_from"), AnnouncedTo: ctx.Query().Get("announced_to"),
			Page: page, PageSize: pageSize,
		}
		return call(ctx, OperationListEvents, request, func(callContext context.Context) (any, error) {
			return service.ListEvents(callContext, request)
		})
	}
}

func allowedEventQueryParameter(name string) bool {
	switch name {
	case "title", "modality", "status", "occurred_from", "occurred_to", "announced_from", "announced_to", "page", "page_size":
		return true
	default:
		return false
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
	if page > 1_000_000 {
		return 0, 0, NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "page must not exceed 1000000")
	}
	pageSize, err := parsePositiveInt(ctx.Query().Get("page_size"), defaultPageSize, "page_size must be positive")
	if err != nil {
		return 0, 0, err
	}
	if pageSize > 100 {
		return 0, 0, NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "page_size must not exceed 100")
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
