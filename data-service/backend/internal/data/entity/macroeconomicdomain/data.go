// Package macroeconomicdomain persists the reviewed macroeconomic domain catalog.
package macroeconomicdomain

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
	ErrInvalidMacroEconomicDomain = errors.New("invalid MacroEconomicDomain")
	ErrConflict                   = errors.New("MacroEconomicDomain conflict")
	ErrNotFound                   = errors.New("MacroEconomicDomain not found")
	ErrPersistence                = errors.New("MacroEconomicDomain persistence failed")
	codePattern                   = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,49}$`)
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

type MacroEconomicDomain struct {
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
		return nil, errors.New("MacroEconomicDomain database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (MacroEconomicDomain, error) {
	if err := validateInput(input.Code, input.Name, input.Description, input.Tactics); err != nil {
		return MacroEconomicDomain{}, err
	}
	id, err := coreid.New(coreid.MacroEconomicDomain)
	if err != nil {
		return MacroEconomicDomain{}, ErrPersistence
	}
	tactics, err := json.Marshal(input.Tactics)
	if err != nil {
		return MacroEconomicDomain{}, ErrInvalidMacroEconomicDomain
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO macro_economics_domain (id, code, name, description, tactics)
VALUES ($1, $2, $3, $4, $5::jsonb)
RETURNING `+macroEconomicDomainColumns,
		id, input.Code, input.Name, input.Description, tactics,
	)
	created, err := scanMacroEconomicDomain(row)
	if err != nil {
		return MacroEconomicDomain{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (MacroEconomicDomain, error) {
	if !coreid.Is(id, coreid.MacroEconomicDomain) {
		return MacroEconomicDomain{}, ErrInvalidMacroEconomicDomain
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+macroEconomicDomainColumns+` FROM macro_economics_domain WHERE id = $1`, id)
	result, err := scanMacroEconomicDomain(row)
	if err != nil {
		return MacroEconomicDomain{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context) ([]MacroEconomicDomain, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT `+macroEconomicDomainColumns+`
FROM macro_economics_domain
ORDER BY code ASC, id ASC`)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]MacroEconomicDomain, 0)
	for rows.Next() {
		item, err := scanMacroEconomicDomain(rows)
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

func (s *Store) Update(ctx context.Context, input UpdateInput) (MacroEconomicDomain, error) {
	if !coreid.Is(input.ID, coreid.MacroEconomicDomain) ||
		validateMutableInput(input.Name, input.Description, input.Tactics) != nil {
		return MacroEconomicDomain{}, ErrInvalidMacroEconomicDomain
	}
	tactics, err := json.Marshal(input.Tactics)
	if err != nil {
		return MacroEconomicDomain{}, ErrInvalidMacroEconomicDomain
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE macro_economics_domain
SET name = $2, description = $3, tactics = $4::jsonb, updated_at = now()
WHERE id = $1
RETURNING `+macroEconomicDomainColumns,
		input.ID, input.Name, input.Description, tactics,
	)
	updated, err := scanMacroEconomicDomain(row)
	if err != nil {
		return MacroEconomicDomain{}, classifyWriteError(err)
	}
	return updated, nil
}

const macroEconomicDomainColumns = `
id, code, name, description, tactics, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanMacroEconomicDomain(row rowScanner) (MacroEconomicDomain, error) {
	var result MacroEconomicDomain
	var tacticsJSON []byte
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &result.Description,
		&tacticsJSON, &result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return MacroEconomicDomain{}, err
	}
	tactics, err := decodeTactics(tacticsJSON)
	if err != nil {
		return MacroEconomicDomain{}, err
	}
	result.Tactics = tactics
	if err := validateStored(result); err != nil {
		return MacroEconomicDomain{}, err
	}
	return result, nil
}

func decodeTactics(payload []byte) ([]Tactic, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var tactics []Tactic
	if err := decoder.Decode(&tactics); err != nil {
		return nil, ErrInvalidMacroEconomicDomain
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, ErrInvalidMacroEconomicDomain
	}
	if !validTactics(tactics) {
		return nil, ErrInvalidMacroEconomicDomain
	}
	return tactics, nil
}

func validateInput(code, name, description string, tactics []Tactic) error {
	if !codePattern.MatchString(code) || validateMutableInput(name, description, tactics) != nil {
		return ErrInvalidMacroEconomicDomain
	}
	return nil
}

func validateMutableInput(name, description string, tactics []Tactic) error {
	if !validRequiredText(name, 50) || strings.TrimSpace(description) == "" || !validTactics(tactics) {
		return ErrInvalidMacroEconomicDomain
	}
	return nil
}

func validateStored(input MacroEconomicDomain) error {
	if !coreid.Is(input.ID, coreid.MacroEconomicDomain) ||
		validateInput(input.Code, input.Name, input.Description, input.Tactics) != nil ||
		input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidMacroEconomicDomain
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
		return ErrInvalidMacroEconomicDomain
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
	if errors.Is(err, ErrInvalidMacroEconomicDomain) {
		return fmt.Errorf("%w: invalid persisted MacroEconomicDomain", ErrPersistence)
	}
	return ErrPersistence
}
