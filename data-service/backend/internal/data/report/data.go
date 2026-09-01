package report

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (Store, error) {
	if db == nil {
		return Store{}, errors.New("Report database is required")
	}
	return Store{db: db}, nil
}

const summarySelect = `SELECT id, source_report_id,
       content ->> 'report_type', content ->> 'title', content ->> 'status',
       (content ->> 'simulation')::boolean, content ->> 'generated_at', content ->> 'timezone',
       content -> 'published_layers', content -> 'statistics', published_at
FROM reports`

func (s Store) ListReports(ctx context.Context, filter reportbiz.ListFilter) (reportbiz.StorePage, error) {
	rows, err := s.db.QueryContext(ctx, summarySelect+`
WHERE ($1::timestamptz IS NULL OR published_at >= $1)
  AND ($2::timestamptz IS NULL OR published_at < $2)
  AND ($3::timestamptz IS NULL OR published_at < $3 OR (published_at = $3 AND id > $4))
ORDER BY published_at DESC, id ASC
LIMIT $5`, nullableTime(filter.PublishedFrom), nullableTime(filter.PublishedTo),
		nullableTime(filter.CursorPublishedAt), filter.CursorID, filter.Limit+1)
	if err != nil {
		return reportbiz.StorePage{}, fmt.Errorf("query Reports: %w", err)
	}
	defer rows.Close()
	items := make([]reportbiz.Summary, 0, filter.Limit+1)
	for rows.Next() {
		item, err := scanSummary(rows)
		if err != nil {
			return reportbiz.StorePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return reportbiz.StorePage{}, fmt.Errorf("iterate Reports: %w", err)
	}
	page := reportbiz.StorePage{Items: items}
	if len(page.Items) > filter.Limit {
		page.HasMore = true
		page.Items = page.Items[:filter.Limit]
	}
	return page, nil
}

func (s Store) GetReport(ctx context.Context, reportID string) (reportbiz.Record, error) {
	return scanRecord(s.db.QueryRowContext(ctx, `SELECT id, source_report_id, contract_version,
       content_hash, content, published_at
FROM reports WHERE id = $1`, reportID))
}

func (s Store) GetHome(ctx context.Context, reportID string) (reportbiz.Home, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, source_report_id,
       content ->> 'report_type', content ->> 'title', content ->> 'status',
       (content ->> 'simulation')::boolean, content ->> 'generated_at', content ->> 'timezone',
       content -> 'published_layers', content -> 'statistics', published_at,
       content -> 'report_cards', content -> 'company',
       jsonb_build_object(
           'anchor', COALESCE((
               SELECT jsonb_object_agg(anchor ->> 'key', jsonb_array_length(anchor -> 'evidence_refs'))
               FROM (
                   SELECT jsonb_array_elements(content #> '{geopolitics,anchors}') AS anchor
                   UNION ALL
                   SELECT jsonb_array_elements(content #> '{macroeconomics,anchors}') AS anchor
               ) AS anchors
           ), '{}'::jsonb),
           'industry_chain_node', COALESCE((
               SELECT jsonb_object_agg(node ->> 'key', jsonb_array_length(node -> 'evidence_refs'))
               FROM jsonb_array_elements(content -> 'industry_chains') AS chains(chain)
               CROSS JOIN LATERAL jsonb_array_elements(chain -> 'nodes') AS nodes(node)
           ), '{}'::jsonb)
       )
FROM reports WHERE id = $1`, reportID)
	var summary reportbiz.Summary
	var generatedAt string
	var layersJSON, statisticsJSON, cardsJSON, companyJSON, countsJSON []byte
	if err := row.Scan(&summary.ID, &summary.SourceReportID, &summary.ReportType, &summary.Title,
		&summary.Status, &summary.Simulation, &generatedAt, &summary.Timezone, &layersJSON,
		&statisticsJSON, &summary.PublishedAt, &cardsJSON, &companyJSON, &countsJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.Home{}, reportbiz.ErrReportNotFound
		}
		return reportbiz.Home{}, fmt.Errorf("read Report home %q: %w", reportID, err)
	}
	if err := decodeSummaryFragments(&summary, generatedAt, layersJSON, statisticsJSON); err != nil {
		return reportbiz.Home{}, fmt.Errorf("read Report home invariant: %w", err)
	}
	var cards []reportbiz.ReportCard
	if err := decodeStoredJSON(cardsJSON, &cards); err != nil || cards == nil {
		return reportbiz.Home{}, persistedInvariant("Report home", "report_cards", "value is not an exact array")
	}
	var company reportbiz.CompanyBoundary
	if err := decodeStoredJSON(companyJSON, &company); err != nil {
		return reportbiz.Home{}, persistedInvariant("Report home", "company", "value is not an exact object")
	}
	var storedCounts map[string]map[string]int
	if err := decodeStoredJSON(countsJSON, &storedCounts); err != nil || storedCounts == nil {
		return reportbiz.Home{}, persistedInvariant("Report home", "evidence_counts", "value is not an exact object")
	}
	counts := make(map[reportbiz.TargetReference]int)
	for key, count := range storedCounts["anchor"] {
		counts[reportbiz.TargetReference{Type: reportbiz.TargetAnchor, Key: key}] = count
	}
	for key, count := range storedCounts["industry_chain_node"] {
		counts[reportbiz.TargetReference{Type: reportbiz.TargetIndustryChainNode, Key: key}] = count
	}
	return reportbiz.Home{Report: summary, ReportCards: cards, Company: company, EvidenceCounts: counts}, nil
}

func (s Store) GetLayer(ctx context.Context, reportID, layerKey string) (reportbiz.Summary, reportbiz.Layer, []reportbiz.IndustryChainSummary, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, source_report_id,
       content ->> 'report_type', content ->> 'title', content ->> 'status',
       (content ->> 'simulation')::boolean, content ->> 'generated_at', content ->> 'timezone',
       content -> 'published_layers', content -> 'statistics', published_at,
       content -> $2,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'key', chain -> 'key',
               'display_order', chain -> 'display_order',
               'name', chain -> 'name',
               'conclusion', chain -> 'conclusion',
               'status', chain -> 'status',
               'result', chain -> 'result',
               'confidence', chain -> 'confidence',
               'time_window', chain -> 'time_window',
               'evidence_count', jsonb_array_length(chain -> 'evidence_refs')
           ) ORDER BY related.ordinality)
           FROM jsonb_array_elements_text((content -> $2) -> 'related_chain_keys') WITH ORDINALITY AS related(key, ordinality)
           JOIN LATERAL (
               SELECT candidate AS chain
               FROM jsonb_array_elements(content -> 'industry_chains') AS chains(candidate)
               WHERE candidate ->> 'key' = related.key
           ) AS selected ON true
       ), '[]'::jsonb)
FROM reports WHERE id = $1`, reportID, layerKey)
	var summary reportbiz.Summary
	var generatedAt string
	var layersJSON, statisticsJSON, layerJSON, relatedJSON []byte
	if err := row.Scan(&summary.ID, &summary.SourceReportID, &summary.ReportType, &summary.Title,
		&summary.Status, &summary.Simulation, &generatedAt, &summary.Timezone, &layersJSON,
		&statisticsJSON, &summary.PublishedAt, &layerJSON, &relatedJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.Summary{}, reportbiz.Layer{}, nil, reportbiz.ErrReportNotFound
		}
		return reportbiz.Summary{}, reportbiz.Layer{}, nil, fmt.Errorf("read Report layer %q: %w", layerKey, err)
	}
	if err := decodeSummaryFragments(&summary, generatedAt, layersJSON, statisticsJSON); err != nil {
		return reportbiz.Summary{}, reportbiz.Layer{}, nil, fmt.Errorf("read Report layer summary invariant: %w", err)
	}
	var layer reportbiz.Layer
	if err := decodeStoredJSON(layerJSON, &layer); err != nil {
		return reportbiz.Summary{}, reportbiz.Layer{}, nil, persistedInvariant("Report layer", layerKey, "value is not an exact object")
	}
	if err := reportbiz.ValidateLayerSnapshot(layer); err != nil {
		return reportbiz.Summary{}, reportbiz.Layer{}, nil, fmt.Errorf("read Report layer invariant: %w", err)
	}
	var related []reportbiz.IndustryChainSummary
	if err := decodeStoredJSON(relatedJSON, &related); err != nil || related == nil {
		return reportbiz.Summary{}, reportbiz.Layer{}, nil, persistedInvariant("Report layer", "related_industry_chains", "value is not an exact array")
	}
	if len(related) != len(layer.RelatedChainKeys) {
		return reportbiz.Summary{}, reportbiz.Layer{}, nil, persistedInvariant("Report layer", "related_industry_chains", "does not close over related_chain_keys")
	}
	for index, item := range related {
		if item.Key != layer.RelatedChainKeys[index] || item.DisplayOrder < 1 || item.EvidenceCount < 0 ||
			strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Conclusion) == "" || strings.TrimSpace(item.Status) == "" ||
			strings.TrimSpace(item.TimeWindow) == "" {
			return reportbiz.Summary{}, reportbiz.Layer{}, nil, persistedInvariant("Report layer", "related_industry_chains", "contains an invalid summary")
		}
	}
	return summary, layer, related, nil
}

