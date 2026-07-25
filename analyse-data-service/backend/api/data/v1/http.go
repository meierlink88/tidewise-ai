package v1

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

const (
	APIPrefix          = "/api/data/v1"
	MaxRequestBodySize = 1_048_576
)

type DataHTTPServer interface {
	ImportReviewedEvents(context.Context, *ImportRequest) (*Response, error)
	ImportResearchThemes(context.Context, *ImportRequest) (*Response, error)
	ImportResearchAnchors(context.Context, *ImportRequest) (*Response, error)
	ListResearchThemes(context.Context, *ListResearchThemesRequest) (*Response, error)
	GetResearchTheme(context.Context, *GetResearchThemeRequest) (*Response, error)
	ListResearchReasoningTrees(context.Context, *ReasoningTreeListRequest) (*Response, error)
	GetResearchReasoningTree(context.Context, *ReasoningTreeDetailRequest) (*Response, error)
	ListRawDocuments(context.Context, *RawDocumentListRequest) (*Response, error)
	ListEvents(context.Context, *EventListRequest) (*Response, error)
}

func RegisterDataHTTPServer(server *kratoshttp.Server, application DataHTTPServer) {
	router := server.Route(APIPrefix)
	router.POST("/reviewed-event-imports", importHandler("data.v1.publishReviewedEvents", application.ImportReviewedEvents))
	router.POST("/research-theme-imports", importHandler("data.v1.importResearchThemes", application.ImportResearchThemes))
	router.POST("/research-anchor-imports", importHandler("data.v1.importResearchAnchors", application.ImportResearchAnchors))
	router.GET("/research/themes", listResearchThemesHandler(application))
	router.GET("/research/themes/{theme_id}", getResearchThemeHandler(application))
	router.GET("/research/themes/{theme_id}/reasoning-trees", listReasoningTreesHandler(application))
	router.GET("/research/themes/{theme_id}/reasoning-trees/{anchor_id}", getReasoningTreeHandler(application))
	router.GET("/raw-documents", listRawDocumentsHandler(application))
	router.GET("/events", listEventsHandler(application))
}

func importHandler(operation string, invoke func(context.Context, *ImportRequest) (*Response, error)) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := io.ReadAll(io.LimitReader(ctx.Request().Body, MaxRequestBodySize+1))
		if err != nil {
			return NewPublicError(http.StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
		}
		if len(payload) > MaxRequestBodySize {
			return NewPublicError(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1048576 bytes", nil)
		}
		request := &ImportRequest{Payload: payload}
		return call(ctx, operation, request, func(callContext context.Context) (*Response, error) {
			return invoke(callContext, request)
		})
	}
}

func listResearchThemesHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListResearchThemesRequest{
			WindowHours: ctx.Query().Get("window_hours"),
			Limit:       ctx.Query().Get("limit"),
			Cursor:      ctx.Query().Get("cursor"),
		}
		return call(ctx, "data.v1.listResearchThemes", request, func(callContext context.Context) (*Response, error) {
			return application.ListResearchThemes(callContext, request)
		})
	}
}

func getResearchThemeHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetResearchThemeRequest{ThemeID: ctx.Vars().Get("theme_id"), WindowHours: ctx.Query().Get("window_hours")}
		return call(ctx, "data.v1.getResearchTheme", request, func(callContext context.Context) (*Response, error) {
			return application.GetResearchTheme(callContext, request)
		})
	}
}

func listReasoningTreesHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ReasoningTreeListRequest{ThemeID: ctx.Vars().Get("theme_id"), HasQuery: ctx.Request().URL.RawQuery != ""}
		return call(ctx, "data.v1.listResearchThemeReasoningTrees", request, func(callContext context.Context) (*Response, error) {
			return application.ListResearchReasoningTrees(callContext, request)
		})
	}
}

func getReasoningTreeHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ReasoningTreeDetailRequest{
			ThemeID: ctx.Vars().Get("theme_id"), AnchorID: ctx.Vars().Get("anchor_id"), HasQuery: ctx.Request().URL.RawQuery != "",
		}
		return call(ctx, "data.v1.getResearchThemeReasoningTree", request, func(callContext context.Context) (*Response, error) {
			return application.GetResearchReasoningTree(callContext, request)
		})
	}
}

func listRawDocumentsHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &RawDocumentListRequest{
			Title: ctx.Query().Get("title"), SourceRef: ctx.Query().Get("source_ref"),
			IngestStatus: ctx.Query().Get("ingest_status"), Page: ctx.Query().Get("page"), PageSize: ctx.Query().Get("page_size"),
		}
		return call(ctx, "data.v1.listAdminRawDocuments", request, func(callContext context.Context) (*Response, error) {
			return application.ListRawDocuments(callContext, request)
		})
	}
}

func listEventsHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		query := ctx.Query()
		request := &EventListRequest{
			Title: query.Get("title"), EventStatus: query.Get("event_status"), FactStatus: query.Get("fact_status"),
			EventTimeFrom: query.Get("event_time_from"), EventTimeTo: query.Get("event_time_to"),
			FirstSeenFrom: query.Get("first_seen_from"), FirstSeenTo: query.Get("first_seen_to"),
			Page: query.Get("page"), PageSize: query.Get("page_size"),
		}
		return call(ctx, "data.v1.listAdminEvents", request, func(callContext context.Context) (*Response, error) {
			return application.ListEvents(callContext, request)
		})
	}
}

func call(ctx kratoshttp.Context, operation string, request any, invoke func(context.Context) (*Response, error)) error {
	kratoshttp.SetOperation(ctx, operation)
	handler := ctx.Middleware(func(callContext context.Context, _ any) (any, error) {
		return invoke(callContext)
	})
	result, err := handler(ctx, request)
	if err != nil {
		return err
	}
	response, ok := result.(*Response)
	if !ok || response == nil {
		return fmt.Errorf("data API operation %s returned an invalid response", operation)
	}
	return ctx.Result(response.Status, response.Result)
}

func ParseBoundedInt(raw string, fallback, minimum, maximum int, name string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, NewPublicError(http.StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("%s must be between %d and %d", name, minimum, maximum), nil)
	}
	return value, nil
}
