package source

import (
	"context"
	"database/sql"
	"errors"

	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
)

func (s *Store) InTransaction(ctx context.Context, run func(sourcebiz.Transaction) error) error {
	if run == nil {
		return errors.New("Source transaction callback is required")
	}
	// The advisory lock serializes every Source mutation. READ COMMITTED makes the
	// post-lock List observe the transaction that released the lock immediately
	// before this one, so concurrent capacity checks cannot use a stale snapshot.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
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

var _ sourcebiz.Transaction = (*transaction)(nil)