func (s Store) GetIndustryChain(ctx context.Context, reportID, chainKey string) (reportbiz.Summary, reportbiz.IndustryChain, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, source_report_id,
       content ->> 'report_type', content ->> 'title', content ->> 'status',
       (content ->> 'simulation')::boolean, content ->> 'generated_at', content ->> 'timezone',
       content -> 'published_layers', content -> 'statistics', published_at,
       (SELECT candidate FROM jsonb_array_elements(content -> 'industry_chains') AS chains(candidate)
        WHERE candidate ->> 'key' = $2)
FROM reports WHERE id = $1`, reportID, chainKey)
	var summary reportbiz.Summary
	var generatedAt string
	var layersJSON, statisticsJSON, chainJSON []byte
	if err := row.Scan(&summary.ID, &summary.SourceReportID, &summary.ReportType, &summary.Title,
		&summary.Status, &summary.Simulation, &generatedAt, &summary.Timezone, &layersJSON,
		&statisticsJSON, &summary.PublishedAt, &chainJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.Summary{}, reportbiz.IndustryChain{}, reportbiz.ErrReportNotFound
		}
		return reportbiz.Summary{}, reportbiz.IndustryChain{}, fmt.Errorf("read Report industry chain %q: %w", chainKey, err)
	}
	if len(chainJSON) == 0 {
		return reportbiz.Summary{}, reportbiz.IndustryChain{}, reportbiz.ErrChainNotFound
	}
	if err := decodeSummaryFragments(&summary, generatedAt, layersJSON, statisticsJSON); err != nil {
		return reportbiz.Summary{}, reportbiz.IndustryChain{}, fmt.Errorf("read Report industry chain summary invariant: %w", err)
	}
	var chain reportbiz.IndustryChain
	if err := decodeStoredJSON(chainJSON, &chain); err != nil {
		return reportbiz.Summary{}, reportbiz.IndustryChain{}, persistedInvariant("Report industry chain", chainKey, "value is not an exact object")
	}
	if err := reportbiz.ValidateIndustryChainSnapshot(chain); err != nil {
		return reportbiz.Summary{}, reportbiz.IndustryChain{}, fmt.Errorf("read Report industry chain invariant: %w", err)
	}
	return summary, chain, nil
}

func (s Store) ReportScopeExists(ctx context.Context, reportID string, scopeType reportbiz.ScopeType, scopeKey string) (bool, bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT CASE $2
    WHEN 'report_card' THEN EXISTS (
        SELECT 1 FROM jsonb_array_elements(content -> 'report_cards') AS items(item) WHERE item ->> 'key' = $3)
    WHEN 'layer' THEN $3 IN ('geopolitics', 'macroeconomics') AND content ? $3
    WHEN 'anchor' THEN EXISTS (
        SELECT 1 FROM (
            SELECT jsonb_array_elements(content #> '{geopolitics,anchors}') AS item
            UNION ALL SELECT jsonb_array_elements(content #> '{macroeconomics,anchors}') AS item
        ) AS items WHERE item ->> 'key' = $3)
    WHEN 'reasoning_step' THEN EXISTS (
        SELECT 1 FROM (
            SELECT jsonb_array_elements(content #> '{geopolitics,reasoning_steps}') AS item
            UNION ALL SELECT jsonb_array_elements(content #> '{macroeconomics,reasoning_steps}') AS item
        ) AS items WHERE item ->> 'key' = $3)
    WHEN 'transmission_path' THEN EXISTS (
        SELECT 1 FROM (
            SELECT jsonb_array_elements(content #> '{geopolitics,downward_transmission,published_paths}') AS item
            UNION ALL SELECT jsonb_array_elements(content #> '{macroeconomics,downward_transmission,published_paths}') AS item
        ) AS items WHERE item ->> 'key' = $3)
    WHEN 'candidate_mechanism' THEN EXISTS (
        SELECT 1 FROM (
            SELECT jsonb_array_elements(content #> '{geopolitics,downward_transmission,candidate_mechanisms}') AS item
            UNION ALL SELECT jsonb_array_elements(content #> '{macroeconomics,downward_transmission,candidate_mechanisms}') AS item
        ) AS items WHERE item ->> 'key' = $3)
    WHEN 'industry_chain' THEN EXISTS (
        SELECT 1 FROM jsonb_array_elements(content -> 'industry_chains') AS items(item) WHERE item ->> 'key' = $3)
    WHEN 'industry_chain_node' THEN EXISTS (
        SELECT 1 FROM jsonb_array_elements(content -> 'industry_chains') AS chains(chain)
        CROSS JOIN LATERAL jsonb_array_elements(chain -> 'nodes') AS nodes(node) WHERE node ->> 'key' = $3)
    ELSE false END
FROM reports WHERE id = $1`, reportID, scopeType, scopeKey).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("read Report Evidence scope: %w", err)
	}
	return true, exists, nil
}

