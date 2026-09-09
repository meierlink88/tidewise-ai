package report

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

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

type publicationTransaction struct {
	tx     *sql.Tx
	record *reportbiz.Record
}

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
FROM report_archive WHERE publisher_report_id = $1`, publisherReportID))
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
	t.record = &record
	report, err := jsonMarshal(record.Report)
	if err != nil {
		return fmt.Errorf("encode Report: %w", err)
	}
	var counts []byte
	if record.EvidenceCounts != nil {
		counts, err = json.Marshal(record.EvidenceCounts)
		if err != nil {
			return fmt.Errorf("encode Report Evidence counts: %w", err)
		}
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO report_archive
    (id, publisher_report_id, content_hash, report, published_at, evidence_counts)
VALUES ($1,$2,$3,$4,$5,$6)`, record.ID, record.PublisherReportID,
		record.ContentHash, report, record.PublishedAt, counts)
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
	if t.record != nil {
		return writeReportStorage(ctx, t.tx, *t.record)
	}
	return nil
}

var jsonMarshal = func(value any) ([]byte, error) {
	return json.Marshal(value)
}

var _ reportbiz.PublicationTransaction = (*publicationTransaction)(nil)

// writeReportStorage projects only the accepted immutable snapshot, using the existing
// Evidence links. It never queries graph facts or changes publisher content/hash.
func writeReportStorage(ctx context.Context, tx *sql.Tx, record reportbiz.Record) error {
	summary, err := scanSummary(tx.QueryRowContext(ctx, `SELECT `+summaryColumns+` FROM report_archive WHERE id=$1`, record.ID))
	if err != nil {
		return err
	}
	raw, err := json.Marshal(record.Report)
	if err != nil {
		return err
	}
	var root map[string]json.RawMessage
	if err = json.Unmarshal(raw, &root); err != nil {
		return err
	}
	kinds := []string{"geopolitical_stories", "macroeconomic_stories", "concept_analyses", "industry_chain_analyses", "company_analyses"}
	metadata := map[string]json.RawMessage{}
	for k, v := range root {
		metadata[k] = v
	}
	for _, k := range append(kinds, "geopolitics", "macroeconomics", "industry_chains") {
		delete(metadata, k)
	}
	meta, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	counts, err := json.Marshal(record.EvidenceCounts)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO report_publications(id,publisher_report_id,content_hash,report,published_at,evidence_counts,has_geopolitics,has_macroeconomics,industry_chain_count) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(id) DO NOTHING`, record.ID, record.PublisherReportID, record.ContentHash, meta, record.PublishedAt, counts, summary.HasGeopolitics, summary.HasMacroeconomics, summary.IndustryChainCount)
	if err != nil {
		return err
	}
	var same bool
	err = tx.QueryRowContext(ctx, `SELECT publisher_report_id=$2 AND content_hash=$3 AND report=$4::jsonb AND published_at=$5 AND evidence_counts=$6::jsonb AND has_geopolitics=$7 AND has_macroeconomics=$8 AND industry_chain_count=$9 FROM report_publications WHERE id=$1`, record.ID, record.PublisherReportID, record.ContentHash, meta, record.PublishedAt, counts, summary.HasGeopolitics, summary.HasMacroeconomics, summary.IndustryChainCount).Scan(&same)
	if err != nil {
		return err
	}
	if !same {
		return errors.New("Report metadata projection conflict")
	}
	if record.Report.V4 == nil {
		return nil
	}
	tokens, err := readScopeTokens(ctx, tx, record.ID)
	if err != nil {
		return err
	}
	expected := 0
	for _, kind := range kinds {
		var units []json.RawMessage
		if len(root[kind]) == 0 {
			continue
		}
		if err = json.Unmarshal(root[kind], &units); err != nil {
			return err
		}
		for i, unitRaw := range units {
			expected++
			var key, source, title string
			var projected any
			if kind == "company_analyses" {
				var c reportbiz.V4Macro
				if err = decodeStoredJSON(unitRaw, &c); err != nil {
					return err
				}
				p, e := projectSignalCompany(c, tokens, record.EvidenceCounts)
				if e != nil {
					return e
				}
				p.Company.VariableSignals = nil
				p.Company.ReasoningSources = nil
				key, source, title, projected = c.LocalKey, c.SourceID, c.Name, p
			} else {
				var u reportbiz.V4Unit
				if err = decodeStoredJSON(unitRaw, &u); err != nil {
					return err
				}
				var p reportbiz.V4ReadUnit
				if err = projectNormalizedEvidence(u, kind+"/"+u.LocalKey, tokens, record.EvidenceCounts, &p); err != nil {
					return err
				}
				card, e := normalizedSummary(p, i+1)
				if e != nil {
					return e
				}
				key, source, title, projected = u.LocalKey, u.SourceID, u.Title, card.V4
			}
			id, e := reportbiz.AnalysisIdentity(record.ID, kind, key)
			if e != nil {
				return e
			}
			card, e := json.Marshal(projected)
			if e != nil {
				return e
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO report_summary(id,report_id,analysis_kind,local_key,source_id,title,ordinal,summary_data) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO NOTHING`, id, record.ID, kind, key, source, title, i+1, card)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO report_detail(id,detail_data) VALUES($1,$2) ON CONFLICT(id) DO NOTHING`, id, []byte(unitRaw))
			if err != nil {
				return err
			}
			err = tx.QueryRowContext(ctx, `SELECT s.report_id=$2 AND s.analysis_kind=$3 AND s.local_key=$4 AND s.source_id=$5 AND s.title=$6 AND s.ordinal=$7 AND s.summary_data=$8::jsonb AND d.detail_data=$9::jsonb FROM report_summary s JOIN report_detail d ON d.id=s.id WHERE s.id=$1`, id, record.ID, kind, key, source, title, i+1, card, []byte(unitRaw)).Scan(&same)
			if err != nil {
				return err
			}
			if !same {
				return errors.New("Report analysis projection conflict")
			}
		}
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM report_summary WHERE report_id=$1`, record.ID).Scan(&count); err != nil {
		return err
	}
	if count != expected {
		return errors.New("Report analysis projection count mismatch")
	}
	return nil
}

