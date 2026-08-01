package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	projection "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/semanticprojection"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
	projectiondata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/semanticprojection"
)

type options struct {
	Apply    bool
	AllowEnv string
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	options, err := parseOptions(args)
	if err != nil {
		return err
	}
	config, err := conf.LoadDatabaseOperation()
	if err != nil {
		return fmt.Errorf("load Data configuration: %w", err)
	}
	if !options.Apply || options.AllowEnv != string(config.App.Env) {
		return errors.New("semantic projector requires explicit -apply and matching -allow-env")
	}
	qdrantURL := strings.TrimSpace(os.Getenv("QDRANT_URL"))
	embeddingBaseURL := strings.TrimSpace(os.Getenv("EMBEDDING_BASE_URL"))
	if qdrantURL == "" || embeddingBaseURL == "" {
		return errors.New("QDRANT_URL and EMBEDDING_BASE_URL are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	database, err := postgres.Open(ctx, config)
	if err != nil {
		return errors.New("open PostgreSQL for semantic projection failed")
	}
	defer database.Close()
	source, err := projectiondata.NewPostgresSource(database)
	if err != nil {
		return err
	}
	embedder, err := projectiondata.NewOpenAIEmbedder(projectiondata.HTTPConfig{
		Endpoint: embeddingBaseURL, APIKey: os.Getenv("EMBEDDING_API_KEY"), Timeout: 60 * time.Second,
		MaxResponseBytes: 64 << 20, BatchSize: 10,
	})
	if err != nil {
		return err
	}
	store, err := projectiondata.NewQdrantStore(projectiondata.HTTPConfig{
		Endpoint: qdrantURL, APIKey: os.Getenv("QDRANT_API_KEY"), Timeout: 60 * time.Second,
		MaxResponseBytes: 8 << 20, BatchSize: 128,
	})
	if err != nil {
		return err
	}
	service, err := projection.New(source, embedder, store)
	if err != nil {
		return err
	}
	result, err := service.Rebuild(ctx)
	if err != nil {
		return fmt.Errorf("rebuild Event Semantic projection: %w", err)
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func parseOptions(args []string) (options, error) {
	flags := flag.NewFlagSet("event-semantic-projector", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var result options
	flags.BoolVar(&result.Apply, "apply", false, "replace both rebuildable semantic collections")
	flags.StringVar(&result.AllowEnv, "allow-env", "", "required environment write authorization")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("unexpected positional arguments")
	}
	if !result.Apply || strings.TrimSpace(result.AllowEnv) == "" {
		return options{}, errors.New("-apply and -allow-env are required")
	}
	return result, nil
}
