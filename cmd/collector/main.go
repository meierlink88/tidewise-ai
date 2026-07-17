package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	projectconfig "github.com/guanchaojia/tidewise-ai-agentrun/internal/config"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/connectors"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/materialize"
)

func main() {
	var objective, queryList, dataRoot, envFile string
	var candidateLimit, timeWindow, maxParallel int
	flag.StringVar(&objective, "objective", "采集最近全球政经、政策、产业、企业和资本市场重要信息，优先服务中国资本市场研究", "search objective")
	flag.StringVar(&queryList, "queries", "全球政经政策产业新闻,中国资本市场重要新闻", "comma-separated search queries")
	flag.StringVar(&dataRoot, "data-root", "data", "output root")
	flag.StringVar(&envFile, "env-file", ".env", "path to an optional dotenv configuration file")
	flag.IntVar(&candidateLimit, "candidate-limit", 10, "maximum results per connector")
	flag.IntVar(&timeWindow, "time-window-hours", 48, "collection time window")
	flag.IntVar(&maxParallel, "max-parallel", 3, "maximum concurrent connectors")
	flag.Parse()
	if err := projectconfig.LoadEnvFile(envFile); err != nil {
		fail(fmt.Errorf("load configuration: %w", err))
	}

	now := time.Now().UTC()
	request := &collector.Request{
		RunID: now.Format("20060102T150405Z"), Objective: objective,
		SearchQueries: splitQueries(queryList), CandidateLimit: candidateLimit,
		TimeWindowHours: timeWindow, CollectedAt: now,
	}
	connectorSet := []collector.Connector{
		connectors.ParallelSearch{APIKey: os.Getenv("PARALLEL_API_KEY")},
		connectors.Tavily{APIKey: os.Getenv("TAVILY_API_KEY")},
		connectors.Bocha{APIKey: os.Getenv("BOCHA_API_KEY")},
	}

	workflow, err := collector.NewWorkflow(context.Background(), connectorSet, maxParallel, materialize.File{Root: dataRoot, NearDuplicateRadius: 3})
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

func splitQueries(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if cleaned := strings.TrimSpace(item); cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
