package report

import (
	"context"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
)

const readBudget = 8 * time.Second

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	if application == nil {
		return
	}
	router := server.Route(v1.APIPrefix)
	router.GET("/reports/home", homeHandler(application))
	router.GET("/reports/{report_id}/layers/{layer_key}", layerHandler(application))
	router.GET("/reports/{report_id}/industry-chains/{chain_key}", industryChainHandler(application))
	router.GET("/reports/{report_id}/evidences", evidenceHandler(application))
}

func homeHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return v1.ErrInvalidRequest
		}
		return callWithBudget(ctx, OperationGetHome, &HomeRequest{}, func(callContext context.Context) (any, error) {
			return application.GetHome(callContext, &HomeRequest{})
		})
	}
}

func layerHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return v1.ErrInvalidRequest
		}
		request := &LayerRequest{ReportID: ctx.Vars().Get("report_id"), LayerKey: ctx.Vars().Get("layer_key")}
		return callWithBudget(ctx, OperationGetLayer, request, func(callContext context.Context) (any, error) {
			return application.GetLayer(callContext, request)
		})
	}
}

func industryChainHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return v1.ErrInvalidRequest
		}
		request := &IndustryChainRequest{ReportID: ctx.Vars().Get("report_id"), ChainKey: ctx.Vars().Get("chain_key")}
		return callWithBudget(ctx, OperationGetChain, request, func(callContext context.Context) (any, error) {
			return application.GetIndustryChain(callContext, request)
		})
	}
}

func evidenceHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		query := ctx.Request().URL.Query()
		request := &EvidenceRequest{
			ReportID:        ctx.Vars().Get("report_id"),
			HasUnknownQuery: hasUnknownQuery(query, "scope_type", "scope_key"),
		}
		if values := query["scope_type"]; len(values) == 1 {
			request.ScopeType = values[0]
		} else {
			request.HasUnknownQuery = true
		}
		if values := query["scope_key"]; len(values) == 1 {
			request.ScopeKey = values[0]
		} else {
			request.HasUnknownQuery = true
		}
		return callWithBudget(ctx, OperationListEvidences, request, func(callContext context.Context) (any, error) {
			return application.ListEvidences(callContext, request)
		})
	}
}

func callWithBudget(ctx kratoshttp.Context, operation string, request any, invoke func(context.Context) (any, error)) error {
	return v1.Call(ctx, operation, request, func(callContext context.Context) (any, error) {
		deadlineContext, cancel := context.WithTimeout(callContext, readBudget)
		defer cancel()
		return invoke(deadlineContext)
	})
}

func hasUnknownQuery(query map[string][]string, allowed ...string) bool {
	known := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		known[name] = struct{}{}
	}
	for name := range query {
		if _, exists := known[name]; !exists {
			return true
		}
	}
	return false
}
