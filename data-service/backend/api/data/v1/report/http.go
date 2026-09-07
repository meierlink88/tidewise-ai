package report

import (
	"context"
	"encoding/json"
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
	router.GET("/reports/{report_id}/analyses/{kind}", analysisListHandler(application))
	router.GET("/reports/{report_id}/analyses/{kind}/{analysis_key}", analysisHandler(application))
	router.GET("/reports/{report_id}/analyses/{kind}/{analysis_key}/industry-chains/{chain_key}", analysisChainHandler(application))
	router.GET("/reports/{report_id}/concept-analyses/{concept_key}/industry-chains/{chain_key}", analysisChainHandler(application))
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
		shape := publicationShape()
		var versionProbe struct {
			Report struct {
				SchemaVersion json.RawMessage `json:"schema_version"`
			} `json:"report"`
		}
		if json.Unmarshal(payload, &versionProbe) == nil && versionProbe.Report.SchemaVersion != nil {
			shape = analysisPublicationShape()
			var version string
			if json.Unmarshal(versionProbe.Report.SchemaVersion, &version) == nil && version == NormalizedSchemaVersion {
				shape = normalizedPublicationShape()
			}
			if json.Unmarshal(versionProbe.Report.SchemaVersion, &version) == nil && version == SignalSchemaVersion {
				shape = requiredShape(map[string]*v1.StrictJSONShape{"publisher_report_id": v1.StrictJSONString(), "report": v5ReportShape()})
			}
		}
		if err := v1.DecodeStrictJSON(payload, shape, request); err != nil {
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
		if !validQuery(ctx, nil, []string{"published_from", "published_to", "limit", "cursor", "schema_version"}) {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "unsupported Report query parameter", nil)
		}
		query := ctx.Query()
		request := &ListRequest{SchemaVersion: query.Get("schema_version"), PublishedFrom: query.Get("published_from"), PublishedTo: query.Get("published_to"),
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

func analysisListHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if !validQuery(ctx, nil, []string{"limit", "cursor"}) {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "unsupported analysis query", nil)
		}
		r := &AnalysisRequest{ReportID: ctx.Vars().Get("report_id"), Kind: ctx.Vars().Get("kind"), Limit: ctx.Query().Get("limit"), Cursor: ctx.Query().Get("cursor")}
		return callWithBudget(ctx, OperationListReportAnalyses, readBudget, r, func(c context.Context) (*v1.Response[AnalysisCollection], error) {
			return application.ListReportAnalyses(c, r)
		})
	}
}
func analysisHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "analysis detail accepts no query parameters", nil)
		}
		r := &AnalysisRequest{ReportID: ctx.Vars().Get("report_id"), Kind: ctx.Vars().Get("kind"), AnalysisKey: ctx.Vars().Get("analysis_key")}
		return callWithBudget(ctx, OperationGetReportAnalysis, readBudget, r, func(c context.Context) (*v1.Response[AnalysisUnitDetail], error) {
			return application.GetReportAnalysis(c, r)
		})
	}
}
func analysisChainHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if len(ctx.Request().URL.Query()) != 0 {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "chain detail accepts no query parameters", nil)
		}
		operation := OperationGetReportAnalysisUnitChain
		kind, key := ctx.Vars().Get("kind"), ctx.Vars().Get("analysis_key")
		if kind == "" {
			operation = OperationGetReportAnalysisChain
			kind, key = "concept_analyses", ctx.Vars().Get("concept_key")
		}
		r := &AnalysisRequest{ReportID: ctx.Vars().Get("report_id"), Kind: kind, AnalysisKey: key, ChainKey: ctx.Vars().Get("chain_key")}
		return callWithBudget(ctx, operation, readBudget, r, func(c context.Context) (*v1.Response[ChainAnalysisDetail], error) {
			return application.GetReportAnalysisChain(c, r)
		})
	}
}

