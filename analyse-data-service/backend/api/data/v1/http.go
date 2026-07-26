package v1

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

const (
	APIPrefix          = "/api/data/v1"
	MaxRequestBodySize = 1_048_576
)

type DataHTTPServer interface {
	ImportReviewedEvents(context.Context, *EventPublicationRequest) (*Response[EventPublicationResult], error)
	ListActiveEventTags(context.Context, *EventTagCatalogRequest) (*Response[EventTagCatalog], error)
	ImportResearchThemes(context.Context, *ResearchThemeImportRequest) (*Response[ResearchThemeImportResult], error)
	ImportResearchAnchors(context.Context, *ResearchAnchorImportRequest) (*Response[ResearchAnchorImportResult], error)
	ListResearchThemes(context.Context, *ListResearchThemesRequest) (*Response[ResearchThemePage], error)
	GetResearchTheme(context.Context, *GetResearchThemeRequest) (*Response[ResearchThemeDetail], error)
	ListResearchReasoningTrees(context.Context, *ReasoningTreeListRequest) (*Response[ResearchReasoningTreeList], error)
	GetResearchReasoningTree(context.Context, *ReasoningTreeDetailRequest) (*Response[ResearchReasoningTreeDetail], error)
	ListRawDocuments(context.Context, *RawDocumentListRequest) (*Response[AdminRawDocumentPage], error)
	ListEvents(context.Context, *EventListRequest) (*Response[AdminEventPage], error)
}

func RegisterDataHTTPServer(server *kratoshttp.Server, application DataHTTPServer) {
	router := server.Route(APIPrefix)
	router.POST("/reviewed-event-imports", eventPublicationImportHandler(application))
	router.GET("/event-tags", listActiveEventTagsHandler(application))
	router.POST("/research-theme-imports", researchThemeImportHandler(application))
	router.POST("/research-anchor-imports", researchAnchorImportHandler(application))
	router.GET("/research/themes", listResearchThemesHandler(application))
	router.GET("/research/themes/{theme_id}", getResearchThemeHandler(application))
	router.GET("/research/themes/{theme_id}/reasoning-trees", listReasoningTreesHandler(application))
	router.GET("/research/themes/{theme_id}/reasoning-trees/{anchor_id}", getReasoningTreeHandler(application))
	router.GET("/raw-documents", listRawDocumentsHandler(application))
	router.GET("/events", listEventsHandler(application))
}

func listActiveEventTagsHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		query := ctx.Request().URL.Query()
		values, exists := query["active"]
		if !exists || len(query) != 1 || len(values) != 1 || values[0] != "true" {
			return NewPublicError(StatusBadRequest, "INVALID_REQUEST", "active must be exactly true", nil)
		}
		request := &EventTagCatalogRequest{Active: true}
		return call(ctx, OperationListActiveEventTags, request, func(callContext context.Context) (*Response[EventTagCatalog], error) {
			return application.ListActiveEventTags(callContext, request)
		})
	}
}

func eventPublicationImportHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := readImportPayload(ctx)
		if err != nil {
			return err
		}
		request, err := decodeEventPublication(payload)
		if err != nil {
			return err
		}
		return call(ctx, OperationPublishReviewedEvents, request, func(callContext context.Context) (*Response[EventPublicationResult], error) {
			return application.ImportReviewedEvents(callContext, request)
		})
	}
}

func researchThemeImportHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := readImportPayload(ctx)
		if err != nil {
			return err
		}
		request, err := decodeResearchThemeImport(payload)
		if err != nil {
			return err
		}
		return call(ctx, OperationImportResearchThemes, request, func(callContext context.Context) (*Response[ResearchThemeImportResult], error) {
			return application.ImportResearchThemes(callContext, request)
		})
	}
}

func researchAnchorImportHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := readImportPayload(ctx)
		if err != nil {
			return err
		}
		request, err := decodeResearchAnchorImport(payload)
		if err != nil {
			return err
		}
		return call(ctx, OperationImportResearchAnchors, request, func(callContext context.Context) (*Response[ResearchAnchorImportResult], error) {
			return application.ImportResearchAnchors(callContext, request)
		})
	}
}

func readImportPayload(ctx kratoshttp.Context) ([]byte, error) {
	payload, err := io.ReadAll(io.LimitReader(ctx.Request().Body, MaxRequestBodySize+1))
	if err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	if len(payload) > MaxRequestBodySize {
		return nil, NewPublicError(StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1048576 bytes", nil)
	}
	return payload, nil
}

func listResearchThemesHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListResearchThemesRequest{
			WindowHours: ctx.Query().Get("window_hours"),
			Limit:       ctx.Query().Get("limit"),
			Cursor:      ctx.Query().Get("cursor"),
		}
		return call(ctx, OperationListResearchThemes, request, func(callContext context.Context) (*Response[ResearchThemePage], error) {
			return application.ListResearchThemes(callContext, request)
		})
	}
}

func getResearchThemeHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetResearchThemeRequest{ThemeID: ctx.Vars().Get("theme_id"), WindowHours: ctx.Query().Get("window_hours")}
		return call(ctx, OperationGetResearchTheme, request, func(callContext context.Context) (*Response[ResearchThemeDetail], error) {
			return application.GetResearchTheme(callContext, request)
		})
	}
}

func listReasoningTreesHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ReasoningTreeListRequest{ThemeID: ctx.Vars().Get("theme_id"), HasQuery: ctx.Request().URL.RawQuery != ""}
		return call(ctx, OperationListResearchThemeReasoningTrees, request, func(callContext context.Context) (*Response[ResearchReasoningTreeList], error) {
			return application.ListResearchReasoningTrees(callContext, request)
		})
	}
}

func getReasoningTreeHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ReasoningTreeDetailRequest{
			ThemeID: ctx.Vars().Get("theme_id"), AnchorID: ctx.Vars().Get("anchor_id"), HasQuery: ctx.Request().URL.RawQuery != "",
		}
		return call(ctx, OperationGetResearchThemeReasoningTree, request, func(callContext context.Context) (*Response[ResearchReasoningTreeDetail], error) {
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
		return call(ctx, OperationListAdminRawDocuments, request, func(callContext context.Context) (*Response[AdminRawDocumentPage], error) {
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
		return call(ctx, OperationListAdminEvents, request, func(callContext context.Context) (*Response[AdminEventPage], error) {
			return application.ListEvents(callContext, request)
		})
	}
}

func call[T any](ctx kratoshttp.Context, operation string, request any, invoke func(context.Context) (*Response[T], error)) error {
	kratoshttp.SetOperation(ctx, operation)
	handler := ctx.Middleware(func(callContext context.Context, _ any) (any, error) {
		return invoke(callContext)
	})
	result, err := handler(ctx, request)
	if err != nil {
		return err
	}
	response, ok := result.(*Response[T])
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
		return 0, NewPublicError(StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("%s must be between %d and %d", name, minimum, maximum), nil)
	}
	return value, nil
}
