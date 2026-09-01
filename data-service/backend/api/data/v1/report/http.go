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
				"request body is not valid for report-publication.v1", map[string]any{
					"path": v1.StrictJSONErrorPath(err),
				})
		}
		return callWithBudget(ctx, OperationPublishReport, publishBudget, request,
			func(callContext context.Context) (*v1.Response[PublicationResult], error) {
				return application.PublishReport(callContext, request)
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

func publicationShape() *v1.StrictJSONShape {
	text := v1.StrictJSONString()
	integer := v1.StrictJSONInteger()
	confidence := requiredShape(map[string]*v1.StrictJSONShape{
		"label": text, "score": v1.StrictJSONNullable(v1.StrictJSONNumber()),
	})
	result := requiredShape(map[string]*v1.StrictJSONShape{"code": text, "label": text})
	nature := requiredShape(map[string]*v1.StrictJSONShape{"code": text, "label": text})
	targetRef := requiredShape(map[string]*v1.StrictJSONShape{"type": text, "key": text})
	evidenceRef := requiredShape(map[string]*v1.StrictJSONShape{
		"evidence_id": text, "role": text, "display_order": integer,
	})
	evidenceRefs := v1.StrictJSONArray(evidenceRef)
	checkpoint := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "summary": text,
	})
	anchor := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "name": text, "current_state": text,
		"result": result, "nature": nature, "reasoning": text, "time_window": text,
		"confidence": confidence, "evidence_refs": evidenceRefs,
	})
	step := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "input": text, "mechanism": text,
		"output": text, "type": text, "confidence": confidence, "evidence_refs": evidenceRefs,
	})
	target := requiredShape(map[string]*v1.StrictJSONShape{
		"ref": targetRef, "label": text, "result": result,
	})
	path := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "source_conclusion": text,
		"target_refs": v1.StrictJSONArray(target), "logic": text, "relation_nature": text, "evidence_role": text,
		"confidence": confidence, "status": text, "evidence_refs": evidenceRefs,
	})
	candidate := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "mechanism": text,
		"evidence_gap": v1.StrictJSONNullableString(), "confidence": confidence, "evidence_refs": evidenceRefs,
	})
	downward := requiredShape(map[string]*v1.StrictJSONShape{
		"summary": text, "published_paths": v1.StrictJSONArray(path),
		"candidate_mechanisms": v1.StrictJSONArray(candidate), "boundary_notes": v1.StrictJSONArray(text),
	})
	layerUncertainty := requiredShape(map[string]*v1.StrictJSONShape{
		"counterevidence": v1.StrictJSONNullableString(), "evidence_gap": v1.StrictJSONNullableString(),
		"boundary": v1.StrictJSONNullableString(), "reversal_condition": v1.StrictJSONNullableString(),
		"checkpoints": v1.StrictJSONArray(checkpoint),
	})
	layer := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "title": text, "conclusion": text,
		"result": result, "confidence": confidence, "time_window": text,
		"anchors": v1.StrictJSONArray(anchor), "reasoning_steps": v1.StrictJSONArray(step),
		"related_anchor_keys": v1.StrictJSONArray(text), "related_chain_keys": v1.StrictJSONArray(text),
		"downward_transmission": downward, "uncertainty": layerUncertainty, "evidence_refs": evidenceRefs,
	})
	node := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "name": text, "impact": text,
		"result": result, "nature": nature, "reasoning": text, "time_window": text,
		"confidence": confidence, "evidence_refs": evidenceRefs,
	})
	edge := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "from_node_key": text,
		"to_node_key": text, "relation_label": text,
	})
	chainUncertainty := requiredShape(map[string]*v1.StrictJSONShape{
		"counterevidence_and_gap": v1.StrictJSONNullableString(),
		"stop_condition":          v1.StrictJSONNullableString(), "checkpoints": v1.StrictJSONArray(checkpoint),
	})
	chain := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "claim_key": text, "display_order": integer, "name": text,
		"conclusion": text, "status": text, "result": result, "confidence": confidence,
		"time_window": text, "path_summary": v1.StrictJSONNullableString(),
		"accepted_hypothesis_summary": v1.StrictJSONNullableString(), "evidence_refs": evidenceRefs,
		"nodes": v1.StrictJSONArray(node), "edges": v1.StrictJSONArray(edge), "uncertainty": chainUncertainty,
	})
	company := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "display_order": integer, "title": text,
		"published": v1.StrictJSONBoolean(), "boundary": text,
	})
	impactItem := requiredShape(map[string]*v1.StrictJSONShape{
		"ref": targetRef, "name": text, "result": result, "confidence": confidence, "time_window": text,
	})
	reportCard := requiredShape(map[string]*v1.StrictJSONShape{
		"key": text, "kind": text, "display_order": integer, "detail_ref": targetRef,
		"title": text, "subtitle": text, "conclusion": text, "result": result,
		"confidence": confidence, "time_window": text, "impact_items": v1.StrictJSONArray(impactItem),
		"evidence_refs": evidenceRefs,
	})
	statistics := requiredShape(map[string]*v1.StrictJSONShape{
		"event_count":                               integer,
		"ordinary_fact_count":                       integer,
		"signal_fact_count":                         integer,
		"transmission_hypothesis_count":             integer,
		"remaining_topology_pending_count":          integer,
		"adaptive_inclusion_threshold":              v1.StrictJSONNumber(),
		"adaptive_continuation_threshold":           v1.StrictJSONNumber(),
		"adaptive_hard_max_hops":                    integer,
		"adaptive_observed_max_hops":                integer,
		"adaptive_stopped_by_confidence":            integer,
		"adaptive_stopped_by_no_unvisited_neighbor": integer,
		"adaptive_rejected_below_inclusion":         integer,
		"geopolitic_anchor_count":                   integer,
		"macroeconomic_anchor_count":                integer,
		"signaled_chain_node_count":                 integer,
		"industry_chain_count":                      integer,
		"unmapped_chain_node_count":                 integer,
	})
	content := requiredShape(map[string]*v1.StrictJSONShape{
		"report_type": text, "title": text, "status": text,
		"simulation": v1.StrictJSONBoolean(), "generated_at": text, "timezone": text,
		"published_layers": v1.StrictJSONArray(text), "statistics": statistics,
		"report_cards": v1.StrictJSONArray(reportCard), "geopolitics": layer, "macroeconomics": layer,
		"industry_chains": v1.StrictJSONArray(chain), "company": company,
	})
	return requiredShape(map[string]*v1.StrictJSONShape{
		"contract_version": text, "source_report_id": text, "content": content,
	})
}

func requiredShape(fields map[string]*v1.StrictJSONShape) *v1.StrictJSONShape {
	required := make([]string, 0, len(fields))
	for name := range fields {
		required = append(required, name)
	}
	return v1.StrictJSONRequiredObject(required, fields)
}
