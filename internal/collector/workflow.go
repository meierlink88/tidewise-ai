package collector

import (
	"context"
	"fmt"
	"regexp"

	"github.com/cloudwego/eino/compose"
)

var nodeNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

const queryPlannerNode = "plan_queries"

func NewWorkflow(
	ctx context.Context,
	planner QueryPlanner,
	connectors []Connector,
	maxParallel int,
	materializer Materializer,
) (compose.Runnable[*Request, *Result], error) {
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

	workflow := compose.NewWorkflow[*Request, *Result]()
	workflow.AddLambdaNode(queryPlannerNode, compose.InvokableLambda(
		func(ctx context.Context, request *Request) (*Request, error) {
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
			func(ctx context.Context, request *Request) (ConnectorRun, error) {
				select {
				case semaphore <- struct{}{}:
					defer func() { <-semaphore }()
				case <-ctx.Done():
					return ConnectorRun{}, ctx.Err()
				}

				results, err := connector.Collect(ctx, *request)
				run := ConnectorRun{Connector: connector.Name(), Results: results}
				if err != nil {
					run.Error = err.Error()
				}
				for index := range run.Results {
					run.Results[index].Connector = connector.Name()
					run.Results[index].ContentOrigin = ContentOrigin
				}
				return run, nil
			},
		)).AddInput(queryPlannerNode)
	}

	aggregate := workflow.AddLambdaNode("materialize", compose.InvokableLambda(
		func(ctx context.Context, runs map[string]ConnectorRun) (*Result, error) {
			var request *Request
			for _, run := range runs {
				_ = run
			}
			if value := ctx.Value(requestContextKey{}); value != nil {
				request, _ = value.(*Request)
			}
			if request == nil {
				return nil, fmt.Errorf("collector request missing from context")
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
	return requestRunnable{Runnable: runnable}, nil
}

type requestContextKey struct{}

type requestRunnable struct {
	compose.Runnable[*Request, *Result]
}

func (r requestRunnable) Invoke(ctx context.Context, input *Request, opts ...compose.Option) (*Result, error) {
	return r.Runnable.Invoke(context.WithValue(ctx, requestContextKey{}, input), input, opts...)
}
