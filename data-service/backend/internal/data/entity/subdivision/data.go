// Package subdivision persists independent administrative-area facts.
package subdivision

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type SubdivisionType string

const (
	SubdivisionTypeProvince  SubdivisionType = "PROVINCE"
	SubdivisionTypeState     SubdivisionType = "STATE"
	SubdivisionTypeSAR       SubdivisionType = "SAR"
	SubdivisionTypeTerritory SubdivisionType = "TERRITORY"
)

var (
	ErrInvalidSubdivision = errors.New("invalid Subdivision")
	ErrConflict           = errors.New("Subdivision conflict")
	ErrCountryNotFound    = errors.New("Subdivision Country not found")
	ErrNotFound           = errors.New("Subdivision not found")
	ErrPersistence        = errors.New("Subdivision persistence failed")
)

type Subdivision struct {
	ID                   string
	Code                 string
	Name                 string
	NameEn               string
	CountryID            string
	SubdivisionType      SubdivisionType
	StrategicPositioning *string
	KeyResources         *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Subdivision database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input Subdivision) (Subdivision, error) {
	if err := validateSubdivision(input); err != nil {
		return Subdivision{}, err
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO subdivisions (
    id, code, name, name_en, country_id, subdivision_type,
    strategic_positioning, key_resources
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, code, name, name_en, country_id, subdivision_type::text,
          strategic_positioning, key_resources, created_at, updated_at`,
		input.ID, input.Code, input.Name, input.NameEn, input.CountryID,
		string(input.SubdivisionType), input.StrategicPositioning, input.KeyResources,
	)
	created, err := scanSubdivision(row)
	if err != nil {
		return Subdivision{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (Subdivision, error) {
	if !coreid.Is(id, coreid.Subdivision) {
		return Subdivision{}, ErrInvalidSubdivision
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id, code, name, name_en, country_id, subdivision_type::text,
       strategic_positioning, key_resources, created_at, updated_at
FROM subdivisions
WHERE id = $1`, id)
	result, err := scanSubdivision(row)
	if err != nil {
		return Subdivision{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) ListByCountry(ctx context.Context, countryID string) ([]Subdivision, error) {
	if !coreid.Is(countryID, coreid.Country) {
		return nil, ErrInvalidSubdivision
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, code, name, name_en, country_id, subdivision_type::text,
       strategic_positioning, key_resources, created_at, updated_at
FROM subdivisions
WHERE country_id = $1
ORDER BY code ASC, id ASC`, countryID)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()

	result := make([]Subdivision, 0)
	for rows.Next() {
		item, err := scanSubdivision(rows)
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

type rowScanner interface{ Scan(...any) error }

func scanSubdivision(row rowScanner) (Subdivision, error) {
	var result Subdivision
	var subdivisionType string
	var strategicPositioning, keyResources sql.NullString
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &result.NameEn, &result.CountryID,
		&subdivisionType, &strategicPositioning, &keyResources,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return Subdivision{}, err
	}
	result.SubdivisionType = SubdivisionType(subdivisionType)
	if strategicPositioning.Valid {
		result.StrategicPositioning = &strategicPositioning.String
	}
	if keyResources.Valid {
		result.KeyResources = &keyResources.String
	}
	if err := validateSubdivision(result); err != nil {
		return Subdivision{}, err
	}
	if result.CreatedAt.IsZero() || result.UpdatedAt.IsZero() || result.UpdatedAt.Before(result.CreatedAt) {
		return Subdivision{}, ErrInvalidSubdivision
	}
	return result, nil
}

func validateSubdivision(input Subdivision) error {
	if !coreid.Is(input.ID, coreid.Subdivision) {
		return ErrInvalidSubdivision
	}
	if input.Code == "" || len(input.Code) > 10 || !isLocalCode(input.Code) {
		return ErrInvalidSubdivision
	}
	if !coreid.Is(input.CountryID, coreid.Country) {
		return ErrInvalidSubdivision
	}
	if err := validateNamesAndOptional(input.Name, input.NameEn, input.StrategicPositioning, input.KeyResources); err != nil {
		return err
	}
	switch input.SubdivisionType {
	case SubdivisionTypeProvince, SubdivisionTypeState, SubdivisionTypeSAR, SubdivisionTypeTerritory:
		return nil
	default:
		return ErrInvalidSubdivision
	}
}

func validateNamesAndOptional(name, nameEn string, strategicPositioning, keyResources *string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 100 {
		return ErrInvalidSubdivision
	}
	if strings.TrimSpace(nameEn) == "" || utf8.RuneCountInString(nameEn) > 100 {
		return ErrInvalidSubdivision
	}
	for _, optional := range []*string{strategicPositioning, keyResources} {
		if optional != nil && strings.TrimSpace(*optional) == "" {
			return ErrInvalidSubdivision
		}
	}
	return nil
}

func isLocalCode(code string) bool {
	for _, character := range code {
		if character < 'A' || character > 'Z' {
			if character < '0' || character > '9' {
				return false
			}
		}
	}
	return true
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ErrInvalidSubdivision) {
		return ErrPersistence
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return ErrPersistence
	}
	switch postgresError.Code {
	case "23505":
		return ErrConflict
	case "23503":
		return ErrCountryNotFound
	case "22001", "22P02", "23502", "23514":
		return ErrInvalidSubdivision
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
	return ErrPersistence
}
