package usecase

import (
	"context"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/materialization"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/planning"
	collectorworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/workflow"
)

type runtimeFactory struct {
	store            Repository
	modelFactory     ModelFactory
	connectorFactory ConnectorFactory
	artifacts        ArtifactStore
	now              func() time.Time
}

func (f runtimeFactory) Build(ctx context.Context, executionID string, runtimeConfig collector.RuntimeConfiguration) (compose.Runnable[*collector.Request, *collector.Result], error) {
	model, err := f.modelFactory.New(ctx, runtimeConfig.ModelProvider)
	if err != nil {
		return nil, err
	}
	planner, err := planning.NewDeepSeekQueryPlanner(model)
	if err != nil {
		return nil, err
	}
	plannerWithState := &trackingPlanner{executionID: executionID, store: f.store, delegate: planner, now: f.now}

	connectorSet, err := f.connectorFactory.New(runtimeConfig)
	if err != nil {
		return nil, err
	}
	tracked := make([]collector.Connector, 0, len(connectorSet))
	for _, connector := range connectorSet {
		tracked = append(tracked, &trackingConnector{executionID: executionID, store: f.store, delegate: connector, now: f.now})
	}
	materializer := &trackingMaterializer{
		executionID: executionID, store: f.store, now: f.now,
		delegate: f.artifacts.Materializer(materialization.DefaultNearDuplicateRadius),
	}
	return collectorworkflow.New(ctx, plannerWithState, tracked, maxParallel, materializer)
}
