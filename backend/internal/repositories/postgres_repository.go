package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) SeedSource(ctx context.Context, source domain.SourceCatalog) error {
	source = normalizeSource(source)
	sourceConfig, err := json.Marshal(source.SourceConfig)
	if err != nil {
		return fmt.Errorf("marshal source config: %w", err)
	}
	policy, err := json.Marshal(source.RateLimitPolicy)
	if err != nil {
		return fmt.Errorf("marshal rate limit policy: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO source_catalogs (
    id, ingest_channel, provider_key, connector_key, parser_key, source_type,
    source_name, source_url, source_level, topic_hint, route_template, code_style,
    auth_required, auth_type, credential_ref, source_config, rate_limit_policy, usage_policy, status
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12,
    $13, $14, $15, $16::jsonb, $17::jsonb, $18, $19
) ON CONFLICT (id) DO UPDATE SET
    ingest_channel = EXCLUDED.ingest_channel,
    provider_key = EXCLUDED.provider_key,
    connector_key = EXCLUDED.connector_key,
    parser_key = EXCLUDED.parser_key,
    source_type = EXCLUDED.source_type,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    source_level = EXCLUDED.source_level,
    topic_hint = EXCLUDED.topic_hint,
    route_template = EXCLUDED.route_template,
    code_style = EXCLUDED.code_style,
    auth_required = EXCLUDED.auth_required,
    auth_type = EXCLUDED.auth_type,
    credential_ref = EXCLUDED.credential_ref,
    source_config = EXCLUDED.source_config,
    rate_limit_policy = EXCLUDED.rate_limit_policy,
    usage_policy = EXCLUDED.usage_policy,
    status = EXCLUDED.status,
    updated_at = now()
`, source.ID, source.IngestChannel, source.ProviderKey, source.ConnectorKey, source.ParserKey, source.SourceType,
		source.SourceName, source.SourceURL, source.SourceLevel, source.TopicHint, source.RouteTemplate, source.CodeStyle,
		source.AuthRequired, source.AuthType, source.CredentialRef, string(sourceConfig), string(policy), source.UsagePolicy, source.Status)
	if err != nil {
		return fmt.Errorf("seed source catalog: %w", err)
	}

	return nil
}

func (r PostgresRepository) ActiveSources(ctx context.Context, filter SourceCatalogFilter) ([]domain.SourceCatalog, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, ingest_channel, provider_key, connector_key, parser_key, source_type,
       source_name, source_url, source_level, topic_hint, route_template, code_style,
       auth_required, auth_type, credential_ref, source_config, rate_limit_policy, usage_policy, status
FROM source_catalogs
WHERE status = 'active'
  AND ($1 = '' OR id::text = $1)
  AND ($2 = '' OR provider_key = $2)
  AND ($3 = '' OR ingest_channel = $3)
  AND ($4 = '' OR source_type = $4)
ORDER BY source_name, id
LIMIT CASE WHEN $5 > 0 THEN $5 ELSE 2147483647 END
`, filter.SourceID, filter.ProviderKey, filter.IngestChannel, filter.SourceType, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("query active source catalogs: %w", err)
	}
	defer rows.Close()

	var sources []domain.SourceCatalog
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source catalogs: %w", err)
	}

	return sources, nil
}

func (r PostgresRepository) SourceCatalogStats(ctx context.Context) (SourceCatalogStats, error) {
	stats := newSourceCatalogStats()
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM source_catalogs`).Scan(&stats.Total); err != nil {
		return SourceCatalogStats{}, fmt.Errorf("count source catalogs: %w", err)
	}
	for _, group := range []struct {
		column string
		target map[string]int
	}{
		{column: "provider_key", target: stats.ByProviderKey},
		{column: "ingest_channel", target: stats.ByIngestChannel},
		{column: "source_type", target: stats.BySourceType},
		{column: "usage_policy", target: stats.ByUsagePolicy},
		{column: "status", target: stats.ByStatus},
	} {
		if err := r.loadSourceCatalogStatsGroup(ctx, group.column, group.target); err != nil {
			return SourceCatalogStats{}, err
		}
	}
	return stats, nil
}

func (r PostgresRepository) loadSourceCatalogStatsGroup(ctx context.Context, column string, target map[string]int) error {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
SELECT COALESCE(NULLIF(%s, ''), 'unknown') AS stat_key, COUNT(*)
FROM source_catalogs
GROUP BY stat_key
`, column))
	if err != nil {
		return fmt.Errorf("query source catalog stats by %s: %w", column, err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return fmt.Errorf("scan source catalog stats by %s: %w", column, err)
		}
		target[key] = count
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate source catalog stats by %s: %w", column, err)
	}
	return nil
}

