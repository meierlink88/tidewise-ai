package report

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	reportapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/report"
	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
)

type UseCase interface {
	Publish(context.Context, string, string, reportbiz.Content) (reportbiz.PublicationResult, error)
	List(context.Context, reportbiz.ListRequest) (reportbiz.Page, error)
	GetHome(context.Context, string) (reportbiz.Home, error)
	GetLayer(context.Context, string, string) (reportbiz.Summary, reportbiz.Layer, []reportbiz.IndustryChainSummary, error)
	GetIndustryChain(context.Context, string, string) (reportbiz.Summary, reportbiz.IndustryChain, error)
	ListEvidence(context.Context, string, reportbiz.ScopeType, string) ([]reportbiz.Evidence, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Report use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) PublishReport(ctx context.Context, request *reportapi.PublicationRequest) (*v1.Response[reportapi.PublicationResult], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report publication is required")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, reportapi.ErrorDataServiceNotReady, "Report service is unavailable")
	}
	generatedAt, err := time.Parse(time.RFC3339Nano, request.Content.GeneratedAt)
	if err != nil {
		return nil, publicError(v1.StatusUnprocessableEntity, reportapi.ErrorInvalidRequest, "content.generated_at must be RFC3339")
	}
	result, err := s.useCase.Publish(ctx, request.ContractVersion, request.SourceReportID, toBizContent(request.Content, generatedAt))
	if err != nil {
		return nil, publicationError(err)
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	return &v1.Response[reportapi.PublicationResult]{Status: status, Result: reportapi.PublicationResult{
		ReportID: result.Record.ID, ContentHash: result.ContentHash,
		PublishedAt: result.Record.PublishedAt.UTC().Format(time.RFC3339Nano), Replayed: result.Replayed,
	}}, nil
}

func (s *Service) ListReports(ctx context.Context, request *reportapi.ListRequest) (*v1.Response[reportapi.Collection], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report query is required")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, reportapi.ErrorDataServiceNotReady, "Report service is unavailable")
	}
	limit := reportbiz.DefaultLimit
	if strings.TrimSpace(request.Limit) != "" {
		parsed, err := strconv.Atoi(request.Limit)
		if err != nil {
			return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "limit must be an integer")
		}
		limit = parsed
	}
	from, err := optionalUTC(request.PublishedFrom)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "published_from must be a UTC RFC3339 timestamp")
	}
	to, err := optionalUTC(request.PublishedTo)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "published_to must be a UTC RFC3339 timestamp")
	}
	page, err := s.useCase.List(ctx, reportbiz.ListRequest{PublishedFrom: from, PublishedTo: to, Limit: limit, Cursor: request.Cursor})
	if err != nil {
		var validation *reportbiz.ValidationError
		if errors.As(err, &validation) {
			return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, validation.Error())
		}
		return nil, publicError(v1.StatusInternalServerError, reportapi.ErrorReportRepositoryFailure, "Report query failed")
	}
	items := make([]reportapi.Summary, len(page.Items))
	for index, item := range page.Items {
		items[index] = summary(item)
	}
	return &v1.Response[reportapi.Collection]{Status: v1.StatusOK, Result: reportapi.Collection{Items: items, NextCursor: page.NextCursor}}, nil
}

func (s *Service) GetReportHome(ctx context.Context, request *reportapi.ReportRequest) (*v1.Response[reportapi.Home], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report identity is required")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, reportapi.ErrorDataServiceNotReady, "Report service is unavailable")
	}
	home, err := s.useCase.GetHome(ctx, request.ReportID)
	if err != nil {
		return nil, readError(err)
	}
	cards := make([]reportapi.ReportCardRead, len(home.ReportCards))
	for index, card := range home.ReportCards {
		cards[index] = reportCardRead(card, home.EvidenceCounts)
	}
	return &v1.Response[reportapi.Home]{Status: v1.StatusOK, Result: reportapi.Home{
		Report: summary(home.Report), IndustryChainCount: home.Report.Statistics.IndustryChainCount,
		ReportCards: cards, Company: apiCompany(home.Company),
	}}, nil
}

