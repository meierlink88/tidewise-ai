package workflow

import (
	"context"
	"fmt"
	"regexp"

	"github.com/cloudwego/eino/compose"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/planning"
)

var nodeNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

const queryPlannerNode = "plan_queries"

func New(
	ctx context.Context,
	planner planning.QueryPlanner,
	connectors []collector.Connector,
	maxParallel int,
	materializer collector.Materializer,
) (compose.Runnable[*collector.Request, *collector.Result], error) {
	if planner == nil {
		return nil, fmt.Errorf("query planner is required")
	}
	if len(connectors) == 0 {
		return nil, fmt.Errorf("at least one connector is required")
	}
	if maxParallel <= 0 {
		return nil, fmt.Errorf("maxParallel must be positive")
	}
	if materializer == nil {
		return nil, fmt.Errorf("materializer is required")
	}

	workflow := compose.NewWorkflow[*collector.Request, *collector.Result]()
	workflow.AddLambdaNode(queryPlannerNode, compose.InvokableLambda(
		func(ctx context.Context, request *collector.Request) (*collector.Request, error) {
			return planner.Plan(ctx, request)
		},
	)).AddInput(compose.START)
	semaphore := make(chan struct{}, maxParallel)
	names := make(map[string]struct{}, len(connectors))

	for _, item := range connectors {
		connector := item
		name := connector.Name()
		if name == queryPlannerNode || name == "materialize" {
			return nil, fmt.Errorf("reserved connector name %q", name)
		}
		if !nodeNamePattern.MatchString(name) {
			return nil, fmt.Errorf("invalid connector name %q", name)
		}
		if _, exists := names[name]; exists {
			return nil, fmt.Errorf("duplicate connector name %q", name)
		}
		names[name] = struct{}{}

		workflow.AddLambdaNode(name, compose.InvokableLambda(
			func(ctx context.Context, request *collector.Request) (collector.ConnectorRun, error) {
				select {
				case semaphore <- struct{}{}:
					defer func() { <-semaphore }()
				case <-ctx.Done():
					return collector.ConnectorRun{}, ctx.Err()
				}

				results, err := connector.Collect(ctx, *request)
				planned := *request
				planned.SearchQueries = append([]string(nil), request.SearchQueries...)
				run := collector.ConnectorRun{Connector: connector.Name(), Request: &planned, Results: results}
				if err != nil {
					run.ErrorCode = "connector_failed"
					run.ErrorSummary = "Connector request failed"
					run.Err = err
				}
				for index := range run.Results {
					run.Results[index].Connector = connector.Name()
					run.Results[index].ContentOrigin = collector.ContentOrigin
					run.Results[index].ResultPosition = index
				}
				return run, nil
			},
		)).AddInput(queryPlannerNode)
	}

	aggregate := workflow.AddLambdaNode("materialize", compose.InvokableLambda(
		func(ctx context.Context, runs map[string]collector.ConnectorRun) (*collector.Result, error) {
			var request *collector.Request
			for _, run := range runs {
				if run.Request != nil {
					request = run.Request
					break
				}
			}
			if request == nil {
				return nil, fmt.Errorf("planned Collector request missing")
			}
			return materializer.Materialize(ctx, *request, runs)
		},
	))
	for name := range names {
		aggregate.AddInput(name, compose.ToField(name))
	}
	workflow.End().AddInput("materialize")

	runnable, err := workflow.Compile(ctx)
	if err != nil {
		return nil, err
	}
	return runnable, nil
}
