package company

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	companyapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/company"
	companybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/company"
)

type UseCase interface {
	ListProjection(context.Context, companybiz.ProjectionListRequest) (companybiz.ProjectionPage, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, fmt.Errorf("Company projection use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) List(ctx context.Context, request *companyapi.ListRequest) (*v1.Response[companyapi.CompanyProjectionPage], error) {
	pageSize, err := v1.ParseBoundedInt(request.PageSize, 50, 1, 100, "page_size")
	if err != nil {
		return nil, err
	}
	result, err := s.useCase.ListProjection(ctx, companybiz.ProjectionListRequest{PageSize: pageSize, Cursor: request.Cursor})
	if err != nil {
		return nil, companyError(err)
	}
	items := make([]companyapi.Company, len(result.Items))
	for index, item := range result.Items {
		items[index] = companyDTO(item)
	}
	return &v1.Response[companyapi.CompanyProjectionPage]{
		Status: v1.StatusOK,
		Result: companyapi.CompanyProjectionPage{
			SchemaVersion: companyapi.ProjectionSchemaVersion,
			SnapshotID:    result.SnapshotID, Items: items, NextCursor: result.NextCursor,
		},
	}, nil
}

func companyError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, companyapi.ErrorTimeout, "Company projection exceeded its execution budget", nil)
	}
	var validation *companybiz.ValidationError
	if errors.As(err, &validation) {
		return v1.NewPublicError(v1.StatusBadRequest, companyapi.ErrorInvalidRequest, "Company projection request is invalid", map[string]any{"field": validation.Field, "message": validation.Message})
	}
	if errors.Is(err, companybiz.ErrProjectionSnapshotChanged) {
		return v1.NewPublicError(v1.StatusConflict, companyapi.ErrorSnapshotChanged, "Company projection snapshot changed while paging", nil)
	}
	if errors.Is(err, companybiz.ErrPersistence) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, companyapi.ErrorPersistenceFailed, "Company projection persistence is unavailable", nil)
	}
	return v1.NewPublicError(v1.StatusInternalServerError, companyapi.ErrorFailed, "Company projection failed", nil)
}

func companyDTO(input companybiz.Company) companyapi.Company {
	aliases := append([]string(nil), input.Aliases...)
	if aliases == nil {
		aliases = []string{}
	}
	links := make([]companyapi.CompanyIndustryLink, len(input.IndustryLinks))
	for index, link := range input.IndustryLinks {
		links[index] = companyapi.CompanyIndustryLink{
			ID: link.ID, CompanyID: string(link.CompanyID), IndustryID: string(link.IndustryID),
			CreatedAt: link.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
	}
	return companyapi.Company{
		ID: string(input.ID), Code: input.Code, Name: input.Name, NameEn: cloneString(input.NameEn),
		LegalName: cloneString(input.LegalName), Aliases: aliases,
		RegistrationCountryID: cloneString(input.RegistrationCountryID), OperatingArea: cloneString(input.OperatingArea),
		HeadquartersCity: cloneString(input.HeadquartersCity), FoundingDate: dateString(input.FoundingDate), IPODate: dateString(input.IPODate),
		LegalForm: cloneString(input.LegalForm), OwnershipType: ownershipString(input.OwnershipType),
		StrategicPositioning: cloneString(input.StrategicPositioning), Description: cloneString(input.Description),
		Status: string(input.Status), CreatedAt: input.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: input.UpdatedAt.UTC().Format(time.RFC3339Nano),
		IndustryLinks: links,
	}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func dateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format("2006-01-02")
	return &formatted
}

func ownershipString(value *companybiz.OwnershipType) *string {
	if value == nil {
		return nil
	}
	formatted := string(*value)
	return &formatted
}

var _ companyapi.Service = (*Service)(nil)