func analysisPublicationShape() *v1.StrictJSONShape {
	codedLabel := requiredShape(map[string]*v1.StrictJSONShape{"code": v1.StrictJSONString(), "label": v1.StrictJSONString()})
	analysisWindow := requiredShape(map[string]*v1.StrictJSONShape{"start": v1.StrictJSONString(), "end": v1.StrictJSONString()})
	evidenceReference := requiredShape(map[string]*v1.StrictJSONShape{"evidence_id": v1.StrictJSONString(), "role": codedLabel})
	impactAssessment := requiredShape(map[string]*v1.StrictJSONShape{"level": codedLabel, "rationale": v1.StrictJSONString(), "evidence_refs": v1.StrictJSONArray(evidenceReference)})
	analysisSummary := v1.StrictJSONRequiredObject([]string{"conclusion", "transmission_logic", "anchor_keys", "evidence_refs"}, map[string]*v1.StrictJSONShape{"impact_assessment": impactAssessment, "conclusion": v1.StrictJSONString(), "transmission_logic": v1.StrictJSONString(), "anchor_keys": v1.StrictJSONArray(v1.StrictJSONString()), "evidence_refs": v1.StrictJSONArray(evidenceReference)})
	reasoningStep := requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "input": v1.StrictJSONString(), "mechanism": v1.StrictJSONString(), "output": v1.StrictJSONString(), "confidence": codedLabel, "evidence_refs": v1.StrictJSONArray(evidenceReference)})
	analysisImpact := requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "target_type": codedLabel, "source_id": v1.StrictJSONString(), "node_local_key": v1.StrictJSONNullableString(), "name": v1.StrictJSONString(), "impact": v1.StrictJSONString(), "result": codedLabel, "conclusion_basis": codedLabel, "validation_status": codedLabel, "reasoning": v1.StrictJSONString(), "transmission_signal": v1.StrictJSONNullableString(), "conditions": v1.StrictJSONArray(v1.StrictJSONString()), "follow_up": v1.StrictJSONArray(v1.StrictJSONString()), "time_window": codedLabel, "confidence": codedLabel, "evidence_refs": v1.StrictJSONArray(evidenceReference)})
	layerUncertainty := requiredShape(map[string]*v1.StrictJSONShape{"counterevidence": v1.StrictJSONNullableString(), "evidence_gap": v1.StrictJSONNullableString(), "boundary": v1.StrictJSONNullableString(), "reversal_condition": v1.StrictJSONNullableString()})
	analysisTopologyNode := requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "name": v1.StrictJSONString()})
	industryChainEdge := requiredShape(map[string]*v1.StrictJSONShape{"from_node_local_key": v1.StrictJSONString(), "to_node_local_key": v1.StrictJSONString(), "relation_label": v1.StrictJSONString()})
	analysisGraph := requiredShape(map[string]*v1.StrictJSONShape{"nodes": v1.StrictJSONArray(analysisTopologyNode), "edges": v1.StrictJSONArray(industryChainEdge)})
	chainAnalysis := requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "name": v1.StrictJSONString(), "conclusion": v1.StrictJSONString(), "transmission_logic": v1.StrictJSONString(), "reasoning_steps": v1.StrictJSONArray(reasoningStep), "graph": analysisGraph, "affected_nodes": v1.StrictJSONArray(analysisImpact), "uncertainty": layerUncertainty, "evidence_refs": v1.StrictJSONArray(evidenceReference)})
	analysisDetail := requiredShape(map[string]*v1.StrictJSONShape{"reasoning_steps": v1.StrictJSONArray(reasoningStep), "affected_anchors": v1.StrictJSONArray(analysisImpact), "uncertainty": layerUncertainty, "industry_chains": v1.StrictJSONArray(chainAnalysis)})
	analysisUnit := requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "title": v1.StrictJSONString(), "summary": analysisSummary, "detail": analysisDetail})
	report := requiredShape(map[string]*v1.StrictJSONShape{"schema_version": v1.StrictJSONString(), "report_type": codedLabel, "generated_at": v1.StrictJSONString(), "timezone": v1.StrictJSONString(), "analysis_window": analysisWindow, "geopolitical_stories": v1.StrictJSONArray(analysisUnit), "macroeconomic_stories": v1.StrictJSONArray(analysisUnit), "concept_analyses": v1.StrictJSONArray(analysisUnit)})
	return requiredShape(map[string]*v1.StrictJSONShape{"publisher_report_id": v1.StrictJSONString(), "report": report})
}

func normalizedPublicationShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"publisher_report_id": v1.StrictJSONString(), "report": v4ReportShape()})
}
func v4CodedLabelShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"code": v1.StrictJSONString(), "label": v1.StrictJSONString()})
}
func v4ClaimShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"text": v1.StrictJSONString(), "basis": v1.StrictJSONString(), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())})
}
func v4ObjectionsShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"summary": v1.StrictJSONString(), "counterevidence": v1.StrictJSONArray(v4ClaimShape()), "buffers": v1.StrictJSONArray(v4ClaimShape()), "counterevidence_status": v1.StrictJSONString(), "evidence_gaps": v1.StrictJSONArray(v1.StrictJSONString()), "scope_limits": v1.StrictJSONArray(v1.StrictJSONString())})
}
func v4WindowShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"kind": v1.StrictJSONString(), "description": v1.StrictJSONString(), "start_at": v1.StrictJSONNullable(v1.StrictJSONString()), "end_at": v1.StrictJSONNullable(v1.StrictJSONString())})
}
func v4AssessmentShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"conclusion": v1.StrictJSONString(), "direction": v1.StrictJSONString(), "conclusion_basis": v1.StrictJSONString(), "validation_status": v1.StrictJSONString(), "confidence": v1.StrictJSONNullable(v1.StrictJSONString()), "forecast_window": v4WindowShape(), "scope": v1.StrictJSONString(), "conditions": v1.StrictJSONArray(v1.StrictJSONString()), "follow_up": v1.StrictJSONArray(v1.StrictJSONString()), "transmission_logic": v1.StrictJSONString(), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())})
}
func v4NodeShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "node_local_key": v1.StrictJSONString(), "name": v1.StrictJSONString(), "assessment": v4AssessmentShape(), "objections": v4ObjectionsShape()})
}
func v4GraphShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"nodes": v1.StrictJSONArray(v4GraphNodesItemShape()), "edges": v1.StrictJSONArray(v4GraphEdgesItemShape())})
}
func v4ChainShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "name": v1.StrictJSONString(), "assessment": v4AssessmentShape(), "reasoning_summary": v4ChainReasoningSummaryShape(), "graph": v4GraphShape(), "affected_nodes": v1.StrictJSONArray(v4NodeShape()), "empty_state": v1.StrictJSONNullable(v4ChainEmptyStateShape())})
}
func v4MacroShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "name": v1.StrictJSONString(), "assessment": v4AssessmentShape(), "objections": v4ObjectionsShape()})
}
func v4AnchorRefShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"target_type": v1.StrictJSONString(), "local_key": v1.StrictJSONString(), "chain_local_key": v1.StrictJSONNullable(v1.StrictJSONString())})
}
func v4UnitShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "title": v1.StrictJSONString(), "summary": v4UnitSummaryShape(), "detail": v4UnitDetailShape()})
}
func v4ReportShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"schema_version": v1.StrictJSONString(), "report_type": v4CodedLabelShape(), "generated_at": v1.StrictJSONString(), "timezone": v1.StrictJSONString(), "analysis_window": v4ReportAnalysisWindowShape(), "geopolitical_stories": v1.StrictJSONArray(v4UnitShape()), "macroeconomic_stories": v1.StrictJSONArray(v4UnitShape()), "concept_analyses": v1.StrictJSONArray(v4UnitShape()), "observations": v1.StrictJSONArray(v4ReportObservationsItemShape()), "limitations": v1.StrictJSONArray(v1.StrictJSONString())})
}
func v4GraphNodesItemShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "name": v1.StrictJSONString()})
}
func v4GraphEdgesItemShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"from_node_local_key": v1.StrictJSONString(), "to_node_local_key": v1.StrictJSONString(), "relation_label": v1.StrictJSONString()})
}
func v4ChainReasoningSummaryShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"logic": v1.StrictJSONString(), "support": v4ClaimShape(), "objections": v4ObjectionsShape()})
}
func v4ChainEmptyStateShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"code": v1.StrictJSONString(), "reason": v1.StrictJSONString(), "follow_up": v1.StrictJSONArray(v1.StrictJSONString())})
}
func v4UnitSummaryShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"conclusion": v1.StrictJSONString(), "transmission_logic": v1.StrictJSONString(), "impact_assessment": v4UnitSummaryImpactAssessmentShape(), "affected_refs": v1.StrictJSONArray(v4AnchorRefShape()), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())})
}
func v4UnitDetailShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"macro_impacts": v1.StrictJSONArray(v4MacroShape()), "industry_chains": v1.StrictJSONArray(v4ChainShape())})
}
func v4ReportAnalysisWindowShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"start": v1.StrictJSONString(), "end": v1.StrictJSONString()})
}
func v4ReportObservationsItemShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "title": v1.StrictJSONString(), "text": v1.StrictJSONString(), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())})
}
func v4UnitSummaryImpactAssessmentShape() *v1.StrictJSONShape {
	return requiredShape(map[string]*v1.StrictJSONShape{"level": v1.StrictJSONString(), "rationale": v1.StrictJSONString(), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())})
}

