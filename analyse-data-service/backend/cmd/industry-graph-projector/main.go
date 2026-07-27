package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	graphbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industrygraphprojection"
	relationshipbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	graphdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/industrygraphprojection"
	neo4jdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/neo4j"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

const (
	defaultPackagePath = "analyse-data-service/backend/data/industry_relationships/2026-07-27-v1"
	defaultNeo4jURI    = "bolt://localhost:7687"
	defaultNeo4jDB     = "neo4j"
	uatPostgreSQLHost  = "775b3ecf9c934ae185c0b8eda157c50din03.internal.cn-east-3.postgresql.rds.myhuaweicloud.com"
	uatNeo4jHost       = "123.60.99.198"
)

type cliOptions struct {
	PackagePath string
	ExpectedSHA string
	AllowEnv    string
	DryRun      bool
	Apply       bool
}

type commandRuntime interface {
	Project(context.Context, graphbiz.ProjectRequest) (graphbiz.Result, error)
	Close(context.Context) error
}

type liveCommandRuntime struct {
	database *sql.DB
	graph    *neo4jdata.Store
	service  *graphbiz.Service
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	options, err := parseCLIOptions(args)
	if err != nil {
		return fmt.Errorf("parse Industry graph projector options: %w", err)
	}
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return fmt.Errorf("load Data configuration: %w", err)
	}
	if err := validateTarget(cfg, options); err != nil {
		return err
	}
	neo4jConfig := loadNeo4jConfig()
	if err := validateNeo4jTarget(cfg.App.Env, neo4jConfig); err != nil {
		return err
	}

	pkg, err := relationshipbiz.LoadDirectory(options.PackagePath, options.ExpectedSHA)
	if err != nil {
		return fmt.Errorf("load approved Industry relationship package: %w", err)
	}
	baseline, err := graphdata.LoadFrozenV1CSVBaseline(pkg)
	if err != nil {
		return fmt.Errorf("load approved Industry graph baseline: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	return executeProjection(
		ctx,
		graphbiz.ProjectRequest{Baseline: baseline, Apply: options.Apply},
		output,
		func(openContext context.Context) (commandRuntime, error) {
			return openLiveCommandRuntime(openContext, cfg, neo4jConfig)
		},
	)
}

func openLiveCommandRuntime(
	ctx context.Context,
	cfg conf.Config,
	neo4jConfig neo4jdata.Config,
) (commandRuntime, error) {
	db, err := postgres.Open(ctx, cfg)
	if err != nil {
		return nil, errors.New("open PostgreSQL for Industry graph projection failed")
	}
	graphStore, err := neo4jdata.Open(ctx, neo4jConfig)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &liveCommandRuntime{
		database: db,
		graph:    graphStore,
		service:  graphbiz.NewService(postgres.NewIndustryGraphSnapshotReader(db), graphStore),
	}, nil
}

func (r *liveCommandRuntime) Project(
	ctx context.Context,
	request graphbiz.ProjectRequest,
) (graphbiz.Result, error) {
	return r.service.Project(ctx, request)
}

func (r *liveCommandRuntime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var graphError, databaseError error
	if r.graph != nil {
		graphError = r.graph.Close(ctx)
	}
	if r.database != nil {
		if err := r.database.Close(); err != nil {
			databaseError = errors.New("close PostgreSQL connection failed")
		}
	}
	return errors.Join(graphError, databaseError)
}

func executeProjection(
	ctx context.Context,
	request graphbiz.ProjectRequest,
	output io.Writer,
	open func(context.Context) (commandRuntime, error),
) (returnError error) {
	runtime, err := open(ctx)
	if err != nil {
		return err
	}
	defer func() {
		closeContext, closeCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer closeCancel()
		returnError = errors.Join(returnError, runtime.Close(closeContext))
	}()

	result, err := runtime.Project(ctx, request)
	if err != nil {
		return fmt.Errorf("project Industry graph: %s", projectionErrorForCLI(err))
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return errors.New("encode Industry graph projection result failed")
	}
	return nil
}

func projectionErrorForCLI(err error) string {
	if err == nil {
		return ""
	}
	if strings.HasPrefix(err.Error(), "read Industry graph source snapshot:") {
		return "read PostgreSQL Industry graph snapshot failed"
	}
	return err.Error()
}

