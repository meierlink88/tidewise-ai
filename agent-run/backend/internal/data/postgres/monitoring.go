package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

func (s *Store) ListMonitoringStatusCounts(ctx context.Context, since time.Time) ([]agentrun.MonitoringStatusCount, error) {
	rows, err := s.database.Query(ctx, `
		SELECT kind, status, count(*)
		FROM (
			SELECT 'collector' AS kind, status FROM agent_executions
			WHERE agent_key = 'collector' AND triggered_at >= $1
			UNION ALL
			SELECT 'artifact_extraction', status FROM event_artifact_extraction_units WHERE updated_at >= $1
			UNION ALL
			SELECT 'semantic', status FROM event_semantic_work_items WHERE updated_at >= $1
		) monitored
		GROUP BY kind, status
		ORDER BY kind, status`, since)
	if err != nil {
		return nil, fmt.Errorf("list monitoring status counts: %w", err)
	}
	defer rows.Close()
	result := make([]agentrun.MonitoringStatusCount, 0)
	for rows.Next() {
		var item agentrun.MonitoringStatusCount
		if err := rows.Scan(&item.Kind, &item.Status, &item.Count); err != nil {
			return nil, fmt.Errorf("scan monitoring status count: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) GetMonitoringBusinessTotals(ctx context.Context, since time.Time) (agentrun.MonitoringBusinessTotals, error) {
	var result agentrun.MonitoringBusinessTotals
	if err := s.database.QueryRow(ctx, `
		SELECT
			COALESCE(sum((candidate_counts->>'raw_results')::int), 0),
			COALESCE(sum((candidate_counts->>'merged_results')::int), 0),
			COALESCE(sum((candidate_counts->>'accepted')::int), 0)
		FROM agent_executions
		WHERE agent_key = 'collector' AND triggered_at >= $1
		  AND status IN ('succeeded', 'succeeded_no_change', 'partially_succeeded')`, since).Scan(
		&result.CollectorRawResults, &result.CollectorMergedResults, &result.CollectorAcceptedArtifacts,
	); err != nil {
		return result, fmt.Errorf("load collector monitoring totals: %w", err)
	}
	if err := s.database.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE status = 'published'),
			count(*) FILTER (WHERE status = 'no_events'),
			COALESCE(sum(CASE WHEN status = 'published' THEN
				jsonb_array_length(jsonb_path_query_array(extraction_result, '$.candidates[*] ? (@.review_state == "auto_approved")'))
				ELSE 0 END), 0)
		FROM event_artifact_extraction_units
		WHERE updated_at >= $1`, since).Scan(
		&result.ArtifactPublished, &result.ArtifactNoEvents, &result.ArtifactFormalEvents,
	); err != nil {
		return result, fmt.Errorf("load artifact monitoring totals: %w", err)
	}
	if err := s.database.QueryRow(ctx, `
		SELECT
			COALESCE(sum((e.candidate_counts->>'submissions')::int), 0),
			COALESCE(sum((e.candidate_counts->>'accepted_candidates')::int), 0),
			COALESCE(sum((e.candidate_counts->>'rejected_candidates')::int), 0)
		FROM event_semantic_work_items w
		JOIN agent_executions e ON e.execution_id = w.current_execution_id
		WHERE w.updated_at >= $1 AND w.status = 'succeeded'`, since).Scan(
		&result.SemanticSubmissions, &result.SemanticAcceptedCandidates, &result.SemanticRejectedCandidates,
	); err != nil {
		return result, fmt.Errorf("load semantic monitoring totals: %w", err)
	}
	return result, nil
}

func (s *Store) ListCollectorMonitoring(ctx context.Context, query agentrun.MonitoringListQuery) (agentrun.CollectorMonitoringPage, error) {
	var total int
	if err := s.database.QueryRow(ctx, `SELECT count(*) FROM agent_executions WHERE agent_key = 'collector' AND triggered_at >= $1 AND status = ANY($2)`, query.Since, query.Statuses).Scan(&total); err != nil {
		return agentrun.CollectorMonitoringPage{}, fmt.Errorf("count collector monitoring: %w", err)
	}
	rows, err := s.database.Query(ctx, `
		SELECT execution_id::text, status, trigger_source, started_at, completed_at,
			COALESCE((candidate_counts->>'raw_results')::int, 0),
			COALESCE((candidate_counts->>'merged_results')::int, 0),
			COALESCE((candidate_counts->>'accepted')::int, 0), COALESCE(error_code, '')
		FROM agent_executions
		WHERE agent_key = 'collector' AND triggered_at >= $1 AND status = ANY($2)
		ORDER BY triggered_at DESC, execution_id DESC LIMIT $3 OFFSET $4`,
		query.Since, query.Statuses, query.PageSize, (query.Page-1)*query.PageSize)
	if err != nil {
		return agentrun.CollectorMonitoringPage{}, fmt.Errorf("list collector monitoring: %w", err)
	}
	defer rows.Close()
	items := make([]agentrun.CollectorMonitoringRecord, 0)
	for rows.Next() {
		var item agentrun.CollectorMonitoringRecord
		if err := rows.Scan(&item.ExecutionID, &item.RawStatus, &item.TriggerSource, &item.StartedAt, &item.CompletedAt, &item.RawResults, &item.MergedResults, &item.AcceptedArtifacts, &item.ErrorCode); err != nil {
			return agentrun.CollectorMonitoringPage{}, fmt.Errorf("scan collector monitoring: %w", err)
		}
		items = append(items, item)
	}
	return agentrun.CollectorMonitoringPage{Items: items, Page: query.Page, PageSize: query.PageSize, TotalItems: total, TotalPages: monitoringTotalPages(total, query.PageSize)}, rows.Err()
}

func (s *Store) ListArtifactExtractionMonitoring(ctx context.Context, query agentrun.MonitoringListQuery) (agentrun.ArtifactExtractionMonitoringPage, error) {
	var total int
	if err := s.database.QueryRow(ctx, `SELECT count(*) FROM event_artifact_extraction_units WHERE updated_at >= $1 AND status = ANY($2)`, query.Since, query.Statuses).Scan(&total); err != nil {
		return agentrun.ArtifactExtractionMonitoringPage{}, fmt.Errorf("count artifact monitoring: %w", err)
	}
	rows, err := s.database.Query(ctx, `
		SELECT u.unit_key, u.artifact_id, u.collector_execution_id::text, u.status, u.updated_at,
			e.started_at, e.completed_at,
			CASE WHEN jsonb_typeof(u.extraction_result->'candidates') = 'array' THEN jsonb_array_length(u.extraction_result->'candidates') ELSE 0 END,
			COALESCE(j.acknowledged, 0), COALESCE(j.total, 0), COALESCE(NULLIF(u.error_code, ''), e.error_code, '')
		FROM event_artifact_extraction_units u
		LEFT JOIN agent_executions e ON e.execution_id = u.current_execution_id
		LEFT JOIN LATERAL (
			SELECT count(*) FILTER (WHERE status = 'acknowledged') AS acknowledged, count(*) AS total
			FROM event_publication_journal WHERE unit_key = u.unit_key
		) j ON true
		WHERE u.updated_at >= $1 AND u.status = ANY($2)
		ORDER BY u.updated_at DESC, u.unit_key DESC LIMIT $3 OFFSET $4`,
		query.Since, query.Statuses, query.PageSize, (query.Page-1)*query.PageSize)
	if err != nil {
		return agentrun.ArtifactExtractionMonitoringPage{}, fmt.Errorf("list artifact monitoring: %w", err)
	}
	defer rows.Close()
	items := make([]agentrun.ArtifactExtractionMonitoringRecord, 0)
	for rows.Next() {
		var item agentrun.ArtifactExtractionMonitoringRecord
		if err := rows.Scan(&item.ExtractionKey, &item.ArtifactID, &item.CollectorExecutionID, &item.RawStatus, &item.UpdatedAt, &item.StartedAt, &item.CompletedAt, &item.EventCandidates, &item.AcknowledgedJournals, &item.TotalJournals, &item.ErrorCode); err != nil {
			return agentrun.ArtifactExtractionMonitoringPage{}, fmt.Errorf("scan artifact monitoring: %w", err)
		}
		items = append(items, item)
	}
	return agentrun.ArtifactExtractionMonitoringPage{Items: items, Page: query.Page, PageSize: query.PageSize, TotalItems: total, TotalPages: monitoringTotalPages(total, query.PageSize)}, rows.Err()
}

func (s *Store) ListSemanticMonitoring(ctx context.Context, query agentrun.MonitoringListQuery) (agentrun.SemanticMonitoringPage, error) {
	var total int
	if err := s.database.QueryRow(ctx, `SELECT count(*) FROM event_semantic_work_items WHERE updated_at >= $1 AND status = ANY($2)`, query.Since, query.Statuses).Scan(&total); err != nil {
		return agentrun.SemanticMonitoringPage{}, fmt.Errorf("count semantic monitoring: %w", err)
	}
	rows, err := s.database.Query(ctx, `
		SELECT w.work_item_id::text, w.event_id::text, w.trigger_source, w.status, w.updated_at,
			e.started_at, e.completed_at, w.attempt_count, w.max_attempts,
			COALESCE((e.candidate_counts->>'accepted_candidates')::int, 0),
			COALESCE((e.candidate_counts->>'rejected_candidates')::int, 0), COALESCE(e.error_code, '')
		FROM event_semantic_work_items w
		LEFT JOIN agent_executions e ON e.execution_id = w.current_execution_id
		WHERE w.updated_at >= $1 AND w.status = ANY($2)
		ORDER BY w.updated_at DESC, w.work_item_id DESC LIMIT $3 OFFSET $4`,
		query.Since, query.Statuses, query.PageSize, (query.Page-1)*query.PageSize)
	if err != nil {
		return agentrun.SemanticMonitoringPage{}, fmt.Errorf("list semantic monitoring: %w", err)
	}
	defer rows.Close()
	items := make([]agentrun.SemanticMonitoringRecord, 0)
	for rows.Next() {
		var item agentrun.SemanticMonitoringRecord
		if err := rows.Scan(&item.WorkItemID, &item.EventID, &item.TriggerSource, &item.RawStatus, &item.UpdatedAt, &item.StartedAt, &item.CompletedAt, &item.AttemptCount, &item.MaxAttempts, &item.AcceptedCandidates, &item.RejectedCandidates, &item.ErrorCode); err != nil {
			return agentrun.SemanticMonitoringPage{}, fmt.Errorf("scan semantic monitoring: %w", err)
		}
		items = append(items, item)
	}
	return agentrun.SemanticMonitoringPage{Items: items, Page: query.Page, PageSize: query.PageSize, TotalItems: total, TotalPages: monitoringTotalPages(total, query.PageSize)}, rows.Err()
}

func monitoringTotalPages(total, pageSize int) int {
	if total == 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}
