package industry

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	industryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/industry"
	industrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/industry"
)

type UseCase interface {
	Create(context.Context, industrybiz.Industry) (industrybiz.Industry, error)
	List(context.Context) ([]industrybiz.Industry, error)
	Get(context.Context, string) (industrybiz.Industry, error)
	Update(context.Context, string, industrybiz.Update) (industrybiz.Industry, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, fmt.Errorf("Industry use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) Create(ctx context.Context, request *industryapi.CreateRequest) (*v1.Response[industryapi.Industry], error) {
	result, err := s.useCase.Create(ctx, industrybiz.Industry{
		Name: request.Name, Aliases: request.Aliases,
		ClassificationSystem: request.ClassificationSystem, IndustryCode: request.IndustryCode,
		ParentIndustryID: request.ParentIndustryID, HierarchyPathCodes: request.HierarchyPathCodes,
		Definition: request.Definition, ReviewStatus: industrybiz.ReviewStatus(request.ReviewStatus),
	})
	return industryResponse(result, err, v1.StatusCreated)
}

func (s *Service) List(ctx context.Context, _ *industryapi.ListRequest) (*v1.Response[industryapi.IndustryList], error) {
	result, err := s.useCase.List(ctx)
	if err != nil {
		return nil, industryError(err)
	}
	items := make([]industryapi.Industry, len(result))
	for index, item := range result {
		items[index] = industryDTO(item)
	}
	return &v1.Response[industryapi.IndustryList]{Status: v1.StatusOK, Result: industryapi.IndustryList{Items: items}}, nil
}

func (s *Service) Get(ctx context.Context, request *industryapi.GetRequest) (*v1.Response[industryapi.Industry], error) {
	result, err := s.useCase.Get(ctx, request.IndustryID)
	return industryResponse(result, err, v1.StatusOK)
}

func (s *Service) Update(ctx context.Context, request *industryapi.UpdateRequest) (*v1.Response[industryapi.Industry], error) {
	result, err := s.useCase.Update(ctx, request.IndustryID, industrybiz.Update{
		Name: request.Name, Aliases: request.Aliases, ParentIndustryID: request.ParentIndustryID,
		HierarchyPathCodes: request.HierarchyPathCodes, Definition: request.Definition,
		ReviewStatus: industrybiz.ReviewStatus(request.ReviewStatus),
	})
	return industryResponse(result, err, v1.StatusOK)
}

func industryResponse(result industrybiz.Industry, err error, status int) (*v1.Response[industryapi.Industry], error) {
	if err != nil {
		return nil, industryError(err)
	}
	return &v1.Response[industryapi.Industry]{Status: status, Result: industryDTO(result)}, nil
}

func industryError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, "INDUSTRY_TIMEOUT", "Industry operation exceeded its execution budget", nil)
	}
	var validation *industrybiz.ValidationError
	if errors.As(err, &validation) {
		return v1.NewPublicError(v1.StatusUnprocessableEntity, "INDUSTRY_INVALID", "Industry data is invalid", map[string]any{"field": validation.Field, "message": validation.Message})
	}
	var reference *industrybiz.ReferenceError
	if errors.As(err, &reference) {
		return v1.NewPublicError(v1.StatusUnprocessableEntity, "INDUSTRY_REFERENCE_INVALID", "Industry references unavailable data", map[string]any{"field": reference.Field, "message": reference.Message})
	}
	if errors.Is(err, industrybiz.ErrNotFound) {
		return v1.NewPublicError(v1.StatusNotFound, "INDUSTRY_NOT_FOUND", "Industry was not found", nil)
	}
	if errors.Is(err, industrybiz.ErrConflict) {
		return v1.NewPublicError(v1.StatusConflict, "INDUSTRY_CONFLICT", "Industry identity conflicts with stored data", nil)
	}
	if errors.Is(err, industrybiz.ErrPersistence) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, "INDUSTRY_PERSISTENCE_FAILED", "Industry persistence is unavailable", nil)
	}
	return v1.NewPublicError(v1.StatusInternalServerError, "INDUSTRY_FAILED", "Industry operation failed", nil)
}

func industryDTO(input industrybiz.Industry) industryapi.Industry {
	aliases := append([]string(nil), input.Aliases...)
	if aliases == nil {
		aliases = []string{}
	}
	path := append([]string(nil), input.HierarchyPathCodes...)
	if path == nil {
		path = []string{}
	}
	return industryapi.Industry{
		ID: input.ID, Name: input.Name, Aliases: aliases,
		ClassificationSystem: input.ClassificationSystem, IndustryCode: input.IndustryCode,
		ParentIndustryID: input.ParentIndustryID, HierarchyPathCodes: path,
		Definition: input.Definition, ReviewStatus: string(input.ReviewStatus),
		CreatedAt: input.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: input.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

var _ industryapi.Service = (*Service)(nil)
