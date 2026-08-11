package event

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/event"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (Store, error) {
	if db == nil {
		return Store{}, errors.New("Event database is required")
	}
	return Store{db: db}, nil
}

func (s Store) ListActiveTags(ctx context.Context) ([]eventbiz.EventTag, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id::text, tag_kind, code, name, is_active
FROM event_tag_defs
WHERE is_active
ORDER BY tag_kind, code, id`)
	if err != nil {
		return nil, fmt.Errorf("query active Event Tag Catalog: %w", err)
	}
	defer rows.Close()
	tags := make([]eventbiz.EventTag, 0)
	for rows.Next() {
		var tag eventbiz.EventTag
		if err := rows.Scan(&tag.ID, &tag.Kind, &tag.Code, &tag.Name, &tag.Active); err != nil {
			return nil, fmt.Errorf("scan active Event Tag Catalog: %w", err)
		}
		if err := validateStoredEventTag(tag, true); err != nil {
			return nil, fmt.Errorf("read active Event Tag invariant: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read active Event Tag Catalog: %w", err)
	}
	return tags, nil
}

func (s Store) ListEvents(ctx context.Context, filter eventbiz.EventListFilter) (eventbiz.EventStorePage, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	var total int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM events
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR event_status = $2)
  AND ($3 = '' OR fact_status = $3)
  AND ($4::timestamptz IS NULL OR event_time >= $4)
  AND ($5::timestamptz IS NULL OR event_time <= $5)
  AND ($6::timestamptz IS NULL OR first_seen_at >= $6)
  AND ($7::timestamptz IS NULL OR first_seen_at <= $7)`,
		filter.Title, string(filter.EventStatus), string(filter.FactStatus), nullableTime(filter.EventTimeFrom),
		nullableTime(filter.EventTimeTo), nullableTime(filter.FirstSeenFrom), nullableTime(filter.FirstSeenTo),
	).Scan(&total); err != nil {
		return eventbiz.EventStorePage{}, fmt.Errorf("count Events: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, title, summary, event_time, first_seen_at, knowable_at,
       event_status, fact_status, dedupe_key
FROM events
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR event_status = $2)
  AND ($3 = '' OR fact_status = $3)
  AND ($4::timestamptz IS NULL OR event_time >= $4)
  AND ($5::timestamptz IS NULL OR event_time <= $5)
  AND ($6::timestamptz IS NULL OR first_seen_at >= $6)
  AND ($7::timestamptz IS NULL OR first_seen_at <= $7)
ORDER BY first_seen_at DESC, event_time DESC NULLS LAST, id
LIMIT $8 OFFSET $9`,
		filter.Title, string(filter.EventStatus), string(filter.FactStatus), nullableTime(filter.EventTimeFrom),
		nullableTime(filter.EventTimeTo), nullableTime(filter.FirstSeenFrom), nullableTime(filter.FirstSeenTo),
		pageSize, (page-1)*pageSize,
	)
	if err != nil {
		return eventbiz.EventStorePage{}, fmt.Errorf("query Events: %w", err)
	}
	defer rows.Close()

	items := make([]eventbiz.EventListItem, 0)
	for rows.Next() {
		item, err := scanEvent(rows)
		if err != nil {
			return eventbiz.EventStorePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return eventbiz.EventStorePage{}, fmt.Errorf("iterate Events: %w", err)
	}
	return eventbiz.EventStorePage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

type rowScanner interface{ Scan(...any) error }

func scanEvent(scanner rowScanner) (eventbiz.EventListItem, error) {
	var item eventbiz.EventListItem
	var eventTime sql.NullTime
	var knowableAt sql.NullTime
	if err := scanner.Scan(
		&item.ID, &item.Title, &item.Summary, &eventTime, &item.FirstSeenAt, &knowableAt,
		&item.EventStatus, &item.FactStatus, &item.DedupeKey,
	); err != nil {
		return eventbiz.EventListItem{}, fmt.Errorf("scan Event: %w", err)
	}
	if eventTime.Valid {
		value := eventTime.Time.UTC()
		item.EventTime = &value
	}
	item.FirstSeenAt = item.FirstSeenAt.UTC()
	if knowableAt.Valid {
		value := knowableAt.Time.UTC()
		item.KnowableAt = &value
	}
	if err := validateStoredEvent(item); err != nil {
		return eventbiz.EventListItem{}, fmt.Errorf("read Event invariant: %w", err)
	}
	return item, nil
}

func validateStoredEvent(item eventbiz.EventListItem) error {
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Title) == "" ||
		strings.TrimSpace(item.DedupeKey) == "" || item.FirstSeenAt.IsZero() {
		return errors.New("required identity, title, dedupe key or first-seen time is missing")
	}
	switch item.EventStatus {
	case eventbiz.EventStatusCandidate, eventbiz.EventStatusConfirmed, eventbiz.EventStatusRejected:
	default:
		return errors.New("Event status is invalid")
	}
	switch item.FactStatus {
	case eventbiz.FactStatusUnverified, eventbiz.FactStatusVerified, eventbiz.FactStatusDisputed:
	default:
		return errors.New("Fact status is invalid")
	}
	return nil
}

func validateStoredEventTag(tag eventbiz.EventTag, requireActive bool) error {
	if strings.TrimSpace(tag.ID) == "" || strings.TrimSpace(tag.Code) == "" || strings.TrimSpace(tag.Name) == "" {
		return errors.New("required Event Tag field is missing")
	}
	if tag.Kind != eventbiz.EventTagKindNewsCategory && tag.Kind != eventbiz.EventTagKindIndexCategory {
		return errors.New("Event Tag kind is invalid")
	}
	if requireActive && !tag.Active {
		return errors.New("Event Tag is inactive")
	}
	return nil
}

func validateStoredEvidenceRecord(record eventbiz.StoredEventEvidenceRecord, artifactID string) error {
	if record.ArtifactID != artifactID || strings.TrimSpace(record.ID) == "" ||
		!validSHA256(record.ContentSHA256) || strings.TrimSpace(record.SourceRef) == "" ||
		strings.TrimSpace(record.SourceName) == "" || strings.TrimSpace(record.SourceType) == "" ||
		strings.TrimSpace(record.Title) == "" || record.CollectedAt.IsZero() {
		return errors.New("Event Evidence Record violates persisted invariants")
	}
	return nil
}

func validateStoredPublicationEvent(record eventbiz.StoredEvent, dedupeKey string) error {
	if record.DedupeKey != dedupeKey || strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.Title) == "" ||
		strings.TrimSpace(record.FactualSummary) == "" || record.FirstSeenAt.IsZero() || record.KnowableAt.IsZero() {
		return errors.New("Event violates persisted publication invariants")
	}
	if record.EventStatus != eventbiz.EventStatusConfirmed || record.FactStatus != eventbiz.FactStatusVerified {
		return errors.New("Event publication status is invalid")
	}
	if err := eventbiz.ValidateFactPayload(record.FactPayload); err != nil {
		return fmt.Errorf("Event fact payload is invalid: %w", err)
	}
	return nil
}

func validateStoredEvidenceLink(record eventbiz.StoredEventEvidenceLink, eventID, rawDocumentID string) error {
	if record.EventID != eventID || record.RawDocumentID != rawDocumentID || strings.TrimSpace(record.ID) == "" ||
		strings.TrimSpace(record.SourceLevel) == "" || strings.TrimSpace(record.EvidenceStatement) == "" ||
		!validSHA256(record.EvidenceHash) {
		return errors.New("Event Evidence Link violates persisted invariants")
	}
	if record.SourceLevel != eventbiz.EventSourceLevelPrimary && record.SourceLevel != eventbiz.EventSourceLevelSecondary {
		return errors.New("Event Evidence Link source level is invalid")
	}
	expectedHash := sha256.Sum256([]byte(record.EvidenceStatement))
	if record.EvidenceHash != hex.EncodeToString(expectedHash[:]) {
		return errors.New("Event Evidence Link hash does not match its statement")
	}
	allowedFields := map[string]struct{}{
		eventbiz.EventFieldTitle: {}, eventbiz.EventFieldFactualSummary: {},
		eventbiz.EventFieldOccurredAt: {}, eventbiz.EventFieldFactPayload: {},
	}
	seenFields := make(map[string]struct{}, len(record.SupportsFields))
	for _, field := range record.SupportsFields {
		if _, allowed := allowedFields[field]; !allowed {
			return errors.New("Event Evidence Link supports field is invalid")
		}
		if _, duplicate := seenFields[field]; duplicate {
			return errors.New("Event Evidence Link supports fields contain a duplicate")
		}
		seenFields[field] = struct{}{}
	}
	if record.EvidenceRelation != eventbiz.EvidenceRelationSupports &&
		record.EvidenceRelation != eventbiz.EvidenceRelationContradicts &&
		record.EvidenceRelation != eventbiz.EvidenceRelationContext {
		return errors.New("Event Evidence Link relation is invalid")
	}
	link := eventbiz.EventEvidenceLink{EvidenceRelation: record.EvidenceRelation, SupportsFields: record.SupportsFields}
	return link.Validate()
}

func validateStoredTagAssignment(record eventbiz.StoredEventTagAssignment, eventID, tagID string) error {
	if record.EventID != eventID || record.TagID != tagID || strings.TrimSpace(record.ID) == "" ||
		strings.TrimSpace(record.AssignSource) == "" || strings.TrimSpace(record.Confidence) == "" ||
		strings.TrimSpace(record.AssignmentReason) == "" || record.ReviewStatus != eventbiz.ReviewStatusApproved {
		return errors.New("Event Tag Assignment violates persisted invariants")
	}
	if record.AssignSource != eventbiz.TagAssignSourceAI && record.AssignSource != eventbiz.TagAssignSourceRule {
		return errors.New("Event Tag Assignment source is invalid")
	}
	confidence, valid := new(big.Rat).SetString(record.Confidence)
	if !valid || confidence.Sign() < 0 || confidence.Cmp(big.NewRat(1, 1)) > 0 {
		return errors.New("Event Tag Assignment confidence is invalid")
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return page, pageSize
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

var _ eventbiz.Store = Store{}
