package v1

import (
	"bytes"
	"context"
	"encoding/json"
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

type StrictJSONShape = bindingShape

func StrictJSONString() *StrictJSONShape         { return stringShape }
func StrictJSONScalar() *StrictJSONShape         { return scalarShape }
func StrictJSONBoolean() *StrictJSONShape        { return booleanShape }
func StrictJSONInteger() *StrictJSONShape        { return integerShape }
func StrictJSONNullableString() *StrictJSONShape { return nullableStringShape }
func StrictJSONAny() *StrictJSONShape            { return anyShape }
func StrictJSONArray(item *StrictJSONShape) *StrictJSONShape {
	return arrayShape(item)
}
func StrictJSONRequiredObject(required []string, fields map[string]*StrictJSONShape) *StrictJSONShape {
	return requiredObjectShape(required, fields)
}
func DecodeStrictJSON(payload []byte, shape *StrictJSONShape, target any) error {
	return decodeStrictBinding(payload, shape, target)
}

type DataHTTPServer interface {
	PublishResearchTheme(context.Context, *ResearchThemeImportRequest) (*Response[ResearchThemeImportResult], error)
	ListResearchThemes(context.Context, *ListResearchThemesRequest) (*Response[ResearchThemePage], error)
	GetResearchTheme(context.Context, *GetResearchThemeRequest) (*Response[ResearchThemeDetail], error)
	ListResearchReasoningTrees(context.Context, *ReasoningTreeListRequest) (*Response[ResearchReasoningTreeList], error)
	GetResearchReasoningTree(context.Context, *ReasoningTreeDetailRequest) (*Response[ResearchReasoningTreeDetail], error)
	ListResearchAnalysisContext(context.Context, *ResearchAnalysisContextRequest) (*Response[ResearchAnalysisContext], error)
	SearchResearchGraph(context.Context, *ResearchGraphSearchRequest) (*Response[ResearchGraphSearchResult], error)
	GetRuntimeHealth(context.Context, *RuntimeHealthRequest) (*Response[RuntimeHealth], error)
}

func RegisterDataHTTPServer(server *kratoshttp.Server, application DataHTTPServer) {
	router := server.Route(APIPrefix)
	router.POST("/research-theme-imports", researchThemeImportHandler(application))
	router.GET("/research/themes", listResearchThemesHandler(application))
	router.GET("/research/themes/{theme_id}", getResearchThemeHandler(application))
	router.GET("/research/themes/{theme_id}/reasoning-trees", listReasoningTreesHandler(application))
	router.GET("/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}", getReasoningTreeHandler(application))
	router.GET("/research-analysis-context", listResearchAnalysisContextHandler(application))
	router.POST("/research-graph:search", searchResearchGraphHandler(application))
	router.GET("/runtime-health", runtimeHealthHandler(application))
}

func runtimeHealthHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &RuntimeHealthRequest{}
		return Call(ctx, OperationGetRuntimeHealth, request, func(callContext context.Context) (*Response[RuntimeHealth], error) {
			return application.GetRuntimeHealth(callContext, request)
		})
	}
}

func searchResearchGraphHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := DecodeStrictJSONBody[ResearchGraphSearchRequest](ctx)
		if err != nil {
			return err
		}
		return Call(
			ctx,
			OperationSearchResearchGraph,
			request,
			func(callContext context.Context) (*Response[ResearchGraphSearchResult], error) {
				return application.SearchResearchGraph(callContext, request)
			},
		)
	}
}

