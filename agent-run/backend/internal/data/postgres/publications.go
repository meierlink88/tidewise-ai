package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

func (s *Store) PreparePublication(ctx context.Context, reference agentrun.PublicationReference) error {
	if reference.ExecutionID == "" || reference.PlanPath == "" || !validSHA256(reference.PlanSHA256) {
		return fmt.Errorf("invalid Artifact publication reference")
	}
	preparedAt := reference.PreparedAt.UTC()
	if preparedAt.IsZero() {
		preparedAt = time.Now().UTC()
	}
	_, err := s.database.Exec(ctx, `
		INSERT INTO collector_artifact_publications (
			execution_id, plan_path, plan_sha256, prepared_at
		)
		SELECT execution_id, $2, $3, $4
		FROM agent_executions
		WHERE execution_id = $1 AND status = 'materializing'
		ON CONFLICT (execution_id) DO NOTHING
	`, reference.ExecutionID, reference.PlanPath, reference.PlanSHA256, preparedAt)
	if err != nil {
		return fmt.Errorf("prepare Artifact publication: %w", err)
	}
	var storedPath, storedHash string
	err = s.database.QueryRow(ctx, `
		SELECT plan_path, plan_sha256
		FROM collector_artifact_publications
		WHERE execution_id = $1
	`, reference.ExecutionID).Scan(&storedPath, &storedHash)
	if err != nil {
		return fmt.Errorf("read prepared Artifact publication: %w", err)
	}
	if storedPath != reference.PlanPath || storedHash != reference.PlanSHA256 {
		return fmt.Errorf("Artifact publication identity conflict")
	}
	return nil
}

func (s *Store) ListPreparedPublications(ctx context.Context) ([]agentrun.PublicationReference, error) {
	rows, err := s.database.Query(ctx, `
		SELECT execution_id::text, plan_path, plan_sha256, prepared_at
		FROM collector_artifact_publications
		ORDER BY prepared_at, execution_id
	`)
	if err != nil {
		return nil, fmt.Errorf("list prepared Artifact publications: %w", err)
	}
	defer rows.Close()
	var references []agentrun.PublicationReference
	for rows.Next() {
		var reference agentrun.PublicationReference
		if err := rows.Scan(&reference.ExecutionID, &reference.PlanPath, &reference.PlanSHA256, &reference.PreparedAt); err != nil {
			return nil, fmt.Errorf("scan prepared Artifact publication: %w", err)
		}
		references = append(references, reference)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list prepared Artifact publications: %w", err)
	}
	return references, nil
}

