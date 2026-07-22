package main

import (
	"context"
	"fmt"
	"os"

	agentrunconfig "github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/config"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
)

func main() {
	cfg, err := agentrunconfig.Load()
	if err != nil {
		fail("could not load AgentRun configuration")
	}
	databaseURL, err := cfg.PostgresURL()
	if err != nil {
		fail("could not build AgentRun database configuration")
	}
	database, err := postgres.Open(context.Background(), databaseURL)
	if err != nil {
		fail("could not open AgentRun database")
	}
	defer database.Close()
	if err := postgres.Migrate(context.Background(), database); err != nil {
		fail("could not migrate AgentRun database")
	}
	fmt.Println("AgentRun database migrations are current")
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