func DecodeStrictJSONBody[T any](ctx kratoshttp.Context) (*T, error) {
	payload, err := io.ReadAll(io.LimitReader(ctx.Request().Body, MaxRequestBodySize+1))
	if err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	if len(payload) > MaxRequestBodySize {
		return nil, NewPublicError(StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1048576 bytes", nil)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var request T
	if err := decoder.Decode(&request); err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	return &request, nil
}

func listResearchAnalysisContextHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		query := ctx.Request().URL.Query()
		allowed := map[string]struct{}{
			"discovery_window_start": {}, "discovery_window_end": {},
			"analysis_as_of": {}, "prediction_horizon_start": {},
			"prediction_horizon_end": {}, "page_size": {}, "cursor": {},
		}
		for key, values := range query {
			if _, ok := allowed[key]; !ok || len(values) != 1 {
				return NewPublicError(
					StatusBadRequest,
					"INVALID_REQUEST",
					"Research Analysis Context query parameters are invalid",
					nil,
				)
			}
		}
		required := []string{
			"discovery_window_start", "discovery_window_end", "analysis_as_of", "page_size",
		}
		for _, key := range required {
			if len(query[key]) != 1 || strings.TrimSpace(query.Get(key)) == "" {
				return NewPublicError(
					StatusBadRequest,
					"INVALID_REQUEST",
					key+" is required",
					nil,
				)
			}
		}
		pageSize, err := ParseBoundedInt(query.Get("page_size"), 0, 1, 50, "page_size")
		if err != nil {
			return err
		}
		request := &ResearchAnalysisContextRequest{
			DiscoveryWindowStart:   query.Get("discovery_window_start"),
			DiscoveryWindowEnd:     query.Get("discovery_window_end"),
			AnalysisAsOf:           query.Get("analysis_as_of"),
			PredictionHorizonStart: optionalQueryValue(query, "prediction_horizon_start"),
			PredictionHorizonEnd:   optionalQueryValue(query, "prediction_horizon_end"),
			PageSize:               pageSize,
			Cursor:                 query.Get("cursor"),
		}
		return Call(
			ctx,
			OperationListResearchAnalysisContext,
			request,
			func(callContext context.Context) (*Response[ResearchAnalysisContext], error) {
				return application.ListResearchAnalysisContext(callContext, request)
			},
		)
	}
}

func optionalQueryValue(query map[string][]string, key string) *string {
	if len(query[key]) == 0 {
		return nil
	}
	value := query[key][0]
	return &value
}

func researchThemeImportHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := ReadImportPayload(ctx)
		if err != nil {
			return err
		}
		request, err := decodeResearchThemeImport(payload)
		if err != nil {
			return err
		}
		return Call(ctx, OperationPublishResearchTheme, request, func(callContext context.Context) (*Response[ResearchThemeImportResult], error) {
			return application.PublishResearchTheme(callContext, request)
		})
	}
}

func ReadImportPayload(ctx kratoshttp.Context) ([]byte, error) {
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
			WindowHours:   ctx.Query().Get("window_hours"),
			PublishedFrom: ctx.Query().Get("published_from"),
			PublishedTo:   ctx.Query().Get("published_to"),
			Limit:         ctx.Query().Get("limit"),
			Cursor:        ctx.Query().Get("cursor"),
		}
		return Call(ctx, OperationListResearchThemes, request, func(callContext context.Context) (*Response[ResearchThemePage], error) {
			return application.ListResearchThemes(callContext, request)
		})
	}
}

func getResearchThemeHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetResearchThemeRequest{ThemeID: ctx.Vars().Get("theme_id"), WindowHours: ctx.Query().Get("window_hours")}
		return Call(ctx, OperationGetResearchTheme, request, func(callContext context.Context) (*Response[ResearchThemeDetail], error) {
			return application.GetResearchTheme(callContext, request)
		})
	}
}

func listReasoningTreesHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ReasoningTreeListRequest{ThemeID: ctx.Vars().Get("theme_id"), HasQuery: ctx.Request().URL.RawQuery != ""}
		return Call(ctx, OperationListResearchThemeReasoningTrees, request, func(callContext context.Context) (*Response[ResearchReasoningTreeList], error) {
			return application.ListResearchReasoningTrees(callContext, request)
		})
	}
}

func getReasoningTreeHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ReasoningTreeDetailRequest{
			ThemeID: ctx.Vars().Get("theme_id"), ReasoningTreeID: ctx.Vars().Get("reasoning_tree_id"), HasQuery: ctx.Request().URL.RawQuery != "",
		}
		return Call(ctx, OperationGetResearchThemeReasoningTree, request, func(callContext context.Context) (*Response[ResearchReasoningTreeDetail], error) {
			return application.GetResearchReasoningTree(callContext, request)
		})
	}
}

func Call[T any](ctx kratoshttp.Context, operation string, request any, invoke func(context.Context) (*Response[T], error)) error {
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
