package report

import (
	"context"
	"errors"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
	api "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1/report"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
)

type Service struct {
	useCase *biz.UseCase
}

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
	return &api.HomeResponse{Selection: api.Selection{
		Mode: result.Selection.Mode, Date: result.Selection.Date, Timezone: result.Selection.Timezone,
	}, Reports: reports}, nil
}

func (s *Service) GetLayer(ctx context.Context, request *api.LayerRequest) (*api.LayerDetail, error) {
	if s == nil || s.useCase == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.useCase.Layer(ctx, request.ReportID, request.LayerKey)
	if err != nil {
		return nil, publicError(err)
	}
	return mapLayerDetail(result), nil
}

func (s *Service) GetIndustryChain(ctx context.Context, request *api.IndustryChainRequest) (*api.IndustryChainDetail, error) {
	if s == nil || s.useCase == nil || request == nil {
		return nil, v1.ErrInvalidRequest
	}
	result, err := s.useCase.IndustryChain(ctx, request.ReportID, request.ChainKey)
	if err != nil {
		return nil, publicError(err)
	}
	return mapIndustryChainDetail(result), nil
}

func (s *Service) ListEvidences(ctx context.Context, request *api.EvidenceRequest) (*api.EvidenceCollection, error) {
	if s == nil || s.useCase == nil || request == nil || request.HasUnknownQuery {
		return nil, v1.ErrInvalidRequest
	}
	scope := biz.EvidenceScope{Type: request.ScopeType, Key: request.ScopeKey}
	result, err := s.useCase.Evidences(ctx, request.ReportID, scope)
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
		items[index] = api.EvidenceItem{PublishedAt: publishedAt, Summary: item.Summary,
			Keywords: cloneStrings(item.Keywords)}
	}
	return &api.EvidenceCollection{ReportID: result.ReportID,
		Scope: mapScope(result.Scope), Items: items}, nil
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

func mapHome(value biz.Home) api.HomeReport {
	cards := make([]api.Card, len(value.Cards))
	for index, card := range value.Cards {
		impacts := make([]api.CardImpactItem, len(card.ImpactItems))
		for impactIndex, impact := range card.ImpactItems {
			impacts[impactIndex] = api.CardImpactItem{Ref: mapReference(impact.Ref), Name: impact.Name,
				Result: mapResult(impact.Result), Confidence: mapConfidence(impact.Confidence),
				TimeWindow: impact.TimeWindow, HasEvidence: impact.HasEvidence}
		}
		cards[index] = api.Card{Key: card.Key, Kind: card.Kind, DisplayOrder: card.Order,
			DetailRef: mapReference(card.DetailRef), Title: card.Title, Subtitle: card.Subtitle,
			Conclusion: card.Conclusion, Result: mapResult(card.Result), Confidence: mapConfidence(card.Confidence),
			TimeWindow: card.TimeWindow, ImpactItems: impacts, HasEvidence: card.HasEvidence}
	}
	return api.HomeReport{Report: mapSummary(value.Report), IndustryChainCount: value.IndustryChainCount,
		Cards: cards}
}

func mapLayerDetail(value biz.LayerDetail) *api.LayerDetail {
	chains := make([]api.IndustryChainSummary, len(value.RelatedIndustryChains))
	for index, chain := range value.RelatedIndustryChains {
		chains[index] = mapIndustryChainSummary(chain)
	}
	return &api.LayerDetail{Report: mapSummary(value.Report), Layer: mapLayer(value.Layer),
		RelatedIndustryChains: chains}
}

func mapLayer(value biz.Layer) api.Layer {
	anchors := make([]api.Anchor, len(value.Anchors))
	for index, item := range value.Anchors {
		anchors[index] = api.Anchor{Key: item.Key, DisplayOrder: item.DisplayOrder, Name: item.Name,
			CurrentState: item.CurrentState, Result: mapResult(item.Result), Nature: mapNature(item.Nature),
			Reasoning: item.Reasoning, TimeWindow: item.TimeWindow, Confidence: mapConfidence(item.Confidence),
			Scope: mapScope(item.Scope), HasEvidence: item.HasEvidence}
	}
	steps := make([]api.ReasoningStep, len(value.ReasoningSteps))
	for index, item := range value.ReasoningSteps {
		steps[index] = api.ReasoningStep{Key: item.Key, DisplayOrder: item.DisplayOrder,
			Input: item.Input, Mechanism: item.Mechanism, Output: item.Output, Type: item.Type,
			Confidence: mapConfidence(item.Confidence), Scope: mapScope(item.Scope), HasEvidence: item.HasEvidence}
	}
	paths := make([]api.TransmissionPath, len(value.DownwardTransmission.PublishedPaths))
	for index, item := range value.DownwardTransmission.PublishedPaths {
		targets := make([]api.TransmissionTarget, len(item.TargetRefs))
		for targetIndex, target := range item.TargetRefs {
			var ref *api.Reference
			if target.Ref != nil {
				mapped := mapReference(*target.Ref)
				ref = &mapped
			}
			targets[targetIndex] = api.TransmissionTarget{Ref: ref, Label: target.Label,
				Result: mapResult(target.Result)}
		}
		paths[index] = api.TransmissionPath{Key: item.Key, DisplayOrder: item.DisplayOrder,
			SourceConclusion: item.SourceConclusion, TargetRefs: targets, Logic: item.Logic,
			RelationNature: item.RelationNature, EvidenceRole: item.EvidenceRole,
			Confidence: mapConfidence(item.Confidence), Status: item.Status,
			Scope: mapScope(item.Scope), HasEvidence: item.HasEvidence}
	}
	candidates := make([]api.CandidateMechanism, len(value.DownwardTransmission.CandidateMechanisms))
	for index, item := range value.DownwardTransmission.CandidateMechanisms {
		candidates[index] = api.CandidateMechanism{Key: item.Key, DisplayOrder: item.DisplayOrder,
			Mechanism: item.Mechanism, EvidenceGap: item.EvidenceGap, Confidence: mapConfidence(item.Confidence),
			Scope: mapScope(item.Scope), HasEvidence: item.HasEvidence}
	}
	checkpoints := mapCheckpoints(value.Uncertainty.Checkpoints)
	return api.Layer{Key: value.Key, DisplayOrder: value.DisplayOrder, Title: value.Title,
		Conclusion: value.Conclusion, Result: mapResult(value.Result), Confidence: mapConfidence(value.Confidence),
		TimeWindow: value.TimeWindow, Anchors: anchors, ReasoningSteps: steps,
		RelatedAnchorKeys: cloneStrings(value.RelatedAnchorKeys),
		RelatedChainKeys:  cloneStrings(value.RelatedChainKeys),
		DownwardTransmission: api.DownwardTransmission{Summary: value.DownwardTransmission.Summary,
			PublishedPaths: paths, CandidateMechanisms: candidates,
			BoundaryNotes: cloneStrings(value.DownwardTransmission.BoundaryNotes)},
		Uncertainty: api.LayerUncertainty{Counterevidence: value.Uncertainty.Counterevidence,
			EvidenceGap: value.Uncertainty.EvidenceGap, Boundary: value.Uncertainty.Boundary,
			ReversalCondition: value.Uncertainty.ReversalCondition, Checkpoints: checkpoints},
		Scope: mapScope(value.Scope), HasEvidence: value.HasEvidence}
}

