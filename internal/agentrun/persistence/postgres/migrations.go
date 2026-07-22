package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

var expectedSchemaColumns = map[string]string{
	"schema_migrations.version": "text:NO", "schema_migrations.applied_at": "timestamptz:NO",
	"agent_definitions.agent_key": "text:NO", "agent_definitions.display_name": "text:NO", "agent_definitions.created_at": "timestamptz:NO",
	"agent_versions.version": "text:NO", "agent_versions.agent_key": "text:NO", "agent_versions.created_at": "timestamptz:NO",
	"agent_executions.execution_id": "uuid:NO", "agent_executions.agent_version": "text:NO", "agent_executions.idempotency_key": "text:NO",
	"agent_executions.prompt": "text:NO", "agent_executions.prompt_sha256": "bpchar:NO", "agent_executions.prompt_bytes": "int4:NO",
	"agent_executions.status": "text:NO", "agent_executions.error_code": "text:YES", "agent_executions.error_summary": "text:YES",
	"agent_executions.candidate_counts": "jsonb:NO", "agent_executions.artifacts": "jsonb:NO", "agent_executions.created_at": "timestamptz:NO",
	"agent_executions.started_at": "timestamptz:YES", "agent_executions.completed_at": "timestamptz:YES", "agent_executions.updated_at": "timestamptz:NO",
	"connector_invocations.execution_id": "uuid:NO", "connector_invocations.connector_key": "text:NO", "connector_invocations.position": "int2:NO",
	"connector_invocations.status": "text:NO", "connector_invocations.result_count": "int4:NO", "connector_invocations.error_code": "text:YES",
	"connector_invocations.error_summary": "text:YES", "connector_invocations.started_at": "timestamptz:YES", "connector_invocations.completed_at": "timestamptz:YES",
	"provider_configs.provider_key": "text:NO", "provider_configs.base_url": "text:NO", "provider_configs.model": "text:NO",
	"provider_configs.api_key": "text:NO", "provider_configs.updated_at": "timestamptz:NO",
}

var expectedSchemaConstraints = map[string]struct{}{
	"agent_definitions_pkey": {}, "agent_versions_pkey": {}, "agent_versions_agent_key_fkey": {},
	"agent_executions_pkey": {}, "agent_executions_agent_version_fkey": {}, "agent_executions_idempotency_key_key": {},
	"agent_executions_prompt_bytes_check": {}, "agent_executions_status_check": {},
	"connector_invocations_pkey": {}, "connector_invocations_execution_id_fkey": {}, "connector_invocations_execution_id_position_key": {},
	"connector_invocations_result_count_check": {}, "connector_invocations_status_check": {}, "provider_configs_pkey": {},
}

func Migrate(ctx context.Context, database *pgxpool.Pool) error {
	if database == nil {
		return errors.New("AgentRun database is required")
	}
	entries, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries)
	if _, err := database.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	for _, name := range entries {
		if err := applyMigration(ctx, database, name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SchemaReady(ctx context.Context) bool {
	entries, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return false
	}
	rows, err := s.database.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return false
	}
	defer rows.Close()
	applied := make(map[string]struct{}, len(entries))
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return false
		}
		applied[version] = struct{}{}
	}
	if rows.Err() != nil || len(applied) != len(entries) {
		return false
	}
	for _, entry := range entries {
		if _, exists := applied[entry]; !exists {
			return false
		}
	}
	return s.schemaShapeReady(ctx)
}

func (s *Store) schemaShapeReady(ctx context.Context) bool {
	rows, err := s.database.Query(ctx, `
		SELECT table_name, column_name, udt_name, is_nullable
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = ANY($1)
	`, []string{"schema_migrations", "agent_definitions", "agent_versions", "agent_executions", "connector_invocations", "provider_configs"})
	if err != nil {
		return false
	}
	columns := make(map[string]string, len(expectedSchemaColumns))
	for rows.Next() {
		var table, column, dataType, nullable string
		if rows.Scan(&table, &column, &dataType, &nullable) != nil {
			rows.Close()
			return false
		}
		columns[table+"."+column] = dataType + ":" + nullable
	}
	if rows.Err() != nil {
		rows.Close()
		return false
	}
	rows.Close()
	if len(columns) != len(expectedSchemaColumns) {
		return false
	}
	for key, expected := range expectedSchemaColumns {
		if columns[key] != expected {
			return false
		}
	}

	rows, err = s.database.Query(ctx, `
		SELECT constraint_name
		FROM information_schema.table_constraints
		WHERE constraint_schema = current_schema()
		  AND table_name = ANY($1)
	`, []string{"agent_definitions", "agent_versions", "agent_executions", "connector_invocations", "provider_configs"})
	if err != nil {
		return false
	}
	constraints := make(map[string]struct{}, len(expectedSchemaConstraints))
	for rows.Next() {
		var name string
		if rows.Scan(&name) != nil {
			rows.Close()
			return false
		}
		constraints[name] = struct{}{}
	}
	if rows.Err() != nil {
		rows.Close()
		return false
	}
	rows.Close()
	for name := range expectedSchemaConstraints {
		if _, exists := constraints[name]; !exists {
			return false
		}
	}
	var activeIndex string
	err = s.database.QueryRow(ctx, `
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = current_schema() AND indexname = 'agent_executions_one_active'
	`).Scan(&activeIndex)
	return err == nil && strings.Contains(activeIndex, "UNIQUE INDEX") && strings.Contains(activeIndex, "materializing")
}

func applyMigration(ctx context.Context, database *pgxpool.Pool, name string) error {
	var applied bool
	err := database.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name).Scan(&applied)
	if err != nil {
		return fmt.Errorf("check migration %s: %w", name, err)
	}
	if applied {
		return nil
	}
	sql, err := migrations.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}
	tx, err := database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}