func (s *Service) GetReportLayer(ctx context.Context, request *reportapi.LayerRequest) (*v1.Response[reportapi.LayerDetail], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report layer identity is required")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, reportapi.ErrorDataServiceNotReady, "Report service is unavailable")
	}
	reportSummary, layer, related, err := s.useCase.GetLayer(ctx, request.ReportID, request.LayerKey)
	if err != nil {
		return nil, readError(err)
	}
	chains := make([]reportapi.IndustryChainSummary, len(related))
	for index, chain := range related {
		chains[index] = industryChainSummary(chain)
	}
	return &v1.Response[reportapi.LayerDetail]{Status: v1.StatusOK, Result: reportapi.LayerDetail{
		Report: summary(reportSummary), Layer: layerRead(layer), RelatedIndustryChains: chains,
	}}, nil
}

func (s *Service) GetReportIndustryChain(ctx context.Context, request *reportapi.ChainRequest) (*v1.Response[reportapi.IndustryChainDetail], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report industry chain identity is required")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, reportapi.ErrorDataServiceNotReady, "Report service is unavailable")
	}
	reportSummary, chain, err := s.useCase.GetIndustryChain(ctx, request.ReportID, request.ChainKey)
	if err != nil {
		return nil, readError(err)
	}
	return &v1.Response[reportapi.IndustryChainDetail]{Status: v1.StatusOK, Result: reportapi.IndustryChainDetail{
		Report: summary(reportSummary), IndustryChain: industryChainRead(chain),
	}}, nil
}

func (s *Service) ListReportEvidence(ctx context.Context, request *reportapi.EvidenceRequest) (*v1.Response[reportapi.EvidenceCollection], error) {
	if request == nil || request.HasUnknownQuery || strings.TrimSpace(request.ScopeType) == "" || strings.TrimSpace(request.ScopeKey) == "" {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "scope_type and scope_key are required")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, reportapi.ErrorDataServiceNotReady, "Report service is unavailable")
	}
	items, err := s.useCase.ListEvidence(ctx, request.ReportID, reportbiz.ScopeType(request.ScopeType), request.ScopeKey)
	if err != nil {
		return nil, readError(err)
	}
	wireItems := make([]reportapi.EvidenceItem, len(items))
	for index, item := range items {
		var publishedAt *string
		if item.PublishedAt != nil {
			formatted := item.PublishedAt.UTC().Format(time.RFC3339Nano)
			publishedAt = &formatted
		}
		wireItems[index] = reportapi.EvidenceItem{EvidenceID: item.EvidenceID, Role: item.Role,
			DisplayOrder: item.DisplayOrder, PublishedAt: publishedAt, Summary: item.Summary, Keywords: item.Keywords}
	}
	return &v1.Response[reportapi.EvidenceCollection]{Status: v1.StatusOK, Result: reportapi.EvidenceCollection{
		ReportID: request.ReportID, ScopeType: request.ScopeType, ScopeKey: request.ScopeKey, Items: wireItems,
	}}, nil
}

func publicationError(err error) error {
	if errors.Is(err, reportbiz.ErrPublicationConflict) {
		return publicError(v1.StatusConflict, reportapi.ErrorReportPublicationConflict, "source_report_id conflicts with another Report payload")
	}
	var validation *reportbiz.ValidationError
	if errors.As(err, &validation) {
		return publicError(v1.StatusUnprocessableEntity, reportapi.ErrorInvalidRequest, validation.Error())
	}
	var reference *reportbiz.ReferenceError
	if errors.As(err, &reference) {
		return publicError(v1.StatusUnprocessableEntity, reportapi.ErrorReportEvidenceReferenceInvalid, "Report publication contains an invalid reference")
	}
	return publicError(v1.StatusInternalServerError, reportapi.ErrorReportRepositoryFailure, "Report publication failed")
}

func readError(err error) error {
	var validation *reportbiz.ValidationError
	switch {
	case errors.As(err, &validation):
		return publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, validation.Error())
	case errors.Is(err, reportbiz.ErrReportNotFound):
		return publicError(v1.StatusNotFound, reportapi.ErrorReportNotFound, "Report was not found")
	case errors.Is(err, reportbiz.ErrLayerNotFound):
		return publicError(v1.StatusNotFound, reportapi.ErrorReportLayerNotFound, "Report layer was not found")
	case errors.Is(err, reportbiz.ErrChainNotFound):
		return publicError(v1.StatusNotFound, reportapi.ErrorReportIndustryChainNotFound, "Report industry chain was not found")
	case errors.Is(err, reportbiz.ErrEvidenceScopeNotFound):
		return publicError(v1.StatusNotFound, reportapi.ErrorReportEvidenceScopeNotFound, "Report Evidence scope was not found")
	default:
		return publicError(v1.StatusInternalServerError, reportapi.ErrorReportRepositoryFailure, "Report query failed")
	}
}

