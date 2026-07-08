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

	ingestionconnectors "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/connectors"
	"github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
	ingestionparsers "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/parsers"
	ingestionruntime "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/runtime"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/database"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type sourceIngestOptions struct {
	providerKey   string
	ingestChannel string
	sourceType    string
	concurrency   int
	rsshubBaseURL string
}

func main() {
	options := sourceIngestOptions{}
	flag.StringVar(&options.providerKey, "provider", "", "filter active source catalogs by provider_key")
	flag.StringVar(&options.ingestChannel, "channel", "", "filter active source catalogs by ingest_channel")
	flag.StringVar(&options.sourceType, "source-type", "", "filter active source catalogs by source_type")
	flag.IntVar(&options.concurrency, "concurrency", 1, "maximum source concurrency")
	flag.StringVar(&options.rsshubBaseURL, "rsshub-base-url", firstEnv("TIDEWISE_RSSHUB_BASE_URL"), "RSSHub base URL for rsshub_feed sources")
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

	client := &http.Client{Timeout: timeout}
	registry := core.NewRegistry()
	ingestionconnectors.RegisterContentConnectors(registry, client, options.rsshubBaseURL)
	registry.RegisterConnector("eastmoney", ingestionconnectors.EastmoneyConnector{Client: client})
	registry.RegisterParser("eastmoney_json", ingestionparsers.EastmoneyJSONParser{})

	repo := repositories.NewPostgresRepository(db)
	job := ingestionruntime.NewIngestionJobWithOptions(
		core.NewSourceRegistry(repo),
		registry,
		core.EnvCredentialResolver{},
		core.NewRateLimiter(),
		core.NewRawDocumentWriter(repo),
		ingestionruntime.IngestionJobOptions{Concurrency: normalizeConcurrency(options.concurrency)},
	)

	report := job.Run(ctx, sourceCatalogFilter(options))
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("encode ingestion report: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(content))
	if report.Failed > 0 {
		log.Fatalf("source ingest failed: %v", report.Errors)
	}
}

func sourceCatalogFilter(options sourceIngestOptions) repositories.SourceCatalogFilter {
	return repositories.SourceCatalogFilter{
		ProviderKey:   options.providerKey,
		IngestChannel: options.ingestChannel,
		SourceType:    options.sourceType,
	}
}

func normalizeConcurrency(value int) int {
	if value < 1 {
		return 1
	}
	return value
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
