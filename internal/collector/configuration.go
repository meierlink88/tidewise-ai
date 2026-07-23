package collector

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
)

type RuntimeConfiguration struct {
	ModelProvider agentrun.ModelProviderConfig
	Connectors    map[string]agentrun.ConnectorConfig
}

const (
	ModelProviderDeepSeek      = "deepseek"
	ConnectorParallelSearch    = "parallel_search"
	ConnectorTavily            = "tavily"
	ConnectorBocha             = "bocha"
	ConnectorCLSTelegraph      = "cls_telegraph"
	ConnectorEastmoneyFastNews = "eastmoney_fastnews"
	ConnectorEastmoneyStock    = "eastmoney_stock_news"
	ConnectorSTCNQuickNews     = "stcn_quicknews"
)

var connectorKeys = []string{
	ConnectorParallelSearch,
	ConnectorTavily,
	ConnectorBocha,
	ConnectorCLSTelegraph,
	ConnectorEastmoneyFastNews,
	ConnectorEastmoneyStock,
	ConnectorSTCNQuickNews,
}

var connectorKeySet = func() map[string]struct{} {
	keys := make(map[string]struct{}, len(connectorKeys))
	for _, key := range connectorKeys {
		keys[key] = struct{}{}
	}
	return keys
}()

func ConnectorKeys() []string {
	return append([]string(nil), connectorKeys...)
}

func ValidateModelProviderConfig(config agentrun.ModelProviderConfig) error {
	return ValidateModelProviderConfigForEnvironment(config, "dev")
}

func ValidateModelProviderConfigForEnvironment(config agentrun.ModelProviderConfig, environment string) error {
	if config.ProviderKey != ModelProviderDeepSeek {
		return fmt.Errorf("unknown Model Provider Configuration key %q", config.ProviderKey)
	}
	if !validConfigurationBaseURL(config.BaseURL, environment) {
		return fmt.Errorf("Model Provider Base URL must be an absolute HTTPS URL or loopback HTTP URL without credentials")
	}
	if strings.TrimSpace(config.Model) == "" {
		return fmt.Errorf("Model Provider model is required")
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("Model Provider API key is required")
	}
	return nil
}

func ValidateConnectorConfig(config agentrun.ConnectorConfig) error {
	return ValidateConnectorConfigForEnvironment(config, "dev")
}

func ValidateConnectorConfigForEnvironment(config agentrun.ConnectorConfig, environment string) error {
	if _, known := connectorKeySet[config.ConnectorKey]; !known {
		return fmt.Errorf("unknown Connector Configuration key %q", config.ConnectorKey)
	}
	if !validConfigurationBaseURL(config.BaseURL, environment) {
		return fmt.Errorf("Connector Base URL must be an absolute HTTPS URL or loopback HTTP URL without credentials")
	}
	return nil
}

func validConfigurationBaseURL(raw, environment string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return true
	case "http":
		if environment != "dev" {
			return false
		}
		host := parsed.Hostname()
		if strings.EqualFold(host, "localhost") {
			return true
		}
		address := net.ParseIP(host)
		return address != nil && address.IsLoopback()
	default:
		return false
	}
}

func BuildRuntimeConfiguration(
	models map[string]agentrun.ModelProviderConfig,
	connectors map[string]agentrun.ConnectorConfig,
) (RuntimeConfiguration, error) {
	return BuildRuntimeConfigurationForEnvironment(models, connectors, "dev")
}

func BuildRuntimeConfigurationForEnvironment(
	models map[string]agentrun.ModelProviderConfig,
	connectors map[string]agentrun.ConnectorConfig,
	environment string,
) (RuntimeConfiguration, error) {
	if len(models) != 1 {
		return RuntimeConfiguration{}, fmt.Errorf("required Model Provider Configuration %s is incomplete", ModelProviderDeepSeek)
	}
	modelConfig, exists := models[ModelProviderDeepSeek]
	if !exists {
		return RuntimeConfiguration{}, fmt.Errorf("required Model Provider Configuration %s is incomplete", ModelProviderDeepSeek)
	}
	if err := ValidateModelProviderConfigForEnvironment(modelConfig, environment); err != nil {
		return RuntimeConfiguration{}, fmt.Errorf("required Model Provider Configuration %s is incomplete", ModelProviderDeepSeek)
	}
	configured := make(map[string]agentrun.ConnectorConfig, len(connectorKeys))
	for _, key := range connectorKeys {
		config, exists := connectors[key]
		if !exists {
			return RuntimeConfiguration{}, fmt.Errorf("required Connector Configuration %s is incomplete", key)
		}
		if err := ValidateConnectorConfigForEnvironment(config, environment); err != nil {
			return RuntimeConfiguration{}, fmt.Errorf("required Connector Configuration %s is incomplete", key)
		}
		configured[key] = config
	}
	if len(connectors) != len(connectorKeys) {
		return RuntimeConfiguration{}, fmt.Errorf("Connector Configurations contain an unknown key")
	}
	return RuntimeConfiguration{ModelProvider: modelConfig, Connectors: configured}, nil
}
