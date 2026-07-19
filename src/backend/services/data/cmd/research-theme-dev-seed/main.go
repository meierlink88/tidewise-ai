package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/adapters/database"
	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
	appimport "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchthemeimport"
)

const (
	defaultManifestPath       = "data/research_themes/local_homepage.json"
	localSeedPublisherSubject = "local-research-theme-dev-seed"
)

func main() {
	manifestPath := flag.String("manifest-file", defaultManifestPath, "local research theme V1 import batch")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := validateLocalTarget(cfg); err != nil {
		log.Fatal(err)
	}
	batch, err := loadBatch(*manifestPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	repository := repositories.NewPostgresRepository(db)
	report, err := appimport.NewService(repository).Import(ctx, localSeedPublisherSubject, batch)
	if err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		log.Fatalf("encode report: %v", err)
	}
}

func loadBatch(path string) (domainimport.Batch, error) {
	file, err := os.Open(path)
	if err != nil {
		return domainimport.Batch{}, fmt.Errorf("open research theme import batch: %w", err)
	}
	defer file.Close()

	batch, err := domainimport.DecodeStrict(io.LimitReader(file, 4<<20))
	if err != nil {
		return domainimport.Batch{}, err
	}
	if _, err := batch.Validate(); err != nil {
		return domainimport.Batch{}, fmt.Errorf("validate research theme import batch: %w", err)
	}
	return batch, nil
}

func validateLocalTarget(cfg config.Config) error {
	if cfg.App.Env != config.EnvLocal {
		return fmt.Errorf("research theme development seed is local-only, got %q", cfg.App.Env)
	}
	host, databaseName := cfg.Database.Host, cfg.Database.Name
	if cfg.Secrets.DatabaseURL != "" {
		parsed, err := url.Parse(cfg.Secrets.DatabaseURL)
		if err != nil {
			return fmt.Errorf("research theme development seed requires a valid local database URL")
		}
		host = parsed.Hostname()
		databaseName = strings.TrimPrefix(parsed.Path, "/")
	}
	if !isLocalDatabaseHost(host) || databaseName != "tidewise_local" {
		return fmt.Errorf("research theme development seed requires a local PostgreSQL host and database tidewise_local")
	}
	return nil
}

func isLocalDatabaseHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1", "postgres":
		return true
	default:
		return false
	}
}
