package company

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

var (
	ErrNotFound    = errors.New("Company not found")
	ErrConflict    = errors.New("Company conflict")
	ErrPersistence = errors.New("Company persistence failed")
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

type ID string

type IndustryID string

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusMerged   Status = "merged"
)

type OwnershipType string

const (
	OwnershipStateControlled       OwnershipType = "STATE_CONTROLLED"
	OwnershipFamilyControlled      OwnershipType = "FAMILY_CONTROLLED"
	OwnershipFounderControlled     OwnershipType = "FOUNDER_CONTROLLED"
	OwnershipInstitutionControlled OwnershipType = "INSTITUTION_CONTROLLED"
	OwnershipDispersed             OwnershipType = "DISPERSED"
	OwnershipOther                 OwnershipType = "OTHER"
)

type Industry struct {
	ID                   IndustryID
	Name                 string
	ClassificationSystem string
	IndustryCode         string
}

type Company struct {
	ID                    ID
	Code                  string
	Name                  string
	NameEn                *string
	LegalName             *string
	Aliases               []string
	RegistrationCountryID *string
	OperatingArea         *string
	HeadquartersCity      *string
	FoundingDate          *time.Time
	IPODate               *time.Time
	LegalForm             *string
	OwnershipType         *OwnershipType
	StrategicPositioning  *string
	Description           *string
	Status                Status
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Industries            []Industry
}

// CreateInput contains only caller-owned Company facts. Identity, timestamps,
// and Industry links are generated or managed by the Data Service.
type CreateInput struct {
	Code                  string
	Name                  string
	NameEn                *string
	LegalName             *string
	Aliases               []string
	RegistrationCountryID *string
	OperatingArea         *string
	HeadquartersCity      *string
	FoundingDate          *time.Time
	IPODate               *time.Time
	LegalForm             *string
	OwnershipType         *OwnershipType
	StrategicPositioning  *string
	Description           *string
	Status                Status
}

type Update struct {
	Name                  string
	NameEn                *string
	LegalName             *string
	Aliases               []string
	RegistrationCountryID *string
	OperatingArea         *string
	HeadquartersCity      *string
	FoundingDate          *time.Time
	IPODate               *time.Time
	LegalForm             *string
	OwnershipType         *OwnershipType
	StrategicPositioning  *string
	Description           *string
	Status                Status
}

type Repository interface {
	Create(context.Context, Company) (Company, error)
	Get(context.Context, ID) (Company, error)
	List(context.Context) ([]Company, error)
	Update(context.Context, ID, Update) (Company, error)
}

type Store interface {
	Repository
	IndustryTransaction
}

type UseCase struct{ repository Store }

func NewUseCase(repository Store) (*UseCase, error) {
	if repository == nil {
		return nil, errors.New("Company repository is required")
	}
	return &UseCase{repository: repository}, nil
}

func (s *UseCase) Create(ctx context.Context, input CreateInput) (Company, error) {
	company := Company{
		Code: input.Code, Name: input.Name, NameEn: input.NameEn, LegalName: input.LegalName,
		Aliases: input.Aliases, RegistrationCountryID: input.RegistrationCountryID,
		OperatingArea: input.OperatingArea, HeadquartersCity: input.HeadquartersCity,
		FoundingDate: input.FoundingDate, IPODate: input.IPODate, LegalForm: input.LegalForm,
		OwnershipType: input.OwnershipType, StrategicPositioning: input.StrategicPositioning,
		Description: input.Description, Status: input.Status,
	}
	if err := validateCompany(company); err != nil {
		return Company{}, err
	}
	id, err := coreid.New(coreid.Company)
	if err != nil {
		return Company{}, fmt.Errorf("generate Company ID: %w", err)
	}
	company.ID = ID(id)
	return s.repository.Create(ctx, cloneCompany(company))
}

func (s *UseCase) Get(ctx context.Context, id ID) (Company, error) {
	if err := validateID(id); err != nil {
		return Company{}, err
	}
	return s.repository.Get(ctx, id)
}

func (s *UseCase) List(ctx context.Context) ([]Company, error) {
	return s.repository.List(ctx)
}