func mapIndustryChainSummary(value biz.IndustryChainSummary) api.IndustryChainSummary {
	return api.IndustryChainSummary{Key: value.Key, DisplayOrder: value.DisplayOrder, Name: value.Name,
		Conclusion: value.Conclusion, Status: value.Status, Result: mapResult(value.Result),
		Confidence: mapConfidence(value.Confidence), TimeWindow: value.TimeWindow,
		Scope: mapScope(value.Scope), HasEvidence: value.HasEvidence}
}

func mapIndustryChainDetail(value biz.IndustryChainDetail) *api.IndustryChainDetail {
	chain := value.IndustryChain
	nodes := make([]api.IndustryChainNode, len(chain.Nodes))
	for index, item := range chain.Nodes {
		nodes[index] = api.IndustryChainNode{Key: item.Key, DisplayOrder: item.DisplayOrder,
			Name: item.Name, Impact: item.Impact, Result: mapResult(item.Result), Nature: mapNature(item.Nature),
			Reasoning: item.Reasoning, TimeWindow: item.TimeWindow, Confidence: mapConfidence(item.Confidence),
			Scope: mapScope(item.Scope), HasEvidence: item.HasEvidence}
	}
	edges := make([]api.IndustryChainEdge, len(chain.Edges))
	for index, item := range chain.Edges {
		edges[index] = api.IndustryChainEdge{Key: item.Key, DisplayOrder: item.DisplayOrder,
			FromNodeKey: item.FromNodeKey, ToNodeKey: item.ToNodeKey, RelationLabel: item.RelationLabel}
	}
	return &api.IndustryChainDetail{Report: mapSummary(value.Report), IndustryChain: api.IndustryChain{
		Key: chain.Key, ClaimKey: chain.ClaimKey, DisplayOrder: chain.DisplayOrder, Name: chain.Name,
		Conclusion: chain.Conclusion, Status: chain.Status, Result: mapResult(chain.Result),
		Confidence: mapConfidence(chain.Confidence), TimeWindow: chain.TimeWindow,
		PathSummary: chain.PathSummary, AcceptedHypothesisSummary: chain.AcceptedHypothesisSummary,
		Nodes: nodes, Edges: edges, Uncertainty: api.ChainUncertainty{
			CounterevidenceAndGap: chain.Uncertainty.CounterevidenceAndGap,
			StopCondition:         chain.Uncertainty.StopCondition,
			Checkpoints:           mapCheckpoints(chain.Uncertainty.Checkpoints)},
		Scope: mapScope(chain.Scope), HasEvidence: chain.HasEvidence,
	}}
}

func mapCheckpoints(values []biz.Checkpoint) []api.Checkpoint {
	items := make([]api.Checkpoint, len(values))
	for index, item := range values {
		items[index] = api.Checkpoint{Key: item.Key, DisplayOrder: item.DisplayOrder, Summary: item.Summary}
	}
	return items
}

func mapSummary(value biz.Summary) api.Summary {
	return api.Summary{ID: value.ID, Title: value.Title,
		GeneratedAt: formatTime(value.GeneratedAt), PublishedAt: formatTime(value.PublishedAt)}
}

func mapResult(value biz.Result) api.Result {
	return api.Result{Code: value.Code, Label: value.Label}
}

func mapNature(value biz.Nature) api.Nature {
	return api.Nature{Code: value.Code, Label: value.Label}
}

func mapConfidence(value biz.Confidence) api.Confidence {
	return api.Confidence{Label: value.Label, Score: value.Score}
}

func mapReference(value biz.Reference) api.Reference {
	return api.Reference{Type: value.Type, Key: value.Key}
}

func mapScope(value biz.EvidenceScope) api.Scope {
	return api.Scope{Type: value.Type, Key: value.Key}
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func cloneStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}

var _ api.Service = (*Service)(nil)
