package organization

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	organizationapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/organization"
	organizationbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/organization"
)

type UseCase interface {
	Create(context.Context, organizationbiz.Organization) (organizationbiz.Organization, error)
	List(context.Context, organizationbiz.Filter) ([]organizationbiz.Organization, error)
	Get(context.Context, string) (organizationbiz.Organization, error)
	Update(context.Context, string, organizationbiz.Update) (organizationbiz.Organization, error)
	ReplaceDomainTags(context.Context, string, []string) (organizationbiz.Organization, error)
	Catalog(context.Context) (organizationbiz.Catalog, error)
	ListMembers(context.Context, string, *time.Time) ([]organizationbiz.Member, error)
	CreateMember(context.Context, organizationbiz.Member) (organizationbiz.Member, error)
	UpdateMember(context.Context, string, string, organizationbiz.Member) (organizationbiz.Member, error)
	DeleteMember(context.Context, string, string) error
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, fmt.Errorf("Organization use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) Create(ctx context.Context, request *organizationapi.CreateRequest) (*v1.Response[organizationapi.Organization], error) {
	var establishedDate *time.Time
	if request.EstablishedDate != nil {
		establishedDate = &request.EstablishedDate.Time
	}
	result, err := s.useCase.Create(ctx, organizationbiz.Organization{
		Code: request.Code, Name: request.Name, NameEn: request.NameEn,
		RegionID: request.RegionID, Category: organizationbiz.CatalogTerm{Code: request.CategoryCode},
		Function: organizationbiz.CatalogTerm{Code: request.FunctionCode}, LegalEntityCode: request.LegalEntityCode,
		DominantPartyID: request.DominantPartyID, BindingPowerLevel: request.BindingPowerLevel,
		InfluenceRating: request.InfluenceRating, StrategicPositioning: request.StrategicPositioning,
		CoreImpactScope: request.CoreImpactScope, FoundingDocument: request.FoundingDocument,
		EstablishedDate: establishedDate, HeadquartersCity: request.HeadquartersCity,
		HeadquartersCountryID:     request.HeadquartersCountryID,
		HeadquartersSubdivisionID: request.HeadquartersSubdivisionID, Description: request.Description,
	})
	return organizationResponse(result, err, v1.StatusCreated)
}

func (s *Service) List(ctx context.Context, request *organizationapi.ListRequest) (*v1.Response[organizationapi.OrganizationList], error) {
	var asOfDate *time.Time
	if request.AsOf != "" {
		parsed, err := time.Parse("2006-01-02", request.AsOf)
		if err != nil {
			return nil, organizationError(&organizationbiz.ValidationError{Field: "as_of", Message: "must use YYYY-MM-DD"})
		}
		asOfDate = &parsed
	}
	result, err := s.useCase.List(ctx, organizationbiz.Filter{
		CategoryCode: request.CategoryCode, FunctionCode: request.FunctionCode, RegionID: request.RegionID,
		CountryID: request.CountryID, AsOfDate: asOfDate,
	})
	if err != nil {
		return nil, organizationError(err)
	}
	items := make([]organizationapi.Organization, len(result))
	for index, item := range result {
		items[index] = organizationDTO(item)
	}
	return &v1.Response[organizationapi.OrganizationList]{Status: v1.StatusOK, Result: organizationapi.OrganizationList{Items: items}}, nil
}

func (s *Service) Get(ctx context.Context, request *organizationapi.GetRequest) (*v1.Response[organizationapi.Organization], error) {
	result, err := s.useCase.Get(ctx, request.OrganizationID)
	return organizationResponse(result, err, v1.StatusOK)
}

func (s *Service) Update(ctx context.Context, request *organizationapi.UpdateRequest) (*v1.Response[organizationapi.Organization], error) {
	result, err := s.useCase.Update(ctx, request.OrganizationID, requestUpdate(request))
	return organizationResponse(result, err, v1.StatusOK)
}

func requestUpdate(request *organizationapi.UpdateRequest) organizationbiz.Update {
	var establishedDate *time.Time
	if request.EstablishedDate != nil {
		establishedDate = &request.EstablishedDate.Time
	}
	return organizationbiz.Update{
		Name: request.Name, NameEn: request.NameEn, RegionID: request.RegionID,
		CategoryCode: request.CategoryCode, FunctionCode: request.FunctionCode,
		LegalEntityCode: request.LegalEntityCode, DominantPartyID: request.DominantPartyID,
		BindingPowerLevel: request.BindingPowerLevel, InfluenceRating: request.InfluenceRating,
		StrategicPositioning: request.StrategicPositioning, CoreImpactScope: request.CoreImpactScope,
		FoundingDocument: request.FoundingDocument, EstablishedDate: establishedDate,
		HeadquartersCity: request.HeadquartersCity, HeadquartersCountryID: request.HeadquartersCountryID,
		HeadquartersSubdivisionID: request.HeadquartersSubdivisionID, Description: request.Description,
		DomainTagCodes: request.DomainTagCodes,
	}
}

func (s *Service) ReplaceDomainTags(ctx context.Context, request *organizationapi.ReplaceDomainTagsRequest) (*v1.Response[organizationapi.Organization], error) {
	result, err := s.useCase.ReplaceDomainTags(ctx, request.OrganizationID, request.DomainTagCodes)
	return organizationResponse(result, err, v1.StatusOK)
}

func (s *Service) GetCatalog(ctx context.Context, _ *organizationapi.CatalogRequest) (*v1.Response[organizationapi.Catalog], error) {
	result, err := s.useCase.Catalog(ctx)
	if err != nil {
		return nil, organizationError(err)
	}
	categories := make([]organizationapi.CatalogTerm, len(result.Categories))
	for i, item := range result.Categories {
		categories[i] = organizationapi.CatalogTerm(item)
	}
	functions := make([]organizationapi.CatalogTerm, len(result.Functions))
	for i, item := range result.Functions {
		functions[i] = organizationapi.CatalogTerm(item)
	}
	tags := make([]organizationapi.DomainTag, len(result.DomainTags))
	for i, item := range result.DomainTags {
		tags[i] = organizationapi.DomainTag{Code: item.Code, FunctionCode: item.FunctionCode, NameZh: item.NameZh}
	}
	return &v1.Response[organizationapi.Catalog]{Status: v1.StatusOK, Result: organizationapi.Catalog{Categories: categories, Functions: functions, DomainTags: tags}}, nil
}

func (s *Service) ListMembers(ctx context.Context, request *organizationapi.ListMembersRequest) (*v1.Response[organizationapi.MemberList], error) {
	var asOf *time.Time
	if request.AsOf != "" {
		parsed, err := time.Parse("2006-01-02", request.AsOf)
		if err != nil {
			return nil, organizationError(&organizationbiz.ValidationError{Field: "as_of", Message: "must use YYYY-MM-DD"})
		}
		asOf = &parsed
	}
	result, err := s.useCase.ListMembers(ctx, request.OrganizationID, asOf)
	if err != nil {
		return nil, organizationError(err)
	}
	items := make([]organizationapi.Member, len(result))
	for i, item := range result {
		items[i] = memberDTO(item)
	}
	return &v1.Response[organizationapi.MemberList]{Status: v1.StatusOK, Result: organizationapi.MemberList{Items: items}}, nil
}

func (s *Service) CreateMember(ctx context.Context, request *organizationapi.CreateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	result, err := s.useCase.CreateMember(ctx, memberInput(request.OrganizationID, request.CountryID, request.MembershipType, request.EffectiveDate, request.ExpiryDate))
	if err != nil {
		return nil, organizationError(err)
	}
	return &v1.Response[organizationapi.Member]{Status: v1.StatusCreated, Result: memberDTO(result)}, nil
}

func (s *Service) UpdateMember(ctx context.Context, request *organizationapi.UpdateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	result, err := s.useCase.UpdateMember(ctx, request.OrganizationID, request.MemberID, memberInput(request.OrganizationID, request.CountryID, request.MembershipType, request.EffectiveDate, request.ExpiryDate))
	if err != nil {
		return nil, organizationError(err)
	}
	return &v1.Response[organizationapi.Member]{Status: v1.StatusOK, Result: memberDTO(result)}, nil
}

func (s *Service) DeleteMember(ctx context.Context, request *organizationapi.DeleteMemberRequest) (*v1.Response[organizationapi.DeleteResult], error) {
	if err := s.useCase.DeleteMember(ctx, request.OrganizationID, request.MemberID); err != nil {
		return nil, organizationError(err)
	}
	return &v1.Response[organizationapi.DeleteResult]{Status: v1.StatusOK, Result: organizationapi.DeleteResult{Deleted: true}}, nil
}

func memberInput(organizationID, countryID, membershipType string, effective, expiry *organizationapi.Date) organizationbiz.Member {
	result := organizationbiz.Member{OrganizationID: organizationID, CountryID: countryID, MembershipType: membershipType}
	if effective != nil {
		result.EffectiveDate = &effective.Time
	}
	if expiry != nil {
		result.ExpiryDate = &expiry.Time
	}
	return result
}

func memberDTO(input organizationbiz.Member) organizationapi.Member {
	return organizationapi.Member{ID: input.ID, OrganizationID: input.OrganizationID, CountryID: input.CountryID, MembershipType: input.MembershipType,
		EffectiveDate: formatDate(input.EffectiveDate), ExpiryDate: formatDate(input.ExpiryDate), CreatedAt: input.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: input.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func formatDate(input *time.Time) *string {
	if input == nil {
		return nil
	}
	value := input.Format("2006-01-02")
	return &value
}

func organizationResponse(result organizationbiz.Organization, err error, status int) (*v1.Response[organizationapi.Organization], error) {
	if err != nil {
		return nil, organizationError(err)
	}
	return &v1.Response[organizationapi.Organization]{Status: status, Result: organizationDTO(result)}, nil
}

func organizationError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, "ORGANIZATION_TIMEOUT", "Organization operation exceeded its execution budget", nil)
	}
	var validation *organizationbiz.ValidationError
	if errors.As(err, &validation) {
		return v1.NewPublicError(v1.StatusUnprocessableEntity, "ORGANIZATION_INVALID", "Organization data is invalid", map[string]any{"field": validation.Field, "message": validation.Message})
	}
	var reference *organizationbiz.ReferenceError
	if errors.As(err, &reference) {
		return v1.NewPublicError(v1.StatusUnprocessableEntity, "ORGANIZATION_REFERENCE_INVALID", "Organization references unavailable data", map[string]any{"field": reference.Field, "message": reference.Message})
	}
	if errors.Is(err, organizationbiz.ErrNotFound) {
		return v1.NewPublicError(v1.StatusNotFound, "ORGANIZATION_NOT_FOUND", "Organization was not found", nil)
	}
	if errors.Is(err, organizationbiz.ErrConflict) {
		return v1.NewPublicError(v1.StatusConflict, "ORGANIZATION_CONFLICT", "Organization identity conflicts with stored data", nil)
	}
	return v1.NewPublicError(v1.StatusInternalServerError, "ORGANIZATION_FAILED", "Organization operation failed", nil)
}

