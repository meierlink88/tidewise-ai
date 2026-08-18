package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/dbmigration"
)

func main() {
	options, err := parseCLIOptions(os.Args[1:])
	if err != nil {
		log.Fatalf("parse migration options: %v", err)
	}
	if err := validateEmptySchemaRebuildConfirmation(options, os.Getenv); err != nil {
		log.Fatalf("validate migration options: %v", err)
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
	rebuildEmptySchema := flags.Bool("rebuild-empty-schema", false, "drop and rebuild only the Data public schema before applying migrations")
	if err := flags.Parse(args); err != nil {
		return dbmigration.ServiceOptions{}, err
	}
	if *targetVersion != "" && !*apply {
		return dbmigration.ServiceOptions{}, fmt.Errorf("-target-version requires -apply")
	}
	if *rebuildEmptySchema && (!*apply || *targetVersion != "58") {
		return dbmigration.ServiceOptions{}, fmt.Errorf("-rebuild-empty-schema requires -apply -target-version 58")
	}
	return dbmigration.ServiceOptions{AutoApply: *apply, TargetVersion: *targetVersion, RebuildEmptySchema: *rebuildEmptySchema}, nil
}

func validateEmptySchemaRebuildConfirmation(options dbmigration.ServiceOptions, getenv func(string) string) error {
	if !options.RebuildEmptySchema {
		return nil
	}
	if getenv("TIDEWISE_EMPTY_DATA_SCHEMA_REBUILD_CONFIRMED") != "issue-266-data-only" {
		return fmt.Errorf("empty Data schema rebuild confirmation is missing")
	}
	return nil
}
