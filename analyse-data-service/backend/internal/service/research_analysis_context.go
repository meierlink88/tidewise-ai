package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
)

func (s *DataService) ListResearchAnalysisContext(
	ctx context.Context,
	request *v1.ResearchAnalysisContextRequest,
) (*v1.Response[v1.ResearchAnalysisContext], error) {
	if s == nil || s.dependencies.ResearchAnalysisContext == nil {
		return nil, publicError(
			v1.StatusServiceUnavailable,
			"RESEARCH_ANALYSIS_CONTEXT_NOT_READY",
			"Research Analysis Context service is unavailable",
		)
	}
	result, err := s.dependencies.ResearchAnalysisContext.List(
		ctx,
		researchanalysiscontext.Request{
			DiscoveryWindowStart:   request.DiscoveryWindowStart,
			DiscoveryWindowEnd:     request.DiscoveryWindowEnd,
			AnalysisAsOf:           request.AnalysisAsOf,
			PredictionHorizonStart: request.PredictionHorizonStart,
			PredictionHorizonEnd:   request.PredictionHorizonEnd,
			PageSize:               request.PageSize,
			Cursor:                 request.Cursor,
		},
	)
	if err != nil {
		if errors.Is(err, researchanalysiscontext.ErrHistoricalSemanticsUnavailable) {
			return nil, publicError(
				v1.StatusUnprocessableEntity,
				"RESEARCH_ANALYSIS_CONTEXT_HISTORY_UNAVAILABLE",
				"strict historical Event semantics are unavailable; choose a current analysis_as_of or use a future snapshot capability",
			)
		}
		var validation *researchanalysiscontext.ValidationError
		if errors.As(err, &validation) {
			return nil, publicError(
				v1.StatusBadRequest,
				"RESEARCH_ANALYSIS_CONTEXT_INVALID",
				validation.Reason,
			)
		}
		var resourceLimit *researchanalysiscontext.ResourceLimitError
		if errors.As(err, &resourceLimit) {
			return nil, publicError(
				v1.StatusTooManyRequests,
				"RESEARCH_ANALYSIS_CONTEXT_RESOURCE_LIMIT",
				resourceLimit.Reason,
			)
		}
		return nil, publicError(
			v1.StatusInternalServerError,
			"RESEARCH_ANALYSIS_CONTEXT_FAILED",
			"Research Analysis Context query failed",
		)
	}
	dto, err := researchAnalysisContextDTO(result)
	if err != nil {
		return nil, publicError(
			v1.StatusInternalServerError,
			"RESEARCH_ANALYSIS_CONTEXT_CONTRACT_DRIFT",
			"Research Analysis Context response contract is unavailable",
		)
	}
	return &v1.Response[v1.ResearchAnalysisContext]{Status: v1.StatusOK, Result: dto}, nil
}

func researchAnalysisContextDTO(
	result researchanalysiscontext.Result,
) (v1.ResearchAnalysisContext, error) {
	var bundles []v1.ResearchAnalysisEventSemanticBundle
	if err := convertResearchAnalysisValue(result.EventSemanticBundles, &bundles); err != nil {
		return v1.ResearchAnalysisContext{}, err
	}
	var dictionaries v1.ResearchAnalysisDictionaries
	if err := convertResearchAnalysisValue(result.Dictionaries, &dictionaries); err != nil {
		return v1.ResearchAnalysisContext{}, err
	}
	return v1.ResearchAnalysisContext{
		ContractVersion:        result.ContractVersion,
		TemporalSemantics:      result.TemporalSemantics,
		TemporalLimitation:     result.TemporalLimitation,
		DictionaryFingerprint:  result.DictionaryFingerprint,
		DiscoveryWindowStart:   result.DiscoveryWindowStart,
		DiscoveryWindowEnd:     result.DiscoveryWindowEnd,
		AnalysisAsOf:           result.AnalysisAsOf,
		PredictionHorizonStart: result.PredictionHorizonStart,
		PredictionHorizonEnd:   result.PredictionHorizonEnd,
		EventSemanticBundles:   bundles,
		Dictionaries:           dictionaries,
		NextCursor:             result.NextCursor,
		HasMore:                result.HasMore,
	}, nil
}

func convertResearchAnalysisValue(source, target any) error {
	payload, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("encode Research Analysis Context Biz DTO: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode Research Analysis Context API DTO: %w", err)
	}
	return nil
}
