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

const summaryColumns = `id, publisher_report_id, report ->> 'generated_at',
       report ? 'geopolitics', report ? 'macroeconomics',
       jsonb_array_length(report -> 'industry_chains'), published_at`

func (s Store) ListReports(ctx context.Context, filter reportbiz.ListFilter) (reportbiz.StorePage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+summaryColumns+` FROM reports
WHERE ($1::timestamptz IS NULL OR published_at >= $1)
  AND ($2::timestamptz IS NULL OR published_at < $2)
  AND ($3::timestamptz IS NULL OR published_at < $3 OR (published_at = $3 AND id > $4))
ORDER BY published_at DESC, id ASC
LIMIT $5`, nullableTime(filter.PublishedFrom), nullableTime(filter.PublishedTo), nullableTime(filter.CursorPublishedAt), filter.CursorID, filter.Limit+1)
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
	return scanRecord(s.db.QueryRowContext(ctx, `SELECT id, publisher_report_id, content_hash, report, published_at FROM reports WHERE id = $1`, reportID))
}

func (s Store) GetHome(ctx context.Context, reportID string) (reportbiz.Home, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+summaryColumns+`, report -> 'geopolitics', report -> 'macroeconomics' FROM reports WHERE id = $1`, reportID)
	var summary reportbiz.Summary
	var generatedAt string
	var geopoliticsJSON, macroeconomicsJSON []byte
	if err := row.Scan(&summary.ID, &summary.PublisherReportID, &generatedAt, &summary.HasGeopolitics,
		&summary.HasMacroeconomics, &summary.IndustryChainCount, &summary.PublishedAt,
		&geopoliticsJSON, &macroeconomicsJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.Home{}, reportbiz.ErrReportNotFound
		}
		return reportbiz.Home{}, fmt.Errorf("read Report home %q: %w", reportID, err)
	}
	if err := finishSummary(&summary, generatedAt); err != nil {
		return reportbiz.Home{}, err
	}
	tokens, err := s.scopeTokens(ctx, reportID)
	if err != nil {
		return reportbiz.Home{}, err
	}
	home := reportbiz.Home{Report: summary}
	if summary.HasGeopolitics {
		layer, err := decodeLayer(geopoliticsJSON)
		if err != nil {
			return reportbiz.Home{}, persistedInvariant("Report home", "geopolitics", err.Error())
		}
		home.Geopolitics = &reportbiz.LayerSnapshot{Key: "geopolitics", Title: layer.Title, Summary: projectLayerSummary(layer.Summary, tokens["geopolitics/summary"])}
	}
	if summary.HasMacroeconomics {
		layer, err := decodeLayer(macroeconomicsJSON)
		if err != nil {
			return reportbiz.Home{}, persistedInvariant("Report home", "macroeconomics", err.Error())
		}
		home.Macroeconomics = &reportbiz.LayerSnapshot{Key: "macroeconomics", Title: layer.Title, Summary: projectLayerSummary(layer.Summary, tokens["macroeconomics/summary"])}
	}
	return home, nil
}

func (s Store) GetLayer(ctx context.Context, reportID, layerKey string) (reportbiz.Summary, reportbiz.LayerProjection, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+summaryColumns+`, report -> $2 FROM reports WHERE id = $1`, reportID, layerKey)
	var summary reportbiz.Summary
	var generatedAt string
	var layerJSON []byte
	if err := row.Scan(&summary.ID, &summary.PublisherReportID, &generatedAt, &summary.HasGeopolitics,
		&summary.HasMacroeconomics, &summary.IndustryChainCount, &summary.PublishedAt, &layerJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.Summary{}, reportbiz.LayerProjection{}, reportbiz.ErrReportNotFound
		}
		return reportbiz.Summary{}, reportbiz.LayerProjection{}, fmt.Errorf("read Report layer %q: %w", layerKey, err)
	}
	if isNullJSON(layerJSON) {
		return reportbiz.Summary{}, reportbiz.LayerProjection{}, reportbiz.ErrLayerNotFound
	}
	if err := finishSummary(&summary, generatedAt); err != nil {
		return reportbiz.Summary{}, reportbiz.LayerProjection{}, err
	}
	layer, err := decodeLayer(layerJSON)
	if err != nil {
		return reportbiz.Summary{}, reportbiz.LayerProjection{}, persistedInvariant("Report layer", layerKey, err.Error())
	}
	tokens, err := s.scopeTokens(ctx, reportID)
	if err != nil {
		return reportbiz.Summary{}, reportbiz.LayerProjection{}, err
	}
	return summary, projectLayer(layerKey, layer, tokens), nil
}

