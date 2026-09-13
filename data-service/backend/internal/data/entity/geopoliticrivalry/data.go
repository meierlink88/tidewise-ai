// Package geopoliticrivalry persists geopolitical storylines and their domain memberships.
package geopoliticrivalry

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
	ErrInvalidGeopoliticRivalry = errors.New("invalid GeopoliticRivalry")
	ErrConflict                 = errors.New("GeopoliticRivalry conflict")
	ErrNotFound                 = errors.New("GeopoliticRivalry not found")
	ErrPersistence              = errors.New("GeopoliticRivalry persistence failed")
)

type CreateInput struct {
	Name                string
	Category            string
	GeopoliticDomainIDs []string
	CoreProposition     string
	CoreActors          string
	MainTransmission    string
	CandidateAssets     []string
}

type UpdateInput struct {
	ID                  string
	Name                string
	Category            string
	GeopoliticDomainIDs []string
	CoreProposition     string
	CoreActors          string
	MainTransmission    string
	CandidateAssets     []string
}

type GeopoliticRivalry struct {
	ShortName           *string
	ID                  string
	Name                string
	Category            string
	GeopoliticDomainIDs []string
	CoreProposition     string
	CoreActors          string
	MainTransmission    string
	CandidateAssets     []string
	CreatedAt           time.Time
	UpdatedAt           time.Time
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
	candidateAssets, err := json.Marshal(input.CandidateAssets)
	if err != nil {
		return GeopoliticRivalry{}, ErrInvalidGeopoliticRivalry
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	row := tx.QueryRowContext(ctx, `
INSERT INTO geopolitic_rivalries (
    id, name, category,
    core_proposition, core_actors, main_transmission, candidate_assets
) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
RETURNING id`,
		id, input.Name, input.Category,
		input.CoreProposition, input.CoreActors, input.MainTransmission, candidateAssets,
	)
	var persistedID string
	if err := row.Scan(&persistedID); err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	if _, err := replaceDomainLinks(ctx, tx, persistedID, input.GeopoliticDomainIDs); err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	created, err := scanGeopoliticRivalry(tx.QueryRowContext(ctx, `SELECT `+geopoliticRivalryColumns+` FROM geopolitic_rivalries WHERE id=$1`, persistedID))
	if err != nil {
		return GeopoliticRivalry{}, classifyReadError(err)
	}
	if err := tx.Commit(); err != nil {
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
WHERE ($1::text IS NULL OR EXISTS (SELECT 1 FROM geopolitic_rivalry_domain_links l WHERE l.geopolitic_rivalry_id=geopolitic_rivalries.id AND l.geopolitic_domain_id=$1))
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
		Name: input.Name, Category: input.Category, GeopoliticDomainIDs: input.GeopoliticDomainIDs,
		CoreProposition: input.CoreProposition, CoreActors: input.CoreActors,
		MainTransmission: input.MainTransmission, CandidateAssets: input.CandidateAssets,
	}) != nil {
		return GeopoliticRivalry{}, ErrInvalidGeopoliticRivalry
	}
	candidateAssets, err := json.Marshal(input.CandidateAssets)
	if err != nil {
		return GeopoliticRivalry{}, ErrInvalidGeopoliticRivalry
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	row := tx.QueryRowContext(ctx, `
UPDATE geopolitic_rivalries
SET name = $2, category = $3,
    core_proposition = $4, core_actors = $5, main_transmission = $6,
    candidate_assets = $7::jsonb,
    updated_at = now()
WHERE id = $1
RETURNING id`,
		input.ID, input.Name, input.Category,
		input.CoreProposition, input.CoreActors, input.MainTransmission, candidateAssets,
	)
	var persistedID string
	if err := row.Scan(&persistedID); err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	if _, err := replaceDomainLinks(ctx, tx, persistedID, input.GeopoliticDomainIDs); err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	updated, err := scanGeopoliticRivalry(tx.QueryRowContext(ctx, `SELECT `+geopoliticRivalryColumns+` FROM geopolitic_rivalries WHERE id=$1`, persistedID))
	if err != nil {
		return GeopoliticRivalry{}, classifyReadError(err)
	}
	if err := tx.Commit(); err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	return updated, nil
}

const geopoliticRivalryColumns = `
id, name, short_name, category, (SELECT COALESCE(jsonb_agg(l.geopolitic_domain_id ORDER BY l.geopolitic_domain_id), '[]'::jsonb) FROM geopolitic_rivalry_domain_links l WHERE l.geopolitic_rivalry_id=geopolitic_rivalries.id), core_proposition,
core_actors, main_transmission, candidate_assets, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanGeopoliticRivalry(row rowScanner) (GeopoliticRivalry, error) {
	var result GeopoliticRivalry
	var candidateAssetsJSON, domainIDsJSON []byte
	if err := row.Scan(
		&result.ID, &result.Name, &result.ShortName, &result.Category, &domainIDsJSON,
		&result.CoreProposition, &result.CoreActors, &result.MainTransmission,
		&candidateAssetsJSON,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return GeopoliticRivalry{}, err
	}
	candidateAssets, err := decodeCandidateAssets(candidateAssetsJSON)
	if err != nil {
		return GeopoliticRivalry{}, err
	}
	result.CandidateAssets = candidateAssets
	if err := json.Unmarshal(domainIDsJSON, &result.GeopoliticDomainIDs); err != nil {
		return GeopoliticRivalry{}, err
	}
	if err := validateStored(result); err != nil {
		return GeopoliticRivalry{}, err
	}
	return result, nil
}

func validateInput(input CreateInput) error {
	if !validRequiredText(input.Name, 100) || !validRequiredText(input.Category, 100) ||
		!validDomainIDs(input.GeopoliticDomainIDs) ||
		strings.TrimSpace(input.CoreProposition) == "" || strings.TrimSpace(input.CoreActors) == "" ||
		strings.TrimSpace(input.MainTransmission) == "" || !validCandidateAssets(input.CandidateAssets) {
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
	if input.ShortName != nil && !validRequiredText(*input.ShortName, 5) {
		return ErrInvalidGeopoliticRivalry
	}
	if !coreid.Is(input.ID, coreid.GeopoliticRivalry) || validateInput(CreateInput{
		Name: input.Name, Category: input.Category, GeopoliticDomainIDs: input.GeopoliticDomainIDs,
		CoreProposition: input.CoreProposition, CoreActors: input.CoreActors,
		MainTransmission: input.MainTransmission, CandidateAssets: input.CandidateAssets,
	}) != nil || input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidGeopoliticRivalry
	}
	return nil
}

func decodeCandidateAssets(payload []byte) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	var candidateAssets []string
	if err := decoder.Decode(&candidateAssets); err != nil {
		return nil, ErrInvalidGeopoliticRivalry
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, ErrInvalidGeopoliticRivalry
	}
	if !validCandidateAssets(candidateAssets) {
		return nil, ErrInvalidGeopoliticRivalry
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

func validDomainIDs(ids []string) bool {
	if len(ids) == 0 {
		return false
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !coreid.Is(id, coreid.GeopoliticDomain) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
