package dbmigration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
)

func CheckPostgres(ctx context.Context, cfg conf.Config, autoApply bool) (ServiceReport, error) {
	return CheckPostgresWithOptions(ctx, cfg, ServiceOptions{AutoApply: autoApply})
}

func CheckPostgresWithOptions(ctx context.Context, cfg conf.Config, options ServiceOptions) (ServiceReport, error) {
	if err := validatePostgresServiceOptions(options); err != nil {
		return ServiceReport{}, err
	}
	db, err := data.OpenPostgres(ctx, cfg)
	if err != nil {
		return ServiceReport{}, err
	}
	defer db.Close()

	executor := NewGooseExecutor(db, cfg.Migration.Directory)
	locker := NewPostgresAdvisoryLocker(db, cfg.Migration.LockKey)
	if options.RebuildEmptySchema {
		return rebuildEmptyPostgresSchema(ctx, db, executor, locker, options.TargetVersion)
	}
	service := NewService(executor, locker)

	return service.Check(ctx, options)
}

func validatePostgresServiceOptions(options ServiceOptions) error {
	if options.RebuildEmptySchema && (!options.AutoApply || options.TargetVersion != "58") {
		return fmt.Errorf("empty Data schema rebuild requires auto-apply target 58")
	}
	return nil
}

func rebuildEmptyPostgresSchema(
	ctx context.Context,
	db sqlExecutor,
	executor Executor,
	locker Locker,
	targetVersion string,
) (ServiceReport, error) {
	if targetVersion != "58" {
		return ServiceReport{}, fmt.Errorf("empty Data schema rebuild target must be 58")
	}
	if err := locker.Lock(ctx); err != nil {
		return ServiceReport{}, err
	}
	if err := resetPublicSchema(ctx, db); err != nil {
		return ServiceReport{}, errors.Join(err, locker.Unlock(ctx))
	}
	beforePending, err := executor.Pending(ctx)
	if err != nil {
		return ServiceReport{}, errors.Join(fmt.Errorf("read migrations after Data schema reset: %w", err), locker.Unlock(ctx))
	}
	_, applyErr := executor.Apply(ctx, targetVersion)
	afterVersion, versionErr := executor.CurrentVersion(ctx)
	afterPending, pendingErr := executor.Pending(ctx)
	report := ServiceReport{
		CurrentVersion: afterVersion,
		Pending:        append([]Migration(nil), beforePending...),
		Remaining:      append([]Migration(nil), afterPending...),
	}
	deriveErr := error(nil)
	if versionErr == nil && pendingErr == nil {
		report.Applied, deriveErr = appliedMigrationDifference(beforePending, afterPending, afterVersion)
	}
	validationErr := error(nil)
	if applyErr == nil && versionErr == nil && pendingErr == nil {
		validationErr = validateAppliedState(targetVersion, afterVersion, afterPending)
	}
	unlockErr := locker.Unlock(ctx)
	if err := errors.Join(applyErr, versionErr, pendingErr, deriveErr, validationErr, unlockErr); err != nil {
		return report, err
	}
	return report, nil
}

type sqlExecutor interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func resetPublicSchema(ctx context.Context, db sqlExecutor) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin empty Data schema rebuild: %w", err)
	}
	for _, statement := range []string{
		"DROP SCHEMA public CASCADE",
		"CREATE SCHEMA public AUTHORIZATION CURRENT_USER",
		"GRANT USAGE ON SCHEMA public TO public",
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return errors.Join(fmt.Errorf("rebuild empty Data schema: %w", err), tx.Rollback())
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit empty Data schema rebuild: %w", err)
	}
	return nil
}
