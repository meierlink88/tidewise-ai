package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	agentrunconfig "github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/config"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
)

func main() {
	if len(os.Args) < 2 {
		configFail("usage: agentrun-config set|list|check")
	}
	runtimeConfig, err := agentrunconfig.Load()
	if err != nil {
		configFail("could not load AgentRun configuration")
	}
	databaseURL, err := runtimeConfig.PostgresURL()
	if err != nil {
		configFail("could not build AgentRun database configuration")
	}
	database, err := postgres.Open(context.Background(), databaseURL)
	if err != nil {
		configFail("could not open AgentRun database")
	}
	defer database.Close()
	store := postgres.New(database)
	if !store.SchemaReady(context.Background()) {
		configFail("AgentRun database schema is not ready; run migrations first")
	}

	switch os.Args[1] {
	case "set":
		setProvider(store, os.Args[2:])
	case "list":
		listProviders(store)
	case "check":
		configs, err := store.LoadProviderConfigs(context.Background())
		if err != nil {
			configFail("could not load Provider configurations")
		}
		if _, err := collector.BuildProviderConfiguration(configs); err != nil {
			configFail("Collector Provider configuration is incomplete")
		}
		fmt.Println("Collector Provider configuration is ready")
	default:
		configFail("usage: agentrun-config set|list|check")
	}
}

func setProvider(store *postgres.Store, arguments []string) {
	flags := flag.NewFlagSet("set", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var config collector.ProviderConfig
	var apiKeyStdin bool
	flags.StringVar(&config.Key, "provider", "", "Provider key")
	flags.StringVar(&config.BaseURL, "base-url", "", "Provider Base URL")
	flags.StringVar(&config.Model, "model", "", "LLM model")
	flags.BoolVar(&apiKeyStdin, "api-key-stdin", false, "Read Provider API key from stdin")
	if err := flags.Parse(arguments); err != nil {
		configFail("invalid Provider configuration arguments")
	}
	if apiKeyStdin {
		key, err := io.ReadAll(io.LimitReader(os.Stdin, 64*1024+1))
		if err != nil || len(key) > 64*1024 {
			configFail("could not read Provider API key from stdin")
		}
		config.APIKey = strings.TrimSpace(string(key))
	}
	if err := collector.ValidateProviderConfig(config); err != nil {
		configFail("Provider configuration is invalid or could not be saved")
	}
	if err := store.UpsertProviderConfig(context.Background(), config); err != nil {
		configFail("Provider configuration is invalid or could not be saved")
	}
	fmt.Printf("Provider %s configured\n", config.Key)
}

func listProviders(store *postgres.Store) {
	views, err := store.ListProviderConfigViews(context.Background())
	if err != nil {
		configFail("could not list Provider configurations")
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(views); err != nil {
		configFail("could not encode Provider configurations")
	}
}

func configFail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
