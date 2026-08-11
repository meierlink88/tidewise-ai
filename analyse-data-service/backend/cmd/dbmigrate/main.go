package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/dbmigration"
)

func main() {
	options, err := parseCLIOptions(os.Args[1:])
	if err != nil {
		log.Fatalf("parse migration options: %v", err)
	}

	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// The database connect timeout belongs to individual connection attempts in
	// the PostgreSQL DSN. A complete forward migration chain can legitimately
	// take longer, especially on a fresh database, so it must not reuse that
	// value as an end-to-end deadline.
	report, err := dbmigration.CheckPostgresWithOptions(context.Background(), cfg, options)
	if err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("encode migration report: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(content))
}

func parseCLIOptions(args []string) (dbmigration.ServiceOptions, error) {
	flags := flag.NewFlagSet("dbmigrate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	apply := flags.Bool("apply", false, "apply pending migrations")
	targetVersion := flags.String("target-version", "", "apply pending migrations up to this exact version")
	if err := flags.Parse(args); err != nil {
		return dbmigration.ServiceOptions{}, err
	}
	if *targetVersion != "" && !*apply {
		return dbmigration.ServiceOptions{}, fmt.Errorf("-target-version requires -apply")
	}
	return dbmigration.ServiceOptions{AutoApply: *apply, TargetVersion: *targetVersion}, nil
}
