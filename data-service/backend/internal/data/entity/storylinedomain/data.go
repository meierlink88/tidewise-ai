// Package storylinedomain persists independent StorylineDomain catalog facts.
package storylinedomain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type DomainCategory string

const (
	DomainCategoryGeopolitical DomainCategory = "GEOPOLITICAL"
	DomainCategoryMacro        DomainCategory = "MACRO"
	DomainCategoryIndustry     DomainCategory = "INDUSTRY"
	DomainCategoryCorporate    DomainCategory = "CORPORATE"
)

var (
	ErrInvalidStorylineDomain = errors.New("invalid StorylineDomain")
	ErrNotFound               = errors.New("StorylineDomain not found")
	ErrPersistence            = errors.New("StorylineDomain persistence failed")
)

type CreateInput struct {
	Name            string
	NameEn          string
	Description     string
	ScopeDefinition string
	DomainCategory  DomainCategory
	IsActive        *bool
}

type UpdateInput struct {
	ID              string
	Name            string
	NameEn          string
	Description     string
	ScopeDefinition string
	DomainCategory  DomainCategory
	IsActive        bool
}

type StorylineDomain struct {
	ID              string
	Name            string
	NameEn          string
	Description     string
	ScopeDefinition string
	DomainCategory  DomainCategory
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Filter struct {
	DomainCategory *DomainCategory
	IsActive       *bool
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("StorylineDomain database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (StorylineDomain, error) {
	if err := validateCreate(input); err != nil {
		return StorylineDomain{}, err
	}
	id, err := coreid.New(coreid.StorylineDomain)
	if err != nil {
		return StorylineDomain{}, ErrPersistence
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO storyline_domains (
    id, name, name_en, description, scope_definition, domain_category, is_active
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING `+storylineDomainColumns,
		id, input.Name, input.NameEn, input.Description, input.ScopeDefinition,
		string(input.DomainCategory), isActive,
	)
	created, err := scanStorylineDomain(row)
	if err != nil {
		return StorylineDomain{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (StorylineDomain, error) {
	if !coreid.Is(id, coreid.StorylineDomain) {
		return StorylineDomain{}, ErrInvalidStorylineDomain
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+storylineDomainColumns+` FROM storyline_domains WHERE id = $1`, id)
	result, err := scanStorylineDomain(row)
	if err != nil {
		return StorylineDomain{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, filter Filter) ([]StorylineDomain, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+storylineDomainColumns+`
FROM storyline_domains
WHERE ($1::storyline_domain_category IS NULL OR domain_category = $1::storyline_domain_category)
  AND ($2::boolean IS NULL OR is_active = $2::boolean)
ORDER BY name_en ASC, name ASC, id ASC`, nullableDomainCategory(filter.DomainCategory), filter.IsActive)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]StorylineDomain, 0)
	for rows.Next() {
		item, err := scanStorylineDomain(rows)
		if err != nil {
			return nil, classifyReadError(err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) Update(ctx context.Context, input UpdateInput) (StorylineDomain, error) {
	if err := validateUpdate(input); err != nil {
		return StorylineDomain{}, err
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE storyline_domains
SET name = $2,
    name_en = $3,
    description = $4,
    scope_definition = $5,
    domain_category = $6,
    is_active = $7,
    updated_at = now()
WHERE id = $1
RETURNING `+storylineDomainColumns,
		input.ID, input.Name, input.NameEn, input.Description, input.ScopeDefinition,
		string(input.DomainCategory), input.IsActive,
	)
	updated, err := scanStorylineDomain(row)
	if err != nil {
		return StorylineDomain{}, classifyWriteError(err)
	}
	return updated, nil
}

const storylineDomainColumns = `
id, name, name_en, description, scope_definition, domain_category::text,
is_active, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanStorylineDomain(row rowScanner) (StorylineDomain, error) {
	var result StorylineDomain
	var category string
	if err := row.Scan(
		&result.ID, &result.Name, &result.NameEn, &result.Description,
		&result.ScopeDefinition, &category, &result.IsActive,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return StorylineDomain{}, err
	}
	result.DomainCategory = DomainCategory(category)
	if err := validateStored(result); err != nil {
		return StorylineDomain{}, err
	}
	return result, nil
}

func validateCreate(input CreateInput) error {
	if !validRequiredText(input.Name, 50) || !validRequiredText(input.NameEn, 50) ||
		strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.ScopeDefinition) == "" ||
		!validDomainCategory(input.DomainCategory) {
		return ErrInvalidStorylineDomain
	}
	return nil
}

func validateUpdate(input UpdateInput) error {
	if !coreid.Is(input.ID, coreid.StorylineDomain) ||
		!validRequiredText(input.Name, 50) || !validRequiredText(input.NameEn, 50) ||
		strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.ScopeDefinition) == "" ||
		!validDomainCategory(input.DomainCategory) {
		return ErrInvalidStorylineDomain
	}
	return nil
}

func validateFilter(filter Filter) error {
	if filter.DomainCategory != nil && !validDomainCategory(*filter.DomainCategory) {
		return ErrInvalidStorylineDomain
	}
	return nil
}

func validateStored(input StorylineDomain) error {
	if !coreid.Is(input.ID, coreid.StorylineDomain) ||
		!validRequiredText(input.Name, 50) || !validRequiredText(input.NameEn, 50) ||
		strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.ScopeDefinition) == "" ||
		!validDomainCategory(input.DomainCategory) || input.CreatedAt.IsZero() ||
		input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidStorylineDomain
	}
	return nil
}

func validRequiredText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes
}

func validDomainCategory(value DomainCategory) bool {
	switch value {
	case DomainCategoryGeopolitical, DomainCategoryMacro, DomainCategoryIndustry, DomainCategoryCorporate:
		return true
	default:
		return false
	}
}

func nullableDomainCategory(value *DomainCategory) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return ErrPersistence
	}
	switch postgresError.Code {
	case "22001", "22P02", "23502", "23514":
		return ErrInvalidStorylineDomain
	default:
		return ErrPersistence
	}
}

func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if errors.Is(err, ErrInvalidStorylineDomain) {
		return fmt.Errorf("%w: invalid persisted StorylineDomain", ErrPersistence)
	}
	return ErrPersistence
}
