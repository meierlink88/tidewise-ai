package report

import (
	"context"
	"errors"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
	api "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1/report"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
)

type Service struct{ useCase *biz.UseCase }

func NewService(useCase *biz.UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Report use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) GetHome(ctx context.Context, request *api.HomeRequest) (*api.HomeResponse, error) {
	if s == nil || s.useCase == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.useCase.Home(ctx)
	if err != nil {
		return nil, publicError(err)
	}
	reports := make([]api.HomeReport, len(result.Reports))
	for index, item := range result.Reports {
		reports[index] = mapHome(item)
	}
	return &api.HomeResponse{Selection: api.Selection{Mode: result.Selection.Mode, Date: result.Selection.Date, Timezone: result.Selection.Timezone}, Reports: reports}, nil
}

func (s *Service) ListEvidences(ctx context.Context, request *api.EvidenceRequest) (*api.EvidenceCollection, error) {
	if s == nil || s.useCase == nil || request == nil || request.HasUnknownQuery {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.useCase.Evidences(ctx, request.ReportID, request.ScopeToken)
	if err != nil {
		return nil, publicError(err)
	}
	items := make([]api.EvidenceItem, len(result.Items))
	for index, item := range result.Items {
		var publishedAt *string
		if item.PublishedAt != nil {
			formatted := formatTime(*item.PublishedAt)
			publishedAt = &formatted
		}
		items[index] = api.EvidenceItem{PublishedAt: publishedAt, Summary: item.Summary, Keywords: cloneStrings(item.Keywords)}
		for _, tag := range item.SemanticTags {
			items[index].SemanticTags = append(items[index].SemanticTags, api.EvidenceTag{Kind: tag.Kind, Text: tag.Text})
		}
	}
	return &api.EvidenceCollection{ReportID: result.ReportID, ScopeToken: result.ScopeToken, Items: items}, nil
}

func mapHome(value biz.Home) api.HomeReport {
	groups := make([]api.AnalysisGroup, len(value.AnalysisGroups))
	for i, g := range value.AnalysisGroups {
		groups[i] = mapAnalysisGroup(g)
	}
	return api.HomeReport{AnalysisGroups: groups, Report: mapSummary(value.Report), Cards: []api.Card{}, NextCursor: value.NextCursor}
}

func publicError(err error) error {
	switch {
	case errors.Is(err, biz.ErrInvalidRequest):
		return v1.ErrInvalidRequest
	case errors.Is(err, biz.ErrReportNotFound):
		return v1.ErrReportNotFound
	case errors.Is(err, biz.ErrLayerNotFound):
		return v1.ErrReportLayerNotFound
	case errors.Is(err, biz.ErrChainNotFound):
		return v1.ErrReportIndustryChainNotFound
	case errors.Is(err, biz.ErrEvidenceScopeNotFound):
		return v1.ErrReportEvidenceScopeNotFound
	default:
		return v1.ErrReportServiceUnavailable
	}
}
func mapSummary(value biz.Summary) api.Summary {
	return api.Summary{SchemaVersion: value.SchemaVersion, ID: value.ID, GeneratedAt: formatTime(value.GeneratedAt), PublishedAt: formatTime(value.PublishedAt), IndustryChainCount: value.IndustryChainCount}
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func cloneStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}

var _ api.Service = (*Service)(nil)

func analysisQuery(q *api.AnalysisQuery) biz.AnalysisQuery {
	return biz.AnalysisQuery{ReportID: q.ReportID, Kind: q.Kind, Key: q.Key, ChainKey: q.ChainKey, Limit: q.Limit, Cursor: q.Cursor}
}

func (s *Service) ListAnalyses(ctx context.Context, q *api.AnalysisQuery) (*api.AnalysisPage, error) {
	if s == nil || s.useCase == nil || q == nil {
		return nil, v1.ErrInvalidRequest
	}
	v, err := s.useCase.Analyses(ctx, analysisQuery(q))
	if err != nil {
		return nil, publicError(err)
	}
	out := mapAnalysisPage(v)
	return &out, nil
}
func (s *Service) GetAnalysis(ctx context.Context, q *api.AnalysisQuery) (*api.NormalizedDetailProjection, error) {
	if s == nil || s.useCase == nil || q == nil {
		return nil, v1.ErrInvalidRequest
	}
	v, err := s.useCase.Analysis(ctx, analysisQuery(q))
	if err != nil {
		return nil, publicError(err)
	}
	out := mapNormalizedDetailProjection(v)
	return &out, nil
}
func (s *Service) GetAnalysisChain(ctx context.Context, q *api.AnalysisQuery) (*api.NormalizedChain, error) {
	if s == nil || s.useCase == nil || q == nil {
		return nil, v1.ErrInvalidRequest
	}
	v, err := s.useCase.AnalysisChain(ctx, analysisQuery(q))
	if err != nil {
		return nil, publicError(err)
	}
	out := mapNormalizedChain(v)
	return &out, nil
}

func mapNormalizedClaim(v biz.NormalizedClaim) api.NormalizedClaim {
	out := api.NormalizedClaim{}
	out.Text = v.Text
	out.Basis = v.Basis
	out.EvidenceScopeToken = v.EvidenceScopeToken
	out.EvidenceCount = v.EvidenceCount
	return out
}

func mapNormalizedObjections(v biz.NormalizedObjections) api.NormalizedObjections {
	out := api.NormalizedObjections{}
	out.Summary = v.Summary
	out.Counterevidence = make([]api.NormalizedClaim, len(v.Counterevidence))
	for i, x := range v.Counterevidence {
		out.Counterevidence[i] = mapNormalizedClaim(x)
	}
	out.Buffers = make([]api.NormalizedClaim, len(v.Buffers))
	for i, x := range v.Buffers {
		out.Buffers[i] = mapNormalizedClaim(x)
	}
	out.CounterevidenceStatus = v.CounterevidenceStatus
	out.EvidenceGaps = v.EvidenceGaps
	out.ScopeLimits = v.ScopeLimits
	return out
}

func mapNormalizedWindow(v biz.NormalizedWindow) api.NormalizedWindow {
	out := api.NormalizedWindow{}
	out.Kind = v.Kind
	out.Description = v.Description
	out.StartAt = v.StartAt
	out.EndAt = v.EndAt
	return out
}

func mapNormalizedAssessment(v biz.NormalizedAssessment) api.NormalizedAssessment {
	out := api.NormalizedAssessment{}
	out.Conclusion = v.Conclusion
	out.Direction = v.Direction
	out.ConclusionBasis = v.ConclusionBasis
	out.ValidationStatus = v.ValidationStatus
	out.Confidence = v.Confidence
	out.ForecastWindow = mapNormalizedWindow(v.ForecastWindow)
	out.Scope = v.Scope
	out.Conditions = v.Conditions
	out.FollowUp = v.FollowUp
	out.TransmissionLogic = v.TransmissionLogic
	out.EvidenceScopeToken = v.EvidenceScopeToken
	out.EvidenceCount = v.EvidenceCount
	return out
}

func mapNormalizedNode(v biz.NormalizedNode) api.NormalizedNode {
	out := api.NormalizedNode{}
	out.JudgmentOrigin = v.JudgmentOrigin
	out.ReasoningSources = mapNormalizedReasoningSources(v.ReasoningSources)
	out.VariableSignals = mapNormalizedSignals(v.VariableSignals)
	out.LocalKey = v.LocalKey
	out.SourceID = v.SourceID
	out.NodeLocalKey = v.NodeLocalKey
	out.Name = v.Name
	out.Assessment = mapNormalizedAssessment(v.Assessment)
	out.Objections = mapNormalizedObjections(v.Objections)
	return out
}

func mapNormalizedGraph(v biz.NormalizedGraph) api.NormalizedGraph {
	out := api.NormalizedGraph{}
	out.Scope = v.Scope
	out.Nodes = make([]api.NormalizedGraphNodesItem, len(v.Nodes))
	for i, x := range v.Nodes {
		out.Nodes[i] = mapNormalizedGraphNodesItem(x)
	}
	out.Edges = make([]api.NormalizedGraphEdgesItem, len(v.Edges))
	for i, x := range v.Edges {
		out.Edges[i] = mapNormalizedGraphEdgesItem(x)
	}
	return out
}

func mapNormalizedChain(v biz.NormalizedChain) api.NormalizedChain {
	out := api.NormalizedChain{}
	out.JudgmentOrigin = v.JudgmentOrigin
	out.ReasoningSources = mapNormalizedReasoningSources(v.ReasoningSources)
	out.VariableSignals = mapNormalizedSignals(v.VariableSignals)
	out.LocalKey = v.LocalKey
	out.SourceID = v.SourceID
	out.Name = v.Name
	out.Assessment = mapNormalizedAssessment(v.Assessment)
	out.ReasoningSummary = mapNormalizedChainReasoningSummary(v.ReasoningSummary)
	out.Graph = mapNormalizedGraph(v.Graph)
	out.AffectedNodes = make([]api.NormalizedNode, len(v.AffectedNodes))
	for i, x := range v.AffectedNodes {
		out.AffectedNodes[i] = mapNormalizedNode(x)
	}
	if v.EmptyState != nil {
		x := mapNormalizedChainEmptyState(*v.EmptyState)
		out.EmptyState = &x
	}
	return out
}

func mapNormalizedMacro(v biz.NormalizedMacro) api.NormalizedMacro {
	out := api.NormalizedMacro{}
	out.JudgmentOrigin = v.JudgmentOrigin
	out.ReasoningSources = mapNormalizedReasoningSources(v.ReasoningSources)
	out.VariableSignals = mapNormalizedSignals(v.VariableSignals)
	out.LocalKey = v.LocalKey
	out.SourceID = v.SourceID
	out.Name = v.Name
	out.Assessment = mapNormalizedAssessment(v.Assessment)
	out.Objections = mapNormalizedObjections(v.Objections)
	return out
}

func mapNormalizedAnchorRef(v biz.NormalizedAnchorRef) api.NormalizedAnchorRef {
	out := api.NormalizedAnchorRef{}
	out.TargetType = v.TargetType
	out.LocalKey = v.LocalKey
	out.ChainLocalKey = v.ChainLocalKey
	return out
}

func mapNormalizedGraphNodesItem(v biz.NormalizedGraphNodesItem) api.NormalizedGraphNodesItem {
	out := api.NormalizedGraphNodesItem{}
	out.LocalKey = v.LocalKey
	out.SourceID = v.SourceID
	out.Name = v.Name
	return out
}

func mapNormalizedGraphEdgesItem(v biz.NormalizedGraphEdgesItem) api.NormalizedGraphEdgesItem {
	out := api.NormalizedGraphEdgesItem{}
	out.FromNodeLocalKey = v.FromNodeLocalKey
	out.ToNodeLocalKey = v.ToNodeLocalKey
	out.RelationLabel = v.RelationLabel
	return out
}

func mapNormalizedChainReasoningSummary(v biz.NormalizedChainReasoningSummary) api.NormalizedChainReasoningSummary {
	out := api.NormalizedChainReasoningSummary{}
	out.Logic = v.Logic
	out.Support = mapNormalizedClaim(v.Support)
	out.Objections = mapNormalizedObjections(v.Objections)
	return out
}

func mapNormalizedChainEmptyState(v biz.NormalizedChainEmptyState) api.NormalizedChainEmptyState {
	out := api.NormalizedChainEmptyState{}
	out.Code = v.Code
	out.Reason = v.Reason
	out.FollowUp = v.FollowUp
	return out
}

func mapNormalizedUnitSummary(v biz.NormalizedUnitSummary) api.NormalizedUnitSummary {
	out := api.NormalizedUnitSummary{}
	out.Conclusion = v.Conclusion
	out.TransmissionLogic = v.TransmissionLogic
	out.ImpactAssessment = mapNormalizedUnitSummaryImpactAssessment(v.ImpactAssessment)
	out.AffectedRefs = make([]api.NormalizedAnchorRef, len(v.AffectedRefs))
	for i, x := range v.AffectedRefs {
		out.AffectedRefs[i] = mapNormalizedAnchorRef(x)
	}
	out.EvidenceScopeToken = v.EvidenceScopeToken
	out.EvidenceCount = v.EvidenceCount
	return out
}

func mapNormalizedUnitSummaryImpactAssessment(v biz.NormalizedUnitSummaryImpactAssessment) api.NormalizedUnitSummaryImpactAssessment {
	out := api.NormalizedUnitSummaryImpactAssessment{}
	out.Level = v.Level
	out.Rationale = v.Rationale
	out.EvidenceScopeToken = v.EvidenceScopeToken
	out.EvidenceCount = v.EvidenceCount
	return out
}

func mapNormalizedResolvedAnchor(v biz.NormalizedResolvedAnchor) api.NormalizedResolvedAnchor {
	out := api.NormalizedResolvedAnchor{}
	out.JudgmentOrigin = v.JudgmentOrigin
	out.Reference = mapNormalizedAnchorRef(v.Reference)
	out.SourceID = v.SourceID
	out.Name = v.Name
	out.Assessment = mapNormalizedAssessment(v.Assessment)
	return out
}

func mapNormalizedSummaryProjection(v biz.NormalizedSummaryProjection) api.NormalizedSummaryProjection {
	out := api.NormalizedSummaryProjection{}
	out.JudgmentOrigin = v.JudgmentOrigin
	out.SchemaVersion = v.SchemaVersion
	out.LocalKey = v.LocalKey
	out.SourceID = v.SourceID
	out.Title = v.Title
	out.Summary = mapNormalizedUnitSummary(v.Summary)
	out.AffectedAnchors = make([]api.NormalizedResolvedAnchor, len(v.AffectedAnchors))
	for i, x := range v.AffectedAnchors {
		out.AffectedAnchors[i] = mapNormalizedResolvedAnchor(x)
	}
	out.ChainCount = v.ChainCount
	return out
}

func mapNormalizedChainHeader(v biz.NormalizedChainHeader) api.NormalizedChainHeader {
	out := api.NormalizedChainHeader{}
	out.JudgmentOrigin = v.JudgmentOrigin
	out.LocalKey = v.LocalKey
	out.SourceID = v.SourceID
	out.Name = v.Name
	out.Assessment = mapNormalizedAssessment(v.Assessment)
	if v.EmptyState != nil {
		x := mapNormalizedChainEmptyState(*v.EmptyState)
		out.EmptyState = &x
	}
	return out
}

func mapNormalizedDetailProjection(v biz.NormalizedDetailProjection) api.NormalizedDetailProjection {
	out := api.NormalizedDetailProjection{}
	if v.PublishedAt != nil {
		formatted := formatTime(*v.PublishedAt)
		out.PublishedAt = &formatted
	}
	out.JudgmentOrigin = v.JudgmentOrigin
	out.ReasoningSources = mapNormalizedReasoningSources(v.ReasoningSources)
	out.VariableSignals = mapNormalizedSignals(v.VariableSignals)
	if v.Companies != nil {
		values := make([]api.NormalizedMacro, len(*v.Companies))
		for i, value := range *v.Companies {
			values[i] = mapNormalizedMacro(value)
		}
		out.Companies = &values
	}
	out.Summary = mapNormalizedSummaryProjection(v.Summary)
	out.MacroImpacts = make([]api.NormalizedMacro, len(v.MacroImpacts))
	for i, x := range v.MacroImpacts {
		out.MacroImpacts[i] = mapNormalizedMacro(x)
	}
	out.IndustryChains = make([]api.NormalizedChainHeader, len(v.IndustryChains))
	for i, x := range v.IndustryChains {
		out.IndustryChains[i] = mapNormalizedChainHeader(x)
	}
	return out
}

func mapAnalysisPage(v biz.AnalysisPage) api.AnalysisPage {
	out := api.AnalysisPage{}
	out.Items = make([]api.NormalizedSummaryProjection, len(v.Items))
	for i, x := range v.Items {
		out.Items[i] = mapNormalizedSummaryProjection(x)
	}
	out.NextCursor = v.NextCursor
	return out
}

func mapAnalysisGroup(v biz.AnalysisGroup) api.AnalysisGroup {
	out := api.AnalysisGroup{}
	out.Kind = v.Kind
	out.Items = make([]api.NormalizedSummaryProjection, len(v.Items))
	for i, x := range v.Items {
		out.Items[i] = mapNormalizedSummaryProjection(x)
	}
	out.NextCursor = v.NextCursor
	return out
}

func mapNormalizedReasoningSources(v *biz.NormalizedReasoningSources) *api.NormalizedReasoningSources {
	if v == nil {
		return nil
	}
	refs := make([]api.NormalizedUpstreamRef, len(v.UpstreamRefs))
	for i, r := range v.UpstreamRefs {
		refs[i] = api.NormalizedUpstreamRef{EntityID: r.EntityID, LocalKey: r.LocalKey, Mechanism: r.Mechanism, Condition: r.Condition}
	}
	return &api.NormalizedReasoningSources{SignalIDs: v.SignalIDs, EventIDs: v.EventIDs, UpstreamRefs: refs}
}
func mapNormalizedSignals(v *[]biz.NormalizedSignal) *[]api.NormalizedSignal {
	if v == nil {
		return nil
	}
	out := make([]api.NormalizedSignal, len(*v))
	for i, s := range *v {
		out[i] = api.NormalizedSignal{VariableID: s.VariableID, VariableName: s.VariableName, SignalID: s.SignalID, Signal: s.Signal, SourceDirection: s.SourceDirection, Adoption: s.Adoption, Qualification: s.Qualification, EventIDs: s.EventIDs, EvidenceScopeToken: s.EvidenceScopeToken, EvidenceCount: s.EvidenceCount}
	}
	return &out
}
