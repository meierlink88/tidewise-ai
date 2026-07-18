package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

type SourceCatalogFilter struct {
	SourceID      string
	ProviderKey   string
	IngestChannel string
	SourceType    string
	Limit         int
}

type SourceCatalogRepository interface {
	ActiveSources(context.Context, SourceCatalogFilter) ([]domain.SourceCatalog, error)
	SourceCatalogStats(context.Context) (SourceCatalogStats, error)
}

type SourceCatalogStats struct {
	Total           int
	ByProviderKey   map[string]int
	ByIngestChannel map[string]int
	BySourceType    map[string]int
	ByUsagePolicy   map[string]int
	ByStatus        map[string]int
}

func (r *InMemoryRepository) SeedSource(_ context.Context, source domain.SourceCatalog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	source = cloneSource(normalizeInMemorySource(source))
	for index, existing := range r.sources {
		if existing.ID == source.ID {
			r.sources[index] = source
			return nil
		}
	}
	r.sources = append(r.sources, source)
	return nil
}

func (r *InMemoryRepository) ActiveSources(_ context.Context, filter SourceCatalogFilter) ([]domain.SourceCatalog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []domain.SourceCatalog
	for _, source := range r.sources {
		if source.Status != domain.SourceCatalogStatusActive {
			continue
		}
		if filter.SourceID != "" && source.ID != filter.SourceID {
			continue
		}
		if filter.ProviderKey != "" && source.ProviderKey != filter.ProviderKey {
			continue
		}
		if filter.IngestChannel != "" && source.IngestChannel != filter.IngestChannel {
			continue
		}
		if filter.SourceType != "" && source.SourceType != filter.SourceType {
			continue
		}
		result = append(result, cloneSource(normalizeInMemorySource(source)))
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	return result, nil
}

func (r *InMemoryRepository) SourceCatalogStats(_ context.Context) (SourceCatalogStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := newSourceCatalogStats()
	for _, source := range r.sources {
		stats.Total++
		incrementStats(stats.ByProviderKey, source.ProviderKey)
		incrementStats(stats.ByIngestChannel, source.IngestChannel)
		incrementStats(stats.BySourceType, source.SourceType)
		incrementStats(stats.ByUsagePolicy, source.UsagePolicy)
		incrementStats(stats.ByStatus, string(source.Status))
	}
	return stats, nil
}

func cloneSource(source domain.SourceCatalog) domain.SourceCatalog {
	source.SourceConfig = cloneMap(source.SourceConfig)
	source.RateLimitPolicy = cloneMap(source.RateLimitPolicy)
	return source
}

func normalizeInMemorySource(source domain.SourceCatalog) domain.SourceCatalog {
	if source.SourceLevel == "" {
		source.SourceLevel = "secondary"
	}
	if source.AuthType == "" {
		source.AuthType = "none"
	}
	if source.Status == "" {
		source.Status = domain.SourceCatalogStatusActive
	}
	if source.SourceConfig == nil {
		source.SourceConfig = map[string]any{}
	}
	if source.RateLimitPolicy == nil {
		source.RateLimitPolicy = map[string]any{}
	}
	return source
}

func newSourceCatalogStats() SourceCatalogStats {
	return SourceCatalogStats{
		ByProviderKey:   map[string]int{},
		ByIngestChannel: map[string]int{},
		BySourceType:    map[string]int{},
		ByUsagePolicy:   map[string]int{},
		ByStatus:        map[string]int{},
	}
}

func incrementStats(counts map[string]int, key string) {
	if key == "" {
		key = "unknown"
	}
	counts[key]++
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