func (s *UseCase) Update(ctx context.Context, id ID, input Update) (Company, error) {
	if err := validateID(id); err != nil {
		return Company{}, err
	}
	if err := validateMutable(
		input.Name, input.NameEn, input.LegalName, input.Aliases, input.RegistrationCountryID,
		input.OperatingArea, input.HeadquartersCity, input.FoundingDate, input.IPODate,
		input.LegalForm, input.OwnershipType, input.StrategicPositioning, input.Description, input.Status,
	); err != nil {
		return Company{}, err
	}
	return s.repository.Update(ctx, id, cloneUpdate(input))
}

func (s *UseCase) ReplaceIndustries(ctx context.Context, id ID, industryIDs []IndustryID) (Company, error) {
	if err := validateID(id); err != nil {
		return Company{}, err
	}
	seen := make(map[IndustryID]struct{}, len(industryIDs))
	links := make([]IndustryLink, 0, len(industryIDs))
	for index, industryID := range industryIDs {
		if !coreid.Is(string(industryID), coreid.Industry) {
			return Company{}, &ValidationError{Field: fmt.Sprintf("industry_ids[%d]", index), Message: "must be a stable Industry ID"}
		}
		if _, duplicate := seen[industryID]; duplicate {
			return Company{}, &ValidationError{Field: fmt.Sprintf("industry_ids[%d]", index), Message: "must be unique"}
		}
		seen[industryID] = struct{}{}
		linkID, err := coreid.Derive(coreid.CompanyIndustryLink, "company-industry-link", string(id), string(industryID))
		if err != nil {
			return Company{}, fmt.Errorf("generate Company Industry Link ID: %w", err)
		}
		links = append(links, IndustryLink{ID: linkID, IndustryID: industryID})
	}
	return s.repository.ReplaceIndustries(ctx, id, links)
}

func IsID(value string) bool { return coreid.Is(value, coreid.Company) }

func ValidatePersisted(input Company) error {
	if err := validateID(input.ID); err != nil {
		return err
	}
	if err := validateCompany(input); err != nil {
		return err
	}
	if input.Industries == nil {
		return &ValidationError{Field: "industries", Message: "must be provided as an array"}
	}
	seen := make(map[IndustryID]struct{}, len(input.Industries))
	for index, industry := range input.Industries {
		if !coreid.Is(string(industry.ID), coreid.Industry) || strings.TrimSpace(industry.Name) == "" ||
			strings.TrimSpace(industry.ClassificationSystem) == "" || strings.TrimSpace(industry.IndustryCode) == "" {
			return &ValidationError{Field: fmt.Sprintf("industries[%d]", index), Message: "must be a valid Industry summary"}
		}
		if _, duplicate := seen[industry.ID]; duplicate {
			return &ValidationError{Field: fmt.Sprintf("industries[%d]", index), Message: "must be unique"}
		}
		seen[industry.ID] = struct{}{}
	}
	if input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return &ValidationError{Field: "timestamps", Message: "must be present and ordered"}
	}
	return nil
}

func validateCompany(input Company) error {
	if strings.TrimSpace(input.Code) == "" || utf8.RuneCountInString(input.Code) > 30 {
		return &ValidationError{Field: "code", Message: "must be nonblank and contain at most 30 characters"}
	}
	return validateMutable(
		input.Name, input.NameEn, input.LegalName, input.Aliases, input.RegistrationCountryID,
		input.OperatingArea, input.HeadquartersCity, input.FoundingDate, input.IPODate,
		input.LegalForm, input.OwnershipType, input.StrategicPositioning, input.Description, input.Status,
	)
}

