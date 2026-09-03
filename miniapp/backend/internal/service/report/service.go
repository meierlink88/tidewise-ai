package report

import (
	"context"
	"errors"
	"strconv"
	"strings"
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

func (s *Service) ListIndustryChains(ctx context.Context, request *api.IndustryChainListRequest) (*api.CardCollection, error) {
	if s == nil || s.useCase == nil || request == nil || request.HasUnknownQuery {
		return nil, v1.ErrInvalidRequest
	}
	limit := 0
	if strings.TrimSpace(request.Limit) != "" {
		value, err := strconv.Atoi(request.Limit)
		if err != nil {
			return nil, v1.ErrInvalidRequest
		}
		limit = value
	}
	page, err := s.useCase.IndustryChains(ctx, request.ReportID, limit, request.Cursor)
	if err != nil {
		return nil, publicError(err)
	}
	items := make([]api.Card, len(page.Items))
	for index, item := range page.Items {
		items[index] = mapCard(item)
	}
	return &api.CardCollection{Items: items, NextCursor: page.NextCursor}, nil
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
	}
	return &api.EvidenceCollection{ReportID: result.ReportID, ScopeToken: result.ScopeToken, Items: items}, nil
}

func mapHome(value biz.Home) api.HomeReport {
	cards := make([]api.Card, len(value.Cards))
	for index, card := range value.Cards {
		cards[index] = mapCard(card)
	}
	return api.HomeReport{Report: mapSummary(value.Report), Cards: cards, NextCursor: value.NextCursor}
}

func mapCard(value biz.Card) api.Card {
	impacts := make([]api.CardImpactItem, len(value.ImpactItems))
	for index, item := range value.ImpactItems {
		impacts[index] = api.CardImpactItem{
			Ref: mapReference(item.Ref), Name: item.Name, Result: mapCoded(item.Result),
			ConclusionBasis: mapOptionalCoded(item.ConclusionBasis), ValidationStatus: mapOptionalCoded(item.ValidationStatus),
			Confidence: mapConfidence(item.Confidence), TimeWindow: mapTimeWindow(item.TimeWindow), EvidenceScopeToken: item.EvidenceScopeToken,
		}
	}
	return api.Card{
		LocalKey: value.LocalKey, Kind: value.Kind, DetailRef: mapReference(value.DetailRef), Title: value.Title,
		Subtitle: value.Subtitle, Conclusion: value.Conclusion, Result: mapCoded(value.Result),
		Confidence: mapConfidence(value.Confidence), TimeWindow: mapTimeWindow(value.TimeWindow),
		ImpactItems: impacts, EvidenceScopeToken: value.EvidenceScopeToken,
	}
}

func mapLayerDetail(value biz.LayerDetail) *api.LayerDetail {
	related := make([]api.RelatedIndustryChain, len(value.RelatedIndustryChains))
	for index, item := range value.RelatedIndustryChains {
		related[index] = api.RelatedIndustryChain{LocalKey: item.LocalKey, Name: item.Name, Result: mapCoded(item.Result)}
	}
	return &api.LayerDetail{Report: mapSummary(value.Report), Layer: mapLayer(value.Layer), RelatedIndustryChains: related}
}

