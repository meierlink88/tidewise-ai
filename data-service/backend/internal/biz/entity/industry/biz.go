package industry

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
	ErrNotFound    = errors.New("Industry not found")
	ErrConflict    = errors.New("Industry conflict")
	ErrPersistence = errors.New("Industry persistence failed")
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

type ReviewStatus string

const (
	ReviewStatusCandidate ReviewStatus = "candidate"
	ReviewStatusApproved  ReviewStatus = "approved"
)

type ID string

type Industry struct {
	ID                   ID
	Name                 string
	Aliases              []string
	ClassificationSystem string
	IndustryCode         string
	ParentIndustryID     *ID
	HierarchyPathCodes   []string
	Definition           string
	ReviewStatus         ReviewStatus
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Update struct {
	Name               string
	Aliases            []string
	ParentIndustryID   *ID
	HierarchyPathCodes []string
	Definition         string
	ReviewStatus       ReviewStatus
}

type ListRequest struct {
	PageSize int
	Cursor   string
}

type ListKey struct {
	ID ID
}

type ListQuery struct {
	PageSize int
	After    *ListKey
}

type ListResult struct {
	Items   []Industry
	HasMore bool
}

type Page struct {
	Items      []Industry
	NextCursor *string
}

type Repository interface {
	Create(context.Context, Industry) (Industry, error)
	Get(context.Context, ID) (Industry, error)
	List(context.Context, ListQuery) (ListResult, error)
	Update(context.Context, ID, Update) (Industry, error)
}

type UseCase struct{ repository Repository }

func NewUseCase(repository Repository) (*UseCase, error) {
	if repository == nil {
		return nil, errors.New("Industry repository is required")
	}
	return &UseCase{repository: repository}, nil
}

func (s *UseCase) Create(ctx context.Context, input Industry) (Industry, error) {
	if strings.TrimSpace(string(input.ID)) != "" {
		return Industry{}, &ValidationError{Field: "id", Message: "must be omitted because Data generates Industry IDs"}
	}
	if err := validateIndustry(input); err != nil {
		return Industry{}, err
	}
	id, err := coreid.New(coreid.Entity)
	if err != nil {
		return Industry{}, fmt.Errorf("generate Industry ID: %w", err)
	}
	input.ID = ID(id)
	return s.repository.Create(ctx, cloneIndustry(input))
}

func (s *UseCase) Get(ctx context.Context, id ID) (Industry, error) {
	if err := validateID("industry_id", id); err != nil {
		return Industry{}, err
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
			return Page{}, fmt.Errorf("encode Industry list cursor: %w", err)
		}
		page.NextCursor = &next
	}
	return page, nil
}

func (s *UseCase) Update(ctx context.Context, id ID, input Update) (Industry, error) {
	if err := validateID("industry_id", id); err != nil {
		return Industry{}, err
	}
	if err := validateMutable(input.Name, input.Aliases, input.ParentIndustryID, input.HierarchyPathCodes, input.Definition, input.ReviewStatus, id); err != nil {
		return Industry{}, err
	}
	return s.repository.Update(ctx, id, cloneUpdate(input))
}

func IsID(value string) bool { return coreid.Is(value, coreid.Entity) }

func ValidatePersisted(input Industry) error {
	if err := validateID("industry_id", input.ID); err != nil {
		return err
	}
	if input.ParentIndustryID != nil && *input.ParentIndustryID == input.ID {
		return &ValidationError{Field: "parent_industry_id", Message: "must identify a different Industry"}
	}
	return validateIndustry(input)
}

func validateIndustry(input Industry) error {
	if strings.TrimSpace(input.ClassificationSystem) == "" {
		return &ValidationError{Field: "classification_system", Message: "must be nonblank"}
	}
	if strings.TrimSpace(input.IndustryCode) == "" {
		return &ValidationError{Field: "industry_code", Message: "must be nonblank"}
	}
	if err := validateMutable(input.Name, input.Aliases, input.ParentIndustryID, input.HierarchyPathCodes, input.Definition, input.ReviewStatus, ""); err != nil {
		return err
	}
	if input.HierarchyPathCodes[len(input.HierarchyPathCodes)-1] != input.IndustryCode {
		return &ValidationError{Field: "hierarchy_path_codes", Message: "must end with industry_code"}
	}
	return nil
}

func validateMutable(name string, aliases []string, parentIndustryID *ID, path []string, definition string, reviewStatus ReviewStatus, currentID ID) error {
	if strings.TrimSpace(name) == "" {
		return &ValidationError{Field: "name", Message: "must be nonblank"}
	}
	if err := validateStringSet("aliases", aliases); err != nil {
		return err
	}
	if parentIndustryID != nil {
		if !IsID(string(*parentIndustryID)) {
			return &ValidationError{Field: "parent_industry_id", Message: "must be a stable Industry ID when present"}
		}
		if currentID != "" && *parentIndustryID == currentID {
			return &ValidationError{Field: "parent_industry_id", Message: "must identify a different Industry"}
		}
	}
	if len(path) == 0 || (parentIndustryID == nil && len(path) != 1) || (parentIndustryID != nil && len(path) < 2) {
		return &ValidationError{Field: "hierarchy_path_codes", Message: "must identify a root or extend a parent path"}
	}
	for index, code := range path {
		if strings.TrimSpace(code) == "" {
			return &ValidationError{Field: fmt.Sprintf("hierarchy_path_codes[%d]", index), Message: "must be nonblank"}
		}
	}
	if strings.TrimSpace(definition) == "" {
		return &ValidationError{Field: "definition", Message: "must be nonblank"}
	}
	if reviewStatus != ReviewStatusCandidate && reviewStatus != ReviewStatusApproved {
		return &ValidationError{Field: "review_status", Message: "must be candidate or approved"}
	}
	return nil
}

func validateID(field string, value ID) error {
	if !IsID(string(value)) {
		return &ValidationError{Field: field, Message: "must equal ENT immediately followed by a canonical lowercase UUID"}
	}
	return nil
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

func cloneIndustry(input Industry) Industry {
	input.Aliases = append([]string(nil), input.Aliases...)
	input.ParentIndustryID = cloneString(input.ParentIndustryID)
	input.HierarchyPathCodes = append([]string(nil), input.HierarchyPathCodes...)
	return input
}

func cloneUpdate(input Update) Update {
	input.Aliases = append([]string(nil), input.Aliases...)
	input.ParentIndustryID = cloneString(input.ParentIndustryID)
	input.HierarchyPathCodes = append([]string(nil), input.HierarchyPathCodes...)
	return input
}

func cloneString(value *ID) *ID {
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

func encodeListCursor(input Industry) (string, error) {
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
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque Industry list cursor"}
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque Industry list cursor"}
	}
	var cursor listCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.Version != 1 || validateID("cursor.id", cursor.ID) != nil {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque Industry list cursor"}
	}
	return &ListKey{ID: cursor.ID}, nil
}
