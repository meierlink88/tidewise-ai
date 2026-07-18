package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/dbmigration"
)

func main() {
	options, err := parseCLIOptions(os.Args[1:])
	if err != nil {
		log.Fatalf("parse migration options: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	timeout := time.Duration(cfg.Database.ConnectTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	report, err := dbmigration.CheckPostgresWithOptions(ctx, cfg, options)
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
