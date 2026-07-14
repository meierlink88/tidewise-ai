package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	entityseed "github.com/meierlink88/tidewise-ai/backend/internal/apps/entityfoundation/seed"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/database"
)

func main() {
	seedDir := flag.String("seed-dir", entityseed.DefaultSeedDir, "entity foundation seed directory")
	manifestFile := flag.String("manifest-file", "", "explicit reviewed entity foundation manifest file")
	includeInactive := flag.Bool("include-inactive", false, "include inactive entities in seed writes")
	applyScope := flag.String("apply-scope", "", "reserved; legacy industry-chain apply scopes are disabled")
	phaseAPreflight := flag.Bool("phase-a-preflight", false, "run the read-only industry model Phase A preflight and exit")
	mappingManifest := flag.String("external-identifier-mapping-manifest", "", "reviewed external identifier mapping manifest")
	mappingDryRun := flag.Bool("external-identifier-mapping-dry-run", false, "validate external identifier mapping manifest without writes")
	mappingPreflight := flag.Bool("external-identifier-mapping-preflight", false, "run read-only preflight for the reviewed external identifier mapping manifest")
	flag.Parse()
	scope, err := validateCommandOptions(commandOptions{applyScope: *applyScope})
	if err != nil {
		log.Fatal(err)
	}
	if strings.TrimSpace(*mappingManifest) != "" && *mappingDryRun {
		if *phaseAPreflight || strings.TrimSpace(*manifestFile) != "" || *includeInactive || strings.TrimSpace(*applyScope) != "" {
			log.Fatal("mapping-only mode cannot combine entity seed options")
		}
		report, err := entityseed.DryRunExternalIdentifierMappings(*mappingManifest)
		if err != nil {
			log.Fatalf("dry-run external identifier mappings: %v", err)
		}
		_ = json.NewEncoder(os.Stdout).Encode(report)
		return
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
	if strings.TrimSpace(*mappingManifest) != "" {
		if *phaseAPreflight || strings.TrimSpace(*manifestFile) != "" || *includeInactive || strings.TrimSpace(*applyScope) != "" {
			log.Fatal("mapping-only mode cannot combine entity seed options")
		}
		manifest, err := entityseed.LoadExternalIdentifierMappingFile(*mappingManifest)
		if err != nil {
			log.Fatalf("load external identifier mappings: %v", err)
		}
		if *mappingPreflight {
			report, err := entityseed.NewPostgresRepository(db).PreflightExternalIdentifierMappings(ctx, manifest.Mappings)
			if err != nil {
				log.Fatalf("preflight external identifier mappings: %v", err)
			}
			_ = json.NewEncoder(os.Stdout).Encode(report)
			return
		}
		report, err := entityseed.NewPostgresRepository(db).ApplyExternalIdentifierBatch(ctx, manifest.Mappings)
		if err != nil {
			log.Fatalf("apply external identifier mappings: %v", err)
		}
		_ = json.NewEncoder(os.Stdout).Encode(report)
		return
	}
	if *phaseAPreflight {
		if strings.TrimSpace(*manifestFile) == "" {
			log.Fatal("phase A preflight requires an explicit reviewed manifest file")
		}
		manifest, err := loadManifest(*seedDir, *manifestFile)
		if err != nil {
			log.Fatalf("load reviewed entity seed manifest: %v", err)
		}
		proof, err := manifestPreflightProof(*manifestFile, manifest)
		if err != nil {
			log.Fatalf("build reviewed manifest preflight proof: %v", err)
		}
		report, err := entityseed.NewPostgresRepository(db).RunPhaseAPreflight(ctx)
		if err != nil {
			log.Fatalf("run phase A preflight: %v", err)
		}
		content, err := json.MarshalIndent(struct {
			Preflight entityseed.PhaseAPreflightReport `json:"preflight"`
			Manifest  manifestPreflight                `json:"manifest"`
		}{Preflight: report, Manifest: proof}, "", "  ")
		if err != nil {
			log.Fatalf("encode phase A preflight report: %v", err)
		}
		fmt.Fprintln(os.Stdout, string(content))
		return
	}

	manifest, err := loadManifest(*seedDir, *manifestFile)
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

type manifestPreflight struct {
	SHA256         string `json:"sha256"`
	EntityCount    int    `json:"entity_count"`
	ChainNodeCount int    `json:"chain_node_count"`
	ProfileCount   int    `json:"profile_count"`
}

func manifestPreflightProof(path string, manifest entityseed.Manifest) (manifestPreflight, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return manifestPreflight{}, err
	}
	proof := manifestPreflight{SHA256: fmt.Sprintf("%x", sha256.Sum256(content)), EntityCount: len(manifest.Entities)}
	for _, entity := range manifest.Entities {
		if entity.EntityType == "chain_node" {
			proof.ChainNodeCount++
		}
		if len(entity.Profile) > 0 {
			proof.ProfileCount++
		}
	}
	for _, profile := range manifest.Profiles {
		if profile.EntityType == "chain_node" {
			proof.ProfileCount++
		}
	}
	return proof, nil
}

func loadManifest(seedDir, manifestFile string) (entityseed.Manifest, error) {
	if strings.TrimSpace(manifestFile) != "" {
		return entityseed.LoadFile(manifestFile)
	}
	return entityseed.LoadFiles(entityseed.DefaultSeedPaths(seedDir)...)
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
