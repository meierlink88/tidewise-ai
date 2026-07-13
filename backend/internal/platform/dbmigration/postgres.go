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
	"strings"

	"github.com/pressly/goose/v3"
)

type GooseExecutor struct {
	db         *sql.DB
	directory  string
	operations gooseOperations
}

type gooseOperations interface {
	currentVersion(context.Context, *sql.DB) (int64, error)
	pending(context.Context, *sql.DB, string, int64) ([]Migration, error)
	up(context.Context, *sql.DB, string) error
	upTo(context.Context, *sql.DB, string, int64) error
}

type defaultGooseOperations struct{}

func NewGooseExecutor(db *sql.DB, directory string) GooseExecutor {
	return GooseExecutor{
		db:         db,
		directory:  ResolveDirectory(directory),
		operations: defaultGooseOperations{},
	}
}

func (e GooseExecutor) goose() gooseOperations {
	if e.operations == nil {
		return defaultGooseOperations{}
	}
	return e.operations
}

func (e GooseExecutor) CurrentVersion(ctx context.Context) (string, error) {
	if err := goose.SetDialect("postgres"); err != nil {
		return "", err
	}

	version, err := e.goose().currentVersion(ctx, e.db)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(version, 10), nil
}

func (e GooseExecutor) Pending(ctx context.Context) ([]Migration, error) {
	if err := goose.SetDialect("postgres"); err != nil {
		return nil, err
	}

	current, err := e.goose().currentVersion(ctx, e.db)
	if err != nil {
		return nil, err
	}

	pending, err := e.goose().pending(ctx, e.db, e.directory, current)
	if err != nil {
		return nil, err
	}
	return pending, nil
}

func (e GooseExecutor) Apply(ctx context.Context, targetVersion string) ([]Migration, error) {
	pending, err := e.Pending(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(targetVersion) == "" {
		if len(pending) == 0 {
			return nil, nil
		}
		if err := e.goose().up(ctx, e.db, e.directory); err != nil {
			return nil, err
		}
		return pending, nil
	}

	currentText, err := e.CurrentVersion(ctx)
	if err != nil {
		return nil, err
	}
	current, err := strconv.ParseInt(currentText, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse current migration version %q: %w", currentText, err)
	}
	target, err := parseTargetVersion(targetVersion)
	if err != nil {
		return nil, err
	}
	if target < current {
		return nil, fmt.Errorf("target version %d is behind current version %d; rollback is not supported", target, current)
	}
	if target == current {
		return nil, nil
	}

	selected := make([]Migration, 0, len(pending))
	targetExists := false
	for _, migration := range pending {
		version, err := strconv.ParseInt(migration.Version, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse pending migration version %q: %w", migration.Version, err)
		}
		if version <= target {
			selected = append(selected, migration)
		}
		if version == target {
			targetExists = true
		}
	}
	if !targetExists {
		return nil, fmt.Errorf("target version %d does not match an available migration", target)
	}
	if err := e.goose().upTo(ctx, e.db, e.directory, target); err != nil {
		return nil, err
	}
	return selected, nil
}

func parseTargetVersion(value string) (int64, error) {
	target, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || target < 1 {
		return 0, fmt.Errorf("invalid target version %q", value)
	}
	return target, nil
}

func (defaultGooseOperations) currentVersion(ctx context.Context, db *sql.DB) (int64, error) {
	return goose.EnsureDBVersionContext(ctx, db)
}

func (defaultGooseOperations) pending(_ context.Context, _ *sql.DB, directory string, current int64) ([]Migration, error) {
	pending, err := goose.CollectMigrations(directory, current, math.MaxInt64)
	if err != nil {
		if err == goose.ErrNoMigrationFiles {
			return nil, nil
		}
		return nil, err
	}
	return convertGooseMigrations(pending), nil
}

func (defaultGooseOperations) up(ctx context.Context, db *sql.DB, directory string) error {
	return goose.UpContext(ctx, db, directory)
}

func (defaultGooseOperations) upTo(ctx context.Context, db *sql.DB, directory string, target int64) error {
	return goose.UpToContext(ctx, db, directory, target)
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