func v5CodedLabelShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"code", "label"}, map[string]*v1.StrictJSONShape{"code": v1.StrictJSONString(), "label": v1.StrictJSONString()})
}

func v5ClaimShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"text", "basis", "evidence_ids"}, map[string]*v1.StrictJSONShape{"text": v1.StrictJSONString(), "basis": v1.StrictJSONString(), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())})
}

func v5ObjectionsShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"summary", "counterevidence", "buffers", "counterevidence_status", "evidence_gaps", "scope_limits"}, map[string]*v1.StrictJSONShape{"summary": v1.StrictJSONString(), "counterevidence": v1.StrictJSONArray(v5ClaimShape()), "buffers": v1.StrictJSONArray(v5ClaimShape()), "counterevidence_status": v1.StrictJSONString(), "evidence_gaps": v1.StrictJSONArray(v1.StrictJSONString()), "scope_limits": v1.StrictJSONArray(v1.StrictJSONString())})
}

func v5WindowShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"kind", "description", "start_at", "end_at"}, map[string]*v1.StrictJSONShape{"kind": v1.StrictJSONString(), "description": v1.StrictJSONString(), "start_at": v1.StrictJSONNullable(v1.StrictJSONString()), "end_at": v1.StrictJSONNullable(v1.StrictJSONString())})
}

func v5AssessmentShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"conclusion", "direction", "conclusion_basis", "validation_status", "confidence", "forecast_window", "scope", "conditions", "follow_up", "transmission_logic", "evidence_ids"}, map[string]*v1.StrictJSONShape{"conclusion": v1.StrictJSONString(), "direction": v1.StrictJSONString(), "conclusion_basis": v1.StrictJSONString(), "validation_status": v1.StrictJSONString(), "confidence": v1.StrictJSONNullable(v1.StrictJSONString()), "forecast_window": v5WindowShape(), "scope": v1.StrictJSONString(), "conditions": v1.StrictJSONArray(v1.StrictJSONString()), "follow_up": v1.StrictJSONArray(v1.StrictJSONString()), "transmission_logic": v1.StrictJSONString(), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())})
}

