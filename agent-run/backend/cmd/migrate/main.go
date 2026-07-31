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
	preparePreviousReleaseRollback := flag.Bool(
		"prepare-previous-release-rollback",
		false,
		"remove the compatibility-safe 010 ledger marker before restoring a pre-010 release",
	)
	flag.Parse()
	if *checkOnly && *preparePreviousReleaseRollback {
		fail("migration operation flags are mutually exclusive")
	}

	cfg, err := agentrunconfig.LoadDatabaseOperation()
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
	if *preparePreviousReleaseRollback {
		if err := postgres.PreparePreviousReleaseRollback(
			context.Background(), database,
		); err != nil {
			fail("could not prepare the previous AgentRun release rollback")
		}
		fmt.Println("AgentRun database is compatible with the previous release")
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
