package migrations

import (
	"context"
	"fmt"
	"sort"
)

type Migration struct {
	Version string
	Name    string
	Path    string
}

type Source interface {
	ListMigrations(context.Context) ([]Migration, error)
}

type Store interface {
	AppliedVersions(context.Context) (map[string]bool, error)
	Apply(context.Context, Migration) error
}

type Locker interface {
	Lock(context.Context) error
	Unlock(context.Context) error
}

type RunnerOptions struct {
	AutoApply bool
}

type Report struct {
	Pending []Migration
	Applied []Migration
}

type Runner struct {
	source Source
	store  Store
	locker Locker
}

func NewRunner(source Source, store Store, locker Locker) Runner {
	return Runner{
		source: source,
		store:  store,
		locker: locker,
	}
}

func (r Runner) Check(ctx context.Context, options RunnerOptions) (Report, error) {
	migrations, err := r.source.ListMigrations(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("list migrations: %w", err)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	appliedVersions, err := r.store.AppliedVersions(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("read applied migration versions: %w", err)
	}

	report := Report{}
	for _, migration := range migrations {
		if !appliedVersions[migration.Version] {
			report.Pending = append(report.Pending, migration)
		}
	}

	if len(report.Pending) == 0 || !options.AutoApply {
		return report, nil
	}

	if err := r.locker.Lock(ctx); err != nil {
		return Report{}, err
	}
	defer func() {
		_ = r.locker.Unlock(ctx)
	}()

	for _, migration := range report.Pending {
		if err := r.store.Apply(ctx, migration); err != nil {
			return report, err
		}
		report.Applied = append(report.Applied, migration)
	}

	return report, nil
}
