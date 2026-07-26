package dbmigration

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	readOnlyLedgerExistsSQL = "SELECT to_regclass('goose_db_version')::text"
	readOnlyLedgerRowsSQL   = "SELECT version_id, is_applied FROM goose_db_version ORDER BY id"
)

// RequirePostgresReadyReadOnly checks an existing Goose ledger inside a
// server-enforced read-only transaction. It never invokes Goose's ledger
// bootstrap or migration apply paths.
func RequirePostgresReadyReadOnly(ctx context.Context, db *sql.DB, directory string) (ServiceReport, error) {
	if db == nil {
		return ServiceReport{}, fmt.Errorf("PostgreSQL connection is required")
	}
	migrations, err := (FileSource{Dir: ResolveDirectory(directory)}).ListMigrations(ctx)
	if err != nil {
		return ServiceReport{}, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return ServiceReport{}, fmt.Errorf("begin read-only migration readiness transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var ledger sql.NullString
	if err := tx.QueryRowContext(ctx, readOnlyLedgerExistsSQL).Scan(&ledger); err != nil {
		return ServiceReport{}, fmt.Errorf("check existing Goose ledger: %w", err)
	}
	if !ledger.Valid || ledger.String == "" {
		return ServiceReport{}, fmt.Errorf("existing Goose migration ledger is required")
	}

	rows, err := tx.QueryContext(ctx, readOnlyLedgerRowsSQL)
	if err != nil {
		return ServiceReport{}, fmt.Errorf("read existing Goose ledger: %w", err)
	}
	applied := make(map[string]bool)
	rowCount := 0
	for rows.Next() {
		var version int64
		var isApplied bool
		if err := rows.Scan(&version, &isApplied); err != nil {
			_ = rows.Close()
			return ServiceReport{}, fmt.Errorf("scan existing Goose ledger: %w", err)
		}
		if version < 0 {
			_ = rows.Close()
			return ServiceReport{}, fmt.Errorf("Goose ledger contains invalid version %d", version)
		}
		rowCount++
		if version != 0 {
			applied[fmt.Sprintf("%06d", version)] = isApplied
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return ServiceReport{}, fmt.Errorf("iterate existing Goose ledger: %w", err)
	}
	if err := rows.Close(); err != nil {
		return ServiceReport{}, fmt.Errorf("close existing Goose ledger rows: %w", err)
	}
	if rowCount == 0 {
		return ServiceReport{}, fmt.Errorf("existing Goose migration ledger is empty")
	}

	known := make(map[string]struct{}, len(migrations))
	report := ServiceReport{}
	for _, migration := range migrations {
		known[migration.Version] = struct{}{}
		if !applied[migration.Version] {
			report.Pending = append(report.Pending, migration)
			continue
		}
		report.CurrentVersion = migration.Version
	}
	for version, isApplied := range applied {
		if _, ok := known[version]; isApplied && !ok {
			return report, fmt.Errorf("Goose ledger contains applied version %s absent from repository migrations", version)
		}
	}
	report.Remaining = append([]Migration(nil), report.Pending...)
	if len(report.Pending) != 0 {
		return report, fmt.Errorf("migration readiness failed: %d migration(s) pending", len(report.Pending))
	}
	if err := tx.Commit(); err != nil {
		return report, fmt.Errorf("commit read-only migration readiness transaction: %w", err)
	}
	return report, nil
}
