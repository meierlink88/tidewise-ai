package service

import (
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func (s *DataService) PublishResearchTheme(ctx context.Context, request *v1.ResearchThemeImportRequest) (*v1.Response[v1.ResearchThemeImportResult], error) {
	if s == nil || s.dependencies.ResearchThemeImports == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research Theme import service is unavailable")
	}
	var result researchpublication.Result
	var err error
	if request.PublicationMode == researchpublication.SnapshotPublicationMode && request.Snapshot != nil {
		result, err = s.dependencies.ResearchThemeImports.PublishSnapshot(ctx, principalIdentity(ctx), researchThemeSnapshotImportInput(request.Snapshot))
	} else {
		result, err = s.dependencies.ResearchThemeImports.Publish(ctx, principalIdentity(ctx), researchThemeImportInput(request))
	}
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
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme aggregate failed validation", map[string]any{
			"path": validation.Path, "reference": validation.Reference,
		})
	}
	var themeValidation *researchthemeimport.ValidationError
	if errors.As(err, &themeValidation) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme aggregate failed validation", map[string]any{
			"path": themeValidation.Path, "reference": themeValidation.Reference,
		})
	}
	var treeValidation *researchreasoningtreeimport.ValidationError
	if errors.As(err, &treeValidation) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme aggregate failed validation", map[string]any{
			"path": treeValidation.Path, "reference": treeValidation.Reference,
		})
	}
	var reference *researchpublication.ReferenceError
	if errors.As(err, &reference) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_REFERENCE_INVALID", "research Theme aggregate references unavailable or inconsistent Data records", map[string]any{
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
