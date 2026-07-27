package main

import (
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
)

const testSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestParseCLIOptionsRequiresOneModeAndPinnedPackage(t *testing.T) {
	options, err := parseCLIOptions([]string{
		"-package", "/tmp/package",
		"-expected-sha256", testSHA,
		"-dry-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !options.DryRun || options.Apply || options.ExpectedSHA != testSHA {
		t.Fatalf("options = %#v", options)
	}

	invalid := [][]string{
		{"-expected-sha256", testSHA},
		{"-expected-sha256", testSHA, "-dry-run", "-apply", "-allow-env", "local"},
		{"-expected-sha256", "ABC", "-dry-run"},
		{"-expected-sha256", testSHA, "-apply"},
		{"-expected-sha256", testSHA, "-apply", "-allow-env", "prod"},
		{"-expected-sha256", testSHA, "-dry-run", "-allow-env", "local"},
	}
	for index, args := range invalid {
		if _, err := parseCLIOptions(args); err == nil {
			t.Fatalf("invalid options %d were accepted: %v", index, args)
		}
	}
}

func TestValidateTargetRejectsProdAndMismatchedOrUnsafeApply(t *testing.T) {
	local := conf.Config{
		App: conf.AppConfig{Env: conf.EnvLocal},
		Database: conf.DatabaseConfig{
			Host: "localhost", Name: "tidewise_local", SSLMode: "disable",
		},
	}
	uat := conf.Config{
		App: conf.AppConfig{Env: conf.EnvUAT},
		Database: conf.DatabaseConfig{
			Host: "private.uat.internal", Name: "tidewise_uat", SSLMode: "require",
		},
	}
	if err := validateTarget(local, cliOptions{Apply: true, AllowEnv: "local"}); err != nil {
		t.Fatalf("valid local target: %v", err)
	}
	if err := validateTarget(uat, cliOptions{Apply: true, AllowEnv: "uat"}); err != nil {
		t.Fatalf("valid uat target: %v", err)
	}
	if err := validateTarget(local, cliOptions{DryRun: true}); err != nil {
		t.Fatalf("valid local dry-run target: %v", err)
	}
	if err := validateTarget(uat, cliOptions{DryRun: true}); err != nil {
		t.Fatalf("valid uat dry-run target: %v", err)
	}
	invalid := []struct {
		cfg     conf.Config
		options cliOptions
	}{
		{
			cfg: conf.Config{
				App:      conf.AppConfig{Env: conf.EnvProd},
				Database: conf.DatabaseConfig{Host: "prod", Name: "tidewise_prod"},
			},
			options: cliOptions{DryRun: true},
		},
		{cfg: local, options: cliOptions{Apply: true, AllowEnv: "uat"}},
		{
			cfg: conf.Config{
				App:      conf.AppConfig{Env: conf.EnvLocal},
				Database: conf.DatabaseConfig{Host: "shared", Name: "tidewise_local"},
			},
			options: cliOptions{Apply: true, AllowEnv: "local"},
		},
		{
			cfg: conf.Config{
				App:      conf.AppConfig{Env: conf.EnvLocal},
				Database: conf.DatabaseConfig{Host: "shared", Name: "tidewise_local"},
			},
			options: cliOptions{DryRun: true},
		},
		{
			cfg: conf.Config{
				App: conf.AppConfig{Env: conf.EnvUAT},
				Database: conf.DatabaseConfig{
					Host: "localhost", Name: "tidewise_uat", SSLMode: "require",
				},
			},
			options: cliOptions{Apply: true, AllowEnv: "uat"},
		},
		{
			cfg: conf.Config{
				App: conf.AppConfig{Env: conf.EnvUAT},
				Database: conf.DatabaseConfig{
					Host: "localhost", Name: "tidewise_uat", SSLMode: "require",
				},
			},
			options: cliOptions{DryRun: true},
		},
	}
	for index, item := range invalid {
		if err := validateTarget(item.cfg, item.options); err == nil {
			t.Fatalf("invalid target %d was accepted", index)
		} else if index == 0 && !strings.Contains(err.Error(), "prod") {
			t.Fatalf("prod error = %v", err)
		}
	}
}
