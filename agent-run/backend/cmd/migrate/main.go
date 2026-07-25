package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	agentrunconfig "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
)

func main() {
	checkOnly := flag.Bool("check-only", false, "print a read-only JSON migration report")
	flag.Parse()

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
	if *checkOnly {
		report, err := postgres.InspectMigrations(context.Background(), database)
		if err != nil {
			fail("could not inspect AgentRun database migrations")
		}
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			fail("could not encode AgentRun migration report")
		}
		return
	}
	if err := postgres.Migrate(context.Background(), database); err != nil {
		fail("could not migrate AgentRun database")
	}
	fmt.Println("AgentRun database migrations are current")
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
