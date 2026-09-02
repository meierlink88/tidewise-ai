package company

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

var (
	ErrNotFound                  = errors.New("Company not found")
	ErrConflict                  = errors.New("Company conflict")
	ErrPersistence               = errors.New("Company persistence failed")
	ErrProjectionSnapshotChanged = errors.New("Company projection snapshot changed")
)

const ProjectionSchemaVersion = "company-projection-snapshot.v1"

var projectionSnapshotIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

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
	IndustryLinks         []IndustryLink
}

type ProjectionListRequest struct {
	PageSize int
	Cursor   string
}

type ProjectionListKey struct {
	Code string
	ID   ID
}

type ProjectionListQuery struct {
	PageSize   int
	SnapshotID string
	After      *ProjectionListKey
}

type ProjectionListResult struct {
	SnapshotID string
	Items      []Company
	HasMore    bool
}

type ProjectionPage struct {
	SnapshotID string
	Items      []Company
	NextCursor *string
}

type ProjectionRepository interface {
	ListProjection(context.Context, ProjectionListQuery) (ProjectionListResult, error)
}

type ProjectionUseCase struct{ repository ProjectionRepository }

func NewProjectionUseCase(repository ProjectionRepository) (*ProjectionUseCase, error) {
	if repository == nil {
		return nil, errors.New("Company projection repository is required")
	}
	return &ProjectionUseCase{repository: repository}, nil
}

