package industry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	industrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/industry"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Industry database is required")
	}
	return &Store{db: db}, nil
}

const industryColumns = `
i.id, i.name, array_to_json(i.aliases), i.classification_system, i.industry_code,
i.parent_industry_id, array_to_json(i.hierarchy_path_codes), i.definition,
i.review_status, i.created_at, i.updated_at`

func (s *Store) Create(ctx context.Context, input industrybiz.Industry) (industrybiz.Industry, error) {
	exists, err := s.objectIdentityExists(ctx, input.ID)
	if err != nil {
		return industrybiz.Industry{}, err
	}
	if exists {
		return industrybiz.Industry{}, industrybiz.ErrConflict
	}
	if err := s.requireParent(ctx, input.ParentIndustryID); err != nil {
		return industrybiz.Industry{}, err
	}
	row := s.db.QueryRowContext(ctx, `
WITH inserted AS (
    INSERT INTO industry (
        id, name, aliases, classification_system, industry_code,
        parent_industry_id, hierarchy_path_codes, definition, review_status
    )
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING *
)
SELECT id, name, array_to_json(aliases), classification_system, industry_code,
       parent_industry_id, array_to_json(hierarchy_path_codes), definition,
       review_status, created_at, updated_at
FROM inserted`, input.ID, input.Name, input.Aliases, input.ClassificationSystem, input.IndustryCode,
		input.ParentIndustryID, input.HierarchyPathCodes, input.Definition, input.ReviewStatus)
	return scanIndustry(row, classifyWriteError)
}

func (s *Store) Get(ctx context.Context, id industrybiz.ID) (industrybiz.Industry, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+industryColumns+` FROM industry i WHERE i.id = $1`, id)
	return scanIndustry(row, classifyReadError)
}

func (s *Store) List(ctx context.Context) ([]industrybiz.Industry, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT `+industryColumns+`
FROM industry i
ORDER BY i.classification_system, i.hierarchy_path_codes, i.industry_code, i.id`)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]industrybiz.Industry, 0)
	for rows.Next() {
		item, err := scanIndustry(rows, classifyReadError)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) Update(ctx context.Context, id industrybiz.ID, input industrybiz.Update) (industrybiz.Industry, error) {
	if err := s.requireParent(ctx, input.ParentIndustryID); err != nil {
		return industrybiz.Industry{}, err
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE industry
SET name = $2,
    aliases = $3,
    parent_industry_id = $4,
    hierarchy_path_codes = $5,
    definition = $6,
    review_status = $7,
    updated_at = now()
WHERE id = $1
  AND ROW(name, aliases, parent_industry_id, hierarchy_path_codes, definition, review_status)
      IS DISTINCT FROM ROW($2::text, $3::text[], $4::text, $5::text[], $6::text, $7::varchar)`,
		id, input.Name, input.Aliases, input.ParentIndustryID, input.HierarchyPathCodes, input.Definition, input.ReviewStatus)
	if err != nil {
		return industrybiz.Industry{}, classifyWriteError(err)
	}
	return s.Get(ctx, id)
}

type rowScanner interface{ Scan(...any) error }

func scanIndustry(row rowScanner, classify func(error) error) (industrybiz.Industry, error) {
	var result industrybiz.Industry
	var aliasesJSON, pathJSON []byte
	var parent sql.NullString
	if err := row.Scan(
		&result.ID, &result.Name, &aliasesJSON, &result.ClassificationSystem, &result.IndustryCode,
		&parent, &pathJSON, &result.Definition, &result.ReviewStatus, &result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return industrybiz.Industry{}, classify(err)
	}
	if parent.Valid {
		parentID := industrybiz.ID(parent.String)
		result.ParentIndustryID = &parentID
	}
	if err := json.Unmarshal(aliasesJSON, &result.Aliases); err != nil {
		return industrybiz.Industry{}, industrybiz.ErrPersistence
	}
	if err := json.Unmarshal(pathJSON, &result.HierarchyPathCodes); err != nil {
		return industrybiz.Industry{}, industrybiz.ErrPersistence
	}
	if result.Aliases == nil || result.HierarchyPathCodes == nil || result.CreatedAt.IsZero() || result.UpdatedAt.IsZero() || result.UpdatedAt.Before(result.CreatedAt) {
		return industrybiz.Industry{}, industrybiz.ErrPersistence
	}
	if err := industrybiz.ValidatePersisted(result); err != nil {
		return industrybiz.Industry{}, industrybiz.ErrPersistence
	}
	return result, nil
}

func (s *Store) objectIdentityExists(ctx context.Context, id industrybiz.ID) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM entity_nodes WHERE id = $1
    UNION ALL SELECT 1 FROM industry WHERE id = $1
    UNION ALL SELECT 1 FROM concept WHERE id = $1
)`, id).Scan(&exists)
	if err != nil {
		return false, classifyReadError(err)
	}
	return exists, nil
}

func (s *Store) requireParent(ctx context.Context, id *industrybiz.ID) error {
	if id == nil {
		return nil
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM industry WHERE id = $1)`, *id).Scan(&exists); err != nil {
		return classifyReadError(err)
	}
	if !exists {
		return &industrybiz.ReferenceError{Field: "parent_industry_id", Message: "identifies an unknown Industry"}
	}
	return nil
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return industrybiz.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return industrybiz.ErrPersistence
	}
	switch postgresError.Code {
	case "23505":
		return industrybiz.ErrConflict
	case "23503":
		return &industrybiz.ReferenceError{Field: "parent_industry_id", Message: "identifies an unknown Industry"}
	case "P0001":
		return &industrybiz.ValidationError{Field: "industry", Message: "violates the persistence contract"}
	case "22001", "23502", "23514":
		return &industrybiz.ValidationError{Field: "industry", Message: "violates the persistence contract"}
	default:
		return industrybiz.ErrPersistence
	}
}

func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return industrybiz.ErrNotFound
	}
	return industrybiz.ErrPersistence
}

var _ industrybiz.Repository = (*Store)(nil)