func (r PostgresRepository) UpsertRawDocument(ctx context.Context, doc domain.RawDocument) (RawDocumentWriteResult, error) {
	doc = normalizeRawDocument(doc)
	if err := doc.Validate(); err != nil {
		return RawDocumentWriteResult{}, err
	}

	existing, ok, err := r.findDuplicate(ctx, doc)
	if err != nil {
		return RawDocumentWriteResult{}, err
	}
	if ok {
		return RawDocumentWriteResult{
			Document:    existing,
			Created:     false,
			DuplicateOf: existing.ID,
		}, nil
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO raw_documents (
    id, source_id, ingest_channel, source_type, source_name, source_url,
    source_external_id, title, content_text, raw_object_uri, raw_mime_type,
    language, published_at, collected_at, content_hash, ingest_status
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16
)
`, doc.ID, doc.SourceID, doc.IngestChannel, doc.SourceType, doc.SourceName, doc.SourceURL,
		nullString(doc.SourceExternalID), doc.Title, doc.ContentText, doc.RawObjectURI, doc.RawMIMEType,
		doc.Language, nullTime(doc.PublishedAt), doc.CollectedAt, doc.ContentHash, doc.IngestStatus)
	if err != nil {
		return RawDocumentWriteResult{}, fmt.Errorf("insert raw document: %w", err)
	}

	return RawDocumentWriteResult{
		Document: doc,
		Created:  true,
	}, nil
}

func (r PostgresRepository) UpdateRawDocumentStatus(ctx context.Context, id string, status domain.IngestStatus) error {
	normalizedID := NormalizeUUID(id)
	result, err := r.db.ExecContext(ctx, `
UPDATE raw_documents
SET ingest_status = $2, updated_at = now()
WHERE id = $1
`, normalizedID, status)
	if err != nil {
		return fmt.Errorf("update raw document status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("raw document %q not found", id)
	}
	return nil
}

func (r PostgresRepository) RawDocumentCount(ctx context.Context, sourceID string) (int, error) {
	var count int
	if sourceID == "" {
		err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM raw_documents
`).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count raw documents: %w", err)
		}
		return count, nil
	}

	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM raw_documents
