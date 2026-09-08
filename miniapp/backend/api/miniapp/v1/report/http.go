package report

import (
	"context"
	"strconv"
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
	router.GET("/reports/{report_id}/analyses/{kind}", analysisHandler(application, "list"))
	router.GET("/reports/{report_id}/analyses/{kind}/{analysis_key}", analysisHandler(application, "unit"))
	router.GET("/reports/{report_id}/analyses/{kind}/{analysis_key}/industry-chains/{chain_key}", analysisHandler(application, "chain"))
	router.GET("/reports/home", homeHandler(application))
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

func evidenceHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		query := ctx.Request().URL.Query()
		request := &EvidenceRequest{
			ReportID:        ctx.Vars().Get("report_id"),
			HasUnknownQuery: hasUnknownQuery(query, "scope_token"),
		}
		if values := query["scope_token"]; len(values) == 1 {
			request.ScopeToken = values[0]
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

func analysisHandler(app Service, mode string) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		q := &AnalysisQuery{ReportID: ctx.Vars().Get("report_id"), Kind: ctx.Vars().Get("kind"), Key: ctx.Vars().Get("analysis_key"), ChainKey: ctx.Vars().Get("chain_key")}
		values := ctx.Request().URL.Query()
		if mode == "list" {
			if hasUnknownQuery(values, "limit", "cursor") || len(values["limit"]) > 1 || len(values["cursor"]) > 1 {
				return v1.ErrInvalidRequest
			}
			q.Cursor = values.Get("cursor")
			if values.Has("limit") {
				n, err := strconv.Atoi(values.Get("limit"))
				if err != nil {
					return v1.ErrInvalidRequest
				}
				q.Limit = n
			}
		} else if len(values) != 0 {
			return v1.ErrInvalidRequest
		}
		return callWithBudget(ctx, "miniapp.v1.reportAnalysis."+mode, q, func(c context.Context) (any, error) {
			switch mode {
			case "list":
				return app.ListAnalyses(c, q)
			case "unit":
				return app.GetAnalysis(c, q)
			default:
				return app.GetAnalysisChain(c, q)
			}
		})
	}
}
