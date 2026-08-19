package source

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgconn"
	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
)

type Store struct{ db *sql.DB }

const maxImportBytes = 2_000_000

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Source database is required")
	}
	return &Store{db: db}, nil
}

func DecodeImport(reader io.Reader) ([]sourcebiz.Source, error) {
	if reader == nil {
		return nil, fmt.Errorf("Source import reader is required")
	}
	decoder := json.NewDecoder(io.LimitReader(reader, maxImportBytes+1))
	decoder.DisallowUnknownFields()
	var document struct {
		Sources []sourcebiz.Source `json:"sources"`
	}
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode Source import: %w", err)
	}
	if document.Sources == nil {
		return nil, fmt.Errorf("decode Source import: sources is required")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode Source import: trailing JSON value")
		}
		return nil, fmt.Errorf("decode Source import: %w", err)
	}
	return document.Sources, nil
}

const sourceColumns = `
id, code, name, ownership_type, channel_type, adapter_key, enabled, endpoint, app_key,
config, priority, timeout_seconds, max_results, default_source_level, created_at, updated_at`

func (s *Store) List(ctx context.Context, activeOnly bool) ([]sourcebiz.Source, error) {
	return list(ctx, s.db, activeOnly)
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
