package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
)

func (s *Store) DispatchPendingSignals(ctx context.Context, agentVersion string, now time.Time) (int, error) {
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin Event extraction dispatch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE agent_executions
		SET status = 'failed', error_code = 'extractor_interrupted',
		    error_summary = 'Event extractor Execution was interrupted',
		    stop_reason = 'reconciled_interrupted_execution',
		    completed_at = $1, updated_at = $1
		WHERE agent_key = $2 AND status = 'running'
		  AND updated_at < $1::timestamptz - interval '15 minutes'
	`, now.UTC(), eventfact.AgentKey); err != nil {
		return 0, fmt.Errorf("reconcile interrupted Event extractor Executions: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items w
		SET status = 'pending', error_code = NULL, error_summary = NULL, updated_at = $1
		FROM agent_executions e
		WHERE w.current_execution_id = e.execution_id
		  AND w.status = 'running'
		  AND e.status = 'failed'
		  AND e.error_code = 'extractor_interrupted'
	`, now.UTC()); err != nil {
		return 0, fmt.Errorf("requeue interrupted Event extraction Work Items: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO artifact_ready_signals (
			collector_execution_id, status, created_at, updated_at
		)
		SELECT execution_id, 'pending', completed_at, $1
		FROM agent_executions
		WHERE agent_key = 'collector'
		  AND status IN ('succeeded', 'partially_succeeded')
		  AND COALESCE((candidate_counts->>'accepted')::integer, 0) > 0
		  AND COALESCE(artifacts->>'manifest', '') <> ''
		  AND completed_at IS NOT NULL
		ON CONFLICT (collector_execution_id) DO NOTHING
	`, now.UTC()); err != nil {
		return 0, fmt.Errorf("reconcile Artifact ready signals: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT collector_execution_id::text
		FROM artifact_ready_signals
		WHERE status = 'pending'
		ORDER BY created_at, collector_execution_id
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		return 0, fmt.Errorf("claim Artifact ready signals: %w", err)
	}
	var collectorIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan Artifact ready signal: %w", err)
		}
		collectorIDs = append(collectorIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate Artifact ready signals: %w", err)
	}
	rows.Close()

	dispatched := 0
	for _, collectorID := range collectorIDs {
		key, ids, err := eventfact.WorkItemIdentity([]string{collectorID}, agentVersion)
		if err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_extraction_work_items (
				work_item_key, collector_execution_ids, extractor_agent_version,
				status, created_at, updated_at
			) VALUES ($1, $2::uuid[], $3, 'pending', $4, $4)
			ON CONFLICT (work_item_key) DO NOTHING
		`, key, ids, agentVersion, now.UTC()); err != nil {
			return 0, fmt.Errorf("upsert Event extraction Work Item: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE artifact_ready_signals
			SET status = 'dispatched', dispatched_at = $2, updated_at = $2
			WHERE collector_execution_id = $1
		`, collectorID, now.UTC()); err != nil {
			return 0, fmt.Errorf("complete Artifact ready dispatch: %w", err)
		}
		dispatched++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit Event extraction dispatch: %w", err)
	}
	return dispatched, nil
}

func (s *Store) EnqueueWork(
	ctx context.Context,
	collectorExecutionIDs []string,
	agentVersion string,
	now time.Time,
) (eventfact.WorkItem, bool, error) {
	key, ids, err := eventfact.WorkItemIdentity(collectorExecutionIDs, agentVersion)
	if err != nil {
		return eventfact.WorkItem{}, false, err
	}
	command, err := s.database.Exec(ctx, `
		INSERT INTO event_extraction_work_items (
			work_item_key, collector_execution_ids, extractor_agent_version,
			status, created_at, updated_at
		)
		SELECT $1, $2::uuid[], $3, 'pending', $4, $4
		WHERE NOT EXISTS (
			SELECT 1
			FROM unnest($2::uuid[]) AS requested(execution_id)
			LEFT JOIN agent_executions e ON e.execution_id = requested.execution_id
			WHERE e.execution_id IS NULL
			   OR e.agent_key <> 'collector'
			   OR e.status NOT IN ('succeeded', 'partially_succeeded')
			   OR COALESCE(e.artifacts->>'manifest', '') = ''
			   OR COALESCE((e.candidate_counts->>'accepted')::integer, 0) <= 0
			   OR COALESCE((e.candidate_counts->>'results_pending')::integer, 0) <> 0
		)
		ON CONFLICT (work_item_key) DO NOTHING
	`, key, ids, agentVersion, now.UTC())
	if err != nil {
		return eventfact.WorkItem{}, false, fmt.Errorf("enqueue Event extraction Work Item: %w", err)
	}
	var work eventfact.WorkItem
	var result []byte
	err = s.database.QueryRow(ctx, `
		SELECT work_item_key, collector_execution_ids::text[], extractor_agent_version,
		       status, COALESCE(current_execution_id::text, ''), extraction_result,
		       COALESCE(tag_catalog_revision, ''), COALESCE(tag_catalog_hash, ''),
		       created_at, updated_at
		FROM event_extraction_work_items
		WHERE work_item_key = $1
	`, key).Scan(
		&work.Key, &work.CollectorExecutionIDs, &work.ExtractorAgentVersion,
		&work.Status, &work.CurrentExecutionID, &result,
		&work.TagCatalogRevision, &work.TagCatalogHash, &work.CreatedAt, &work.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return eventfact.WorkItem{}, false, errors.New("Collector Executions are not eligible for Event extraction")
	}
	if err != nil {
		return eventfact.WorkItem{}, false, fmt.Errorf("read enqueued Event extraction Work Item: %w", err)
	}
	work.ExtractionResult = append(json.RawMessage(nil), result...)
	return work, command.RowsAffected() == 1, nil
}

func (s *Store) ClaimNextWork(
	ctx context.Context,
	snapshot eventfact.ExtractionSnapshot,
	now time.Time,
) (eventfact.ExecutionAttempt, bool, error) {
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return eventfact.ExecutionAttempt{}, false, fmt.Errorf("begin Event extraction claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var work eventfact.WorkItem
	var result []byte
	err = tx.QueryRow(ctx, `
		SELECT work_item_key, collector_execution_ids::text[], extractor_agent_version,
		       status, COALESCE(current_execution_id::text, ''), extraction_result,
		       COALESCE(tag_catalog_revision, ''), COALESCE(tag_catalog_hash, ''),
		       created_at, updated_at
		FROM event_extraction_work_items
		WHERE status IN ('pending', 'awaiting_tag_catalog', 'retry_wait')
		  AND NOT EXISTS (
		      SELECT 1
		      FROM event_publication_journal j
		      WHERE j.work_item_key = event_extraction_work_items.work_item_key
		  )
		ORDER BY updated_at, work_item_key
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(
		&work.Key, &work.CollectorExecutionIDs, &work.ExtractorAgentVersion,
		&work.Status, &work.CurrentExecutionID, &result,
		&work.TagCatalogRevision, &work.TagCatalogHash, &work.CreatedAt, &work.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return eventfact.ExecutionAttempt{}, false, fmt.Errorf("commit empty Event extraction claim: %w", err)
		}
		return eventfact.ExecutionAttempt{}, false, nil
	}
	if err != nil {
		return eventfact.ExecutionAttempt{}, false, fmt.Errorf("claim Event extraction Work Item: %w", err)
	}
	work.ExtractionResult = append(json.RawMessage(nil), result...)
	executionID := uuid.NewString()
	input, err := json.Marshal(map[string]any{
		"work_item_key":           work.Key,
		"collector_execution_ids": work.CollectorExecutionIDs,
	})
	if err != nil {
		return eventfact.ExecutionAttempt{}, false, err
	}
	idempotencyKey := "event-fact-execution:" + work.Key + ":" + executionID
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
	`, executionID, eventfact.AgentKey, work.ExtractorAgentVersion, idempotencyKey, input,
		now.UTC(), snapshot.PromptSHA256); err != nil {
		return eventfact.ExecutionAttempt{}, false, fmt.Errorf("create Event extractor Execution: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO event_extractor_executions (
			execution_id, work_item_key, prompt_sha256, schema_sha256, provider_key, model
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, executionID, work.Key, snapshot.PromptSHA256, snapshot.SchemaSHA256,
		snapshot.ProviderKey, snapshot.Model); err != nil {
		return eventfact.ExecutionAttempt{}, false, fmt.Errorf("snapshot Event extractor Execution: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = 'running', current_execution_id = $2,
		    error_code = NULL, error_summary = NULL, updated_at = $3
		WHERE work_item_key = $1
	`, work.Key, executionID, now.UTC()); err != nil {
		return eventfact.ExecutionAttempt{}, false, fmt.Errorf("start Event extraction Work Item: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return eventfact.ExecutionAttempt{}, false, fmt.Errorf("commit Event extraction claim: %w", err)
	}
	return eventfact.ExecutionAttempt{
		ID:       executionID,
		WorkItem: work,
		Snapshot: snapshot,
	}, true, nil
}