func (s Store) ListIndustryChains(ctx context.Context, filter reportbiz.IndustryChainListFilter) (reportbiz.IndustryChainStorePage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT chains.ordinality, chains.chain
FROM reports AS reports
CROSS JOIN LATERAL jsonb_array_elements(reports.report -> 'industry_chains') WITH ORDINALITY AS chains(chain, ordinality)
WHERE reports.id = $1 AND chains.ordinality > $2
ORDER BY chains.ordinality ASC
LIMIT $3`, filter.ReportID, filter.AfterOrdinal, filter.Limit+1)
	if err != nil {
		return reportbiz.IndustryChainStorePage{}, fmt.Errorf("query Report industry-chain summaries: %w", err)
	}
	defer rows.Close()
	type storedChain struct {
		ordinal int
		chain   reportbiz.IndustryChain
	}
	stored := make([]storedChain, 0, filter.Limit+1)
	for rows.Next() {
		var ordinal int
		var payload []byte
		if err := rows.Scan(&ordinal, &payload); err != nil {
			return reportbiz.IndustryChainStorePage{}, fmt.Errorf("scan Report industry-chain summary: %w", err)
		}
		var chain reportbiz.IndustryChain
		if err := decodeStoredJSON(payload, &chain); err != nil {
			return reportbiz.IndustryChainStorePage{}, persistedInvariant("Report industry-chain summary", "item", err.Error())
		}
		stored = append(stored, storedChain{ordinal: ordinal, chain: chain})
	}
	if err := rows.Err(); err != nil {
		return reportbiz.IndustryChainStorePage{}, fmt.Errorf("iterate Report industry-chain summaries: %w", err)
	}
	if len(stored) == 0 {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM reports WHERE id = $1)`, filter.ReportID).Scan(&exists); err != nil {
			return reportbiz.IndustryChainStorePage{}, fmt.Errorf("check Report existence: %w", err)
		}
		if !exists {
			return reportbiz.IndustryChainStorePage{}, reportbiz.ErrReportNotFound
		}
	}
	tokens, err := s.scopeTokens(ctx, filter.ReportID)
	if err != nil {
		return reportbiz.IndustryChainStorePage{}, err
	}
	items := make([]reportbiz.IndustryChainSummary, len(stored))
	for index, item := range stored {
		items[index] = projectChainSummary(item.ordinal, item.chain, tokens)
	}
	page := reportbiz.IndustryChainStorePage{Items: items}
	if len(page.Items) > filter.Limit {
		page.HasMore = true
		page.Items = page.Items[:filter.Limit]
	}
	return page, nil
}

