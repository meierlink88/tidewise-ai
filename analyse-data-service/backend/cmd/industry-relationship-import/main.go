package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	industryrelationshipimportdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/industryrelationshipimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

const (
	defaultPackagePath = "/app/data/industry_relationships/2026-07-27-v1"
	defaultCaller      = "codex-industry-relationship-build-v1"
)

type cliOptions struct {
	PackagePath   string
	ExpectedSHA   string
	CallerSubject string
	AllowEnv      string
	DryRun        bool
	Apply         bool
}

func main() {
	options, err := parseCLIOptions(os.Args[1:])
	if err != nil {
		log.Fatalf("parse Industry relationship import options: %v", err)
	}
	cfg, err := conf.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := validateTarget(cfg, options); err != nil {
		log.Fatal(err)
	}
	pkg, err := biz.LoadDirectory(options.PackagePath, options.ExpectedSHA)
	if err != nil {
		log.Fatalf("load Industry relationship package: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	db, err := postgres.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	repository := industryrelationshipimportdata.NewRepository(db)
	service := biz.NewService(repository)
	var result biz.Result
	if options.DryRun {
		result, err = service.Preflight(ctx, options.CallerSubject, pkg)
	} else {
		result, err = service.Import(ctx, options.CallerSubject, pkg)
	}
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		log.Fatalf("encode Industry relationship import result: %v", err)
	}
}

func parseCLIOptions(args []string) (cliOptions, error) {
	flags := flag.NewFlagSet("industry-relationship-import", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options cliOptions
	flags.StringVar(&options.PackagePath, "package", defaultPackagePath, "frozen relationship package directory")
	flags.StringVar(&options.ExpectedSHA, "expected-sha256", "", "required approved package SHA-256")
	flags.StringVar(&options.CallerSubject, "caller-subject", defaultCaller, "audited import caller")
	flags.StringVar(&options.AllowEnv, "allow-env", "", "required write target: local or uat")
	flags.BoolVar(&options.DryRun, "dry-run", false, "validate package and database preflight without writes")
	flags.BoolVar(&options.Apply, "apply", false, "apply package in one transaction")
	if err := flags.Parse(args); err != nil {
		return cliOptions{}, err
	}
	if flags.NArg() != 0 {
		return cliOptions{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	if options.DryRun == options.Apply {
		return cliOptions{}, errorsText("exactly one of -dry-run or -apply is required")
	}
	if !isSHA256(options.ExpectedSHA) {
		return cliOptions{}, errorsText("-expected-sha256 must be 64 lowercase hexadecimal characters")
	}
	if strings.TrimSpace(options.PackagePath) == "" {
		return cliOptions{}, errorsText("-package is required")
	}
	if strings.TrimSpace(options.CallerSubject) == "" || len(strings.TrimSpace(options.CallerSubject)) > 200 {
		return cliOptions{}, errorsText("-caller-subject must contain 1..200 characters")
	}
	if options.Apply && options.AllowEnv != "local" && options.AllowEnv != "uat" {
		return cliOptions{}, errorsText("-apply requires -allow-env local|uat")
	}
	if options.DryRun && options.AllowEnv != "" {
		return cliOptions{}, errorsText("-allow-env is only valid with -apply")
	}
	return options, nil
}

func validateTarget(cfg conf.Config, options cliOptions) error {
	if cfg.App.Env == conf.EnvProd {
		return fmt.Errorf("Industry relationship importer always rejects prod")
	}
	switch cfg.App.Env {
	case conf.EnvLocal:
		if !isLocalDatabaseHost(cfg.Database.Host) || cfg.Database.Name != "tidewise_local" {
			return fmt.Errorf("local target requires a local PostgreSQL host and tidewise_local")
		}
	case conf.EnvUAT:
		if isLocalDatabaseHost(cfg.Database.Host) || isLoopbackDatabaseHost(cfg.Database.Host) ||
			cfg.Database.Name != "tidewise_uat" ||
			cfg.Database.SSLMode != "require" {
			return fmt.Errorf("uat target requires the non-local tidewise_uat database with ssl_mode=require")
		}
	default:
		return fmt.Errorf("unsupported Industry relationship import environment %q", cfg.App.Env)
	}
	if options.DryRun {
		return nil
	}
	if string(cfg.App.Env) != options.AllowEnv {
		return fmt.Errorf("configured APP_ENV %q does not match -allow-env %q", cfg.App.Env, options.AllowEnv)
	}
	return nil
}

func isLocalDatabaseHost(host string) bool {
	return strings.EqualFold(strings.TrimSpace(host), "host.docker.internal")
}

func isLoopbackDatabaseHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func errorsText(value string) error {
	return fmt.Errorf("%s", value)
}
