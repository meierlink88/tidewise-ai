package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/migrations"
)

func main() {
	apply := flag.Bool("apply", false, "apply pending migrations")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	timeout := time.Duration(cfg.Database.ConnectTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	report, err := migrations.CheckPostgres(ctx, cfg, *apply)
	if err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("encode migration report: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(content))
}