func (s Store) ListEvidence(ctx context.Context, reportID string, scopeType reportbiz.ScopeType, scopeKey string) ([]reportbiz.Evidence, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT link.evidence_id, link.role, link.display_order,
       raw.published_at, evidence.summary, to_json(evidence.keywords)
FROM report_evidence_links AS link
JOIN evidences AS evidence ON evidence.id = link.evidence_id
JOIN raw_evidences AS raw ON raw.id = evidence.raw_evidence_id
WHERE link.report_id = $1 AND link.scope_type = $2 AND link.scope_key = $3
ORDER BY link.display_order ASC`, reportID, scopeType, scopeKey)
	if err != nil {
		return nil, fmt.Errorf("query Report Evidence: %w", err)
	}
	defer rows.Close()
	result := make([]reportbiz.Evidence, 0)
	expectedOrder := 1
	for rows.Next() {
		var item reportbiz.Evidence
		var publishedAt sql.NullTime
		var keywordsJSON []byte
		if err := rows.Scan(&item.EvidenceID, &item.Role, &item.DisplayOrder, &publishedAt, &item.Summary, &keywordsJSON); err != nil {
			return nil, fmt.Errorf("scan Report Evidence: %w", err)
		}
		if publishedAt.Valid {
			value := publishedAt.Time.UTC()
			item.PublishedAt = &value
		}
		if err := json.Unmarshal(keywordsJSON, &item.Keywords); err != nil || item.Keywords == nil {
			return nil, persistedInvariant("Report Evidence", "keywords", "value is not an array")
		}
		if !coreid.Is(item.EvidenceID, coreid.Evidence) || strings.TrimSpace(item.Role) == "" ||
			item.Role != strings.TrimSpace(item.Role) || item.DisplayOrder != expectedOrder ||
			strings.TrimSpace(item.Summary) == "" {
			return nil, persistedInvariant("Report Evidence", "item", "stored relationship or Evidence is invalid")
		}
		result = append(result, item)
		expectedOrder++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Report Evidence: %w", err)
	}
	return result, nil
}

type scanner interface{ Scan(...any) error }

func scanRecord(row scanner) (reportbiz.Record, error) {
	var record reportbiz.Record
	var contentJSON []byte
	if err := row.Scan(&record.ID, &record.SourceReportID, &record.ContractVersion,
		&record.ContentHash, &contentJSON, &record.PublishedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.Record{}, reportbiz.ErrReportNotFound
		}
		return reportbiz.Record{}, fmt.Errorf("read Report: %w", err)
	}
	if err := decodeStoredJSON(contentJSON, &record.Content); err != nil {
		return reportbiz.Record{}, persistedInvariant("Report", "content", "value is not an exact report-publication.v1 object")
	}
	if err := reportbiz.ValidateContent(record.Content); err != nil {
		return reportbiz.Record{}, fmt.Errorf("read Report content invariant: %w", err)
	}
	wantHash, err := reportbiz.ContentHash(record.ContractVersion, record.Content)
	if err != nil {
		return reportbiz.Record{}, fmt.Errorf("recompute Report content hash: %w", err)
	}
	if !coreid.Is(record.ID, coreid.Report) || strings.TrimSpace(record.SourceReportID) == "" ||
		record.SourceReportID != strings.TrimSpace(record.SourceReportID) ||
		record.ContractVersion != reportbiz.ContractVersion || !contentHashPattern.MatchString(record.ContentHash) ||
		record.ContentHash != wantHash || record.PublishedAt.IsZero() {
		return reportbiz.Record{}, persistedInvariant("Report", "row", "stored identity, hash, version or publication time is invalid")
	}
	record.PublishedAt = record.PublishedAt.UTC()
	return record, nil
}

func scanSummary(row scanner) (reportbiz.Summary, error) {
	var result reportbiz.Summary
	var generatedAt string
	var layersJSON, statisticsJSON []byte
	if err := row.Scan(&result.ID, &result.SourceReportID, &result.ReportType, &result.Title,
		&result.Status, &result.Simulation, &generatedAt, &result.Timezone,
		&layersJSON, &statisticsJSON, &result.PublishedAt); err != nil {
		return reportbiz.Summary{}, fmt.Errorf("scan Report summary: %w", err)
	}
	if err := decodeSummaryFragments(&result, generatedAt, layersJSON, statisticsJSON); err != nil {
		return reportbiz.Summary{}, err
	}
	return result, nil
}

func decodeSummaryFragments(result *reportbiz.Summary, generatedAt string, layersJSON, statisticsJSON []byte) error {
	parsed, err := time.Parse(time.RFC3339Nano, generatedAt)
	if err != nil || parsed.IsZero() {
		return persistedInvariant("Report summary", "generated_at", "value is not RFC3339")
	}
	result.GeneratedAt = parsed
	result.PublishedAt = result.PublishedAt.UTC()
	if err := decodeStoredJSON(layersJSON, &result.PublishedLayers); err != nil || result.PublishedLayers == nil {
		return persistedInvariant("Report summary", "published_layers", "value is not an exact array")
	}
	if err := decodeStoredJSON(statisticsJSON, &result.Statistics); err != nil {
		return persistedInvariant("Report summary", "statistics", "value is not an exact object")
	}
	if !coreid.Is(result.ID, coreid.Report) || strings.TrimSpace(result.SourceReportID) == "" ||
		result.SourceReportID != strings.TrimSpace(result.SourceReportID) ||
		strings.TrimSpace(result.ReportType) == "" || strings.TrimSpace(result.Title) == "" ||
		strings.TrimSpace(result.Status) == "" || strings.TrimSpace(result.Timezone) == "" || result.PublishedAt.IsZero() {
		return persistedInvariant("Report summary", "row", "stored metadata is invalid")
	}
	return nil
}

func decodeStoredJSON(payload []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("JSON contains another value")
		}
		return err
	}
	return nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

var contentHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type persistedInvariantError struct {
	resource string
	field    string
	reason   string
}

func (e *persistedInvariantError) Error() string {
	return fmt.Sprintf("persisted %s %s invariant: %s", e.resource, e.field, e.reason)
}

func persistedInvariant(resource, field, reason string) error {
	return &persistedInvariantError{resource: resource, field: field, reason: reason}
}

var _ reportbiz.Store = Store{}
