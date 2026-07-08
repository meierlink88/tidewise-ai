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
	"github.com/meierlink88/tidewise-ai/backend/internal/database"
	"github.com/meierlink88/tidewise-ai/backend/internal/entityseed"
)

func main() {
	seedDir := flag.String("seed-dir", entityseed.DefaultSeedDir, "entity foundation seed directory")
	includeInactive := flag.Bool("include-inactive", false, "include inactive entities in seed writes")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	timeout := time.Duration(cfg.Database.ConnectTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	manifest, err := entityseed.LoadFiles(entityseed.DefaultSeedPaths(*seedDir)...)
	if err != nil {
		log.Fatalf("load entity seed files: %v", err)
	}

	db, err := database.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	service := entityseed.NewService(entityseed.NewPostgresRepository(db))
	report, err := service.Apply(ctx, manifest, entityseed.ApplyOptions{IncludeInactive: *includeInactive})
	if err != nil {
		log.Fatalf("apply entity seed: %v", err)
	}

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("encode entity seed report: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(content))
}
