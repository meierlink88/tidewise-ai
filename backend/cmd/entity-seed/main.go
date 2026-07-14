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
	mappingApprovedFirstBatch := flag.Bool("external-identifier-mapping-approved-first-batch", false, "allow only the frozen first-batch external identifier mapping write")
	relationManifest := flag.String("chain-node-relation-manifest", "", "reviewed chain node relation manifest")
	relationDryRun := flag.Bool("chain-node-relation-dry-run", false, "run DB snapshot dry-run for chain node relations")
	relationApprovedWrite := flag.Bool("chain-node-relation-approved-data-write", false, "allow only the frozen first-batch chain node relation data write")
	allianceEconomyManifest := flag.String("alliance-economy-approved-manifest", "", "frozen approved alliance/economy manifest for this change")
	allianceEconomyDependencyAudit := flag.Bool("alliance-economy-dependency-audit", false, "read-only local dependency audit for this change")
	allianceEconomyCleanupApprovedLocal := flag.Bool("alliance-economy-cleanup-approved-local", false, "execute separately approved local cleanup for this change")
	allianceEconomyDependencyChecksum := flag.String("alliance-economy-dependency-checksum", "", "reviewed dependency audit checksum required by local cleanup")
	allianceEconomyRebuildApprovedLocal := flag.Bool("alliance-economy-rebuild-approved-local", false, "execute separately approved local rebuild for this change")
	flag.Parse()
	allianceEconomyOptions := allianceEconomyCommandOptions{manifest: *allianceEconomyManifest, dependencyAudit: *allianceEconomyDependencyAudit, cleanupApprovedLocal: *allianceEconomyCleanupApprovedLocal, dependencyChecksum: *allianceEconomyDependencyChecksum, rebuildApprovedLocal: *allianceEconomyRebuildApprovedLocal, seedDir: *seedDir, manifestFile: *manifestFile, includeInactive: *includeInactive, applyScope: *applyScope, phaseAPreflight: *phaseAPreflight, mappingManifest: *mappingManifest, mappingDryRun: *mappingDryRun, mappingPreflight: *mappingPreflight, mappingApproved: *mappingApprovedFirstBatch, relationManifest: *relationManifest, relationDryRun: *relationDryRun, relationApproved: *relationApprovedWrite}
	if err := validateAllianceEconomyCommandOptions(allianceEconomyOptions); err != nil {
		log.Fatal(err)
	}
	if err := validateRelationCommandOptions(relationCommandOptions{manifest: *relationManifest, dryRun: *relationDryRun, approvedWrite: *relationApprovedWrite, seedDir: *seedDir, manifestFile: *manifestFile, includeInactive: *includeInactive, applyScope: *applyScope, phaseAPreflight: *phaseAPreflight, mappingManifest: *mappingManifest, mappingDryRun: *mappingDryRun, mappingPreflight: *mappingPreflight, mappingApproved: *mappingApprovedFirstBatch}); err != nil {
		log.Fatal(err)
	}
	scope, mappingMode, err := validateCommandOptions(commandOptions{
		seedDir: *seedDir, manifestFile: *manifestFile, includeInactive: *includeInactive, applyScope: *applyScope,
		phaseAPreflight: *phaseAPreflight, mappingManifest: *mappingManifest, mappingDryRun: *mappingDryRun, mappingPreflight: *mappingPreflight, mappingApprovedFirstBatch: *mappingApprovedFirstBatch,
	})
	if err != nil {
		log.Fatal(err)
	}
	var relationInput entityseed.ChainNodeRelationManifest
	if strings.TrimSpace(*relationManifest) != "" {
		relationInput, err = loadRelationDryRunManifest(*relationManifest)
		if err != nil {
			log.Fatalf("load chain node relation manifest: %v", err)
		}
		if err := entityseed.ValidateFrozenFirstBatchChainNodeRelationManifest(*relationManifest, relationInput); err != nil {
			log.Fatalf("validate frozen chain node relation manifest: %v", err)
		}
	}
	var allianceEconomyInput entityseed.AllianceEconomyManifest
	if strings.TrimSpace(*allianceEconomyManifest) != "" {
		allianceEconomyInput, err = entityseed.LoadApprovedAllianceEconomyManifest(*allianceEconomyManifest)
		if err != nil {
			log.Fatalf("load approved alliance economy manifest: %v", err)
		}
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if strings.TrimSpace(*allianceEconomyManifest) != "" {
		if err := validateAllianceEconomyLocalTarget(cfg); err != nil {
			log.Fatal(err)
		}
	}

	timeout := time.Duration(cfg.Database.ConnectTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	db, err := database.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if strings.TrimSpace(*allianceEconomyManifest) != "" {
		repository := entityseed.NewPostgresRepository(db)
		var result any
		switch {
		case *allianceEconomyDependencyAudit:
			result, err = repository.AuditAllianceEconomyRebuildDependencies(ctx)
		case *allianceEconomyCleanupApprovedLocal:
			result, err = repository.CleanupAllianceEconomyLocal(ctx, *allianceEconomyDependencyChecksum)
		case *allianceEconomyRebuildApprovedLocal:
			result, err = repository.RebuildApprovedAllianceEconomyLocal(ctx, allianceEconomyInput)
		}
		if err != nil {
			log.Fatalf("process approved alliance economy batch: %v", err)
		}
		_ = json.NewEncoder(os.Stdout).Encode(result)
		return
	}
	if strings.TrimSpace(*relationManifest) != "" {
		repository := entityseed.NewPostgresRepository(db)
		var report entityseed.ChainNodeRelationReport
		if *relationDryRun {
			report, err = repository.DryRunFrozenFirstBatchChainNodeRelations(ctx, relationInput.Relations)
		} else {
			report, err = repository.ApplyFrozenFirstBatchChainNodeRelations(ctx, relationInput.Relations)
		}
		if err != nil {
			log.Fatalf("process frozen chain node relations: %v", err)
		}
		_ = json.NewEncoder(os.Stdout).Encode(report)
		return
	}
	if mappingMode {
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
		if *mappingDryRun {
			report, err := entityseed.NewPostgresRepository(db).DryRunExternalIdentifierBatch(ctx, manifest.Mappings)
			if err != nil {
				log.Fatalf("dry-run external identifier mappings: %v", err)
			}
			_ = json.NewEncoder(os.Stdout).Encode(report)
			return
		}
		if !*mappingApprovedFirstBatch {
			log.Fatal("mapping write requires -external-identifier-mapping-approved-first-batch")
		}
		if err := entityseed.ValidateFrozenFirstBatchExternalIdentifierManifest(*mappingManifest, manifest.Mappings); err != nil {
			log.Fatalf("validate frozen first-batch mapping manifest: %v", err)
		}
		report, err := entityseed.NewPostgresRepository(db).ApplyFrozenFirstBatchExternalIdentifiers(ctx, manifest.Mappings)
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

func loadRelationDryRunManifest(path string) (entityseed.ChainNodeRelationManifest, error) {
	manifest, err := entityseed.LoadChainNodeRelationManifest(path)
	if err != nil {
		return manifest, err
	}
	if err := entityseed.ValidateChainNodeRelationDryRunManifest(manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

type commandOptions struct {
	seedDir, manifestFile, applyScope, mappingManifest                                           string
	includeInactive, phaseAPreflight, mappingDryRun, mappingPreflight, mappingApprovedFirstBatch bool
}

type relationCommandOptions struct {
	manifest, seedDir, manifestFile, applyScope, mappingManifest                                              string
	dryRun, approvedWrite, includeInactive, phaseAPreflight, mappingDryRun, mappingPreflight, mappingApproved bool
}

type allianceEconomyCommandOptions struct {
	manifest, dependencyChecksum, seedDir, manifestFile, applyScope, mappingManifest, relationManifest string
	dependencyAudit, cleanupApprovedLocal, rebuildApprovedLocal, includeInactive, phaseAPreflight      bool
	mappingDryRun, mappingPreflight, mappingApproved, relationDryRun, relationApproved                 bool
}

func validateAllianceEconomyCommandOptions(o allianceEconomyCommandOptions) error {
	hasManifest := strings.TrimSpace(o.manifest) != ""
	modes := 0
	for _, enabled := range []bool{o.dependencyAudit, o.cleanupApprovedLocal, o.rebuildApprovedLocal} {
		if enabled {
			modes++
		}
	}
	if !hasManifest && modes == 0 {
		return nil
	}
	if !hasManifest || modes != 1 {
		return fmt.Errorf("alliance/economy change mode requires its frozen manifest and exactly one operation")
	}
	if o.cleanupApprovedLocal && strings.TrimSpace(o.dependencyChecksum) == "" {
		return fmt.Errorf("alliance/economy local cleanup requires the reviewed dependency checksum")
	}
	if !o.cleanupApprovedLocal && strings.TrimSpace(o.dependencyChecksum) != "" {
		return fmt.Errorf("alliance/economy dependency checksum is cleanup-only")
	}
	if o.seedDir != entityseed.DefaultSeedDir || strings.TrimSpace(o.manifestFile) != "" || o.includeInactive || strings.TrimSpace(o.applyScope) != "" || o.phaseAPreflight || strings.TrimSpace(o.mappingManifest) != "" || o.mappingDryRun || o.mappingPreflight || o.mappingApproved || strings.TrimSpace(o.relationManifest) != "" || o.relationDryRun || o.relationApproved {
		return fmt.Errorf("alliance/economy change mode cannot combine other seed modes")
	}
	return nil
}

func validateAllianceEconomyLocalTarget(cfg config.Config) error {
	if cfg.App.Env != config.EnvLocal || cfg.Database.Name != "tidewise_local" {
		return fmt.Errorf("alliance/economy cleanup and rebuild are restricted to APP_ENV=local and database tidewise_local")
	}
	return nil
}

func validateRelationCommandOptions(o relationCommandOptions) error {
	hasManifest := strings.TrimSpace(o.manifest) != ""
	if (o.dryRun || o.approvedWrite) && !hasManifest {
		return fmt.Errorf("relation dry-run/write requires -chain-node-relation-manifest")
	}
	if !hasManifest {
		return nil
	}
	if o.dryRun == o.approvedWrite {
		return fmt.Errorf("relation-only mode requires exactly one of dry-run or approved data write")
	}
	if o.seedDir != entityseed.DefaultSeedDir || strings.TrimSpace(o.manifestFile) != "" || o.includeInactive || strings.TrimSpace(o.applyScope) != "" || o.phaseAPreflight || strings.TrimSpace(o.mappingManifest) != "" || o.mappingDryRun || o.mappingPreflight || o.mappingApproved {
		return fmt.Errorf("relation-only mode cannot combine entity or external identifier seed options")
	}
	return nil
}

func validateCommandOptions(options commandOptions) (entityseed.ApplyScope, bool, error) {
	scope, err := entityseed.ParseApplyScope(options.applyScope)
	if err != nil {
		return "", false, err
	}
	mappingManifest := strings.TrimSpace(options.mappingManifest)
	if options.mappingDryRun || options.mappingPreflight {
		if mappingManifest == "" {
			return "", false, fmt.Errorf("mapping dry-run/preflight requires -external-identifier-mapping-manifest")
		}
	}
	if options.mappingApprovedFirstBatch && mappingManifest == "" {
		return "", false, fmt.Errorf("first-batch mapping write approval requires -external-identifier-mapping-manifest")
	}
	if mappingManifest == "" {
		return scope, false, nil
	}
	if options.mappingDryRun && options.mappingPreflight {
		return "", false, fmt.Errorf("mapping dry-run and preflight are mutually exclusive")
	}
	if options.seedDir != entityseed.DefaultSeedDir || strings.TrimSpace(options.manifestFile) != "" || options.includeInactive || strings.TrimSpace(options.applyScope) != "" || options.phaseAPreflight {
		return "", false, fmt.Errorf("mapping-only mode cannot combine entity seed options")
	}
	return scope, true, nil
}
