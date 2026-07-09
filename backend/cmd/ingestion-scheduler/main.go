package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	ingestionconnectors "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/connectors"
	"github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
	ingestionparsers "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/parsers"
	ingestionruntime "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/runtime"
	"github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/scheduler"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/database"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type schedulerOptions struct {
	once          bool
	dryRun        bool
	tickSeconds   int
	rsshubBaseURL string
	promptRoot    string
}

func main() {
	options, err := parseSchedulerOptions(os.Args[1:])
	if err != nil {
		log.Fatalf("parse options: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	db, err := database.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	timeout := time.Duration(cfg.Ingestion.DefaultTimeoutSeconds) * time.Second
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
	runner := runtimeRunner{
		sources:     core.NewSourceRegistry(repo),
		registry:    registry,
		credentials: core.EnvCredentialResolver{},
		limiter:     core.NewRateLimiter(),
		writer:      core.NewRawDocumentWriter(repo),
	}
	service := scheduler.NewService(repo, runner, scheduler.ServiceOptions{})

	if options.dryRun {
		config, err := repo.LoadSchedulerConfig(ctx)
		if err != nil {
			log.Fatalf("load scheduler config: %v", err)
		}
		printJSON(config)
		return
	}

	if options.once {
		report, err := service.RunOnce(ctx, domain.SchedulerTriggerManualOnce)
		if err != nil {
			log.Fatalf("run scheduler once: %v", err)
		}
		printJSON(report)
		if report.Status == domain.SchedulerRunStatusFailed {
			os.Exit(1)
		}
		return
	}

	tick := time.Duration(normalizeTickSeconds(options.tickSeconds, cfg.Ingestion.SchedulerTickSeconds)) * time.Second
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			report, err := service.RunDue(ctx)
			if err != nil {
				log.Printf("scheduler tick failed: %v", err)
				continue
			}
			if report.Status != domain.SchedulerRunStatusSkipped {
				printJSON(report)
			}
		}
	}
}

func parseSchedulerOptions(args []string) (schedulerOptions, error) {
	options := schedulerOptions{}
	flags := flag.NewFlagSet("ingestion-scheduler", flag.ContinueOnError)
	flags.BoolVar(&options.once, "once", false, "run one scheduler cycle")
	flags.BoolVar(&options.dryRun, "dry-run", false, "print scheduler config without running ingestion")
	flags.IntVar(&options.tickSeconds, "tick-seconds", 0, "override scheduler loop tick seconds")
	flags.StringVar(&options.rsshubBaseURL, "rsshub-base-url", firstEnv("TIDEWISE_RSSHUB_BASE_URL"), "RSSHub base URL for rsshub_feed sources")
	flags.StringVar(&options.promptRoot, "prompt-root", defaultPromptRoot(), "repo prompt root for AI ingestion sources")
	if err := flags.Parse(args); err != nil {
		return schedulerOptions{}, err
	}
	return options, nil
}

func normalizeTickSeconds(value int, defaultSeconds int) int {
	if value > 0 {
		return value
	}
	return defaultSeconds
}

type runtimeRunner struct {
	sources     ingestionruntime.SourceRegistry
	registry    *core.Registry
	credentials ingestionruntime.CredentialResolver
	limiter     ingestionruntime.RateLimiter
	writer      ingestionruntime.RawDocumentWriter
}

func (r runtimeRunner) Run(ctx context.Context, filter repositories.SourceCatalogFilter, options scheduler.RunOptions) ingestionruntime.IngestionReport {
	if options.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	job := ingestionruntime.NewIngestionJobWithOptions(
		r.sources,
		r.registry,
		r.credentials,
		r.limiter,
		r.writer,
		ingestionruntime.IngestionJobOptions{Concurrency: options.Concurrency},
	)
	return job.Run(ctx, filter)
}

func printJSON(value any) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatalf("encode json: %v", err)
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
