package report

import (
	"context"
	"net/url"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	readBudget    = 5 * time.Second
	publishBudget = 20 * time.Second
)

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.POST("/report-publications", publishHandler(application))
	router.GET("/reports", listHandler(application))
	router.GET("/reports/{report_id}/home", homeHandler(application))
	router.GET("/reports/{report_id}/layers/{layer_key}", layerHandler(application))
	router.GET("/reports/{report_id}/industry-chains", chainListHandler(application))
	router.GET("/reports/{report_id}/industry-chains/{chain_key}", chainHandler(application))
	router.GET("/reports/{report_id}/evidences", evidenceHandler(application))
}

func callWithBudget[T any](ctx kratoshttp.Context, operation string, budget time.Duration, request any, invoke func(context.Context) (*v1.Response[T], error)) error {
	return v1.Call(ctx, operation, request, func(callContext context.Context) (*v1.Response[T], error) {
		deadlineContext, cancel := context.WithTimeout(callContext, budget)
		defer cancel()
		return invoke(deadlineContext)
	})
}

func publishHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := v1.ReadImportPayload(ctx)
		if err != nil {
			return err
		}
		request := new(PublicationRequest)
		if err := v1.DecodeStrictJSON(payload, publicationShape(), request); err != nil {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest,
				"request body is not valid for Report publication", map[string]any{
					"path": v1.StrictJSONErrorPath(err),
				})
		}
		return callWithBudget(ctx, OperationPublishReport, publishBudget, request,
			func(callContext context.Context) (*v1.Response[PublicationResult], error) {
				return application.PublishReport(callContext, request)
			})
	}
}

func chainListHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		valid := validQuery(ctx, nil, []string{"limit", "cursor"})
		request := &ChainListRequest{ReportID: ctx.Vars().Get("report_id"), Limit: ctx.Query().Get("limit"), Cursor: ctx.Query().Get("cursor"), HasUnknownQuery: !valid}
		return callWithBudget(ctx, OperationListReportChains, readBudget, request,
			func(callContext context.Context) (*v1.Response[IndustryChainCollection], error) {
				return application.ListReportIndustryChains(callContext, request)
			})
	}
}

func listHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if !validQuery(ctx, nil, []string{"published_from", "published_to", "limit", "cursor"}) {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "unsupported Report query parameter", nil)
		}
		query := ctx.Query()
		request := &ListRequest{PublishedFrom: query.Get("published_from"), PublishedTo: query.Get("published_to"),
			Limit: query.Get("limit"), Cursor: query.Get("cursor")}
		return callWithBudget(ctx, OperationListReports, readBudget, request,
			func(callContext context.Context) (*v1.Response[Collection], error) {
				return application.ListReports(callContext, request)
			})
	}
}

func homeHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "Report home does not accept query parameters", nil)
		}
		request := &ReportRequest{ReportID: ctx.Vars().Get("report_id")}
		return callWithBudget(ctx, OperationGetReportHome, readBudget, request,
			func(callContext context.Context) (*v1.Response[Home], error) {
				return application.GetReportHome(callContext, request)
			})
	}
}

func layerHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "Report layer does not accept query parameters", nil)
		}
		request := &LayerRequest{ReportID: ctx.Vars().Get("report_id"), LayerKey: ctx.Vars().Get("layer_key")}
		return callWithBudget(ctx, OperationGetReportLayer, readBudget, request,
			func(callContext context.Context) (*v1.Response[LayerDetail], error) {
				return application.GetReportLayer(callContext, request)
			})
	}
}

func chainHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "Report industry chain does not accept query parameters", nil)
		}
		request := &ChainRequest{ReportID: ctx.Vars().Get("report_id"), ChainKey: ctx.Vars().Get("chain_key")}
		return callWithBudget(ctx, OperationGetReportChain, readBudget, request,
			func(callContext context.Context) (*v1.Response[IndustryChainDetail], error) {
				return application.GetReportIndustryChain(callContext, request)
			})
	}
}

func evidenceHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		valid := validQuery(ctx, []string{"scope_token"}, nil)
		request := &EvidenceRequest{ReportID: ctx.Vars().Get("report_id"),
			ScopeToken:      ctx.Query().Get("scope_token"),
			HasUnknownQuery: !valid}
		return callWithBudget(ctx, OperationListReportEvidence, readBudget, request,
			func(callContext context.Context) (*v1.Response[EvidenceCollection], error) {
				return application.ListReportEvidence(callContext, request)
			})
	}
}

func validQuery(ctx kratoshttp.Context, required, optional []string) bool {
	return validQueryValues(ctx.Request().URL.Query(), required, optional)
}

func validQueryValues(query url.Values, required, optional []string) bool {
	allowed := make(map[string]struct{}, len(required)+len(optional))
	for _, name := range required {
		allowed[name] = struct{}{}
	}
	for _, name := range optional {
		allowed[name] = struct{}{}
	}
	for name, values := range query {
		if _, ok := allowed[name]; !ok || len(values) != 1 {
			return false
		}
	}
	for _, name := range required {
		if len(query[name]) != 1 {
			return false
		}
	}
	return true
}

