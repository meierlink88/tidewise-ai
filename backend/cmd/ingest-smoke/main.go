package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/database"
	"github.com/meierlink88/tidewise-ai/backend/internal/jobs"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func main() {
	sourceURL := flag.String("source-url", firstEnv("TIDEWISE_SMOKE_SOURCE_URL"), "RSS source URL for local ingestion smoke")
	sourceName := flag.String("source-name", firstEnv("TIDEWISE_SMOKE_SOURCE_NAME"), "RSS source name for local ingestion smoke")
	maxDocuments := flag.Int("max-documents", 3, "maximum documents to write")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	timeout := time.Duration(cfg.Ingestion.DefaultTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	db, err := database.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	repo := repositories.NewPostgresRepository(db)
	runner := jobs.NewIngestionSmokeRunner(repo, &http.Client{Timeout: timeout})
	report, err := runner.Run(ctx, jobs.IngestionSmokeOptions{
		SourceURL:    *sourceURL,
		SourceName:   *sourceName,
		MaxDocuments: *maxDocuments,
		Timeout:      timeout,
	})
	if err != nil {
		content, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr == nil {
			fmt.Fprintln(os.Stdout, string(content))
		}
		log.Fatalf("run ingestion smoke: %v", err)
	}

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("encode smoke report: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(content))
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
