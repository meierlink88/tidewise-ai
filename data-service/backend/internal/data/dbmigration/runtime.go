package dbmigration

import (
	"context"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
)

func CheckPostgres(ctx context.Context, cfg conf.Config, autoApply bool) (ServiceReport, error) {
	return CheckPostgresWithOptions(ctx, cfg, ServiceOptions{AutoApply: autoApply})
}

func CheckPostgresWithOptions(ctx context.Context, cfg conf.Config, options ServiceOptions) (ServiceReport, error) {
	db, err := data.OpenPostgres(ctx, cfg)
	if err != nil {
		return ServiceReport{}, err
	}
	defer db.Close()

	executor := NewGooseExecutor(db, cfg.Migration.Directory)
	locker := NewPostgresAdvisoryLocker(db, cfg.Migration.LockKey)
	service := NewService(executor, locker)

	return service.Check(ctx, options)
}
