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
	applyScope := flag.String("apply-scope", "", "explicit apply scope; supported: industry-chain-master, industry-chain-membership")
	applySectorConvergence := flag.Bool("apply-sector-convergence", false, "apply the reviewed initial sector convergence")
	applySectorConvergenceCorrection := flag.Bool("apply-sector-convergence-correction", false, "apply a reviewed forward sector convergence correction")
	flag.Parse()
	scope, err := validateCommandOptions(commandOptions{
		applyScope:                       *applyScope,
		applySectorConvergence:           *applySectorConvergence,
		applySectorConvergenceCorrection: *applySectorConvergenceCorrection,
	})
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
	if *applySectorConvergence || *applySectorConvergenceCorrection {
		convergence, err := entityseed.LoadSectorConvergenceFile(*seedDir + "/sector_convergences.json")
		if err != nil {
			log.Fatalf("load sector convergence manifest: %v", err)
		}
		mode := entityseed.SectorConvergenceModeInitial
		if *applySectorConvergenceCorrection {
			mode = entityseed.SectorConvergenceModeCorrection
		}
		report, err := service.ApplySectorConvergence(ctx, manifest, convergence, mode)
		if err != nil {
			log.Fatalf("apply sector convergence: %v", err)
		}
		content, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Fatalf("encode sector convergence report: %v", err)
		}
		fmt.Fprintln(os.Stdout, string(content))
		return
	}
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
	applyScope                       string
	applySectorConvergence           bool
	applySectorConvergenceCorrection bool
}

func validateCommandOptions(options commandOptions) (entityseed.ApplyScope, error) {
	scope, err := entityseed.ParseApplyScope(options.applyScope)
	if err != nil {
		return "", err
	}
	if options.applySectorConvergence && options.applySectorConvergenceCorrection {
		return "", fmt.Errorf("apply-sector-convergence and apply-sector-convergence-correction cannot be used together")
	}
	if scope != entityseed.ApplyScopeAll && (options.applySectorConvergence || options.applySectorConvergenceCorrection) {
		return "", fmt.Errorf("apply-scope %q cannot be combined with sector convergence", scope)
	}
	return scope, nil
}
