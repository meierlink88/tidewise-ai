package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

const eventSemanticExecutionLease = 20 * time.Minute

func (s *Store) EnsureInitialWorkItems(
	ctx context.Context,
	events []eventsemantic.EligibleEvent,
	now time.Time,
) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin initial Event Semantic Work Item dispatch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	created := 0
	for _, event := range events {
		if _, err := uuid.Parse(event.EventID); err != nil {
			return 0, errors.New("eligible Event has an invalid ID")
		}
		command, err := tx.Exec(ctx, `
			INSERT INTO event_semantic_work_items (
			    work_item_id, event_id, trigger_source, reason, idempotency_key,
			    status, attempt_count, max_attempts, created_at, updated_at
			) VALUES ($1, $2, 'eligible_event', '', $3, 'pending', 0, 2, $4, $4)
			ON CONFLICT (idempotency_key) DO NOTHING
		`, uuid.NewString(), event.EventID, "event-semantic-initial:"+event.EventID, now.UTC())
		if err != nil {
			return 0, fmt.Errorf("enqueue initial Event Semantic Work Item: %w", err)
		}
		created += int(command.RowsAffected())
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit initial Event Semantic Work Item dispatch: %w", err)
	}
	return created, nil
}

func (s *Store) EnqueueReanalysis(
	ctx context.Context,
	request eventsemantic.ReanalysisRequest,
	now time.Time,
) (eventsemantic.WorkItem, bool, error) {
	if _, err := uuid.Parse(request.EventID); err != nil {
		return eventsemantic.WorkItem{}, false, errors.New("reanalysis Event ID is invalid")
	}
	if _, err := uuid.Parse(request.SupersedesSubmissionID); err != nil {
		return eventsemantic.WorkItem{}, false, errors.New("superseded Submission ID is invalid")
	}
	if request.IdempotencyKey == "" {
		return eventsemantic.WorkItem{}, false, errors.New("reanalysis idempotency key is required")
	}
	workItemID := uuid.NewString()
	command, err := s.database.Exec(ctx, `
		INSERT INTO event_semantic_work_items (
		    work_item_id, event_id, supersedes_submission_id, trigger_source, reason,
		    idempotency_key, status, attempt_count, max_attempts, created_at, updated_at
		) VALUES ($1, $2, $3, 'explicit_reanalysis', $4, $5, 'pending', 0, 2, $6, $6)
		ON CONFLICT (idempotency_key) DO NOTHING
	`, workItemID, request.EventID, request.SupersedesSubmissionID,
		request.Reason, request.IdempotencyKey, now.UTC())
	if err != nil {
		return eventsemantic.WorkItem{}, false, fmt.Errorf("enqueue Event Semantic reanalysis: %w", err)
	}
	item, err := s.readEventSemanticWorkItemByIdempotencyKey(ctx, request.IdempotencyKey)
	if err != nil {
		return eventsemantic.WorkItem{}, false, err
	}
	replayed := command.RowsAffected() == 0
	if replayed && (item.EventID != request.EventID ||
		item.SupersedesSubmissionID != request.SupersedesSubmissionID ||
		item.Reason != request.Reason ||
		item.TriggerSource != "explicit_reanalysis") {
		return eventsemantic.WorkItem{}, false, eventsemantic.ErrReanalysisIdempotencyConflict
	}
	return item, replayed, nil
}

