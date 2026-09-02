package evidence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const (
	preV74RecoveryLockName    = "tidewise:pre-v74-evidence-recovery"
	preV74RecoveryLockSQL     = `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`
	preV74MigrationVersionSQL = `
SELECT version_id, is_applied
FROM goose_db_version
ORDER BY id DESC
LIMIT 1`
	preV74CountsSQL = `
SELECT
    (SELECT COUNT(*) FROM raw_evidences) AS raw_evidences,
    (SELECT COUNT(*) FROM evidences) AS evidences,
    (SELECT COUNT(*) FROM raw_evidence_category_links) AS raw_evidence_category_links,
    (SELECT COUNT(*) FROM events) AS events,
    (SELECT COUNT(*) FROM event_evidence_links) AS event_evidence_links,
    (SELECT COUNT(*) FROM event_actor_links) AS event_actor_links,
    (SELECT COUNT(*) FROM event_asset_links) AS event_asset_links,
    (SELECT COUNT(*) FROM event_publication_receipts) AS event_publication_receipts`
	deleteRawEvidenceCategoryLinksSQL = `DELETE FROM raw_evidence_category_links`
	deleteEvidenceSQL                 = `DELETE FROM evidences`
	deleteRawEvidenceSQL              = `DELETE FROM raw_evidences`
)

// PreV74EvidenceCounts reports only aggregate row counts. It deliberately does
// not expose any Evidence content in operational logs or workflow artifacts.
type PreV74EvidenceCounts struct {
	RawEvidences             int64 `json:"raw_evidences"`
	Evidences                int64 `json:"evidences"`
	RawEvidenceCategoryLinks int64 `json:"raw_evidence_category_links"`
	Events                   int64 `json:"events"`
	EventEvidenceLinks       int64 `json:"event_evidence_links"`
	EventActorLinks          int64 `json:"event_actor_links"`
	EventAssetLinks          int64 `json:"event_asset_links"`
	EventPublicationReceipts int64 `json:"event_publication_receipts"`
}

func (c PreV74EvidenceCounts) empty() bool {
	return c == (PreV74EvidenceCounts{})
}

func (c PreV74EvidenceCounts) eventDatasetEmpty() bool {
	return c.Events == 0 && c.EventEvidenceLinks == 0 && c.EventActorLinks == 0 &&
		c.EventAssetLinks == 0 && c.EventPublicationReceipts == 0
}

type PreV74EvidenceRecoveryReport struct {
	MigrationVersion int64                `json:"migration_version"`
	Applied          bool                 `json:"applied"`
	Before           PreV74EvidenceCounts `json:"before"`
	After            PreV74EvidenceCounts `json:"after"`
}

// RecoverPreV74Evidence clears only the legacy Raw Evidence aggregate that
// migration 74 cannot convert. The operation is valid only at migration 73 and
// only while the Event aggregate remains empty. Apply mode executes all deletes
// and verification in one serializable transaction.
func RecoverPreV74Evidence(ctx context.Context, db *sql.DB, apply bool) (PreV74EvidenceRecoveryReport, error) {
	if db == nil {
		return PreV74EvidenceRecoveryReport{}, errors.New("Evidence database is required")
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return PreV74EvidenceRecoveryReport{}, fmt.Errorf("begin pre-v74 Evidence recovery: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, preV74RecoveryLockSQL, preV74RecoveryLockName); err != nil {
		return PreV74EvidenceRecoveryReport{}, fmt.Errorf("lock pre-v74 Evidence recovery: %w", err)
	}

	version, applied, err := readPreV74MigrationVersion(ctx, tx)
	if err != nil {
		return PreV74EvidenceRecoveryReport{}, err
	}
	if version != 73 || !applied {
		return PreV74EvidenceRecoveryReport{}, fmt.Errorf("pre-v74 Evidence recovery requires applied migration 73, found version %d applied=%t", version, applied)
	}

	before, err := readPreV74Counts(ctx, tx)
	if err != nil {
		return PreV74EvidenceRecoveryReport{}, err
	}
	report := PreV74EvidenceRecoveryReport{
		MigrationVersion: version,
		Applied:          false,
		Before:           before,
		After:            before,
	}
	if !before.eventDatasetEmpty() {
		return report, errors.New("pre-v74 Evidence recovery requires an empty Event dataset")
	}
	if !apply {
		if err := tx.Rollback(); err != nil {
			return PreV74EvidenceRecoveryReport{}, fmt.Errorf("finish pre-v74 Evidence recovery check: %w", err)
		}
		return report, nil
	}

	if err := deleteExpectedRows(ctx, tx, deleteRawEvidenceCategoryLinksSQL, before.RawEvidenceCategoryLinks); err != nil {
		return report, fmt.Errorf("clear Raw Evidence category links: %w", err)
	}
	if err := deleteExpectedRows(ctx, tx, deleteEvidenceSQL, before.Evidences); err != nil {
		return report, fmt.Errorf("clear Evidence: %w", err)
	}
	if err := deleteExpectedRows(ctx, tx, deleteRawEvidenceSQL, before.RawEvidences); err != nil {
		return report, fmt.Errorf("clear Raw Evidence: %w", err)
	}

	after, err := readPreV74Counts(ctx, tx)
	if err != nil {
		return report, err
	}
	report.After = after
	if !after.empty() {
		return report, errors.New("pre-v74 Evidence recovery verification found remaining incompatible rows")
	}
	if err := tx.Commit(); err != nil {
		return report, fmt.Errorf("commit pre-v74 Evidence recovery: %w", err)
	}
	report.Applied = true
	return report, nil
}

func readPreV74MigrationVersion(ctx context.Context, tx *sql.Tx) (int64, bool, error) {
	var version int64
	var applied bool
	if err := tx.QueryRowContext(ctx, preV74MigrationVersionSQL).Scan(&version, &applied); err != nil {
		return 0, false, fmt.Errorf("read migration version for pre-v74 Evidence recovery: %w", err)
	}
	return version, applied, nil
}

func readPreV74Counts(ctx context.Context, tx *sql.Tx) (PreV74EvidenceCounts, error) {
	var counts PreV74EvidenceCounts
	err := tx.QueryRowContext(ctx, preV74CountsSQL).Scan(
		&counts.RawEvidences,
		&counts.Evidences,
		&counts.RawEvidenceCategoryLinks,
		&counts.Events,
		&counts.EventEvidenceLinks,
		&counts.EventActorLinks,
		&counts.EventAssetLinks,
		&counts.EventPublicationReceipts,
	)
	if err != nil {
		return PreV74EvidenceCounts{}, fmt.Errorf("count pre-v74 Evidence recovery rows: %w", err)
	}
	return counts, nil
}

func deleteExpectedRows(ctx context.Context, tx *sql.Tx, statement string, expected int64) error {
	result, err := tx.ExecContext(ctx, statement)
	if err != nil {
		return err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if deleted != expected {
		return fmt.Errorf("deleted %d rows, expected %d", deleted, expected)
	}
	return nil
}