func requiredShape(fields map[string]*v1.StrictJSONShape) *v1.StrictJSONShape {
	required := make([]string, 0, len(fields))
	for name := range fields {
		required = append(required, name)
	}
	return v1.StrictJSONRequiredObject(required, fields)
}

func publicationShape() *v1.StrictJSONShape {
	text := v1.StrictJSONString()
	codedLabel := requiredShape(map[string]*v1.StrictJSONShape{"code": text, "label": text})
	confidence := codedLabel
	timeWindow := codedLabel
	evidenceRef := requiredShape(map[string]*v1.StrictJSONShape{"evidence_id": text, "role": codedLabel})
	evidenceRefs := v1.StrictJSONArray(evidenceRef)
	uncertainty := requiredShape(map[string]*v1.StrictJSONShape{
		"counterevidence": v1.StrictJSONNullableString(), "evidence_gap": v1.StrictJSONNullableString(),
		"boundary": v1.StrictJSONNullableString(), "reversal_condition": v1.StrictJSONNullableString(),
	})
	anchor := requiredShape(map[string]*v1.StrictJSONShape{
		"local_key": text, "name": text, "current_state": text, "result": codedLabel,
		"conclusion_basis": codedLabel, "validation_status": codedLabel, "reasoning": text,
		"time_window": timeWindow, "confidence": confidence, "evidence_refs": evidenceRefs,
	})
	step := requiredShape(map[string]*v1.StrictJSONShape{
		"local_key": text, "input": text, "mechanism": text, "output": text,
		"confidence": confidence, "evidence_refs": evidenceRefs,
	})
	target := requiredShape(map[string]*v1.StrictJSONShape{
		"target_type": codedLabel, "target_local_key": text, "target_name": text, "result": codedLabel,
	})
	transmission := requiredShape(map[string]*v1.StrictJSONShape{
		"local_key": text, "source_conclusion": text, "targets": v1.StrictJSONArray(target),
		"transmission_logic": text, "transmission_kind": codedLabel,
		"confidence": confidence, "status": codedLabel,
	})
	transmissionGroup := requiredShape(map[string]*v1.StrictJSONShape{"summary": text, "paths": v1.StrictJSONArray(transmission)})
	geoDownward := requiredShape(map[string]*v1.StrictJSONShape{
		"to_macroeconomics": transmissionGroup, "to_industry_chains": transmissionGroup,
	})
	macroDownward := requiredShape(map[string]*v1.StrictJSONShape{"to_industry_chains": transmissionGroup})
	layerFields := map[string]*v1.StrictJSONShape{
		"local_key": text, "title": text, "conclusion": text, "result": codedLabel,
		"time_window": timeWindow, "confidence": confidence,
		"affected_anchors": v1.StrictJSONArray(anchor), "reasoning_steps": v1.StrictJSONArray(step),
		"uncertainty": uncertainty, "evidence_refs": evidenceRefs,
	}
	geopoliticsFields := make(map[string]*v1.StrictJSONShape, len(layerFields)+1)
	macroeconomicsFields := make(map[string]*v1.StrictJSONShape, len(layerFields)+1)
	for name, shape := range layerFields {
		geopoliticsFields[name], macroeconomicsFields[name] = shape, shape
	}
	geopoliticsFields["downward_transmission"] = geoDownward
	macroeconomicsFields["downward_transmission"] = macroDownward
	geopolitics := requiredShape(geopoliticsFields)
	macroeconomics := requiredShape(macroeconomicsFields)
	impact := requiredShape(map[string]*v1.StrictJSONShape{
		"local_key": text, "name": text, "impact": text, "result": codedLabel,
		"conclusion_basis": codedLabel, "validation_status": codedLabel, "reasoning": text,
		"time_window": timeWindow, "confidence": confidence, "evidence_refs": evidenceRefs,
	})
	edge := requiredShape(map[string]*v1.StrictJSONShape{
		"from_node_local_key": text, "to_node_local_key": text, "relation_label": text,
	})
	chainUncertainty := requiredShape(map[string]*v1.StrictJSONShape{
		"counterevidence_and_gap": v1.StrictJSONNullableString(), "stop_condition": v1.StrictJSONNullableString(),
	})
	chain := requiredShape(map[string]*v1.StrictJSONShape{
		"local_key": text, "name": text, "conclusion": text, "result": codedLabel,
		"time_window": timeWindow, "confidence": confidence,
		"path_summary": v1.StrictJSONNullableString(), "accepted_hypothesis_summary": v1.StrictJSONNullableString(),
		"nodes": v1.StrictJSONArray(impact), "edges": v1.StrictJSONArray(edge),
		"uncertainty": chainUncertainty, "evidence_refs": evidenceRefs,
	})
	reportFields := map[string]*v1.StrictJSONShape{
		"report_type": codedLabel, "generated_at": text, "timezone": text,
		"geopolitics": geopolitics, "macroeconomics": macroeconomics,
		"industry_chains": v1.StrictJSONArray(chain),
	}
	report := v1.StrictJSONRequiredObject([]string{"report_type", "generated_at", "timezone", "industry_chains"}, reportFields)
	return requiredShape(map[string]*v1.StrictJSONShape{"publisher_report_id": text, "report": report})
}
