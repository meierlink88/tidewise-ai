package collector

import (
	"strings"
	"testing"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
)

func TestProviderConfigurationOwnsCollectorRequirements(t *testing.T) {
	t.Parallel()

	if err := ValidateProviderConfig(agentrun.ProviderConfig{Key: "unknown", BaseURL: "https://example.test"}); err == nil {
		t.Fatal("unknown Collector Provider was accepted")
	}
	if err := ValidateProviderConfig(agentrun.ProviderConfig{Key: ProviderDeepSeek, BaseURL: "https://deepseek.test", APIKey: "key"}); err == nil {
		t.Fatal("DeepSeek configuration without a model was accepted")
	}
	if err := ValidateProviderConfig(agentrun.ProviderConfig{Key: ProviderCLSTelegraph, BaseURL: "https://cls.test"}); err != nil {
		t.Fatalf("keyless feed configuration was rejected: %v", err)
	}

	loaded := make(map[string]agentrun.ProviderConfig)
	for _, key := range append([]string{ProviderDeepSeek}, ConnectorKeys()...) {
		loaded[key] = agentrun.ProviderConfig{Key: key, BaseURL: "https://provider.test", Model: "deepseek-chat", APIKey: "key"}
	}
	configuration, err := BuildProviderConfiguration(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.DeepSeek.Key != ProviderDeepSeek || len(configuration.Connectors) != 7 {
		t.Fatalf("configuration = %#v", configuration)
	}

	delete(loaded, ProviderTavily)
	_, err = BuildProviderConfiguration(loaded)
	if err == nil || !strings.Contains(err.Error(), ProviderTavily) {
		t.Fatalf("missing Tavily error = %v", err)
	}
}

func TestConnectorKeysReturnsIndependentCopy(t *testing.T) {
	t.Parallel()

	first := ConnectorKeys()
	first[0] = "changed"
	if second := ConnectorKeys(); second[0] == "changed" {
		t.Fatal("ConnectorKeys exposed mutable package state")
	}
}
