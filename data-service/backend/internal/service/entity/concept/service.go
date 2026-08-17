package concept

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	conceptapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/concept"
	conceptbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/concept"
)

type UseCase interface {
	Create(context.Context, conceptbiz.Concept) (conceptbiz.Concept, error)
	List(context.Context, conceptbiz.ListRequest) (conceptbiz.Page, error)
	Get(context.Context, conceptbiz.ID) (conceptbiz.Concept, error)
	Update(context.Context, conceptbiz.ID, conceptbiz.Update) (conceptbiz.Concept, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, fmt.Errorf("Concept use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) Create(ctx context.Context, request *conceptapi.CreateRequest) (*v1.Response[conceptapi.Concept], error) {
	result, err := s.useCase.Create(ctx, conceptbiz.Concept{
		Name: request.Name, Aliases: request.Aliases, ConceptType: conceptbiz.Type(request.ConceptType),
		Definition: request.Definition, ReviewStatus: conceptbiz.ReviewStatus(request.ReviewStatus),
	})
	return conceptResponse(result, err, v1.StatusCreated)
}

func (s *Service) List(ctx context.Context, request *conceptapi.ListRequest) (*v1.Response[conceptapi.ConceptList], error) {
	pageSize, err := v1.ParseBoundedInt(request.PageSize, 50, 1, 100, "page_size")
	if err != nil {
		return nil, err
	}
	result, err := s.useCase.List(ctx, conceptbiz.ListRequest{PageSize: pageSize, Cursor: request.Cursor})
	if err != nil {
		return nil, conceptError(err)
	}
	items := make([]conceptapi.Concept, len(result.Items))
	for index, item := range result.Items {
		items[index] = conceptDTO(item)
	}
	return &v1.Response[conceptapi.ConceptList]{Status: v1.StatusOK, Result: conceptapi.ConceptList{Items: items, NextCursor: result.NextCursor}}, nil
}

func (s *Service) Get(ctx context.Context, request *conceptapi.GetRequest) (*v1.Response[conceptapi.Concept], error) {
	result, err := s.useCase.Get(ctx, conceptbiz.ID(request.ConceptID))
	return conceptResponse(result, err, v1.StatusOK)
}

func (s *Service) Update(ctx context.Context, request *conceptapi.UpdateRequest) (*v1.Response[conceptapi.Concept], error) {
	result, err := s.useCase.Update(ctx, conceptbiz.ID(request.ConceptID), conceptbiz.Update{
		Name: request.Name, Aliases: request.Aliases, ConceptType: conceptbiz.Type(request.ConceptType),
		Definition: request.Definition, ReviewStatus: conceptbiz.ReviewStatus(request.ReviewStatus),
	})
	return conceptResponse(result, err, v1.StatusOK)
}

func conceptResponse(result conceptbiz.Concept, err error, status int) (*v1.Response[conceptapi.Concept], error) {
	if err != nil {
		return nil, conceptError(err)
	}
	return &v1.Response[conceptapi.Concept]{Status: status, Result: conceptDTO(result)}, nil
}

func conceptError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, conceptapi.ErrorTimeout, "Concept operation exceeded its execution budget", nil)
	}
	var validation *conceptbiz.ValidationError
	if errors.As(err, &validation) {
		return v1.NewPublicError(v1.StatusUnprocessableEntity, conceptapi.ErrorInvalid, "Concept data is invalid", map[string]any{"field": validation.Field, "message": validation.Message})
	}
	if errors.Is(err, conceptbiz.ErrNotFound) {
		return v1.NewPublicError(v1.StatusNotFound, conceptapi.ErrorNotFound, "Concept was not found", nil)
	}
	if errors.Is(err, conceptbiz.ErrConflict) {
		return v1.NewPublicError(v1.StatusConflict, conceptapi.ErrorConflict, "Concept identity conflicts with stored data", nil)
	}
	if errors.Is(err, conceptbiz.ErrPersistence) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, conceptapi.ErrorPersistenceFailed, "Concept persistence is unavailable", nil)
	}
	return v1.NewPublicError(v1.StatusInternalServerError, conceptapi.ErrorFailed, "Concept operation failed", nil)
}

func conceptDTO(input conceptbiz.Concept) conceptapi.Concept {
	aliases := append([]string(nil), input.Aliases...)
	if aliases == nil {
		aliases = []string{}
	}
	return conceptapi.Concept{
		ID: string(input.ID), Name: input.Name, Aliases: aliases, ConceptType: string(input.ConceptType),
		Definition: input.Definition, ReviewStatus: string(input.ReviewStatus),
		CreatedAt: input.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: input.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

var _ conceptapi.Service = (*Service)(nil)
