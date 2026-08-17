package concept

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
	ErrNotFound    = errors.New("Concept not found")
	ErrConflict    = errors.New("Concept conflict")
	ErrPersistence = errors.New("Concept persistence failed")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

type Type string

const (
	TypeTechnology       Type = "technology"
	TypePolicy           Type = "policy"
	TypeApplication      Type = "application"
	TypeDemand           Type = "demand"
	TypeBusinessModel    Type = "business_model"
	TypeCompanyEcosystem Type = "company_ecosystem"
	TypeProductEcosystem Type = "product_ecosystem"
	TypeEventNarrative   Type = "event_narrative"
	TypeMarketTheme      Type = "market_theme"
)

type ReviewStatus string

const (
	ReviewStatusCandidate ReviewStatus = "candidate"
	ReviewStatusApproved  ReviewStatus = "approved"
)

type ID string

type Concept struct {
	ID           ID
	Name         string
	Aliases      []string
	ConceptType  Type
	Definition   string
	ReviewStatus ReviewStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Update struct {
	Name         string
	Aliases      []string
	ConceptType  Type
	Definition   string
	ReviewStatus ReviewStatus
}

type ListRequest struct {
	PageSize int
	Cursor   string
}

type ListKey struct {
	Name string
	ID   ID
}

type ListQuery struct {
	PageSize int
	After    *ListKey
}

type ListResult struct {
	Items   []Concept
	HasMore bool
}

type Page struct {
	Items      []Concept
	NextCursor *string
}

type Repository interface {
	Create(context.Context, Concept) (Concept, error)
	Get(context.Context, ID) (Concept, error)
	List(context.Context, ListQuery) (ListResult, error)
	Update(context.Context, ID, Update) (Concept, error)
}

type UseCase struct{ repository Repository }

func NewUseCase(repository Repository) (*UseCase, error) {
	if repository == nil {
		return nil, errors.New("Concept repository is required")
	}
	return &UseCase{repository: repository}, nil
}

func (s *UseCase) Create(ctx context.Context, input Concept) (Concept, error) {
	if strings.TrimSpace(string(input.ID)) != "" {
		return Concept{}, &ValidationError{Field: "id", Message: "must be omitted because Data generates Concept IDs"}
	}
	if err := validateValues(input.Name, input.Aliases, input.ConceptType, input.Definition, input.ReviewStatus); err != nil {
		return Concept{}, err
	}
	id, err := coreid.New(coreid.Entity)
	if err != nil {
		return Concept{}, fmt.Errorf("generate Concept ID: %w", err)
	}
	input.ID = ID(id)
	return s.repository.Create(ctx, cloneConcept(input))
}

func (s *UseCase) Get(ctx context.Context, id ID) (Concept, error) {
	if err := validateID(id); err != nil {
		return Concept{}, err
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
			return Page{}, fmt.Errorf("encode Concept list cursor: %w", err)
		}
		page.NextCursor = &next
	}
	return page, nil
}

func (s *UseCase) Update(ctx context.Context, id ID, input Update) (Concept, error) {
	if err := validateID(id); err != nil {
		return Concept{}, err
	}
	if err := validateValues(input.Name, input.Aliases, input.ConceptType, input.Definition, input.ReviewStatus); err != nil {
		return Concept{}, err
	}
	return s.repository.Update(ctx, id, cloneUpdate(input))
}

func IsID(value string) bool { return coreid.Is(value, coreid.Entity) }

func ValidatePersisted(input Concept) error {
	if err := validateID(input.ID); err != nil {
		return err
	}
	return validateValues(input.Name, input.Aliases, input.ConceptType, input.Definition, input.ReviewStatus)
}

func validateID(value ID) error {
	if !IsID(string(value)) {
		return &ValidationError{Field: "concept_id", Message: "must equal ENT immediately followed by a canonical lowercase UUID"}
	}
	return nil
}

func validateValues(name string, aliases []string, conceptType Type, definition string, reviewStatus ReviewStatus) error {
	if strings.TrimSpace(name) == "" {
		return &ValidationError{Field: "name", Message: "must be nonblank"}
	}
	if err := validateStringSet("aliases", aliases); err != nil {
		return err
	}
	if !validType(conceptType) {
		return &ValidationError{Field: "concept_type", Message: "is unsupported"}
	}
	if strings.TrimSpace(definition) == "" {
		return &ValidationError{Field: "definition", Message: "must be nonblank"}
	}
	if reviewStatus != ReviewStatusCandidate && reviewStatus != ReviewStatusApproved {
		return &ValidationError{Field: "review_status", Message: "must be candidate or approved"}
	}
	return nil
}

func validType(value Type) bool {
	switch value {
	case TypeTechnology, TypePolicy, TypeApplication, TypeDemand, TypeBusinessModel,
		TypeCompanyEcosystem, TypeProductEcosystem, TypeEventNarrative, TypeMarketTheme:
		return true
	default:
		return false
	}
}

func validateStringSet(field string, values []string) error {
	if values == nil {
		return &ValidationError{Field: field, Message: "must be provided as an array"}
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

func cloneConcept(input Concept) Concept {
	input.Aliases = append([]string(nil), input.Aliases...)
	return input
}

type listCursor struct {
	Version int    `json:"v"`
	Name    string `json:"name"`
	ID      ID     `json:"id"`
}

func encodeListCursor(input Concept) (string, error) {
	payload, err := json.Marshal(listCursor{Version: 1, Name: input.Name, ID: input.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeListCursor(value string) (*ListKey, error) {
	if value == "" {
		return nil, nil
	}
	if len(value) > 2048 {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque Concept list cursor"}
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque Concept list cursor"}
	}
	var cursor listCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.Version != 1 || strings.TrimSpace(cursor.Name) == "" || validateID(cursor.ID) != nil {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque Concept list cursor"}
	}
	return &ListKey{Name: cursor.Name, ID: cursor.ID}, nil
}

func cloneUpdate(input Update) Update {
	input.Aliases = append([]string(nil), input.Aliases...)
	return input
}