func v5NodeShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"local_key", "source_id", "node_local_key", "name", "assessment", "objections", "judgment_origin", "reasoning_sources", "variable_signals"}, map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "node_local_key": v1.StrictJSONString(), "name": v1.StrictJSONString(), "assessment": v5AssessmentShape(), "objections": v5ObjectionsShape(), "judgment_origin": v1.StrictJSONString(), "reasoning_sources": v5ReasoningSourcesShape(), "variable_signals": v1.StrictJSONArray(v5SignalShape())})
}

func v5GraphShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"nodes", "edges", "scope"}, map[string]*v1.StrictJSONShape{"nodes": v1.StrictJSONArray(v1.StrictJSONRequiredObject([]string{"local_key", "source_id", "name"}, map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "name": v1.StrictJSONString()})), "edges": v1.StrictJSONArray(v1.StrictJSONRequiredObject([]string{"from_node_local_key", "to_node_local_key", "relation_label"}, map[string]*v1.StrictJSONShape{"from_node_local_key": v1.StrictJSONString(), "to_node_local_key": v1.StrictJSONString(), "relation_label": v1.StrictJSONString()})), "scope": v1.StrictJSONString()})
}

func v5ChainShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"local_key", "source_id", "name", "assessment", "reasoning_summary", "graph", "affected_nodes", "empty_state", "judgment_origin", "reasoning_sources", "variable_signals"}, map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "name": v1.StrictJSONString(), "assessment": v5AssessmentShape(), "reasoning_summary": v1.StrictJSONRequiredObject([]string{"logic", "support", "objections"}, map[string]*v1.StrictJSONShape{"logic": v1.StrictJSONString(), "support": v5ClaimShape(), "objections": v5ObjectionsShape()}), "graph": v5GraphShape(), "affected_nodes": v1.StrictJSONArray(v5NodeShape()), "empty_state": v1.StrictJSONNullable(v1.StrictJSONRequiredObject([]string{"code", "reason", "follow_up"}, map[string]*v1.StrictJSONShape{"code": v1.StrictJSONString(), "reason": v1.StrictJSONString(), "follow_up": v1.StrictJSONArray(v1.StrictJSONString())})), "judgment_origin": v1.StrictJSONString(), "reasoning_sources": v5ReasoningSourcesShape(), "variable_signals": v1.StrictJSONArray(v5SignalShape())})
}

func v5MacroShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"local_key", "source_id", "name", "assessment", "objections", "judgment_origin", "reasoning_sources", "variable_signals"}, map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "name": v1.StrictJSONString(), "assessment": v5AssessmentShape(), "objections": v5ObjectionsShape(), "judgment_origin": v1.StrictJSONString(), "reasoning_sources": v5ReasoningSourcesShape(), "variable_signals": v1.StrictJSONArray(v5SignalShape())})
}

func v5AnchorRefShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"target_type", "local_key", "chain_local_key"}, map[string]*v1.StrictJSONShape{"target_type": v1.StrictJSONString(), "local_key": v1.StrictJSONString(), "chain_local_key": v1.StrictJSONNullable(v1.StrictJSONString())})
}

func v5UnitShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"local_key", "source_id", "title", "summary", "detail", "judgment_origin", "reasoning_sources"}, map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "title": v1.StrictJSONString(), "summary": v1.StrictJSONRequiredObject([]string{"conclusion", "transmission_logic", "impact_assessment", "affected_refs", "evidence_ids"}, map[string]*v1.StrictJSONShape{"conclusion": v1.StrictJSONString(), "transmission_logic": v1.StrictJSONString(), "impact_assessment": v1.StrictJSONRequiredObject([]string{"level", "rationale", "evidence_ids"}, map[string]*v1.StrictJSONShape{"level": v1.StrictJSONString(), "rationale": v1.StrictJSONString(), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())}), "affected_refs": v1.StrictJSONArray(v5AnchorRefShape()), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())}), "detail": v1.StrictJSONRequiredObject([]string{"macro_impacts", "industry_chains", "variable_signals", "companies"}, map[string]*v1.StrictJSONShape{"macro_impacts": v1.StrictJSONArray(v5MacroShape()), "industry_chains": v1.StrictJSONArray(v5ChainShape()), "variable_signals": v1.StrictJSONArray(v5SignalShape()), "companies": v1.StrictJSONArray(v5CompanyShape())}), "judgment_origin": v1.StrictJSONString(), "reasoning_sources": v5ReasoningSourcesShape()})
}

