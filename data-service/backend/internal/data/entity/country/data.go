package country

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	countrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/country"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Country database is required")
	}
	return &Store{db: db}, nil
}

const countryColumns = `
c.id, btrim(c.code), c.name, c.name_en, c.strategic_positioning, c.key_resources,
c.created_at, c.updated_at,
COALESCE(
    jsonb_agg(jsonb_build_object(
        'id', r.id,
        'code', r.code,
        'name', r.name,
        'name_en', r.name_en,
        'region_type', r.region_type::text
    ) ORDER BY r.code, r.id) FILTER (WHERE r.id IS NOT NULL),
    '[]'::jsonb
)`

func (s *Store) Create(ctx context.Context, input countrybiz.Country) (countrybiz.Country, error) {
	row := s.db.QueryRowContext(ctx, `
WITH inserted AS (
    INSERT INTO countries (id, code, name, name_en, strategic_positioning, key_resources)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING *
)
SELECT id, btrim(code), name, name_en, strategic_positioning, key_resources,
       created_at, updated_at, '[]'::jsonb
FROM inserted`, input.ID, input.Code, input.Name, input.NameEn, input.StrategicPositioning, input.KeyResources)
	return scanCountry(row, classifyWriteError)
}

func (s *Store) Get(ctx context.Context, id string) (countrybiz.Country, error) {
	return s.get(ctx, `c.id = $1`, id)
}

func (s *Store) List(ctx context.Context, regionID string) ([]countrybiz.Country, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT `+countryColumns+`
FROM countries c
LEFT JOIN country_region_links link ON link.country_id = c.id
LEFT JOIN regions r ON r.id = link.region_id
WHERE $1 = '' OR EXISTS (
    SELECT 1 FROM country_region_links selected
    WHERE selected.country_id = c.id AND selected.region_id = $1
)
GROUP BY c.id
ORDER BY btrim(c.code), c.id`, regionID)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]countrybiz.Country, 0)
	for rows.Next() {
		country, err := scanCountry(rows, classifyReadError)
		if err != nil {
			return nil, err
		}
		result = append(result, country)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) Update(ctx context.Context, id string, input countrybiz.Update) (countrybiz.Country, error) {
	_, err := s.db.ExecContext(ctx, `
UPDATE countries
SET name = $2,
    name_en = $3,
    strategic_positioning = $4,
    key_resources = $5,
    updated_at = now()
WHERE id = $1
  AND ROW(name, name_en, strategic_positioning, key_resources)
      IS DISTINCT FROM ROW($2::varchar, $3::varchar, $4::text, $5::text)`,
		id, input.Name, input.NameEn, input.StrategicPositioning, input.KeyResources)
	if err != nil {
		return countrybiz.Country{}, classifyWriteError(err)
	}
	return s.Get(ctx, id)
}

func (s *Store) ReplaceRegions(ctx context.Context, id string, regionIDs []string) (countrybiz.Country, error) {
	regionIDs = append([]string{}, regionIDs...)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return countrybiz.Country{}, classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT true FROM countries WHERE id = $1 FOR UPDATE`, id).Scan(&exists); err != nil {
		return countrybiz.Country{}, classifyWriteError(err)
	}
	var regionCount int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM regions WHERE id = ANY($1::text[])`, regionIDs).Scan(&regionCount); err != nil {
		return countrybiz.Country{}, classifyWriteError(err)
	}
	if regionCount != len(regionIDs) {
		return countrybiz.Country{}, &countrybiz.ReferenceError{Field: "region_ids", Message: "contains an unknown Region"}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM country_region_links WHERE country_id = $1 AND NOT (region_id = ANY($2::text[]))`, id, regionIDs); err != nil {
		return countrybiz.Country{}, classifyWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO country_region_links (country_id, region_id)
SELECT $1, region_id FROM unnest($2::text[]) AS region_id
ON CONFLICT (country_id, region_id) DO NOTHING`, id, regionIDs); err != nil {
		return countrybiz.Country{}, classifyWriteError(err)
	}
	if err := tx.Commit(); err != nil {
		return countrybiz.Country{}, classifyWriteError(err)
	}
	return s.Get(ctx, id)
}

type rowScanner interface{ Scan(...any) error }

func (s *Store) get(ctx context.Context, predicate string, value any) (countrybiz.Country, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT `+countryColumns+`
FROM countries c
LEFT JOIN country_region_links link ON link.country_id = c.id
LEFT JOIN regions r ON r.id = link.region_id
WHERE `+predicate+`
GROUP BY c.id`, value)
	return scanCountry(row, classifyReadError)
}

func scanCountry(row rowScanner, classify func(error) error) (countrybiz.Country, error) {
	var result countrybiz.Country
	var strategicPositioning, keyResources sql.NullString
	var regionsJSON []byte
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &result.NameEn,
		&strategicPositioning, &keyResources, &result.CreatedAt, &result.UpdatedAt, &regionsJSON,
	); err != nil {
		return countrybiz.Country{}, classify(err)
	}
	if strategicPositioning.Valid {
		result.StrategicPositioning = &strategicPositioning.String
	}
	if keyResources.Valid {
		result.KeyResources = &keyResources.String
	}
	var regions []struct {
		ID         string `json:"id"`
		Code       string `json:"code"`
		Name       string `json:"name"`
		NameEn     string `json:"name_en"`
		RegionType string `json:"region_type"`
	}
	if err := json.Unmarshal(regionsJSON, &regions); err != nil {
		return countrybiz.Country{}, countrybiz.ErrPersistence
	}
	result.Regions = make([]countrybiz.Region, len(regions))
	for index, region := range regions {
		result.Regions[index] = countrybiz.Region(region)
	}
	if result.CreatedAt.IsZero() || result.UpdatedAt.IsZero() || result.UpdatedAt.Before(result.CreatedAt) {
		return countrybiz.Country{}, countrybiz.ErrPersistence
	}
	return result, nil
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return countrybiz.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return countrybiz.ErrPersistence
	}
	switch postgresError.Code {
	case "23505":
		return countrybiz.ErrConflict
	case "23503":
		return &countrybiz.ReferenceError{Field: "region_ids", Message: "contains an unknown Region"}
	case "22001", "23502", "23514":
		return &countrybiz.ValidationError{Field: "country", Message: "violates the persistence contract"}
	default:
		return countrybiz.ErrPersistence
	}
}

func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return countrybiz.ErrNotFound
	}
	return fmt.Errorf("%w", countrybiz.ErrPersistence)
}

var _ countrybiz.Store = (*Store)(nil)
