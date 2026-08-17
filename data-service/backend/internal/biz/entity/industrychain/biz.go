package industrychain

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

var (
	ErrNotFound    = errors.New("IndustryChain not found")
	ErrConflict    = errors.New("IndustryChain conflict")
	ErrPersistence = errors.New("IndustryChain persistence failed")
)

type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

type ReferenceError struct{ Field, Message string }

func (e *ReferenceError) Error() string { return e.Field + ": " + e.Message }

type ReviewStatus string

const (
	ReviewStatusCandidate ReviewStatus = "candidate"
	ReviewStatusApproved  ReviewStatus = "approved"
)

type ID string

type IndustryChain struct {
	ID                       ID
	Name                     string
	Aliases                  []string
	Scope                    string
	TargetOutput             string
	EndUse                   string
	Geography                string
	PrimaryCountryID         *string
	AsOfDate                 time.Time
	ReviewStatus             ReviewStatus
	ReviewNote               *string
	TechnologyRouteQualifier *string
	ObservableVariables      []string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type Update struct {
	Name                     string
	Aliases                  []string
	Scope                    string
	TargetOutput             string
	EndUse                   string
	Geography                string
	PrimaryCountryID         *string
	AsOfDate                 time.Time
	ReviewStatus             ReviewStatus
	ReviewNote               *string
	TechnologyRouteQualifier *string
	ObservableVariables      []string
}
type ListRequest struct {
	PageSize int
	Cursor   string
}
type ListKey struct{ ID ID }
type ListQuery struct {
	PageSize int
	After    *ListKey
}
type ListResult struct {
	Items   []IndustryChain
	HasMore bool
}
type Page struct {
	Items      []IndustryChain
	NextCursor *string
}

type Repository interface {
	Create(context.Context, IndustryChain) (IndustryChain, error)
	Get(context.Context, ID) (IndustryChain, error)
	List(context.Context, ListQuery) (ListResult, error)
	Update(context.Context, ID, Update) (IndustryChain, error)
}
type UseCase struct{ repository Repository }

func NewUseCase(repository Repository) (*UseCase, error) {
	if repository == nil {
		return nil, errors.New("IndustryChain repository is required")
	}
	return &UseCase{repository: repository}, nil
}

func (s *UseCase) Create(ctx context.Context, input IndustryChain) (IndustryChain, error) {
	if strings.TrimSpace(string(input.ID)) != "" {
		return IndustryChain{}, &ValidationError{Field: "id", Message: "must be omitted because Data generates IndustryChain IDs"}
	}
	if err := validateValues(input.Name, input.Aliases, input.Scope, input.TargetOutput, input.EndUse, input.Geography, input.PrimaryCountryID, input.AsOfDate, input.ReviewStatus, input.ReviewNote, input.TechnologyRouteQualifier, input.ObservableVariables); err != nil {
		return IndustryChain{}, err
	}
	id, err := coreid.New(coreid.IndustryChain)
	if err != nil {
		return IndustryChain{}, fmt.Errorf("generate IndustryChain ID: %w", err)
	}
	input.ID = ID(id)
	return s.repository.Create(ctx, cloneIndustryChain(input))
}

func (s *UseCase) Get(ctx context.Context, id ID) (IndustryChain, error) {
	if err := validateID(id); err != nil {
		return IndustryChain{}, err
	}
	return s.repository.Get(ctx, id)
}

func (s *UseCase) List(ctx context.Context, request ListRequest) (Page, error) {
	if request.PageSize < 1 || request.PageSize > 100 {
		return Page{}, &ValidationError{Field: "page_size", Message: "must be between 1 and 100"}
	}
	after, err := decodeListCursor(request.Cursor)
	if err != nil {
		return Page{}, err
	}
	result, err := s.repository.List(ctx, ListQuery{PageSize: request.PageSize, After: after})
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: result.Items}
	if result.HasMore && len(result.Items) > 0 {
		next, err := encodeListCursor(result.Items[len(result.Items)-1])
		if err != nil {
			return Page{}, fmt.Errorf("encode IndustryChain list cursor: %w", err)
		}
		page.NextCursor = &next
	}
	return page, nil
}

func (s *UseCase) Update(ctx context.Context, id ID, input Update) (IndustryChain, error) {
	if err := validateID(id); err != nil {
		return IndustryChain{}, err
	}
	if err := validateValues(input.Name, input.Aliases, input.Scope, input.TargetOutput, input.EndUse, input.Geography, input.PrimaryCountryID, input.AsOfDate, input.ReviewStatus, input.ReviewNote, input.TechnologyRouteQualifier, input.ObservableVariables); err != nil {
		return IndustryChain{}, err
	}
	return s.repository.Update(ctx, id, cloneUpdate(input))
}

