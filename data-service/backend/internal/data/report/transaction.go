package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
)

func (s Store) InPublicationTransaction(
	ctx context.Context,
	fn func(reportbiz.PublicationTransaction) error,
) (resultErr error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Report publication transaction: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		failure := recover()
		rollbackErr := tx.Rollback()
		if errors.Is(rollbackErr, sql.ErrTxDone) {
			rollbackErr = nil
		}
		if failure != nil {
			if rollbackErr != nil {
				panic(fmt.Errorf("Report publication panic (%v) and rollback failed: %w", failure, rollbackErr))
			}
			panic(failure)
		}
		if rollbackErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("roll back Report publication transaction: %w", rollbackErr))
		}
	}()
	if err := fn(&publicationTransaction{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Report publication transaction: %w", err)
	}
	committed = true
	return nil
}

type publicationTransaction struct{ tx *sql.Tx }

func (t *publicationTransaction) Lock(ctx context.Context, publisherReportID string) error {
	_, err := t.tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		"report-publication:"+publisherReportID)
	if err != nil {
		return fmt.Errorf("lock Report publisher identity: %w", err)
	}
	return nil
}

func (t *publicationTransaction) ReportByPublisherID(ctx context.Context, publisherReportID string) (*reportbiz.Record, error) {
	record, err := scanRecord(t.tx.QueryRowContext(ctx, `SELECT id, publisher_report_id,
       content_hash, report, published_at
FROM reports WHERE publisher_report_id = $1`, publisherReportID))
	if errors.Is(err, reportbiz.ErrReportNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (t *publicationTransaction) ExistingEvidenceIDs(ctx context.Context, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	rows, err := t.tx.QueryContext(ctx, `SELECT id FROM evidences WHERE id = ANY($1::text[]) ORDER BY id`, ids)
	if err != nil {
		return nil, fmt.Errorf("read Report Evidence references: %w", err)
	}
	defer rows.Close()
	result := make([]string, 0, len(ids))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan Report Evidence reference: %w", err)
		}
		result = append(result, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Report Evidence references: %w", err)
	}
	sort.Strings(result)
	return result, nil
}

func (t *publicationTransaction) InsertReport(ctx context.Context, record reportbiz.Record) error {
	report, err := jsonMarshal(record.Report)
	if err != nil {
		return fmt.Errorf("encode Report: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO reports
    (id, publisher_report_id, content_hash, report, published_at)
VALUES ($1,$2,$3,$4,$5)`, record.ID, record.PublisherReportID,
		record.ContentHash, report, record.PublishedAt)
	if err != nil {
		return fmt.Errorf("insert Report %q: %w", record.ID, err)
	}
	return nil
}

func (t *publicationTransaction) InsertEvidenceLinks(ctx context.Context, links []reportbiz.EvidenceLink) error {
	for _, link := range links {
		_, err := t.tx.ExecContext(ctx, `INSERT INTO report_evidence_links
    (id, report_id, evidence_id, scope_type, scope_path, position)
VALUES ($1,$2,$3,$4,$5,$6)`, link.ID, link.ReportID, link.EvidenceID,
			link.ScopeType, link.ScopePath, link.Position)
		if err != nil {
			return fmt.Errorf("insert Report Evidence Link %q: %w", link.ID, err)
		}
	}
	return nil
}

var jsonMarshal = func(value any) ([]byte, error) {
	return json.Marshal(value)
}

var _ reportbiz.PublicationTransaction = (*publicationTransaction)(nil)
