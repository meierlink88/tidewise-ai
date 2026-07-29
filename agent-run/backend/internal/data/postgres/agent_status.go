package postgres

import (
	"context"
	"fmt"

	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

func (s *Store) ListAgentStatuses(ctx context.Context) ([]agentrun.AgentStatus, error) {
	rows, err := s.database.Query(ctx, `
		SELECT definition.agent_key,
		       definition.display_name,
		       version.version,
		       active.status IS NOT NULL,
		       COALESCE(active.status, 'idle'),
		       COALESCE(active.updated_at, latest.updated_at, version.created_at, definition.created_at)
		FROM agent_definitions definition
		JOIN LATERAL (
		    SELECT candidate.version, candidate.created_at
		    FROM agent_versions candidate
		    WHERE candidate.agent_key = definition.agent_key
		    ORDER BY candidate.created_at DESC, candidate.version DESC
		    LIMIT 1
		) version ON true
		LEFT JOIN LATERAL (
		    SELECT execution.status, execution.updated_at
		    FROM agent_executions execution
		    WHERE execution.agent_key = definition.agent_key
		      AND execution.status IN ('queued', 'planning', 'collecting', 'materializing', 'running')
		    ORDER BY execution.updated_at DESC, execution.execution_id DESC
		    LIMIT 1
		) active ON true
		LEFT JOIN LATERAL (
		    SELECT execution.updated_at
		    FROM agent_executions execution
		    WHERE execution.agent_key = definition.agent_key
		    ORDER BY execution.updated_at DESC, execution.execution_id DESC
		    LIMIT 1
		) latest ON true
		ORDER BY definition.agent_key
	`)
	if err != nil {
		return nil, fmt.Errorf("list Agent statuses: %w", err)
	}
	defer rows.Close()
	statuses := make([]agentrun.AgentStatus, 0)
	for rows.Next() {
		var status agentrun.AgentStatus
		if err := rows.Scan(
			&status.AgentKey,
			&status.DisplayName,
			&status.CurrentVersion,
			&status.IsWorking,
			&status.CurrentExecutionStatus,
			&status.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan Agent status: %w", err)
		}
		statuses = append(statuses, status)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list Agent statuses: %w", err)
	}
	return statuses, nil
}