func (s *ProjectionUseCase) ListProjection(ctx context.Context, request ProjectionListRequest) (ProjectionPage, error) {
	if request.PageSize < 1 || request.PageSize > 100 {
		return ProjectionPage{}, &ValidationError{Field: "page_size", Message: "must be between 1 and 100"}
	}
	cursor, err := decodeProjectionCursor(request.Cursor)
	if err != nil {
		return ProjectionPage{}, err
	}
	query := ProjectionListQuery{PageSize: request.PageSize}
	if cursor != nil {
		query.SnapshotID = cursor.SnapshotID
		query.After = &ProjectionListKey{Code: cursor.Code, ID: cursor.ID}
	}
	result, err := s.repository.ListProjection(ctx, query)
	if err != nil {
		return ProjectionPage{}, err
	}
	if !projectionSnapshotIDPattern.MatchString(result.SnapshotID) {
		return ProjectionPage{}, ErrPersistence
	}
	if cursor != nil && result.SnapshotID != cursor.SnapshotID {
		return ProjectionPage{}, ErrProjectionSnapshotChanged
	}
	if len(result.Items) > request.PageSize || (result.HasMore && len(result.Items) != request.PageSize) {
		return ProjectionPage{}, ErrPersistence
	}
	seenIDs := make(map[ID]struct{}, len(result.Items))
	seenCodes := make(map[string]struct{}, len(result.Items))
	for _, item := range result.Items {
		if err := ValidateProjectionCompany(item); err != nil {
			return ProjectionPage{}, ErrPersistence
		}
		if _, duplicate := seenIDs[item.ID]; duplicate {
			return ProjectionPage{}, ErrPersistence
		}
		if _, duplicate := seenCodes[item.Code]; duplicate {
			return ProjectionPage{}, ErrPersistence
		}
		seenIDs[item.ID] = struct{}{}
		seenCodes[item.Code] = struct{}{}
	}
	if cursor != nil && len(result.Items) > 0 {
		first := result.Items[0]
		if first.Code == cursor.Code && first.ID == cursor.ID {
			return ProjectionPage{}, ErrPersistence
		}
	}
	page := ProjectionPage{SnapshotID: result.SnapshotID, Items: result.Items}
	if result.HasMore {
		if len(result.Items) == 0 {
			return ProjectionPage{}, ErrPersistence
		}
		last := result.Items[len(result.Items)-1]
		next, err := encodeProjectionCursor(projectionCursor{
			Version: 1, SnapshotID: result.SnapshotID, Code: last.Code, ID: last.ID,
		})
		if err != nil {
			return ProjectionPage{}, fmt.Errorf("encode Company projection cursor: %w", err)
		}
		page.NextCursor = &next
	}
	return page, nil
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

func ValidateProjectionCompany(input Company) error {
	if err := ValidatePersisted(input); err != nil {
		return err
	}
	if _, offset := input.CreatedAt.Zone(); offset != 0 {
		return &ValidationError{Field: "created_at", Message: "must be UTC"}
	}
	if _, offset := input.UpdatedAt.Zone(); offset != 0 {
		return &ValidationError{Field: "updated_at", Message: "must be UTC"}
	}
	if input.IndustryLinks == nil {
		return &ValidationError{Field: "industry_links", Message: "must be provided as an array"}
	}
	seenIDs := make(map[string]struct{}, len(input.IndustryLinks))
	seenEndpoints := make(map[IndustryID]struct{}, len(input.IndustryLinks))
	availableIndustries := make(map[IndustryID]struct{}, len(input.Industries))
	for _, industry := range input.Industries {
		availableIndustries[industry.ID] = struct{}{}
	}
	if len(input.IndustryLinks) != len(input.Industries) {
		return &ValidationError{Field: "industry_links", Message: "must match resolved Industry summaries"}
	}
	for index, link := range input.IndustryLinks {
		if !coreid.Is(link.ID, coreid.CompanyIndustryLink) {
			return &ValidationError{Field: fmt.Sprintf("industry_links[%d].id", index), Message: "must be a stable Company Industry Link ID"}
		}
		if link.CompanyID != input.ID {
			return &ValidationError{Field: fmt.Sprintf("industry_links[%d].company_id", index), Message: "must identify the containing Company"}
		}
		if !coreid.Is(string(link.IndustryID), coreid.Industry) {
			return &ValidationError{Field: fmt.Sprintf("industry_links[%d].industry_id", index), Message: "must be a stable Industry ID"}
		}
		expectedLinkID, err := coreid.Derive(
			coreid.CompanyIndustryLink,
			"company-industry-link",
			string(input.ID),
			string(link.IndustryID),
		)
		if err != nil || link.ID != expectedLinkID {
			return &ValidationError{
				Field:   fmt.Sprintf("industry_links[%d].id", index),
				Message: "must be derived from the Company and Industry endpoints",
			}
		}
		if link.CreatedAt.IsZero() {
			return &ValidationError{Field: fmt.Sprintf("industry_links[%d].created_at", index), Message: "must be present"}
		}
		if _, offset := link.CreatedAt.Zone(); offset != 0 {
			return &ValidationError{Field: fmt.Sprintf("industry_links[%d].created_at", index), Message: "must be UTC"}
		}
		if _, exists := availableIndustries[link.IndustryID]; !exists {
			return &ValidationError{Field: fmt.Sprintf("industry_links[%d].industry_id", index), Message: "must resolve to an Industry summary"}
		}
		if _, duplicate := seenIDs[link.ID]; duplicate {
			return &ValidationError{Field: fmt.Sprintf("industry_links[%d].id", index), Message: "must be unique"}
		}
		if _, duplicate := seenEndpoints[link.IndustryID]; duplicate {
			return &ValidationError{Field: fmt.Sprintf("industry_links[%d].industry_id", index), Message: "must be unique per Company"}
		}
		seenIDs[link.ID] = struct{}{}
		seenEndpoints[link.IndustryID] = struct{}{}
	}
	return nil
}

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

type projectionCursor struct {
	Version    int    `json:"v"`
	SnapshotID string `json:"snapshot_id"`
	Code       string `json:"after_code"`
	ID         ID     `json:"after_id"`
}

func encodeProjectionCursor(cursor projectionCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeProjectionCursor(value string) (*projectionCursor, error) {
	invalid := func() (*projectionCursor, error) {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque Company projection cursor"}
	}
	if value == "" {
		return nil, nil
	}
	if len(value) > 512 {
		return invalid()
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return invalid()
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	var cursor projectionCursor
	if err := decoder.Decode(&cursor); err != nil {
		return invalid()
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return invalid()
	}
	if cursor.Version != 1 || !projectionSnapshotIDPattern.MatchString(cursor.SnapshotID) ||
		strings.TrimSpace(cursor.Code) == "" || utf8.RuneCountInString(cursor.Code) > 30 || !IsID(string(cursor.ID)) {
		return invalid()
	}
	return &cursor, nil
}
