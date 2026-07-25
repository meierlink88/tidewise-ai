package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform"
)

func (s *Store) ListAgentExecutions(ctx context.Context, query agentrun.ExecutionListQuery) (agentrun.ExecutionPage, error) {
	if query.Page < 1 || query.PageSize < 1 || query.PageSize > 100 {
		return agentrun.ExecutionPage{}, errors.New("Agent Execution pagination is invalid")
	}
	var where string
	var args []any
	if strings.TrimSpace(query.AgentKey) != "" {
		where = " WHERE agent_key = $1"
		args = append(args, strings.TrimSpace(query.AgentKey))
	}
	var total int
	if err := s.database.QueryRow(ctx, `SELECT count(*) FROM agent_executions`+where, args...).Scan(&total); err != nil {
		return agentrun.ExecutionPage{}, fmt.Errorf("count Agent Executions: %w", err)
	}
	order := "DESC"
	if query.Ascending {
		order = "ASC"
	}
	limitPosition := len(args) + 1
	offsetPosition := limitPosition + 1
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.database.Query(ctx, fmt.Sprintf(`
		SELECT execution_id::text, agent_key, agent_version, trigger_source,
		       COALESCE(schedule_id::text, ''), status,
		       COALESCE(error_code, ''), COALESCE(error_summary, ''),
		       COALESCE(stop_reason, ''), COALESCE(blocked_by_execution_id::text, ''),
		       created_at, triggered_at, started_at, completed_at
		FROM agent_executions%s
		ORDER BY created_at %s, execution_id %s
		LIMIT $%d OFFSET $%d
	`, where, order, order, limitPosition, offsetPosition), args...)
	if err != nil {
		return agentrun.ExecutionPage{}, fmt.Errorf("list Agent Executions: %w", err)
	}
	defer rows.Close()
	items := make([]agentrun.ExecutionListItem, 0, query.PageSize)
	for rows.Next() {
		var item agentrun.ExecutionListItem
		if err := rows.Scan(
			&item.ID, &item.AgentKey, &item.AgentVersion, &item.TriggerSource,
			&item.ScheduleID, &item.Status, &item.ErrorCode, &item.ErrorSummary,
			&item.StopReason, &item.BlockedByExecutionID, &item.CreatedAt,
			&item.TriggeredAt, &item.StartedAt, &item.CompletedAt,
		); err != nil {
			return agentrun.ExecutionPage{}, fmt.Errorf("scan Agent Execution list item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return agentrun.ExecutionPage{}, fmt.Errorf("list Agent Executions: %w", err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + query.PageSize - 1) / query.PageSize
	}
	return agentrun.ExecutionPage{
		Items: items, Page: query.Page, PageSize: query.PageSize,
		TotalItems: total, TotalPages: totalPages,
	}, nil
}