// StoragePlan is a frozen, explicit maintenance inventory, never an automatic retention rule.
type StoragePlan struct {
	RetainFrom time.Time       `json:"retain_from"`
	Keep       []StorageRecord `json:"keep"`
	Delete     []StorageRecord `json:"delete"`
}
type StorageRecord struct {
	ID          string    `json:"id"`
	ContentHash string    `json:"content_hash"`
	PublishedAt time.Time `json:"published_at"`
}

func planStorage(ctx context.Context, tx *sql.Tx, cutoff time.Time) (StoragePlan, error) {
	plan := StoragePlan{RetainFrom: cutoff, Keep: []StorageRecord{}, Delete: []StorageRecord{}}
	rows, err := tx.QueryContext(ctx, `SELECT id,content_hash,published_at FROM report_archive ORDER BY published_at,id`)
	if err != nil {
		return plan, err
	}
	defer rows.Close()
	for rows.Next() {
		var r StorageRecord
		if err = rows.Scan(&r.ID, &r.ContentHash, &r.PublishedAt); err != nil {
			return plan, err
		}
		if !cutoff.IsZero() && r.PublishedAt.Before(cutoff) {
			plan.Delete = append(plan.Delete, r)
		} else {
			plan.Keep = append(plan.Keep, r)
		}
	}
	return plan, rows.Err()
}

