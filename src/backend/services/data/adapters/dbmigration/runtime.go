package dbmigration

import (
	"context"

	"github.com/meierlink88/tidewise-ai/backend/services/data/adapters/database"
	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
)

func CheckPostgres(ctx context.Context, cfg config.Config, autoApply bool) (ServiceReport, error) {
	return CheckPostgresWithOptions(ctx, cfg, ServiceOptions{AutoApply: autoApply})
}

func CheckPostgresWithOptions(ctx context.Context, cfg config.Config, options ServiceOptions) (ServiceReport, error) {
	db, err := database.Open(ctx, cfg)
	if err != nil {
		return ServiceReport{}, err
	}
	defer db.Close()

	executor := NewGooseExecutor(db, cfg.Migration.Directory)
	locker := NewPostgresAdvisoryLocker(db, cfg.Migration.LockKey)
	service := NewService(executor, locker)

	return service.Check(ctx, options)
}
