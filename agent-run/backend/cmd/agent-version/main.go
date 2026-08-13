package main

import (
	"context"
	"fmt"
	"os"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	agentrunconfig "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
)

const usage = "usage: agentrun-agent-version publish-current"

var currentAgentVersions = []agentrun.AgentVersion{
	{AgentKey: collector.AgentKey, Version: collector.AgentVersion},
	{AgentKey: eventfact.AgentKey, Version: eventfact.AgentVersion},
	{AgentKey: eventsemantic.AgentKey, Version: eventsemantic.AgentVersion},
}

func main() {
	if len(os.Args) != 2 || os.Args[1] != "publish-current" {
		fail(usage)
	}
	cfg, err := agentrunconfig.LoadDatabaseOperation()
	if err != nil {
		fail("could not load AgentRun configuration")
	}
	databaseURL, err := cfg.PostgresURL()
	if err != nil {
		fail("could not build AgentRun database configuration")
	}
	ctx := context.Background()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		fail("could not open AgentRun database")
	}
	defer database.Close()
	store := postgres.New(database)
	if !store.SchemaReady(ctx) {
		fail("AgentRun database schema is not ready; run migrations first")
	}
	if err := store.PublishAgentVersions(ctx, currentAgentVersions); err != nil {
		fail("could not publish current Agent Versions")
	}
	fmt.Println("AgentRun current Agent Versions are published")
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
