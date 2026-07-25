package service

import (
	"bytes"
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	eventpublicationapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	publicationdomain "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	researchanchordomainimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanchorimport"
	researchanchorimportapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanchorimport"
	researchdomainimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
	researchimportapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func (s *DataService) ImportReviewedEvents(ctx context.Context, request *v1.ImportRequest) (*v1.Response, error) {
	if s == nil || s.dependencies.EventPublications == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "Event Publication service is unavailable")
	}
	publication, err := publicationdomain.DecodeStrict(bytes.NewReader(request.Payload))
	if err != nil {
		return nil, decodeError(err)
	}
	result, err := s.dependencies.EventPublications.Import(ctx, principalIdentity(ctx), publication)
	if err == nil {
		return &v1.Response{Status: v1.StatusCreated, Result: eventPublicationDTO(result)}, nil
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

func (s *DataService) ImportResearchThemes(ctx context.Context, request *v1.ImportRequest) (*v1.Response, error) {
	if s == nil || s.dependencies.ResearchThemeImports == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research Theme import service is unavailable")
	}
	batch, err := researchdomainimport.DecodeStrict(bytes.NewReader(request.Payload))
	if err != nil {
		var contract *researchdomainimport.DecodeError
		if errors.As(err, &contract) {
			return nil, publicErrorWithDetails(v1.StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Theme V1 contract", map[string]any{
				"theme_key": contract.ThemeKey, "path": contract.Path,
			})
		}
		return nil, decodeError(err)
	}
	result, err := s.dependencies.ResearchThemeImports.Import(ctx, principalIdentity(ctx), batch)
	if err != nil {
		return nil, researchThemeImportError(err)
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	return &v1.Response{Status: status, Result: researchThemeImportDTO(result)}, nil
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

func (s *DataService) ImportResearchAnchors(ctx context.Context, request *v1.ImportRequest) (*v1.Response, error) {
	if s == nil || s.dependencies.ResearchAnchorImports == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research Anchor import service is unavailable")
	}
	publication, err := researchanchordomainimport.DecodeStrict(bytes.NewReader(request.Payload))
	if err != nil {
		var contract *researchanchordomainimport.DecodeError
		if errors.As(err, &contract) {
			return nil, researchAnchorError(v1.StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Anchor V1 contract", contract.CenterChainNodeID, contract.Path, "")
		}
		return nil, decodeError(err)
	}
	result, err := s.dependencies.ResearchAnchorImports.Import(ctx, principalIdentity(ctx), publication)
	if err != nil {
		return nil, researchAnchorImportError(err)
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	return &v1.Response{Status: status, Result: researchAnchorImportDTO(result)}, nil
}

func researchAnchorImportError(err error) error {
	var validation *researchanchordomainimport.ValidationError
	if errors.As(err, &validation) {
		return researchAnchorError(v1.StatusBadRequest, "RESEARCH_ANCHOR_IMPORT_REJECTED", "research Anchor publication failed validation", validation.CenterChainNodeID, validation.Path, validation.Reference)
	}
	var contract *researchanchorimportapp.ContractError
	if errors.As(err, &contract) {
		return researchAnchorError(v1.StatusBadRequest, "RESEARCH_ANCHOR_IMPORT_REJECTED", "research Anchor publication failed validation", contract.CenterChainNodeID, contract.Path, contract.Reference)
	}
	var reference *researchanchorimportapp.ReferenceError
	if errors.As(err, &reference) {
		code, message := "RESEARCH_ANCHOR_REFERENCE_NOT_FOUND", "research Anchor publication references missing data"
		if reference.Kind == researchanchorimportapp.ReferenceInvalid {
			code, message = "RESEARCH_ANCHOR_REFERENCE_INVALID", "research Anchor publication references data outside its Theme boundary"
		}
		return researchAnchorError(v1.StatusUnprocessableEntity, code, message, reference.CenterChainNodeID, reference.Path, reference.Reference)
	}
	switch {
	case errors.Is(err, researchanchorimportapp.ErrPayloadConflict):
		return publicError(v1.StatusConflict, "RESEARCH_ANCHOR_PAYLOAD_CONFLICT", "theme_id conflicts with the published Research Anchor payload")
	case errors.Is(err, researchanchorimportapp.ErrPublisherConflict):
		return publicError(v1.StatusConflict, "RESEARCH_ANCHOR_PUBLISHER_CONFLICT", "Theme or Anchor receipt belongs to another publisher subject")
	default:
		return publicError(v1.StatusInternalServerError, "RESEARCH_ANCHOR_IMPORT_FAILED", "research Anchor import failed")
	}
}

func researchAnchorError(status int, code, message, centerID, path, reference string) error {
	return publicErrorWithDetails(status, code, message, map[string]any{
		"center_chain_node_id": centerID, "path": path, "reference": reference,
	})
}