WHERE source_id = $1
`, NormalizeUUID(sourceID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count raw documents: %w", err)
	}
	return count, nil
}

func (r PostgresRepository) ListRawDocuments(ctx context.Context, filter RawDocumentListFilter) (RawDocumentPage, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	var total int
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM raw_documents
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
`, filter.Title).Scan(&total); err != nil {
		return RawDocumentPage{}, fmt.Errorf("count raw documents: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, source_id, ingest_channel, source_type, source_name, source_url,
       source_external_id, title, content_text, raw_object_uri, raw_mime_type,
       language, published_at, collected_at, content_hash, ingest_status
FROM raw_documents
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
ORDER BY collected_at DESC, id
LIMIT $2 OFFSET $3
`, filter.Title, pageSize, (page-1)*pageSize)
	if err != nil {
		return RawDocumentPage{}, fmt.Errorf("query raw documents: %w", err)
	}
	defer rows.Close()

	items := make([]domain.RawDocument, 0)
	for rows.Next() {
		doc, err := scanRawDocument(rows)
		if err != nil {
			return RawDocumentPage{}, err
		}
		items = append(items, doc)
	}
	if err := rows.Err(); err != nil {
		return RawDocumentPage{}, fmt.Errorf("iterate raw documents: %w", err)
	}
	return RawDocumentPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r PostgresRepository) ListEvents(ctx context.Context, filter EventListFilter) (EventPage, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	var total int
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM events
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR event_status = $2)
  AND ($3 = '' OR fact_status = $3)
  AND ($4::timestamptz IS NULL OR event_time >= $4)
  AND ($5::timestamptz IS NULL OR event_time <= $5)
  AND ($6::timestamptz IS NULL OR first_seen_at >= $6)
  AND ($7::timestamptz IS NULL OR first_seen_at <= $7)
`, filter.Title, string(filter.EventStatus), string(filter.FactStatus), nullTime(filter.EventTimeFrom), nullTime(filter.EventTimeTo), nullTime(filter.FirstSeenFrom), nullTime(filter.FirstSeenTo)).Scan(&total); err != nil {
		return EventPage{}, fmt.Errorf("count events: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, title, summary, event_time, first_seen_at, knowable_at,
       event_status, fact_status, dedupe_key, primary_source_id
FROM events
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR event_status = $2)
  AND ($3 = '' OR fact_status = $3)
  AND ($4::timestamptz IS NULL OR event_time >= $4)
  AND ($5::timestamptz IS NULL OR event_time <= $5)
  AND ($6::timestamptz IS NULL OR first_seen_at >= $6)
  AND ($7::timestamptz IS NULL OR first_seen_at <= $7)
ORDER BY first_seen_at DESC, event_time DESC NULLS LAST, id
LIMIT $8 OFFSET $9
`, filter.Title, string(filter.EventStatus), string(filter.FactStatus), nullTime(filter.EventTimeFrom), nullTime(filter.EventTimeTo), nullTime(filter.FirstSeenFrom), nullTime(filter.FirstSeenTo), pageSize, (page-1)*pageSize)
	if err != nil {
		return EventPage{}, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Event, 0)
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return EventPage{}, err
		}
		items = append(items, event)
	}
	if err := rows.Err(); err != nil {
		return EventPage{}, fmt.Errorf("iterate events: %w", err)
	}
	return EventPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r PostgresRepository) ListSourceCatalogs(ctx context.Context, filter SourceCatalogListFilter) ([]domain.SourceCatalog, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, ingest_channel, provider_key, connector_key, parser_key, source_type,
       source_name, source_url, source_level, topic_hint, route_template, code_style,
       auth_required, auth_type, credential_ref, source_config, rate_limit_policy, usage_policy, status
FROM source_catalogs
WHERE ($1 = '' OR status = $1)
ORDER BY provider_key, source_name, id
`, string(filter.Status))
	if err != nil {
		return nil, fmt.Errorf("query source catalogs: %w", err)
	}
	defer rows.Close()

	items := make([]domain.SourceCatalog, 0)
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source catalogs: %w", err)
	}
	return items, nil
}

func (r PostgresRepository) ListGraphEntityNodes(ctx context.Context) ([]GraphEntityNode, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id,
       COALESCE(NULLIF(entity_key, ''), entity_type || ':' || id::text) AS entity_key,
       entity_type, layer_code, name, canonical_name, status, updated_at
FROM entity_nodes
ORDER BY id
`)
	if err != nil {
		return nil, fmt.Errorf("query graph entity nodes: %w", err)
	}
	defer rows.Close()

	nodes := make([]GraphEntityNode, 0)
	for rows.Next() {
		var node GraphEntityNode
		if err := rows.Scan(
			&node.ID,
			&node.EntityKey,
			&node.EntityType,
			&node.LayerCode,
			&node.Name,
			&node.CanonicalName,
			&node.Status,
			&node.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan graph entity node: %w", err)
		}
		nodes = append(nodes, normalizeGraphEntityNode(node))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate graph entity nodes: %w", err)
	}
	return nodes, nil
}

func (r PostgresRepository) ListGraphEntityEdges(ctx context.Context) ([]GraphEntityEdge, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, from_entity_id, to_entity_id, relation_type, evidence_note, status, updated_at
FROM entity_edges
ORDER BY id
`)
	if err != nil {
		return nil, fmt.Errorf("query graph entity edges: %w", err)
	}
	defer rows.Close()

	edges := make([]GraphEntityEdge, 0)
	for rows.Next() {
		var edge GraphEntityEdge
		if err := rows.Scan(
			&edge.ID,
			&edge.FromEntityID,
			&edge.ToEntityID,
			&edge.RelationType,
			&edge.EvidenceNote,
			&edge.Status,
			&edge.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan graph entity edge: %w", err)
		}
		edges = append(edges, edge)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate graph entity edges: %w", err)
	}
	return edges, nil
}

