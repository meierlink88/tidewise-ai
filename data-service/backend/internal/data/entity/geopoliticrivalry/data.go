// Package geopoliticrivalry persists geopolitical storylines and their primary domain.
package geopoliticrivalry

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

var (
	ErrInvalidGeopoliticRivalry = errors.New("invalid GeopoliticRivalry")
	ErrConflict                 = errors.New("GeopoliticRivalry conflict")
	ErrNotFound                 = errors.New("GeopoliticRivalry not found")
	ErrPersistence              = errors.New("GeopoliticRivalry persistence failed")
)

type CreateInput struct {
	Name               string
	Category           string
	GeopoliticDomainID string
	CoreProposition    string
	CoreActors         string
	MainTransmission   string
}

type UpdateInput struct {
	ID                 string
	Name               string
	Category           string
	GeopoliticDomainID string
	CoreProposition    string
	CoreActors         string
	MainTransmission   string
}

type GeopoliticRivalry struct {
	ID                 string
	Name               string
	Category           string
	GeopoliticDomainID string
	CoreProposition    string
	CoreActors         string
	MainTransmission   string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Filter struct {
	GeopoliticDomainID *string
	Category           *string
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("GeopoliticRivalry database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (GeopoliticRivalry, error) {
	if err := validateInput(input); err != nil {
		return GeopoliticRivalry{}, err
	}
	id, err := coreid.New(coreid.GeopoliticRivalry)
	if err != nil {
		return GeopoliticRivalry{}, ErrPersistence
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO geopolitic_rivalries (
    id, name, category, geopolitic_domain_id,
    core_proposition, core_actors, main_transmission
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING `+geopoliticRivalryColumns,
		id, input.Name, input.Category, input.GeopoliticDomainID,
		input.CoreProposition, input.CoreActors, input.MainTransmission,
	)
	created, err := scanGeopoliticRivalry(row)
	if err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (GeopoliticRivalry, error) {
	if !coreid.Is(id, coreid.GeopoliticRivalry) {
		return GeopoliticRivalry{}, ErrInvalidGeopoliticRivalry
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+geopoliticRivalryColumns+` FROM geopolitic_rivalries WHERE id = $1`, id)
	result, err := scanGeopoliticRivalry(row)
	if err != nil {
		return GeopoliticRivalry{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, filter Filter) ([]GeopoliticRivalry, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+geopoliticRivalryColumns+`
FROM geopolitic_rivalries
WHERE ($1::text IS NULL OR geopolitic_domain_id = $1)
  AND ($2::text IS NULL OR category = $2)
ORDER BY name ASC, id ASC`, nullableString(filter.GeopoliticDomainID), nullableString(filter.Category))
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]GeopoliticRivalry, 0)
	for rows.Next() {
		item, err := scanGeopoliticRivalry(rows)
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

func (s *Store) Update(ctx context.Context, input UpdateInput) (GeopoliticRivalry, error) {
	if !coreid.Is(input.ID, coreid.GeopoliticRivalry) || validateInput(CreateInput{
		Name: input.Name, Category: input.Category, GeopoliticDomainID: input.GeopoliticDomainID,
		CoreProposition: input.CoreProposition, CoreActors: input.CoreActors,
		MainTransmission: input.MainTransmission,
	}) != nil {
		return GeopoliticRivalry{}, ErrInvalidGeopoliticRivalry
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE geopolitic_rivalries
SET name = $2, category = $3, geopolitic_domain_id = $4,
    core_proposition = $5, core_actors = $6, main_transmission = $7,
    updated_at = now()
WHERE id = $1
RETURNING `+geopoliticRivalryColumns,
		input.ID, input.Name, input.Category, input.GeopoliticDomainID,
		input.CoreProposition, input.CoreActors, input.MainTransmission,
	)
	updated, err := scanGeopoliticRivalry(row)
	if err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	return updated, nil
}

const geopoliticRivalryColumns = `
id, name, category, geopolitic_domain_id, core_proposition,
core_actors, main_transmission, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanGeopoliticRivalry(row rowScanner) (GeopoliticRivalry, error) {
	var result GeopoliticRivalry
	if err := row.Scan(
		&result.ID, &result.Name, &result.Category, &result.GeopoliticDomainID,
		&result.CoreProposition, &result.CoreActors, &result.MainTransmission,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return GeopoliticRivalry{}, err
	}
	if err := validateStored(result); err != nil {
		return GeopoliticRivalry{}, err
	}
	return result, nil
}

func validateInput(input CreateInput) error {
	if !validRequiredText(input.Name, 100) || !validRequiredText(input.Category, 100) ||
		!coreid.Is(input.GeopoliticDomainID, coreid.GeopoliticDomain) ||
		strings.TrimSpace(input.CoreProposition) == "" || strings.TrimSpace(input.CoreActors) == "" ||
		strings.TrimSpace(input.MainTransmission) == "" {
		return ErrInvalidGeopoliticRivalry
	}
	return nil
}

func validateFilter(filter Filter) error {
	if filter.GeopoliticDomainID != nil && !coreid.Is(*filter.GeopoliticDomainID, coreid.GeopoliticDomain) {
		return ErrInvalidGeopoliticRivalry
	}
	if filter.Category != nil && strings.TrimSpace(*filter.Category) == "" {
		return ErrInvalidGeopoliticRivalry
	}
	return nil
}

func validateStored(input GeopoliticRivalry) error {
	if !coreid.Is(input.ID, coreid.GeopoliticRivalry) || validateInput(CreateInput{
		Name: input.Name, Category: input.Category, GeopoliticDomainID: input.GeopoliticDomainID,
		CoreProposition: input.CoreProposition, CoreActors: input.CoreActors,
		MainTransmission: input.MainTransmission,
	}) != nil || input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidGeopoliticRivalry
	}
	return nil
}

func validRequiredText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
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
	case "22001", "23502", "23503", "23514":
		return ErrInvalidGeopoliticRivalry
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
	if errors.Is(err, ErrInvalidGeopoliticRivalry) {
		return fmt.Errorf("%w: invalid persisted GeopoliticRivalry", ErrPersistence)
	}
	return ErrPersistence
}
