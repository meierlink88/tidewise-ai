package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

type GraphProjectionType string

const (
	GraphProjectionTypeEntityGraph GraphProjectionType = "entity_graph"
)

type GraphProjectionMode string

const (
	GraphProjectionModeProjectEntities GraphProjectionMode = "project_entities"
	GraphProjectionModeRebuildEntities GraphProjectionMode = "rebuild_entities"
)

type GraphProjectionRunStatus string

const (
	GraphProjectionRunStatusRunning   GraphProjectionRunStatus = "running"
	GraphProjectionRunStatusSucceeded GraphProjectionRunStatus = "succeeded"
	GraphProjectionRunStatusFailed    GraphProjectionRunStatus = "failed"
	GraphProjectionRunStatusPartial   GraphProjectionRunStatus = "partial"
)

type GraphProjectionRunItemType string

const (
	GraphProjectionRunItemTypeEntity       GraphProjectionRunItemType = "entity_node"
	GraphProjectionRunItemTypeRelationship GraphProjectionRunItemType = "entity_relationship"
)

type GraphProjectionRunItemStatus string

const (
	GraphProjectionRunItemStatusProjected GraphProjectionRunItemStatus = "projected"
	GraphProjectionRunItemStatusSkipped   GraphProjectionRunItemStatus = "skipped"
	GraphProjectionRunItemStatusFailed    GraphProjectionRunItemStatus = "failed"
)

type GraphEntityNode struct {
	ID                 string
	EntityKey          string
	EntityType         domain.EntityType
	LayerCode          string
	Name               string
	CanonicalName      string
	Aliases            []string
	ClassificationCode domain.SectorClassification
	Status             domain.Status
	UpdatedAt          time.Time
}

type GraphEntityEdge struct {
	ID           string
	FromEntityID string
	ToEntityID   string
	RelationType string
	EvidenceNote string
	Source       string
	Status       domain.Status
	UpdatedAt    time.Time
}

type GraphProjectionRun struct {
	ID             string
	ProjectionType GraphProjectionType
	Mode           GraphProjectionMode
	Status         GraphProjectionRunStatus
	StartedAt      time.Time
	FinishedAt     *time.Time
	SourceRowCount int
	ProjectedCount int
	SkippedCount   int
	FailedCount    int
	ErrorSummary   string
	ConfigSummary  map[string]any
}

type GraphProjectionRunItem struct {
	ID           string
	RunID        string
	ItemType     GraphProjectionRunItemType
	ItemKey      string
	Status       GraphProjectionRunItemStatus
	ErrorMessage string
}

type GraphProjectionRepository interface {
	ListGraphEntityNodes(context.Context) ([]GraphEntityNode, error)
	ListGraphEntityEdges(context.Context) ([]GraphEntityEdge, error)
	CreateGraphProjectionRun(context.Context, GraphProjectionRun) (GraphProjectionRun, error)
	RecordGraphProjectionRunItem(context.Context, GraphProjectionRunItem) error
	CompleteGraphProjectionRun(context.Context, GraphProjectionRun) error
	RecentGraphProjectionRuns(context.Context, int) ([]GraphProjectionRun, error)
}

func (r *InMemoryRepository) SeedGraphEntity(entity GraphEntityNode) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.graphEntities[entity.ID] = normalizeGraphEntityNode(entity)
}

func (r *InMemoryRepository) SeedGraphEdge(edge GraphEntityEdge) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.graphEdges[edge.ID] = edge
}

func (r *InMemoryRepository) ListGraphEntityNodes(context.Context) ([]GraphEntityNode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodes := make([]GraphEntityNode, 0, len(r.graphEntities))
	for _, node := range r.graphEntities {
		if node.Status != domain.StatusActive {
			continue
		}
		nodes = append(nodes, normalizeGraphEntityNode(node))
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return nodes, nil
}

func (r *InMemoryRepository) ListGraphEntityEdges(context.Context) ([]GraphEntityEdge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	edges := make([]GraphEntityEdge, 0, len(r.graphEdges))
	for _, edge := range r.graphEdges {
		from, fromOK := r.graphEntities[edge.FromEntityID]
		to, toOK := r.graphEntities[edge.ToEntityID]
		if edge.Status != domain.StatusActive || !fromOK || !toOK || from.Status != domain.StatusActive || to.Status != domain.StatusActive {
			continue
		}
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})
	return edges, nil
}