func organizationDTO(input organizationbiz.Organization) organizationapi.Organization {
	var establishedDate *string
	if input.EstablishedDate != nil {
		value := input.EstablishedDate.Format("2006-01-02")
		establishedDate = &value
	}
	return organizationapi.Organization{
		ID: input.ID, Code: input.Code, Name: input.Name, NameEn: input.NameEn, RegionID: input.RegionID,
		Category: organizationapi.CatalogTerm(input.Category), Function: organizationapi.CatalogTerm(input.Function),
		LegalEntityCode: input.LegalEntityCode, DominantPartyID: input.DominantPartyID,
		BindingPowerLevel: input.BindingPowerLevel, InfluenceRating: input.InfluenceRating,
		StrategicPositioning: input.StrategicPositioning, CoreImpactScope: input.CoreImpactScope,
		FoundingDocument: input.FoundingDocument, EstablishedDate: establishedDate,
		HeadquartersCity: input.HeadquartersCity, HeadquartersCountryID: input.HeadquartersCountryID,
		HeadquartersSubdivisionID: input.HeadquartersSubdivisionID, Description: input.Description,
		DomainTags: domainTagDTOs(input.DomainTags),
		CreatedAt:  input.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: input.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func domainTagDTOs(input []organizationbiz.DomainTag) []organizationapi.DomainTag {
	result := make([]organizationapi.DomainTag, len(input))
	for i, item := range input {
		result[i] = organizationapi.DomainTag{Code: item.Code, FunctionCode: item.FunctionCode, NameZh: item.NameZh}
	}
	return result
}

var _ organizationapi.Service = (*Service)(nil)
