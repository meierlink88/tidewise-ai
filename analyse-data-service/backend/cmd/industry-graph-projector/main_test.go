package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	graphbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industrygraphprojection"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	neo4jdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/neo4j"
)

const testPackageSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestParseCLIOptionsRequiresPinnedPackageAndExactlyOneMode(t *testing.T) {
	options, err := parseCLIOptions([]string{
		"-package", "/tmp/package",
		"-expected-sha256", testPackageSHA,
		"-dry-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !options.DryRun || options.Apply || options.ExpectedSHA != testPackageSHA {
		t.Fatalf("options = %#v", options)
	}
	uat, err := parseCLIOptions([]string{
		"-package", "/tmp/package",
		"-expected-sha256", testPackageSHA,
		"-apply",
		"-allow-env", "uat",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !uat.Apply || uat.AllowEnv != "uat" {
		t.Fatalf("UAT options = %#v", uat)
	}

	invalid := [][]string{
		{"-expected-sha256", testPackageSHA},
		{"-expected-sha256", testPackageSHA, "-dry-run", "-apply", "-allow-env", "local"},
		{"-expected-sha256", "ABC", "-dry-run"},
		{"-expected-sha256", testPackageSHA, "-apply"},
		{"-expected-sha256", testPackageSHA, "-apply", "-allow-env", "prod"},
		{"-expected-sha256", testPackageSHA, "-dry-run", "-allow-env", "local"},
	}
	for index, args := range invalid {
		if _, err := parseCLIOptions(args); err == nil {
			t.Fatalf("invalid options %d were accepted: %v", index, args)
		}
	}
}

func TestProjectionErrorForCLIHidesPostgreSQLDriverDetails(t *testing.T) {
	raw := errors.New(
		"read Industry graph source snapshot: query failed at postgres://user:secret@localhost/tidewise_local",
	)
	message := projectionErrorForCLI(raw)
	if strings.Contains(message, "secret") || strings.Contains(message, "postgres://") {
		t.Fatalf("projectionErrorForCLI() leaked driver details: %s", message)
	}
	if !strings.Contains(message, "PostgreSQL") {
		t.Fatalf("projectionErrorForCLI() = %q, want stable PostgreSQL operation", message)
	}

	safe := errors.New("PostgreSQL Industry graph snapshot differs from the approved projection baseline")
	if got := projectionErrorForCLI(safe); got != safe.Error() {
		t.Fatalf("projectionErrorForCLI() = %q, want safe business error %q", got, safe)
	}
}

func TestExecuteProjectionClosesRuntimeWithBoundedCleanupContextOnFailure(t *testing.T) {
	runtime := &fakeCommandRuntime{projectError: errors.New("projection failed")}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := executeProjection(
		ctx,
		graphbiz.ProjectRequest{},
		io.Discard,
		func(context.Context) (commandRuntime, error) { return runtime, nil },
	)
	if err == nil || !strings.Contains(err.Error(), "projection failed") {
		t.Fatalf("executeProjection() error = %v", err)
	}
	if !runtime.closed {
		t.Fatal("executeProjection() did not close its runtime")
	}
	if !runtime.closeContextUsable || !runtime.closeContextBounded {
		t.Fatalf(
			"cleanup context usable/bounded = %t/%t, want true/true",
			runtime.closeContextUsable,
			runtime.closeContextBounded,
		)
	}
}

type fakeCommandRuntime struct {
	projectError        error
	closed              bool
	closeContextUsable  bool
	closeContextBounded bool
}

func (f *fakeCommandRuntime) Project(
	context.Context,
	graphbiz.ProjectRequest,
) (graphbiz.Result, error) {
	return graphbiz.Result{}, f.projectError
}

func (f *fakeCommandRuntime) Close(ctx context.Context) error {
	f.closed = true
	f.closeContextUsable = ctx.Err() == nil
	deadline, ok := ctx.Deadline()
	f.closeContextBounded = ok && time.Until(deadline) > 0 && time.Until(deadline) <= 10*time.Second
	return nil
}

func TestValidateTargetAcceptsOnlyApprovedLocalAndUATPostgreSQL(t *testing.T) {
	local := conf.Config{
		App: conf.AppConfig{Env: conf.EnvLocal},
		Database: conf.DatabaseConfig{
			Host: "localhost", Name: "tidewise_local", SSLMode: "disable",
		},
	}
	if err := validateTarget(local, cliOptions{Apply: true, AllowEnv: "local"}); err != nil {
		t.Fatalf("valid target: %v", err)
	}
	uat := conf.Config{
		App: conf.AppConfig{Env: conf.EnvUAT},
		Database: conf.DatabaseConfig{
			Host: uatPostgreSQLHost, Port: 5432, Name: "tidewise_uat",
			User: "tidewise_uat", SSLMode: "require",
		},
	}
	if err := validateTarget(uat, cliOptions{Apply: true, AllowEnv: "uat"}); err != nil {
		t.Fatalf("valid UAT target: %v", err)
	}

	invalid := []struct {
		config  conf.Config
		options cliOptions
	}{
		{
			config:  uat,
			options: cliOptions{Apply: true, AllowEnv: "local"},
		},
		{
			config: conf.Config{
				App: conf.AppConfig{Env: conf.EnvUAT},
				Database: conf.DatabaseConfig{
					Host: "localhost", Port: 5432, Name: "tidewise_uat",
					User: "tidewise_uat", SSLMode: "require",
				},
			},
			options: cliOptions{DryRun: true},
		},
		{
			config: conf.Config{
				App: conf.AppConfig{Env: conf.EnvProd},
				Database: conf.DatabaseConfig{
					Host: uatPostgreSQLHost, Port: 5432, Name: "tidewise_uat",
					User: "tidewise_uat", SSLMode: "require",
				},
			},
			options: cliOptions{DryRun: true},
		},
		{
			config: conf.Config{
				App: conf.AppConfig{Env: conf.EnvLocal},
				Database: conf.DatabaseConfig{
					Host: "postgres", Name: "tidewise_local", SSLMode: "disable",
				},
			},
			options: cliOptions{DryRun: true},
		},
		{
			config: conf.Config{
				App: conf.AppConfig{Env: conf.EnvLocal},
				Database: conf.DatabaseConfig{
					Host: "127.0.0.1", Name: "tidewise_uat", SSLMode: "disable",
				},
			},
			options: cliOptions{DryRun: true},
		},
	}
	for index, target := range invalid {
		if err := validateTarget(target.config, target.options); err == nil {
			t.Fatalf("invalid PostgreSQL target %d was accepted", index)
		}
	}
}

func TestValidateNeo4jTargetAcceptsOnlyEnvironmentSpecificTarget(t *testing.T) {
	valid := neo4jdata.Config{
		URI: "bolt://localhost:7687", Username: "neo4j",
		Password: "secret", Database: "neo4j",
	}
	if err := validateNeo4jTarget(conf.EnvLocal, valid); err != nil {
		t.Fatalf("valid Neo4j target: %v", err)
	}
	uat := neo4jdata.Config{
		URI: "bolt://host.docker.internal:7687", Username: "neo4j",
		Password: "secret", Database: "neo4j",
	}
	if err := validateNeo4jTarget(conf.EnvUAT, uat); err != nil {
		t.Fatalf("valid UAT Neo4j target: %v", err)
	}
	for index, target := range []struct {
		environment conf.Environment
		config      neo4jdata.Config
	}{
		{conf.EnvLocal, neo4jdata.Config{URI: "bolt://graph.internal:7687", Username: "neo4j", Password: "secret", Database: "neo4j"}},
		{conf.EnvUAT, neo4jdata.Config{URI: "bolt://localhost:7687", Username: "neo4j", Password: "secret", Database: "neo4j"}},
		{conf.EnvUAT, neo4jdata.Config{URI: "bolt://123.60.99.198:7687", Username: "neo4j", Password: "secret", Database: "neo4j"}},
		{conf.EnvUAT, neo4jdata.Config{URI: "bolt://203.0.113.10:7687", Username: "neo4j", Password: "secret", Database: "neo4j"}},
		{conf.EnvUAT, neo4jdata.Config{URI: "neo4j://" + uatNeo4jHost + ":7687", Username: "neo4j", Password: "secret", Database: "neo4j"}},
		{conf.EnvUAT, neo4jdata.Config{URI: "bolt://neo4j:secret@" + uatNeo4jHost + ":7687", Username: "neo4j", Password: "secret", Database: "neo4j"}},
		{conf.EnvUAT, neo4jdata.Config{URI: "bolt://" + uatNeo4jHost + ":7687", Username: "neo4j", Password: "secret", Database: "system"}},
		{conf.EnvUAT, neo4jdata.Config{URI: "bolt://" + uatNeo4jHost + ":7687", Username: "", Password: "secret", Database: "neo4j"}},
		{conf.EnvProd, uat},
	} {
		if err := validateNeo4jTarget(target.environment, target.config); err == nil {
			t.Fatalf("invalid Neo4j target %d was accepted: %#v", index, target.config)
		} else if strings.Contains(err.Error(), "secret") {
			t.Fatalf("Neo4j target error exposed credentials: %v", err)
		}
	}
}
