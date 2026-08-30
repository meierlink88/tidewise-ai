package evidence

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type Store struct{ db *sql.DB }

var categoryCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

const categorySelect = `
SELECT category.id, category.code, category.name, category.description, category.created_at
FROM evidence_categories AS category`

const categoryCatalogQuery = categorySelect + ` ORDER BY category.code COLLATE "C", category.id COLLATE "C"`

func NewStore(db *sql.DB) (Store, error) {
	if db == nil {
		return Store{}, errors.New("Evidence database is required")
	}
	return Store{db: db}, nil
}

func (s Store) ListCategories(ctx context.Context) ([]evidencebiz.Category, error) {
	rows, err := s.db.QueryContext(ctx, categoryCatalogQuery)
	if err != nil {
		return nil, fmt.Errorf("query Evidence Category Catalog: %w", err)
	}
	defer rows.Close()
	categories := make([]evidencebiz.Category, 0)
	seenIDs := make(map[evidencebiz.CategoryID]struct{})
	seenCodes := make(map[string]struct{})
	for rows.Next() {
		var category evidencebiz.Category
		var id string
		if err := rows.Scan(&id, &category.Code, &category.Name, &category.Description, &category.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan Evidence Category Catalog: %w", err)
		}
		category.ID = evidencebiz.CategoryID(id)
		if err := validateStoredCategory(&category); err != nil {
			return nil, fmt.Errorf("read Evidence Category Catalog invariant: %w", err)
		}
		if _, duplicate := seenIDs[category.ID]; duplicate {
			return nil, persistedInvariant("Evidence Category Catalog", "id", "query returned a duplicate identity")
		}
		if _, duplicate := seenCodes[category.Code]; duplicate {
			return nil, persistedInvariant("Evidence Category Catalog", "code", "query returned a duplicate code")
		}
		if len(categories) > 0 {
			previous := categories[len(categories)-1]
			if previous.Code > category.Code || previous.Code == category.Code && previous.ID >= category.ID {
				return nil, persistedInvariant("Evidence Category Catalog", "order", "query result is not ordered by code and ID")
			}
		}
		seenIDs[category.ID] = struct{}{}
		seenCodes[category.Code] = struct{}{}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Evidence Category Catalog: %w", err)
	}
	if len(categories) == 0 {
		return nil, persistedInvariant("Evidence Category Catalog", "categories", "catalog is empty")
	}
	return categories, nil
}

const evidenceListWhere = `
FROM evidences AS evidence
JOIN raw_evidences AS raw ON raw.id = evidence.raw_evidence_id
WHERE ($1 = '' OR strpos(lower(raw.title), lower($1)) > 0)
  AND ($2 = '' OR strpos(lower(evidence.summary), lower($2)) > 0)
  AND ($3 = '' OR EXISTS (
      SELECT 1 FROM raw_evidence_category_links AS selected_category
      WHERE selected_category.raw_evidence_id = raw.id AND selected_category.category_id = $3
  ))
  AND ($4 = '' OR raw.source_id = $4)
  AND ($5 = '' OR strpos(lower(raw.source_name), lower($5)) > 0)
  AND ($6 = '' OR raw.source_level = $6)
  AND ($7::boolean IS NULL OR evidence.is_split = $7)
  AND ($8::timestamptz IS NULL OR raw.published_at >= $8)
  AND ($9::timestamptz IS NULL OR raw.published_at <= $9)
  AND ($10::timestamptz IS NULL OR raw.collected_at >= $10)
  AND ($11::timestamptz IS NULL OR raw.collected_at <= $11)`

