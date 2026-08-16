package evidence

import (
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	evidenceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/evidence"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
)

type UseCase interface {
	PublishRawEvidence(context.Context, evidencebiz.RawEvidence) (evidencebiz.RawEvidenceResult, error)
	GetRawEvidence(context.Context, string) (evidencebiz.StoredRawEvidence, error)
	PublishEvidence(context.Context, string, []evidencebiz.Evidence) (evidencebiz.EvidenceResult, error)
	ListCategories(context.Context) (evidencebiz.CategoryCatalog, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Evidence use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) ListEvidenceCategories(ctx context.Context) (*v1.Response[evidenceapi.EvidenceCategoryCatalog], error) {
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, evidenceapi.ErrorDataServiceNotReady, "Evidence Category Catalog service is unavailable")
	}
	catalog, err := s.useCase.ListCategories(ctx)
	if err != nil {
		return nil, evidenceCategoryCatalogError(err)
	}
	categories := make([]evidenceapi.EvidenceCategory, len(catalog.Categories))
	for index, category := range catalog.Categories {
		categories[index] = evidenceapi.EvidenceCategory{
			ID: string(category.ID), Code: category.Code, Name: category.Name, Description: category.Description,
		}
	}
	return &v1.Response[evidenceapi.EvidenceCategoryCatalog]{Status: v1.StatusOK, Result: evidenceapi.EvidenceCategoryCatalog{
		Categories: categories,
	}}, nil
}

func evidenceCategoryCatalogError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return publicError(v1.StatusServiceUnavailable, evidenceapi.ErrorEvidenceCategoryCatalogTimeout, "Evidence Category Catalog execution budget exceeded")
	}
	return publicError(v1.StatusInternalServerError, evidenceapi.ErrorEvidenceCategoryCatalogFailed, "Evidence Category Catalog is unavailable")
}

func (s *Service) GetRawEvidence(ctx context.Context, request *evidenceapi.GetRawEvidenceRequest) (*v1.Response[evidenceapi.RawEvidenceReadResult], error) {
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, evidenceapi.ErrorDataServiceNotReady, "Evidence service is unavailable")
	}
	result, err := s.useCase.GetRawEvidence(ctx, request.RawEvidenceID)
	if err != nil {
		return nil, rawEvidenceReadError(err)
	}
	return &v1.Response[evidenceapi.RawEvidenceReadResult]{
		Status: v1.StatusOK,
		Result: evidenceapi.RawEvidenceReadResult{RawEvidence: rawEvidenceReadDTO(result)},
	}, nil
}

func (s *Service) PublishRawEvidence(ctx context.Context, request *evidenceapi.RawEvidencePublicationRequest) (*v1.Response[evidenceapi.RawEvidencePublicationResult], error) {
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, evidenceapi.ErrorDataServiceNotReady, "Evidence Publication service is unavailable")
	}
	result, err := s.useCase.PublishRawEvidence(ctx, rawEvidenceInput(request.RawEvidence))
	if err != nil {
		return nil, rawEvidencePublicationError(err)
	}
	return &v1.Response[evidenceapi.RawEvidencePublicationResult]{Status: v1.StatusCreated, Result: rawEvidenceResultDTO(result)}, nil
}

func (s *Service) PublishEvidence(ctx context.Context, request *evidenceapi.EvidencePublicationRequest) (*v1.Response[evidenceapi.EvidencePublicationResult], error) {
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, evidenceapi.ErrorDataServiceNotReady, "Evidence Publication service is unavailable")
	}
	items := make([]evidencebiz.Evidence, len(request.Evidences))
	for index, item := range request.Evidences {
		items[index] = evidenceInput(item)
	}
	result, err := s.useCase.PublishEvidence(ctx, request.RawEvidenceID, items)
	if err != nil {
		return nil, evidencePublicationError(err)
	}
	return &v1.Response[evidenceapi.EvidencePublicationResult]{Status: v1.StatusCreated, Result: evidenceResultDTO(result)}, nil
}

func evidencePublicationError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return publicError(v1.StatusServiceUnavailable, evidenceapi.ErrorEvidencePublicationTimeout, "Evidence Publication execution budget exceeded")
	}
	var validation *evidencebiz.ValidationError
	if errors.As(err, &validation) {
		status := evidenceValidationStatus(validation.Issues)
		code := evidenceapi.ErrorEvidencePublicationInvalid
		if status == v1.StatusBadRequest {
			code = evidenceapi.ErrorInvalidRequest
		}
		return publicErrorWithDetails(status, code, "Evidence Publication failed validation", map[string]any{"issues": validation.Issues})
	}
	var reference *evidencebiz.ReferenceError
	if errors.As(err, &reference) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, evidenceapi.ErrorEvidencePublicationReferenceInvalid, "Evidence Publication references unavailable data", map[string]any{"issues": reference.Issues})
	}
	var conflict *evidencebiz.ConflictError
	if errors.As(err, &conflict) {
		return publicErrorWithDetails(v1.StatusConflict, evidenceapi.ErrorEvidencePublicationConflict, "Evidence Publication conflicts with stored data", map[string]any{"issues": conflict.Issues})
	}
	return publicError(v1.StatusInternalServerError, evidenceapi.ErrorEvidencePublicationFailed, "Evidence Publication failed")
}

