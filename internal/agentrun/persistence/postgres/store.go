package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	database *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse AgentRun database configuration: %w", err)
	}
	name := config.ConnConfig.Database
	if name != "tidewise_ai_server" && !strings.HasPrefix(name, "tidewise_ai_server_test") {
		return nil, fmt.Errorf("AgentRun database must be tidewise_ai_server or an isolated tidewise_ai_server_test database")
	}
	database, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open AgentRun database: %w", err)
	}
	if err := database.Ping(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("ping AgentRun database: %w", err)
	}
	return database, nil
}

func New(database *pgxpool.Pool) *Store {
	return &Store{database: database}
}

func (s *Store) GetAgentVersion(ctx context.Context, version string) (agentrun.AgentVersion, error) {
	var result agentrun.AgentVersion
	err := s.database.QueryRow(ctx, `SELECT agent_key, version FROM agent_versions WHERE version = $1`, version).Scan(&result.AgentKey, &result.Version)
	if err != nil {
		return agentrun.AgentVersion{}, fmt.Errorf("get agent version: %w", err)
	}
	return result, nil
}

func (s *Store) CreateExecution(ctx context.Context, input agentrun.CreateExecutionInput) (agentrun.Execution, agentrun.CreateDisposition, error) {
	if strings.TrimSpace(input.AgentVersion) == "" {
		return agentrun.Execution{}, "", errors.New("Agent Version is required")
	}
	if len(input.InvocationKeys) == 0 {
		return agentrun.Execution{}, "", errors.New("at least one Invocation key is required")
	}
	seenInvocationKeys := make(map[string]struct{}, len(input.InvocationKeys))
	for _, key := range input.InvocationKeys {
		if strings.TrimSpace(key) == "" {
			return agentrun.Execution{}, "", errors.New("Invocation key is required")
		}
		if _, exists := seenInvocationKeys[key]; exists {
			return agentrun.Execution{}, "", fmt.Errorf("duplicate Invocation key %q", key)
		}
		seenInvocationKeys[key] = struct{}{}
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return agentrun.Execution{}, "", fmt.Errorf("begin create execution: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('agentrun_single_active_execution'))`); err != nil {
		return agentrun.Execution{}, "", fmt.Errorf("lock execution creation: %w", err)
	}

	var existingID, existingPrompt string
	err = tx.QueryRow(ctx, `SELECT execution_id::text, prompt FROM agent_executions WHERE idempotency_key = $1`, input.IdempotencyKey).Scan(&existingID, &existingPrompt)
	if err == nil {
		if existingPrompt != input.Prompt {
			return agentrun.Execution{}, "", agentrun.ErrIdempotencyConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return agentrun.Execution{}, "", fmt.Errorf("commit execution replay: %w", err)
		}
		execution, err := s.GetExecution(ctx, existingID)
		return execution, agentrun.ExecutionReplayed, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return agentrun.Execution{}, "", fmt.Errorf("check idempotency key: %w", err)
	}

	var activeID string
	err = tx.QueryRow(ctx, `SELECT execution_id::text FROM agent_executions WHERE status IN ('queued', 'planning', 'collecting', 'materializing') LIMIT 1`).Scan(&activeID)
	if err == nil {
		id := uuid.NewString()
		createdAt := input.CreatedAt.UTC()
		sum := sha256.Sum256([]byte(input.Prompt))
		promptHash := hex.EncodeToString(sum[:])
		_, err = tx.Exec(ctx, `
			INSERT INTO agent_executions (
				execution_id, agent_version, idempotency_key, prompt, prompt_sha256,
				prompt_bytes, status, stop_reason, blocked_by_execution_id,
				created_at, completed_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, 'skipped', 'skipped_previous_run_active', $7, $8, $8, $8)
		`, id, input.AgentVersion, input.IdempotencyKey, input.Prompt, promptHash,
			len([]byte(input.Prompt)), activeID, createdAt)
		if err != nil {
			return agentrun.Execution{}, "", fmt.Errorf("insert skipped execution: %w", err)
		}
		for position, key := range input.InvocationKeys {
			if _, err := tx.Exec(ctx, `
				INSERT INTO connector_invocations (
					execution_id, connector_key, position, status, error_code,
					error_summary, completed_at
				) VALUES ($1, $2, $3, 'not_invoked', 'not_invoked',
				          'Connector was not invoked because another Agent Execution was active', $4)
			`, id, key, position, createdAt); err != nil {
				return agentrun.Execution{}, "", fmt.Errorf("insert skipped connector invocation: %w", err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return agentrun.Execution{}, "", fmt.Errorf("commit skipped execution: %w", err)
		}
		execution, err := s.GetExecution(ctx, id)
		return execution, agentrun.ExecutionSkipped, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return agentrun.Execution{}, "", fmt.Errorf("check active execution: %w", err)
	}

	id := uuid.NewString()
	createdAt := input.CreatedAt.UTC()
	sum := sha256.Sum256([]byte(input.Prompt))
	promptHash := hex.EncodeToString(sum[:])
	_, err = tx.Exec(ctx, `
		INSERT INTO agent_executions (
			execution_id, agent_version, idempotency_key, prompt, prompt_sha256,
			prompt_bytes, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, 'queued', $7, $7)
	`, id, input.AgentVersion, input.IdempotencyKey, input.Prompt, promptHash, len([]byte(input.Prompt)), createdAt)
	if err != nil {
		return agentrun.Execution{}, "", fmt.Errorf("insert execution: %w", err)
	}
	for position, key := range input.InvocationKeys {
		if _, err := tx.Exec(ctx, `INSERT INTO connector_invocations (execution_id, connector_key, position, status) VALUES ($1, $2, $3, 'pending')`, id, key, position); err != nil {
			return agentrun.Execution{}, "", fmt.Errorf("insert connector invocation: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return agentrun.Execution{}, "", fmt.Errorf("commit execution creation: %w", err)
	}
	execution, err := s.GetExecution(ctx, id)
	return execution, agentrun.ExecutionCreated, err
}

func (s *Store) FindExecutionByIdempotencyKey(ctx context.Context, idempotencyKey, prompt string) (agentrun.Execution, bool, error) {
	var id, storedPrompt string
	err := s.database.QueryRow(ctx, `SELECT execution_id::text, prompt FROM agent_executions WHERE idempotency_key = $1`, idempotencyKey).Scan(&id, &storedPrompt)
	if errors.Is(err, pgx.ErrNoRows) {
		return agentrun.Execution{}, false, nil
	}
	if err != nil {
		return agentrun.Execution{}, false, fmt.Errorf("find idempotent execution: %w", err)
	}
	if storedPrompt != prompt {
		return agentrun.Execution{}, false, agentrun.ErrIdempotencyConflict
	}
	execution, err := s.GetExecution(ctx, id)
	if err != nil {
		return agentrun.Execution{}, false, err
	}
	return execution, true, nil
}

func (s *Store) GetExecution(ctx context.Context, id string) (agentrun.Execution, error) {
	var result agentrun.Execution
	var countsJSON, artifactsJSON []byte
	err := s.database.QueryRow(ctx, `
		SELECT execution_id::text, agent_version, idempotency_key, prompt, prompt_sha256,
		       prompt_bytes, status, COALESCE(error_code, ''), COALESCE(error_summary, ''),
		       COALESCE(stop_reason, ''), COALESCE(blocked_by_execution_id::text, ''),
		       candidate_counts, artifacts, created_at, started_at, completed_at
		FROM agent_executions WHERE execution_id = $1
	`, id).Scan(
		&result.ID, &result.AgentVersion, &result.IdempotencyKey, &result.Prompt,
		&result.PromptSHA256, &result.PromptBytes, &result.Status, &result.ErrorCode,
		&result.ErrorSummary, &result.StopReason, &result.BlockedByExecutionID,
		&countsJSON, &artifactsJSON, &result.CreatedAt,
		&result.StartedAt, &result.CompletedAt,
	)
	if err != nil {
		return agentrun.Execution{}, err
	}
	if err := json.Unmarshal(countsJSON, &result.CandidateCounts); err != nil {
		return agentrun.Execution{}, fmt.Errorf("decode candidate counts: %w", err)
	}
	if err := json.Unmarshal(artifactsJSON, &result.Artifacts); err != nil {
		return agentrun.Execution{}, fmt.Errorf("decode artifacts: %w", err)
	}
	rows, err := s.database.Query(ctx, `
		SELECT connector_key, status, result_count, COALESCE(error_code, ''),
		       COALESCE(error_summary, ''), started_at, completed_at
		FROM connector_invocations WHERE execution_id = $1 ORDER BY position
	`, id)
	if err != nil {
		return agentrun.Execution{}, fmt.Errorf("list connector invocations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var invocation agentrun.ConnectorInvocation
		if err := rows.Scan(&invocation.ConnectorKey, &invocation.Status, &invocation.ResultCount, &invocation.ErrorCode, &invocation.ErrorSummary, &invocation.StartedAt, &invocation.CompletedAt); err != nil {
			return agentrun.Execution{}, fmt.Errorf("scan connector invocation: %w", err)
		}
		result.Invocations = append(result.Invocations, invocation)
	}
	if err := rows.Err(); err != nil {
		return agentrun.Execution{}, fmt.Errorf("list connector invocations: %w", err)
	}
	return result, nil
}

func (s *Store) FailStaleExecutions(ctx context.Context, now time.Time) error {
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin stale execution cleanup: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		UPDATE agent_executions
		SET status = 'failed', error_code = 'process_restarted',
		    error_summary = 'AgentRun restarted before execution completed',
		    stop_reason = 'agent_or_tool_limit',
		    completed_at = $1, updated_at = $1
		WHERE status IN ('queued', 'planning', 'collecting', 'materializing')
		  AND NOT EXISTS (
		      SELECT 1 FROM collector_artifact_publications publication
		      WHERE publication.execution_id = agent_executions.execution_id
		  )
		RETURNING execution_id
	`, now.UTC())
	if err != nil {
		return fmt.Errorf("mark stale executions failed: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan stale execution: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("scan stale executions: %w", err)
	}
	rows.Close()
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `
			UPDATE connector_invocations
			SET status = CASE status WHEN 'pending' THEN 'not_invoked' ELSE 'failed' END,
			    error_code = CASE status WHEN 'pending' THEN 'not_invoked' ELSE 'process_restarted' END,
			    error_summary = CASE status
			        WHEN 'pending' THEN 'Connector was not invoked before AgentRun restarted'
			        ELSE 'AgentRun restarted before connector completed'
			    END,
			    completed_at = $2
			WHERE execution_id = $1 AND status IN ('pending', 'running')
		`, id, now.UTC()); err != nil {
			return fmt.Errorf("mark stale connector invocations failed: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit stale execution cleanup: %w", err)
	}
	return nil
}

func (s *Store) SetExecutionStatus(ctx context.Context, id string, status agentrun.ExecutionStatus, now time.Time) error {
	var previous agentrun.ExecutionStatus
	switch status {
	case agentrun.StatusPlanning:
		previous = agentrun.StatusQueued
	case agentrun.StatusCollecting:
		previous = agentrun.StatusPlanning
	case agentrun.StatusMaterializing:
		previous = agentrun.StatusCollecting
	default:
		return fmt.Errorf("invalid running Execution status %q", status)
	}
	startedAt := any(nil)
	if status == agentrun.StatusPlanning {
		startedAt = now.UTC()
	}
	command, err := s.database.Exec(ctx, `
		UPDATE agent_executions
		SET status = $2, started_at = COALESCE(started_at, $3), updated_at = $4
		WHERE execution_id = $1 AND status = $5
	`, id, status, startedAt, now.UTC(), previous)
	if err != nil {
		return fmt.Errorf("update execution status: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("execution is not active")
	}
	return nil
}

func (s *Store) StartInvocation(ctx context.Context, executionID, connectorKey string, now time.Time) error {
	command, err := s.database.Exec(ctx, `
		UPDATE connector_invocations
		SET status = 'running', started_at = $3
		WHERE execution_id = $1 AND connector_key = $2 AND status = 'pending'
	`, executionID, connectorKey, now.UTC())
	if err != nil {
		return fmt.Errorf("start connector invocation: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("connector invocation is not pending")
	}
	return nil
}

func (s *Store) FinishInvocation(ctx context.Context, completion agentrun.InvocationCompletion) error {
	if completion.Status != agentrun.InvocationCompleted && completion.Status != agentrun.InvocationFailed {
		return fmt.Errorf("invalid terminal Invocation status %q", completion.Status)
	}
	command, err := s.database.Exec(ctx, `
		UPDATE connector_invocations
		SET status = $3, result_count = $4, error_code = NULLIF($5, ''),
		    error_summary = NULLIF($6, ''), completed_at = $7
		WHERE execution_id = $1 AND connector_key = $2 AND status = 'running'
	`, completion.ExecutionID, completion.ConnectorKey, completion.Status, completion.ResultCount, completion.ErrorCode, completion.ErrorSummary, completion.CompletedAt.UTC())
	if err != nil {
		return fmt.Errorf("finish connector invocation: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("connector invocation is not running")
	}
	return nil
}

func (s *Store) FailExecutionAndIncompleteInvocations(ctx context.Context, failure agentrun.ExecutionFailure) error {
	if failure.Artifacts == nil {
		failure.Artifacts = map[string]string{}
	}
	artifactsJSON, err := json.Marshal(failure.Artifacts)
	if err != nil {
		return fmt.Errorf("encode failed Execution Artifact paths: %w", err)
	}
	tx, err := s.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin fail execution: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `
		UPDATE agent_executions
		SET status = 'failed', error_code = $2, error_summary = $3,
		    stop_reason = $4, artifacts = $5,
		    completed_at = $6, updated_at = $6
		WHERE execution_id = $1 AND status IN ('queued', 'planning', 'collecting', 'materializing')
		  AND NOT EXISTS (
		      SELECT 1 FROM collector_artifact_publications
		      WHERE execution_id = $1
		  )
	`, failure.ExecutionID, failure.ErrorCode, failure.ErrorSummary, failure.StopReason,
		artifactsJSON, failure.CompletedAt.UTC())
	if err != nil {
		return fmt.Errorf("fail execution: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("execution is not active")
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
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit failed execution: %w", err)
	}
	return nil
}