// FailPublicationReconciliation atomically disposes any prepared publication,
// preserves its identity on the terminal Execution, and releases the active slot.
// It also handles the uncertain-prepare case where no publication row exists.
func (s *Store) FailPublicationReconciliation(
	ctx context.Context,
	failure agentrun.ExecutionFailure,
) error {
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin failed Artifact publication reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	artifacts := make(map[string]string, len(failure.Artifacts)+2)
	for key, value := range failure.Artifacts {
		artifacts[key] = value
	}
	var planPath, planSHA256 string
	err = tx.QueryRow(ctx, `
		SELECT plan_path, plan_sha256
		FROM collector_artifact_publications
		WHERE execution_id = $1
		FOR UPDATE
	`, failure.ExecutionID).Scan(&planPath, &planSHA256)
	switch {
	case err == nil:
		artifacts["failed_publication_plan"] = planPath
		artifacts["failed_publication_sha256"] = planSHA256
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return fmt.Errorf("lock failed Artifact publication: %w", err)
	}
	artifactsJSON, err := json.Marshal(artifacts)
	if err != nil {
		return fmt.Errorf("encode failed Artifact publication evidence: %w", err)
	}
	command, err := tx.Exec(ctx, `
		UPDATE agent_executions
		SET status = 'failed', error_code = $2, error_summary = $3,
		    stop_reason = $4, artifacts = $5,
		    completed_at = $6, updated_at = $6
		WHERE execution_id = $1 AND status = 'materializing'
	`, failure.ExecutionID, failure.ErrorCode, failure.ErrorSummary,
		failure.StopReason, artifactsJSON, failure.CompletedAt.UTC())
	if err != nil {
		return fmt.Errorf("fail Artifact publication reconciliation: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("Execution is not awaiting Artifact publication reconciliation")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE connector_invocations
		SET status = CASE status WHEN 'pending' THEN 'not_invoked' ELSE 'failed' END,
		    error_code = CASE status WHEN 'pending' THEN 'not_invoked' ELSE 'execution_interrupted' END,
		    error_summary = CASE status WHEN 'pending' THEN $2 ELSE 'Connector did not complete because Agent Execution stopped' END,
		    completed_at = $3
		WHERE execution_id = $1 AND status IN ('pending', 'running')
	`, failure.ExecutionID, failure.NotInvokedSummary, failure.CompletedAt.UTC()); err != nil {
		return fmt.Errorf("fail incomplete connector invocations: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM collector_artifact_publications WHERE execution_id = $1
	`, failure.ExecutionID); err != nil {
		return fmt.Errorf("dispose failed Artifact publication: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit failed Artifact publication reconciliation: %w", err)
	}
	return nil
}

func (s *Store) CommitPreparedPublication(ctx context.Context, reference agentrun.PublicationReference, completion agentrun.ExecutionCompletion) error {
	if completion.ExecutionID != reference.ExecutionID {
		return fmt.Errorf("Artifact publication Execution identity mismatch")
	}
	if err := validatePublicationCompletion(completion); err != nil {
		return err
	}
	countsJSON, err := json.Marshal(completion.CandidateCounts)
	if err != nil {
		return fmt.Errorf("encode Candidate counts: %w", err)
	}
	artifactsJSON, err := json.Marshal(completion.Artifacts)
	if err != nil {
		return fmt.Errorf("encode Artifact paths: %w", err)
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin Artifact publication commit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var storedPath, storedHash string
	err = tx.QueryRow(ctx, `
		SELECT plan_path, plan_sha256
		FROM collector_artifact_publications
		WHERE execution_id = $1
		FOR UPDATE
	`, reference.ExecutionID).Scan(&storedPath, &storedHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return commitAlreadyApplied(ctx, tx, completion)
	}
	if err != nil {
		return fmt.Errorf("lock prepared Artifact publication: %w", err)
	}
	if storedPath != reference.PlanPath || storedHash != reference.PlanSHA256 {
		return fmt.Errorf("Artifact publication identity conflict")
	}
	command, err := tx.Exec(ctx, `
		UPDATE agent_executions
		SET status = $2, stop_reason = $3,
		    error_code = NULLIF($4, ''), error_summary = NULLIF($5, ''),
		    candidate_counts = $6, artifacts = $7,
		    completed_at = $8, updated_at = $8
		WHERE execution_id = $1 AND status = 'materializing'
	`, completion.ExecutionID, completion.Status, completion.StopReason,
		completion.ErrorCode, completion.ErrorSummary, countsJSON,
		artifactsJSON, completion.CompletedAt.UTC())
	if err != nil {
		return fmt.Errorf("commit Agent Execution publication: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("Execution is not awaiting Artifact publication")
	}
	if shouldSignalEventFactExtraction(completion) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO artifact_ready_signals (
				collector_execution_id, status, created_at, updated_at
			) VALUES ($1, 'pending', $2, $2)
			ON CONFLICT (collector_execution_id) DO NOTHING
		`, completion.ExecutionID, completion.CompletedAt.UTC()); err != nil {
			return fmt.Errorf("record Artifact ready signal: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM collector_artifact_publications WHERE execution_id = $1`, reference.ExecutionID); err != nil {
		return fmt.Errorf("clear prepared Artifact publication: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit Artifact publication transaction: %w", err)
	}
	return nil
}

func shouldSignalEventFactExtraction(completion agentrun.ExecutionCompletion) bool {
	if completion.Status != agentrun.StatusSucceeded &&
		completion.Status != agentrun.StatusPartiallySucceeded {
		return false
	}
	return completion.CandidateCounts["accepted"] > 0 && completion.Artifacts["manifest"] != ""
}

func (s *Store) AttachTerminalArtifacts(ctx context.Context, executionID string, artifacts map[string]string, now time.Time) error {
	if executionID == "" || len(artifacts) == 0 {
		return fmt.Errorf("terminal Artifact identity is required")
	}
	encoded, err := json.Marshal(artifacts)
	if err != nil {
		return fmt.Errorf("encode terminal Artifact paths: %w", err)
	}
	command, err := s.database.Exec(ctx, `
		UPDATE agent_executions
		SET artifacts = artifacts || $2::jsonb, updated_at = $3
		WHERE execution_id = $1
		  AND status IN ('failed', 'skipped')
		  AND (
		      artifacts = '{}'::jsonb
		      OR artifacts = $2::jsonb
		      OR (
		          artifacts ? 'failed_publication_plan'
		          AND artifacts ? 'failed_publication_sha256'
		          AND NOT ($2::jsonb ? 'failed_publication_plan')
		          AND NOT ($2::jsonb ? 'failed_publication_sha256')
		          AND (
		              artifacts - 'failed_publication_plan' - 'failed_publication_sha256' = '{}'::jsonb
		              OR artifacts - 'failed_publication_plan' - 'failed_publication_sha256' = $2::jsonb
		          )
		      )
		  )
	`, executionID, encoded, now.UTC())
	if err != nil {
		return fmt.Errorf("attach terminal Artifact paths: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("terminal Artifact paths conflict")
	}
	return nil
}

func (s *Store) ListTerminalExecutionsWithoutArtifacts(ctx context.Context) ([]agentrun.Execution, error) {
	rows, err := s.database.Query(ctx, `
		SELECT execution_id::text
		FROM agent_executions
		WHERE status IN ('failed', 'skipped')
		  AND artifacts - 'failed_publication_plan' - 'failed_publication_sha256' = '{}'::jsonb
		ORDER BY created_at, execution_id
	`)
	if err != nil {
		return nil, fmt.Errorf("list terminal Executions without Artifacts: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan terminal Execution without Artifacts: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("list terminal Executions without Artifacts: %w", err)
	}
	rows.Close()
	executions := make([]agentrun.Execution, 0, len(ids))
	for _, id := range ids {
		execution, err := s.GetExecution(ctx, id)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}
	return executions, nil
}

func commitAlreadyApplied(ctx context.Context, tx pgx.Tx, completion agentrun.ExecutionCompletion) error {
	var status agentrun.ExecutionStatus
	var stopReason, errorCode, errorSummary string
	var countsJSON, artifactsJSON []byte
	err := tx.QueryRow(ctx, `
		SELECT status, COALESCE(stop_reason, ''), COALESCE(error_code, ''),
		       COALESCE(error_summary, ''), candidate_counts, artifacts
		FROM agent_executions
		WHERE execution_id = $1
	`, completion.ExecutionID).Scan(&status, &stopReason, &errorCode, &errorSummary, &countsJSON, &artifactsJSON)
	if err != nil {
		return fmt.Errorf("read committed Agent Execution publication: %w", err)
	}
	var counts map[string]int
	var artifacts map[string]string
	if json.Unmarshal(countsJSON, &counts) != nil || json.Unmarshal(artifactsJSON, &artifacts) != nil {
		return fmt.Errorf("decode committed Agent Execution publication")
	}
	if status != completion.Status || stopReason != completion.StopReason ||
		errorCode != completion.ErrorCode || errorSummary != completion.ErrorSummary ||
		!reflect.DeepEqual(counts, completion.CandidateCounts) ||
		!reflect.DeepEqual(artifacts, completion.Artifacts) {
		return fmt.Errorf("Artifact publication is not prepared")
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit idempotent Artifact publication read: %w", err)
	}
	return nil
}

func validatePublicationCompletion(completion agentrun.ExecutionCompletion) error {
	switch completion.Status {
	case agentrun.StatusSucceeded, agentrun.StatusSucceededNoChange, agentrun.StatusPartiallySucceeded:
		if completion.ErrorCode != "" || completion.ErrorSummary != "" {
			return fmt.Errorf("successful Execution must not contain a terminal error")
		}
	case agentrun.StatusFailed:
		if completion.ErrorCode == "" || completion.ErrorSummary == "" {
			return fmt.Errorf("failed Execution requires a safe terminal error")
		}
	default:
		return fmt.Errorf("invalid Artifact publication terminal status %q", completion.Status)
	}
	if pending, exists := completion.CandidateCounts["results_pending"]; !exists || pending != 0 {
		return fmt.Errorf("Artifact publication requires results_pending=0")
	}
	if completion.StopReason == "" {
		return fmt.Errorf("Artifact publication requires a stop reason")
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
