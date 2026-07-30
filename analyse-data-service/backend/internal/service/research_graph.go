package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchgraph"
)

func (s *DataService) SearchResearchGraph(
	ctx context.Context,
	request *v1.ResearchGraphSearchRequest,
) (*v1.Response[v1.ResearchGraphSearchResult], error) {
	if s == nil || s.dependencies.ResearchGraph == nil {
		return nil, publicError(
			v1.StatusServiceUnavailable,
			"RESEARCH_GRAPH_NOT_READY",
			"Research Graph service is unavailable",
		)
	}
	filters := make([]researchgraph.RelationFilter, 0, len(request.RelationFilters))
	for _, filter := range request.RelationFilters {
		filters = append(filters, researchgraph.RelationFilter{
			RelationType: filter.RelationType,
			Direction:    researchgraph.Direction(filter.Direction),
		})
	}
	result, err := s.dependencies.ResearchGraph.Search(ctx, researchgraph.Request{
		AnalysisAsOf:          request.AnalysisAsOf,
		SeedEntityIDs:         request.SeedEntityIDs,
		RelationFilters:       filters,
		MaxDepth:              request.MaxDepth,
		IndustryChainEntityID: request.IndustryChainEntityID,
		NodeBudget:            request.NodeBudget,
		EdgeBudget:            request.EdgeBudget,
	})
	if err != nil {
		var validation *researchgraph.ValidationError
		if errors.As(err, &validation) {
			return nil, publicError(
				v1.StatusBadRequest,
				"RESEARCH_GRAPH_INVALID",
				validation.Reason,
			)
		}
		var resourceLimit *researchgraph.ResourceLimitError
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
		return nil, publicError(
			v1.StatusInternalServerError,
			"RESEARCH_GRAPH_FAILED",
			"Research Graph query failed",
		)
	}
	var dto v1.ResearchGraphSearchResult
	payload, err := json.Marshal(result)
	if err == nil {
		decoder := json.NewDecoder(bytes.NewReader(payload))
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&dto)
	}
	if err != nil {
		return nil, publicError(
			v1.StatusInternalServerError,
			"RESEARCH_GRAPH_CONTRACT_DRIFT",
			"Research Graph response contract is unavailable",
		)
	}
	return &v1.Response[v1.ResearchGraphSearchResult]{
		Status: v1.StatusOK,
		Result: dto,
	}, nil
}
