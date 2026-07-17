package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	projectconfig "github.com/guanchaojia/tidewise-ai-agentrun/internal/config"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/connectors"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/materialize"
)

const defaultCollectorPromptFile = "agents/collector/prompts/query_planner_v1.md"

type collectorOptions struct {
	promptFile     string
	dataRoot       string
	envFile        string
	candidateLimit int
	timeWindow     int
	maxParallel    int
}

func parseCollectorOptions(arguments []string) (collectorOptions, error) {
	var options collectorOptions
	flags := flag.NewFlagSet("collector", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.promptFile, "prompt-file", defaultCollectorPromptFile, "path to the collector intent prompt")
	flags.StringVar(&options.dataRoot, "data-root", "data", "output root")
	flags.StringVar(&options.envFile, "env-file", ".env", "path to an optional dotenv configuration file")
	flags.IntVar(&options.candidateLimit, "candidate-limit", 10, "maximum results per connector")
	flags.IntVar(&options.timeWindow, "time-window-hours", 48, "collection time window")
	flags.IntVar(&options.maxParallel, "max-parallel", 3, "maximum concurrent connectors")
	if err := flags.Parse(arguments); err != nil {
		return collectorOptions{}, err
	}
	return options, nil
}

func prepareCollector(
	ctx context.Context,
	options collectorOptions,
	now time.Time,
	factory deepSeekModelFactory,
) (*collector.Request, collector.QueryPlanner, error) {
	objective, err := collector.LoadCollectorPrompt(options.promptFile)
	if err != nil {
		return nil, nil, err
	}
	deepSeekConfig, err := projectconfig.LoadDeepSeekConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("load DeepSeek configuration: %w", err)
	}
	planner, err := buildDeepSeekPlanner(ctx, deepSeekConfig, factory)
	if err != nil {
		return nil, nil, err
	}
	collectedAt := now.UTC()
	request := &collector.Request{
		RunID: collectedAt.Format("20060102T150405Z"), Objective: objective,
		CandidateLimit: options.candidateLimit, TimeWindowHours: options.timeWindow,
		CollectedAt: collectedAt,
	}
	return request, planner, nil
}

func main() {
	options, err := parseCollectorOptions(os.Args[1:])
	if err != nil {
		fail(err)
	}
	if err = projectconfig.LoadEnvFile(options.envFile); err != nil {
		fail(fmt.Errorf("load configuration: %w", err))
	}
	request, planner, err := prepareCollector(context.Background(), options, time.Now(), newDeepSeekChatModel)
	if err != nil {
		fail(err)
	}

	connectorSet := []collector.Connector{
		connectors.ParallelSearch{APIKey: os.Getenv("PARALLEL_API_KEY")},
		connectors.Tavily{APIKey: os.Getenv("TAVILY_API_KEY")},
		connectors.Bocha{APIKey: os.Getenv("BOCHA_API_KEY")},
	}

	workflow, err := collector.NewWorkflow(context.Background(), planner, connectorSet, options.maxParallel, materialize.File{Root: options.dataRoot, NearDuplicateRadius: 3})
	if err != nil {
		fail(err)
	}
	result, err := workflow.Invoke(context.Background(), request)
	if err != nil {
		fail(err)
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