func v5SignalShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"variable_id", "variable_name", "signal_id", "signal", "source_direction", "adoption", "qualification", "event_ids", "evidence_ids"}, map[string]*v1.StrictJSONShape{"variable_id": v1.StrictJSONString(), "variable_name": v1.StrictJSONString(), "signal_id": v1.StrictJSONString(), "signal": v1.StrictJSONString(), "source_direction": v1.StrictJSONString(), "adoption": v1.StrictJSONString(), "qualification": v1.StrictJSONString(), "event_ids": v1.StrictJSONArray(v1.StrictJSONString()), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())})
}

func v5ReasoningSourcesShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"signal_ids", "event_ids", "upstream_refs"}, map[string]*v1.StrictJSONShape{"signal_ids": v1.StrictJSONArray(v1.StrictJSONString()), "event_ids": v1.StrictJSONArray(v1.StrictJSONString()), "upstream_refs": v1.StrictJSONArray(v1.StrictJSONRequiredObject([]string{"entity_id", "local_key"}, map[string]*v1.StrictJSONShape{"entity_id": v1.StrictJSONString(), "local_key": v1.StrictJSONString(), "mechanism": v1.StrictJSONString(), "condition": v1.StrictJSONString()}))})
}

func v5CompanyShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"local_key", "source_id", "name", "assessment", "objections", "judgment_origin", "reasoning_sources", "variable_signals"}, map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "source_id": v1.StrictJSONString(), "name": v1.StrictJSONString(), "assessment": v5AssessmentShape(), "objections": v5ObjectionsShape(), "judgment_origin": v1.StrictJSONString(), "reasoning_sources": v5ReasoningSourcesShape(), "variable_signals": v1.StrictJSONArray(v5SignalShape())})
}

func v5ReportShape() *v1.StrictJSONShape {
	return v1.StrictJSONRequiredObject([]string{"schema_version", "report_type", "generated_at", "timezone", "analysis_window", "geopolitical_stories", "macroeconomic_stories", "concept_analyses", "observations", "limitations", "industry_chain_analyses", "company_analyses"}, map[string]*v1.StrictJSONShape{"schema_version": v1.StrictJSONString(), "report_type": v5CodedLabelShape(), "generated_at": v1.StrictJSONString(), "timezone": v1.StrictJSONString(), "analysis_window": v1.StrictJSONRequiredObject([]string{"start", "end"}, map[string]*v1.StrictJSONShape{"start": v1.StrictJSONString(), "end": v1.StrictJSONString()}), "geopolitical_stories": v1.StrictJSONArray(v5UnitShape()), "macroeconomic_stories": v1.StrictJSONArray(v5UnitShape()), "concept_analyses": v1.StrictJSONArray(v5UnitShape()), "observations": v1.StrictJSONArray(v1.StrictJSONRequiredObject([]string{"local_key", "title", "text", "evidence_ids"}, map[string]*v1.StrictJSONShape{"local_key": v1.StrictJSONString(), "title": v1.StrictJSONString(), "text": v1.StrictJSONString(), "evidence_ids": v1.StrictJSONArray(v1.StrictJSONString())})), "limitations": v1.StrictJSONArray(v1.StrictJSONString()), "industry_chain_analyses": v1.StrictJSONArray(v5UnitShape()), "company_analyses": v1.StrictJSONArray(v5CompanyShape())})
}
