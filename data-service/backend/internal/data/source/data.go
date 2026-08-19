package source

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Source database is required")
	}
	return &Store{db: db}, nil
}

const sourceColumns = `
id, code, name, ownership_type, channel_type, adapter_key, enabled, endpoint, app_key,
config, priority, timeout_seconds, max_results, default_source_level, created_at, updated_at`

func (s *Store) List(ctx context.Context, activeOnly bool) ([]sourcebiz.Source, error) {
	return list(ctx, s.db, activeOnly)
}

func (s *Store) InTransaction(ctx context.Context, run func(sourcebiz.Transaction) error) error {
	if run == nil {
		return errors.New("Source transaction callback is required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return classify(err)
	}
	defer func() { _ = tx.Rollback() }()
	adapter := &transaction{tx: tx}
	if err := run(adapter); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return classify(err)
	}
	return nil
}

type transaction struct{ tx *sql.Tx }

func (t *transaction) Lock(ctx context.Context) error {
	_, err := t.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('tidewise_source_mutation'))`)
	return classify(err)
}

func (t *transaction) List(ctx context.Context) ([]sourcebiz.Source, error) {
	return list(ctx, t.tx, false)
}

func (t *transaction) Insert(ctx context.Context, value sourcebiz.Source) (sourcebiz.Source, error) {
	var row *sql.Row
	if value.CreatedAt.IsZero() {
		row = t.tx.QueryRowContext(ctx, `
INSERT INTO sources (
    id, code, name, ownership_type, channel_type, adapter_key, enabled, endpoint, app_key,
    config, priority, timeout_seconds, max_results, default_source_level
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
RETURNING `+sourceColumns,
			value.ID, value.Code, value.Name, value.OwnershipType, value.ChannelType, value.AdapterKey,
			value.Enabled, value.Endpoint, value.AppKey, string(value.Config), value.Priority,
			value.TimeoutSeconds, value.MaxResults, value.DefaultSourceLevel)
	} else {
		row = t.tx.QueryRowContext(ctx, `
INSERT INTO sources (
    id, code, name, ownership_type, channel_type, adapter_key, enabled, endpoint, app_key,
    config, priority, timeout_seconds, max_results, default_source_level, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
RETURNING `+sourceColumns,
			value.ID, value.Code, value.Name, value.OwnershipType, value.ChannelType, value.AdapterKey,
			value.Enabled, value.Endpoint, value.AppKey, string(value.Config), value.Priority,
			value.TimeoutSeconds, value.MaxResults, value.DefaultSourceLevel, value.CreatedAt, value.UpdatedAt)
	}
	return scan(row)
}

func (t *transaction) Update(ctx context.Context, value sourcebiz.Source) (sourcebiz.Source, error) {
	row := t.tx.QueryRowContext(ctx, `
UPDATE sources
SET name=$2, adapter_key=$3, enabled=$4, endpoint=$5, app_key=$6, config=$7,
    priority=$8, timeout_seconds=$9, max_results=$10, default_source_level=$11, updated_at=now()
WHERE id=$1
RETURNING `+sourceColumns,
		value.ID, value.Name, value.AdapterKey, value.Enabled, value.Endpoint, value.AppKey, string(value.Config),
		value.Priority, value.TimeoutSeconds, value.MaxResults, value.DefaultSourceLevel)
	return scan(row)
}

func (t *transaction) Delete(ctx context.Context, id string) error {
	result, err := t.tx.ExecContext(ctx, `DELETE FROM sources WHERE id=$1`, id)
	if err != nil {
		return classify(err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return classify(err)
	}
	if count == 0 {
		return sourcebiz.ErrNotFound
	}
	return nil
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func list(ctx context.Context, db queryer, activeOnly bool) ([]sourcebiz.Source, error) {
	rows, err := db.QueryContext(ctx, `
SELECT `+sourceColumns+`
FROM sources
WHERE NOT $1 OR enabled
ORDER BY channel_type, priority, code, id`, activeOnly)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	result := make([]sourcebiz.Source, 0)
	for rows.Next() {
		value, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, classify(err)
	}
	return result, nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (sourcebiz.Source, error) {
	var value sourcebiz.Source
	var config []byte
	err := row.Scan(
		&value.ID, &value.Code, &value.Name, &value.OwnershipType, &value.ChannelType,
		&value.AdapterKey, &value.Enabled, &value.Endpoint, &value.AppKey, &config,
		&value.Priority, &value.TimeoutSeconds, &value.MaxResults, &value.DefaultSourceLevel,
		&value.CreatedAt, &value.UpdatedAt,
	)
	if err != nil {
		return sourcebiz.Source{}, classify(err)
	}
	if !json.Valid(config) {
		return sourcebiz.Source{}, fmt.Errorf("%w: stored Source config is invalid JSON", sourcebiz.ErrPersistence)
	}
	value.Config = append(json.RawMessage(nil), config...)
	return value, nil
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return sourcebiz.ErrNotFound
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) && postgres.Code == "23505" {
		return sourcebiz.ErrConflict
	}
	return fmt.Errorf("%w: Source database operation", sourcebiz.ErrPersistence)
}

var _ sourcebiz.Store = (*Store)(nil)
var _ sourcebiz.Transaction = (*transaction)(nil)