func (s Store) ListEvidence(ctx context.Context, filter evidencebiz.EvidenceListFilter) (evidencebiz.EvidencePage, error) {
	args := []any{
		filter.Title, filter.Summary, string(filter.CategoryID), filter.SourceID, filter.SourceName, string(filter.SourceLevel),
		optionalBoolValue(filter.IsSplit), optionalTimeValue(filter.PublishedFrom), optionalTimeValue(filter.PublishedTo),
		optionalTimeValue(filter.CollectedFrom), optionalTimeValue(filter.CollectedTo),
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)`+evidenceListWhere, args...).Scan(&total); err != nil {
		return evidencebiz.EvidencePage{}, fmt.Errorf("count Evidence: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT evidence.id, evidence.raw_evidence_id, evidence.is_split, evidence.summary, evidence.semantic,
       raw.title, raw.source_id, raw.source_name, raw.source_level, raw.source_url, raw.is_original, raw.quoted_source_name,
       raw.published_at, raw.collected_at, array_to_json(evidence.keywords),
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'id', category.id,
               'code', category.code,
               'name', category.name,
               'description', category.description,
               'created_at', category.created_at
           ) ORDER BY category.code COLLATE "C", category.id COLLATE "C")
           FROM raw_evidence_category_links AS category_link
           JOIN evidence_categories AS category ON category.id = category_link.category_id
           WHERE category_link.raw_evidence_id = raw.id
       ), '[]'::jsonb) AS categories
`+evidenceListWhere+`
ORDER BY raw.published_at DESC NULLS LAST, raw.collected_at DESC, evidence.id COLLATE "C"
LIMIT $12 OFFSET $13`, append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)...)
	if err != nil {
		return evidencebiz.EvidencePage{}, fmt.Errorf("query Evidence: %w", err)
	}
	defer rows.Close()
	items := make([]evidencebiz.EvidenceListItem, 0)
	seen := make(map[string]struct{})
	for rows.Next() {
		item, scanErr := scanEvidenceListItem(rows)
		if scanErr != nil {
			return evidencebiz.EvidencePage{}, scanErr
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return evidencebiz.EvidencePage{}, persistedInvariant("Evidence list", "id", "query returned a duplicate identity")
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return evidencebiz.EvidencePage{}, fmt.Errorf("iterate Evidence: %w", err)
	}
	if total < 0 || len(items) > filter.PageSize || len(items) > total {
		return evidencebiz.EvidencePage{}, persistedInvariant("Evidence list", "page", "query returned an invalid page")
	}
	return evidencebiz.EvidencePage{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

type evidenceListScanner interface{ Scan(...any) error }

func scanEvidenceListItem(scanner evidenceListScanner) (evidencebiz.EvidenceListItem, error) {
	var item evidencebiz.EvidenceListItem
	var title sql.NullString
	var quotedSourceName sql.NullString
	var publishedAt sql.NullTime
	var semanticJSON []byte
	var keywordsJSON []byte
	var categoriesJSON []byte
	if err := scanner.Scan(
		&item.ID, &item.RawEvidenceID, &item.IsSplit, &item.Summary, &semanticJSON, &title,
		&item.SourceID, &item.SourceName, &item.SourceLevel, &item.SourceURL, &item.IsOriginal, &quotedSourceName,
		&publishedAt, &item.CollectedAt, &keywordsJSON, &categoriesJSON,
	); err != nil {
		return evidencebiz.EvidenceListItem{}, fmt.Errorf("scan Evidence list: %w", err)
	}
	if title.Valid {
		item.Title = &title.String
	}
	if publishedAt.Valid {
		value := publishedAt.Time.UTC()
		item.PublishedAt = &value
	}
	if quotedSourceName.Valid {
		item.QuotedSourceName = &quotedSourceName.String
	}
	item.CollectedAt = item.CollectedAt.UTC()
	if err := decodeStoredSemantic(semanticJSON, &item.Semantic); err != nil {
		return evidencebiz.EvidenceListItem{}, fmt.Errorf("decode Evidence list semantic: %w", err)
	}
	if err := json.Unmarshal(keywordsJSON, &item.Keywords); err != nil || item.Keywords == nil {
		return evidencebiz.EvidenceListItem{}, persistedInvariant("Evidence list", "keywords", "value is not an array")
	}
	categories, err := decodeEvidenceListCategories(categoriesJSON)
	if err != nil {
		return evidencebiz.EvidenceListItem{}, fmt.Errorf("decode Evidence list categories: %w", err)
	}
	item.Categories = categories
	if err := validateEvidenceListItem(item); err != nil {
		return evidencebiz.EvidenceListItem{}, fmt.Errorf("read Evidence list invariant: %w", err)
	}
	return item, nil
}

type evidenceListCategoryJSON struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func decodeEvidenceListCategories(payload []byte) ([]evidencebiz.Category, error) {
	var values []evidenceListCategoryJSON
	if err := json.Unmarshal(payload, &values); err != nil || values == nil {
		return nil, persistedInvariant("Evidence list", "categories", "value is not an array")
	}
	categories := make([]evidencebiz.Category, len(values))
	for index, value := range values {
		category := evidencebiz.Category{
			ID: evidencebiz.CategoryID(value.ID), Code: value.Code, Name: value.Name,
			Description: value.Description, CreatedAt: value.CreatedAt,
		}
		if err := validateStoredCategory(&category); err != nil {
			return nil, err
		}
		if index > 0 {
			previous := categories[index-1]
			if previous.Code > category.Code || previous.Code == category.Code && previous.ID >= category.ID {
				return nil, persistedInvariant("Evidence list", "categories", "collection is not uniquely ordered")
			}
		}
		categories[index] = category
	}
	return categories, nil
}

func validateEvidenceListItem(item evidencebiz.EvidenceListItem) error {
	if err := validateStoredRequired("Evidence list", "id", item.ID, 39); err != nil {
		return err
	}
	if err := validateStoredRequired("Evidence list", "raw_evidence_id", item.RawEvidenceID, 39); err != nil {
		return err
	}
	if err := validateStoredRequired("Evidence list", "summary", item.Summary, 200); err != nil {
		return err
	}
	if err := validateStoredRequired("Evidence list", "source_id", item.SourceID, 32); err != nil {
		return err
	}
	if err := validateStoredRequired("Evidence list", "source_name", item.SourceName, 100); err != nil {
		return err
	}
	if err := validateStoredRequired("Evidence list", "source_url", item.SourceURL, 2048); err != nil {
		return err
	}
	if !coreIDIsEvidence(item.ID) || !coreIDIsRawEvidence(item.RawEvidenceID) {
		return persistedInvariant("Evidence list", "id", "value is not a stable domain identity")
	}
	if err := validateStoredOptional("Evidence list", "title", item.Title, 500); err != nil {
		return err
	}
	switch item.SourceLevel {
	case evidencebiz.SourceLevelOfficial, evidencebiz.SourceLevelWire, evidencebiz.SourceLevelMedia, evidencebiz.SourceLevelSocial:
	default:
		return persistedInvariant("Evidence list", "source_level", "value is not supported")
	}
	parsedURL, err := url.Parse(item.SourceURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return persistedInvariant("Evidence list", "source_url", "value is not an absolute HTTP(S) URL")
	}
	if item.IsOriginal && item.QuotedSourceName != nil {
		return persistedInvariant("Evidence list", "is_original", "original content declares a quoted source")
	}
	if !item.IsOriginal && (item.QuotedSourceName == nil || strings.TrimSpace(*item.QuotedSourceName) == "") {
		return persistedInvariant("Evidence list", "quoted_source_name", "reposted content has no quoted source name")
	}
	if err := validateStoredOptional("Evidence list", "quoted_source_name", item.QuotedSourceName, 100); err != nil {
		return err
	}
	if err := validateStoredEvidence(&evidencebiz.StoredEvidence{Evidence: evidencebiz.Evidence{ID: item.ID, Summary: item.Summary, Keywords: item.Keywords, Semantic: item.Semantic}, RawEvidenceID: item.RawEvidenceID, IsSplit: item.IsSplit}); err != nil {
		return err
	}
	if item.CollectedAt.IsZero() || item.Categories == nil {
		return persistedInvariant("Evidence list", "time", "required projection value is missing")
	}
	return nil
}

func coreIDIsEvidence(value string) bool {
	return coreid.Is(value, coreid.Evidence)
}

func coreIDIsRawEvidence(value string) bool {
	return coreid.Is(value, coreid.RawEvidence)
}

func optionalBoolValue(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func optionalTimeValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

type persistedInvariantError struct {
	resource string
	field    string
	reason   string
}

func (e *persistedInvariantError) Error() string {
	return fmt.Sprintf("persisted %s field %s violates invariants: %s", e.resource, e.field, e.reason)
}

func persistedInvariant(resource, field, reason string) error {
	return &persistedInvariantError{resource: resource, field: field, reason: reason}
}

func validateStoredRawEvidence(record *evidencebiz.StoredRawEvidence, expectedID string) error {
	if err := validateStoredRawEvidenceBase(record, expectedID); err != nil {
		return err
	}
	const resource = "Raw Evidence"
	if record.CategoryIDs == nil || record.Categories == nil {
		return persistedInvariant(resource, "categories", "collection is null")
	}
	if len(record.CategoryIDs) != len(record.Categories) {
		return persistedInvariant(resource, "categories", "identity and value counts differ")
	}
	for index, category := range record.Categories {
		if err := validateStoredCategory(&category); err != nil {
			return err
		}
		if record.CategoryIDs[index] != category.ID || index > 0 && record.CategoryIDs[index-1] >= category.ID {
			return persistedInvariant(resource, "categories", "collection is not uniquely ordered by category ID")
		}
	}
	return nil
}

func validateStoredRawEvidenceBase(record *evidencebiz.StoredRawEvidence, expectedID string) error {
	const resource = "Raw Evidence"
	if record.ID != expectedID {
		return persistedInvariant(resource, "id", "query identity does not match the stored row")
	}
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{name: "id", value: record.ID, max: 39},
		{name: "source_id", value: record.SourceID, max: 32},
		{name: "source_name", value: record.SourceName, max: 100},
		{name: "source_url", value: record.SourceURL},
		{name: "raw_text", value: record.RawText},
	} {
		if err := validateStoredRequired(resource, field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if err := validateStoredOptional(resource, "quoted_source_id", record.QuotedSourceID, 32); err != nil {
		return err
	}
	if err := validateStoredOptional(resource, "quoted_source_name", record.QuotedSourceName, 100); err != nil {
		return err
	}
	if err := validateStoredOptional(resource, "title", record.Title, 500); err != nil {
		return err
	}
	switch record.SourceLevel {
	case evidencebiz.SourceLevelOfficial, evidencebiz.SourceLevelWire,
		evidencebiz.SourceLevelMedia, evidencebiz.SourceLevelSocial:
	default:
		return persistedInvariant(resource, "source_level", "value is not a supported source level")
	}
	parsedURL, err := url.Parse(record.SourceURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return persistedInvariant(resource, "source_url", "value is not an absolute HTTP(S) URL")
	}
	if record.IsOriginal && (record.QuotedSourceID != nil || record.QuotedSourceName != nil) {
		return persistedInvariant(resource, "is_original", "original content declares a quoted source")
	}
	if !record.IsOriginal && (record.QuotedSourceName == nil || strings.TrimSpace(*record.QuotedSourceName) == "") {
		return persistedInvariant(resource, "quoted_source_name", "reposted content has no quoted source name")
	}
	if record.CollectedAt.IsZero() {
		return persistedInvariant(resource, "collected_at", "timestamp is zero")
	}
	record.CollectedAt = record.CollectedAt.UTC()
	if record.PublishedAt != nil {
		value := record.PublishedAt.UTC()
		record.PublishedAt = &value
	}
	digest := sha256.Sum256([]byte(record.RawText))
	if record.ContentHash != hex.EncodeToString(digest[:]) {
		return persistedInvariant(resource, "content_hash", "value does not match raw_text")
	}
	return nil
}

func validateStoredCategory(record *evidencebiz.Category) error {
	const resource = "Evidence Category"
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{name: "id", value: string(record.ID), max: 39},
		{name: "code", value: record.Code, max: 50},
		{name: "name", value: record.Name, max: 50},
		{name: "description", value: record.Description},
	} {
		if err := validateStoredRequired(resource, field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if !record.ID.IsValid() {
		return persistedInvariant(resource, "id", "value is not a stable EVC identity")
	}
	if !categoryCodePattern.MatchString(record.Code) {
		return persistedInvariant(resource, "code", "value is not a stable machine code")
	}
	if record.CreatedAt.IsZero() {
		return persistedInvariant(resource, "created_at", "timestamp is zero")
	}
	record.CreatedAt = record.CreatedAt.UTC()
	return nil
}

func validateStoredEvidence(record *evidencebiz.StoredEvidence) error {
	const resource = "Evidence"
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{name: "id", value: record.ID, max: 39},
		{name: "raw_evidence_id", value: record.RawEvidenceID, max: 39},
		{name: "summary", value: record.Summary, max: 200},
		{name: "semantic.action", value: record.Semantic.Action, max: 200},
	} {
		if err := validateStoredRequired(resource, field.name, field.value, field.max); err != nil {
			return err
		}
	}
	validateStoredSemanticCollections := func(path string, values []string, min, max, maxRunes int) error {
		if values == nil || len(values) < min || len(values) > max {
			return persistedInvariant(resource, path, "collection is incomplete")
		}
		seen := make(map[string]struct{}, len(values))
		for _, value := range values {
			if strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > maxRunes {
				return persistedInvariant(resource, path, "collection contains a blank value")
			}
			if _, duplicate := seen[value]; duplicate {
				return persistedInvariant(resource, path, "collection contains a duplicate value")
			}
			seen[value] = struct{}{}
		}
		return nil
	}
	if err := validateStoredSemanticCollections("semantic.actors", record.Semantic.Actors, 1, 20, 100); err != nil {
		return err
	}
	if err := validateStoredSemanticCollections("semantic.objects", record.Semantic.Objects, 1, 20, 200); err != nil {
		return err
	}
	if err := validateStoredSemanticCollections("semantic.jurisdictions", record.Semantic.Jurisdictions, 0, 20, 100); err != nil {
		return err
	}
	switch record.Semantic.Stage {
	case evidencebiz.EvidenceStageOccurred, evidencebiz.EvidenceStageAnnounced, evidencebiz.EvidenceStageEffective,
		evidencebiz.EvidenceStageImplemented, evidencebiz.EvidenceStageUpdated, evidencebiz.EvidenceStageSuspended,
		evidencebiz.EvidenceStageTerminated, evidencebiz.EvidenceStageExpected:
	default:
		return persistedInvariant(resource, "semantic.stage", "value is not supported")
	}
	switch record.Semantic.Modality {
	case evidencebiz.EvidenceModalityFact, evidencebiz.EvidenceModalityPlan, evidencebiz.EvidenceModalitySpec:
	default:
		return persistedInvariant(resource, "semantic.modality", "value is not supported")
	}
	switch record.Semantic.Time.Precision {
	case evidencebiz.EvidenceTimeInstant, evidencebiz.EvidenceTimeDay, evidencebiz.EvidenceTimeRange,
		evidencebiz.EvidenceTimeMonth, evidencebiz.EvidenceTimeQuarter, evidencebiz.EvidenceTimeYear,
		evidencebiz.EvidenceTimeUnknown:
	default:
		return persistedInvariant(resource, "semantic.time.precision", "value is not supported")
	}
	if (record.Semantic.Time.StartAt == nil) != (record.Semantic.Time.EndAt == nil) {
		return persistedInvariant(resource, "semantic.time", "bounds must both be present or both be null")
	}
	if record.Semantic.Time.StartAt != nil && record.Semantic.Time.EndAt != nil {
		_, startOffset := record.Semantic.Time.StartAt.Zone()
		_, endOffset := record.Semantic.Time.EndAt.Zone()
		if startOffset != 0 || endOffset != 0 || record.Semantic.Time.StartAt.After(*record.Semantic.Time.EndAt) {
			return persistedInvariant(resource, "semantic.time", "bounds are not ordered UTC timestamps")
		}
	}
	for _, value := range []struct {
		path  string
		value *string
		max   int
	}{
		{path: "semantic.reason", value: record.Semantic.Reason, max: 500},
		{path: "semantic.method", value: record.Semantic.Method, max: 500},
		{path: "semantic.time.raw", value: record.Semantic.Time.Raw, max: 200},
	} {
		if err := validateStoredOptional(resource, value.path, value.value, value.max); err != nil {
			return err
		}
	}
	if record.Semantic.Attribution == nil {
		return persistedInvariant(resource, "semantic.attribution", "object is null")
	}
	for _, value := range []struct {
		path  string
		value *string
	}{
		{path: "semantic.attribution.reported_by", value: record.Semantic.Attribution.ReportedBy},
		{path: "semantic.attribution.claimed_by", value: record.Semantic.Attribution.ClaimedBy},
	} {
		if err := validateStoredOptional(resource, value.path, value.value, 100); err != nil {
			return err
		}
	}
	if record.Semantic.Metrics == nil {
		return persistedInvariant(resource, "semantic.metrics", "collection is null")
	}
	seenMetrics := make(map[string]struct{}, len(record.Semantic.Metrics))
	for _, metric := range record.Semantic.Metrics {
		if strings.TrimSpace(metric.Name) == "" || utf8.RuneCountInString(metric.Name) > 100 || metric.Value == nil && metric.Change == nil {
			return persistedInvariant(resource, "semantic.metrics", "metric is incomplete")
		}
		for _, value := range []struct {
			path  string
			value *string
			max   int
		}{
			{path: "value", value: metric.Value, max: 100},
			{path: "unit", value: metric.Unit, max: 50},
			{path: "change", value: metric.Change, max: 100},
			{path: "period", value: metric.Period, max: 100},
		} {
			if err := validateStoredOptional(resource, "semantic.metrics."+value.path, value.value, value.max); err != nil {
				return err
			}
		}
		identity := strings.ToLower(strings.TrimSpace(metric.Name)) + "\x00"
		if metric.Period != nil {
			identity += strings.ToLower(strings.TrimSpace(*metric.Period))
		}
		if _, duplicate := seenMetrics[identity]; duplicate {
			return persistedInvariant(resource, "semantic.metrics", "metric identity is duplicated")
		}
		seenMetrics[identity] = struct{}{}
	}
	if record.Keywords == nil || len(record.Keywords) < 1 || len(record.Keywords) > 5 {
		return persistedInvariant(resource, "keywords", "collection must contain one to five values")
	}
	seenKeywords := make(map[string]struct{}, len(record.Keywords))
	for _, keyword := range record.Keywords {
		if strings.TrimSpace(keyword) == "" || utf8.RuneCountInString(keyword) > 6 {
			return persistedInvariant(resource, "keywords", "collection contains an invalid value")
		}
		if _, duplicate := seenKeywords[keyword]; duplicate {
			return persistedInvariant(resource, "keywords", "collection contains a duplicate value")
		}
		seenKeywords[keyword] = struct{}{}
	}
	return nil
}

func validateStoredEvidenceSet(expectedRawEvidenceID string, records []evidencebiz.StoredEvidence) error {
	if len(records) == 0 {
		return nil
	}
	expectedSplit := len(records) > 1
	for position, record := range records {
		if record.RawEvidenceID != expectedRawEvidenceID {
			return persistedInvariant("Evidence", "raw_evidence_id", "query identity does not match the stored row")
		}
		if position > 0 && records[position-1].ID >= record.ID {
			return persistedInvariant("Evidence", "id", "stored set is not uniquely ordered by ID")
		}
		if record.IsSplit != expectedSplit {
			return persistedInvariant("Evidence", "is_split", "value does not match the stored set cardinality")
		}
	}
	return nil
}

func validateStoredEvidenceIdentities(expectedIDs []string, records []evidencebiz.StoredEvidence) error {
	expected := make(map[string]struct{}, len(expectedIDs))
	for _, id := range expectedIDs {
		expected[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		if _, ok := expected[record.ID]; !ok {
			return persistedInvariant("Evidence", "id", "query returned an unrequested identity")
		}
		if _, ok := seen[record.ID]; ok {
			return persistedInvariant("Evidence", "id", "query returned a duplicate identity")
		}
		seen[record.ID] = struct{}{}
	}
	return nil
}

func validateStoredRequired(resource, field, value string, max int) error {
	if strings.TrimSpace(value) == "" {
		return persistedInvariant(resource, field, "value is blank")
	}
	if max > 0 && len([]rune(value)) > max {
		return persistedInvariant(resource, field, "value exceeds the storage contract")
	}
	return nil
}

func validateStoredOptional(resource, field string, value *string, max int) error {
	if value == nil {
		return nil
	}
	return validateStoredRequired(resource, field, *value, max)
}

var _ evidencebiz.Store = Store{}
