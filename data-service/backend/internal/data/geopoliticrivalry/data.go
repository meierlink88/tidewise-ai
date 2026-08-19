// Package geopoliticrivalry persists independent geopolitical-rivalry blueprints.
package geopoliticrivalry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type RivalryType string

const (
	RivalryTypeGeopolitical RivalryType = "GEOPOLITICAL"
	RivalryTypeMilitaryWar  RivalryType = "MILITARY_WAR"
)

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusDormant  Status = "DORMANT"
	StatusResolved Status = "RESOLVED"
)

var (
	ErrInvalidGeopoliticRivalry = errors.New("invalid GeopoliticRivalry")
	ErrNotFound                 = errors.New("GeopoliticRivalry not found")
	ErrPersistence              = errors.New("GeopoliticRivalry persistence failed")
)

type CreateInput struct {
	Name              string
	NameEn            string
	RivalryType       RivalryType
	Description       string
	CoreActors        string
	PeripheralActors  *string
	InfluencedRegions []string
	Status            Status
}

type UpdateInput struct {
	ID                string
	Name              string
	NameEn            string
	RivalryType       RivalryType
	Description       string
	CoreActors        string
	PeripheralActors  *string
	InfluencedRegions []string
	Status            Status
}

type GeopoliticRivalry struct {
	ID                string
	Name              string
	NameEn            string
	RivalryType       RivalryType
	Description       string
	CoreActors        string
	PeripheralActors  *string
	InfluencedRegions []string
	Status            Status
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Filter struct {
	RivalryType *RivalryType
	Status      *Status
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("GeopoliticRivalry database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (GeopoliticRivalry, error) {
	if input.Status == "" {
		input.Status = StatusActive
	}
	if err := validateCreate(input); err != nil {
		return GeopoliticRivalry{}, err
	}
	id, err := coreid.New(coreid.GeopoliticRivalry)
	if err != nil {
		return GeopoliticRivalry{}, ErrPersistence
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO geopolitic_rivalries (
    id, name, name_en, rivalry_type, description, core_actors,
    peripheral_actors, influenced_regions, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::text[], $9)
RETURNING `+geopoliticRivalryColumns,
		id, input.Name, input.NameEn, string(input.RivalryType), input.Description,
		input.CoreActors, input.PeripheralActors, input.InfluencedRegions, string(input.Status),
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
WHERE ($1::geopolitic_rivalry_type IS NULL OR rivalry_type = $1::geopolitic_rivalry_type)
  AND ($2::geopolitic_rivalry_status IS NULL OR status = $2::geopolitic_rivalry_status)
ORDER BY name_en ASC, name ASC, id ASC`, nullableRivalryType(filter.RivalryType), nullableStatus(filter.Status))
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
	if err := validateUpdate(input); err != nil {
		return GeopoliticRivalry{}, err
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE geopolitic_rivalries
SET name = $2,
    name_en = $3,
    rivalry_type = $4,
    description = $5,
    core_actors = $6,
    peripheral_actors = $7,
    influenced_regions = $8::text[],
    status = $9,
    updated_at = now()
WHERE id = $1
RETURNING `+geopoliticRivalryColumns,
		input.ID, input.Name, input.NameEn, string(input.RivalryType), input.Description,
		input.CoreActors, input.PeripheralActors, input.InfluencedRegions, string(input.Status),
	)
	updated, err := scanGeopoliticRivalry(row)
	if err != nil {
		return GeopoliticRivalry{}, classifyWriteError(err)
	}
	return updated, nil
}

const geopoliticRivalryColumns = `
id, name, name_en, rivalry_type::text, description, core_actors,
peripheral_actors, to_json(influenced_regions) AS influenced_regions,
status::text, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanGeopoliticRivalry(row rowScanner) (GeopoliticRivalry, error) {
	var result GeopoliticRivalry
	var peripheralActors sql.NullString
	var rivalryType, status string
	var influencedRegionsJSON []byte
	if err := row.Scan(
		&result.ID, &result.Name, &result.NameEn, &rivalryType,
		&result.Description, &result.CoreActors, &peripheralActors,
		&influencedRegionsJSON, &status, &result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return GeopoliticRivalry{}, err
	}
	result.RivalryType = RivalryType(rivalryType)
	result.PeripheralActors = nullableString(peripheralActors)
	if influencedRegionsJSON != nil {
		if err := json.Unmarshal(influencedRegionsJSON, &result.InfluencedRegions); err != nil {
			return GeopoliticRivalry{}, err
		}
	}
	result.Status = Status(status)
	if err := validateStored(result); err != nil {
		return GeopoliticRivalry{}, err
	}
	return result, nil
}

func validateCreate(input CreateInput) error {
	if !validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) ||
		strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.CoreActors) == "" ||
		!validRivalryType(input.RivalryType) || !validStatus(input.Status) {
		return ErrInvalidGeopoliticRivalry
	}
	return nil
}

func validateUpdate(input UpdateInput) error {
	if !coreid.Is(input.ID, coreid.GeopoliticRivalry) ||
		!validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) ||
		strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.CoreActors) == "" ||
		!validRivalryType(input.RivalryType) || !validStatus(input.Status) {
		return ErrInvalidGeopoliticRivalry
	}
	return nil
}

func validateFilter(filter Filter) error {
	if filter.RivalryType != nil && !validRivalryType(*filter.RivalryType) {
		return ErrInvalidGeopoliticRivalry
	}
	if filter.Status != nil && !validStatus(*filter.Status) {
		return ErrInvalidGeopoliticRivalry
	}
	return nil
}

func validateStored(input GeopoliticRivalry) error {
	if !coreid.Is(input.ID, coreid.GeopoliticRivalry) ||
		!validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) ||
		strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.CoreActors) == "" ||
		!validRivalryType(input.RivalryType) || !validStatus(input.Status) ||
		input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidGeopoliticRivalry
	}
	return nil
}

func validRequiredText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes
}

func validRivalryType(value RivalryType) bool {
	switch value {
	case RivalryTypeGeopolitical, RivalryTypeMilitaryWar:
		return true
	default:
		return false
	}
}

func validStatus(value Status) bool {
	switch value {
	case StatusActive, StatusDormant, StatusResolved:
		return true
	default:
		return false
	}
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullableRivalryType(value *RivalryType) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func nullableStatus(value *Status) any {
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