// PlanStorage reads identity/hash/time only. Applying it requires an unchanged inventory
// and an explicit backup reference supplied by the maintenance operator.
func (s Store) PlanStorage(ctx context.Context, cutoff time.Time) (StoragePlan, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return StoragePlan{}, err
	}
	defer tx.Rollback()
	p, err := planStorage(ctx, tx, cutoff)
	if err != nil {
		return p, err
	}
	return p, tx.Commit()
}
func (s Store) ApplyStorage(ctx context.Context, expected StoragePlan, backupReference string) (resultErr error) {
	if strings.TrimSpace(backupReference) == "" {
		return errors.New("verified backup reference is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if e := tx.Rollback(); e != nil && !errors.Is(e, sql.ErrTxDone) {
			resultErr = errors.Join(resultErr, e)
		}
	}()
	// Prevent concurrent publication or maintenance between inventory and cutover.
	if _, err = tx.ExecContext(ctx, `LOCK TABLE report_archive,report_publications,report_summary,report_detail,report_evidence_links IN ACCESS EXCLUSIVE MODE`); err != nil {
		return err
	}
	actual, err := planStorage(ctx, tx, expected.RetainFrom)
	if err != nil {
		return err
	}
	a, err := json.Marshal(actual)
	if err != nil {
		return err
	}
	b, err := json.Marshal(expected)
	if err != nil {
		return err
	}
	if !bytes.Equal(a, b) {
		return errors.New("Report storage inventory changed; regenerate plan")
	}
	for _, item := range actual.Keep {
		record, e := scanRecord(tx.QueryRowContext(ctx, `SELECT id,publisher_report_id,content_hash,report,published_at FROM report_archive WHERE id=$1`, item.ID))
		if e != nil {
			return e
		}
		// Load the published counts (including NULL for older contracts), not regenerated IDs.
		var counts []byte
		if e = tx.QueryRowContext(ctx, `SELECT evidence_counts FROM report_archive WHERE id=$1`, item.ID).Scan(&counts); e != nil {
			return e
		}
		if len(counts) > 0 {
			if e = json.Unmarshal(counts, &record.EvidenceCounts); e != nil {
				return e
			}
		}
		if e = writeReportStorage(ctx, tx, record); e != nil {
			return e
		}
	}
	// Only report-owned rows are deleted, after all retained projections verify.
	if len(actual.Delete) > 0 {
		for _, statement := range []string{
			`ALTER TABLE report_detail DISABLE TRIGGER report_detail_immutable`,
			`ALTER TABLE report_summary DISABLE TRIGGER report_summary_immutable`,
			`ALTER TABLE report_publications DISABLE TRIGGER report_publications_immutable`,
			`ALTER TABLE report_evidence_links DISABLE TRIGGER trg_report_evidence_links_immutable`,
			`ALTER TABLE report_archive DISABLE TRIGGER trg_reports_immutable`,
		} {
			if _, err = tx.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
		ids := make([]string, len(actual.Delete))
		for i, r := range actual.Delete {
			ids[i] = r.ID
		}
		for _, statement := range []string{
			`DELETE FROM report_detail WHERE id IN(SELECT id FROM report_summary WHERE report_id=ANY($1::text[]))`,
			`DELETE FROM report_summary WHERE report_id=ANY($1::text[])`,
			`DELETE FROM report_publications WHERE id=ANY($1::text[])`,
			`DELETE FROM report_evidence_links WHERE report_id=ANY($1::text[])`,
			`DELETE FROM report_archive WHERE id=ANY($1::text[])`,
		} {
			if _, err = tx.ExecContext(ctx, statement, ids); err != nil {
				return err
			}
		}
		for _, statement := range []string{
			`ALTER TABLE report_detail ENABLE TRIGGER report_detail_immutable`,
			`ALTER TABLE report_summary ENABLE TRIGGER report_summary_immutable`,
			`ALTER TABLE report_publications ENABLE TRIGGER report_publications_immutable`,
			`ALTER TABLE report_evidence_links ENABLE TRIGGER trg_report_evidence_links_immutable`,
			`ALTER TABLE report_archive ENABLE TRIGGER trg_reports_immutable`,
		} {
			if _, err = tx.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// VerifyStorage is a read-only release gate: normalized reports must have every
// summary/detail pair before the new application can serve traffic.
func (s Store) VerifyStorage(ctx context.Context) error {
	var missing int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM report_archive a LEFT JOIN report_publications p ON p.id=a.id WHERE p.id IS NULL OR (a.report->>'schema_version' IN ('report-publication/v4','report-publication/v5') AND ((SELECT count(*) FROM report_summary s JOIN report_detail d ON d.id=s.id WHERE s.report_id=a.id) <> (SELECT count(*) FROM jsonb_each(a.report) kv CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN kv.key IN ('geopolitical_stories','macroeconomic_stories','concept_analyses','industry_chain_analyses','company_analyses') THEN kv.value ELSE '[]'::jsonb END) u)))`).Scan(&missing)
	if err != nil {
		return err
	}
	if missing != 0 {
		return fmt.Errorf("%d Report archives lack complete split storage; run explicit maintenance before serving", missing)
	}
	return nil
}
