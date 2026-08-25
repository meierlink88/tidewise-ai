package event

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

import eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"

func (s Store) InEventPublicationTransaction(
	ctx context.Context,
	fn func(eventbiz.PublicationTransaction) error,
) (resultErr error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Event publication transaction: %w", err)
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
				panic(fmt.Errorf("Event publication panic (%v) and rollback failed: %w", failure, rollbackErr))
			}
			panic(failure)
		}
		if rollbackErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("roll back Event publication transaction: %w", rollbackErr))
		}
	}()
	if err := fn(&publicationTransaction{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Event publication transaction: %w", err)
	}
	committed = true
	return nil
}

type publicationTransaction struct{ tx *sql.Tx }

func (t *publicationTransaction) Lock(ctx context.Context, identity string) error {
	_, err := t.tx.ExecContext(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		"event-publication-v1:"+identity,
	)
	if err != nil {
		return fmt.Errorf("lock Event publication identity: %w", err)
	}
	return nil
}

func (t *publicationTransaction) Receipt(
	ctx context.Context,
	publisherSubject string,
	publicationKey string,
) (*eventbiz.PublicationReceipt, error) {
	var receipt eventbiz.PublicationReceipt
	err := t.tx.QueryRowContext(ctx, `SELECT id, publisher_subject, publication_key,
       payload_hash, event_id, published_at
FROM event_publication_receipts
WHERE publisher_subject = $1 AND publication_key = $2`, publisherSubject, publicationKey).Scan(
		&receipt.ID,
		&receipt.PublisherSubject,
		&receipt.PublicationKey,
		&receipt.PayloadHash,
		&receipt.EventID,
		&receipt.PublishedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Event publication receipt: %w", err)
	}
	return &receipt, nil
}

func (t *publicationTransaction) ExistingEvidenceIDs(
	ctx context.Context,
	ids []string,
) ([]string, error) {
	rows, err := t.tx.QueryContext(
		ctx,
		`SELECT id FROM evidences WHERE id = ANY($1::text[]) ORDER BY id`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("read Event Evidence references: %w", err)
	}
	defer rows.Close()
	result := make([]string, 0, len(ids))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan Event Evidence reference: %w", err)
		}
		result = append(result, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Event Evidence references: %w", err)
	}
	sort.Strings(result)
	return result, nil
}

func (t *publicationTransaction) InsertAggregate(
	ctx context.Context,
	aggregate eventbiz.Aggregate,
) error {
	return insertAggregate(ctx, t.tx, aggregate)
}

func (t *publicationTransaction) InsertReceipt(
	ctx context.Context,
	receipt eventbiz.PublicationReceipt,
) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO event_publication_receipts
    (id, publisher_subject, publication_key, payload_hash, event_id, published_at)
VALUES ($1,$2,$3,$4,$5,$6)`, receipt.ID, receipt.PublisherSubject,
		receipt.PublicationKey, receipt.PayloadHash, receipt.EventID, receipt.PublishedAt)
	if err != nil {
		return fmt.Errorf("insert Event publication receipt: %w", err)
	}
	return nil
}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertAggregate(ctx context.Context, executor sqlExecutor, aggregate eventbiz.Aggregate) error {
	semantic, err := encodeSemantic(aggregate.Event.Semantic)
	if err != nil {
		return err
	}
	if _, err := executor.ExecContext(ctx, `INSERT INTO events (
    id, title, summary, semantic, modality, occurred_at, announced_at, status
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, aggregate.Event.ID, aggregate.Event.Title,
		aggregate.Event.Summary, semantic, aggregate.Event.Modality, nullableTime(aggregate.Event.OccurredAt),
		nullableTime(aggregate.Event.AnnouncedAt), aggregate.Event.Status); err != nil {
		return fmt.Errorf("insert Event %q: %w", aggregate.Event.ID, err)
	}
	for _, link := range aggregate.Evidence {
		if _, err := executor.ExecContext(ctx, `INSERT INTO event_evidence_links
    (id, event_id, evidence_id, contribution_weight) VALUES ($1,$2,$3,$4)`, link.ID,
			link.EventID, link.EvidenceID, link.ContributionWeight); err != nil {
			return fmt.Errorf("insert Event Evidence Link %q: %w", link.ID, err)
		}
	}
	for _, link := range aggregate.Actors {
		if _, err := executor.ExecContext(ctx, `INSERT INTO event_actor_links (
    id, event_id, actor_id, actor_type, actor_name, relation_type, relation_strength, confidence
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, link.ID, link.EventID, link.ActorID,
			nullableActorType(link.ActorType), nullableString(link.ActorName), link.RelationType,
			nullableFloat(link.RelationStrength), link.Confidence); err != nil {
			return fmt.Errorf("insert Event Actor Link %q: %w", link.ID, err)
		}
	}
	for _, link := range aggregate.Assets {
		if _, err := executor.ExecContext(ctx, `INSERT INTO event_asset_links (
    id, event_id, asset_id, asset_type, asset_name, impact_direction, impact_magnitude
) VALUES ($1,$2,$3,$4,$5,$6,$7)`, link.ID, link.EventID, link.AssetID,
			nullableAssetType(link.AssetType), nullableString(link.AssetName), link.ImpactDirection,
			nullableFloat(link.ImpactMagnitude)); err != nil {
			return fmt.Errorf("insert Event Asset Link %q: %w", link.ID, err)
		}
	}
	return nil
}

var _ eventbiz.PublicationTransaction = (*publicationTransaction)(nil)
