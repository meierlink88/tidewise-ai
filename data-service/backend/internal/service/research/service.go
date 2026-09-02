package research

import (
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	researchapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/research"
	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
)

type UseCase interface {
	Search(context.Context, researchbiz.GraphSearchRequest) (researchbiz.GraphSearchResult, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Research Graph use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) SearchResearchGraph(
	ctx context.Context,
	request *researchapi.ResearchGraphSearchRequest,
) (*v1.Response[researchapi.ResearchGraphSearchResult], error) {
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusServiceUnavailable, "RESEARCH_GRAPH_NOT_READY", "Research Graph service is unavailable")
	}
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, "RESEARCH_GRAPH_INVALID", "request is required")
	}
	filters := make([]researchbiz.RelationFilter, 0, len(request.RelationFilters))
	for _, filter := range request.RelationFilters {
		filters = append(filters, researchbiz.RelationFilter{
			RelationType: filter.RelationType,
			Direction:    researchbiz.Direction(filter.Direction),
		})
	}
	result, err := s.useCase.Search(ctx, researchbiz.GraphSearchRequest{
		AnalysisAsOf:    request.AnalysisAsOf,
		SeedEntityIDs:   request.SeedEntityIDs,
		RelationFilters: filters,
		MaxDepth:        request.MaxDepth,
		IndustryChainID: request.IndustryChainID,
		NodeBudget:      request.NodeBudget,
		EdgeBudget:      request.EdgeBudget,
	})
	if err != nil {
		var validation *researchbiz.GraphValidationError
		if errors.As(err, &validation) {
			return nil, publicError(v1.StatusBadRequest, "RESEARCH_GRAPH_INVALID", validation.Reason)
		}
		var resourceLimit *researchbiz.ResearchResourceLimitError
		if errors.As(err, &resourceLimit) {
			return nil, publicErrorWithDetails(
				v1.StatusTooManyRequests,
				"RESEARCH_GRAPH_RESOURCE_LIMIT",
				resourceLimit.Reason,
				researchResourceLimitDetails(
					resourceLimit.Component,
					resourceLimit.ActualRows,
					resourceLimit.MaxRows,
					resourceLimit.ActualBytes,
					resourceLimit.MaxBytes,
					resourceLimit.RetryGuidance,
				),
			)
		}
		return nil, publicError(v1.StatusInternalServerError, "RESEARCH_GRAPH_FAILED", "Research Graph query failed")
	}
	dto := researchapi.ResearchGraphSearchResult{
		ContractVersion:          result.ContractVersion,
		AnalysisAsOf:             result.AnalysisAsOf,
		QueryFingerprint:         result.QueryFingerprint,
		GraphFingerprint:         result.GraphFingerprint,
		ActualDepth:              result.ActualDepth,
		Entities:                 researchGraphEntityDTOs(result.Entities),
		RelationDefinitions:      researchGraphRelationDTOs(result.RelationDefinitions),
		EntityRelations:          researchGraphEntityRelationDTOs(result.EntityRelations),
		IndustryChains:           researchGraphIndustryChainDTOs(result.IndustryChains),
		IndustryChainMemberships: researchGraphMembershipDTOs(result.IndustryChainMemberships),
		IndustryChainGraphEdges:  researchGraphIndustryEdgeDTOs(result.IndustryChainGraphEdges),
	}
	return &v1.Response[researchapi.ResearchGraphSearchResult]{Status: v1.StatusOK, Result: dto}, nil
}

func researchResourceLimitDetails(
	component string,
	actualRows *int64,
	maxRows *int64,
	actualBytes *int64,
	maxBytes *int64,
	retryGuidance string,
) researchapi.ResearchResourceLimitDetails {
	return researchapi.ResearchResourceLimitDetails{
		Component:     component,
		ActualRows:    actualRows,
		MaxRows:       maxRows,
		ActualBytes:   actualBytes,
		MaxBytes:      maxBytes,
		RetryGuidance: retryGuidance,
	}
}

func researchGraphEntityDTOs(values []researchbiz.GraphEntity) []researchapi.ResearchGraphEntity {
	result := make([]researchapi.ResearchGraphEntity, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphEntity{
			EntityID: item.EntityID, EntityType: item.EntityType, Name: item.Name,
			CanonicalName: item.CanonicalName, Aliases: item.Aliases, Status: item.Status,
		})
	}
	return result
}

func researchGraphRelationDTOs(values []researchbiz.GraphRelationDefinition) []researchapi.ResearchGraphRelationDefinition {
	result := make([]researchapi.ResearchGraphRelationDefinition, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphRelationDefinition{RelationType: item.RelationType, Direction: item.Direction})
	}
	return result
}

func researchGraphEntityRelationDTOs(values []researchbiz.GraphEntityRelation) []researchapi.ResearchGraphEntityRelation {
	result := make([]researchapi.ResearchGraphEntityRelation, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphEntityRelation{
			EntityRelationID: item.EntityRelationID,
			FromEntityID:     item.FromEntityID,
			ToEntityID:       item.ToEntityID,
			RelationType:     item.RelationType,
			Status:           item.Status,
		})
	}
	return result
}

func researchGraphIndustryChainDTOs(values []researchbiz.GraphIndustryChain) []researchapi.ResearchGraphIndustryChain {
	result := make([]researchapi.ResearchGraphIndustryChain, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphIndustryChain{
			IndustryChainID: item.IndustryChainID,
			Scope:           item.Scope,
			TargetOutput:    item.TargetOutput,
			EndUse:          item.EndUse,
			Geography:       item.Geography,
			AsOfDate:        item.AsOfDate,
			ReviewStatus:    item.ReviewStatus,
		})
	}
	return result
}

func researchGraphMembershipDTOs(values []researchbiz.GraphIndustryChainMembership) []researchapi.ResearchGraphIndustryChainMembership {
	result := make([]researchapi.ResearchGraphIndustryChainMembership, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphIndustryChainMembership{
			IndustryChainID: item.IndustryChainID,
			ChainNodeID:     item.ChainNodeID,
			Position:        item.Position,
			ContextualStage: item.ContextualStage,
		})
	}
	return result
}

func researchGraphIndustryEdgeDTOs(values []researchbiz.GraphIndustryChainEdge) []researchapi.ResearchGraphIndustryChainGraphEdge {
	result := make([]researchapi.ResearchGraphIndustryChainGraphEdge, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphIndustryChainGraphEdge{
			IndustryChainGraphEdgeID: item.IndustryChainGraphEdgeID,
			IndustryChainID:          item.IndustryChainID,
			FromChainNodeID:          item.FromChainNodeID,
			ToChainNodeID:            item.ToChainNodeID,
			RelationType:             item.RelationType,
		})
	}
	return result
}

func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}

func publicErrorWithDetails(status int, code, message string, details any) error {
	return v1.NewPublicError(status, code, message, details)
}

var _ researchapi.Service = (*Service)(nil)