func (s *Store) StartNextExecution(
	ctx context.Context,
	workerID string,
	workflowHash string,
	now time.Time,
) (eventsemantic.ExecutionAttempt, bool, error) {
	if workerID == "" || len(workflowHash) != 64 {
		return eventsemantic.ExecutionAttempt{}, false, errors.New("Event Semantic execution identity is invalid")
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("begin Event Semantic execution: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE agent_executions e
		SET status = 'failed',
		    error_code = 'event_semantic_execution_lease_expired',
		    error_summary = 'Event Semantic execution lease expired',
		    stop_reason = 'retry_wait',
		    completed_at = $1,
		    updated_at = $1
		FROM event_semantic_work_items w
		WHERE w.current_execution_id = e.execution_id
		  AND w.status = 'running'
		  AND w.lease_expires_at <= $1
		  AND e.status = 'running'
	`, now.UTC()); err != nil {
		return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("expire Event Semantic Executions: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_semantic_work_items
		SET status = CASE WHEN attempt_count < max_attempts THEN 'pending' ELSE 'failed' END,
		    lease_expires_at = NULL,
		    updated_at = $1
		WHERE status = 'running' AND lease_expires_at <= $1
	`, now.UTC()); err != nil {
		return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("reconcile Event Semantic Work Items: %w", err)
	}

	var item eventsemantic.WorkItem
	err = scanEventSemanticWorkItem(tx.QueryRow(ctx, `
		SELECT work_item_id::text, event_id::text,
		       COALESCE(supersedes_submission_id::text, ''), trigger_source, reason,
		       idempotency_key, status, attempt_count, max_attempts, lease_expires_at,
		       COALESCE(current_execution_id::text, ''), created_at, updated_at
		FROM event_semantic_work_items
		WHERE status = 'pending'
		  AND attempt_count < max_attempts
		  AND NOT EXISTS (
		      SELECT 1 FROM agent_executions
		      WHERE agent_key = $1
		        AND status IN ('queued', 'planning', 'collecting', 'materializing', 'running')
		  )
		ORDER BY created_at, work_item_id
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, eventsemantic.AgentKey), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("commit empty Event Semantic execution: %w", err)
		}
		return eventsemantic.ExecutionAttempt{}, false, nil
	}
	if err != nil {
		return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("claim Event Semantic Work Item: %w", err)
	}

	executionID := item.CurrentExecutionID
	if executionID == "" {
		executionID = uuid.NewString()
	}
	attemptNumber := item.AttemptCount + 1
	input, err := json.Marshal(map[string]any{
		"work_item_id":             item.ID,
		"event_id":                 item.EventID,
		"supersedes_submission_id": item.SupersedesSubmissionID,
		"attempt":                  attemptNumber,
		"worker_id":                workerID,
	})
	if err != nil {
		return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("encode Event Semantic input: %w", err)
	}
	if item.CurrentExecutionID == "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO agent_executions (
			    execution_id, agent_key, agent_version, idempotency_key, input_payload,
			    trigger_source, triggered_at, prompt, prompt_sha256, prompt_bytes,
			    status, created_at, started_at, updated_at
			) VALUES (
			    $1, $2, $3, $4, $5,
			    'dependent', $6, '', $7, 0,
			    'running', $6, $6, $6
			)
		`, executionID, eventsemantic.AgentKey, eventsemantic.AgentVersion,
			"event-semantic-execution:"+item.ID, input, now.UTC(), workflowHash); err != nil {
			return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("create Event Semantic Execution: %w", err)
		}
	} else {
		command, err := tx.Exec(ctx, `
			UPDATE agent_executions
			SET status = 'running',
			    input_payload = $2,
			    prompt_sha256 = $3,
			    error_code = NULL,
			    error_summary = NULL,
			    stop_reason = NULL,
			    completed_at = NULL,
			    updated_at = $4
			WHERE execution_id = $1 AND status = 'failed'
		`, executionID, input, workflowHash, now.UTC())
		if err != nil {
			return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("retry Event Semantic Execution: %w", err)
		}
		if command.RowsAffected() != 1 {
			return eventsemantic.ExecutionAttempt{}, false, errors.New("Event Semantic Execution is not retryable")
		}
	}
	leaseExpiresAt := now.UTC().Add(eventSemanticExecutionLease)
	if _, err := tx.Exec(ctx, `
		UPDATE event_semantic_work_items
		SET status = 'running',
		    attempt_count = $2,
		    lease_expires_at = $3,
		    current_execution_id = $4,
		    updated_at = $5
		WHERE work_item_id = $1 AND status = 'pending'
	`, item.ID, attemptNumber, leaseExpiresAt, executionID, now.UTC()); err != nil {
		return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("start Event Semantic Work Item: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return eventsemantic.ExecutionAttempt{}, false, fmt.Errorf("commit Event Semantic execution: %w", err)
	}
	item.Status = "running"
	item.AttemptCount = attemptNumber
	item.LeaseExpiresAt = &leaseExpiresAt
	item.CurrentExecutionID = executionID
	item.UpdatedAt = now.UTC()
	return eventsemantic.ExecutionAttempt{ID: executionID, WorkItem: item}, true, nil
}

func (s *Store) CompleteExecution(
	ctx context.Context,
	completion eventsemantic.ExecutionCompletion,
) error {
	if completion.Status != "succeeded" && completion.Status != "failed" {
		return errors.New("Event Semantic completion status is invalid")
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin Event Semantic completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var item eventsemantic.WorkItem
	err = scanEventSemanticWorkItem(tx.QueryRow(ctx, `
		SELECT w.work_item_id::text, w.event_id::text,
		       COALESCE(w.supersedes_submission_id::text, ''), w.trigger_source, w.reason,
		       w.idempotency_key, w.status, w.attempt_count, w.max_attempts,
		       w.lease_expires_at, COALESCE(w.current_execution_id::text, ''),
		       w.created_at, w.updated_at
		FROM event_semantic_work_items w
		WHERE w.current_execution_id = $1 AND w.status = 'running'
		FOR UPDATE
	`, completion.ExecutionID), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("Event Semantic Execution is not running")
	}
	if err != nil {
		return fmt.Errorf("lock Event Semantic Work Item: %w", err)
	}
	stopReason := ""
	workStatus := "succeeded"
	if completion.Status == "failed" {
		if completion.Retryable && item.AttemptCount < item.MaxAttempts {
			workStatus = "pending"
			stopReason = "retry_wait"
		} else {
			workStatus = "failed"
			stopReason = "retry_exhausted"
		}
	}
	command, err := tx.Exec(ctx, `
		UPDATE agent_executions
		SET status = $2,
		    error_code = NULLIF($3, ''),
		    error_summary = NULLIF($4, ''),
		    stop_reason = NULLIF($5, ''),
		    completed_at = $6,
		    updated_at = $6
		WHERE execution_id = $1 AND status = 'running'
	`, completion.ExecutionID, completion.Status, completion.ErrorCode,
		completion.ErrorSummary, stopReason, completion.CompletedAt.UTC())
	if err != nil {
		return fmt.Errorf("complete Event Semantic Execution: %w", err)
	}
	if command.RowsAffected() != 1 {
		return errors.New("Event Semantic Execution is not running")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_semantic_work_items
		SET status = $2,
		    lease_expires_at = NULL,
		    updated_at = $3
		WHERE work_item_id = $1
	`, item.ID, workStatus, completion.CompletedAt.UTC()); err != nil {
		return fmt.Errorf("complete Event Semantic Work Item: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit Event Semantic completion: %w", err)
	}
	return nil
}

func (s *Store) readEventSemanticWorkItemByIdempotencyKey(
	ctx context.Context,
	idempotencyKey string,
) (eventsemantic.WorkItem, error) {
	var item eventsemantic.WorkItem
	err := scanEventSemanticWorkItem(s.database.QueryRow(ctx, `
		SELECT work_item_id::text, event_id::text,
		       COALESCE(supersedes_submission_id::text, ''), trigger_source, reason,
		       idempotency_key, status, attempt_count, max_attempts, lease_expires_at,
		       COALESCE(current_execution_id::text, ''), created_at, updated_at
		FROM event_semantic_work_items
		WHERE idempotency_key = $1
	`, idempotencyKey), &item)
	if err != nil {
		return eventsemantic.WorkItem{}, fmt.Errorf("read Event Semantic Work Item: %w", err)
	}
	return item, nil
}

type eventSemanticWorkItemRow interface {
	Scan(...any) error
}

func scanEventSemanticWorkItem(row eventSemanticWorkItemRow, item *eventsemantic.WorkItem) error {
	return row.Scan(
		&item.ID, &item.EventID, &item.SupersedesSubmissionID,
		&item.TriggerSource, &item.Reason, &item.IdempotencyKey,
		&item.Status, &item.AttemptCount, &item.MaxAttempts,
		&item.LeaseExpiresAt, &item.CurrentExecutionID,
		&item.CreatedAt, &item.UpdatedAt,
	)
}