func (r PostgresRepository) CreateGraphProjectionRun(ctx context.Context, run GraphProjectionRun) (GraphProjectionRun, error) {
	run = normalizeGraphProjectionRun(run)
	run.ID = NormalizeUUID(run.ID)
	configSummary, err := json.Marshal(cloneMap(run.ConfigSummary))
	if err != nil {
		return GraphProjectionRun{}, fmt.Errorf("marshal graph projection config summary: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
INSERT INTO graph_projection_runs (
    id, projection_type, mode, status, started_at, finished_at, source_row_count,
    projected_count, skipped_count, failed_count, error_summary, config_summary
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12::jsonb
) RETURNING id, projection_type, mode, status, started_at, finished_at,
            source_row_count, projected_count, skipped_count, failed_count,
            error_summary, config_summary
`, run.ID, run.ProjectionType, run.Mode, run.Status, run.StartedAt, nullTime(run.FinishedAt),
		run.SourceRowCount, run.ProjectedCount, run.SkippedCount, run.FailedCount, run.ErrorSummary, string(configSummary))

	created, err := scanGraphProjectionRun(row)
	if err != nil {
		return GraphProjectionRun{}, fmt.Errorf("create graph projection run: %w", err)
	}
	return created, nil
}

func (r PostgresRepository) RecordGraphProjectionRunItem(ctx context.Context, item GraphProjectionRunItem) error {
	item.ID = NormalizeUUID(item.ID)
	item.RunID = NormalizeUUID(item.RunID)

	_, err := r.db.ExecContext(ctx, `
INSERT INTO graph_projection_run_items (
    id, run_id, item_type, item_key, status, error_message
) VALUES (
    $1, $2, $3, $4, $5, $6
)
`, item.ID, item.RunID, item.ItemType, item.ItemKey, item.Status, item.ErrorMessage)
	if err != nil {
		return fmt.Errorf("record graph projection run item: %w", err)
	}
	return nil
}

func (r PostgresRepository) CompleteGraphProjectionRun(ctx context.Context, run GraphProjectionRun) error {
	run.ID = NormalizeUUID(run.ID)
	run = normalizeGraphProjectionRun(run)
	configSummary, err := json.Marshal(cloneMap(run.ConfigSummary))
	if err != nil {
		return fmt.Errorf("marshal completed graph projection config summary: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE graph_projection_runs
SET status = $2,
    finished_at = $3,
    source_row_count = $4,
    projected_count = $5,
    skipped_count = $6,
    failed_count = $7,
    error_summary = $8,
    config_summary = $9::jsonb,
    updated_at = now()
WHERE id = $1
`, run.ID, run.Status, nullTime(run.FinishedAt), run.SourceRowCount, run.ProjectedCount,
		run.SkippedCount, run.FailedCount, run.ErrorSummary, string(configSummary))
	if err != nil {
		return fmt.Errorf("complete graph projection run: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read completed graph projection run affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("graph projection run %q not found", run.ID)
	}
	return nil
}

func (r PostgresRepository) RecentGraphProjectionRuns(ctx context.Context, limit int) ([]GraphProjectionRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, projection_type, mode, status, started_at, finished_at,
       source_row_count, projected_count, skipped_count, failed_count,
       error_summary, config_summary
FROM graph_projection_runs
ORDER BY started_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent graph projection runs: %w", err)
	}
	defer rows.Close()

	runs := make([]GraphProjectionRun, 0)
	for rows.Next() {
		run, err := scanGraphProjectionRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent graph projection runs: %w", err)
	}
	return runs, nil
}

func (r PostgresRepository) LoadSchedulerConfig(ctx context.Context) (domain.SchedulerConfig, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, enabled, mode, interval_minutes, fixed_times, concurrency,
       batch_size, timeout_seconds, source_filter, timezone, config_version,
       last_run_id, last_run_at, created_at, updated_at
FROM ingestion_scheduler_configs
WHERE id = 'default'
`)
	config, err := scanSchedulerConfig(row)
	if err == sql.ErrNoRows {
		return defaultSchedulerConfig(), nil
	}
	if err != nil {
		return domain.SchedulerConfig{}, fmt.Errorf("load scheduler config: %w", err)
	}
	return config, nil
}

func (r PostgresRepository) SaveSchedulerConfig(ctx context.Context, config domain.SchedulerConfig) (domain.SchedulerConfig, error) {
	config = normalizeSchedulerConfig(config)
	if err := config.Validate(); err != nil {
		return domain.SchedulerConfig{}, err
	}
	fixedTimes, err := json.Marshal(config.FixedTimes)
	if err != nil {
		return domain.SchedulerConfig{}, fmt.Errorf("marshal fixed times: %w", err)
	}
	sourceFilter, err := json.Marshal(schedulerSourceFilterMap(config.SourceFilter))
	if err != nil {
		return domain.SchedulerConfig{}, fmt.Errorf("marshal scheduler source filter: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
INSERT INTO ingestion_scheduler_configs (
    id, enabled, mode, interval_minutes, fixed_times, concurrency,
    batch_size, timeout_seconds, source_filter, timezone, config_version
) VALUES (
    $1, $2, $3, $4, $5::jsonb, $6,
    $7, $8, $9::jsonb, $10, 1
) ON CONFLICT (id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    mode = EXCLUDED.mode,
    interval_minutes = EXCLUDED.interval_minutes,
    fixed_times = EXCLUDED.fixed_times,
    concurrency = EXCLUDED.concurrency,
    batch_size = EXCLUDED.batch_size,
    timeout_seconds = EXCLUDED.timeout_seconds,
    source_filter = EXCLUDED.source_filter,
    timezone = EXCLUDED.timezone,
    config_version = ingestion_scheduler_configs.config_version + 1,
    updated_at = now()
RETURNING id, enabled, mode, interval_minutes, fixed_times, concurrency,
          batch_size, timeout_seconds, source_filter, timezone, config_version,
          last_run_id, last_run_at, created_at, updated_at
`, config.ID, config.Enabled, config.Mode, nullablePositiveInt(config.IntervalMinutes), string(fixedTimes), config.Concurrency,
		config.BatchSize, config.TimeoutSeconds, string(sourceFilter), config.Timezone)

	saved, err := scanSchedulerConfig(row)
	if err != nil {
		return domain.SchedulerConfig{}, fmt.Errorf("save scheduler config: %w", err)
	}
	return saved, nil
}

