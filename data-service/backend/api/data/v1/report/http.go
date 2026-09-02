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
		if err := v1.DecodeStrictJSON(payload, publicationV2Shape(), request); err != nil {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest,
				"request body is not valid for report-publication.v2", map[string]any{
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
		valid := validQuery(ctx, []string{"scope_type", "scope_key"}, nil)
		request := &EvidenceRequest{ReportID: ctx.Vars().Get("report_id"),
			ScopeType: ctx.Query().Get("scope_type"), ScopeKey: ctx.Query().Get("scope_key"),
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

func publicationV2Shape() *v1.StrictJSONShape {
	text, integer := v1.StrictJSONString(), v1.StrictJSONInteger()
	result := requiredShape(map[string]*v1.StrictJSONShape{"code": text, "label": text})
	nature := requiredShape(map[string]*v1.StrictJSONShape{"code": text, "label": text})
	confidence := requiredShape(map[string]*v1.StrictJSONShape{"code": text, "label": text, "score": v1.StrictJSONNullable(v1.StrictJSONNumber())})
	timeWindow := requiredShape(map[string]*v1.StrictJSONShape{"horizons": v1.StrictJSONArray(text), "lag": v1.StrictJSONNullableString(), "label": text})
	effect := requiredShape(map[string]*v1.StrictJSONShape{"display_order": integer, "dimension": text, "direction": text, "confidence": text})
	evidenceRef := requiredShape(map[string]*v1.StrictJSONShape{"evidence_id": text, "role": text, "display_order": integer})
	evidenceRefs := v1.StrictJSONArray(evidenceRef)
	claim := requiredShape(map[string]*v1.StrictJSONShape{"key": text, "text": text})
	checkpoint := requiredShape(map[string]*v1.StrictJSONShape{"key": text, "display_order": integer, "summary": text})
	uncertainty := requiredShape(map[string]*v1.StrictJSONShape{
		"counterevidence": text, "evidence_gap": v1.StrictJSONNullableString(),
		"boundary": text, "reversal_condition": text,
		"checkpoints": v1.StrictJSONArray(checkpoint),
	})
	anchor := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "name": text, "effects": v1.StrictJSONArray(effect),
		"result": result, "nature": nature, "reasoning": text, "time_window": timeWindow,
		"confidence": confidence, "source_ref": v1.StrictJSONNullableString(), "evidence_refs": evidenceRefs,
	})
	step := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "input": text, "mechanism": text, "output": text,
		"type": text, "confidence": confidence, "evidence_refs": evidenceRefs,
	})
	targetRef := requiredShape(map[string]*v1.StrictJSONShape{"type": text, "key": text})
	namedResult := requiredShape(map[string]*v1.StrictJSONShape{"name": text, "result": result})
	targetFields := map[string]*v1.StrictJSONShape{"ref": targetRef, "label": text, "results": v1.StrictJSONArray(namedResult)}
	target := v1.StrictJSONRequiredObject([]string{"label", "results"}, targetFields)
	transmission := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "source_claim_key": text, "source_conclusion": text,
		"targets": v1.StrictJSONArray(target), "logic": text, "relation_nature": text,
		"confidence": confidence, "status": text, "evidence_refs": evidenceRefs,
	})
	layerSummary := requiredShape(map[string]*v1.StrictJSONShape{"claim": claim, "transmissions": v1.StrictJSONArray(transmission), "uncertainty": uncertainty, "evidence_refs": evidenceRefs})
	layerDetail := requiredShape(map[string]*v1.StrictJSONShape{"anchors": v1.StrictJSONArray(anchor), "reasoning_steps": v1.StrictJSONArray(step), "related_chain_keys": v1.StrictJSONArray(text)})
	layer := requiredShape(map[string]*v1.StrictJSONShape{"key": text, "title": text, "summary": layerSummary, "detail": layerDetail})
	topologyNode := requiredShape(map[string]*v1.StrictJSONShape{"key": text, "display_order": integer, "name": text})
	impact := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "node_key": text, "effects": v1.StrictJSONArray(effect),
		"result": result, "nature": nature, "reasoning": text, "time_window": timeWindow,
		"confidence": confidence, "evidence_refs": evidenceRefs,
	})
	edge := requiredShape(map[string]*v1.StrictJSONShape{"key": text, "display_order": integer, "from_node_key": text, "to_node_key": text, "relation_label": text})
	chainGraph := requiredShape(map[string]*v1.StrictJSONShape{"nodes": v1.StrictJSONArray(topologyNode), "edges": v1.StrictJSONArray(edge)})
	chainUncertainty := requiredShape(map[string]*v1.StrictJSONShape{"counterevidence_and_gap": text, "stop_condition": text})
	chainSummary := requiredShape(map[string]*v1.StrictJSONShape{
		"claim": claim, "status": text, "result": result, "confidence": confidence, "time_window": timeWindow,
		"path": text, "accepted_hypothesis_summary": v1.StrictJSONNullableString(), "graph": chainGraph,
		"uncertainty": chainUncertainty, "evidence_refs": evidenceRefs,
	})
	chainDetail := requiredShape(map[string]*v1.StrictJSONShape{"node_impacts": v1.StrictJSONArray(impact)})
	chain := requiredShape(map[string]*v1.StrictJSONShape{"key": text, "display_order": integer, "name": text, "summary": chainSummary, "detail": chainDetail})
	statistics := requiredShape(map[string]*v1.StrictJSONShape{
		"event_count": integer, "ordinary_fact_count": integer, "signal_fact_count": integer,
		"transmission_hypothesis_count": integer, "geopolitic_anchor_count": integer,
		"macroeconomic_anchor_count": integer, "signaled_chain_node_count": integer, "industry_chain_count": integer,
	})
	analysisWindow := requiredShape(map[string]*v1.StrictJSONShape{"started_at": text, "ended_at": text})
	template := requiredShape(map[string]*v1.StrictJSONShape{"name": text, "version": text, "role": text})
	provenance := requiredShape(map[string]*v1.StrictJSONShape{
		"derived_from_report_id": v1.StrictJSONNullableString(), "frozen_source_sha256": v1.StrictJSONNullableString(),
		"frozen_source_commit": v1.StrictJSONNullableString(), "template": template,
	})
	contentFields := map[string]*v1.StrictJSONShape{
		"report_type": text, "title": text, "generation_status": text, "simulation": v1.StrictJSONBoolean(),
		"generated_at": text, "analysis_window": analysisWindow, "timezone": text, "provenance": provenance,
		"statistics": statistics, "geopolitics": layer, "macroeconomics": layer, "industry_chains": v1.StrictJSONArray(chain),
	}
	content := v1.StrictJSONRequiredObject([]string{"report_type", "title", "generation_status", "simulation", "generated_at", "analysis_window", "timezone", "provenance", "statistics", "industry_chains"}, contentFields)
	return requiredShape(map[string]*v1.StrictJSONShape{"contract_version": text, "publisher_report_id": text, "content": content})
}
