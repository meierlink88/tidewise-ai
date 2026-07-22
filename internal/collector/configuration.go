package collector

import (
	"fmt"
	"strings"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
)

type ProviderConfig = agentrun.ProviderConfig

type ProviderConfiguration struct {
	DeepSeek   agentrun.ProviderConfig
	Connectors map[string]agentrun.ProviderConfig
}

type ProviderConfigView = agentrun.ProviderConfigView

const (
	ProviderDeepSeek          = "deepseek"
	ProviderParallelSearch    = "parallel_search"
	ProviderTavily            = "tavily"
	ProviderBocha             = "bocha"
	ProviderCLSTelegraph      = "cls_telegraph"
	ProviderEastmoneyFastNews = "eastmoney_fastnews"
	ProviderEastmoneyStock    = "eastmoney_stock_news"
	ProviderSTCNQuickNews     = "stcn_quicknews"
)

var connectorKeys = []string{
	ProviderParallelSearch,
	ProviderTavily,
	ProviderBocha,
	ProviderCLSTelegraph,
	ProviderEastmoneyFastNews,
	ProviderEastmoneyStock,
	ProviderSTCNQuickNews,
}

var providerRequirements = map[string]struct {
	model bool
	key   bool
}{
	ProviderDeepSeek:          {model: true, key: true},
	ProviderParallelSearch:    {key: true},
	ProviderTavily:            {key: true},
	ProviderBocha:             {key: true},
	ProviderCLSTelegraph:      {},
	ProviderEastmoneyFastNews: {},
	ProviderEastmoneyStock:    {},
	ProviderSTCNQuickNews:     {},
}

func ConnectorKeys() []string {
	return append([]string(nil), connectorKeys...)
}

func ValidateProviderConfig(config agentrun.ProviderConfig) error {
	requirement, known := providerRequirements[config.Key]
	if !known {
		return fmt.Errorf("unknown Provider configuration key %q", config.Key)
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		return fmt.Errorf("Provider Base URL is required")
	}
	if requirement.model && strings.TrimSpace(config.Model) == "" {
		return fmt.Errorf("Provider model is required")
	}
	if requirement.key && strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("Provider API key is required")
	}
	return nil
}

func BuildProviderConfiguration(loaded map[string]agentrun.ProviderConfig) (ProviderConfiguration, error) {
	for key := range providerRequirements {
		config, exists := loaded[key]
		if !exists {
			return ProviderConfiguration{}, fmt.Errorf("required Provider configuration %s is incomplete", key)
		}
		if err := ValidateProviderConfig(config); err != nil {
			return ProviderConfiguration{}, fmt.Errorf("required Provider configuration %s is incomplete", key)
		}
	}
	connectors := make(map[string]agentrun.ProviderConfig, len(connectorKeys))
	for _, key := range connectorKeys {
		connectors[key] = loaded[key]
	}
	return ProviderConfiguration{DeepSeek: loaded[ProviderDeepSeek], Connectors: connectors}, nil
}