func mapLayer(value biz.Layer) api.Layer {
	anchors := make([]api.Anchor, len(value.Anchors))
	for index, item := range value.Anchors {
		anchors[index] = api.Anchor{
			LocalKey: item.LocalKey, Name: item.Name, CurrentState: item.CurrentState, Result: mapCoded(item.Result),
			ConclusionBasis: mapOptionalCoded(item.ConclusionBasis), ValidationStatus: mapOptionalCoded(item.ValidationStatus),
			TransmissionLogic: item.TransmissionLogic, TimeWindow: mapTimeWindow(item.TimeWindow), Confidence: mapConfidence(item.Confidence), EvidenceScopeToken: item.EvidenceScopeToken,
		}
	}
	steps := make([]api.ReasoningStep, len(value.ReasoningSteps))
	for index, item := range value.ReasoningSteps {
		steps[index] = api.ReasoningStep{Input: item.Input, Mechanism: item.Mechanism, Output: item.Output, ReasoningType: mapCoded(item.ReasoningType), Confidence: mapConfidence(item.Confidence), EvidenceScopeToken: item.EvidenceScopeToken}
	}
	transmissions := make([]api.TransmissionPath, len(value.Transmissions))
	for index, item := range value.Transmissions {
		targets := make([]api.TransmissionTarget, len(item.Targets))
		for targetIndex, target := range item.Targets {
			targets[targetIndex] = api.TransmissionTarget{Ref: mapReference(target.Ref), Name: target.Name, Result: mapCoded(target.Result)}
		}
		transmissions[index] = api.TransmissionPath{LocalKey: item.LocalKey, SourceConclusion: item.SourceConclusion, Targets: targets, Logic: item.Logic, Kind: mapCoded(item.Kind), Confidence: mapConfidence(item.Confidence), Status: mapCoded(item.Status)}
	}
	return api.Layer{Key: value.Key, Title: value.Title, Conclusion: value.Conclusion, Result: mapCoded(value.Result), Confidence: mapConfidence(value.Confidence), TimeWindow: mapTimeWindow(value.TimeWindow), Anchors: anchors, ReasoningSteps: steps, Transmissions: transmissions, Uncertainty: api.LayerUncertainty{Counterevidence: value.Uncertainty.Counterevidence, EvidenceGap: value.Uncertainty.EvidenceGap, Boundary: value.Uncertainty.Boundary, ReversalCondition: value.Uncertainty.ReversalCondition}, EvidenceScopeToken: value.EvidenceScopeToken}
}

func mapIndustryChainDetail(value biz.IndustryChainDetail) *api.IndustryChainDetail {
	chain := value.IndustryChain
	topologyNodes := make([]api.IndustryChainTopologyNode, len(chain.TopologyNodes))
	for index, item := range chain.TopologyNodes {
		topologyNodes[index] = api.IndustryChainTopologyNode{LocalKey: item.LocalKey, Name: item.Name}
	}
	nodes := make([]api.IndustryChainNode, len(chain.Nodes))
	for index, item := range chain.Nodes {
		nodes[index] = api.IndustryChainNode{LocalKey: item.LocalKey, NodeLocalKey: item.NodeLocalKey, Name: item.Name, Impact: item.Impact, Result: mapCoded(item.Result), ConclusionBasis: mapOptionalCoded(item.ConclusionBasis), ValidationStatus: mapOptionalCoded(item.ValidationStatus), TransmissionLogic: item.TransmissionLogic, TimeWindow: mapTimeWindow(item.TimeWindow), Confidence: mapConfidence(item.Confidence), EvidenceScopeToken: item.EvidenceScopeToken}
	}
	edges := make([]api.IndustryChainEdge, len(chain.Edges))
	for index, item := range chain.Edges {
		edges[index] = api.IndustryChainEdge{FromNodeKey: item.FromNodeKey, ToNodeKey: item.ToNodeKey, Relation: mapCoded(item.Relation)}
	}
	return &api.IndustryChainDetail{Report: mapSummary(value.Report), IndustryChain: api.IndustryChain{LocalKey: chain.LocalKey, Name: chain.Name, Conclusion: chain.Conclusion, Status: chain.Status, Result: mapCoded(chain.Result), Confidence: mapConfidence(chain.Confidence), TimeWindow: mapTimeWindow(chain.TimeWindow), Path: chain.Path, TopologyNodes: topologyNodes, Nodes: nodes, Edges: edges, CounterevidenceAndGap: chain.CounterevidenceAndGap, StopCondition: chain.StopCondition, EvidenceScopeToken: chain.EvidenceScopeToken}}
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
	return api.Summary{ID: value.ID, GeneratedAt: formatTime(value.GeneratedAt), PublishedAt: formatTime(value.PublishedAt), IndustryChainCount: value.IndustryChainCount}
}
func mapCoded(value biz.CodedLabel) api.CodedLabel {
	return api.CodedLabel{Code: value.Code, Label: value.Label}
}
func mapOptionalCoded(value *biz.CodedLabel) *api.CodedLabel {
	if value == nil {
		return nil
	}
	mapped := mapCoded(*value)
	return &mapped
}
func mapConfidence(value biz.Confidence) api.Confidence {
	return api.Confidence{Code: value.Code, Label: value.Label, Score: value.Score}
}
func mapTimeWindow(value biz.TimeWindow) api.TimeWindow {
	return api.TimeWindow{Code: value.Code, Label: value.Label}
}
func mapReference(value biz.Reference) api.Reference {
	return api.Reference{Type: value.Type, LocalKey: value.LocalKey}
}
func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func cloneStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}

var _ api.Service = (*Service)(nil)
