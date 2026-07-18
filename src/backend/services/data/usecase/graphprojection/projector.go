package graphprojection

import (
	"context"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

type Clock func() time.Time

type ProjectOptions struct {
	Mode repositories.GraphProjectionMode
}

type Projector struct {
	repository repositories.GraphProjectionRepository
	writer     GraphWriter
	namespace  string
	clock      Clock
}

func NewProjector(repository repositories.GraphProjectionRepository, writer GraphWriter, namespace string, clock Clock) Projector {
	if namespace == "" {
		namespace = "tidewise"
	}
	if clock == nil {
		clock = time.Now
	}
	return Projector{
		repository: repository,
		writer:     writer,
		namespace:  namespace,
		clock:      clock,
	}
}

func (p Projector) ProjectEntities(ctx context.Context, options ProjectOptions) (repositories.GraphProjectionRun, error) {
	if options.Mode == "" {
		options.Mode = repositories.GraphProjectionModeProjectEntities
	}
	started := p.clock()
	run := repositories.GraphProjectionRun{
		ID:             repositories.NormalizeUUID("graph_projection_run", string(options.Mode), started.Format(time.RFC3339Nano)),
		ProjectionType: repositories.GraphProjectionTypeEntityGraph,
		Mode:           options.Mode,
		Status:         repositories.GraphProjectionRunStatusRunning,
		StartedAt:      started,
		ConfigSummary:  map[string]any{"namespace": p.namespace},
	}
	created, err := p.repository.CreateGraphProjectionRun(ctx, run)
	if err != nil {
		return repositories.GraphProjectionRun{}, err
	}
	run = created

	if options.Mode == repositories.GraphProjectionModeRebuildEntities {
		if err := p.writer.DeleteNamespace(ctx, p.namespace); err != nil {
			return p.finishWithError(ctx, run, 1, err)
		}
	}

	nodes, err := p.repository.ListGraphEntityNodes(ctx)
	if err != nil {
		return p.finishWithError(ctx, run, 1, err)
	}
	edges, err := p.repository.ListGraphEntityEdges(ctx)
	if err != nil {
		return p.finishWithError(ctx, run, 1, err)
	}
	nodes, edges = activeProjectionSource(nodes, edges)
	run.SourceRowCount = len(nodes) + len(edges)

	graphNodes := make([]GraphNode, 0, len(nodes))
	nodeIndex := map[string]GraphNode{}
	for _, node := range nodes {
		graphNode, err := MapEntityNode(node, p.namespace)
		if err != nil {
			run.FailedCount++
			_ = p.repository.RecordGraphProjectionRunItem(ctx, repositories.GraphProjectionRunItem{
				ID:           repositories.NormalizeUUID(run.ID, "entity", node.ID),
				RunID:        run.ID,
				ItemType:     repositories.GraphProjectionRunItemTypeEntity,
				ItemKey:      node.ID,
				Status:       repositories.GraphProjectionRunItemStatusFailed,
				ErrorMessage: err.Error(),
			})
			continue
		}
		graphNodes = append(graphNodes, graphNode)
		nodeIndex[graphNode.EntityID] = graphNode
	}

	if _, err := p.writer.UpsertEntities(ctx, graphNodes); err != nil {
		failed := len(graphNodes)
		if failed == 0 {
			failed = 1
		}
		return p.finishWithError(ctx, run, failed, err)
	}
	run.ProjectedCount += len(graphNodes)

	graphRelationships, relationReports := MapEntityRelationships(edges, nodeIndex, p.namespace)
	for _, report := range relationReports {
		switch report.Status {
		case RelationshipMapStatusSkipped:
			run.SkippedCount++
			_ = p.repository.RecordGraphProjectionRunItem(ctx, repositories.GraphProjectionRunItem{
				ID:           repositories.NormalizeUUID(run.ID, "relationship", report.EdgeID),
				RunID:        run.ID,
				ItemType:     repositories.GraphProjectionRunItemTypeRelationship,
				ItemKey:      report.EdgeID,
				Status:       repositories.GraphProjectionRunItemStatusSkipped,
				ErrorMessage: report.Reason,
			})
		case RelationshipMapStatusFailed:
			run.FailedCount++
			_ = p.repository.RecordGraphProjectionRunItem(ctx, repositories.GraphProjectionRunItem{
				ID:           repositories.NormalizeUUID(run.ID, "relationship", report.EdgeID),
				RunID:        run.ID,
				ItemType:     repositories.GraphProjectionRunItemTypeRelationship,
				ItemKey:      report.EdgeID,
				Status:       repositories.GraphProjectionRunItemStatusFailed,
				ErrorMessage: report.Reason,
			})
		}
	}

	if _, err := p.writer.UpsertRelationships(ctx, graphRelationships); err != nil {
		failed := len(graphRelationships)
		if failed == 0 {
			failed = 1
		}
		return p.finishWithError(ctx, run, failed, err)
	}
	run.ProjectedCount += len(graphRelationships)

	if run.FailedCount > 0 || run.SkippedCount > 0 {
		run.Status = repositories.GraphProjectionRunStatusPartial
	} else {
		run.Status = repositories.GraphProjectionRunStatusSucceeded
	}
	return p.finish(ctx, run)
}

func activeProjectionSource(nodes []repositories.GraphEntityNode, edges []repositories.GraphEntityEdge) ([]repositories.GraphEntityNode, []repositories.GraphEntityEdge) {
	activeNodes := make([]repositories.GraphEntityNode, 0, len(nodes))
	nodeStatuses := make(map[string]domain.Status, len(nodes))
	for _, node := range nodes {
		nodeStatuses[node.ID] = node.Status
		if node.Status != domain.StatusActive {
			continue
		}
		activeNodes = append(activeNodes, node)
	}
	activeEdges := make([]repositories.GraphEntityEdge, 0, len(edges))
	for _, edge := range edges {
		if edge.Status != domain.StatusActive {
			continue
		}
		fromStatus, fromKnown := nodeStatuses[edge.FromEntityID]
		toStatus, toKnown := nodeStatuses[edge.ToEntityID]
		if (fromKnown && fromStatus != domain.StatusActive) || (toKnown && toStatus != domain.StatusActive) {
			continue
		}
		activeEdges = append(activeEdges, edge)
	}
	return activeNodes, activeEdges
}

func (p Projector) finishWithError(ctx context.Context, run repositories.GraphProjectionRun, failedCount int, err error) (repositories.GraphProjectionRun, error) {
	run.Status = repositories.GraphProjectionRunStatusFailed
	run.FailedCount += failedCount
	run.ErrorSummary = err.Error()
	finished, finishErr := p.finish(ctx, run)
	if finishErr != nil {
		return finished, fmt.Errorf("%w; finish graph projection run: %v", err, finishErr)
	}
	return finished, err
}

func (p Projector) finish(ctx context.Context, run repositories.GraphProjectionRun) (repositories.GraphProjectionRun, error) {
	finishedAt := p.clock()
	run.FinishedAt = &finishedAt
	if err := p.repository.CompleteGraphProjectionRun(ctx, run); err != nil {
		return run, err
	}
	return run, nil
}
