package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	agentrunconfig "github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/config"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
)

const configurationUsage = "usage: agentrun-config model set|list | connector set|list | check"

type configurationStore interface {
	UpsertModelProviderConfig(context.Context, agentrun.ModelProviderConfig) error
	LoadModelProviderConfigs(context.Context) (map[string]agentrun.ModelProviderConfig, error)
	ListModelProviderConfigViews(context.Context) ([]agentrun.ModelProviderConfigView, error)
	UpsertConnectorConfig(context.Context, agentrun.ConnectorConfig) error
	LoadConnectorConfigs(context.Context) (map[string]agentrun.ConnectorConfig, error)
	ListConnectorConfigViews(context.Context) ([]agentrun.ConnectorConfigView, error)
}

func main() {
	if len(os.Args) < 2 {
		configFail(configurationUsage)
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
	if err := runConfigurationCommand(
		context.Background(),
		store,
		string(runtimeConfig.App.Env),
		os.Args[1:],
		os.Stdin,
		os.Stdout,
	); err != nil {
		configFail(err.Error())
	}
}

func runConfigurationCommand(
	ctx context.Context,
	store configurationStore,
	environment string,
	arguments []string,
	stdin io.Reader,
	stdout io.Writer,
) error {
	if store == nil || len(arguments) == 0 {
		return errors.New(configurationUsage)
	}
	if len(arguments) == 1 && arguments[0] == "check" {
		return checkConfiguration(ctx, store, environment, stdout)
	}
	if len(arguments) != 2 && (len(arguments) < 2 || arguments[1] != "set") {
		return errors.New(configurationUsage)
	}
	switch arguments[0] {
	case "model":
		switch arguments[1] {
		case "set":
			return setModelProviderConfiguration(ctx, store, environment, arguments[2:], stdin, stdout)
		case "list":
			return listModelProviderConfigurations(ctx, store, stdout)
		}
	case "connector":
		switch arguments[1] {
		case "set":
			return setConnectorConfiguration(ctx, store, environment, arguments[2:], stdin, stdout)
		case "list":
			return listConnectorConfigurations(ctx, store, stdout)
		}
	}
	return errors.New(configurationUsage)
}

func setModelProviderConfiguration(
	ctx context.Context,
	store configurationStore,
	environment string,
	arguments []string,
	stdin io.Reader,
	stdout io.Writer,
) error {
	flags := flag.NewFlagSet("model set", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var config agentrun.ModelProviderConfig
	var apiKeyStdin bool
	flags.StringVar(&config.ProviderKey, "provider", "", "Model Provider key")
	flags.StringVar(&config.BaseURL, "base-url", "", "Model Provider Base URL")
	flags.StringVar(&config.Model, "model", "", "model name")
	flags.BoolVar(&apiKeyStdin, "api-key-stdin", false, "read Model Provider API key from stdin")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return errors.New("invalid Model Provider Configuration arguments")
	}
	if apiKeyStdin {
		key, err := readAPIKey(stdin)
		if err != nil {
			return errors.New("could not read Model Provider API key from stdin")
		}
		config.APIKey = key
	}
	if err := collector.ValidateModelProviderConfigForEnvironment(config, environment); err != nil {
		return errors.New("Model Provider Configuration is invalid or could not be saved")
	}
	if err := store.UpsertModelProviderConfig(ctx, config); err != nil {
		return errors.New("Model Provider Configuration is invalid or could not be saved")
	}
	_, err := fmt.Fprintf(stdout, "Model Provider %s configured; the next Agent Execution will use the change\n", config.ProviderKey)
	return err
}

func setConnectorConfiguration(
	ctx context.Context,
	store configurationStore,
	environment string,
	arguments []string,
	stdin io.Reader,
	stdout io.Writer,
) error {
	flags := flag.NewFlagSet("connector set", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var config agentrun.ConnectorConfig
	var apiKeyStdin bool
	flags.StringVar(&config.ConnectorKey, "connector", "", "Connector key")
	flags.StringVar(&config.BaseURL, "base-url", "", "Connector Base URL")
	flags.BoolVar(&apiKeyStdin, "api-key-stdin", false, "read Connector API key from stdin")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return errors.New("invalid Connector Configuration arguments")
	}
	if apiKeyStdin {
		key, err := readAPIKey(stdin)
		if err != nil {
			return errors.New("could not read Connector API key from stdin")
		}
		config.APIKey = key
	}
	if err := collector.ValidateConnectorConfigForEnvironment(config, environment); err != nil {
		return errors.New("Connector Configuration is invalid or could not be saved")
	}
	if err := store.UpsertConnectorConfig(ctx, config); err != nil {
		return errors.New("Connector Configuration is invalid or could not be saved")
	}
	_, err := fmt.Fprintf(stdout, "Connector %s configured; the next Agent Execution will use the change\n", config.ConnectorKey)
	return err
}

func readAPIKey(reader io.Reader) (string, error) {
	key, err := io.ReadAll(io.LimitReader(reader, 64*1024+1))
	if err != nil || len(key) > 64*1024 {
		return "", errors.New("API key input is invalid")
	}
	return strings.TrimSpace(string(key)), nil
}

func listModelProviderConfigurations(ctx context.Context, store configurationStore, stdout io.Writer) error {
	views, err := store.ListModelProviderConfigViews(ctx)
	if err != nil {
		return errors.New("could not list Model Provider Configurations")
	}
	if err := encodeConfigurationViews(stdout, views); err != nil {
		return errors.New("could not encode Model Provider Configurations")
	}
	return nil
}

func listConnectorConfigurations(ctx context.Context, store configurationStore, stdout io.Writer) error {
	views, err := store.ListConnectorConfigViews(ctx)
	if err != nil {
		return errors.New("could not list Connector Configurations")
	}
	if err := encodeConfigurationViews(stdout, views); err != nil {
		return errors.New("could not encode Connector Configurations")
	}
	return nil
}

func encodeConfigurationViews(writer io.Writer, views any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(views)
}

func checkConfiguration(
	ctx context.Context,
	store configurationStore,
	environment string,
	stdout io.Writer,
) error {
	models, err := store.LoadModelProviderConfigs(ctx)
	if err != nil {
		return errors.New("could not load Model Provider Configurations")
	}
	connectors, err := store.LoadConnectorConfigs(ctx)
	if err != nil {
		return errors.New("could not load Connector Configurations")
	}
	if _, err := collector.BuildRuntimeConfigurationForEnvironment(models, connectors, environment); err != nil {
		return errors.New("Collector configuration is incomplete")
	}
	_, err = fmt.Fprintln(stdout, "Collector configuration is ready")
	return err
}

func configFail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