func (r *InMemoryRepository) CreateGraphProjectionRun(_ context.Context, run GraphProjectionRun) (GraphProjectionRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	run = normalizeGraphProjectionRun(run)
	r.graphRuns[run.ID] = cloneGraphProjectionRun(run)
	return cloneGraphProjectionRun(run), nil
}

func (r *InMemoryRepository) RecordGraphProjectionRunItem(_ context.Context, item GraphProjectionRunItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.graphRuns[item.RunID]; !ok {
		return fmt.Errorf("graph projection run %q not found", item.RunID)
	}
	r.graphRunItems[item.RunID] = append(r.graphRunItems[item.RunID], item)
	return nil
}

func (r *InMemoryRepository) CompleteGraphProjectionRun(_ context.Context, run GraphProjectionRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.graphRuns[run.ID]; !ok {
		return fmt.Errorf("graph projection run %q not found", run.ID)
	}
	r.graphRuns[run.ID] = cloneGraphProjectionRun(normalizeGraphProjectionRun(run))
	return nil
}

func (r *InMemoryRepository) RecentGraphProjectionRuns(_ context.Context, limit int) ([]GraphProjectionRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	runs := make([]GraphProjectionRun, 0, len(r.graphRuns))
	for _, run := range r.graphRuns {
		runs = append(runs, cloneGraphProjectionRun(run))
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func normalizeGraphEntityNode(node GraphEntityNode) GraphEntityNode {
	node.Aliases = append([]string(nil), node.Aliases...)
	if node.EntityKey == "" && node.EntityType != "" && node.ID != "" {
		node.EntityKey = fmt.Sprintf("%s:%s", node.EntityType, node.ID)
	}
	if node.Status == "" {
		node.Status = domain.StatusActive
	}
	return node
}

func normalizeGraphProjectionRun(run GraphProjectionRun) GraphProjectionRun {
	if run.ProjectionType == "" {
		run.ProjectionType = GraphProjectionTypeEntityGraph
	}
	if run.Mode == "" {
		run.Mode = GraphProjectionModeProjectEntities
	}
	if run.Status == "" {
		run.Status = GraphProjectionRunStatusRunning
	}
	if run.ConfigSummary == nil {
		run.ConfigSummary = map[string]any{}
	}
	return run
}

func cloneGraphProjectionRun(run GraphProjectionRun) GraphProjectionRun {
	if run.FinishedAt != nil {
		value := *run.FinishedAt
		run.FinishedAt = &value
	}
	run.ConfigSummary = cloneMap(run.ConfigSummary)
	return run
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	copied := make(map[string]any, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}

const graphEntityNodesQuery = `
SELECT node.id,
       COALESCE(NULLIF(node.entity_key, ''), node.entity_type || ':' || node.id::text) AS entity_key,
       node.entity_type, node.layer_code, node.name, node.canonical_name,
       COALESCE(array_to_json(node.aliases)::text, '[]') AS aliases,
       node.status, node.updated_at
FROM entity_nodes node
WHERE node.status = 'active'
  AND node.entity_type IN ('alliance_org', 'economy', 'chain_node')
ORDER BY node.id
`

const graphEntityEdgesQuery = `
SELECT source.id, source.from_entity_id, source.to_entity_id, source.relation_type,
       source.evidence_note, source.source, source.status, source.updated_at
FROM (
    SELECT edge.id, edge.from_entity_id, edge.to_entity_id, edge.relation_type,
           edge.evidence_note, 'postgres_entity_edges' AS source, edge.status, edge.updated_at
    FROM entity_edges edge
    JOIN entity_nodes from_node ON from_node.id = edge.from_entity_id
    JOIN entity_nodes to_node ON to_node.id = edge.to_entity_id
    WHERE edge.status = 'active'
      AND from_node.status = 'active'
      AND to_node.status = 'active'
      AND from_node.entity_type IN ('alliance_org', 'economy', 'chain_node')
      AND to_node.entity_type IN ('alliance_org', 'economy', 'chain_node')
      AND edge.source_name <> '' AND edge.source_url <> '' AND edge.verified_at IS NOT NULL
    UNION ALL
    SELECT relation.id, relation.from_chain_node_entity_id, relation.to_chain_node_entity_id,
           relation.relation_type, relation.evidence_note, 'postgres_chain_node_relations' AS source,
           relation.status, relation.updated_at
    FROM chain_node_relations relation
    JOIN entity_nodes from_node ON from_node.id = relation.from_chain_node_entity_id
    JOIN entity_nodes to_node ON to_node.id = relation.to_chain_node_entity_id
    WHERE relation.status = 'active'
      AND from_node.status = 'active'
      AND to_node.status = 'active'
      AND from_node.entity_type = 'chain_node'
      AND to_node.entity_type = 'chain_node'
      AND relation.relation_type IN ('is_subcategory_of', 'is_component_of', 'input_to', 'depends_on')
) source
ORDER BY source.id
`

func (r PostgresRepository) ListGraphEntityNodes(ctx context.Context) ([]GraphEntityNode, error) {
	rows, err := r.db.QueryContext(ctx, graphEntityNodesQuery)
	if err != nil {
		return nil, fmt.Errorf("query graph entity nodes: %w", err)
	}
	defer rows.Close()

	nodes := make([]GraphEntityNode, 0)
	for rows.Next() {
		var node GraphEntityNode
		var aliasesJSON string
		if err := rows.Scan(
			&node.ID,
			&node.EntityKey,
			&node.EntityType,
			&node.LayerCode,
			&node.Name,
			&node.CanonicalName,
			&aliasesJSON,
			&node.Status,
			&node.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan graph entity node: %w", err)
		}
		if err := json.Unmarshal([]byte(aliasesJSON), &node.Aliases); err != nil {
			return nil, fmt.Errorf("decode graph entity aliases: %w", err)
		}
		nodes = append(nodes, normalizeGraphEntityNode(node))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate graph entity nodes: %w", err)
	}
	return nodes, nil
}

func (r PostgresRepository) ListGraphEntityEdges(ctx context.Context) ([]GraphEntityEdge, error) {
	rows, err := r.db.QueryContext(ctx, graphEntityEdgesQuery)
	if err != nil {
		return nil, fmt.Errorf("query graph entity edges: %w", err)
	}
	defer rows.Close()

	edges := make([]GraphEntityEdge, 0)
	for rows.Next() {
		var edge GraphEntityEdge
		if err := rows.Scan(
			&edge.ID,
			&edge.FromEntityID,
			&edge.ToEntityID,
			&edge.RelationType,
			&edge.EvidenceNote,
			&edge.Source,
			&edge.Status,
			&edge.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan graph entity edge: %w", err)
		}
		edges = append(edges, edge)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate graph entity edges: %w", err)
	}
	return edges, nil
}

func (r PostgresRepository) CreateGraphProjectionRun(ctx context.Context, run GraphProjectionRun) (GraphProjectionRun, error) {
	run = normalizeGraphProjectionRun(run)
	run.ID = NormalizeUUID(run.ID)
	configSummary, err := json.Marshal(cloneMap(run.ConfigSummary))
	if err != nil {
		return GraphProjectionRun{}, fmt.Errorf("marshal graph projection config summary: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
INSERT INTO graph_projection_runs (
    id, projection_type, mode, status, started_at, finished_at, source_row_count,
    projected_count, skipped_count, failed_count, error_summary, config_summary
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12::jsonb
) RETURNING id, projection_type, mode, status, started_at, finished_at,
            source_row_count, projected_count, skipped_count, failed_count,
            error_summary, config_summary
`, run.ID, run.ProjectionType, run.Mode, run.Status, run.StartedAt, nullTime(run.FinishedAt),
		run.SourceRowCount, run.ProjectedCount, run.SkippedCount, run.FailedCount, run.ErrorSummary, string(configSummary))

	created, err := scanGraphProjectionRun(row)
	if err != nil {
		return GraphProjectionRun{}, fmt.Errorf("create graph projection run: %w", err)
	}
	return created, nil
}

func (r PostgresRepository) RecordGraphProjectionRunItem(ctx context.Context, item GraphProjectionRunItem) error {
	item.ID = NormalizeUUID(item.ID)
	item.RunID = NormalizeUUID(item.RunID)

	_, err := r.db.ExecContext(ctx, `
INSERT INTO graph_projection_run_items (
    id, run_id, item_type, item_key, status, error_message
) VALUES (
    $1, $2, $3, $4, $5, $6
)
`, item.ID, item.RunID, item.ItemType, item.ItemKey, item.Status, item.ErrorMessage)
	if err != nil {
		return fmt.Errorf("record graph projection run item: %w", err)
	}
	return nil
}

func (r PostgresRepository) CompleteGraphProjectionRun(ctx context.Context, run GraphProjectionRun) error {
	run.ID = NormalizeUUID(run.ID)
	run = normalizeGraphProjectionRun(run)
	configSummary, err := json.Marshal(cloneMap(run.ConfigSummary))
	if err != nil {
		return fmt.Errorf("marshal completed graph projection config summary: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE graph_projection_runs
SET status = $2,
    finished_at = $3,
    source_row_count = $4,
    projected_count = $5,
    skipped_count = $6,
    failed_count = $7,
    error_summary = $8,
    config_summary = $9::jsonb,
    updated_at = now()
WHERE id = $1
`, run.ID, run.Status, nullTime(run.FinishedAt), run.SourceRowCount, run.ProjectedCount,
		run.SkippedCount, run.FailedCount, run.ErrorSummary, string(configSummary))
	if err != nil {
		return fmt.Errorf("complete graph projection run: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read completed graph projection run affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("graph projection run %q not found", run.ID)
	}
	return nil
}

func (r PostgresRepository) RecentGraphProjectionRuns(ctx context.Context, limit int) ([]GraphProjectionRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, projection_type, mode, status, started_at, finished_at,
       source_row_count, projected_count, skipped_count, failed_count,
       error_summary, config_summary
FROM graph_projection_runs
ORDER BY started_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent graph projection runs: %w", err)
	}
	defer rows.Close()

	runs := make([]GraphProjectionRun, 0)
	for rows.Next() {
		run, err := scanGraphProjectionRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent graph projection runs: %w", err)
	}
	return runs, nil
}

func scanGraphProjectionRun(scanner rawDocumentScanner) (GraphProjectionRun, error) {
	var run GraphProjectionRun
	var finishedAt sql.NullTime
	var configSummaryBytes []byte
	if err := scanner.Scan(
		&run.ID,
		&run.ProjectionType,
		&run.Mode,
		&run.Status,
		&run.StartedAt,
		&finishedAt,
		&run.SourceRowCount,
		&run.ProjectedCount,
		&run.SkippedCount,
		&run.FailedCount,
		&run.ErrorSummary,
		&configSummaryBytes,
	); err != nil {
		return GraphProjectionRun{}, err
	}
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	if len(configSummaryBytes) > 0 {
		if err := json.Unmarshal(configSummaryBytes, &run.ConfigSummary); err != nil {
			return GraphProjectionRun{}, fmt.Errorf("parse graph projection config summary: %w", err)
		}
	}
	if run.ConfigSummary == nil {
		run.ConfigSummary = map[string]any{}
	}
	return run, nil
}
