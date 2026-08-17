package chainnode

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
	ErrNotFound    = errors.New("ChainNode not found")
	ErrConflict    = errors.New("ChainNode conflict")
	ErrPersistence = errors.New("ChainNode persistence failed")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

type ReviewStatus string

const (
	ReviewStatusCandidate ReviewStatus = "candidate"
	ReviewStatusApproved  ReviewStatus = "approved"
)

type ID string

type ChainNode struct {
	ID           ID
	Name         string
	Aliases      []string
	Definition   string
	ReviewStatus ReviewStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Update struct {
	Name         string
	Aliases      []string
	Definition   string
	ReviewStatus ReviewStatus
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
	Items   []ChainNode
	HasMore bool
}
type Page struct {
	Items      []ChainNode
	NextCursor *string
}

type Repository interface {
	Create(context.Context, ChainNode) (ChainNode, error)
	Get(context.Context, ID) (ChainNode, error)
	List(context.Context, ListQuery) (ListResult, error)
	Update(context.Context, ID, Update) (ChainNode, error)
}

type UseCase struct{ repository Repository }

func NewUseCase(repository Repository) (*UseCase, error) {
	if repository == nil {
		return nil, errors.New("ChainNode repository is required")
	}
	return &UseCase{repository: repository}, nil
}

func (s *UseCase) Create(ctx context.Context, input ChainNode) (ChainNode, error) {
	if strings.TrimSpace(string(input.ID)) != "" {
		return ChainNode{}, &ValidationError{Field: "id", Message: "must be omitted because Data generates ChainNode IDs"}
	}
	if err := validateValues(input.Name, input.Aliases, input.Definition, input.ReviewStatus); err != nil {
		return ChainNode{}, err
	}
	id, err := coreid.New(coreid.Entity)
	if err != nil {
		return ChainNode{}, fmt.Errorf("generate ChainNode ID: %w", err)
	}
	input.ID = ID(id)
	return s.repository.Create(ctx, cloneChainNode(input))
}

func (s *UseCase) Get(ctx context.Context, id ID) (ChainNode, error) {
	if err := validateID(id); err != nil {
		return ChainNode{}, err
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
			return Page{}, fmt.Errorf("encode ChainNode list cursor: %w", err)
		}
		page.NextCursor = &next
	}
	return page, nil
}

func (s *UseCase) Update(ctx context.Context, id ID, input Update) (ChainNode, error) {
	if err := validateID(id); err != nil {
		return ChainNode{}, err
	}
	if err := validateValues(input.Name, input.Aliases, input.Definition, input.ReviewStatus); err != nil {
		return ChainNode{}, err
	}
	input.Aliases = append([]string{}, input.Aliases...)
	return s.repository.Update(ctx, id, input)
}

func IsID(value string) bool { return coreid.Is(value, coreid.Entity) }

func ValidatePersisted(input ChainNode) error {
	if err := validateID(input.ID); err != nil {
		return err
	}
	return validateValues(input.Name, input.Aliases, input.Definition, input.ReviewStatus)
}

func validateID(value ID) error {
	if !IsID(string(value)) {
		return &ValidationError{Field: "chain_node_id", Message: "must equal ENT immediately followed by a canonical lowercase UUID"}
	}
	return nil
}

func validateValues(name string, aliases []string, definition string, reviewStatus ReviewStatus) error {
	if strings.TrimSpace(name) == "" {
		return &ValidationError{Field: "name", Message: "must be nonblank"}
	}
	if aliases == nil {
		return &ValidationError{Field: "aliases", Message: "must be provided as an array"}
	}
	seen := make(map[string]struct{}, len(aliases))
	for index, alias := range aliases {
		if strings.TrimSpace(alias) == "" {
			return &ValidationError{Field: fmt.Sprintf("aliases[%d]", index), Message: "must be nonblank"}
		}
		if _, duplicate := seen[alias]; duplicate {
			return &ValidationError{Field: fmt.Sprintf("aliases[%d]", index), Message: "must be unique"}
		}
		seen[alias] = struct{}{}
	}
	if strings.TrimSpace(definition) == "" {
		return &ValidationError{Field: "definition", Message: "must be nonblank"}
	}
	if reviewStatus != ReviewStatusCandidate && reviewStatus != ReviewStatusApproved {
		return &ValidationError{Field: "review_status", Message: "must be candidate or approved"}
	}
	return nil
}

func cloneChainNode(input ChainNode) ChainNode {
	input.Aliases = append([]string{}, input.Aliases...)
	return input
}

type listCursor struct {
	Version int `json:"v"`
	ID      ID  `json:"id"`
}

func encodeListCursor(input ChainNode) (string, error) {
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
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque ChainNode list cursor"}
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque ChainNode list cursor"}
	}
	var cursor listCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.Version != 1 || validateID(cursor.ID) != nil {
		return nil, &ValidationError{Field: "cursor", Message: "must be an opaque ChainNode list cursor"}
	}
	return &ListKey{ID: cursor.ID}, nil
}
