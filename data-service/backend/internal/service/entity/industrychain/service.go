package industrychain

import (
	"context"
	"errors"
	"fmt"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	industrychainapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/industrychain"
	industrychainbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/industrychain"
	"time"
)

type UseCase interface {
	Create(context.Context, industrychainbiz.IndustryChain) (industrychainbiz.IndustryChain, error)
	List(context.Context, industrychainbiz.ListRequest) (industrychainbiz.Page, error)
	Get(context.Context, industrychainbiz.ID) (industrychainbiz.IndustryChain, error)
	Update(context.Context, industrychainbiz.ID, industrychainbiz.Update) (industrychainbiz.IndustryChain, error)
}
type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, fmt.Errorf("IndustryChain use case is required")
	}
	return &Service{useCase: useCase}, nil
}
func (s *Service) Create(ctx context.Context, request *industrychainapi.CreateRequest) (*v1.Response[industrychainapi.IndustryChain], error) {
	date, err := parseDate(request.AsOfDate)
	if err != nil {
		return nil, err
	}
	result, callErr := s.useCase.Create(ctx, industrychainbiz.IndustryChain{Name: request.Name, Aliases: request.Aliases, Scope: request.Scope, TargetOutput: request.TargetOutput, EndUse: request.EndUse, Geography: request.Geography, PrimaryCountryID: request.PrimaryCountryID, AsOfDate: date, ReviewStatus: industrychainbiz.ReviewStatus(request.ReviewStatus), ReviewNote: request.ReviewNote, TechnologyRouteQualifier: request.TechnologyRouteQualifier, ObservableVariables: request.ObservableVariables})
	return industryChainResponse(result, callErr, v1.StatusCreated)
}
func (s *Service) List(ctx context.Context, request *industrychainapi.ListRequest) (*v1.Response[industrychainapi.IndustryChainList], error) {
	pageSize, err := v1.ParseBoundedInt(request.PageSize, 50, 1, 100, "page_size")
	if err != nil {
		return nil, err
	}
	result, err := s.useCase.List(ctx, industrychainbiz.ListRequest{PageSize: pageSize, Cursor: request.Cursor})
	if err != nil {
		return nil, industryChainError(err)
	}
	items := make([]industrychainapi.IndustryChain, len(result.Items))
	for index, item := range result.Items {
		items[index] = industryChainDTO(item)
	}
	return &v1.Response[industrychainapi.IndustryChainList]{Status: v1.StatusOK, Result: industrychainapi.IndustryChainList{Items: items, NextCursor: result.NextCursor}}, nil
}
func (s *Service) Get(ctx context.Context, request *industrychainapi.GetRequest) (*v1.Response[industrychainapi.IndustryChain], error) {
	result, err := s.useCase.Get(ctx, industrychainbiz.ID(request.IndustryChainID))
	return industryChainResponse(result, err, v1.StatusOK)
}
func (s *Service) Update(ctx context.Context, request *industrychainapi.UpdateRequest) (*v1.Response[industrychainapi.IndustryChain], error) {
	date, err := parseDate(request.AsOfDate)
	if err != nil {
		return nil, err
	}
	result, callErr := s.useCase.Update(ctx, industrychainbiz.ID(request.IndustryChainID), industrychainbiz.Update{Name: request.Name, Aliases: request.Aliases, Scope: request.Scope, TargetOutput: request.TargetOutput, EndUse: request.EndUse, Geography: request.Geography, PrimaryCountryID: request.PrimaryCountryID, AsOfDate: date, ReviewStatus: industrychainbiz.ReviewStatus(request.ReviewStatus), ReviewNote: request.ReviewNote, TechnologyRouteQualifier: request.TechnologyRouteQualifier, ObservableVariables: request.ObservableVariables})
	return industryChainResponse(result, callErr, v1.StatusOK)
}
func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, v1.NewPublicError(v1.StatusUnprocessableEntity, industrychainapi.ErrorInvalid, "IndustryChain data is invalid", map[string]any{"field": "as_of_date", "message": "must use YYYY-MM-DD"})
	}
	return parsed, nil
}
func industryChainResponse(result industrychainbiz.IndustryChain, err error, status int) (*v1.Response[industrychainapi.IndustryChain], error) {
	if err != nil {
		return nil, industryChainError(err)
	}
	return &v1.Response[industrychainapi.IndustryChain]{Status: status, Result: industryChainDTO(result)}, nil
}
func industryChainError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, industrychainapi.ErrorTimeout, "IndustryChain operation exceeded its execution budget", nil)
	}
	var validation *industrychainbiz.ValidationError
	if errors.As(err, &validation) {
		if validation.Field == "cursor" {
			return v1.NewPublicError(v1.StatusBadRequest, industrychainapi.ErrorInvalidRequest, "IndustryChain list cursor is invalid", nil)
		}
		return v1.NewPublicError(v1.StatusUnprocessableEntity, industrychainapi.ErrorInvalid, "IndustryChain data is invalid", map[string]any{"field": validation.Field, "message": validation.Message})
	}
	var reference *industrychainbiz.ReferenceError
	if errors.As(err, &reference) {
		return v1.NewPublicError(v1.StatusUnprocessableEntity, industrychainapi.ErrorReferenceInvalid, "IndustryChain references unavailable data", map[string]any{"field": reference.Field, "message": reference.Message})
	}
	if errors.Is(err, industrychainbiz.ErrNotFound) {
		return v1.NewPublicError(v1.StatusNotFound, industrychainapi.ErrorNotFound, "IndustryChain was not found", nil)
	}
	if errors.Is(err, industrychainbiz.ErrConflict) {
		return v1.NewPublicError(v1.StatusConflict, industrychainapi.ErrorConflict, "IndustryChain identity conflicts with stored data", nil)
	}
	if errors.Is(err, industrychainbiz.ErrPersistence) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, industrychainapi.ErrorPersistenceFailed, "IndustryChain persistence is unavailable", nil)
	}
	return v1.NewPublicError(v1.StatusInternalServerError, industrychainapi.ErrorFailed, "IndustryChain operation failed", nil)
}
func industryChainDTO(input industrychainbiz.IndustryChain) industrychainapi.IndustryChain {
	aliases := append([]string(nil), input.Aliases...)
	if aliases == nil {
		aliases = []string{}
	}
	variables := append([]string(nil), input.ObservableVariables...)
	if variables == nil {
		variables = []string{}
	}
	return industrychainapi.IndustryChain{ID: string(input.ID), Name: input.Name, Aliases: aliases, Scope: input.Scope, TargetOutput: input.TargetOutput, EndUse: input.EndUse, Geography: input.Geography, PrimaryCountryID: cloneString(input.PrimaryCountryID), AsOfDate: input.AsOfDate.Format("2006-01-02"), ReviewStatus: string(input.ReviewStatus), ReviewNote: cloneString(input.ReviewNote), TechnologyRouteQualifier: cloneString(input.TechnologyRouteQualifier), ObservableVariables: variables, CreatedAt: input.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: input.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}
func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ industrychainapi.Service = (*Service)(nil)
