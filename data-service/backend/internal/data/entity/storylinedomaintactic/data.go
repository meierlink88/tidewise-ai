// Package storylinedomaintactic persists independent StorylineDomainTactic catalog facts.
package storylinedomaintactic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

var (
	ErrInvalidStorylineDomainTactic = errors.New("invalid StorylineDomainTactic")
	ErrConflict                     = errors.New("StorylineDomainTactic conflict")
	ErrNotFound                     = errors.New("StorylineDomainTactic not found")
	ErrPersistence                  = errors.New("StorylineDomainTactic persistence failed")
	tacticKeyPattern                = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,29}$`)
)

type CreateInput struct {
	Key         string
	Name        string
	NameEn      string
	Description string
}

type UpdateInput struct {
	ID          string
	Name        string
	NameEn      string
	Description string
}

type StorylineDomainTactic struct {
	ID          string
	Key         string
	Name        string
	NameEn      string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("StorylineDomainTactic database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (StorylineDomainTactic, error) {
	if err := validateCreate(input); err != nil {
		return StorylineDomainTactic{}, err
	}
	id, err := coreid.New(coreid.StorylineDomainTactic)
	if err != nil {
		return StorylineDomainTactic{}, ErrPersistence
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO storyline_domain_tactics (id, key, name, name_en, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING `+storylineDomainTacticColumns,
		id, input.Key, input.Name, input.NameEn, input.Description,
	)
	created, err := scanStorylineDomainTactic(row)
	if err != nil {
		return StorylineDomainTactic{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (StorylineDomainTactic, error) {
	if !coreid.Is(id, coreid.StorylineDomainTactic) {
		return StorylineDomainTactic{}, ErrInvalidStorylineDomainTactic
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+storylineDomainTacticColumns+` FROM storyline_domain_tactics WHERE id = $1`, id)
	result, err := scanStorylineDomainTactic(row)
	if err != nil {
		return StorylineDomainTactic{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context) ([]StorylineDomainTactic, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT `+storylineDomainTacticColumns+`
FROM storyline_domain_tactics
ORDER BY key ASC, id ASC`)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]StorylineDomainTactic, 0)
	for rows.Next() {
		item, err := scanStorylineDomainTactic(rows)
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

func (s *Store) Update(ctx context.Context, input UpdateInput) (StorylineDomainTactic, error) {
	if err := validateUpdate(input); err != nil {
		return StorylineDomainTactic{}, err
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE storyline_domain_tactics
SET name = $2,
    name_en = $3,
    description = $4,
    updated_at = now()
WHERE id = $1
RETURNING `+storylineDomainTacticColumns,
		input.ID, input.Name, input.NameEn, input.Description,
	)
	updated, err := scanStorylineDomainTactic(row)
	if err != nil {
		return StorylineDomainTactic{}, classifyWriteError(err)
	}
	return updated, nil
}

const storylineDomainTacticColumns = `
id, key, name, name_en, description, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanStorylineDomainTactic(row rowScanner) (StorylineDomainTactic, error) {
	var result StorylineDomainTactic
	if err := row.Scan(
		&result.ID, &result.Key, &result.Name, &result.NameEn,
		&result.Description, &result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return StorylineDomainTactic{}, err
	}
	if err := validateStored(result); err != nil {
		return StorylineDomainTactic{}, err
	}
	return result, nil
}

func validateCreate(input CreateInput) error {
	if !validKey(input.Key) || !validRequiredText(input.Name, 50) ||
		!validRequiredText(input.NameEn, 50) || strings.TrimSpace(input.Description) == "" {
		return ErrInvalidStorylineDomainTactic
	}
	return nil
}

func validateUpdate(input UpdateInput) error {
	if !coreid.Is(input.ID, coreid.StorylineDomainTactic) ||
		!validRequiredText(input.Name, 50) || !validRequiredText(input.NameEn, 50) ||
		strings.TrimSpace(input.Description) == "" {
		return ErrInvalidStorylineDomainTactic
	}
	return nil
}

func validateStored(input StorylineDomainTactic) error {
	if !coreid.Is(input.ID, coreid.StorylineDomainTactic) || !validKey(input.Key) ||
		!validRequiredText(input.Name, 50) || !validRequiredText(input.NameEn, 50) ||
		strings.TrimSpace(input.Description) == "" || input.CreatedAt.IsZero() ||
		input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidStorylineDomainTactic
	}
	return nil
}

func validKey(value string) bool {
	return tacticKeyPattern.MatchString(value)
}

func validRequiredText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes
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
	case "22001", "23502", "23514":
		return ErrInvalidStorylineDomainTactic
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
	if errors.Is(err, ErrInvalidStorylineDomainTactic) {
		return fmt.Errorf("%w: invalid persisted StorylineDomainTactic", ErrPersistence)
	}
	return ErrPersistence
}
