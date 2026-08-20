// Package storylinedomain persists independent StorylineDomain catalog facts.
package storylinedomain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type DomainCategory string

type CatalogPublicationMode string

const (
	DomainCategoryGeopolitical DomainCategory = "GEOPOLITICAL"
	DomainCategoryMacro        DomainCategory = "MACRO"
	DomainCategoryIndustry     DomainCategory = "INDUSTRY"
	DomainCategoryCorporate    DomainCategory = "CORPORATE"

	CatalogPublicationModeReconcile CatalogPublicationMode = "reconcile"
)

var (
	ErrInvalidStorylineDomain = errors.New("invalid StorylineDomain")
	ErrConflict               = errors.New("StorylineDomain conflict")
	ErrNotFound               = errors.New("StorylineDomain not found")
	ErrPersistence            = errors.New("StorylineDomain persistence failed")
)

type CreateInput struct {
	Code            string
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
	Code            string
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

type CatalogItem struct {
	Code           string         `json:"code"`
	Name           string         `json:"name"`
	NameEn         string         `json:"name_en"`
	Description    string         `json:"description"`
	DomainCategory DomainCategory `json:"domain_category"`
}

type CatalogPublication struct {
	SchemaVersion    int                    `json:"schema_version"`
	PublicationMode  CatalogPublicationMode `json:"publication_mode"`
	StorylineDomains []CatalogItem          `json:"storyline_domains"`
}

func LoadCatalog(ctx context.Context, path string) (CatalogPublication, error) {
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return CatalogPublication{}, fmt.Errorf("open StorylineDomain catalog: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var publication CatalogPublication
	if err := decoder.Decode(&publication); err != nil {
		return CatalogPublication{}, fmt.Errorf("decode StorylineDomain catalog: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return CatalogPublication{}, fmt.Errorf("decode StorylineDomain catalog trailing data: %w", err)
	}
	if err := validateCatalog(publication); err != nil {
		return CatalogPublication{}, err
	}
	return publication, nil
}

func PublishCatalog(ctx context.Context, db *sql.DB, publication CatalogPublication) error {
	if db == nil {
		return errors.New("StorylineDomain catalog database is required")
	}
	if err := validateCatalog(publication); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, item := range publication.StorylineDomains {
		id, err := coreid.Derive(coreid.StorylineDomain, "storyline-domain", item.Code)
		if err != nil {
			return ErrInvalidStorylineDomain
		}
		var publishedID string
		err = tx.QueryRowContext(ctx, `
INSERT INTO storyline_domains (
    id, code, name, name_en, description, scope_definition, domain_category, is_active
) VALUES ($1, $2, $3, $4, $5, $5, $6, TRUE)
ON CONFLICT (code) DO UPDATE SET
    name = excluded.name,
    name_en = excluded.name_en,
    description = excluded.description,
    scope_definition = excluded.scope_definition,
    domain_category = excluded.domain_category,
    is_active = excluded.is_active,
    updated_at = CASE
        WHEN (storyline_domains.name, storyline_domains.name_en, storyline_domains.description,
              storyline_domains.scope_definition, storyline_domains.domain_category, storyline_domains.is_active)
          IS DISTINCT FROM
             (excluded.name, excluded.name_en, excluded.description,
              excluded.scope_definition, excluded.domain_category, excluded.is_active)
        THEN now()
        ELSE storyline_domains.updated_at
    END
WHERE storyline_domains.id = excluded.id
RETURNING id`, id, item.Code, item.Name, item.NameEn, item.Description, string(item.DomainCategory)).Scan(&publishedID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return classifyWriteError(err)
		}
		if publishedID != id {
			return ErrConflict
		}
	}
	if err := tx.Commit(); err != nil {
		return classifyWriteError(err)
	}
	return nil
}

func validateCatalog(publication CatalogPublication) error {
	if publication.SchemaVersion != 1 || publication.PublicationMode != CatalogPublicationModeReconcile ||
		len(publication.StorylineDomains) != 35 {
		return ErrInvalidStorylineDomain
	}
	seenCodes := make(map[string]struct{}, len(publication.StorylineDomains))
	categoryCounts := make(map[DomainCategory]int, 4)
	for _, item := range publication.StorylineDomains {
		if !validCatalogFields(item.Code, item.Name, item.NameEn, item.Description, item.DomainCategory) {
			return ErrInvalidStorylineDomain
		}
		if _, duplicate := seenCodes[item.Code]; duplicate {
			return ErrInvalidStorylineDomain
		}
		seenCodes[item.Code] = struct{}{}
		categoryCounts[item.DomainCategory]++
	}
	if categoryCounts[DomainCategoryGeopolitical] != 7 || categoryCounts[DomainCategoryMacro] != 12 ||
		categoryCounts[DomainCategoryIndustry] != 8 || categoryCounts[DomainCategoryCorporate] != 8 {
		return ErrInvalidStorylineDomain
	}
	return nil
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
    id, code, name, name_en, description, scope_definition, domain_category, is_active
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING `+storylineDomainColumns,
		id, input.Code, input.Name, input.NameEn, input.Description, input.ScopeDefinition,
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
id, code, name, name_en, description, scope_definition, domain_category::text,
is_active, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanStorylineDomain(row rowScanner) (StorylineDomain, error) {
	var result StorylineDomain
	var category string
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &result.NameEn, &result.Description,
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
	if !validCatalogFields(input.Code, input.Name, input.NameEn, input.Description, input.DomainCategory) ||
		strings.TrimSpace(input.ScopeDefinition) == "" {
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
		!validCatalogFields(input.Code, input.Name, input.NameEn, input.Description, input.DomainCategory) ||
		strings.TrimSpace(input.ScopeDefinition) == "" || input.CreatedAt.IsZero() ||
		input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidStorylineDomain
	}
	return nil
}

func validCatalogFields(code, name, nameEn, description string, category DomainCategory) bool {
	return validCode(code) && validRequiredText(name, 50) && validRequiredText(nameEn, 50) &&
		strings.TrimSpace(description) != "" && validDomainCategory(category)
}

func validRequiredText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes
}

func validCode(value string) bool {
	if len(value) == 0 || len(value) > 30 || value[0] < 'A' || value[0] > 'Z' {
		return false
	}
	for index := 1; index < len(value); index++ {
		character := value[index]
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
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
	case "23505":
		return ErrConflict
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
