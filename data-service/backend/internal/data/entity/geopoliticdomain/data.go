// Package geopoliticdomain persists the reviewed geopolitical domain catalog.
package geopoliticdomain

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

var (
	ErrInvalidGeopoliticDomain = errors.New("invalid GeopoliticDomain")
	ErrConflict                = errors.New("GeopoliticDomain conflict")
	ErrNotFound                = errors.New("GeopoliticDomain not found")
	ErrPersistence             = errors.New("GeopoliticDomain persistence failed")
	codePattern                = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,49}$`)
)

type Tactic struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateInput struct {
	Code        string
	Name        string
	Description string
	Tactics     []Tactic
}

type UpdateInput struct {
	ID          string
	Name        string
	Description string
	Tactics     []Tactic
}

type GeopoliticDomain struct {
	ID          string
	Code        string
	Name        string
	Description string
	Tactics     []Tactic
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("GeopoliticDomain database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (GeopoliticDomain, error) {
	if err := validateInput(input.Code, input.Name, input.Description, input.Tactics); err != nil {
		return GeopoliticDomain{}, err
	}
	id, err := coreid.New(coreid.GeopoliticDomain)
	if err != nil {
		return GeopoliticDomain{}, ErrPersistence
	}
	tactics, err := json.Marshal(input.Tactics)
	if err != nil {
		return GeopoliticDomain{}, ErrInvalidGeopoliticDomain
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO geopolitic_domains (id, code, name, description, tactics)
VALUES ($1, $2, $3, $4, $5::jsonb)
RETURNING `+geopoliticDomainColumns,
		id, input.Code, input.Name, input.Description, tactics,
	)
	created, err := scanGeopoliticDomain(row)
	if err != nil {
		return GeopoliticDomain{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (GeopoliticDomain, error) {
	if !coreid.Is(id, coreid.GeopoliticDomain) {
		return GeopoliticDomain{}, ErrInvalidGeopoliticDomain
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+geopoliticDomainColumns+` FROM geopolitic_domains WHERE id = $1`, id)
	result, err := scanGeopoliticDomain(row)
	if err != nil {
		return GeopoliticDomain{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context) ([]GeopoliticDomain, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT `+geopoliticDomainColumns+`
FROM geopolitic_domains
ORDER BY code ASC, id ASC`)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]GeopoliticDomain, 0)
	for rows.Next() {
		item, err := scanGeopoliticDomain(rows)
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

func (s *Store) Update(ctx context.Context, input UpdateInput) (GeopoliticDomain, error) {
	if !coreid.Is(input.ID, coreid.GeopoliticDomain) ||
		validateMutableInput(input.Name, input.Description, input.Tactics) != nil {
		return GeopoliticDomain{}, ErrInvalidGeopoliticDomain
	}
	tactics, err := json.Marshal(input.Tactics)
	if err != nil {
		return GeopoliticDomain{}, ErrInvalidGeopoliticDomain
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE geopolitic_domains
SET name = $2, description = $3, tactics = $4::jsonb, updated_at = now()
WHERE id = $1
RETURNING `+geopoliticDomainColumns,
		input.ID, input.Name, input.Description, tactics,
	)
	updated, err := scanGeopoliticDomain(row)
	if err != nil {
		return GeopoliticDomain{}, classifyWriteError(err)
	}
	return updated, nil
}

const geopoliticDomainColumns = `
id, code, name, description, tactics, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanGeopoliticDomain(row rowScanner) (GeopoliticDomain, error) {
	var result GeopoliticDomain
	var tacticsJSON []byte
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &result.Description,
		&tacticsJSON, &result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return GeopoliticDomain{}, err
	}
	tactics, err := decodeTactics(tacticsJSON)
	if err != nil {
		return GeopoliticDomain{}, err
	}
	result.Tactics = tactics
	if err := validateStored(result); err != nil {
		return GeopoliticDomain{}, err
	}
	return result, nil
}

func decodeTactics(payload []byte) ([]Tactic, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var tactics []Tactic
	if err := decoder.Decode(&tactics); err != nil {
		return nil, ErrInvalidGeopoliticDomain
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, ErrInvalidGeopoliticDomain
	}
	if !validTactics(tactics) {
		return nil, ErrInvalidGeopoliticDomain
	}
	return tactics, nil
}

func validateInput(code, name, description string, tactics []Tactic) error {
	if !codePattern.MatchString(code) || validateMutableInput(name, description, tactics) != nil {
		return ErrInvalidGeopoliticDomain
	}
	return nil
}

func validateMutableInput(name, description string, tactics []Tactic) error {
	if !validRequiredText(name, 50) || strings.TrimSpace(description) == "" || !validTactics(tactics) {
		return ErrInvalidGeopoliticDomain
	}
	return nil
}

func validateStored(input GeopoliticDomain) error {
	if !coreid.Is(input.ID, coreid.GeopoliticDomain) ||
		validateInput(input.Code, input.Name, input.Description, input.Tactics) != nil ||
		input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidGeopoliticDomain
	}
	return nil
}

func validTactics(tactics []Tactic) bool {
	if len(tactics) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(tactics))
	for _, tactic := range tactics {
		if !validRequiredText(tactic.Name, 50) || strings.TrimSpace(tactic.Description) == "" {
			return false
		}
		if _, duplicate := seen[tactic.Name]; duplicate {
			return false
		}
		seen[tactic.Name] = struct{}{}
	}
	return true
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
	case "22001", "22P02", "23502", "23503", "23514":
		return ErrInvalidGeopoliticDomain
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
	if errors.Is(err, ErrInvalidGeopoliticDomain) {
		return fmt.Errorf("%w: invalid persisted GeopoliticDomain", ErrPersistence)
	}
	return ErrPersistence
}