func (s Store) GetIndustryChain(ctx context.Context, reportID, chainKey string) (reportbiz.Summary, reportbiz.IndustryChainProjection, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+summaryColumns+`, (
    SELECT chain FROM jsonb_array_elements(report -> 'industry_chains') AS chains(chain)
    WHERE chain ->> 'local_key' = $2 LIMIT 1
) FROM reports WHERE id = $1`, reportID, chainKey)
	var summary reportbiz.Summary
	var generatedAt string
	var chainJSON []byte
	if err := row.Scan(&summary.ID, &summary.PublisherReportID, &generatedAt, &summary.HasGeopolitics,
		&summary.HasMacroeconomics, &summary.IndustryChainCount, &summary.PublishedAt, &chainJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.Summary{}, reportbiz.IndustryChainProjection{}, reportbiz.ErrReportNotFound
		}
		return reportbiz.Summary{}, reportbiz.IndustryChainProjection{}, fmt.Errorf("read Report industry chain %q: %w", chainKey, err)
	}
	if isNullJSON(chainJSON) {
		return reportbiz.Summary{}, reportbiz.IndustryChainProjection{}, reportbiz.ErrChainNotFound
	}
	if err := finishSummary(&summary, generatedAt); err != nil {
		return reportbiz.Summary{}, reportbiz.IndustryChainProjection{}, err
	}
	var chain reportbiz.IndustryChain
	if err := decodeStoredJSON(chainJSON, &chain); err != nil {
		return reportbiz.Summary{}, reportbiz.IndustryChainProjection{}, persistedInvariant("Report industry chain", chainKey, err.Error())
	}
	tokens, err := s.scopeTokens(ctx, reportID)
	if err != nil {
		return reportbiz.Summary{}, reportbiz.IndustryChainProjection{}, err
	}
	return summary, projectChain(chain, tokens), nil
}

func (s Store) ListEvidence(ctx context.Context, reportID, scopeToken string) ([]reportbiz.Evidence, error) {
	var scopePath string
	err := s.db.QueryRowContext(ctx, `SELECT scope_path FROM report_evidence_links WHERE report_id = $1 AND id = $2`, reportID, scopeToken).Scan(&scopePath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, reportbiz.ErrEvidenceScopeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resolve Report Evidence scope: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT raw.published_at, evidence.summary, to_json(evidence.keywords)
FROM report_evidence_links AS link
JOIN evidences AS evidence ON evidence.id = link.evidence_id
JOIN raw_evidences AS raw ON raw.id = evidence.raw_evidence_id
WHERE link.report_id = $1 AND link.scope_path = $2
ORDER BY link.position ASC`, reportID, scopePath)
	if err != nil {
		return nil, fmt.Errorf("query Report Evidence: %w", err)
	}
	defer rows.Close()
	result := make([]reportbiz.Evidence, 0)
	for rows.Next() {
		var item reportbiz.Evidence
		var publishedAt sql.NullTime
		var keywordsJSON []byte
		if err := rows.Scan(&publishedAt, &item.Summary, &keywordsJSON); err != nil {
			return nil, fmt.Errorf("scan Report Evidence: %w", err)
		}
		if publishedAt.Valid {
			value := publishedAt.Time.UTC()
			item.PublishedAt = &value
		}
		if err := json.Unmarshal(keywordsJSON, &item.Keywords); err != nil || item.Keywords == nil {
			return nil, persistedInvariant("Report Evidence", "keywords", "value is not an array")
		}
		if strings.TrimSpace(item.Summary) == "" {
			return nil, persistedInvariant("Report Evidence", "summary", "value is blank")
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Report Evidence: %w", err)
	}
	return result, nil
}

func (s Store) scopeTokens(ctx context.Context, reportID string) (map[string]*string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT ON (scope_path) scope_path, id
FROM report_evidence_links WHERE report_id = $1 ORDER BY scope_path, position, id`, reportID)
	if err != nil {
		return nil, fmt.Errorf("query Report Evidence scope tokens: %w", err)
	}
	defer rows.Close()
	result := map[string]*string{}
	for rows.Next() {
		var path, token string
		if err := rows.Scan(&path, &token); err != nil {
			return nil, fmt.Errorf("scan Report Evidence scope token: %w", err)
		}
		value := token
		result[path] = &value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Report Evidence scope tokens: %w", err)
	}
	return result, nil
}

func projectLayerSummary(summary reportbiz.LayerSummary, token *string) reportbiz.LayerSummaryProjection {
	return reportbiz.LayerSummaryProjection{
		Conclusion: summary.Conclusion, Result: summary.Result, Confidence: summary.Confidence,
		TimeWindow: summary.TimeWindow, DownwardTransmission: summary.DownwardTransmission,
		Uncertainty: summary.Uncertainty, EvidenceScopeToken: token,
	}
}

