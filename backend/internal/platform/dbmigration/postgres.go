package dbmigration

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pressly/goose/v3"
)

type GooseExecutor struct {
	db        *sql.DB
	directory string
}

func NewGooseExecutor(db *sql.DB, directory string) GooseExecutor {
	return GooseExecutor{
		db:        db,
		directory: ResolveDirectory(directory),
	}
}

func (e GooseExecutor) CurrentVersion(ctx context.Context) (string, error) {
	if err := goose.SetDialect("postgres"); err != nil {
		return "", err
	}

	version, err := goose.EnsureDBVersionContext(ctx, e.db)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(version, 10), nil
}

func (e GooseExecutor) Pending(ctx context.Context) ([]Migration, error) {
	if err := goose.SetDialect("postgres"); err != nil {
		return nil, err
	}

	current, err := goose.EnsureDBVersionContext(ctx, e.db)
	if err != nil {
		return nil, err
	}

	pending, err := goose.CollectMigrations(e.directory, current, math.MaxInt64)
	if err != nil {
		if err == goose.ErrNoMigrationFiles {
			return nil, nil
		}
		return nil, err
	}

	return convertGooseMigrations(pending), nil
}

func (e GooseExecutor) Apply(ctx context.Context) ([]Migration, error) {
	pending, err := e.Pending(ctx)
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return nil, nil
	}
	if err := goose.UpContext(ctx, e.db, e.directory); err != nil {
		return nil, err
	}

	return pending, nil
}

type PostgresAdvisoryLocker struct {
	db  *sql.DB
	key int64
}

func NewPostgresAdvisoryLocker(db *sql.DB, key string) PostgresAdvisoryLocker {
	return PostgresAdvisoryLocker{
		db:  db,
		key: AdvisoryLockKey(key),
	}
}

func (l PostgresAdvisoryLocker) Lock(ctx context.Context) error {
	if _, err := l.db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", l.key); err != nil {
		return fmt.Errorf("acquire migration advisory lock: %w", err)
	}
	return nil
}

func (l PostgresAdvisoryLocker) Unlock(ctx context.Context) error {
	var unlocked bool
	if err := l.db.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", l.key).Scan(&unlocked); err != nil {
		return fmt.Errorf("release migration advisory lock: %w", err)
	}
	if !unlocked {
		return fmt.Errorf("migration advisory lock was not held")
	}
	return nil
}

func AdvisoryLockKey(value string) int64 {
	if value == "" {
		value = "tidewise_schema_migration"
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(value))
	return int64(hash.Sum64())
}

func ResolveDirectory(directory string) string {
	if directory == "" {
		return directory
	}
	if exists(directory) {
		return directory
	}
	candidate := filepath.Join("backend", directory)
	if exists(candidate) {
		return candidate
	}
	return directory
}

func convertGooseMigrations(items goose.Migrations) []Migration {
	migrations := make([]Migration, 0, len(items))
	for _, item := range items {
		migrations = append(migrations, Migration{
			Version: fmt.Sprintf("%06d", item.Version),
			Name:    filepath.Base(item.Source),
			Path:    item.Source,
		})
	}
	return migrations
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