func validateMutable(
	name string,
	nameEn, legalName *string,
	aliases []string,
	registrationCountryID, operatingArea, headquartersCity *string,
	foundingDate, ipoDate *time.Time,
	legalForm *string,
	ownershipType *OwnershipType,
	strategicPositioning, description *string,
	status Status,
) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 200 {
		return &ValidationError{Field: "name", Message: "must be nonblank and contain at most 200 characters"}
	}
	if err := validateStringSet("aliases", aliases, 200); err != nil {
		return err
	}
	for _, optional := range []struct {
		field     string
		value     *string
		maxLength int
	}{
		{"name_en", nameEn, 200},
		{"legal_name", legalName, 300},
		{"operating_area", operatingArea, 0},
		{"headquarters_city", headquartersCity, 100},
		{"legal_form", legalForm, 64},
		{"strategic_positioning", strategicPositioning, 0},
		{"description", description, 0},
	} {
		if optional.value != nil && (strings.TrimSpace(*optional.value) == "" ||
			(optional.maxLength > 0 && utf8.RuneCountInString(*optional.value) > optional.maxLength)) {
			return &ValidationError{Field: optional.field, Message: "must be nonblank and within its maximum length when present"}
		}
	}
	if registrationCountryID != nil && !coreid.Is(*registrationCountryID, coreid.Country) {
		return &ValidationError{Field: "registration_country_id", Message: "must be a stable Country ID when present"}
	}
	if ownershipType != nil && !validOwnership(*ownershipType) {
		return &ValidationError{Field: "ownership_type", Message: "must use the controlled Company ownership vocabulary"}
	}
	if foundingDate != nil && ipoDate != nil && dateBefore(*ipoDate, *foundingDate) {
		return &ValidationError{Field: "ipo_date", Message: "must not precede founding_date"}
	}
	if status != StatusActive && status != StatusInactive && status != StatusMerged {
		return &ValidationError{Field: "status", Message: "must be active, inactive, or merged"}
	}
	return nil
}

func validOwnership(value OwnershipType) bool {
	switch value {
	case OwnershipStateControlled, OwnershipFamilyControlled, OwnershipFounderControlled,
		OwnershipInstitutionControlled, OwnershipDispersed, OwnershipOther:
		return true
	default:
		return false
	}
}

func validateID(id ID) error {
	if !IsID(string(id)) {
		return &ValidationError{Field: "company_id", Message: "must equal COM immediately followed by a canonical lowercase UUID"}
	}
	return nil
}

func validateStringSet(field string, values []string, maxLength int) error {
	if values == nil {
		return &ValidationError{Field: field, Message: "must be provided as an array"}
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > maxLength {
			return &ValidationError{Field: fmt.Sprintf("%s[%d]", field, index), Message: "must be nonblank and within its maximum length"}
		}
		if _, duplicate := seen[value]; duplicate {
			return &ValidationError{Field: fmt.Sprintf("%s[%d]", field, index), Message: "must be unique"}
		}
		seen[value] = struct{}{}
	}
	return nil
}

func dateBefore(left, right time.Time) bool {
	leftDate := left.Format("2006-01-02")
	rightDate := right.Format("2006-01-02")
	return leftDate < rightDate
}

func cloneCompany(input Company) Company {
	input.NameEn = cloneString(input.NameEn)
	input.LegalName = cloneString(input.LegalName)
	input.Aliases = append([]string{}, input.Aliases...)
	input.RegistrationCountryID = cloneString(input.RegistrationCountryID)
	input.OperatingArea = cloneString(input.OperatingArea)
	input.HeadquartersCity = cloneString(input.HeadquartersCity)
	input.FoundingDate = cloneTime(input.FoundingDate)
	input.IPODate = cloneTime(input.IPODate)
	input.LegalForm = cloneString(input.LegalForm)
	input.OwnershipType = cloneOwnership(input.OwnershipType)
	input.StrategicPositioning = cloneString(input.StrategicPositioning)
	input.Description = cloneString(input.Description)
	input.Industries = append([]Industry{}, input.Industries...)
	return input
}

func cloneUpdate(input Update) Update {
	input.NameEn = cloneString(input.NameEn)
	input.LegalName = cloneString(input.LegalName)
	input.Aliases = append([]string{}, input.Aliases...)
	input.RegistrationCountryID = cloneString(input.RegistrationCountryID)
	input.OperatingArea = cloneString(input.OperatingArea)
	input.HeadquartersCity = cloneString(input.HeadquartersCity)
	input.FoundingDate = cloneTime(input.FoundingDate)
	input.IPODate = cloneTime(input.IPODate)
	input.LegalForm = cloneString(input.LegalForm)
	input.OwnershipType = cloneOwnership(input.OwnershipType)
	input.StrategicPositioning = cloneString(input.StrategicPositioning)
	input.Description = cloneString(input.Description)
	return input
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneOwnership(value *OwnershipType) *OwnershipType {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