func rawEvidencePublicationError(err error) error {
	var validation *evidencebiz.ValidationError
	if errors.As(err, &validation) {
		return publicErrorWithDetails(v1.StatusBadRequest, evidenceapi.ErrorInvalidRequest, "Raw Evidence Publication request is invalid", map[string]any{"issues": validation.Issues})
	}
	return evidencePublicationError(err)
}

func rawEvidenceReadError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return publicError(v1.StatusServiceUnavailable, evidenceapi.ErrorRawEvidenceReadTimeout, "Raw Evidence read execution budget exceeded")
	}
	var validation *evidencebiz.ValidationError
	if errors.As(err, &validation) {
		return publicErrorWithDetails(v1.StatusBadRequest, evidenceapi.ErrorInvalidRequest, "Raw Evidence identity is invalid", map[string]any{"issues": validation.Issues})
	}
	if errors.Is(err, evidencebiz.ErrRawEvidenceNotFound) {
		return publicError(v1.StatusNotFound, evidenceapi.ErrorRawEvidenceNotFound, "Raw Evidence was not found")
	}
	return publicError(v1.StatusInternalServerError, evidenceapi.ErrorRawEvidenceReadFailed, "Raw Evidence read failed")
}

func evidenceValidationStatus(issues []evidencebiz.Issue) int {
	for _, issue := range issues {
		if issue.Path == "evidences" || issue.Code == evidencebiz.IssueDuplicate {
			continue
		}
		return v1.StatusBadRequest
	}
	return v1.StatusUnprocessableEntity
}

func rawEvidenceInput(input evidenceapi.RawEvidence) evidencebiz.RawEvidence {
	return evidencebiz.RawEvidence{
		PublicationKey: input.PublicationKey, SourceID: input.SourceID, SourceName: input.SourceName,
		SourceLevel: evidencebiz.SourceLevel(input.SourceLevel), SourceURL: input.SourceURL, IsOriginal: input.IsOriginal,
		QuotedSourceID: input.QuotedSourceID, QuotedSourceName: input.QuotedSourceName,
		Title: input.Title, RawText: input.RawText, PublishedAt: input.PublishedAt,
		CollectedAt: input.CollectedAt, Keywords: append([]string(nil), input.Keywords...),
		CategoryIDs: categoryIDsInput(input.CategoryIDs),
	}
}

func categoryIDsInput(input []string) []evidencebiz.CategoryID {
	result := make([]evidencebiz.CategoryID, len(input))
	for index, categoryID := range input {
		result[index] = evidencebiz.CategoryID(categoryID)
	}
	return result
}

func rawEvidenceReadDTO(input evidencebiz.StoredRawEvidence) evidenceapi.RawEvidenceRead {
	categories := make([]evidenceapi.EvidenceCategory, len(input.Categories))
	for index, category := range input.Categories {
		categories[index] = evidenceapi.EvidenceCategory{
			ID: string(category.ID), Code: category.Code, Name: category.Name, Description: category.Description,
		}
	}
	return evidenceapi.RawEvidenceRead{
		RawEvidenceID: input.RawEvidenceID, SourceID: input.SourceID, SourceName: input.SourceName,
		SourceLevel: string(input.SourceLevel), SourceURL: input.SourceURL, IsOriginal: input.IsOriginal,
		QuotedSourceID: input.QuotedSourceID, QuotedSourceName: input.QuotedSourceName,
		Title: input.Title, RawText: input.RawText, PublishedAt: input.PublishedAt,
		CollectedAt: input.CollectedAt, Keywords: append([]string(nil), input.Keywords...), Categories: categories,
	}
}

func evidenceInput(input evidenceapi.AtomicEvidence) evidencebiz.Evidence {
	return evidencebiz.Evidence{
		SplitOrder: input.SplitOrder, LayerType: evidencebiz.LayerType(input.LayerType),
		SourceWho: input.SourceWho, SourceWhat: input.SourceWhat, SourceWhen: input.SourceWhen,
		SourceWhenRaw: input.SourceWhenRaw, SourceWhere: input.SourceWhere, SourceWhy: input.SourceWhy,
		SourceHow: input.SourceHow, SourceWhoCore: input.SourceWhoCore, SourceWhatCore: input.SourceWhatCore,
		SourceWhenCore: input.SourceWhenCore, SourceWhenRawCore: input.SourceWhenRawCore,
		SourceWhereCore: input.SourceWhereCore, SourceWhyCore: input.SourceWhyCore, SourceHowCore: input.SourceHowCore,
		ExpressionFingerprint: input.ExpressionFingerprint, ExpressionKey: input.ExpressionKey,
		FingerprintVersion: input.FingerprintVersion,
	}
}

func rawEvidenceResultDTO(result evidencebiz.RawEvidenceResult) evidenceapi.RawEvidencePublicationResult {
	return evidenceapi.RawEvidencePublicationResult{RawEvidenceID: result.RawEvidenceID}
}

func evidenceResultDTO(result evidencebiz.EvidenceResult) evidenceapi.EvidencePublicationResult {
	return evidenceapi.EvidencePublicationResult{
		RawEvidenceID: result.RawEvidenceID,
		EvidenceIDs:   append([]string(nil), result.EvidenceIDs...),
	}
}

func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}

func publicErrorWithDetails(status int, code, message string, details any) error {
	return v1.NewPublicError(status, code, message, details)
}

var _ evidenceapi.Service = (*Service)(nil)
