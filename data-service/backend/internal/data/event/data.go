package event

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (Store, error) {
	if db == nil {
		return Store{}, errors.New("Event database is required")
	}
	return Store{db: db}, nil
}

func (s Store) CreateEvent(ctx context.Context, aggregate eventbiz.Aggregate) (resultErr error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Event create transaction: %w", err)
	}
	defer func() {
		if resultErr != nil {
			rollbackErr := tx.Rollback()
			if !errors.Is(rollbackErr, sql.ErrTxDone) {
				resultErr = errors.Join(resultErr, rollbackErr)
			}
		}
	}()
	semantic, err := json.Marshal(aggregate.Event.Semantic)
	if err != nil {
		return fmt.Errorf("encode Event semantic: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO events (
    id, title, summary, semantic, modality, occurred_at, announced_at, status
) VALUES (
    $1,$2,$3,$4,
    $4::jsonb ->> 'modality',
    ($4::jsonb #>> '{time,occurred_at}')::timestamptz,
    ($4::jsonb #>> '{time,announced_at}')::timestamptz,
    $5
)`,
		aggregate.Event.ID, aggregate.Event.Title, aggregate.Event.Summary, semantic,
		aggregate.Event.Status,
	); err != nil {
		return fmt.Errorf("insert Event %q: %w", aggregate.Event.ID, err)
	}
	for _, link := range aggregate.Evidence {
		if _, err := tx.ExecContext(ctx, `INSERT INTO event_evidence_links (
    id, event_id, evidence_id, contribution_weight
) VALUES ($1,$2,$3,$4)`, link.ID, link.EventID, link.EvidenceID, link.ContributionWeight); err != nil {
			return fmt.Errorf("insert Event Evidence Link %q: %w", link.ID, err)
		}
	}
	for _, link := range aggregate.Actors {
		if _, err := tx.ExecContext(ctx, `INSERT INTO event_actor_links (
    id, event_id, actor_id, actor_type, actor_name, relation_type, relation_strength, confidence
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, link.ID, link.EventID, link.ActorID,
			nullableActorType(link.ActorType), nullableString(link.ActorName), link.RelationType,
			nullableFloat(link.RelationStrength), link.Confidence); err != nil {
			return fmt.Errorf("insert Event Actor Link %q: %w", link.ID, err)
		}
	}
	for _, link := range aggregate.Assets {
		if _, err := tx.ExecContext(ctx, `INSERT INTO event_asset_links (
    id, event_id, asset_id, asset_type, asset_name, impact_direction, impact_magnitude
) VALUES ($1,$2,$3,$4,$5,$6,$7)`, link.ID, link.EventID, link.AssetID,
			nullableAssetType(link.AssetType), nullableString(link.AssetName), link.ImpactDirection,
			nullableFloat(link.ImpactMagnitude)); err != nil {
			return fmt.Errorf("insert Event Asset Link %q: %w", link.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Event create transaction: %w", err)
	}
	return nil
}

func encodeSemantic(semantic eventbiz.Semantic) ([]byte, error) {
	payload, err := json.Marshal(semantic)
	if err != nil {
		return nil, fmt.Errorf("encode Event semantic: %w", err)
	}
	return payload, nil
}

func (s Store) EventByID(ctx context.Context, eventID string) (eventbiz.Aggregate, error) {
	var aggregate eventbiz.Aggregate
	var semantic []byte
	var occurredAt, announcedAt sql.NullTime
	var modality eventbiz.Modality
	err := s.db.QueryRowContext(ctx, `SELECT id, title, summary, semantic, modality, occurred_at, announced_at, status
FROM events WHERE id = $1`, eventID).Scan(
		&aggregate.Event.ID, &aggregate.Event.Title, &aggregate.Event.Summary, &semantic,
		&modality, &occurredAt, &announcedAt, &aggregate.Event.Status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return eventbiz.Aggregate{}, eventbiz.ErrEventNotFound
	}
	if err != nil {
		return eventbiz.Aggregate{}, fmt.Errorf("read Event %q: %w", eventID, err)
	}
	if err := decodeSemantic(semantic, &aggregate.Event.Semantic); err != nil {
		return eventbiz.Aggregate{}, fmt.Errorf("read Event semantic invariant: %w", err)
	}
	if aggregate.Event.Semantic.Modality != modality ||
		!sameOptionalTime(aggregate.Event.Semantic.Time.OccurredAt, nullTimePointer(occurredAt)) ||
		!sameOptionalTime(aggregate.Event.Semantic.Time.AnnouncedAt, nullTimePointer(announcedAt)) {
		return eventbiz.Aggregate{}, errors.New("persisted Event time projections conflict with semantic")
	}
	if err := validatePersistedEvent(aggregate.Event); err != nil {
		return eventbiz.Aggregate{}, fmt.Errorf("read Event invariant: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id, event_id, evidence_id, contribution_weight
FROM event_evidence_links WHERE event_id = $1 ORDER BY id`, eventID)
	if err != nil {
		return eventbiz.Aggregate{}, fmt.Errorf("query Event Evidence Links: %w", err)
	}
	for rows.Next() {
		var link eventbiz.EvidenceLink
		if err := rows.Scan(&link.ID, &link.EventID, &link.EvidenceID, &link.ContributionWeight); err != nil {
			_ = rows.Close()
			return eventbiz.Aggregate{}, fmt.Errorf("scan Event Evidence Link: %w", err)
		}
		if err := validatePersistedEvidenceLink(link, eventID); err != nil {
			_ = rows.Close()
			return eventbiz.Aggregate{}, fmt.Errorf("read Event Evidence Link invariant: %w", err)
		}
		aggregate.Evidence = append(aggregate.Evidence, link)
	}
	if err := closeRows(rows); err != nil {
		return eventbiz.Aggregate{}, fmt.Errorf("read Event Evidence Links: %w", err)
	}

	actorRows, err := s.db.QueryContext(ctx, `SELECT id, event_id, actor_id, actor_type, actor_name,
       relation_type, relation_strength, confidence, created_at, updated_at
FROM event_actor_links WHERE event_id = $1 ORDER BY id`, eventID)
	if err != nil {
		return eventbiz.Aggregate{}, fmt.Errorf("query Event Actor Links: %w", err)
	}
	for actorRows.Next() {
		var link eventbiz.ActorLink
		var actorType, actorName sql.NullString
		var strength sql.NullFloat64
		if err := actorRows.Scan(&link.ID, &link.EventID, &link.ActorID, &actorType, &actorName,
			&link.RelationType, &strength, &link.Confidence, &link.CreatedAt, &link.UpdatedAt); err != nil {
			_ = actorRows.Close()
			return eventbiz.Aggregate{}, fmt.Errorf("scan Event Actor Link: %w", err)
		}
		if actorType.Valid {
			link.ActorType = eventbiz.ActorType(actorType.String)
		}
		link.ActorName = nullStringPointer(actorName)
		link.RelationStrength = nullFloatPointer(strength)
		link.CreatedAt = link.CreatedAt.UTC()
		link.UpdatedAt = link.UpdatedAt.UTC()
		if err := validatePersistedActorLink(link, eventID); err != nil {
			_ = actorRows.Close()
			return eventbiz.Aggregate{}, fmt.Errorf("read Event Actor Link invariant: %w", err)
		}
		aggregate.Actors = append(aggregate.Actors, link)
	}
	if err := closeRows(actorRows); err != nil {
		return eventbiz.Aggregate{}, fmt.Errorf("read Event Actor Links: %w", err)
	}

	assetRows, err := s.db.QueryContext(ctx, `SELECT id, event_id, asset_id, asset_type, asset_name,
       impact_direction, impact_magnitude
FROM event_asset_links WHERE event_id = $1 ORDER BY id`, eventID)
	if err != nil {
		return eventbiz.Aggregate{}, fmt.Errorf("query Event Asset Links: %w", err)
	}
	for assetRows.Next() {
		var link eventbiz.AssetLink
		var assetType, assetName sql.NullString
		var magnitude sql.NullFloat64
		if err := assetRows.Scan(&link.ID, &link.EventID, &link.AssetID, &assetType, &assetName,
			&link.ImpactDirection, &magnitude); err != nil {
			_ = assetRows.Close()
			return eventbiz.Aggregate{}, fmt.Errorf("scan Event Asset Link: %w", err)
		}
		if assetType.Valid {
			link.AssetType = eventbiz.AssetType(assetType.String)
		}
		link.AssetName = nullStringPointer(assetName)
		link.ImpactMagnitude = nullFloatPointer(magnitude)
		if err := validatePersistedAssetLink(link, eventID); err != nil {
			_ = assetRows.Close()
			return eventbiz.Aggregate{}, fmt.Errorf("read Event Asset Link invariant: %w", err)
		}
		aggregate.Assets = append(aggregate.Assets, link)
	}
	if err := closeRows(assetRows); err != nil {
		return eventbiz.Aggregate{}, fmt.Errorf("read Event Asset Links: %w", err)
	}
	if len(aggregate.Evidence) == 0 {
		return eventbiz.Aggregate{}, errors.New("persisted Event has no Evidence Link")
	}
	return aggregate, nil
}

func (s Store) ListEvents(ctx context.Context, filter eventbiz.EventListFilter) (eventbiz.EventStorePage, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	args := []any{
		filter.Title, string(filter.Modality), string(filter.Status), nullableTime(filter.OccurredFrom),
		nullableTime(filter.OccurredTo), nullableTime(filter.AnnouncedFrom), nullableTime(filter.AnnouncedTo),
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)
FROM events
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR modality = $2)
  AND ($3 = '' OR status = $3)
  AND ($4::timestamptz IS NULL OR occurred_at >= $4)
  AND ($5::timestamptz IS NULL OR occurred_at <= $5)
  AND ($6::timestamptz IS NULL OR announced_at >= $6)
  AND ($7::timestamptz IS NULL OR announced_at <= $7)`, args...).Scan(&total); err != nil {
		return eventbiz.EventStorePage{}, fmt.Errorf("count Events: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, summary, semantic, modality, occurred_at, announced_at, status
FROM events
WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
  AND ($2 = '' OR modality = $2)
  AND ($3 = '' OR status = $3)
  AND ($4::timestamptz IS NULL OR occurred_at >= $4)
  AND ($5::timestamptz IS NULL OR occurred_at <= $5)
  AND ($6::timestamptz IS NULL OR announced_at >= $6)
  AND ($7::timestamptz IS NULL OR announced_at <= $7)
ORDER BY occurred_at DESC NULLS LAST, announced_at DESC NULLS LAST, id
LIMIT $8 OFFSET $9`, append(args, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return eventbiz.EventStorePage{}, fmt.Errorf("query Events: %w", err)
	}
	items := make([]eventbiz.Event, 0)
	for rows.Next() {
		item, scanErr := scanEvent(rows)
		if scanErr != nil {
			_ = rows.Close()
			return eventbiz.EventStorePage{}, scanErr
		}
		items = append(items, item)
	}
	if err := closeRows(rows); err != nil {
		return eventbiz.EventStorePage{}, fmt.Errorf("read Events: %w", err)
	}
	return eventbiz.EventStorePage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

type rowScanner interface{ Scan(...any) error }

func scanEvent(scanner rowScanner) (eventbiz.Event, error) {
	var item eventbiz.Event
	var semantic []byte
	var occurredAt, announcedAt sql.NullTime
	var modality eventbiz.Modality
	if err := scanner.Scan(&item.ID, &item.Title, &item.Summary, &semantic, &modality,
		&occurredAt, &announcedAt, &item.Status); err != nil {
		return eventbiz.Event{}, fmt.Errorf("scan Event: %w", err)
	}
	if err := decodeSemantic(semantic, &item.Semantic); err != nil {
		return eventbiz.Event{}, fmt.Errorf("decode Event semantic: %w", err)
	}
	if item.Semantic.Modality != modality ||
		!sameOptionalTime(item.Semantic.Time.OccurredAt, nullTimePointer(occurredAt)) ||
		!sameOptionalTime(item.Semantic.Time.AnnouncedAt, nullTimePointer(announcedAt)) {
		return eventbiz.Event{}, errors.New("read Event projections conflict with semantic")
	}
	if err := validatePersistedEvent(item); err != nil {
		return eventbiz.Event{}, fmt.Errorf("read Event invariant: %w", err)
	}
	return item, nil
}

func validatePersistedEvent(event eventbiz.Event) error {
	if !coreid.Is(event.ID, coreid.Event) || strings.TrimSpace(event.Title) == "" || strings.TrimSpace(event.Summary) == "" {
		return errors.New("required Event field is missing")
	}
	if event.Semantic.Modality != eventbiz.ModalityFact && event.Semantic.Modality != eventbiz.ModalityPlan && event.Semantic.Modality != eventbiz.ModalitySpec {
		return errors.New("Event modality is invalid")
	}
	if event.Status != eventbiz.LifecycleStatusActive && event.Status != eventbiz.LifecycleStatusDeprecated && event.Status != eventbiz.LifecycleStatusArchived {
		return errors.New("Event status is invalid")
	}
	return nil
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	// PostgreSQL timestamptz stores microseconds, while the canonical semantic JSON
	// may retain Go's nanosecond precision. Compare at the projection's precision.
	return left.Round(time.Microsecond).Equal(right.Round(time.Microsecond))
}

func validatePersistedEvidenceLink(link eventbiz.EvidenceLink, eventID string) error {
	if !coreid.Is(link.ID, coreid.EventEvidenceLink) || link.EventID != eventID ||
		!coreid.Is(link.EvidenceID, coreid.Evidence) || !validUnitInterval(link.ContributionWeight) {
		return errors.New("Event Evidence Link is invalid")
	}
	return nil
}

func validatePersistedActorLink(link eventbiz.ActorLink, eventID string) error {
	if !coreid.Is(link.ID, coreid.EventActorLink) || link.EventID != eventID || strings.TrimSpace(link.ActorID) == "" ||
		(link.ActorType != "" && link.ActorType != eventbiz.ActorTypeCountry && link.ActorType != eventbiz.ActorTypePerson &&
			link.ActorType != eventbiz.ActorTypeOrganization && link.ActorType != eventbiz.ActorTypeCompany) ||
		(link.RelationType != eventbiz.ActorRelationMentions && link.RelationType != eventbiz.ActorRelationAffects &&
			link.RelationType != eventbiz.ActorRelationOriginatesFrom && link.RelationType != eventbiz.ActorRelationTargets) ||
		(link.RelationStrength != nil && !validUnitInterval(*link.RelationStrength)) ||
		!validUnitInterval(link.Confidence) || link.Confidence > 0.99 || link.CreatedAt.IsZero() || link.UpdatedAt.IsZero() {
		return errors.New("Event Actor Link is invalid")
	}
	return nil
}

func validatePersistedAssetLink(link eventbiz.AssetLink, eventID string) error {
	if !coreid.Is(link.ID, coreid.EventAssetLink) || link.EventID != eventID || strings.TrimSpace(link.AssetID) == "" ||
		(link.AssetType != "" && link.AssetType != eventbiz.AssetTypeSecurity && link.AssetType != eventbiz.AssetTypeCommodity &&
			link.AssetType != eventbiz.AssetTypeIndex && link.AssetType != eventbiz.AssetTypeRate &&
			link.AssetType != eventbiz.AssetTypeForex && link.AssetType != eventbiz.AssetTypeDerivative) ||
		(link.ImpactDirection != eventbiz.ImpactDirectionPositive && link.ImpactDirection != eventbiz.ImpactDirectionNegative &&
			link.ImpactDirection != eventbiz.ImpactDirectionNeutral) ||
		(link.ImpactMagnitude != nil && !validUnitInterval(*link.ImpactMagnitude)) {
		return errors.New("Event Asset Link is invalid")
	}
	return nil
}

func decodeSemantic(payload []byte, target *eventbiz.Semantic) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("semantic contains trailing JSON")
	}
	return nil
}

func closeRows(rows *sql.Rows) error {
	iterationErr := rows.Err()
	return errors.Join(iterationErr, rows.Close())
}

func validUnitInterval(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
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
	return value.UTC()
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullFloatPointer(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	result := value.Float64
	return &result
}

func nullableActorType(value eventbiz.ActorType) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableAssetType(value eventbiz.AssetType) any {
	if value == "" {
		return nil
	}
	return value
}

var _ eventbiz.Store = Store{}
