package v1

import (
	"context"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

func RegisterResearchHTTPServer(server *kratoshttp.Server, service ResearchHTTPServer) {
	router := server.Route(APIPrefix)
	router.GET("/research/themes", listResearchThemesHandler(service))
	router.GET("/research/themes/{theme_id}/reasoning-trees", listResearchThemeReasoningTreesHandler(service))
	router.GET("/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}", getResearchThemeReasoningTreeHandler(service))
	router.GET("/research/themes/{theme_id}", getResearchThemeHandler(service))
}

func listResearchThemesHandler(service ResearchHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListResearchThemesRequest{
			WindowHours: parseIntQuery(ctx, "window_hours"),
			Limit:       parseIntQuery(ctx, "limit"),
			Cursor:      ctx.Query().Get("cursor"),
		}
		return call(ctx, OperationListResearchThemes, request, func(callContext context.Context) (any, error) {
			return service.ListResearchThemes(callContext, request)
		})
	}
}

func getResearchThemeHandler(service ResearchHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetResearchThemeRequest{
			ThemeID:     ctx.Vars().Get("theme_id"),
			WindowHours: parseIntQuery(ctx, "window_hours"),
		}
		return call(ctx, OperationGetResearchTheme, request, func(callContext context.Context) (any, error) {
			return service.GetResearchTheme(callContext, request)
		})
	}
}

func listResearchThemeReasoningTreesHandler(service ResearchHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListResearchThemeReasoningTreesRequest{ThemeID: ctx.Vars().Get("theme_id")}
		return call(ctx, OperationListResearchThemeReasoningTrees, request, func(callContext context.Context) (any, error) {
			if ctx.Request().URL.RawQuery != "" {
				return nil, ErrInvalidRequest
			}
			return service.ListResearchThemeReasoningTrees(callContext, request)
		})
	}
}

func getResearchThemeReasoningTreeHandler(service ResearchHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetResearchThemeReasoningTreeRequest{
			ThemeID:         ctx.Vars().Get("theme_id"),
			ReasoningTreeID: ctx.Vars().Get("reasoning_tree_id"),
		}
		return call(ctx, OperationGetResearchThemeReasoningTree, request, func(callContext context.Context) (any, error) {
			if ctx.Request().URL.RawQuery != "" {
				return nil, ErrInvalidRequest
			}
			return service.GetResearchThemeReasoningTree(callContext, request)
		})
	}
}

func call(ctx kratoshttp.Context, operation string, request any, invoke func(context.Context) (any, error)) error {
	kratoshttp.SetOperation(ctx, operation)
	handler := ctx.Middleware(func(callContext context.Context, _ any) (any, error) {
		return invoke(callContext)
	})
	response, err := handler(ctx, request)
	if err != nil {
		return err
	}
	return ctx.Result(200, response)
}

func parseIntQuery(ctx kratoshttp.Context, key string) int {
	value := strings.TrimSpace(ctx.Query().Get(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return parsed
}