func (r PostgresRepository) CreateIngestionRun(ctx context.Context, run domain.IngestionRun) (domain.IngestionRun, error) {
	run.ID = NormalizeUUID(run.ID)
	if err := run.Validate(); err != nil {
		return domain.IngestionRun{}, err
	}
	schedulerConfig, err := json.Marshal(cloneMap(run.SchedulerConfig))
	if err != nil {
		return domain.IngestionRun{}, fmt.Errorf("marshal run scheduler config: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
INSERT INTO ingestion_runs (
    id, trigger_type, status, started_at, finished_at, total_sources,
    succeeded_sources, failed_sources, skipped_sources, scheduler_config, error_summary
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10::jsonb, $11
) RETURNING id, trigger_type, status, started_at, finished_at, total_sources,
            succeeded_sources, failed_sources, skipped_sources, scheduler_config,
            error_summary, created_at, updated_at
`, run.ID, run.TriggerType, run.Status, run.StartedAt, nullTime(run.FinishedAt),
		run.TotalSources, run.SucceededSources, run.FailedSources, run.SkippedSources, string(schedulerConfig), run.ErrorSummary)

	created, err := scanIngestionRun(row)
	if err != nil {
		return domain.IngestionRun{}, fmt.Errorf("create ingestion run: %w", err)
	}
	return created, nil
}

func (r PostgresRepository) RecordIngestionRunSource(ctx context.Context, result domain.IngestionRunSource) error {
	result.ID = NormalizeUUID(result.ID)
	result.RunID = NormalizeUUID(result.RunID)
	result.SourceID = NormalizeUUID(result.SourceID)
	if err := result.Validate(); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx, `
INSERT INTO ingestion_run_sources (
    id, run_id, source_id, status, documents_written, documents_duplicate,
    error_message, started_at, finished_at, duration_millis
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10
) ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    documents_written = EXCLUDED.documents_written,
    documents_duplicate = EXCLUDED.documents_duplicate,
    error_message = EXCLUDED.error_message,
    started_at = EXCLUDED.started_at,
    finished_at = EXCLUDED.finished_at,
    duration_millis = EXCLUDED.duration_millis,
    updated_at = now()
`, result.ID, result.RunID, result.SourceID, result.Status, result.DocumentsWritten, result.DocumentsDuplicate,
		result.ErrorMessage, result.StartedAt, nullTime(result.FinishedAt), result.DurationMillis)
	if err != nil {
		return fmt.Errorf("record ingestion run source: %w", err)
	}
	return nil
}

func (r PostgresRepository) CompleteIngestionRun(ctx context.Context, run domain.IngestionRun) error {
	run.ID = NormalizeUUID(run.ID)
	if err := run.Validate(); err != nil {
		return err
	}
	schedulerConfig, err := json.Marshal(cloneMap(run.SchedulerConfig))
	if err != nil {
		return fmt.Errorf("marshal completed run scheduler config: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE ingestion_runs
SET status = $2,
    finished_at = $3,
    total_sources = $4,
    succeeded_sources = $5,
    failed_sources = $6,
    skipped_sources = $7,
    scheduler_config = $8::jsonb,
    error_summary = $9,
    updated_at = now()
WHERE id = $1
`, run.ID, run.Status, nullTime(run.FinishedAt), run.TotalSources, run.SucceededSources,
		run.FailedSources, run.SkippedSources, string(schedulerConfig), run.ErrorSummary)
	if err != nil {
		return fmt.Errorf("complete ingestion run: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read completed run affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("ingestion run %q not found", run.ID)
	}

	_, err = r.db.ExecContext(ctx, `
UPDATE ingestion_scheduler_configs
SET last_run_id = $1,
    last_run_at = $2,
    updated_at = now()
WHERE id = 'default'
`, run.ID, run.StartedAt)
	if err != nil {
		return fmt.Errorf("update scheduler last run: %w", err)
	}
	return nil
}

func (r PostgresRepository) RecentIngestionRuns(ctx context.Context, limit int) ([]domain.IngestionRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, trigger_type, status, started_at, finished_at, total_sources,
       succeeded_sources, failed_sources, skipped_sources, scheduler_config,
       error_summary, created_at, updated_at
FROM ingestion_runs
ORDER BY started_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent ingestion runs: %w", err)
	}
	defer rows.Close()

	runs := make([]domain.IngestionRun, 0)
	for rows.Next() {
		run, err := scanIngestionRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent ingestion runs: %w", err)
	}
	return runs, nil
}

func (r PostgresRepository) findDuplicate(ctx context.Context, doc domain.RawDocument) (domain.RawDocument, bool, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, source_id, ingest_channel, source_type, source_name, source_url,
       source_external_id, title, content_text, raw_object_uri, raw_mime_type,
       language, published_at, collected_at, content_hash, ingest_status
FROM raw_documents
WHERE source_id = $1
  AND (
    ($2 <> '' AND source_external_id = $2)
    OR content_hash = $3
  )
LIMIT 1
`, doc.SourceID, doc.SourceExternalID, doc.ContentHash)

	existing, err := scanRawDocument(row)
	if err == sql.ErrNoRows {
		return domain.RawDocument{}, false, nil
	}
	if err != nil {
		return domain.RawDocument{}, false, fmt.Errorf("find duplicate raw document: %w", err)
	}

	return existing, true, nil
}

type sourceScanner interface {
	Scan(dest ...any) error
}

func scanSource(scanner sourceScanner) (domain.SourceCatalog, error) {
	var source domain.SourceCatalog
	var configBytes []byte
	var policyBytes []byte
	if err := scanner.Scan(
		&source.ID,
		&source.IngestChannel,
		&source.ProviderKey,
		&source.ConnectorKey,
		&source.ParserKey,
		&source.SourceType,
		&source.SourceName,
		&source.SourceURL,
		&source.SourceLevel,
		&source.TopicHint,
		&source.RouteTemplate,
		&source.CodeStyle,
		&source.AuthRequired,
		&source.AuthType,
		&source.CredentialRef,
		&configBytes,
		&policyBytes,
		&source.UsagePolicy,
		&source.Status,
	); err != nil {
		return domain.SourceCatalog{}, fmt.Errorf("scan source catalog: %w", err)
	}
	if len(policyBytes) > 0 {
		if err := json.Unmarshal(policyBytes, &source.RateLimitPolicy); err != nil {
			return domain.SourceCatalog{}, fmt.Errorf("parse rate limit policy: %w", err)
		}
	}
	if len(configBytes) > 0 {
		if err := json.Unmarshal(configBytes, &source.SourceConfig); err != nil {
			return domain.SourceCatalog{}, fmt.Errorf("parse source config: %w", err)
		}
	}
	if source.SourceConfig == nil {
		source.SourceConfig = map[string]any{}
	}
	if source.RateLimitPolicy == nil {
		source.RateLimitPolicy = map[string]any{}
	}
	return source, nil
}

type rawDocumentScanner interface {
	Scan(dest ...any) error
}

func scanSchedulerConfig(scanner rawDocumentScanner) (domain.SchedulerConfig, error) {
	var config domain.SchedulerConfig
	var interval sql.NullInt64
	var fixedTimesBytes []byte
	var sourceFilterBytes []byte
	var lastRunID sql.NullString
	var lastRunAt sql.NullTime
	if err := scanner.Scan(
		&config.ID,
		&config.Enabled,
		&config.Mode,
		&interval,
		&fixedTimesBytes,
		&config.Concurrency,
		&config.BatchSize,
		&config.TimeoutSeconds,
		&sourceFilterBytes,
		&config.Timezone,
		&config.ConfigVersion,
		&lastRunID,
		&lastRunAt,
		&config.CreatedAt,
		&config.UpdatedAt,
	); err != nil {
		return domain.SchedulerConfig{}, err
	}
	if interval.Valid {
		config.IntervalMinutes = int(interval.Int64)
	}
	if len(fixedTimesBytes) > 0 {
		if err := json.Unmarshal(fixedTimesBytes, &config.FixedTimes); err != nil {
			return domain.SchedulerConfig{}, fmt.Errorf("parse scheduler fixed times: %w", err)
		}
	}
	if len(sourceFilterBytes) > 0 {
		var sourceFilter map[string]string
		if err := json.Unmarshal(sourceFilterBytes, &sourceFilter); err != nil {
			return domain.SchedulerConfig{}, fmt.Errorf("parse scheduler source filter: %w", err)
		}
		config.SourceFilter = domain.SchedulerSourceFilter{
			ProviderKey:   sourceFilter["provider_key"],
			IngestChannel: sourceFilter["ingest_channel"],
			SourceType:    sourceFilter["source_type"],
		}
	}
	if lastRunID.Valid {
		config.LastRunID = lastRunID.String
	}
	if lastRunAt.Valid {
		config.LastRunAt = &lastRunAt.Time
	}
	return config, nil
}

func scanRawDocument(scanner rawDocumentScanner) (domain.RawDocument, error) {
	var doc domain.RawDocument
	var sourceExternalID sql.NullString
	var publishedAt sql.NullTime
	if err := scanner.Scan(
		&doc.ID,
		&doc.SourceID,
		&doc.IngestChannel,
		&doc.SourceType,
		&doc.SourceName,
		&doc.SourceURL,
		&sourceExternalID,
		&doc.Title,
		&doc.ContentText,
		&doc.RawObjectURI,
		&doc.RawMIMEType,
		&doc.Language,
		&publishedAt,
		&doc.CollectedAt,
		&doc.ContentHash,
		&doc.IngestStatus,
	); err != nil {
		return domain.RawDocument{}, err
	}
	if sourceExternalID.Valid {
		doc.SourceExternalID = sourceExternalID.String
	}
	if publishedAt.Valid {
		doc.PublishedAt = &publishedAt.Time
	}
	return doc, nil
}

func scanEvent(scanner rawDocumentScanner) (domain.Event, error) {
	var event domain.Event
	var eventTime sql.NullTime
	var knowableAt sql.NullTime
	var primarySourceID sql.NullString
	if err := scanner.Scan(
		&event.ID,
		&event.Title,
		&event.Summary,
		&eventTime,
		&event.FirstSeenAt,
		&knowableAt,
		&event.EventStatus,
		&event.FactStatus,
		&event.DedupeKey,
		&primarySourceID,
	); err != nil {
		return domain.Event{}, fmt.Errorf("scan event: %w", err)
	}
	if eventTime.Valid {
		event.EventTime = &eventTime.Time
	}
	if knowableAt.Valid {
		event.KnowableAt = &knowableAt.Time
	}
	if primarySourceID.Valid {
		event.PrimarySourceID = primarySourceID.String
	}
	return event, nil
}

func scanIngestionRun(scanner rawDocumentScanner) (domain.IngestionRun, error) {
	var run domain.IngestionRun
	var finishedAt sql.NullTime
	var schedulerConfigBytes []byte
	if err := scanner.Scan(
		&run.ID,
		&run.TriggerType,
		&run.Status,
		&run.StartedAt,
		&finishedAt,
		&run.TotalSources,
		&run.SucceededSources,
		&run.FailedSources,
		&run.SkippedSources,
		&schedulerConfigBytes,
		&run.ErrorSummary,
		&run.CreatedAt,
		&run.UpdatedAt,
	); err != nil {
		return domain.IngestionRun{}, err
	}
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	if len(schedulerConfigBytes) > 0 {
		if err := json.Unmarshal(schedulerConfigBytes, &run.SchedulerConfig); err != nil {
			return domain.IngestionRun{}, fmt.Errorf("parse run scheduler config: %w", err)
		}
	}
	if run.SchedulerConfig == nil {
		run.SchedulerConfig = map[string]any{}
	}
	return run, nil
}

func scanGraphProjectionRun(scanner rawDocumentScanner) (GraphProjectionRun, error) {
	var run GraphProjectionRun
	var finishedAt sql.NullTime
	var configSummaryBytes []byte
	if err := scanner.Scan(
		&run.ID,
		&run.ProjectionType,
		&run.Mode,
		&run.Status,
		&run.StartedAt,
		&finishedAt,
		&run.SourceRowCount,
		&run.ProjectedCount,
		&run.SkippedCount,
		&run.FailedCount,
		&run.ErrorSummary,
		&configSummaryBytes,
	); err != nil {
		return GraphProjectionRun{}, err
	}
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	if len(configSummaryBytes) > 0 {
		if err := json.Unmarshal(configSummaryBytes, &run.ConfigSummary); err != nil {
			return GraphProjectionRun{}, fmt.Errorf("parse graph projection config summary: %w", err)
		}
	}
	if run.ConfigSummary == nil {
		run.ConfigSummary = map[string]any{}
	}
	return run, nil
}

func normalizeSource(source domain.SourceCatalog) domain.SourceCatalog {
	source.ID = NormalizeUUID(source.ID)
	if source.SourceLevel == "" {
		source.SourceLevel = "secondary"
	}
	if source.AuthType == "" {
		source.AuthType = "none"
	}
	if source.Status == "" {
		source.Status = domain.SourceCatalogStatusActive
	}
	if source.RateLimitPolicy == nil {
		source.RateLimitPolicy = map[string]any{}
	}
	if source.SourceConfig == nil {
		source.SourceConfig = map[string]any{}
	}
	return source
}

func normalizeRawDocument(doc domain.RawDocument) domain.RawDocument {
	doc.SourceID = NormalizeUUID(doc.SourceID)
	doc.ID = RawDocumentUUID(doc.SourceID, doc.ID, doc.SourceExternalID, doc.ContentHash)
	if doc.CollectedAt.IsZero() {
		doc.CollectedAt = time.Now()
	}
	if doc.IngestStatus == "" {
		doc.IngestStatus = domain.IngestStatusCollected
	}
	return doc
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullablePositiveInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}

func schedulerSourceFilterMap(filter domain.SchedulerSourceFilter) map[string]string {
	result := map[string]string{}
	if filter.ProviderKey != "" {
		result["provider_key"] = filter.ProviderKey
	}
	if filter.IngestChannel != "" {
		result["ingest_channel"] = filter.IngestChannel
	}
	if filter.SourceType != "" {
		result["source_type"] = filter.SourceType
	}
	return result
}
