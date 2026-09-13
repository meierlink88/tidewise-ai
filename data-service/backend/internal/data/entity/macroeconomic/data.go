// Package macroeconomic persists macroeconomic storylines and their domain memberships.
package macroeconomic

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

var (
	ErrInvalidMacroEconomic = errors.New("invalid MacroEconomic")
	ErrConflict             = errors.New("MacroEconomic conflict")
	ErrNotFound             = errors.New("MacroEconomic not found")
	ErrPersistence          = errors.New("MacroEconomic persistence failed")
)

type CreateInput struct {
	Name                   string
	MacroEconomicDomainIDs []string
	CoreProposition        string
	CandidateAssets        []string
}

type UpdateInput struct {
	ID                     string
	Name                   string
	MacroEconomicDomainIDs []string
	CoreProposition        string
	CandidateAssets        []string
}

type MacroEconomic struct {
	ShortName              *string
	ID                     string
	Name                   string
	MacroEconomicDomainIDs []string
	CoreProposition        string
	CandidateAssets        []string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type Filter struct {
	MacroEconomicDomainID *string
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("MacroEconomic database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (MacroEconomic, error) {
	if err := validateInput(input); err != nil {
		return MacroEconomic{}, err
	}
	id, err := coreid.New(coreid.MacroEconomic)
	if err != nil {
		return MacroEconomic{}, ErrPersistence
	}
	candidateAssets, err := json.Marshal(input.CandidateAssets)
	if err != nil {
		return MacroEconomic{}, ErrInvalidMacroEconomic
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	row := tx.QueryRowContext(ctx, `
INSERT INTO macro_economics (
    id, name, core_proposition, candidate_assets
) VALUES ($1, $2, $3, $4::jsonb)
RETURNING id`,
		id, input.Name,
		input.CoreProposition, candidateAssets,
	)
	var persistedID string
	if err := row.Scan(&persistedID); err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	if _, err := replaceDomainLinks(ctx, tx, persistedID, input.MacroEconomicDomainIDs); err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	created, err := scanMacroEconomic(tx.QueryRowContext(ctx, `SELECT `+macroEconomicColumns+` FROM macro_economics WHERE id=$1`, persistedID))
	if err != nil {
		return MacroEconomic{}, classifyReadError(err)
	}
	if err := tx.Commit(); err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (MacroEconomic, error) {
	if !coreid.Is(id, coreid.MacroEconomic) {
		return MacroEconomic{}, ErrInvalidMacroEconomic
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+macroEconomicColumns+` FROM macro_economics WHERE id = $1`, id)
	result, err := scanMacroEconomic(row)
	if err != nil {
		return MacroEconomic{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, filter Filter) ([]MacroEconomic, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+macroEconomicColumns+`
FROM macro_economics
WHERE ($1::text IS NULL OR EXISTS (SELECT 1 FROM macro_economic_domain_links l WHERE l.macro_economic_id=macro_economics.id AND l.macro_economic_domain_id=$1))
ORDER BY name ASC, id ASC`, nullableString(filter.MacroEconomicDomainID))
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]MacroEconomic, 0)
	for rows.Next() {
		item, err := scanMacroEconomic(rows)
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

func (s *Store) Update(ctx context.Context, input UpdateInput) (MacroEconomic, error) {
	if !coreid.Is(input.ID, coreid.MacroEconomic) || validateInput(CreateInput{
		Name: input.Name, MacroEconomicDomainIDs: input.MacroEconomicDomainIDs,
		CoreProposition: input.CoreProposition,
		CandidateAssets: input.CandidateAssets,
	}) != nil {
		return MacroEconomic{}, ErrInvalidMacroEconomic
	}
	candidateAssets, err := json.Marshal(input.CandidateAssets)
	if err != nil {
		return MacroEconomic{}, ErrInvalidMacroEconomic
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	row := tx.QueryRowContext(ctx, `
UPDATE macro_economics
SET name = $2,
    core_proposition = $3, candidate_assets = $4::jsonb,
    updated_at = now()
WHERE id = $1
RETURNING id`,
		input.ID, input.Name,
		input.CoreProposition, candidateAssets,
	)
	var persistedID string
	if err := row.Scan(&persistedID); err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	if _, err := replaceDomainLinks(ctx, tx, persistedID, input.MacroEconomicDomainIDs); err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	updated, err := scanMacroEconomic(tx.QueryRowContext(ctx, `SELECT `+macroEconomicColumns+` FROM macro_economics WHERE id=$1`, persistedID))
	if err != nil {
		return MacroEconomic{}, classifyReadError(err)
	}
	if err := tx.Commit(); err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	return updated, nil
}

const macroEconomicColumns = `
id, name, short_name, (SELECT COALESCE(jsonb_agg(l.macro_economic_domain_id ORDER BY l.macro_economic_domain_id), '[]'::jsonb) FROM macro_economic_domain_links l WHERE l.macro_economic_id=macro_economics.id), core_proposition, candidate_assets, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanMacroEconomic(row rowScanner) (MacroEconomic, error) {
	var result MacroEconomic
	var candidateAssetsJSON, domainIDsJSON []byte
	if err := row.Scan(
		&result.ID, &result.Name, &result.ShortName, &domainIDsJSON,
		&result.CoreProposition,
		&candidateAssetsJSON,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return MacroEconomic{}, err
	}
	candidateAssets, err := decodeCandidateAssets(candidateAssetsJSON)
	if err != nil {
		return MacroEconomic{}, err
	}
	result.CandidateAssets = candidateAssets
	if err := json.Unmarshal(domainIDsJSON, &result.MacroEconomicDomainIDs); err != nil {
		return MacroEconomic{}, err
	}
	if err := validateStored(result); err != nil {
		return MacroEconomic{}, err
	}
	return result, nil
}

func validateInput(input CreateInput) error {
	if !validRequiredText(input.Name, 100) ||
		!validDomainIDs(input.MacroEconomicDomainIDs) ||
		strings.TrimSpace(input.CoreProposition) == "" || !validCandidateAssets(input.CandidateAssets) {
		return ErrInvalidMacroEconomic
	}
	return nil
}

func validateFilter(filter Filter) error {
	if filter.MacroEconomicDomainID != nil && !coreid.Is(*filter.MacroEconomicDomainID, coreid.MacroEconomicDomain) {
		return ErrInvalidMacroEconomic
	}
	return nil
}

func validateStored(input MacroEconomic) error {
	if input.ShortName != nil && !validRequiredText(*input.ShortName, 5) {
		return ErrInvalidMacroEconomic
	}
	if !coreid.Is(input.ID, coreid.MacroEconomic) || validateInput(CreateInput{
		Name: input.Name, MacroEconomicDomainIDs: input.MacroEconomicDomainIDs,
		CoreProposition: input.CoreProposition,
		CandidateAssets: input.CandidateAssets,
	}) != nil || input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidMacroEconomic
	}
	return nil
}

func decodeCandidateAssets(payload []byte) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	var candidateAssets []string
	if err := decoder.Decode(&candidateAssets); err != nil {
		return nil, ErrInvalidMacroEconomic
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, ErrInvalidMacroEconomic
	}
	if !validCandidateAssets(candidateAssets) {
		return nil, ErrInvalidMacroEconomic
	}
	return candidateAssets, nil
}

func validCandidateAssets(candidateAssets []string) bool {
	if len(candidateAssets) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(candidateAssets))
	for _, asset := range candidateAssets {
		if !validRequiredText(asset, 100) || asset != strings.TrimSpace(asset) {
			return false
		}
		if _, duplicate := seen[asset]; duplicate {
			return false
		}
		seen[asset] = struct{}{}
	}
	return true
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
		return ErrInvalidMacroEconomic
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
	if errors.Is(err, ErrInvalidMacroEconomic) {
		return fmt.Errorf("%w: invalid persisted MacroEconomic", ErrPersistence)
	}
	return ErrPersistence
}

func validDomainIDs(ids []string) bool {
	if len(ids) == 0 {
		return false
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !coreid.Is(id, coreid.MacroEconomicDomain) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
