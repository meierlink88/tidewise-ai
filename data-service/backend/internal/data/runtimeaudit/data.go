package runtimeaudit

import (
	"context"
	"database/sql"
	"fmt"
)

type Report struct {
	CurrentDatabase string `json:"current_database"`
	CurrentRole     string `json:"current_role"`
	RetiredDatabase string `json:"retired_database"`
	DatabasePresent bool   `json:"retired_database_present"`
	RetiredRole     string `json:"retired_role"`
	RolePresent     bool   `json:"retired_role_present"`
}

type Store struct {
	database *sql.DB
}

func NewStore(database *sql.DB) (Store, error) {
	if database == nil {
		return Store{}, fmt.Errorf("database is required")
	}
	return Store{database: database}, nil
}

func (s Store) Inspect(ctx context.Context, retiredDatabase, retiredRole string) (Report, error) {
	report := Report{RetiredDatabase: retiredDatabase, RetiredRole: retiredRole}
	err := s.database.QueryRowContext(ctx,
		`SELECT current_database(), current_user,
                EXISTS (SELECT 1 FROM pg_database WHERE datname = $1),
                EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $2)`,
		retiredDatabase,
		retiredRole,
	).Scan(&report.CurrentDatabase, &report.CurrentRole, &report.DatabasePresent, &report.RolePresent)
	if err != nil {
		return Report{}, fmt.Errorf("inspect retired PostgreSQL objects: %w", err)
	}
	return report, nil
}
