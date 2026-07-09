package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
	sourceID       string
	providerKey    string
	ingestChannel  string
	sourceType     string
	concurrency    int
	timeoutSeconds int
	rsshubBaseURL  string
	promptRoot     string
	requiredEnv    string
}

func main() {
	options := sourceIngestOptions{}
	flag.StringVar(&options.sourceID, "source-id", "", "filter active source catalogs by id")
	flag.StringVar(&options.providerKey, "provider", "", "filter active source catalogs by provider_key")
	flag.StringVar(&options.ingestChannel, "channel", "", "filter active source catalogs by ingest_channel")
	flag.StringVar(&options.sourceType, "source-type", "", "filter active source catalogs by source_type")
	flag.IntVar(&options.concurrency, "concurrency", 1, "maximum source concurrency")
	flag.IntVar(&options.timeoutSeconds, "timeout-seconds", 0, "override ingestion timeout seconds for this run")
	flag.StringVar(&options.rsshubBaseURL, "rsshub-base-url", firstEnv("TIDEWISE_RSSHUB_BASE_URL"), "RSSHub base URL for rsshub_feed sources")
	flag.StringVar(&options.promptRoot, "prompt-root", defaultPromptRoot(), "repo prompt root for AI ingestion sources")
	flag.StringVar(&options.requiredEnv, "require-env", "", "comma-separated environment variables required before running gated ingestion smoke")
	flag.Parse()

	if missing := missingRequiredEnvNames(parseRequiredEnvNames(options.requiredEnv), os.Getenv); len(missing) > 0 {
		log.Fatalf("missing required environment variables: %s", strings.Join(missing, ","))
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	timeout := time.Duration(ingestionTimeoutSeconds(options, cfg.Ingestion.DefaultTimeoutSeconds)) * time.Second
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
	ingestionconnectors.RegisterAIWebResearchConnectors(registry, ingestionconnectors.AIWebResearchRegistryOptions{
		Client:             client,
		PromptRoot:         options.promptRoot,
		CredentialResolver: core.EnvCredentialResolver{},
	})
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
		SourceID:      options.sourceID,
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

func ingestionTimeoutSeconds(options sourceIngestOptions, defaultSeconds int) int {
	if options.timeoutSeconds > 0 {
		return options.timeoutSeconds
	}
	return defaultSeconds
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func defaultPromptRoot() string {
	if value := firstEnv("TIDEWISE_PROMPT_ROOT"); value != "" {
		return value
	}
	for _, candidate := range []string{"data/prompts", "backend/data/prompts"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "data/prompts"
}

func parseRequiredEnvNames(value string) []string {
	parts := strings.Split(value, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func missingRequiredEnvNames(names []string, lookup func(string) string) []string {
	missing := make([]string, 0)
	for _, name := range names {
		if lookup(name) == "" {
			missing = append(missing, name)
		}
	}
	return missing
}
