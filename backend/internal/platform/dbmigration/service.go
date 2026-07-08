package dbmigration

import (
	"context"
	"fmt"
)

type Executor interface {
	CurrentVersion(context.Context) (string, error)
	Pending(context.Context) ([]Migration, error)
	Apply(context.Context) ([]Migration, error)
}

type ServiceOptions struct {
	AutoApply bool
}

type ServiceReport struct {
	CurrentVersion string      `json:"current_version"`
	Pending        []Migration `json:"pending"`
	Applied        []Migration `json:"applied"`
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
	}
	if len(pending) == 0 || !options.AutoApply {
		return report, nil
	}

	if err := s.locker.Lock(ctx); err != nil {
		return ServiceReport{}, err
	}
	defer func() {
		_ = s.locker.Unlock(ctx)
	}()

	applied, err := s.executor.Apply(ctx)
	if err != nil {
		return ServiceReport{}, err
	}
	report.Applied = applied

	return report, nil
}
