package chainnode

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	chainnodebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/chainnode"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("ChainNode database is required")
	}
	return &Store{db: db}, nil
}

const chainNodeColumns = `c.id, c.name, array_to_json(c.aliases), c.definition,
c.review_status, c.created_at, c.updated_at`

func (s *Store) Create(ctx context.Context, input chainnodebiz.ChainNode) (chainnodebiz.ChainNode, error) {
	exists, err := s.objectIdentityExists(ctx, input.ID)
	if err != nil {
		return chainnodebiz.ChainNode{}, err
	}
	if exists {
		return chainnodebiz.ChainNode{}, chainnodebiz.ErrConflict
	}
	row := s.db.QueryRowContext(ctx, `
WITH inserted AS (
    INSERT INTO chain_node (id, name, aliases, definition, review_status)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT id, name, array_to_json(aliases), definition, review_status, created_at, updated_at
FROM inserted`, input.ID, input.Name, input.Aliases, input.Definition, input.ReviewStatus)
	return scanChainNode(row, classifyWriteError)
}

func (s *Store) Get(ctx context.Context, id chainnodebiz.ID) (chainnodebiz.ChainNode, error) {
	return scanChainNode(s.db.QueryRowContext(ctx, `SELECT `+chainNodeColumns+` FROM chain_node c WHERE c.id = $1`, id), classifyReadError)
}

func (s *Store) List(ctx context.Context, query chainnodebiz.ListQuery) (chainnodebiz.ListResult, error) {
	var afterID any
	if query.After != nil {
		afterID = query.After.ID
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+chainNodeColumns+`
FROM chain_node c
WHERE $1::text IS NULL OR c.id > $1
ORDER BY c.id
LIMIT $2`, afterID, query.PageSize+1)
	if err != nil {
		return chainnodebiz.ListResult{}, classifyReadError(err)
	}
	defer rows.Close()
	items := make([]chainnodebiz.ChainNode, 0, query.PageSize+1)
	for rows.Next() {
		item, err := scanChainNode(rows, classifyReadError)
		if err != nil {
			return chainnodebiz.ListResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return chainnodebiz.ListResult{}, classifyReadError(err)
	}
	hasMore := len(items) > query.PageSize
	if hasMore {
		items = items[:query.PageSize]
	}
	return chainnodebiz.ListResult{Items: items, HasMore: hasMore}, nil
}

func (s *Store) Update(ctx context.Context, id chainnodebiz.ID, input chainnodebiz.Update) (chainnodebiz.ChainNode, error) {
	_, err := s.db.ExecContext(ctx, `
UPDATE chain_node
SET name = $2, aliases = $3, definition = $4, review_status = $5, updated_at = now()
WHERE id = $1
  AND ROW(name, aliases, definition, review_status)
      IS DISTINCT FROM ROW($2::text, $3::text[], $4::text, $5::varchar)`,
		id, input.Name, input.Aliases, input.Definition, input.ReviewStatus)
	if err != nil {
		return chainnodebiz.ChainNode{}, classifyWriteError(err)
	}
	return s.Get(ctx, id)
}

type rowScanner interface{ Scan(...any) error }

func scanChainNode(row rowScanner, classify func(error) error) (chainnodebiz.ChainNode, error) {
	var result chainnodebiz.ChainNode
	var aliasesJSON []byte
	if err := row.Scan(&result.ID, &result.Name, &aliasesJSON, &result.Definition, &result.ReviewStatus, &result.CreatedAt, &result.UpdatedAt); err != nil {
		return chainnodebiz.ChainNode{}, classify(err)
	}
	if err := json.Unmarshal(aliasesJSON, &result.Aliases); err != nil {
		return chainnodebiz.ChainNode{}, chainnodebiz.ErrPersistence
	}
	if result.Aliases == nil || result.CreatedAt.IsZero() || result.UpdatedAt.IsZero() || result.UpdatedAt.Before(result.CreatedAt) {
		return chainnodebiz.ChainNode{}, chainnodebiz.ErrPersistence
	}
	if err := chainnodebiz.ValidatePersisted(result); err != nil {
		return chainnodebiz.ChainNode{}, chainnodebiz.ErrPersistence
	}
	return result, nil
}

func (s *Store) objectIdentityExists(ctx context.Context, id chainnodebiz.ID) (bool, error) {
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
		return chainnodebiz.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return chainnodebiz.ErrPersistence
	}
	switch postgresError.Code {
	case "23505", "P0001":
		return chainnodebiz.ErrConflict
	case "22001", "23502", "23514":
		return &chainnodebiz.ValidationError{Field: "chain_node", Message: "violates the persistence contract"}
	default:
		return chainnodebiz.ErrPersistence
	}
}

func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return chainnodebiz.ErrNotFound
	}
	return chainnodebiz.ErrPersistence
}

var _ chainnodebiz.Repository = (*Store)(nil)
