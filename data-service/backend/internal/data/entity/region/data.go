package region

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
)

type RegionType string

const (
	RegionTypeContinent    RegionType = "CONTINENT"
	RegionTypeGeographic   RegionType = "GEOGRAPHIC"
	RegionTypeMultilateral RegionType = "MULTILATERAL"
	RegionTypeInvestment   RegionType = "INVESTMENT"
)

var (
	ErrInvalidRegion = errors.New("invalid region")
	ErrConflict      = errors.New("region conflict")
	ErrNotFound      = errors.New("region not found")
	ErrPersistence   = errors.New("region persistence failed")
)

type Region struct {
	ID          string
	Code        string
	Name        string
	NameEn      string
	RegionType  RegionType
	Description *string
	CreatedAt   time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("region database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, region Region) (Region, error) {
	if err := validateRegion(region); err != nil {
		return Region{}, err
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO regions (id, code, name, name_en, region_type, description)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, name, name_en, region_type::text, description, created_at
`, region.ID, region.Code, region.Name, region.NameEn, string(region.RegionType), region.Description)
	created, err := scanRegion(row)
	if err != nil {
		return Region{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) GetByID(ctx context.Context, id string) (Region, error) {
	return s.get(ctx, `
SELECT id, code, name, name_en, region_type::text, description, created_at
FROM regions
WHERE id = $1
`, id)
}

func (s *Store) GetByCode(ctx context.Context, code string) (Region, error) {
	return s.get(ctx, `
SELECT id, code, name, name_en, region_type::text, description, created_at
FROM regions
WHERE code = $1
`, code)
}

func (s *Store) List(ctx context.Context) ([]Region, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, code, name, name_en, region_type::text, description, created_at
FROM regions
ORDER BY code ASC, id ASC
`)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()

	regions := make([]Region, 0)
	for rows.Next() {
		region, err := scanRegion(rows)
		if err != nil {
			return nil, classifyReadError(err)
		}
		regions = append(regions, region)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return regions, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *Store) get(ctx context.Context, query, identity string) (Region, error) {
	region, err := scanRegion(s.db.QueryRowContext(ctx, query, identity))
	if err != nil {
		return Region{}, classifyReadError(err)
	}
	return region, nil
}

func scanRegion(row rowScanner) (Region, error) {
	var region Region
	var regionType string
	var description sql.NullString
	if err := row.Scan(
		&region.ID,
		&region.Code,
		&region.Name,
		&region.NameEn,
		&regionType,
		&description,
		&region.CreatedAt,
	); err != nil {
		return Region{}, err
	}
	region.RegionType = RegionType(regionType)
	if description.Valid {
		region.Description = &description.String
	}
	if err := validateRegion(region); err != nil {
		return Region{}, fmt.Errorf("%w: persisted value", ErrInvalidRegion)
	}
	if region.CreatedAt.IsZero() {
		return Region{}, fmt.Errorf("%w: persisted creation time", ErrInvalidRegion)
	}
	return region, nil
}

func validateRegion(region Region) error {
	if region.Code == "" || len(region.Code) > 20 || !isStableCode(region.Code) {
		return ErrInvalidRegion
	}
	if region.ID != "REG_"+region.Code || len(region.ID) > 32 {
		return ErrInvalidRegion
	}
	if strings.TrimSpace(region.Name) == "" || utf8.RuneCountInString(region.Name) > 50 {
		return ErrInvalidRegion
	}
	if strings.TrimSpace(region.NameEn) == "" || utf8.RuneCountInString(region.NameEn) > 100 {
		return ErrInvalidRegion
	}
	switch region.RegionType {
	case RegionTypeContinent, RegionTypeGeographic, RegionTypeMultilateral, RegionTypeInvestment:
		return nil
	default:
		return ErrInvalidRegion
	}
}

func isStableCode(code string) bool {
	for index, character := range code {
		if character >= 'A' && character <= 'Z' {
			continue
		}
		if index > 0 && (character == '_' || character >= '0' && character <= '9') {
			continue
		}
		return false
	}
	return true
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return ErrPersistence
	}
	switch postgresError.Code {
	case "23505":
		return ErrConflict
	case "22001", "22P02", "23502", "23514":
		return ErrInvalidRegion
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
	if errors.Is(err, ErrInvalidRegion) {
		return err
	}
	return ErrPersistence
}