func projectLayer(key string, layer reportbiz.Layer, tokens map[string]*string) reportbiz.LayerProjection {
	result := reportbiz.LayerProjection{Key: key, Title: layer.Title, Summary: projectLayerSummary(layer.Summary, tokens[key+"/summary"])}
	result.AffectedAnchors = make([]reportbiz.AnchorProjection, len(layer.Detail.AffectedAnchors))
	for index, anchor := range layer.Detail.AffectedAnchors {
		result.AffectedAnchors[index] = reportbiz.AnchorProjection{
			LocalKey: anchor.LocalKey, Name: anchor.Name, CurrentState: anchor.CurrentState, Result: anchor.Result,
			ConclusionBasis: anchor.ConclusionBasis, ValidationStatus: anchor.ValidationStatus,
			TransmissionLogic: anchor.TransmissionLogic, TimeWindow: anchor.TimeWindow, Confidence: anchor.Confidence,
			EvidenceScopeToken: tokens[key+"/detail/affected_anchors/"+anchor.LocalKey],
		}
	}
	result.ReasoningSteps = make([]reportbiz.ReasoningStepProjection, len(layer.Detail.ReasoningSteps))
	for index, step := range layer.Detail.ReasoningSteps {
		result.ReasoningSteps[index] = reportbiz.ReasoningStepProjection{
			Input: step.Input, Mechanism: step.Mechanism, Output: step.Output, ReasoningType: step.ReasoningType,
			Confidence: step.Confidence, EvidenceScopeToken: tokens[fmt.Sprintf("%s/detail/reasoning_steps/%d", key, index+1)],
		}
	}
	return result
}

func projectChainSummary(ordinal int, chain reportbiz.IndustryChain, tokens map[string]*string) reportbiz.IndustryChainSummary {
	prefix := "industry_chains/" + chain.LocalKey
	result := reportbiz.IndustryChainSummary{
		Ordinal: ordinal, LocalKey: chain.LocalKey, Name: chain.Name, Conclusion: chain.Summary.Conclusion,
		Status: chain.Summary.Status, Result: chain.Summary.Result, Confidence: chain.Summary.Confidence,
		TimeWindow: chain.Summary.TimeWindow, EvidenceScopeToken: tokens[prefix+"/summary"],
		ImpactItems: make([]reportbiz.IndustryChainImpactSummary, len(chain.Detail.AffectedNodes)),
	}
	for index, node := range chain.Detail.AffectedNodes {
		result.ImpactItems[index] = reportbiz.IndustryChainImpactSummary{
			LocalKey: node.LocalKey, Name: node.Name, Result: node.Result, ConclusionBasis: node.ConclusionBasis,
			ValidationStatus: node.ValidationStatus, Confidence: node.Confidence, TimeWindow: node.TimeWindow,
			EvidenceScopeToken: tokens[prefix+"/detail/affected_nodes/"+node.LocalKey],
		}
	}
	return result
}

func projectChain(chain reportbiz.IndustryChain, tokens map[string]*string) reportbiz.IndustryChainProjection {
	prefix := "industry_chains/" + chain.LocalKey
	result := reportbiz.IndustryChainProjection{
		LocalKey: chain.LocalKey, Name: chain.Name, Conclusion: chain.Summary.Conclusion, Status: chain.Summary.Status,
		Result: chain.Summary.Result, Confidence: chain.Summary.Confidence, TimeWindow: chain.Summary.TimeWindow,
		Path: chain.Summary.Path, Graph: chain.Summary.Graph, CounterevidenceAndGap: chain.Summary.CounterevidenceAndGap,
		StopCondition: chain.Summary.StopCondition, EvidenceScopeToken: tokens[prefix+"/summary"],
		AffectedNodes: make([]reportbiz.IndustryChainNodeProjection, len(chain.Detail.AffectedNodes)),
	}
	for index, node := range chain.Detail.AffectedNodes {
		result.AffectedNodes[index] = reportbiz.IndustryChainNodeProjection{
			LocalKey: node.LocalKey, NodeLocalKey: node.NodeLocalKey, Name: node.Name, Impact: node.Impact,
			Result: node.Result, ConclusionBasis: node.ConclusionBasis, ValidationStatus: node.ValidationStatus,
			TransmissionLogic: node.TransmissionLogic, TimeWindow: node.TimeWindow, Confidence: node.Confidence,
			EvidenceScopeToken: tokens[prefix+"/detail/affected_nodes/"+node.LocalKey],
		}
	}
	return result
}

