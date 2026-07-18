package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	ingestionsourcecatalog "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/sourcecatalog"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/database"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func main() {
	seedDir := flag.String("seed-dir", ingestionsourcecatalog.DefaultSeedDir, "source catalog seed directory")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	timeout := time.Duration(cfg.Database.ConnectTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	manifest, err := ingestionsourcecatalog.LoadFiles(ingestionsourcecatalog.DefaultSeedPaths(*seedDir)...)
	if err != nil {
		log.Fatalf("load source catalog seed files: %v", err)
	}

	db, err := database.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	service := ingestionsourcecatalog.NewService(repositories.NewPostgresRepository(db))
	report, err := service.Apply(ctx, manifest)
	if err != nil {
		log.Fatalf("apply source catalog seed: %v", err)
	}

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("encode source catalog seed report: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(content))
}
