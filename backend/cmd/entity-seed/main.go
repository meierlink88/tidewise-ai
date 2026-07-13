package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	entityseed "github.com/meierlink88/tidewise-ai/backend/internal/apps/entityfoundation/seed"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/database"
)

func main() {
	seedDir := flag.String("seed-dir", entityseed.DefaultSeedDir, "entity foundation seed directory")
	includeInactive := flag.Bool("include-inactive", false, "include inactive entities in seed writes")
	applyScope := flag.String("apply-scope", "", "reserved; legacy industry-chain apply scopes are disabled")
	phaseAPreflight := flag.Bool("phase-a-preflight", false, "run the read-only industry model Phase A preflight and exit")
	flag.Parse()
	scope, err := validateCommandOptions(commandOptions{applyScope: *applyScope})
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	timeout := time.Duration(cfg.Database.ConnectTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	db, err := database.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if *phaseAPreflight {
		report, err := entityseed.NewPostgresRepository(db).RunPhaseAPreflight(ctx)
		if err != nil {
			log.Fatalf("run phase A preflight: %v", err)
		}
		content, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Fatalf("encode phase A preflight report: %v", err)
		}
		fmt.Fprintln(os.Stdout, string(content))
		return
	}

	manifest, err := entityseed.LoadFiles(entityseed.DefaultSeedPaths(*seedDir)...)
	if err != nil {
		log.Fatalf("load entity seed files: %v", err)
	}

	service := entityseed.NewService(entityseed.NewPostgresRepository(db))
	report, err := service.Apply(ctx, manifest, entityseed.ApplyOptions{IncludeInactive: *includeInactive, Scope: scope})
	if err != nil {
		log.Fatalf("apply entity seed: %v", err)
	}

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("encode entity seed report: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(content))
}

type commandOptions struct {
	applyScope string
}

func validateCommandOptions(options commandOptions) (entityseed.ApplyScope, error) {
	scope, err := entityseed.ParseApplyScope(options.applyScope)
	if err != nil {
		return "", err
	}
	return scope, nil
}
