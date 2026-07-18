package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/adapters/database"
	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories/researchseed/postgresstore"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchseed"
)

func main() {
	manifestPath := flag.String("manifest-file", researchseed.DefaultManifestPath, "local research theme manifest")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := validateLocalTarget(cfg); err != nil {
		log.Fatal(err)
	}
	manifest, err := researchseed.LoadFile(*manifestPath)
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

	report, err := researchseed.NewService(postgresstore.New(db)).Apply(ctx, manifest, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		log.Fatalf("encode report: %v", err)
	}
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