func IsID(value string) bool { return coreid.Is(value, coreid.IndustryChain) }
func ValidatePersisted(input IndustryChain) error {
	if err := validateID(input.ID); err != nil {
		return err
	}
	return validateValues(input.Name, input.Aliases, input.Scope, input.TargetOutput, input.EndUse, input.Geography, input.PrimaryCountryID, input.AsOfDate, input.ReviewStatus, input.ReviewNote, input.TechnologyRouteQualifier, input.ObservableVariables)
}
func validateID(value ID) error {
	if !IsID(string(value)) {
		return &ValidationError{Field: "industry_chain_id", Message: "must equal ICH immediately followed by a canonical lowercase UUID"}
	}
	return nil
}
func validateValues(name string, aliases []string, scope, targetOutput, endUse, geography string, primaryCountryID *string, asOfDate time.Time, reviewStatus ReviewStatus, reviewNote, qualifier *string, variables []string) error {
	for _, value := range []struct{ field, text string }{{"name", name}, {"scope", scope}, {"target_output", targetOutput}, {"end_use", endUse}, {"geography", geography}} {
		if strings.TrimSpace(value.text) == "" {
			return &ValidationError{Field: value.field, Message: "must be nonblank"}
		}
	}
	if err := validateStringSet("aliases", aliases, true); err != nil {
		return err
	}
	if primaryCountryID != nil && !coreid.Is(*primaryCountryID, coreid.Country) {
		return &ValidationError{Field: "primary_country_id", Message: "must be a stable Country ID when present"}
	}
	if asOfDate.IsZero() {
		return &ValidationError{Field: "as_of_date", Message: "must be a calendar date"}
	}
	if reviewStatus != ReviewStatusCandidate && reviewStatus != ReviewStatusApproved {
		return &ValidationError{Field: "review_status", Message: "must be candidate or approved"}
	}
	for _, optional := range []struct {
		field string
		value *string
	}{{"review_note", reviewNote}, {"technology_route_qualifier", qualifier}} {
		if optional.value != nil && strings.TrimSpace(*optional.value) == "" {
			return &ValidationError{Field: optional.field, Message: "must be nonblank when present"}
		}
	}
	return validateStringSet("observable_variables", variables, false)
}
func validateStringSet(field string, values []string, allowEmpty bool) error {
	if values == nil || (!allowEmpty && len(values) == 0) {
		return &ValidationError{Field: field, Message: "must be provided as a nonempty array"}
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if strings.TrimSpace(value) == "" {
			return &ValidationError{Field: fmt.Sprintf("%s[%d]", field, index), Message: "must be nonblank"}
		}
		if _, duplicate := seen[value]; duplicate {
			return &ValidationError{Field: fmt.Sprintf("%s[%d]", field, index), Message: "must be unique"}
		}
		seen[value] = struct{}{}
	}
	return nil
}
func cloneIndustryChain(input IndustryChain) IndustryChain {
	input.Aliases = append([]string{}, input.Aliases...)
	input.ObservableVariables = append([]string{}, input.ObservableVariables...)
	input.PrimaryCountryID = cloneString(input.PrimaryCountryID)
	input.ReviewNote = cloneString(input.ReviewNote)
	input.TechnologyRouteQualifier = cloneString(input.TechnologyRouteQualifier)
	return input
}
func cloneUpdate(input Update) Update {
	input.Aliases = append([]string{}, input.Aliases...)
	input.ObservableVariables = append([]string{}, input.ObservableVariables...)
	input.PrimaryCountryID = cloneString(input.PrimaryCountryID)
	input.ReviewNote = cloneString(input.ReviewNote)
	input.TechnologyRouteQualifier = cloneString(input.TechnologyRouteQualifier)
	return input
}
func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

type listCursor struct {
	Version int `json:"v"`
	ID      ID  `json:"id"`
}

func encodeListCursor(input IndustryChain) (string, error) {
	payload, err := json.Marshal(listCursor{Version: 1, ID: input.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}
func decodeListCursor(value string) (*ListKey, error) {
	if value == "" {
		return nil, nil
	}
	if len(value) > 256 {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque IndustryChain list cursor"}
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque IndustryChain list cursor"}
	}
	var cursor listCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.Version != 1 || validateID(cursor.ID) != nil {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque IndustryChain list cursor"}
	}
	return &ListKey{ID: cursor.ID}, nil
}