func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}

func optionalUTC(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil || parsed.Location() != time.UTC {
		return nil, errors.New("timestamp must be UTC RFC3339")
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func toBizContent(value reportapi.Content, generatedAt time.Time) reportbiz.Content {
	chains := make([]reportbiz.IndustryChain, len(value.IndustryChains))
	for index, chain := range value.IndustryChains {
		chains[index] = toBizIndustryChain(chain)
	}
	cards := make([]reportbiz.ReportCard, len(value.ReportCards))
	for index, card := range value.ReportCards {
		cards[index] = toBizReportCard(card)
	}
	return reportbiz.Content{
		ReportType: value.ReportType, Title: value.Title, Status: value.Status, Simulation: value.Simulation,
		GeneratedAt: generatedAt, Timezone: value.Timezone, PublishedLayers: value.PublishedLayers,
		Statistics: toBizStatistics(value.Statistics), ReportCards: cards,
		Geopolitics: toBizLayer(value.Geopolitics), Macroeconomics: toBizLayer(value.Macroeconomics),
		IndustryChains: chains, Company: reportbiz.CompanyBoundary{Key: value.Company.Key,
			DisplayOrder: value.Company.DisplayOrder, Title: value.Company.Title,
			Published: value.Company.Published, Boundary: value.Company.Boundary},
	}
}

func toBizReportCard(value reportapi.ReportCard) reportbiz.ReportCard {
	impacts := make([]reportbiz.ImpactItem, len(value.ImpactItems))
	for index, item := range value.ImpactItems {
		impacts[index] = reportbiz.ImpactItem{Ref: toBizTargetRef(item.Ref), Name: item.Name,
			Result: toBizResult(item.Result), Confidence: toBizConfidence(item.Confidence), TimeWindow: item.TimeWindow}
	}
	return reportbiz.ReportCard{Key: value.Key, Kind: reportbiz.CardKind(value.Kind), DisplayOrder: value.DisplayOrder,
		DetailRef: toBizTargetRef(value.DetailRef), Title: value.Title, Subtitle: value.Subtitle,
		Conclusion: value.Conclusion, Result: toBizResult(value.Result), Confidence: toBizConfidence(value.Confidence),
		TimeWindow: value.TimeWindow, ImpactItems: impacts, EvidenceRefs: toBizEvidenceRefs(value.EvidenceRefs)}
}

func toBizLayer(value reportapi.Layer) reportbiz.Layer {
	anchors := make([]reportbiz.Anchor, len(value.Anchors))
	for index, anchor := range value.Anchors {
		anchors[index] = reportbiz.Anchor{Key: anchor.Key, DisplayOrder: anchor.DisplayOrder, Name: anchor.Name,
			CurrentState: anchor.CurrentState, Result: toBizResult(anchor.Result), Nature: toBizNature(anchor.Nature),
			Reasoning: anchor.Reasoning, TimeWindow: anchor.TimeWindow, Confidence: toBizConfidence(anchor.Confidence),
			EvidenceRefs: toBizEvidenceRefs(anchor.EvidenceRefs)}
	}
	steps := make([]reportbiz.ReasoningStep, len(value.ReasoningSteps))
	for index, step := range value.ReasoningSteps {
		steps[index] = reportbiz.ReasoningStep{Key: step.Key, DisplayOrder: step.DisplayOrder, Input: step.Input,
			Mechanism: step.Mechanism, Output: step.Output, Type: step.Type, Confidence: toBizConfidence(step.Confidence),
			EvidenceRefs: toBizEvidenceRefs(step.EvidenceRefs)}
	}
	paths := make([]reportbiz.TransmissionPath, len(value.DownwardTransmission.PublishedPaths))
	for index, path := range value.DownwardTransmission.PublishedPaths {
		targets := make([]reportbiz.TransmissionTarget, len(path.TargetRefs))
		for targetIndex, target := range path.TargetRefs {
			targets[targetIndex] = reportbiz.TransmissionTarget{Ref: toBizTargetRef(target.Ref), Label: target.Label, Result: toBizResult(target.Result)}
		}
		paths[index] = reportbiz.TransmissionPath{Key: path.Key, DisplayOrder: path.DisplayOrder,
			SourceConclusion: path.SourceConclusion, TargetRefs: targets, Logic: path.Logic,
			RelationNature: path.RelationNature, EvidenceRole: path.EvidenceRole,
			Confidence: toBizConfidence(path.Confidence), Status: path.Status, EvidenceRefs: toBizEvidenceRefs(path.EvidenceRefs)}
	}
	candidates := make([]reportbiz.CandidateMechanism, len(value.DownwardTransmission.CandidateMechanisms))
	for index, candidate := range value.DownwardTransmission.CandidateMechanisms {
		candidates[index] = reportbiz.CandidateMechanism{Key: candidate.Key, DisplayOrder: candidate.DisplayOrder,
			Mechanism: candidate.Mechanism, EvidenceGap: candidate.EvidenceGap,
			Confidence: toBizConfidence(candidate.Confidence), EvidenceRefs: toBizEvidenceRefs(candidate.EvidenceRefs)}
	}
	return reportbiz.Layer{Key: value.Key, DisplayOrder: value.DisplayOrder, Title: value.Title,
		Conclusion: value.Conclusion, Result: toBizResult(value.Result), Confidence: toBizConfidence(value.Confidence),
		TimeWindow: value.TimeWindow, Anchors: anchors, ReasoningSteps: steps,
		RelatedAnchorKeys: value.RelatedAnchorKeys, RelatedChainKeys: value.RelatedChainKeys,
		DownwardTransmission: reportbiz.DownwardTransmission{Summary: value.DownwardTransmission.Summary,
			PublishedPaths: paths, CandidateMechanisms: candidates, BoundaryNotes: value.DownwardTransmission.BoundaryNotes},
		Uncertainty: toBizLayerUncertainty(value.Uncertainty), EvidenceRefs: toBizEvidenceRefs(value.EvidenceRefs)}
}

func toBizIndustryChain(value reportapi.IndustryChain) reportbiz.IndustryChain {
	nodes := make([]reportbiz.IndustryChainNode, len(value.Nodes))
	for index, node := range value.Nodes {
		nodes[index] = reportbiz.IndustryChainNode{Key: node.Key, DisplayOrder: node.DisplayOrder, Name: node.Name,
			Impact: node.Impact, Result: toBizResult(node.Result), Nature: toBizNature(node.Nature), Reasoning: node.Reasoning,
			TimeWindow: node.TimeWindow, Confidence: toBizConfidence(node.Confidence), EvidenceRefs: toBizEvidenceRefs(node.EvidenceRefs)}
	}
	edges := make([]reportbiz.IndustryChainEdge, len(value.Edges))
	for index, edge := range value.Edges {
		edges[index] = reportbiz.IndustryChainEdge{Key: edge.Key, DisplayOrder: edge.DisplayOrder,
			FromNodeKey: edge.FromNodeKey, ToNodeKey: edge.ToNodeKey, RelationLabel: edge.RelationLabel}
	}
	return reportbiz.IndustryChain{Key: value.Key, ClaimKey: value.ClaimKey, DisplayOrder: value.DisplayOrder,
		Name: value.Name, Conclusion: value.Conclusion, Status: value.Status, Result: toBizResult(value.Result),
		Confidence: toBizConfidence(value.Confidence), TimeWindow: value.TimeWindow, PathSummary: value.PathSummary,
		AcceptedHypothesisSummary: value.AcceptedHypothesisSummary, EvidenceRefs: toBizEvidenceRefs(value.EvidenceRefs),
		Nodes: nodes, Edges: edges, Uncertainty: reportbiz.ChainUncertainty{
			CounterevidenceAndGap: value.Uncertainty.CounterevidenceAndGap,
			StopCondition:         value.Uncertainty.StopCondition, Checkpoints: toBizCheckpoints(value.Uncertainty.Checkpoints)},
	}
}

func toBizStatistics(value reportapi.Statistics) reportbiz.Statistics {
	return reportbiz.Statistics{
		EventCount: value.EventCount, OrdinaryFactCount: value.OrdinaryFactCount, SignalFactCount: value.SignalFactCount,
		TransmissionHypothesisCount:   value.TransmissionHypothesisCount,
		RemainingTopologyPendingCount: value.RemainingTopologyPendingCount,
		AdaptiveInclusionThreshold:    value.AdaptiveInclusionThreshold,
		AdaptiveContinuationThreshold: value.AdaptiveContinuationThreshold,
		AdaptiveHardMaxHops:           value.AdaptiveHardMaxHops, AdaptiveObservedMaxHops: value.AdaptiveObservedMaxHops,
		AdaptiveStoppedByConfidence:          value.AdaptiveStoppedByConfidence,
		AdaptiveStoppedByNoUnvisitedNeighbor: value.AdaptiveStoppedByNoUnvisitedNeighbor,
		AdaptiveRejectedBelowInclusion:       value.AdaptiveRejectedBelowInclusion,
		GeopoliticAnchorCount:                value.GeopoliticAnchorCount, MacroeconomicAnchorCount: value.MacroeconomicAnchorCount,
		SignaledChainNodeCount: value.SignaledChainNodeCount, IndustryChainCount: value.IndustryChainCount,
		UnmappedChainNodeCount: value.UnmappedChainNodeCount,
	}
}

func toBizLayerUncertainty(value reportapi.LayerUncertainty) reportbiz.LayerUncertainty {
	return reportbiz.LayerUncertainty{Counterevidence: value.Counterevidence, EvidenceGap: value.EvidenceGap,
		Boundary: value.Boundary, ReversalCondition: value.ReversalCondition, Checkpoints: toBizCheckpoints(value.Checkpoints)}
}

func toBizCheckpoints(values []reportapi.Checkpoint) []reportbiz.Checkpoint {
	result := make([]reportbiz.Checkpoint, len(values))
	for index, value := range values {
		result[index] = reportbiz.Checkpoint{Key: value.Key, DisplayOrder: value.DisplayOrder, Summary: value.Summary}
	}
	return result
}

func toBizEvidenceRefs(values []reportapi.EvidenceReference) []reportbiz.EvidenceReference {
	result := make([]reportbiz.EvidenceReference, len(values))
	for index, value := range values {
		result[index] = reportbiz.EvidenceReference{EvidenceID: value.EvidenceID, Role: value.Role, DisplayOrder: value.DisplayOrder}
	}
	return result
}

func toBizTargetRef(value reportapi.TargetReference) reportbiz.TargetReference {
	return reportbiz.TargetReference{Type: reportbiz.TargetType(value.Type), Key: value.Key}
}

func toBizResult(value reportapi.Result) reportbiz.Result {
	return reportbiz.Result{Code: reportbiz.ResultCode(value.Code), Label: value.Label}
}

func toBizNature(value reportapi.Nature) reportbiz.Nature {
	return reportbiz.Nature{Code: reportbiz.NatureCode(value.Code), Label: value.Label}
}

func toBizConfidence(value reportapi.Confidence) reportbiz.Confidence {
	return reportbiz.Confidence{Label: value.Label, Score: value.Score}
}

func summary(value reportbiz.Summary) reportapi.Summary {
	return reportapi.Summary{ID: value.ID, SourceReportID: value.SourceReportID, ReportType: value.ReportType,
		Title: value.Title, Status: value.Status, Simulation: value.Simulation,
		GeneratedAt: value.GeneratedAt.UTC().Format(time.RFC3339Nano), Timezone: value.Timezone,
		PublishedLayers: value.PublishedLayers, Statistics: apiStatistics(value.Statistics),
		PublishedAt: value.PublishedAt.UTC().Format(time.RFC3339Nano)}
}

func reportCardRead(value reportbiz.ReportCard, evidenceCounts map[reportbiz.TargetReference]int) reportapi.ReportCardRead {
	impacts := make([]reportapi.ImpactItemRead, len(value.ImpactItems))
	for index, item := range value.ImpactItems {
		impacts[index] = reportapi.ImpactItemRead{Ref: apiTargetRef(item.Ref), Name: item.Name,
			Result: apiResult(item.Result), Confidence: apiConfidence(item.Confidence), TimeWindow: item.TimeWindow,
			EvidenceCount: evidenceCounts[item.Ref]}
	}
	return reportapi.ReportCardRead{Key: value.Key, Kind: string(value.Kind), DisplayOrder: value.DisplayOrder,
		DetailRef: apiTargetRef(value.DetailRef), Title: value.Title, Subtitle: value.Subtitle,
		Conclusion: value.Conclusion, Result: apiResult(value.Result), Confidence: apiConfidence(value.Confidence),
		TimeWindow: value.TimeWindow, ImpactItems: impacts, EvidenceCount: len(value.EvidenceRefs)}
}

func layerRead(value reportbiz.Layer) reportapi.LayerRead {
	anchors := make([]reportapi.AnchorRead, len(value.Anchors))
	for index, anchor := range value.Anchors {
		anchors[index] = reportapi.AnchorRead{Key: anchor.Key, DisplayOrder: anchor.DisplayOrder, Name: anchor.Name,
			CurrentState: anchor.CurrentState, Result: apiResult(anchor.Result), Nature: apiNature(anchor.Nature),
			Reasoning: anchor.Reasoning, TimeWindow: anchor.TimeWindow, Confidence: apiConfidence(anchor.Confidence),
			EvidenceCount: len(anchor.EvidenceRefs)}
	}
	steps := make([]reportapi.ReasoningStepRead, len(value.ReasoningSteps))
	for index, step := range value.ReasoningSteps {
		steps[index] = reportapi.ReasoningStepRead{Key: step.Key, DisplayOrder: step.DisplayOrder, Input: step.Input,
			Mechanism: step.Mechanism, Output: step.Output, Type: step.Type,
			Confidence: apiConfidence(step.Confidence), EvidenceCount: len(step.EvidenceRefs)}
	}
	paths := make([]reportapi.TransmissionPathRead, len(value.DownwardTransmission.PublishedPaths))
	for index, path := range value.DownwardTransmission.PublishedPaths {
		targets := make([]reportapi.TransmissionTarget, len(path.TargetRefs))
		for targetIndex, target := range path.TargetRefs {
			targets[targetIndex] = reportapi.TransmissionTarget{Ref: apiTargetRef(target.Ref), Label: target.Label, Result: apiResult(target.Result)}
		}
		paths[index] = reportapi.TransmissionPathRead{Key: path.Key, DisplayOrder: path.DisplayOrder,
			SourceConclusion: path.SourceConclusion, TargetRefs: targets, Logic: path.Logic,
			RelationNature: path.RelationNature, EvidenceRole: path.EvidenceRole,
			Confidence: apiConfidence(path.Confidence), Status: path.Status, EvidenceCount: len(path.EvidenceRefs)}
	}
	candidates := make([]reportapi.CandidateMechanismRead, len(value.DownwardTransmission.CandidateMechanisms))
	for index, candidate := range value.DownwardTransmission.CandidateMechanisms {
		candidates[index] = reportapi.CandidateMechanismRead{Key: candidate.Key, DisplayOrder: candidate.DisplayOrder,
			Mechanism: candidate.Mechanism, EvidenceGap: candidate.EvidenceGap,
			Confidence: apiConfidence(candidate.Confidence), EvidenceCount: len(candidate.EvidenceRefs)}
	}
	return reportapi.LayerRead{Key: value.Key, DisplayOrder: value.DisplayOrder, Title: value.Title,
		Conclusion: value.Conclusion, Result: apiResult(value.Result), Confidence: apiConfidence(value.Confidence),
		TimeWindow: value.TimeWindow, Anchors: anchors, ReasoningSteps: steps,
		RelatedAnchorKeys: value.RelatedAnchorKeys, RelatedChainKeys: value.RelatedChainKeys,
		DownwardTransmission: reportapi.DownwardTransmissionRead{Summary: value.DownwardTransmission.Summary,
			PublishedPaths: paths, CandidateMechanisms: candidates, BoundaryNotes: value.DownwardTransmission.BoundaryNotes},
		Uncertainty: apiLayerUncertainty(value.Uncertainty), EvidenceCount: len(value.EvidenceRefs)}
}

func industryChainSummary(value reportbiz.IndustryChainSummary) reportapi.IndustryChainSummary {
	return reportapi.IndustryChainSummary{Key: value.Key, DisplayOrder: value.DisplayOrder, Name: value.Name,
		Conclusion: value.Conclusion, Status: value.Status, Result: apiResult(value.Result),
		Confidence: apiConfidence(value.Confidence), TimeWindow: value.TimeWindow, EvidenceCount: value.EvidenceCount}
}

func industryChainRead(value reportbiz.IndustryChain) reportapi.IndustryChainRead {
	nodes := make([]reportapi.IndustryChainNodeRead, len(value.Nodes))
	for index, node := range value.Nodes {
		nodes[index] = reportapi.IndustryChainNodeRead{Key: node.Key, DisplayOrder: node.DisplayOrder, Name: node.Name,
			Impact: node.Impact, Result: apiResult(node.Result), Nature: apiNature(node.Nature), Reasoning: node.Reasoning,
			TimeWindow: node.TimeWindow, Confidence: apiConfidence(node.Confidence), EvidenceCount: len(node.EvidenceRefs)}
	}
	edges := make([]reportapi.IndustryChainEdge, len(value.Edges))
	for index, edge := range value.Edges {
		edges[index] = reportapi.IndustryChainEdge{Key: edge.Key, DisplayOrder: edge.DisplayOrder,
			FromNodeKey: edge.FromNodeKey, ToNodeKey: edge.ToNodeKey, RelationLabel: edge.RelationLabel}
	}
	return reportapi.IndustryChainRead{Key: value.Key, ClaimKey: value.ClaimKey, DisplayOrder: value.DisplayOrder,
		Name: value.Name, Conclusion: value.Conclusion, Status: value.Status, Result: apiResult(value.Result),
		Confidence: apiConfidence(value.Confidence), TimeWindow: value.TimeWindow, PathSummary: value.PathSummary,
		AcceptedHypothesisSummary: value.AcceptedHypothesisSummary, Nodes: nodes, Edges: edges,
		Uncertainty: reportapi.ChainUncertainty{CounterevidenceAndGap: value.Uncertainty.CounterevidenceAndGap,
			StopCondition: value.Uncertainty.StopCondition, Checkpoints: apiCheckpoints(value.Uncertainty.Checkpoints)},
		EvidenceCount: len(value.EvidenceRefs)}
}

func apiCompany(value reportbiz.CompanyBoundary) reportapi.CompanyBoundary {
	return reportapi.CompanyBoundary{Key: value.Key, DisplayOrder: value.DisplayOrder,
		Title: value.Title, Published: value.Published, Boundary: value.Boundary}
}

func apiStatistics(value reportbiz.Statistics) reportapi.Statistics {
	return reportapi.Statistics{
		EventCount: value.EventCount, OrdinaryFactCount: value.OrdinaryFactCount, SignalFactCount: value.SignalFactCount,
		TransmissionHypothesisCount:   value.TransmissionHypothesisCount,
		RemainingTopologyPendingCount: value.RemainingTopologyPendingCount,
		AdaptiveInclusionThreshold:    value.AdaptiveInclusionThreshold,
		AdaptiveContinuationThreshold: value.AdaptiveContinuationThreshold,
		AdaptiveHardMaxHops:           value.AdaptiveHardMaxHops, AdaptiveObservedMaxHops: value.AdaptiveObservedMaxHops,
		AdaptiveStoppedByConfidence:          value.AdaptiveStoppedByConfidence,
		AdaptiveStoppedByNoUnvisitedNeighbor: value.AdaptiveStoppedByNoUnvisitedNeighbor,
		AdaptiveRejectedBelowInclusion:       value.AdaptiveRejectedBelowInclusion,
		GeopoliticAnchorCount:                value.GeopoliticAnchorCount, MacroeconomicAnchorCount: value.MacroeconomicAnchorCount,
		SignaledChainNodeCount: value.SignaledChainNodeCount, IndustryChainCount: value.IndustryChainCount,
		UnmappedChainNodeCount: value.UnmappedChainNodeCount,
	}
}

func apiLayerUncertainty(value reportbiz.LayerUncertainty) reportapi.LayerUncertainty {
	return reportapi.LayerUncertainty{Counterevidence: value.Counterevidence, EvidenceGap: value.EvidenceGap,
		Boundary: value.Boundary, ReversalCondition: value.ReversalCondition, Checkpoints: apiCheckpoints(value.Checkpoints)}
}

func apiCheckpoints(values []reportbiz.Checkpoint) []reportapi.Checkpoint {
	result := make([]reportapi.Checkpoint, len(values))
	for index, value := range values {
		result[index] = reportapi.Checkpoint{Key: value.Key, DisplayOrder: value.DisplayOrder, Summary: value.Summary}
	}
	return result
}

func apiTargetRef(value reportbiz.TargetReference) reportapi.TargetReference {
	return reportapi.TargetReference{Type: string(value.Type), Key: value.Key}
}

func apiResult(value reportbiz.Result) reportapi.Result {
	return reportapi.Result{Code: string(value.Code), Label: value.Label}
}

func apiNature(value reportbiz.Nature) reportapi.Nature {
	return reportapi.Nature{Code: string(value.Code), Label: value.Label}
}

func apiConfidence(value reportbiz.Confidence) reportapi.Confidence {
	return reportapi.Confidence{Label: value.Label, Score: value.Score}
}
