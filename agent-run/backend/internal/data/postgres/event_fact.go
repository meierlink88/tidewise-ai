package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
		UPDATE event_artifact_extraction_units u
		SET status = 'pending', error_code = NULL, error_summary = NULL, updated_at = $1
		FROM agent_executions e
		WHERE u.current_execution_id = e.execution_id
		  AND u.status = 'running'
		  AND e.status = 'failed'
		  AND e.error_code = 'extractor_interrupted'
	`, now.UTC()); err != nil {
		return 0, fmt.Errorf("requeue interrupted Event Artifact Units: %w", err)
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
		       created_at, updated_at
		FROM event_extraction_work_items
		WHERE work_item_key = $1
	`, key).Scan(
		&work.Key, &work.CollectorExecutionIDs, &work.ExtractorAgentVersion,
		&work.Status, &work.CurrentExecutionID, &result,
		&work.CreatedAt, &work.UpdatedAt,
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

func (s *Store) NextUnplannedWork(ctx context.Context) (eventfact.WorkItem, bool, error) {
	var work eventfact.WorkItem
	var result []byte
	err := s.database.QueryRow(ctx, `
		SELECT work_item_key, collector_execution_ids::text[], extractor_agent_version,
		       status, COALESCE(current_execution_id::text, ''), extraction_result,
		       created_at, updated_at
		FROM event_extraction_work_items w
		WHERE w.status = 'pending'
		  AND NOT EXISTS (
		      SELECT 1
		      FROM event_artifact_extraction_units u
		      WHERE u.work_item_key = w.work_item_key
		  )
		ORDER BY created_at DESC, work_item_key
		LIMIT 1
	`).Scan(
		&work.Key, &work.CollectorExecutionIDs, &work.ExtractorAgentVersion,
		&work.Status, &work.CurrentExecutionID, &result,
		&work.CreatedAt, &work.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return eventfact.WorkItem{}, false, nil
	}
	if err != nil {
		return eventfact.WorkItem{}, false, fmt.Errorf("read unplanned Event extraction Work Item: %w", err)
	}
	work.ExtractionResult = append(json.RawMessage(nil), result...)
	return work, true, nil
}

func (s *Store) InitializeArtifactUnits(
	ctx context.Context,
	work eventfact.WorkItem,
	artifacts []eventfact.ArtifactSummary,
	now time.Time,
) error {
	if len(artifacts) == 0 {
		return errors.New("Event extraction Batch has no Artifact Units")
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].ArtifactID < artifacts[j].ArtifactID
	})
	allowedCollectors := make(map[string]struct{}, len(work.CollectorExecutionIDs))
	for _, id := range work.CollectorExecutionIDs {
		allowedCollectors[id] = struct{}{}
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for index, artifact := range artifacts {
		if _, exists := allowedCollectors[artifact.CollectorExecutionID]; !exists {
			return errors.New("Artifact Unit references a Collector outside the Batch")
		}
		unitKey, err := eventfact.ArtifactUnitIdentity(
			work.Key, artifact.ArtifactID, artifact.ContentSHA256,
		)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_artifact_extraction_units (
				unit_key, work_item_key, artifact_ordinal, artifact_id,
				collector_execution_id, content_sha256, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7, $7)
			ON CONFLICT (unit_key) DO NOTHING
		`, unitKey, work.Key, index+1, artifact.ArtifactID,
			artifact.CollectorExecutionID, artifact.ContentSHA256, now.UTC()); err != nil {
			return fmt.Errorf("initialize Event Artifact Unit: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) RejectUnplannedWork(
	ctx context.Context,
	work eventfact.WorkItem,
	summary string,
	now time.Time,
) error {
	_, err := s.database.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = 'rejected', error_code = 'artifact_unit_planning_failed',
		    error_summary = $2, updated_at = $3
		WHERE work_item_key = $1 AND status = 'pending'
		  AND NOT EXISTS (
		      SELECT 1 FROM event_artifact_extraction_units u
		      WHERE u.work_item_key = event_extraction_work_items.work_item_key
		  )
	`, work.Key, summary, now.UTC())
	if err != nil {
		return fmt.Errorf("reject unplanned Event extraction Work Item: %w", err)
	}
	return nil
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
	var unit eventfact.ArtifactUnit
	var result []byte
	err = tx.QueryRow(ctx, `
		SELECT w.work_item_key, w.collector_execution_ids::text[], w.extractor_agent_version,
		       w.status, COALESCE(w.current_execution_id::text, ''), w.extraction_result,
		       w.created_at, w.updated_at,
		       u.unit_key, u.work_item_key, u.artifact_ordinal, u.artifact_id,
		       u.collector_execution_id::text, u.content_sha256, u.status,
		       COALESCE(u.current_execution_id::text, ''), u.extraction_result,
		       u.created_at, u.updated_at
		FROM event_artifact_extraction_units u
		JOIN event_extraction_work_items w ON w.work_item_key = u.work_item_key
		WHERE u.status IN ('pending', 'awaiting_tag_catalog', 'retry_wait')
		  AND NOT EXISTS (
		      SELECT 1
		      FROM event_artifact_extraction_units previous
		      WHERE previous.work_item_key = u.work_item_key
		        AND previous.artifact_ordinal < u.artifact_ordinal
		        AND previous.status NOT IN ('published', 'no_events', 'rejected', 'blocked')
		  )
		ORDER BY w.created_at DESC, u.artifact_ordinal, u.unit_key
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(
		&work.Key, &work.CollectorExecutionIDs, &work.ExtractorAgentVersion,
		&work.Status, &work.CurrentExecutionID, &result,
		&work.CreatedAt, &work.UpdatedAt,
		&unit.Key, &unit.WorkItemKey, &unit.ArtifactOrdinal, &unit.ArtifactID,
		&unit.CollectorExecutionID, &unit.ContentSHA256, &unit.Status,
		&unit.CurrentExecutionID, &unit.ExtractionResult,
		&unit.CreatedAt, &unit.UpdatedAt,
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
		"unit_key":                unit.Key,
		"artifact_id":             unit.ArtifactID,
		"artifact_ordinal":        unit.ArtifactOrdinal,
		"collector_execution_ids": work.CollectorExecutionIDs,
	})
	if err != nil {
		return eventfact.ExecutionAttempt{}, false, err
	}
	idempotencyKey := "event-fact-execution:" + unit.Key + ":" + executionID
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
			execution_id, work_item_key, unit_key, prompt_sha256, schema_sha256, provider_key, model
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, executionID, work.Key, unit.Key, snapshot.PromptSHA256, snapshot.SchemaSHA256,
		snapshot.ProviderKey, snapshot.Model); err != nil {
		return eventfact.ExecutionAttempt{}, false, fmt.Errorf("snapshot Event extractor Execution: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_artifact_extraction_units
		SET status = 'running', current_execution_id = $2,
		    error_code = NULL, error_summary = NULL, updated_at = $3
		WHERE unit_key = $1
	`, unit.Key, executionID, now.UTC()); err != nil {
		return eventfact.ExecutionAttempt{}, false, fmt.Errorf("start Event Artifact Unit: %w", err)
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
		Unit:     unit,
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
		UPDATE event_artifact_extraction_units
		SET status = 'retry_wait', extraction_result = $3,
		    error_code = 'extractor_model_unavailable', error_summary = $4, updated_at = $5
		WHERE unit_key = $1 AND current_execution_id = $2
	`, attempt.Unit.Key, attempt.ID, encoded, errorSummary, now.UTC()); err != nil {
		return fmt.Errorf("schedule Event Artifact Unit retry: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = 'retry_wait',
		    error_code = 'extractor_model_unavailable', error_summary = $3, updated_at = $4
		WHERE work_item_key = $1 AND current_execution_id = $2
	`, attempt.WorkItem.Key, attempt.ID, errorSummary, now.UTC()); err != nil {
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
		UPDATE event_artifact_extraction_units
		SET status = 'awaiting_tag_catalog', extraction_result = $3,
		    error_code = 'tag_catalog_unavailable', error_summary = $4, updated_at = $5
		WHERE unit_key = $1 AND current_execution_id = $2
	`, attempt.Unit.Key, attempt.ID, encoded, errorSummary, now.UTC()); err != nil {
		return fmt.Errorf("pause Event Artifact Unit for Tag Catalog: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = 'awaiting_tag_catalog',
		    error_code = 'tag_catalog_unavailable', error_summary = $3, updated_at = $4
		WHERE work_item_key = $1 AND current_execution_id = $2
	`, attempt.WorkItem.Key, attempt.ID, errorSummary, now.UTC()); err != nil {
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
	var lockedWorkItemKey string
	if err := tx.QueryRow(ctx, `
		SELECT work_item_key
		FROM event_extraction_work_items
		WHERE work_item_key = $1
		FOR UPDATE
	`, attempt.WorkItem.Key).Scan(&lockedWorkItemKey); err != nil {
		return fmt.Errorf("allocate Event publication journal ordinals: %w", err)
	}
	var nextBatchOrdinal int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(max(batch_ordinal), 0)
		FROM event_publication_journal
		WHERE work_item_key = $1
	`, lockedWorkItemKey).Scan(&nextBatchOrdinal); err != nil {
		return fmt.Errorf("read Event publication journal ordinal: %w", err)
	}
	if err := completeEventExtractorExecution(ctx, tx, attempt, result, "succeeded", "ready_to_publish", now); err != nil {
		return err
	}
	for index, entry := range journals {
		batchOrdinal := nextBatchOrdinal + index + 1
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_publication_journal (
				work_item_key, unit_key, batch_ordinal, package_id, payload_bytes, payload_sha256,
				status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, 'prepared', $7, $7)
			ON CONFLICT (work_item_key, batch_ordinal) DO NOTHING
		`, attempt.WorkItem.Key, attempt.Unit.Key, batchOrdinal, entry.PackageID, entry.Payload,
			entry.PayloadHash, now.UTC()); err != nil {
			return fmt.Errorf("persist Event publication journal: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_artifact_extraction_units
		SET status = 'ready_to_publish', extraction_result = $3, updated_at = $4
		WHERE unit_key = $1 AND current_execution_id = $2
	`, attempt.Unit.Key, attempt.ID, encoded, now.UTC()); err != nil {
		return fmt.Errorf("complete Event Artifact Unit extraction: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = 'ready_to_publish', updated_at = $3
		WHERE work_item_key = $1 AND current_execution_id = $2
	`, attempt.WorkItem.Key, attempt.ID, now.UTC()); err != nil {
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
		UPDATE event_artifact_extraction_units
		SET status = $3, extraction_result = $4, updated_at = $5
		WHERE unit_key = $1 AND current_execution_id = $2
	`, attempt.Unit.Key, attempt.ID, status, encoded, now.UTC()); err != nil {
		return fmt.Errorf("complete non-publication Event Artifact Unit: %w", err)
	}
	if err := refreshEventExtractionWorkStatus(ctx, tx, attempt.WorkItem.Key, now); err != nil {
		return err
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
		if result.FailureCode != "" {
			errorCode = result.FailureCode
			errorSummary = fmt.Sprintf(
				"Event Fact model contract failed at %s: %s",
				result.FailureStage, result.FailureViolation,
			)
		}
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

func refreshEventExtractionWorkStatus(
	ctx context.Context,
	tx pgx.Tx,
	workItemKey string,
	now time.Time,
) error {
	var total, pending, running, awaitingCatalog, retryWait, ready, publishing int
	var published, noEvents, rejected, blocked int
	if err := tx.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE status = 'pending'),
		       count(*) FILTER (WHERE status = 'running'),
		       count(*) FILTER (WHERE status = 'awaiting_tag_catalog'),
		       count(*) FILTER (WHERE status = 'retry_wait'),
		       count(*) FILTER (WHERE status = 'ready_to_publish'),
		       count(*) FILTER (WHERE status = 'publishing'),
		       count(*) FILTER (WHERE status = 'published'),
		       count(*) FILTER (WHERE status = 'no_events'),
		       count(*) FILTER (WHERE status = 'rejected'),
		       count(*) FILTER (WHERE status = 'blocked')
		FROM event_artifact_extraction_units
		WHERE work_item_key = $1
	`, workItemKey).Scan(
		&total, &pending, &running, &awaitingCatalog, &retryWait, &ready, &publishing,
		&published, &noEvents, &rejected, &blocked,
	); err != nil {
		return fmt.Errorf("summarize Event Artifact Units: %w", err)
	}
	status := eventfact.WorkPending
	switch {
	case publishing > 0:
		status = eventfact.WorkPublishing
	case ready > 0:
		status = eventfact.WorkReadyToPublish
	case running > 0:
		status = eventfact.WorkRunning
	case retryWait > 0:
		status = eventfact.WorkRetryWait
	case awaitingCatalog > 0:
		status = eventfact.WorkAwaitingTagCatalog
	case pending > 0 || total == 0:
		status = eventfact.WorkPending
	case published > 0 && rejected+blocked > 0:
		status = eventfact.WorkPartiallyPublished
	case published > 0:
		status = eventfact.WorkPublished
	case noEvents == total:
		status = eventfact.WorkNoEvents
	case blocked > 0:
		status = eventfact.WorkBlocked
	default:
		status = eventfact.WorkRejected
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_extraction_work_items
		SET status = $2, current_execution_id = NULL,
		    error_code = NULL, error_summary = NULL, updated_at = $3
		WHERE work_item_key = $1
	`, workItemKey, status, now.UTC()); err != nil {
		return fmt.Errorf("refresh Event extraction Batch status: %w", err)
	}
	return nil
}

func (s *Store) ListDeliverableJournals(
	ctx context.Context,
	now time.Time,
) ([]eventfact.JournalEntry, error) {
	rows, err := s.database.Query(ctx, `
		SELECT work_item_key, COALESCE(unit_key::text, ''), batch_ordinal, package_id, payload_bytes,
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
			&entry.WorkItemKey, &entry.UnitKey, &entry.BatchOrdinal, &entry.PackageID, &entry.Payload,
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
		UPDATE event_artifact_extraction_units
		SET status = 'publishing', error_code = NULL, error_summary = NULL, updated_at = $2
		WHERE unit_key = $1 AND status = 'ready_to_publish'
	`, entry.UnitKey, now.UTC()); err != nil {
		return false, fmt.Errorf("mark Event Artifact Unit sending: %w", err)
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
	if err := refreshEventArtifactUnitPublicationStatus(ctx, tx, entry.UnitKey, now); err != nil {
		return err
	}
	if err := refreshEventExtractionWorkStatus(ctx, tx, entry.WorkItemKey, now); err != nil {
		return err
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
	if err := refreshEventArtifactUnitPublicationStatus(ctx, tx, entry.UnitKey, now); err != nil {
		return err
	}
	if err := refreshEventExtractionWorkStatus(ctx, tx, entry.WorkItemKey, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func refreshEventArtifactUnitPublicationStatus(
	ctx context.Context,
	tx pgx.Tx,
	unitKey string,
	now time.Time,
) error {
	var total, acknowledged, blocked, active int
	var errorCode, errorSummary string
	if err := tx.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE status = 'acknowledged'),
		       count(*) FILTER (WHERE status = 'blocked'),
		       count(*) FILTER (WHERE status = 'sending'),
		       COALESCE((
		           SELECT error_code
		           FROM event_publication_journal failed
		           WHERE failed.unit_key = $1
		             AND failed.status IN ('blocked', 'retry_wait')
		           ORDER BY failed.updated_at DESC, failed.batch_ordinal
		           LIMIT 1
		       ), ''),
		       COALESCE((
		           SELECT error_summary
		           FROM event_publication_journal failed
		           WHERE failed.unit_key = $1
		             AND failed.status IN ('blocked', 'retry_wait')
		           ORDER BY failed.updated_at DESC, failed.batch_ordinal
		           LIMIT 1
		       ), '')
		FROM event_publication_journal
		WHERE unit_key = $1
	`, unitKey).Scan(
		&total, &acknowledged, &blocked, &active, &errorCode, &errorSummary,
	); err != nil {
		return fmt.Errorf("summarize Event Artifact Unit journals: %w", err)
	}
	status := eventfact.WorkReadyToPublish
	switch {
	case total > 0 && acknowledged == total:
		status = eventfact.WorkPublished
	case blocked > 0:
		status = eventfact.WorkBlocked
	case active > 0 || acknowledged > 0:
		status = eventfact.WorkPublishing
	}
	if status == eventfact.WorkPublished {
		errorCode, errorSummary = "", ""
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_artifact_extraction_units
		SET status = $2, error_code = NULLIF($3, ''), error_summary = NULLIF($4, ''),
		    updated_at = $5
		WHERE unit_key = $1
	`, unitKey, status, errorCode, errorSummary, now.UTC()); err != nil {
		return fmt.Errorf("refresh Event Artifact Unit publication status: %w", err)
	}
	return nil
}