func parseCLIOptions(args []string) (cliOptions, error) {
	flags := flag.NewFlagSet("industry-graph-projector", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options cliOptions
	flags.StringVar(&options.PackagePath, "package", defaultPackagePath, "approved relationship package directory")
	flags.StringVar(&options.ExpectedSHA, "expected-sha256", "", "required approved package SHA-256")
	flags.StringVar(&options.AllowEnv, "allow-env", "", "required write target: local or uat")
	flags.BoolVar(&options.DryRun, "dry-run", false, "validate PostgreSQL and inspect Neo4j without writes")
	flags.BoolVar(&options.Apply, "apply", false, "atomically replace the fixed Neo4j namespace")
	if err := flags.Parse(args); err != nil {
		return cliOptions{}, err
	}
	if flags.NArg() != 0 {
		return cliOptions{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	if options.DryRun == options.Apply {
		return cliOptions{}, errors.New("exactly one of -dry-run or -apply is required")
	}
	if !graphbiz.ValidPackageSHA256(options.ExpectedSHA) {
		return cliOptions{}, errors.New("-expected-sha256 must be 64 lowercase hexadecimal characters")
	}
	if strings.TrimSpace(options.PackagePath) == "" {
		return cliOptions{}, errors.New("-package is required")
	}
	if options.Apply && options.AllowEnv != string(conf.EnvLocal) && options.AllowEnv != string(conf.EnvUAT) {
		return cliOptions{}, errors.New("-apply requires -allow-env local or -allow-env uat")
	}
	if options.DryRun && options.AllowEnv != "" {
		return cliOptions{}, errors.New("-allow-env is only valid with -apply")
	}
	return options, nil
}

func validateTarget(cfg conf.Config, options cliOptions) error {
	if options.Apply && options.AllowEnv != string(cfg.App.Env) {
		return errors.New("Industry graph projector write authorization does not match APP_ENV")
	}
	switch cfg.App.Env {
	case conf.EnvLocal:
		if !isLoopbackHost(cfg.Database.Host) ||
			cfg.Database.Name != "tidewise_local" ||
			cfg.Database.SSLMode != "disable" {
			return errors.New("Industry graph projector requires loopback tidewise_local PostgreSQL with ssl_mode=disable")
		}
	case conf.EnvUAT:
		if cfg.Database.Host != uatPostgreSQLHost ||
			cfg.Database.Port != 5432 ||
			cfg.Database.Name != "tidewise_uat" ||
			cfg.Database.User != "tidewise_uat" ||
			cfg.Database.SSLMode != "require" {
			return errors.New("Industry graph projector requires the repository-controlled tidewise_uat PostgreSQL target")
		}
	default:
		return errors.New("Industry graph projector only accepts APP_ENV=local or APP_ENV=uat")
	}
	return nil
}

func loadNeo4jConfig() neo4jdata.Config {
	uri := strings.TrimSpace(os.Getenv("NEO4J_URI"))
	if uri == "" {
		uri = defaultNeo4jURI
	}
	database := strings.TrimSpace(os.Getenv("NEO4J_DATABASE"))
	if database == "" {
		database = defaultNeo4jDB
	}
	return neo4jdata.Config{
		URI:      uri,
		Username: os.Getenv("NEO4J_USERNAME"),
		Password: os.Getenv("NEO4J_PASSWORD"),
		Database: database,
	}
}

func validateNeo4jTarget(environment conf.Environment, config neo4jdata.Config) error {
	if strings.TrimSpace(config.Username) == "" ||
		config.Password == "" ||
		config.Database != defaultNeo4jDB {
		return errors.New("Industry graph projector requires Neo4j credentials and database neo4j")
	}
	target, err := url.Parse(config.URI)
	if err != nil ||
		target.Scheme != "bolt" ||
		target.Hostname() == "" ||
		target.User != nil ||
		target.RawQuery != "" ||
		target.Fragment != "" ||
		(target.Path != "" && target.Path != "/") {
		return errors.New("Industry graph projector requires a credential-free bolt URI")
	}
	if target.Port() != "" {
		port, err := net.LookupPort("tcp", target.Port())
		if err != nil || port <= 0 || port > 65535 {
			return errors.New("Industry graph projector Neo4j URI has an invalid port")
		}
	}
	switch environment {
	case conf.EnvLocal:
		if !isLoopbackHost(target.Hostname()) {
			return errors.New("Industry graph projector local target requires a loopback bolt URI")
		}
	case conf.EnvUAT:
		if target.Hostname() != uatNeo4jHost || target.Port() != "7687" {
			return errors.New("Industry graph projector UAT target requires the approved bolt endpoint")
		}
	default:
		return errors.New("Industry graph projector Neo4j target only accepts local or uat")
	}
	return nil
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}
