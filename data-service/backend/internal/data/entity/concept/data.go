package concept

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	conceptbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/concept"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Concept database is required")
	}
	return &Store{db: db}, nil
}

const conceptColumns = `
c.id, c.name, array_to_json(c.aliases), c.concept_type, c.definition,
c.review_status, c.created_at, c.updated_at`

func (s *Store) Create(ctx context.Context, input conceptbiz.Concept) (conceptbiz.Concept, error) {
	exists, err := s.objectIdentityExists(ctx, input.ID)
	if err != nil {
		return conceptbiz.Concept{}, err
	}
	if exists {
		return conceptbiz.Concept{}, conceptbiz.ErrConflict
	}
	row := s.db.QueryRowContext(ctx, `
WITH inserted AS (
    INSERT INTO concept (id, name, aliases, concept_type, definition, review_status)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING *
)
SELECT id, name, array_to_json(aliases), concept_type, definition,
       review_status, created_at, updated_at
FROM inserted`, input.ID, input.Name, input.Aliases, input.ConceptType, input.Definition, input.ReviewStatus)
	return scanConcept(row, classifyWriteError)
}

func (s *Store) Get(ctx context.Context, id conceptbiz.ID) (conceptbiz.Concept, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+conceptColumns+` FROM concept c WHERE c.id = $1`, id)
	return scanConcept(row, classifyReadError)
}

func (s *Store) List(ctx context.Context, query conceptbiz.ListQuery) (conceptbiz.ListResult, error) {
	var afterID any
	if query.After != nil {
		afterID = query.After.ID
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+conceptColumns+`
FROM concept c
WHERE $1::text IS NULL OR (c.name, c.id) > (
    SELECT anchor.name, anchor.id FROM concept anchor WHERE anchor.id = $1
)
ORDER BY c.name, c.id
LIMIT $2`, afterID, query.PageSize+1)
	if err != nil {
		return conceptbiz.ListResult{}, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]conceptbiz.Concept, 0, query.PageSize+1)
	for rows.Next() {
		item, err := scanConcept(rows, classifyReadError)
		if err != nil {
			return conceptbiz.ListResult{}, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return conceptbiz.ListResult{}, classifyReadError(err)
	}
	hasMore := len(result) > query.PageSize
	if hasMore {
		result = result[:query.PageSize]
	}
	return conceptbiz.ListResult{Items: result, HasMore: hasMore}, nil
}

func (s *Store) Update(ctx context.Context, id conceptbiz.ID, input conceptbiz.Update) (conceptbiz.Concept, error) {
	_, err := s.db.ExecContext(ctx, `
UPDATE concept
SET name = $2,
    aliases = $3,
    concept_type = $4,
    definition = $5,
    review_status = $6,
    updated_at = now()
WHERE id = $1
  AND ROW(name, aliases, concept_type, definition, review_status)
      IS DISTINCT FROM ROW($2::text, $3::text[], $4::varchar, $5::text, $6::varchar)`,
		id, input.Name, input.Aliases, input.ConceptType, input.Definition, input.ReviewStatus)
	if err != nil {
		return conceptbiz.Concept{}, classifyWriteError(err)
	}
	return s.Get(ctx, id)
}

type rowScanner interface{ Scan(...any) error }

func scanConcept(row rowScanner, classify func(error) error) (conceptbiz.Concept, error) {
	var result conceptbiz.Concept
	var aliasesJSON []byte
	if err := row.Scan(
		&result.ID, &result.Name, &aliasesJSON, &result.ConceptType, &result.Definition,
		&result.ReviewStatus, &result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return conceptbiz.Concept{}, classify(err)
	}
	if err := json.Unmarshal(aliasesJSON, &result.Aliases); err != nil {
		return conceptbiz.Concept{}, conceptbiz.ErrPersistence
	}
	if result.Aliases == nil || result.CreatedAt.IsZero() || result.UpdatedAt.IsZero() || result.UpdatedAt.Before(result.CreatedAt) {
		return conceptbiz.Concept{}, conceptbiz.ErrPersistence
	}
	if err := conceptbiz.ValidatePersisted(result); err != nil {
		return conceptbiz.Concept{}, conceptbiz.ErrPersistence
	}
	return result, nil
}

func (s *Store) objectIdentityExists(ctx context.Context, id conceptbiz.ID) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM entity_nodes WHERE id = $1
    UNION ALL SELECT 1 FROM industry WHERE id = $1
    UNION ALL SELECT 1 FROM concept WHERE id = $1
	UNION ALL SELECT 1 FROM chain_node WHERE id = $1
	UNION ALL SELECT 1 FROM industry_chain WHERE id = $1
)`, id).Scan(&exists)
	if err != nil {
		return false, classifyReadError(err)
	}
	return exists, nil
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return conceptbiz.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return conceptbiz.ErrPersistence
	}
	switch postgresError.Code {
	case "23505":
		return conceptbiz.ErrConflict
	case "P0001":
		return conceptbiz.ErrConflict
	case "22001", "23502", "23514":
		return &conceptbiz.ValidationError{Field: "concept", Message: "violates the persistence contract"}
	default:
		return conceptbiz.ErrPersistence
	}
}

func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return conceptbiz.ErrNotFound
	}
	return conceptbiz.ErrPersistence
}

var _ conceptbiz.Repository = (*Store)(nil)
