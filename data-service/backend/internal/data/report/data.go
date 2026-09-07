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
       (report ? 'geopolitics' OR jsonb_array_length(COALESCE(report->'geopolitical_stories','[]'::jsonb))>0), (report ? 'macroeconomics' OR jsonb_array_length(COALESCE(report->'macroeconomic_stories','[]'::jsonb))>0),
       CASE WHEN report->>'schema_version'='report-publication/v3' THEN (SELECT COALESCE(sum(jsonb_array_length(u#>'{detail,industry_chains}')),0) FROM jsonb_array_elements(report->'concept_analyses') u) ELSE jsonb_array_length(report -> 'industry_chains') END, published_at, COALESCE(report->>'schema_version',''), COALESCE(report#>>'{analysis_window,start}',''), COALESCE(report#>>'{analysis_window,end}','')`

func (s Store) ListReports(ctx context.Context, filter reportbiz.ListFilter) (reportbiz.StorePage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+summaryColumns+` FROM reports
WHERE COALESCE(report->>'schema_version','')=$6 AND ($1::timestamptz IS NULL OR published_at >= $1)
  AND ($2::timestamptz IS NULL OR published_at < $2)
  AND ($3::timestamptz IS NULL OR published_at < $3 OR (published_at = $3 AND id > $4))
ORDER BY published_at DESC, id ASC
LIMIT $5`, nullableTime(filter.PublishedFrom), nullableTime(filter.PublishedTo), nullableTime(filter.CursorPublishedAt), filter.CursorID, filter.Limit+1, filter.SchemaVersion)
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
		&summary.HasMacroeconomics, &summary.IndustryChainCount, &summary.PublishedAt, &summary.SchemaVersion, &summary.AnalysisWindowStart, &summary.AnalysisWindowEnd,
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
	if !isNullJSON(geopoliticsJSON) {
		layer, err := decodeLayer(geopoliticsJSON)
		if err != nil {
			return reportbiz.Home{}, persistedInvariant("Report home", "geopolitics", err.Error())
		}
		home.Geopolitics = &reportbiz.LayerSnapshot{Key: "geopolitics", Title: layer.Title, Summary: projectLayerSummary(layer, tokens["geopolitics/evidence_refs"])}
	}
	if !isNullJSON(macroeconomicsJSON) {
		layer, err := decodeLayer(macroeconomicsJSON)
		if err != nil {
			return reportbiz.Home{}, persistedInvariant("Report home", "macroeconomics", err.Error())
		}
		home.Macroeconomics = &reportbiz.LayerSnapshot{Key: "macroeconomics", Title: layer.Title, Summary: projectLayerSummary(layer, tokens["macroeconomics/evidence_refs"])}
	}
	return home, nil
}

func (s Store) GetLayer(ctx context.Context, reportID, layerKey string) (reportbiz.Summary, reportbiz.LayerProjection, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+summaryColumns+`, report -> $2 FROM reports WHERE id = $1`, reportID, layerKey)
	var summary reportbiz.Summary
	var generatedAt string
	var layerJSON []byte
	if err := row.Scan(&summary.ID, &summary.PublisherReportID, &generatedAt, &summary.HasGeopolitics,
		&summary.HasMacroeconomics, &summary.IndustryChainCount, &summary.PublishedAt, &summary.SchemaVersion, &summary.AnalysisWindowStart, &summary.AnalysisWindowEnd, &layerJSON); err != nil {
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
		&summary.HasMacroeconomics, &summary.IndustryChainCount, &summary.PublishedAt, &summary.SchemaVersion, &summary.AnalysisWindowStart, &summary.AnalysisWindowEnd, &chainJSON); err != nil {
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

func projectLayerSummary(layer reportbiz.Layer, token *string) reportbiz.LayerSummaryProjection {
	return reportbiz.LayerSummaryProjection{
		Conclusion: layer.Conclusion, Result: layer.Result, Confidence: layer.Confidence,
		TimeWindow: layer.TimeWindow, DownwardTransmission: layer.DownwardTransmission,
		Uncertainty: layer.Uncertainty, EvidenceScopeToken: token,
	}
}

func projectLayer(key string, layer reportbiz.Layer, tokens map[string]*string) reportbiz.LayerProjection {
	result := reportbiz.LayerProjection{Key: key, Title: layer.Title, Summary: projectLayerSummary(layer, tokens[key+"/evidence_refs"])}
	result.AffectedAnchors = make([]reportbiz.AnchorProjection, len(layer.AffectedAnchors))
	for index, anchor := range layer.AffectedAnchors {
		result.AffectedAnchors[index] = reportbiz.AnchorProjection{
			LocalKey: anchor.LocalKey, Name: anchor.Name, CurrentState: anchor.CurrentState, Result: anchor.Result,
			ConclusionBasis: anchor.ConclusionBasis, ValidationStatus: anchor.ValidationStatus,
			Reasoning: anchor.Reasoning, TimeWindow: anchor.TimeWindow, Confidence: anchor.Confidence,
			EvidenceScopeToken: tokens[key+"/affected_anchors/"+anchor.LocalKey+"/evidence_refs"],
		}
	}
	result.ReasoningSteps = make([]reportbiz.ReasoningStepProjection, len(layer.ReasoningSteps))
	for index, step := range layer.ReasoningSteps {
		result.ReasoningSteps[index] = reportbiz.ReasoningStepProjection{
			LocalKey: step.LocalKey, Input: step.Input, Mechanism: step.Mechanism, Output: step.Output,
			Confidence: step.Confidence, EvidenceScopeToken: tokens[key+"/reasoning_steps/"+step.LocalKey+"/evidence_refs"],
		}
	}
	return result
}

func projectChainSummary(ordinal int, chain reportbiz.IndustryChain, tokens map[string]*string) reportbiz.IndustryChainSummary {
	prefix := "industry_chains/" + chain.LocalKey
	result := reportbiz.IndustryChainSummary{
		Ordinal: ordinal, LocalKey: chain.LocalKey, Name: chain.Name, Conclusion: chain.Conclusion,
		Result: chain.Result, Confidence: chain.Confidence, TimeWindow: chain.TimeWindow,
		EvidenceScopeToken: tokens[prefix+"/evidence_refs"],
		ImpactItems:        make([]reportbiz.IndustryChainImpactSummary, len(chain.Nodes)),
	}
	for index, node := range chain.Nodes {
		result.ImpactItems[index] = reportbiz.IndustryChainImpactSummary{
			LocalKey: node.LocalKey, Name: node.Name, Result: node.Result,
			ConclusionBasis: node.ConclusionBasis, ValidationStatus: node.ValidationStatus,
			Confidence: node.Confidence, TimeWindow: node.TimeWindow,
			EvidenceScopeToken: tokens[prefix+"/nodes/"+node.LocalKey+"/evidence_refs"],
		}
	}
	return result
}

func projectChain(chain reportbiz.IndustryChain, tokens map[string]*string) reportbiz.IndustryChainProjection {
	prefix := "industry_chains/" + chain.LocalKey
	graph := reportbiz.IndustryChainGraph{
		Nodes: make([]reportbiz.IndustryChainTopologyNode, len(chain.Nodes)),
		Edges: make([]reportbiz.IndustryChainEdgeProjection, len(chain.Edges)),
	}
	for index, node := range chain.Nodes {
		graph.Nodes[index] = reportbiz.IndustryChainTopologyNode{LocalKey: node.LocalKey, Name: node.Name}
	}
	for index, edge := range chain.Edges {
		graph.Edges[index] = reportbiz.IndustryChainEdgeProjection{
			FromNodeLocalKey: edge.FromNodeLocalKey, ToNodeLocalKey: edge.ToNodeLocalKey, RelationLabel: edge.RelationLabel,
		}
	}
	result := reportbiz.IndustryChainProjection{
		LocalKey: chain.LocalKey, Name: chain.Name, Conclusion: chain.Conclusion,
		Result: chain.Result, Confidence: chain.Confidence, TimeWindow: chain.TimeWindow,
		PathSummary: chain.PathSummary, AcceptedHypothesisSummary: chain.AcceptedHypothesisSummary,
		Graph: graph, CounterevidenceAndGap: chain.Uncertainty.CounterevidenceAndGap,
		StopCondition: chain.Uncertainty.StopCondition, EvidenceScopeToken: tokens[prefix+"/evidence_refs"],
		AffectedNodes: make([]reportbiz.IndustryChainNodeProjection, len(chain.Nodes)),
	}
	for index, node := range chain.Nodes {
		result.AffectedNodes[index] = reportbiz.IndustryChainNodeProjection{
			LocalKey: node.LocalKey, Name: node.Name, Impact: node.Impact, Result: node.Result,
			ConclusionBasis: node.ConclusionBasis, ValidationStatus: node.ValidationStatus,
			Reasoning: node.Reasoning, TimeWindow: node.TimeWindow, Confidence: node.Confidence,
			EvidenceScopeToken: tokens[prefix+"/nodes/"+node.LocalKey+"/evidence_refs"],
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
		&result.HasMacroeconomics, &result.IndustryChainCount, &result.PublishedAt, &result.SchemaVersion, &result.AnalysisWindowStart, &result.AnalysisWindowEnd); err != nil {
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
	if result.SchemaVersion != "" {
		start, e1 := time.Parse(time.RFC3339Nano, result.AnalysisWindowStart)
		end, e2 := time.Parse(time.RFC3339Nano, result.AnalysisWindowEnd)
		if result.SchemaVersion != reportbiz.AnalysisSchemaVersion || e1 != nil || e2 != nil || !start.Before(end) {
			return persistedInvariant("Report summary", "version/window", "invalid analysis metadata")
		}
	} else if result.IndustryChainCount < 1 {
		return persistedInvariant("Report summary", "industry_chain_count", "legacy Report requires a chain")
	}
	result.GeneratedAt = parsed
	result.PublishedAt = result.PublishedAt.UTC()
	if !coreid.Is(result.ID, coreid.Report) || strings.TrimSpace(result.PublisherReportID) == "" ||
		result.PublisherReportID != strings.TrimSpace(result.PublisherReportID) || result.IndustryChainCount < 0 || result.PublishedAt.IsZero() {
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

// Analysis collections stay in the immutable JSONB snapshot. SQL slices the requested
// collection before Go decoding; chain details are fetched independently.
func (s Store) ListAnalyses(ctx context.Context, f reportbiz.AnalysisListFilter) (reportbiz.AnalysisStorePage, error) {
	if err := s.requireAnalysisReport(ctx, f.ReportID); err != nil {
		return reportbiz.AnalysisStorePage{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT unit.ordinality,`+analysisUnitReadJSON+` FROM reports r
 CROSS JOIN LATERAL jsonb_array_elements(r.report -> $2) WITH ORDINALITY unit(value,ordinality)
 WHERE r.id=$1 AND unit.ordinality>$3 ORDER BY unit.ordinality LIMIT $4`, f.ReportID, f.Kind, f.AfterOrdinal, f.Limit+1)
	if err != nil {
		return reportbiz.AnalysisStorePage{}, fmt.Errorf("query analyses: %w", err)
	}
	defer rows.Close()
	type entry struct {
		ordinal int
		unit    reportbiz.AnalysisUnit
	}
	entries := []entry{}
	for rows.Next() {
		var e entry
		var raw []byte
		if err := rows.Scan(&e.ordinal, &raw); err != nil {
			return reportbiz.AnalysisStorePage{}, err
		}
		if err := decodeStoredJSON(raw, &e.unit); err != nil {
			return reportbiz.AnalysisStorePage{}, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return reportbiz.AnalysisStorePage{}, err
	}
	page := reportbiz.AnalysisStorePage{Items: []reportbiz.AnalysisUnitSummary{}, HasMore: len(entries) > f.Limit}
	if page.HasMore {
		entries = entries[:f.Limit]
	}
	tokens, err := s.scopeTokens(ctx, f.ReportID)
	if err != nil {
		return page, err
	}
	for _, e := range entries {
		summary, err := projectAnalysisSummary(f.Kind, e.unit, e.ordinal, tokens)
		if err != nil {
			return reportbiz.AnalysisStorePage{}, err
		}
		page.Items = append(page.Items, summary)
	}
	return page, nil
}
func (s Store) requireAnalysisReport(ctx context.Context, id string) error {
	var version string
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(report->>'schema_version','') FROM reports WHERE id=$1`, id).Scan(&version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.ErrReportNotFound
		}
		return err
	}
	if version != reportbiz.AnalysisSchemaVersion {
		return reportbiz.ErrLayerNotFound
	}
	return nil
}
func (s Store) GetAnalysis(ctx context.Context, id, kind, key string) (reportbiz.AnalysisUnitDetail, error) {
	if err := s.requireAnalysisReport(ctx, id); err != nil {
		return reportbiz.AnalysisUnitDetail{}, err
	}
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT `+analysisUnitReadJSON+`
 FROM reports r CROSS JOIN LATERAL jsonb_array_elements(r.report->$2) WITH ORDINALITY unit(value,ordinality)
 WHERE r.id=$1 AND unit.value->>'local_key'=$3`, id, kind, key).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.AnalysisUnitDetail{}, reportbiz.ErrLayerNotFound
		}
		return reportbiz.AnalysisUnitDetail{}, err
	}
	var unit reportbiz.AnalysisUnit
	if err := decodeStoredJSON(raw, &unit); err != nil {
		return reportbiz.AnalysisUnitDetail{}, err
	}
	tokens, err := s.scopeTokens(ctx, id)
	if err != nil {
		return reportbiz.AnalysisUnitDetail{}, err
	}
	summary, err := projectAnalysisSummary(kind, unit, 0, tokens)
	if err != nil {
		return reportbiz.AnalysisUnitDetail{}, err
	}
	result := reportbiz.AnalysisUnitDetail{Summary: summary, ReasoningSteps: projectAnalysisSteps(kind+"/"+key+"/detail", unit.Detail.ReasoningSteps, tokens), AffectedAnchors: []reportbiz.AnalysisImpactProjection{}, Uncertainty: unit.Detail.Uncertainty, IndustryChains: []reportbiz.ChainAnalysisSummary{}}
	for _, a := range unit.Detail.AffectedAnchors {
		result.AffectedAnchors = append(result.AffectedAnchors, projectAnalysisImpact(kind+"/"+key+"/detail/affected_anchors", a, tokens))
	}
	for _, c := range unit.Detail.IndustryChains {
		result.IndustryChains = append(result.IndustryChains, reportbiz.ChainAnalysisSummary{LocalKey: c.LocalKey, SourceID: c.SourceID, Name: c.Name, Conclusion: c.Conclusion})
	}
	return result, nil
}
func (s Store) GetAnalysisChain(ctx context.Context, id, concept, chain string) (reportbiz.ChainAnalysisDetail, error) {
	if err := s.requireAnalysisReport(ctx, id); err != nil {
		return reportbiz.ChainAnalysisDetail{}, err
	}
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT c FROM reports r CROSS JOIN LATERAL jsonb_array_elements(r.report->'concept_analyses') u
 CROSS JOIN LATERAL jsonb_array_elements(u#>'{detail,industry_chains}') c WHERE r.id=$1 AND u->>'local_key'=$2 AND c->>'local_key'=$3`, id, concept, chain).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reportbiz.ChainAnalysisDetail{}, reportbiz.ErrChainNotFound
		}
		return reportbiz.ChainAnalysisDetail{}, err
	}
	var c reportbiz.ChainAnalysis
	if err := decodeStoredJSON(raw, &c); err != nil {
		return reportbiz.ChainAnalysisDetail{}, err
	}
	tokens, err := s.scopeTokens(ctx, id)
	if err != nil {
		return reportbiz.ChainAnalysisDetail{}, err
	}
	p := "concept_analyses/" + concept + "/detail/industry_chains/" + chain
	result := reportbiz.ChainAnalysisDetail{LocalKey: c.LocalKey, SourceID: c.SourceID, Name: c.Name, Conclusion: c.Conclusion, TransmissionLogic: c.TransmissionLogic, ReasoningSteps: projectAnalysisSteps(p, c.ReasoningSteps, tokens), Graph: c.Graph, AffectedNodes: []reportbiz.AnalysisImpactProjection{}, Uncertainty: c.Uncertainty, EvidenceScopeToken: tokens[p+"/evidence_refs"]}
	for _, a := range c.AffectedNodes {
		result.AffectedNodes = append(result.AffectedNodes, projectAnalysisImpact(p+"/affected_nodes", a, tokens))
	}
	return result, nil
}
func projectAnalysisSteps(p string, steps []reportbiz.ReasoningStep, tokens map[string]*string) []reportbiz.ReasoningStepProjection {
	result := make([]reportbiz.ReasoningStepProjection, 0, len(steps))
	for _, s := range steps {
		result = append(result, reportbiz.ReasoningStepProjection{LocalKey: s.LocalKey, Input: s.Input, Mechanism: s.Mechanism, Output: s.Output, Confidence: s.Confidence, EvidenceScopeToken: tokens[p+"/reasoning_steps/"+s.LocalKey+"/evidence_refs"]})
	}
	return result
}
func projectAnalysisImpact(p string, a reportbiz.AnalysisImpact, tokens map[string]*string) reportbiz.AnalysisImpactProjection {
	return reportbiz.AnalysisImpactProjection{LocalKey: a.LocalKey, TargetType: a.TargetType, SourceID: a.SourceID, NodeLocalKey: a.NodeLocalKey, Name: a.Name, Impact: a.Impact, Result: a.Result, ConclusionBasis: a.ConclusionBasis, ValidationStatus: a.ValidationStatus, Reasoning: a.Reasoning, TransmissionSignal: a.TransmissionSignal, Conditions: a.Conditions, FollowUp: a.FollowUp, TimeWindow: a.TimeWindow, Confidence: a.Confidence, EvidenceScopeToken: tokens[p+"/"+a.LocalKey+"/evidence_refs"]}
}
func projectAnalysisSummary(kind string, u reportbiz.AnalysisUnit, ordinal int, tokens map[string]*string) (reportbiz.AnalysisUnitSummary, error) {
	p := kind + "/" + u.LocalKey
	result := reportbiz.AnalysisUnitSummary{LocalKey: u.LocalKey, SourceID: u.SourceID, Title: u.Title, Conclusion: u.Summary.Conclusion, TransmissionLogic: u.Summary.TransmissionLogic, AffectedAnchors: []reportbiz.AnalysisImpactProjection{}, ChainCount: len(u.Detail.IndustryChains), Ordinal: ordinal, EvidenceScopeToken: tokens[p+"/summary/evidence_refs"]}
	impacts := map[string]reportbiz.AnalysisImpactProjection{}
	for _, a := range u.Detail.AffectedAnchors {
		impacts[a.LocalKey] = projectAnalysisImpact(p+"/detail/affected_anchors", a, tokens)
	}
	for _, c := range u.Detail.IndustryChains {
		for _, a := range c.AffectedNodes {
			impacts[a.LocalKey] = projectAnalysisImpact(p+"/detail/industry_chains/"+c.LocalKey+"/affected_nodes", a, tokens)
		}
	}
	for _, key := range u.Summary.AnchorKeys {
		a, ok := impacts[key]
		if !ok {
			return reportbiz.AnalysisUnitSummary{}, persistedInvariant("analysis summary", "anchor_keys", "reference does not resolve in this unit")
		}
		result.AffectedAnchors = append(result.AffectedAnchors, a)
	}
	return result, nil
}

// Preserve only chain headers and explicitly referenced summary impacts. Full chain
// graphs and reasoning are loaded through GetAnalysisChain.
const analysisUnitReadJSON = `jsonb_set(unit.value,'{detail,industry_chains}',
 COALESCE((SELECT jsonb_agg(jsonb_build_object(
 'local_key',c->'local_key','source_id',c->'source_id','name',c->'name','conclusion',c->'conclusion',
 'affected_nodes',COALESCE((SELECT jsonb_agg(a ORDER BY pos) FROM jsonb_array_elements(c->'affected_nodes') WITH ORDINALITY impacts(a,pos)
 WHERE a->>'local_key' IN (SELECT jsonb_array_elements_text(unit.value#>'{summary,anchor_keys}'))),'[]'::jsonb)) ORDER BY ord)
 FROM jsonb_array_elements(unit.value#>'{detail,industry_chains}') WITH ORDINALITY chains(c,ord)),'[]'::jsonb))`
