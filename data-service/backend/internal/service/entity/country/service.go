package country

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	countryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/country"
	countrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/country"
)

type UseCase interface {
	Create(context.Context, countrybiz.Country) (countrybiz.Country, error)
	List(context.Context, string) ([]countrybiz.Country, error)
	Get(context.Context, string) (countrybiz.Country, error)
	Update(context.Context, string, countrybiz.Update) (countrybiz.Country, error)
	ReplaceRegions(context.Context, string, []string) (countrybiz.Country, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, fmt.Errorf("Country use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) Create(ctx context.Context, request *countryapi.CreateRequest) (*v1.Response[countryapi.Country], error) {
	result, err := s.useCase.Create(ctx, countrybiz.Country{
		ID: request.ID, Code: request.Code, Name: request.Name, NameEn: request.NameEn,
		StrategicPositioning: request.StrategicPositioning, KeyResources: request.KeyResources,
	})
	return countryResponse(result, err, v1.StatusCreated)
}

func (s *Service) List(ctx context.Context, request *countryapi.ListRequest) (*v1.Response[countryapi.CountryList], error) {
	result, err := s.useCase.List(ctx, request.RegionID)
	if err != nil {
		return nil, countryError(err)
	}
	items := make([]countryapi.Country, len(result))
	for index, item := range result {
		items[index] = countryDTO(item)
	}
	return &v1.Response[countryapi.CountryList]{Status: v1.StatusOK, Result: countryapi.CountryList{Items: items}}, nil
}

func (s *Service) Get(ctx context.Context, request *countryapi.GetRequest) (*v1.Response[countryapi.Country], error) {
	result, err := s.useCase.Get(ctx, request.CountryID)
	return countryResponse(result, err, v1.StatusOK)
}

func (s *Service) Update(ctx context.Context, request *countryapi.UpdateRequest) (*v1.Response[countryapi.Country], error) {
	result, err := s.useCase.Update(ctx, request.CountryID, countrybiz.Update{
		Name: request.Name, NameEn: request.NameEn,
		StrategicPositioning: request.StrategicPositioning, KeyResources: request.KeyResources,
	})
	return countryResponse(result, err, v1.StatusOK)
}

func (s *Service) ReplaceRegions(ctx context.Context, request *countryapi.ReplaceRegionsRequest) (*v1.Response[countryapi.Country], error) {
	result, err := s.useCase.ReplaceRegions(ctx, request.CountryID, request.RegionIDs)
	return countryResponse(result, err, v1.StatusOK)
}

func countryResponse(result countrybiz.Country, err error, status int) (*v1.Response[countryapi.Country], error) {
	if err != nil {
		return nil, countryError(err)
	}
	return &v1.Response[countryapi.Country]{Status: status, Result: countryDTO(result)}, nil
}

func countryError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, "COUNTRY_TIMEOUT", "Country operation exceeded its execution budget", nil)
	}
	var validation *countrybiz.ValidationError
	if errors.As(err, &validation) {
		return v1.NewPublicError(v1.StatusUnprocessableEntity, "COUNTRY_INVALID", "Country data is invalid", map[string]any{"field": validation.Field, "message": validation.Message})
	}
	var reference *countrybiz.ReferenceError
	if errors.As(err, &reference) {
		return v1.NewPublicError(v1.StatusUnprocessableEntity, "COUNTRY_REFERENCE_INVALID", "Country references unavailable data", map[string]any{"field": reference.Field, "message": reference.Message})
	}
	if errors.Is(err, countrybiz.ErrNotFound) {
		return v1.NewPublicError(v1.StatusNotFound, "COUNTRY_NOT_FOUND", "Country was not found", nil)
	}
	if errors.Is(err, countrybiz.ErrConflict) {
		return v1.NewPublicError(v1.StatusConflict, "COUNTRY_CONFLICT", "Country identity conflicts with stored data", nil)
	}
	return v1.NewPublicError(v1.StatusInternalServerError, "COUNTRY_FAILED", "Country operation failed", nil)
}

func countryDTO(input countrybiz.Country) countryapi.Country {
	regions := make([]countryapi.Region, len(input.Regions))
	for index, region := range input.Regions {
		regions[index] = countryapi.Region{
			ID: region.ID, Code: region.Code, Name: region.Name, NameEn: region.NameEn, RegionType: region.RegionType,
		}
	}
	return countryapi.Country{
		ID: input.ID, Code: input.Code, Name: input.Name, NameEn: input.NameEn,
		StrategicPositioning: input.StrategicPositioning, KeyResources: input.KeyResources,
		CreatedAt: input.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: input.UpdatedAt.UTC().Format(time.RFC3339Nano),
		Regions: regions,
	}
}

var _ countryapi.Service = (*Service)(nil)