func (s *Store) RetryExtraction(
	ctx context.Context,
	attempt eventfact.ExecutionAttempt,
	result eventfact.Result,
	errorSummary string,
	now time.Time,
) error {
	encoded, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode retryable Event extraction result: %w", err)
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = 'retry_wait', extraction_result = $3,
		    error_code = 'extractor_model_unavailable', error_summary = $4, updated_at = $5
		WHERE work_item_key = $1 AND current_execution_id = $2
	`, attempt.WorkItem.Key, attempt.ID, encoded, errorSummary, now.UTC()); err != nil {
		return fmt.Errorf("schedule Event extraction retry: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extractor_executions
		SET extraction_model_calls = $2, review_model_calls = $3
		WHERE execution_id = $1
	`, attempt.ID, result.ExtractionModelCalls, result.ReviewModelCalls); err != nil {
		return fmt.Errorf("record retryable Event extractor calls: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE agent_executions
		SET status = 'failed', error_code = 'extractor_model_unavailable',
		    error_summary = $2, stop_reason = 'retry_wait',
		    completed_at = $3, updated_at = $3
		WHERE execution_id = $1 AND status = 'running'
	`, attempt.ID, errorSummary, now.UTC()); err != nil {
		return fmt.Errorf("complete retryable Event extractor Execution: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) SetAwaitingTagCatalog(
	ctx context.Context,
	attempt eventfact.ExecutionAttempt,
	result eventfact.Result,
	errorSummary string,
	now time.Time,
) error {
	encoded, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode partial Event extraction result: %w", err)
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = 'awaiting_tag_catalog', extraction_result = $3,
		    error_code = 'tag_catalog_unavailable', error_summary = $4, updated_at = $5
		WHERE work_item_key = $1 AND current_execution_id = $2
	`, attempt.WorkItem.Key, attempt.ID, encoded, errorSummary, now.UTC()); err != nil {
		return fmt.Errorf("pause Event extraction for Tag Catalog: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extractor_executions
		SET extraction_model_calls = $2, review_model_calls = $3
		WHERE execution_id = $1
	`, attempt.ID, result.ExtractionModelCalls, result.ReviewModelCalls); err != nil {
		return fmt.Errorf("record Event extractor model calls: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE agent_executions
		SET status = 'failed', error_code = 'tag_catalog_unavailable',
		    error_summary = $2, stop_reason = 'awaiting_tag_catalog',
		    completed_at = $3, updated_at = $3
		WHERE execution_id = $1 AND status = 'running'
	`, attempt.ID, errorSummary, now.UTC()); err != nil {
		return fmt.Errorf("complete awaiting Catalog Execution: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) SetExecutionCatalog(
	ctx context.Context,
	executionID, revision, hash string,
	now time.Time,
) error {
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		UPDATE event_extractor_executions
		SET tag_catalog_revision = $2, tag_catalog_hash = $3
		WHERE execution_id = $1
	`, executionID, revision, hash); err != nil {
		return fmt.Errorf("snapshot Event Tag Catalog: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items w
		SET tag_catalog_revision = $2, tag_catalog_hash = $3, updated_at = $4
		WHERE current_execution_id = $1
	`, executionID, revision, hash, now.UTC()); err != nil {
		return fmt.Errorf("snapshot Work Item Tag Catalog: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) CompleteExtraction(
	ctx context.Context,
	attempt eventfact.ExecutionAttempt,
	result eventfact.Result,
	journals []eventfact.JournalEntry,
	now time.Time,
) error {
	encoded, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode Event extraction result: %w", err)
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := completeEventExtractorExecution(ctx, tx, attempt, result, "succeeded", "ready_to_publish", now); err != nil {
		return err
	}
	for _, entry := range journals {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_publication_journal (
				work_item_key, batch_ordinal, package_id, payload_bytes, payload_sha256,
				status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, 'prepared', $6, $6)
			ON CONFLICT (work_item_key, batch_ordinal) DO NOTHING
		`, attempt.WorkItem.Key, entry.BatchOrdinal, entry.PackageID, entry.Payload,
			entry.PayloadHash, now.UTC()); err != nil {
			return fmt.Errorf("persist Event publication journal: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = 'ready_to_publish', extraction_result = $3, updated_at = $4
		WHERE work_item_key = $1 AND current_execution_id = $2
	`, attempt.WorkItem.Key, attempt.ID, encoded, now.UTC()); err != nil {
		return fmt.Errorf("complete Event extraction Work Item: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) CompleteWithoutPublication(
	ctx context.Context,
	attempt eventfact.ExecutionAttempt,
	result eventfact.Result,
	status eventfact.WorkStatus,
	now time.Time,
) error {
	switch status {
	case eventfact.WorkAwaitingReview, eventfact.WorkRejected, eventfact.WorkNoEvents:
	default:
		return fmt.Errorf("invalid non-publication Work status %q", status)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode Event extraction result: %w", err)
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	executionStatus := "succeeded"
	if status == eventfact.WorkRejected {
		executionStatus = "failed"
	}
	if err := completeEventExtractorExecution(ctx, tx, attempt, result, executionStatus, string(status), now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = $3, extraction_result = $4, updated_at = $5
		WHERE work_item_key = $1 AND current_execution_id = $2
	`, attempt.WorkItem.Key, attempt.ID, status, encoded, now.UTC()); err != nil {
		return fmt.Errorf("complete non-publication Event extraction Work Item: %w", err)
	}
	return tx.Commit(ctx)
}

func completeEventExtractorExecution(
	ctx context.Context,
	tx pgx.Tx,
	attempt eventfact.ExecutionAttempt,
	result eventfact.Result,
	status, stopReason string,
	now time.Time,
) error {
	if _, err := tx.Exec(ctx, `
		UPDATE event_extractor_executions
		SET extraction_model_calls = $2, review_model_calls = $3
		WHERE execution_id = $1
	`, attempt.ID, result.ExtractionModelCalls, result.ReviewModelCalls); err != nil {
		return fmt.Errorf("record Event extractor call counts: %w", err)
	}
	errorCode, errorSummary := "", ""
	if status == "failed" {
		errorCode, errorSummary = "event_fact_rejected", "Event facts failed deterministic validation"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE agent_executions
		SET status = $2, stop_reason = $3, error_code = NULLIF($4, ''),
		    error_summary = NULLIF($5, ''), completed_at = $6, updated_at = $6
		WHERE execution_id = $1 AND status = 'running'
	`, attempt.ID, status, stopReason, errorCode, errorSummary, now.UTC()); err != nil {
		return fmt.Errorf("complete Event extractor Execution: %w", err)
	}
	return nil
}

func (s *Store) ListDeliverableJournals(
	ctx context.Context,
	now time.Time,
) ([]eventfact.JournalEntry, error) {
	rows, err := s.database.Query(ctx, `
		SELECT work_item_key, batch_ordinal, package_id, payload_bytes,
		       payload_sha256, status, COALESCE(receipt_id, ''), attempt_count
		FROM event_publication_journal
		WHERE status IN ('prepared', 'retry_wait')
		   OR (status = 'sending' AND updated_at <= $1::timestamptz - interval '15 minutes')
		ORDER BY updated_at, work_item_key, batch_ordinal
	`, now.UTC())
	if err != nil {
		return nil, fmt.Errorf("list Event publication journals: %w", err)
	}
	defer rows.Close()
	var result []eventfact.JournalEntry
	for rows.Next() {
		var entry eventfact.JournalEntry
		if err := rows.Scan(
			&entry.WorkItemKey, &entry.BatchOrdinal, &entry.PackageID, &entry.Payload,
			&entry.PayloadHash, &entry.Status, &entry.ReceiptID, &entry.AttemptCount,
		); err != nil {
			return nil, fmt.Errorf("scan Event publication journal: %w", err)
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (s *Store) FindCanonicalEvents(
	ctx context.Context,
	identityHashes []string,
) ([]eventfact.CanonicalEvent, error) {
	if len(identityHashes) == 0 {
		return nil, nil
	}
	rows, err := s.database.Query(ctx, `
		WITH exact AS (
			SELECT dedupe_key, identity_hash, core_facts, published_at
			FROM event_fact_canonical_events
			WHERE identity_hash::text = ANY($1::text[])
		),
		recent AS (
			SELECT dedupe_key, identity_hash, core_facts, published_at
			FROM event_fact_canonical_events
			ORDER BY published_at DESC, dedupe_key
			LIMIT 500
		)
		SELECT DISTINCT ON (dedupe_key)
		       dedupe_key, identity_hash, core_facts
		FROM (
			SELECT * FROM exact
			UNION ALL
			SELECT * FROM recent
		) recalled
		ORDER BY dedupe_key, published_at DESC
	`, identityHashes)
	if err != nil {
		return nil, fmt.Errorf("recall canonical Event facts: %w", err)
	}
	defer rows.Close()
	var result []eventfact.CanonicalEvent
	for rows.Next() {
		var item eventfact.CanonicalEvent
		if err := rows.Scan(&item.DedupeKey, &item.IdentityHash, &item.CoreFacts); err != nil {
			return nil, fmt.Errorf("scan canonical Event fact: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read canonical Event facts: %w", err)
	}
	return result, nil
}

func (s *Store) MarkJournalSending(
	ctx context.Context,
	entry eventfact.JournalEntry,
	now time.Time,
) (bool, error) {
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `
		UPDATE event_publication_journal
		SET status = 'sending', attempt_count = attempt_count + 1,
		    error_code = NULL, error_summary = NULL, updated_at = $5
		WHERE work_item_key = $1 AND batch_ordinal = $2 AND payload_sha256 = $3
		  AND status = $4 AND attempt_count = $6
		  AND (
		      status IN ('prepared', 'retry_wait')
		      OR (status = 'sending' AND updated_at <= $5::timestamptz - interval '15 minutes')
		  )
	`, entry.WorkItemKey, entry.BatchOrdinal, entry.PayloadHash, entry.Status, now.UTC(),
		entry.AttemptCount)
	if err != nil {
		return false, fmt.Errorf("mark Event publication sending: %w", err)
	}
	if command.RowsAffected() != 1 {
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit skipped Event publication claim: %w", err)
		}
		return false, nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = 'publishing', error_code = NULL, error_summary = NULL, updated_at = $2
		WHERE work_item_key = $1 AND status IN ('ready_to_publish', 'publishing')
	`, entry.WorkItemKey, now.UTC()); err != nil {
		return false, fmt.Errorf("mark Event publication Work Item sending: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit Event publication claim: %w", err)
	}
	return true, nil
}

func (s *Store) MarkJournalRetry(
	ctx context.Context,
	entry eventfact.JournalEntry,
	code, summary string,
	now time.Time,
) error {
	return s.updateJournalFailure(ctx, entry, "retry_wait", code, summary, now)
}

func (s *Store) MarkJournalBlocked(
	ctx context.Context,
	entry eventfact.JournalEntry,
	code, summary string,
	now time.Time,
) error {
	return s.updateJournalFailure(ctx, entry, "blocked", code, summary, now)
}

func (s *Store) updateJournalFailure(
	ctx context.Context,
	entry eventfact.JournalEntry,
	status, code, summary string,
	now time.Time,
) error {
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `
		UPDATE event_publication_journal
		SET status = $4, error_code = $5, error_summary = $6, updated_at = $7
		WHERE work_item_key = $1 AND batch_ordinal = $2 AND payload_sha256 = $3
		  AND status = 'sending' AND attempt_count = $8
	`, entry.WorkItemKey, entry.BatchOrdinal, entry.PayloadHash, status, code, summary, now.UTC(),
		entry.AttemptCount+1)
	if err != nil {
		return fmt.Errorf("record Event publication failure: %w", err)
	}
	if command.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	workStatus := eventfact.WorkReadyToPublish
	if status == "blocked" {
		workStatus = eventfact.WorkBlocked
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = $2, error_code = $3, error_summary = $4, updated_at = $5
		WHERE work_item_key = $1
	`, entry.WorkItemKey, workStatus, code, summary, now.UTC()); err != nil {
		return fmt.Errorf("record Event publication Work failure: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) AcknowledgeJournal(
	ctx context.Context,
	entry eventfact.JournalEntry,
	receiptID string,
	canonical []eventfact.CanonicalEvent,
	now time.Time,
) error {
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `
		UPDATE event_publication_journal
		SET status = 'acknowledged', receipt_id = $4,
		    error_code = NULL, error_summary = NULL, updated_at = $5
		WHERE work_item_key = $1 AND batch_ordinal = $2 AND payload_sha256 = $3
		  AND status = 'sending' AND attempt_count = $6
	`, entry.WorkItemKey, entry.BatchOrdinal, entry.PayloadHash, receiptID, now.UTC(),
		entry.AttemptCount+1)
	if err != nil {
		return fmt.Errorf("acknowledge Event publication: %w", err)
	}
	if command.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	for _, event := range canonical {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_fact_canonical_events (
				dedupe_key, identity_hash, core_facts, published_at
			) VALUES ($1, $2, $3, $4)
			ON CONFLICT (dedupe_key) DO NOTHING
		`, event.DedupeKey, event.IdentityHash, event.CoreFacts, now.UTC()); err != nil {
			return fmt.Errorf("record canonical Event fact: %w", err)
		}
	}
	var pending int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM event_publication_journal
		WHERE work_item_key = $1 AND status <> 'acknowledged'
	`, entry.WorkItemKey).Scan(&pending); err != nil {
		return fmt.Errorf("count pending Event publications: %w", err)
	}
	if pending == 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE event_extraction_work_items
			SET status = 'published', error_code = NULL, error_summary = NULL, updated_at = $2
			WHERE work_item_key = $1
		`, entry.WorkItemKey, now.UTC()); err != nil {
			return fmt.Errorf("complete Event publication Work Item: %w", err)
		}
	}
	return tx.Commit(ctx)
}