type scanner interface{ Scan(...any) error }

func scanRecord(row scanner) (reportbiz.Record, error) {
	var record reportbiz.Record
	var reportJSON []byte
	if err := row.Scan(&record.ID, &record.PublisherReportID, &record.ContentHash, &reportJSON, &record.PublishedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.Record{}, reportbiz.ErrReportNotFound
		}
		return reportbiz.Record{}, fmt.Errorf("read Report: %w", err)
	}
	if err := decodeStoredJSON(reportJSON, &record.Report); err != nil {
		return reportbiz.Record{}, persistedInvariant("Report", "report", err.Error())
	}
	if err := reportbiz.ValidateReport(record.Report); err != nil {
		return reportbiz.Record{}, fmt.Errorf("read Report invariant: %w", err)
	}
	wantHash, err := reportbiz.ContentHash(record.Report)
	if err != nil {
		return reportbiz.Record{}, fmt.Errorf("recompute Report content hash: %w", err)
	}
	if !coreid.Is(record.ID, coreid.Report) || strings.TrimSpace(record.PublisherReportID) == "" ||
		record.PublisherReportID != strings.TrimSpace(record.PublisherReportID) || !contentHashPattern.MatchString(record.ContentHash) ||
		record.ContentHash != wantHash || record.PublishedAt.IsZero() {
		return reportbiz.Record{}, persistedInvariant("Report", "row", "stored identity, hash or publication time is invalid")
	}
	record.PublishedAt = record.PublishedAt.UTC()
	return record, nil
}

func scanSummary(row scanner) (reportbiz.Summary, error) {
	var result reportbiz.Summary
	var generatedAt string
	if err := row.Scan(&result.ID, &result.PublisherReportID, &generatedAt, &result.HasGeopolitics,
		&result.HasMacroeconomics, &result.IndustryChainCount, &result.PublishedAt); err != nil {
		return reportbiz.Summary{}, fmt.Errorf("scan Report summary: %w", err)
	}
	if err := finishSummary(&result, generatedAt); err != nil {
		return reportbiz.Summary{}, err
	}
	return result, nil
}

func finishSummary(result *reportbiz.Summary, generatedAt string) error {
	parsed, err := time.Parse(time.RFC3339Nano, generatedAt)
	if err != nil || parsed.IsZero() {
		return persistedInvariant("Report summary", "generated_at", "value is not RFC3339")
	}
	result.GeneratedAt = parsed
	result.PublishedAt = result.PublishedAt.UTC()
	if !coreid.Is(result.ID, coreid.Report) || strings.TrimSpace(result.PublisherReportID) == "" ||
		result.PublisherReportID != strings.TrimSpace(result.PublisherReportID) || result.IndustryChainCount < 1 || result.PublishedAt.IsZero() {
		return persistedInvariant("Report summary", "row", "stored metadata is invalid")
	}
	return nil
}

func decodeLayer(payload []byte) (reportbiz.Layer, error) {
	var layer reportbiz.Layer
	if err := decodeStoredJSON(payload, &layer); err != nil {
		return reportbiz.Layer{}, err
	}
	return layer, nil
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

func isNullJSON(payload []byte) bool {
	return len(payload) == 0 || bytes.Equal(bytes.TrimSpace(payload), []byte("null"))
}
func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

var contentHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type persistedInvariantError struct{ resource, field, reason string }

func (e *persistedInvariantError) Error() string {
	return fmt.Sprintf("persisted %s %s invariant: %s", e.resource, e.field, e.reason)
}
func persistedInvariant(resource, field, reason string) error {
	return &persistedInvariantError{resource: resource, field: field, reason: reason}
}

var _ reportbiz.Store = Store{}
