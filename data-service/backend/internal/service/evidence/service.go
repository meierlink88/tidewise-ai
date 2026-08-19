package evidence

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	evidenceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/evidence"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
)

type UseCase interface {
	PublishRawEvidence(context.Context, evidencebiz.RawEvidence) (evidencebiz.RawEvidenceResult, error)
	GetRawEvidence(context.Context, string) (evidencebiz.StoredRawEvidence, error)
	PublishEvidence(context.Context, string, []evidencebiz.Evidence) (evidencebiz.EvidenceResult, error)
	ListCategories(context.Context) (evidencebiz.CategoryCatalog, error)
	ListEvidence(context.Context, evidencebiz.EvidenceListFilter) (evidencebiz.EvidencePage, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Evidence use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) ListEvidence(ctx context.Context, request *evidenceapi.ListRequest) (*v1.Response[evidenceapi.Page], error) {
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, evidenceapi.ErrorDataServiceNotReady, "Evidence list service is unavailable")
	}
	filter, err := evidenceListFilter(request)
	if err != nil {
		return nil, err
	}
	result, err := s.useCase.ListEvidence(ctx, filter)
	if err != nil {
		return nil, evidenceListError(err)
	}
	items := make([]evidenceapi.ListItem, len(result.Items))
	for index, item := range result.Items {
		categories := make([]evidenceapi.EvidenceCategory, len(item.Categories))
		for categoryIndex, category := range item.Categories {
			categories[categoryIndex] = evidenceapi.EvidenceCategory{
				ID: string(category.ID), Code: category.Code, Name: category.Name, Description: category.Description,
			}
		}
		items[index] = evidenceapi.ListItem{
			ID: item.ID, RawEvidenceID: item.RawEvidenceID, Title: item.Title, Summary: item.Summary,
			Categories: categories, SourceName: item.SourceName, SourceLevel: string(item.SourceLevel),
			IsSplit: item.IsSplit, PublishedAt: formatOptionalListTime(item.PublishedAt),
			CollectedAt: item.CollectedAt.UTC().Format(time.RFC3339Nano),
		}
	}
	return &v1.Response[evidenceapi.Page]{Status: v1.StatusOK, Result: evidenceapi.Page{
		Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize,
	}}, nil
}

func evidenceListFilter(request *evidenceapi.ListRequest) (evidencebiz.EvidenceListFilter, error) {
	if request == nil {
		return evidencebiz.EvidenceListFilter{}, publicError(v1.StatusBadRequest, evidenceapi.ErrorInvalidRequest, "Evidence query is required")
	}
	page, err := v1.ParseBoundedInt(request.Page, 1, 1, 1_000_000, "page")
	if err != nil {
		return evidencebiz.EvidenceListFilter{}, err
	}
	pageSize, err := v1.ParseBoundedInt(request.PageSize, 50, 1, 100, "page_size")
	if err != nil {
		return evidencebiz.EvidenceListFilter{}, err
	}
	filter := evidencebiz.EvidenceListFilter{
		Title: strings.TrimSpace(request.Title), Summary: strings.TrimSpace(request.Summary),
		CategoryID: evidencebiz.CategoryID(strings.TrimSpace(request.CategoryID)),
		SourceName: strings.TrimSpace(request.SourceName), SourceLevel: evidencebiz.SourceLevel(strings.TrimSpace(request.SourceLevel)),
		Page: page, PageSize: pageSize,
	}
	if raw := strings.TrimSpace(request.IsSplit); raw != "" {
		if raw != "true" && raw != "false" {
			return evidencebiz.EvidenceListFilter{}, publicError(v1.StatusBadRequest, evidenceapi.ErrorInvalidRequest, "is_split must be true or false")
		}
		value, _ := strconv.ParseBool(raw)
		filter.IsSplit = &value
	}
	for _, value := range []struct {
		name   string
		raw    string
		target **time.Time
	}{
		{name: "published_from", raw: request.PublishedFrom, target: &filter.PublishedFrom},
		{name: "published_to", raw: request.PublishedTo, target: &filter.PublishedTo},
		{name: "collected_from", raw: request.CollectedFrom, target: &filter.CollectedFrom},
		{name: "collected_to", raw: request.CollectedTo, target: &filter.CollectedTo},
	} {
		*value.target, err = optionalListTime(value.raw)
		if err != nil {
			return evidencebiz.EvidenceListFilter{}, publicError(v1.StatusBadRequest, evidenceapi.ErrorInvalidRequest, value.name+" must be a UTC RFC3339 timestamp")
		}
	}
	return filter, nil
}

func optionalListTime(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil || value.Location() != time.UTC {
		return nil, errors.New("timestamp must use UTC RFC3339")
	}
	value = value.UTC()
	return &value, nil
}

func formatOptionalListTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func evidenceListError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return publicError(v1.StatusServiceUnavailable, evidenceapi.ErrorEvidenceListTimeout, "Evidence list execution budget exceeded")
	}
	var validation *evidencebiz.ValidationError
	if errors.As(err, &validation) {
		return publicErrorWithDetails(v1.StatusBadRequest, evidenceapi.ErrorInvalidRequest, "Evidence query is invalid", map[string]any{"issues": validation.Issues})
	}
	return publicError(v1.StatusInternalServerError, evidenceapi.ErrorEvidenceListFailed, "Evidence list is unavailable")
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
	result, err := s.useCase.GetRawEvidence(ctx, request.ID)
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
		ID: input.ID, SourceID: input.SourceID, SourceName: input.SourceName,
		SourceLevel: string(input.SourceLevel), SourceURL: input.SourceURL, IsOriginal: input.IsOriginal,
		QuotedSourceID: input.QuotedSourceID, QuotedSourceName: input.QuotedSourceName,
		Title: input.Title, RawText: input.RawText, PublishedAt: input.PublishedAt,
		CollectedAt: input.CollectedAt, Keywords: append([]string(nil), input.Keywords...), Categories: categories,
	}
}

func evidenceInput(input evidenceapi.AtomicEvidence) evidencebiz.Evidence {
	return evidencebiz.Evidence{
		Summary: input.Summary,
		Semantic: evidencebiz.Semantic{
			Who: input.Semantic.Who, What: input.Semantic.What, When: input.Semantic.When,
			Where: input.Semantic.Where, Why: input.Semantic.Why, How: input.Semantic.How,
		},
	}
}

func rawEvidenceResultDTO(result evidencebiz.RawEvidenceResult) evidenceapi.RawEvidencePublicationResult {
	return evidenceapi.RawEvidencePublicationResult{ID: result.ID}
}

func evidenceResultDTO(result evidencebiz.EvidenceResult) evidenceapi.EvidencePublicationResult {
	return evidenceapi.EvidencePublicationResult{
		RawEvidenceID: result.RawEvidenceID,
		IDs:           append([]string(nil), result.IDs...),
	}
}

func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}

func publicErrorWithDetails(status int, code, message string, details any) error {
	return v1.NewPublicError(status, code, message, details)
}

var _ evidenceapi.Service = (*Service)(nil)
