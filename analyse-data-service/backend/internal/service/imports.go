package service

import (
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	eventpublicationapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	publicationdomain "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
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

func (s *DataService) PublishResearchTheme(ctx context.Context, request *v1.ResearchThemeImportRequest) (*v1.Response[v1.ResearchThemeImportResult], error) {
	if s == nil || s.dependencies.ResearchThemeImports == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research Theme import service is unavailable")
	}
	batch := researchThemeImportInput(request)
	result, err := s.dependencies.ResearchThemeImports.Publish(ctx, principalIdentity(ctx), batch)
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
	var validation *researchpublication.ValidationError
	if errors.As(err, &validation) {
		return publicErrorWithDetails(v1.StatusBadRequest, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme aggregate failed validation", map[string]any{
			"path": validation.Path, "reference": validation.Reference,
		})
	}
	var themeValidation *researchthemeimport.ValidationError
	if errors.As(err, &themeValidation) {
		return publicErrorWithDetails(v1.StatusBadRequest, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme aggregate failed validation", map[string]any{
			"path": themeValidation.Path, "reference": themeValidation.Reference,
		})
	}
	var treeValidation *researchreasoningtreeimport.ValidationError
	if errors.As(err, &treeValidation) {
		return publicErrorWithDetails(v1.StatusBadRequest, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme aggregate failed validation", map[string]any{
			"path": treeValidation.Path, "reference": treeValidation.Reference,
		})
	}
	var reference *researchpublication.ReferenceError
	if errors.As(err, &reference) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_REFERENCE_INVALID", "research Theme aggregate references unavailable or inconsistent formal data", map[string]any{
			"path": reference.Path, "reference": reference.Reference,
		})
	}
	switch {
	case errors.Is(err, researchpublication.ErrPayloadConflict):
		return publicError(v1.StatusConflict, "RESEARCH_THEME_PAYLOAD_CONFLICT", "analysis_batch_id conflicts with the published payload")
	case errors.Is(err, researchpublication.ErrPublisherConflict):
		return publicError(v1.StatusConflict, "RESEARCH_THEME_PUBLISHER_CONFLICT", "analysis_batch_id belongs to another publisher subject")
	default:
		return publicError(v1.StatusInternalServerError, "RESEARCH_THEME_IMPORT_FAILED", "research Theme import failed")
	}
}
