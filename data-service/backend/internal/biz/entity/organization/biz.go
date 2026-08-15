package organization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	entitybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
)

var (
	ErrNotFound    = errors.New("Organization not found")
	ErrConflict    = errors.New("Organization conflict")
	ErrPersistence = errors.New("Organization persistence failed")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

type ReferenceError struct {
	Field   string
	Message string
}

func (e *ReferenceError) Error() string { return e.Field + ": " + e.Message }

type CatalogTerm struct {
	Code   string `json:"code"`
	NameZh string `json:"name_zh"`
}

type Organization struct {
	ID                        string
	Code                      string
	Name                      string
	NameEn                    string
	RegionID                  *string
	Category                  CatalogTerm
	Function                  CatalogTerm
	LegalEntityCode           *string
	DominantPartyID           *string
	BindingPowerLevel         *string
	InfluenceRating           *string
	StrategicPositioning      *string
	CoreImpactScope           *string
	FoundingDocument          *string
	EstablishedDate           *time.Time
	HeadquartersCity          *string
	HeadquartersCountryID     *string
	HeadquartersSubdivisionID *string
	Description               *string
	DomainTags                []DomainTag
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type Filter struct {
	CategoryCode string
	FunctionCode string
	RegionID     string
	CountryID    string
	AsOfDate     *time.Time
}

type Update struct {
	Name                      string
	NameEn                    string
	RegionID                  *string
	CategoryCode              string
	FunctionCode              string
	LegalEntityCode           *string
	DominantPartyID           *string
	BindingPowerLevel         *string
	InfluenceRating           *string
	StrategicPositioning      *string
	CoreImpactScope           *string
	FoundingDocument          *string
	EstablishedDate           *time.Time
	HeadquartersCity          *string
	HeadquartersCountryID     *string
	HeadquartersSubdivisionID *string
	Description               *string
	DomainTagCodes            *[]string
}

type Catalog struct {
	Categories []CatalogTerm
	Functions  []CatalogTerm
	DomainTags []DomainTag
}

type DomainTag struct {
	Code         string `json:"code"`
	FunctionCode string `json:"function_code"`
	NameZh       string `json:"name_zh"`
}

type Member struct {
	ID             int64
	OrganizationID string
	CountryID      string
	MembershipType string
	EffectiveDate  *time.Time
	ExpiryDate     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Repository interface {
	Create(context.Context, Organization) (Organization, error)
	Get(context.Context, string) (Organization, error)
	List(context.Context, Filter) ([]Organization, error)
	Update(context.Context, string, Update) (Organization, error)
	ReplaceDomainTags(context.Context, string, []string) (Organization, error)
	Catalog(context.Context) (Catalog, error)
	ListMembers(context.Context, string, *time.Time) ([]Member, error)
	CreateMember(context.Context, Member) (Member, error)
	UpdateMember(context.Context, string, int64, Member) (Member, error)
	DeleteMember(context.Context, string, int64) error
}

type UseCase struct{ repository Repository }

func NewUseCase(repository Repository) (*UseCase, error) {
	if repository == nil {
		return nil, errors.New("Organization repository is required")
	}
	return &UseCase{repository: repository}, nil
}

func (s *UseCase) Create(ctx context.Context, input Organization) (Organization, error) {
	if err := validateOrganization(input); err != nil {
		return Organization{}, err
	}
	return s.repository.Create(ctx, cloneOrganization(input))
}

func (s *UseCase) Get(ctx context.Context, id string) (Organization, error) {
	if !entitybiz.IsOrganizationID(id) {
		return Organization{}, &ValidationError{Field: "organization_id", Message: "must be a stable Organization ID"}
	}
	return s.repository.Get(ctx, id)
}

func (s *UseCase) List(ctx context.Context, filter Filter) ([]Organization, error) {
	for _, item := range []struct {
		field string
		value string
	}{{"category_code", filter.CategoryCode}, {"function_code", filter.FunctionCode}} {
		if item.value != "" && !validCode(item.value, 50) {
			return nil, &ValidationError{Field: item.field, Message: "must be an uppercase stable code"}
		}
	}
	if filter.RegionID != "" && !entitybiz.IsRegionID(filter.RegionID) {
		return nil, &ValidationError{Field: "region_id", Message: "must be a stable Region ID"}
	}
	if filter.CountryID != "" && !entitybiz.IsCountryID(filter.CountryID) {
		return nil, &ValidationError{Field: "country_id", Message: "must be a stable Country ID"}
	}
	return s.repository.List(ctx, filter)
}

func (s *UseCase) Update(ctx context.Context, id string, input Update) (Organization, error) {
	probe := Organization{
		ID: id, Code: "UNCHANGED", Name: input.Name, NameEn: input.NameEn,
		RegionID: input.RegionID, Category: CatalogTerm{Code: input.CategoryCode}, Function: CatalogTerm{Code: input.FunctionCode},
		LegalEntityCode: input.LegalEntityCode, DominantPartyID: input.DominantPartyID, BindingPowerLevel: input.BindingPowerLevel,
		InfluenceRating: input.InfluenceRating, StrategicPositioning: input.StrategicPositioning, CoreImpactScope: input.CoreImpactScope,
		FoundingDocument: input.FoundingDocument, EstablishedDate: input.EstablishedDate, HeadquartersCity: input.HeadquartersCity,
		HeadquartersCountryID: input.HeadquartersCountryID, HeadquartersSubdivisionID: input.HeadquartersSubdivisionID, Description: input.Description,
	}
	if err := validateOrganization(probe); err != nil {
		return Organization{}, err
	}
	if input.DomainTagCodes != nil {
		seen := map[string]struct{}{}
		for _, code := range *input.DomainTagCodes {
			if !validCode(code, 50) {
				return Organization{}, &ValidationError{Field: "domain_tag_codes", Message: "must contain uppercase stable codes"}
			}
			if _, ok := seen[code]; ok {
				return Organization{}, &ValidationError{Field: "domain_tag_codes", Message: "must not contain duplicates"}
			}
			seen[code] = struct{}{}
		}
	}
	return s.repository.Update(ctx, id, input)
}

func (s *UseCase) ReplaceDomainTags(ctx context.Context, id string, codes []string) (Organization, error) {
	if !entitybiz.IsOrganizationID(id) {
		return Organization{}, &ValidationError{Field: "organization_id", Message: "must be a stable Organization ID"}
	}
	seen := map[string]struct{}{}
	for _, code := range codes {
		if !validCode(code, 50) {
			return Organization{}, &ValidationError{Field: "domain_tag_codes", Message: "must contain uppercase stable codes"}
		}
		if _, duplicate := seen[code]; duplicate {
			return Organization{}, &ValidationError{Field: "domain_tag_codes", Message: "must not contain duplicates"}
		}
		seen[code] = struct{}{}
	}
	return s.repository.ReplaceDomainTags(ctx, id, append([]string{}, codes...))
}

func (s *UseCase) Catalog(ctx context.Context) (Catalog, error) { return s.repository.Catalog(ctx) }

func (s *UseCase) ListMembers(ctx context.Context, id string, asOf *time.Time) ([]Member, error) {
	if !entitybiz.IsOrganizationID(id) {
		return nil, &ValidationError{Field: "organization_id", Message: "must be a stable Organization ID"}
	}
	return s.repository.ListMembers(ctx, id, asOf)
}

func (s *UseCase) CreateMember(ctx context.Context, input Member) (Member, error) {
	if err := validateMember(input); err != nil {
		return Member{}, err
	}
	return s.repository.CreateMember(ctx, input)
}

func (s *UseCase) UpdateMember(ctx context.Context, organizationID string, id int64, input Member) (Member, error) {
	input.OrganizationID = organizationID
	if id <= 0 {
		return Member{}, &ValidationError{Field: "member_id", Message: "must be positive"}
	}
	if err := validateMember(input); err != nil {
		return Member{}, err
	}
	return s.repository.UpdateMember(ctx, organizationID, id, input)
}

func (s *UseCase) DeleteMember(ctx context.Context, organizationID string, id int64) error {
	if !entitybiz.IsOrganizationID(organizationID) || id <= 0 {
		return &ValidationError{Field: "member", Message: "must identify an Organization member"}
	}
	return s.repository.DeleteMember(ctx, organizationID, id)
}

func validateMember(input Member) error {
	if !entitybiz.IsOrganizationID(input.OrganizationID) {
		return &ValidationError{Field: "organization_id", Message: "must be a stable Organization ID"}
	}
	if !entitybiz.IsCountryID(input.CountryID) {
		return &ValidationError{Field: "country_id", Message: "must be a stable Country ID"}
	}
	if !oneOf(input.MembershipType, "FULL_MEMBER", "OBSERVER", "ASSOCIATE", "PARTNER", "CANDIDATE") {
		return &ValidationError{Field: "membership_type", Message: "is not supported"}
	}
	if input.EffectiveDate != nil && input.ExpiryDate != nil && input.ExpiryDate.Before(*input.EffectiveDate) {
		return &ValidationError{Field: "expiry_date", Message: "must not precede effective_date"}
	}
	return nil
}

func validateOrganization(input Organization) error {
	if !entitybiz.IsOrganizationID(input.ID) {
		return &ValidationError{Field: "id", Message: "must equal ORG immediately followed by a canonical lowercase UUID"}
	}
	if !validCode(input.Code, 30) {
		return &ValidationError{Field: "code", Message: "must be an uppercase stable code with at most 30 characters"}
	}
	if strings.TrimSpace(input.Name) == "" || utf8.RuneCountInString(input.Name) > 100 {
		return &ValidationError{Field: "name", Message: "must be nonblank and contain at most 100 characters"}
	}
	if strings.TrimSpace(input.NameEn) == "" || utf8.RuneCountInString(input.NameEn) > 100 {
		return &ValidationError{Field: "name_en", Message: "must be nonblank and contain at most 100 characters"}
	}
	if !validCode(input.Category.Code, 30) {
		return &ValidationError{Field: "category_code", Message: "must be an uppercase stable code"}
	}
	if !validCode(input.Function.Code, 30) {
		return &ValidationError{Field: "function_code", Message: "must be an uppercase stable code"}
	}
	if input.RegionID != nil && !entitybiz.IsRegionID(*input.RegionID) {
		return &ValidationError{Field: "region_id", Message: "must be a stable Region ID"}
	}
	for _, country := range []struct {
		field string
		value *string
	}{{"dominant_party_id", input.DominantPartyID}, {"headquarters_country_id", input.HeadquartersCountryID}} {
		if country.value != nil && !entitybiz.IsCountryID(*country.value) {
			return &ValidationError{Field: country.field, Message: "must be a stable Country ID"}
		}
	}
	if input.BindingPowerLevel != nil && !oneOf(*input.BindingPowerLevel, "HIGH", "MEDIUM", "LOW") {
		return &ValidationError{Field: "binding_power_level", Message: "must be HIGH, MEDIUM, or LOW"}
	}
	if input.InfluenceRating != nil && !oneOf(*input.InfluenceRating, "S", "A", "B") {
		return &ValidationError{Field: "influence_rating", Message: "must be S, A, or B"}
	}
	if input.LegalEntityCode != nil && (len(*input.LegalEntityCode) != 20 || !lettersAndDigits(*input.LegalEntityCode)) {
		return &ValidationError{Field: "legal_entity_code", Message: "must contain 20 uppercase ASCII letters or digits"}
	}
	for _, optional := range []struct {
		field string
		value *string
		limit int
	}{
		{"strategic_positioning", input.StrategicPositioning, 0},
		{"core_impact_scope", input.CoreImpactScope, 0},
		{"founding_document", input.FoundingDocument, 200},
		{"headquarters_city", input.HeadquartersCity, 100},
		{"headquarters_subdivision_id", input.HeadquartersSubdivisionID, 32},
		{"description", input.Description, 0},
	} {
		if optional.value != nil && (strings.TrimSpace(*optional.value) == "" || optional.limit > 0 && utf8.RuneCountInString(*optional.value) > optional.limit) {
			return &ValidationError{Field: optional.field, Message: fmt.Sprintf("must be nonblank%s", limitMessage(optional.limit))}
		}
	}
	return nil
}

func validCode(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	if value[0] < 'A' || value[0] > 'Z' {
		return false
	}
	for _, character := range value {
		if (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '_' {
			continue
		}
		return false
	}
	return true
}

func lettersAndDigits(value string) bool {
	for _, character := range value {
		if (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') {
			continue
		}
		return false
	}
	return value != ""
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func limitMessage(limit int) string {
	if limit == 0 {
		return ""
	}
	return fmt.Sprintf(" and contain at most %d characters", limit)
}

func cloneOrganization(input Organization) Organization {
	input.RegionID = cloneString(input.RegionID)
	input.LegalEntityCode = cloneString(input.LegalEntityCode)
	input.DominantPartyID = cloneString(input.DominantPartyID)
	input.BindingPowerLevel = cloneString(input.BindingPowerLevel)
	input.InfluenceRating = cloneString(input.InfluenceRating)
	input.StrategicPositioning = cloneString(input.StrategicPositioning)
	input.CoreImpactScope = cloneString(input.CoreImpactScope)
	input.FoundingDocument = cloneString(input.FoundingDocument)
	input.HeadquartersCity = cloneString(input.HeadquartersCity)
	input.HeadquartersCountryID = cloneString(input.HeadquartersCountryID)
	input.HeadquartersSubdivisionID = cloneString(input.HeadquartersSubdivisionID)
	input.Description = cloneString(input.Description)
	input.DomainTags = append([]DomainTag{}, input.DomainTags...)
	return input
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
