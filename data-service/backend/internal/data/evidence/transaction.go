package evidence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
)

func (s Store) InTransaction(ctx context.Context, fn func(evidencebiz.Transaction) error) (resultErr error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Evidence Publication transaction: %w", err)
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
				panic(fmt.Errorf("Evidence Publication panic (%v) and rollback failed: %w", failure, rollbackErr))
			}
			panic(failure)
		}
		if rollbackErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("roll back Evidence Publication transaction: %w", rollbackErr))
		}
	}()
	wrapper := &transaction{tx: tx}
	if err := fn(wrapper); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Evidence Publication transaction: %w", err)
	}
	committed = true
	return nil
}

type transaction struct{ tx *sql.Tx }

func (t *transaction) LockIdentities(ctx context.Context, identities []string) error {
	keys := append([]string(nil), identities...)
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := t.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
			return fmt.Errorf("lock Evidence Publication identity: %w", err)
		}
	}
	return nil
}

func (t *transaction) RawEvidence(ctx context.Context, id string) (*evidencebiz.StoredRawEvidence, error) {
	var record evidencebiz.StoredRawEvidence
	err := t.tx.QueryRowContext(ctx, `
SELECT id, source_id, source_name, source_level, source_url, is_original,
       quoted_source_id, quoted_source_name, title, raw_text, published_at, collected_at,
       content_hash
FROM raw_evidences
WHERE id = $1`, id).Scan(
		&record.ID, &record.SourceID, &record.SourceName, &record.SourceLevel,
		&record.SourceURL, &record.IsOriginal, &record.QuotedSourceID, &record.QuotedSourceName,
		&record.Title, &record.RawText, &record.PublishedAt, &record.CollectedAt, &record.ContentHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Raw Evidence: %w", err)
	}
	if err := validateStoredRawEvidenceBase(&record, id); err != nil {
		return nil, fmt.Errorf("read Raw Evidence invariant: %w", err)
	}
	categories, err := t.categoriesByRawEvidence(ctx, id)
	if err != nil {
		return nil, err
	}
	record.Categories = categories
	record.CategoryIDs = make([]evidencebiz.CategoryID, len(categories))
	for index, category := range categories {
		record.CategoryIDs[index] = category.ID
	}
	if err := validateStoredRawEvidence(&record, id); err != nil {
		return nil, fmt.Errorf("read Raw Evidence invariant: %w", err)
	}
	return &record, nil
}

func (t *transaction) CategoriesByIDs(ctx context.Context, ids []evidencebiz.CategoryID) ([]evidencebiz.Category, error) {
	if len(ids) == 0 {
		return []evidencebiz.Category{}, nil
	}
	return t.categories(ctx, categorySelect+` WHERE id = ANY($1) ORDER BY id`, categoryIDStrings(ids))
}

func (t *transaction) categoriesByRawEvidence(ctx context.Context, rawEvidenceID string) ([]evidencebiz.Category, error) {
	return t.categories(ctx, categorySelect+`
JOIN raw_evidence_category_links AS link ON link.category_id = category.id
WHERE link.raw_evidence_id = $1
ORDER BY category.id`, rawEvidenceID)
}

