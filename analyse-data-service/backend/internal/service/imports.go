package service

import (
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	eventpublicationapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	publicationdomain "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	researchtreedomainimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	researchtreeimportapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	researchdomainimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
	researchimportapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func (s *DataService) ImportReviewedEvents(ctx context.Context, request *v1.EventPublicationRequest) (*v1.Response[v1.EventPublicationResult], error) {
	if s == nil || s.dependencies.EventPublications == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "Event Publication service is unavailable")
	}
	publication := eventPublicationInput(request)
	result, err := s.dependencies.EventPublications.Import(ctx, principalIdentity(ctx), publication)
	if err == nil {
		return &v1.Response[v1.EventPublicationResult]{Status: v1.StatusCreated, Result: eventPublicationDTO(result)}, nil
	}
	var validation *publicationdomain.ValidationError
	if errors.As(err, &validation) {
		return nil, publicErrorWithDetails(v1.StatusUnprocessableEntity, "EVENT_PUBLICATION_INVALID", "Event Publication failed validation", map[string]any{"issues": validation.Issues})
	}
	var conflict *eventpublicationapp.ConflictError
	if errors.As(err, &conflict) {
		return nil, publicErrorWithDetails(v1.StatusConflict, "EVENT_PUBLICATION_CONFLICT", "Event Publication conflicts with stored data", map[string]any{"issues": conflict.Issues})
	}
	return nil, publicError(v1.StatusInternalServerError, "EVENT_PUBLICATION_FAILED", "Event Publication failed")
}

func (s *DataService) ImportResearchThemes(ctx context.Context, request *v1.ResearchThemeImportRequest) (*v1.Response[v1.ResearchThemeImportResult], error) {
	if s == nil || s.dependencies.ResearchThemeImports == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research Theme import service is unavailable")
	}
	batch := researchThemeImportInput(request)
	result, err := s.dependencies.ResearchThemeImports.Import(ctx, principalIdentity(ctx), batch)
	if err != nil {
		return nil, researchThemeImportError(err)
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	return &v1.Response[v1.ResearchThemeImportResult]{Status: status, Result: researchThemeImportDTO(result)}, nil
}

func researchThemeImportError(err error) error {
	var validation *researchdomainimport.ValidationError
	if errors.As(err, &validation) {
		return publicErrorWithDetails(v1.StatusBadRequest, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme batch failed validation", map[string]any{
			"theme_key": validation.ThemeKey, "path": validation.Path, "reference": validation.Reference,
		})
	}
	var reference *researchimportapp.ReferenceError
	if errors.As(err, &reference) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_REFERENCE_NOT_FOUND", "research Theme batch references missing master data", map[string]any{
			"theme_key": reference.ThemeKey, "path": reference.Path, "reference": reference.Reference,
		})
	}
	switch {
	case errors.Is(err, researchimportapp.ErrPayloadConflict):
		return publicError(v1.StatusConflict, "RESEARCH_THEME_PAYLOAD_CONFLICT", "analysis_batch_id conflicts with the published payload")
	case errors.Is(err, researchimportapp.ErrPublisherConflict):
		return publicError(v1.StatusConflict, "RESEARCH_THEME_PUBLISHER_CONFLICT", "analysis_batch_id belongs to another publisher subject")
	default:
		return publicError(v1.StatusInternalServerError, "RESEARCH_THEME_IMPORT_FAILED", "research Theme import failed")
	}
}

func (s *DataService) ImportResearchReasoningTrees(ctx context.Context, request *v1.ResearchReasoningTreeImportRequest) (*v1.Response[v1.ResearchReasoningTreeImportResult], error) {
	if s == nil || s.dependencies.ResearchReasoningTreeImports == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research Reason Tree import service is unavailable")
	}
	publication := researchReasoningTreeImportInput(request)
	result, err := s.dependencies.ResearchReasoningTreeImports.Import(ctx, principalIdentity(ctx), publication)
	if err != nil {
		return nil, researchReasoningTreeImportError(err)
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	return &v1.Response[v1.ResearchReasoningTreeImportResult]{Status: status, Result: researchReasoningTreeImportDTO(result)}, nil
}

func researchReasoningTreeImportError(err error) error {
	var validation *researchtreedomainimport.ValidationError
	if errors.As(err, &validation) {
		return researchReasoningTreeError(v1.StatusBadRequest, "RESEARCH_REASONING_TREE_IMPORT_REJECTED", "research Reason Tree publication failed validation", validation.IndustryChainEntityID, validation.Path, validation.Reference)
	}
	var contract *researchtreeimportapp.ContractError
	if errors.As(err, &contract) {
		return researchReasoningTreeError(v1.StatusBadRequest, "RESEARCH_REASONING_TREE_IMPORT_REJECTED", "research Reason Tree publication failed validation", "", contract.Path, contract.Reference)
	}
	var reference *researchtreeimportapp.ReferenceError
	if errors.As(err, &reference) {
		code, message := "RESEARCH_REASONING_TREE_REFERENCE_NOT_FOUND", "research Reason Tree publication references missing data"
		if reference.Kind == researchtreeimportapp.ReferenceInvalid {
			code, message = "RESEARCH_REASONING_TREE_REFERENCE_INVALID", "research Reason Tree publication references data outside its Theme or Industry Chain boundary"
		}
		return researchReasoningTreeError(v1.StatusUnprocessableEntity, code, message, reference.IndustryChainEntityID, reference.Path, reference.Reference)
	}
	switch {
	case errors.Is(err, researchtreeimportapp.ErrPayloadConflict):
		return publicError(v1.StatusConflict, "RESEARCH_REASONING_TREE_PAYLOAD_CONFLICT", "theme_id conflicts with the published Research Reason Tree payload")
	case errors.Is(err, researchtreeimportapp.ErrPublisherConflict):
		return publicError(v1.StatusConflict, "RESEARCH_REASONING_TREE_PUBLISHER_CONFLICT", "Theme or Reason Tree receipt belongs to another publisher subject")
	default:
		return publicError(v1.StatusInternalServerError, "RESEARCH_REASONING_TREE_IMPORT_FAILED", "research Reason Tree import failed")
	}
}

func researchReasoningTreeError(status int, code, message, chainID, path, reference string) error {
	return publicErrorWithDetails(status, code, message, map[string]any{
		"industry_chain_entity_id": chainID, "path": path, "reference": reference,
	})
}
