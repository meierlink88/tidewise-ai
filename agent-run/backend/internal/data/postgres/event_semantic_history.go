package postgres

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

func (s *Store) PlanHistoricalEventDisposition(
	ctx context.Context,
	manifest eventsemantic.HistoricalManifest,
) (eventsemantic.HistoricalDispositionReport, error) {
	if err := manifest.Validate(); err != nil {
		return eventsemantic.HistoricalDispositionReport{}, err
	}
	items, err := s.historicalInitialWorkItems(ctx, manifest, false)
	if err != nil {
		return eventsemantic.HistoricalDispositionReport{}, err
	}
	report := classifyHistoricalDisposition(manifest, items)
	report.Mode = "dry_run"
	return report, nil
}

func (s *Store) ApplyHistoricalEventDisposition(
	ctx context.Context,
	manifest eventsemantic.HistoricalManifest,
	now time.Time,
) (eventsemantic.HistoricalDispositionReport, error) {
	if err := manifest.Validate(); err != nil {
		return eventsemantic.HistoricalDispositionReport{}, err
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return eventsemantic.HistoricalDispositionReport{}, fmt.Errorf("begin historical Event disposition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	items, err := historicalInitialWorkItems(ctx, tx, manifest, true)
	if err != nil {
		return eventsemantic.HistoricalDispositionReport{}, err
	}
	report := classifyHistoricalDisposition(manifest, items)
	report.Mode = "apply"
	if len(report.BlockingRunningEventIDs) > 0 {
		return eventsemantic.HistoricalDispositionReport{}, errors.New(
			"historical Event disposition is blocked by running Work Items",
		)
	}
	byEventID := historicalWorkItemsByEventID(items)
	for _, eventID := range manifest.InvalidEventIDs {
		item, exists := byEventID[eventID]
		if !exists {
			if _, err := tx.Exec(ctx, `
				INSERT INTO event_semantic_work_items (
				    work_item_id, event_id, trigger_source, reason, idempotency_key,
				    status, attempt_count, max_attempts, created_at, updated_at
				) VALUES ($1, $2, 'eligible_event', '', $3, 'skipped', 0, 1, $4, $4)
				ON CONFLICT (idempotency_key) DO NOTHING
			`, uuid.NewString(), eventID, "event-semantic-initial:"+eventID, now.UTC()); err != nil {
				return eventsemantic.HistoricalDispositionReport{}, fmt.Errorf(
					"create historical skipped Work Item: %w", err,
				)
			}
			continue
		}
		if item.Status != "pending" && item.Status != "failed" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE event_semantic_work_items
			SET status = 'skipped', lease_expires_at = NULL, updated_at = $2
			WHERE work_item_id = $1 AND status IN ('pending', 'failed')
		`, item.ID, now.UTC()); err != nil {
			return eventsemantic.HistoricalDispositionReport{}, fmt.Errorf(
				"skip historical Event Semantic Work Item: %w", err,
			)
		}
	}
	for _, eventID := range manifest.ValidEventIDs {
		item, exists := byEventID[eventID]
		if !exists || item.Status != "failed" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE event_semantic_work_items
			SET status = 'pending',
			    max_attempts = attempt_count + 1,
			    lease_expires_at = NULL,
			    updated_at = $2
			WHERE work_item_id = $1 AND status = 'failed'
		`, item.ID, now.UTC()); err != nil {
			return eventsemantic.HistoricalDispositionReport{}, fmt.Errorf(
				"recover historical Event Semantic Work Item: %w", err,
			)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return eventsemantic.HistoricalDispositionReport{}, fmt.Errorf(
			"commit historical Event disposition: %w", err,
		)
	}
	return report, nil
}

func (s *Store) historicalInitialWorkItems(
	ctx context.Context,
	manifest eventsemantic.HistoricalManifest,
	forUpdate bool,
) ([]eventsemantic.WorkItem, error) {
	return historicalInitialWorkItems(ctx, s.database, manifest, forUpdate)
}

type historicalQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func historicalInitialWorkItems(
	ctx context.Context,
	queryer historicalQueryer,
	manifest eventsemantic.HistoricalManifest,
	forUpdate bool,
) ([]eventsemantic.WorkItem, error) {
	ids := make([]uuid.UUID, 0, len(manifest.ValidEventIDs)+len(manifest.InvalidEventIDs))
	for _, value := range append(
		append([]string(nil), manifest.ValidEventIDs...),
		manifest.InvalidEventIDs...,
	) {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return nil, errors.New("historical Event manifest contains an invalid Event ID")
		}
		ids = append(ids, parsed)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	query := `
		SELECT work_item_id::text, event_id::text,
		       COALESCE(supersedes_submission_id::text, ''), trigger_source, reason,
		       idempotency_key, status, attempt_count, max_attempts, lease_expires_at,
		       COALESCE(current_execution_id::text, ''), created_at, updated_at
		FROM event_semantic_work_items
		WHERE trigger_source = 'eligible_event'
		  AND event_id = ANY($1::uuid[])
		ORDER BY event_id, created_at, work_item_id
	`
	if forUpdate {
		query += " FOR UPDATE"
	}
	rows, err := queryer.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("read historical Event Semantic Work Items: %w", err)
	}
	defer rows.Close()
	var result []eventsemantic.WorkItem
	for rows.Next() {
		var item eventsemantic.WorkItem
		if err := scanEventSemanticWorkItem(rows, &item); err != nil {
			return nil, fmt.Errorf("scan historical Event Semantic Work Item: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read historical Event Semantic Work Items: %w", err)
	}
	return result, nil
}

func classifyHistoricalDisposition(
	manifest eventsemantic.HistoricalManifest,
	items []eventsemantic.WorkItem,
) eventsemantic.HistoricalDispositionReport {
	report := eventsemantic.HistoricalDispositionReport{
		HistoricalCandidates: len(manifest.ValidEventIDs) + len(manifest.InvalidEventIDs),
		InvalidEventIDs:      append([]string(nil), manifest.InvalidEventIDs...),
		ValidEventIDs:        append([]string(nil), manifest.ValidEventIDs...),
		Before:               make([]eventsemantic.HistoricalWorkItemSnapshot, 0, len(items)),
	}
	byEventID := historicalWorkItemsByEventID(items)
	for _, item := range items {
		report.Before = append(report.Before, eventsemantic.HistoricalWorkItemSnapshot{
			WorkItemID: item.ID, EventID: item.EventID,
			SupersedesSubmissionID: item.SupersedesSubmissionID,
			TriggerSource:          item.TriggerSource,
			Reason:                 item.Reason,
			IdempotencyKey:         item.IdempotencyKey,
			Status:                 item.Status,
			AttemptCount:           item.AttemptCount,
			MaxAttempts:            item.MaxAttempts,
			LeaseExpiresAt:         item.LeaseExpiresAt,
			CurrentExecutionID:     item.CurrentExecutionID,
			CreatedAt:              item.CreatedAt,
			UpdatedAt:              item.UpdatedAt,
		})
	}
	for _, eventID := range manifest.InvalidEventIDs {
		item, exists := byEventID[eventID]
		if !exists {
			report.SkippedCreated++
			report.SkippedCreatedEventIDs = append(
				report.SkippedCreatedEventIDs, eventID,
			)
			continue
		}
		switch item.Status {
		case "pending", "failed":
			report.SkippedUpdated++
			report.SkippedUpdatedEventIDs = append(
				report.SkippedUpdatedEventIDs, eventID,
			)
		case "skipped":
			report.AlreadySkipped++
			report.AlreadySkippedEventIDs = append(
				report.AlreadySkippedEventIDs, eventID,
			)
		case "succeeded":
			report.SucceededPreserved++
			report.SucceededPreservedEventIDs = append(
				report.SucceededPreservedEventIDs, eventID,
			)
		case "running":
			report.BlockingRunningEventIDs = append(
				report.BlockingRunningEventIDs, eventID,
			)
		}
	}
	for _, eventID := range manifest.ValidEventIDs {
		item, exists := byEventID[eventID]
		if !exists {
			report.MissingValidWorkItems++
			report.MissingValidWorkItemEventIDs = append(
				report.MissingValidWorkItemEventIDs, eventID,
			)
			continue
		}
		switch item.Status {
		case "failed":
			report.ValidFailuresRecovered++
			report.ValidFailuresRecoveredEventIDs = append(
				report.ValidFailuresRecoveredEventIDs, eventID,
			)
		case "pending":
			report.PendingPreserved++
			report.PendingPreservedEventIDs = append(
				report.PendingPreservedEventIDs, eventID,
			)
		case "succeeded":
			report.SucceededPreserved++
			report.SucceededPreservedEventIDs = append(
				report.SucceededPreservedEventIDs, eventID,
			)
		case "skipped":
			report.AlreadySkipped++
			report.AlreadySkippedEventIDs = append(
				report.AlreadySkippedEventIDs, eventID,
			)
		case "running":
			report.BlockingRunningEventIDs = append(
				report.BlockingRunningEventIDs, eventID,
			)
		}
	}
	for _, eventIDs := range [][]string{
		report.InvalidEventIDs,
		report.ValidEventIDs,
		report.SkippedCreatedEventIDs,
		report.SkippedUpdatedEventIDs,
		report.ValidFailuresRecoveredEventIDs,
		report.AlreadySkippedEventIDs,
		report.SucceededPreservedEventIDs,
		report.PendingPreservedEventIDs,
		report.MissingValidWorkItemEventIDs,
		report.BlockingRunningEventIDs,
	} {
		sort.Strings(eventIDs)
	}
	return report
}

func historicalWorkItemsByEventID(
	items []eventsemantic.WorkItem,
) map[string]eventsemantic.WorkItem {
	result := make(map[string]eventsemantic.WorkItem, len(items))
	for _, item := range items {
		result[item.EventID] = item
	}
	return result
}
