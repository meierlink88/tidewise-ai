package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	agentrunconfig "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
)

type options struct {
	ManifestPath string
	ExportPath   string
	AllowEnv     string
	DryRun       bool
	Apply        bool
}

func main() {
	options, err := parseOptions(os.Args[1:])
	if err != nil {
		fail(err.Error())
	}
	manifest, err := readManifest(options.ManifestPath)
	if err != nil {
		fail(err.Error())
	}
	cfg, err := agentrunconfig.LoadDatabaseOperation()
	if err != nil {
		fail("could not load AgentRun configuration")
	}
	if err := validateTarget(cfg, options); err != nil {
		fail(err.Error())
	}
	databaseURL, err := cfg.PostgresURL()
	if err != nil {
		fail("could not build AgentRun database configuration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		fail("could not open AgentRun database")
	}
	defer database.Close()
	store := postgres.New(database)
	plan, err := store.PlanHistoricalEventDisposition(ctx, manifest)
	if err != nil {
		fail("could not plan historical Event disposition")
	}
	if options.DryRun {
		if err := encodeJSON(os.Stdout, plan); err != nil {
			fail("could not encode historical Event disposition plan")
		}
		return
	}
	if len(plan.BlockingRunningEventIDs) > 0 {
		fail("historical Event disposition is blocked by running Work Items")
	}
	if err := writeExclusiveJSON(options.ExportPath, plan); err != nil {
		fail(err.Error())
	}
	report, err := store.ApplyHistoricalEventDisposition(
		ctx, manifest, time.Now(),
	)
	if err != nil {
		fail("could not apply historical Event disposition")
	}
	if err := encodeJSON(os.Stdout, report); err != nil {
		fail("could not encode historical Event disposition report")
	}
}

func parseOptions(arguments []string) (options, error) {
	flags := flag.NewFlagSet("event-semantic-history", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var result options
	flags.StringVar(&result.ManifestPath, "manifest", "", "reviewed Data audit manifest")
	flags.StringVar(&result.ExportPath, "export", "", "new pre-change Work Item snapshot path")
	flags.StringVar(&result.AllowEnv, "allow-env", "", "required write target: dev or uat")
	flags.BoolVar(&result.DryRun, "dry-run", false, "read and classify Work Items without writes")
	flags.BoolVar(&result.Apply, "apply", false, "apply the reviewed historical disposition")
	if err := flags.Parse(arguments); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("unexpected positional arguments")
	}
	if result.DryRun == result.Apply {
		return options{}, errors.New("exactly one of -dry-run or -apply is required")
	}
	if strings.TrimSpace(result.ManifestPath) == "" {
		return options{}, errors.New("-manifest is required")
	}
	if result.Apply {
		if result.AllowEnv != "dev" && result.AllowEnv != "uat" {
			return options{}, errors.New("-apply requires -allow-env dev|uat")
		}
		if strings.TrimSpace(result.ExportPath) == "" {
			return options{}, errors.New("-apply requires -export")
		}
	} else if result.AllowEnv != "" || result.ExportPath != "" {
		return options{}, errors.New("-allow-env and -export are only valid with -apply")
	}
	return result, nil
}

func validateTarget(cfg agentrunconfig.Config, options options) error {
	if !options.Apply {
		return nil
	}
	if string(cfg.App.Env) != options.AllowEnv {
		return fmt.Errorf(
			"configured APP_ENV %q does not match -allow-env %q",
			cfg.App.Env, options.AllowEnv,
		)
	}
	if cfg.App.Env == agentrunconfig.EnvUAT && cfg.Database.SSLMode != "require" {
		return errors.New("uat historical disposition requires ssl_mode=require")
	}
	return nil
}

func readManifest(path string) (eventsemantic.HistoricalManifest, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return eventsemantic.HistoricalManifest{}, errors.New(
			"could not read historical Event manifest",
		)
	}
	if len(payload) > 1024*1024 {
		return eventsemantic.HistoricalManifest{}, errors.New(
			"historical Event manifest exceeds 1 MiB",
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var manifest eventsemantic.HistoricalManifest
	if err := decoder.Decode(&manifest); err != nil {
		return eventsemantic.HistoricalManifest{}, errors.New(
			"historical Event manifest is invalid",
		)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return eventsemantic.HistoricalManifest{}, errors.New(
			"historical Event manifest is invalid",
		)
	}
	if err := manifest.Validate(); err != nil {
		return eventsemantic.HistoricalManifest{}, err
	}
	return manifest, nil
}

func writeExclusiveJSON(path string, value any) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New(
			"could not create pre-change export without overwriting an existing file",
		)
	}
	defer file.Close()
	if err := encodeJSON(file, value); err != nil {
		return errors.New("could not encode pre-change Work Item export")
	}
	return nil
}

func encodeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