func (t *transaction) categories(ctx context.Context, query string, argument any) ([]evidencebiz.Category, error) {
	rows, err := t.tx.QueryContext(ctx, query, argument)
	if err != nil {
		return nil, fmt.Errorf("read Evidence Categories: %w", err)
	}
	defer rows.Close()
	result := make([]evidencebiz.Category, 0)
	for rows.Next() {
		var category evidencebiz.Category
		var id string
		if err := rows.Scan(&id, &category.Code, &category.Name, &category.Description, &category.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan Evidence Category: %w", err)
		}
		category.ID = evidencebiz.CategoryID(id)
		if err := validateStoredCategory(&category); err != nil {
			return nil, fmt.Errorf("read Evidence Category invariant: %w", err)
		}
		result = append(result, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Evidence Categories: %w", err)
	}
	return result, nil
}

func (t *transaction) InsertRawEvidence(ctx context.Context, record evidencebiz.StoredRawEvidence) error {
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO raw_evidences (
    id, source_id, source_name, source_level, source_url, is_original,
    quoted_source_id, quoted_source_name, title, raw_text, published_at, collected_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		record.ID, record.SourceID, record.SourceName, record.SourceLevel,
		record.SourceURL, record.IsOriginal, record.QuotedSourceID, record.QuotedSourceName,
		record.Title, record.RawText, record.PublishedAt, record.CollectedAt,
	)
	if err != nil {
		return fmt.Errorf("insert Raw Evidence: %w", err)
	}
	return nil
}

func (t *transaction) InsertRawEvidenceCategoryLinks(ctx context.Context, rawEvidenceID string, links []evidencebiz.RawEvidenceCategoryLink) error {
	if len(links) == 0 {
		return nil
	}
	linkIDs := make([]string, len(links))
	categoryIDs := make([]evidencebiz.CategoryID, len(links))
	for index, link := range links {
		linkIDs[index] = link.ID
		categoryIDs[index] = link.CategoryID
	}
	_, err := t.tx.ExecContext(ctx, `
INSERT INTO raw_evidence_category_links (id, raw_evidence_id, category_id)
SELECT link_id, $1, category_id
FROM unnest($2::text[], $3::text[]) AS link(link_id, category_id)`, rawEvidenceID, linkIDs, categoryIDStrings(categoryIDs))
	if err != nil {
		return fmt.Errorf("insert Raw Evidence Category links: %w", err)
	}
	return nil
}

func categoryIDStrings(categoryIDs []evidencebiz.CategoryID) []string {
	result := make([]string, len(categoryIDs))
	for index, categoryID := range categoryIDs {
		result[index] = string(categoryID)
	}
	return result
}

func (t *transaction) EvidencesByRawEvidence(ctx context.Context, rawEvidenceID string) ([]evidencebiz.StoredEvidence, error) {
	rows, err := t.tx.QueryContext(ctx, evidenceSelect+` WHERE raw_evidence_id = $1 ORDER BY id`, rawEvidenceID)
	if err != nil {
		return nil, fmt.Errorf("read Evidence set: %w", err)
	}
	defer rows.Close()
	records, err := scanEvidences(rows)
	if err != nil {
		return nil, err
	}
	if err := validateStoredEvidenceSet(rawEvidenceID, records); err != nil {
		return nil, fmt.Errorf("read Evidence set invariant: %w", err)
	}
	return records, nil
}

func (t *transaction) EvidencesByIDs(ctx context.Context, ids []string) ([]evidencebiz.StoredEvidence, error) {
	rows, err := t.tx.QueryContext(ctx, evidenceSelect+` WHERE id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return nil, fmt.Errorf("read Evidence identities: %w", err)
	}
	defer rows.Close()
	records, err := scanEvidences(rows)
	if err != nil {
		return nil, err
	}
	if err := validateStoredEvidenceIdentities(ids, records); err != nil {
		return nil, fmt.Errorf("read Evidence identities invariant: %w", err)
	}
	return records, nil
}

const evidenceSelect = `
SELECT id, raw_evidence_id, is_split, summary, array_to_json(keywords), semantic
FROM evidences`

type evidenceRows interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func scanEvidences(rows evidenceRows) ([]evidencebiz.StoredEvidence, error) {
	result := make([]evidencebiz.StoredEvidence, 0)
	for rows.Next() {
		var record evidencebiz.StoredEvidence
		var semanticJSON []byte
		var keywordsJSON []byte
		if err := rows.Scan(
			&record.ID, &record.RawEvidenceID, &record.IsSplit, &record.Summary, &keywordsJSON, &semanticJSON,
		); err != nil {
			return nil, fmt.Errorf("scan Evidence: %w", err)
		}
		if err := decodeStoredSemantic(semanticJSON, &record.Semantic); err != nil {
			return nil, fmt.Errorf("decode Evidence semantic: %w", err)
		}
		if err := json.Unmarshal(keywordsJSON, &record.Keywords); err != nil || record.Keywords == nil {
			return nil, errors.New("decode Evidence keywords: value is not an array")
		}
		if err := validateStoredEvidence(&record); err != nil {
			return nil, fmt.Errorf("read Evidence row invariant: %w", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Evidences: %w", err)
	}
	return result, nil
}

func (t *transaction) InsertEvidence(ctx context.Context, record evidencebiz.StoredEvidence) error {
	semanticJSON, err := json.Marshal(record.Semantic)
	if err != nil {
		return fmt.Errorf("encode Evidence semantic: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `
INSERT INTO evidences (
    id, raw_evidence_id, is_split, summary, keywords, semantic
) VALUES ($1,$2,$3,$4,$5,$6)`,
		record.ID, record.RawEvidenceID, record.IsSplit, record.Summary, record.Keywords, semanticJSON,
	)
	if err != nil {
		return fmt.Errorf("insert Evidence: %w", err)
	}
	return nil
}

func decodeStoredSemantic(value []byte, target *evidencebiz.Semantic) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(value, &fields); err != nil {
		return err
	}
	required := [...]string{"actors", "action", "objects", "stage", "modality", "time", "jurisdictions", "reason", "method", "metrics", "attribution"}
	if len(fields) != len(required) {
		return errors.New("semantic must contain exactly the business proposition fields")
	}
	for _, field := range required {
		if _, ok := fields[field]; !ok {
			return fmt.Errorf("semantic field %s is missing", field)
		}
	}
	if err := json.Unmarshal(value, target); err != nil {
		return err
	}
	return nil
}

var _ evidencebiz.Transaction = (*transaction)(nil)
