package dbmigration

import (
	"context"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/database"
)

func CheckPostgres(ctx context.Context, cfg config.Config, autoApply bool) (ServiceReport, error) {
	db, err := database.Open(ctx, cfg)
	if err != nil {
		return ServiceReport{}, err
	}
	defer db.Close()

	executor := NewGooseExecutor(db, cfg.Migration.Directory)
	locker := NewPostgresAdvisoryLocker(db, cfg.Migration.LockKey)
	service := NewService(executor, locker)

	return service.Check(ctx, ServiceOptions{AutoApply: autoApply})
}
