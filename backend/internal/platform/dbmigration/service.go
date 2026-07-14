package dbmigration

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	if !options.AutoApply {
		version, pending, err := s.readState(ctx)
		if err != nil {
			return ServiceReport{}, err
		}
		return ServiceReport{CurrentVersion: version, Pending: pending, Remaining: append([]Migration(nil), pending...)}, nil
	}

	if err := s.locker.Lock(ctx); err != nil {
		return ServiceReport{}, err
	}
	beforeVersion, beforePending, stateErr := s.readState(ctx)
	if stateErr != nil {
		return ServiceReport{}, errors.Join(stateErr, s.locker.Unlock(ctx))
	}
	report := ServiceReport{
		CurrentVersion: beforeVersion,
		Pending:        append([]Migration(nil), beforePending...),
		Remaining:      append([]Migration(nil), beforePending...),
	}
	if len(beforePending) == 0 && strings.TrimSpace(options.TargetVersion) == "" {
		return report, s.locker.Unlock(ctx)
	}

	_, applyErr := s.executor.Apply(ctx, options.TargetVersion)
	afterVersion, afterPending, afterErr := s.readState(ctx)
	deriveErr := error(nil)
	if afterErr == nil {
		report.CurrentVersion = afterVersion
		report.Remaining = append([]Migration(nil), afterPending...)
		report.Applied, deriveErr = appliedMigrationDifference(beforePending, afterPending, afterVersion)
	}
	validationErr := error(nil)
	if applyErr == nil && afterErr == nil {
		validationErr = validateAppliedState(options.TargetVersion, afterVersion, afterPending)
	}
	unlockErr := s.locker.Unlock(ctx)
	if err := errors.Join(applyErr, afterErr, deriveErr, validationErr, unlockErr); err != nil {
		return report, err
	}

	return report, nil
}

func (s Service) readState(ctx context.Context) (string, []Migration, error) {
	version, err := s.executor.CurrentVersion(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("read migration version: %w", err)
	}
	pending, err := s.executor.Pending(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("read pending migrations: %w", err)
	}
	return version, pending, nil
}

func appliedMigrationDifference(before, after []Migration, currentVersion string) ([]Migration, error) {
	current, err := strconv.ParseInt(strings.TrimSpace(currentVersion), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse post-apply migration version %q: %w", currentVersion, err)
	}
	remainingVersions := make(map[string]struct{}, len(after))
	for _, migration := range after {
		remainingVersions[migration.Version] = struct{}{}
	}
	applied := make([]Migration, 0, len(before))
	for _, migration := range before {
		if _, remains := remainingVersions[migration.Version]; !remains {
			version, err := strconv.ParseInt(strings.TrimSpace(migration.Version), 10, 64)
			if err != nil {
				return applied, fmt.Errorf("parse migration version %q: %w", migration.Version, err)
			}
			if version > current {
				return applied, fmt.Errorf("post-apply migration state is inconsistent: version %s is absent from pending but current version is %d", migration.Version, current)
			}
			applied = append(applied, migration)
		}
	}
	return applied, nil
}

func validateAppliedState(targetVersion, currentVersion string, remaining []Migration) error {
	if strings.TrimSpace(targetVersion) == "" {
		if len(remaining) != 0 {
			return fmt.Errorf("migration apply completed but %d migrations remain pending", len(remaining))
		}
		return nil
	}
	target, err := parseTargetVersion(targetVersion)
	if err != nil {
		return err
	}
	current, err := strconv.ParseInt(strings.TrimSpace(currentVersion), 10, 64)
	if err != nil {
		return fmt.Errorf("parse post-apply migration version %q: %w", currentVersion, err)
	}
	if current != target {
		return fmt.Errorf("migration apply did not reach target version %d; current version is %d", target, current)
	}
	return nil
}
