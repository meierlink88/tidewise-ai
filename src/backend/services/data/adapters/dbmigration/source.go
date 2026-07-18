package dbmigration

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
)

type FileSource struct {
	Dir string
}

func (s FileSource) ListMigrations(context.Context) ([]Migration, error) {
	matches, err := filepath.Glob(filepath.Join(s.Dir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("find migration files: %w", err)
	}

	versionPattern := regexp.MustCompile(`^(\d{6})_[a-z0-9_]+\.sql$`)
	seenVersions := map[string]string{}
	migrations := make([]Migration, 0, len(matches))
	for _, path := range matches {
		name := filepath.Base(path)
		parts := versionPattern.FindStringSubmatch(name)
		if len(parts) != 2 {
			return nil, fmt.Errorf("migration file %q must use 000001_name.sql format", name)
		}

		version := parts[1]
		if previous, ok := seenVersions[version]; ok {
			return nil, fmt.Errorf("migration version %s is duplicated by %q and %q", version, previous, name)
		}
		seenVersions[version] = name

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			Path:    path,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}
