package dbmigration

import (
	"context"
	"fmt"
)

type Executor interface {
	CurrentVersion(context.Context) (string, error)
	Pending(context.Context) ([]Migration, error)
	Apply(context.Context, string) ([]Migration, error)
}

type ServiceOptions struct {
	AutoApply     bool
	TargetVersion string
}

type ServiceReport struct {
	CurrentVersion string      `json:"current_version"`
	Pending        []Migration `json:"pending"`
	Applied        []Migration `json:"applied"`
	Remaining      []Migration `json:"remaining"`
}

type Service struct {
	executor Executor
	locker   Locker
}

func NewService(executor Executor, locker Locker) Service {
	return Service{executor: executor, locker: locker}
}

func (s Service) Check(ctx context.Context, options ServiceOptions) (ServiceReport, error) {
	version, err := s.executor.CurrentVersion(ctx)
	if err != nil {
		return ServiceReport{}, fmt.Errorf("read migration version: %w", err)
	}

	pending, err := s.executor.Pending(ctx)
	if err != nil {
		return ServiceReport{}, fmt.Errorf("read pending migrations: %w", err)
	}

	report := ServiceReport{
		CurrentVersion: version,
		Pending:        pending,
		Remaining:      append([]Migration(nil), pending...),
	}
	if !options.AutoApply || (len(pending) == 0 && options.TargetVersion == "") {
		return report, nil
	}

	if err := s.locker.Lock(ctx); err != nil {
		return ServiceReport{}, err
	}
	defer func() {
		_ = s.locker.Unlock(ctx)
	}()

	applied, err := s.executor.Apply(ctx, options.TargetVersion)
	if err != nil {
		return ServiceReport{}, err
	}
	report.Applied = applied
	report.Remaining = remainingMigrations(pending, applied)

	return report, nil
}

func remainingMigrations(pending, applied []Migration) []Migration {
	appliedVersions := make(map[string]struct{}, len(applied))
	for _, migration := range applied {
		appliedVersions[migration.Version] = struct{}{}
	}
	remaining := make([]Migration, 0, len(pending))
	for _, migration := range pending {
		if _, exists := appliedVersions[migration.Version]; !exists {
			remaining = append(remaining, migration)
		}
	}
	return remaining
}
