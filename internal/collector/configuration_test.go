package collector

import (
	"strings"
	"testing"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
)

func TestRuntimeConfigurationSeparatesModelAndConnectorRequirements(t *testing.T) {
	t.Parallel()

	if err := ValidateModelProviderConfig(agentrun.ModelProviderConfig{
		ProviderKey: "unknown", BaseURL: "https://example.test", Model: "model", APIKey: "key",
	}); err == nil {
		t.Fatal("unknown Model Provider was accepted")
	}
	if err := ValidateModelProviderConfig(agentrun.ModelProviderConfig{
		ProviderKey: ModelProviderDeepSeek, BaseURL: "https://deepseek.test", APIKey: "key",
	}); err == nil {
		t.Fatal("DeepSeek configuration without a model was accepted")
	}
	if err := ValidateModelProviderConfig(agentrun.ModelProviderConfig{
		ProviderKey: ModelProviderDeepSeek, BaseURL: "https://deepseek.test", Model: "deepseek-chat",
	}); err == nil {
		t.Fatal("DeepSeek configuration without a key was accepted")
	}
	if err := ValidateConnectorConfig(agentrun.ConnectorConfig{
		ConnectorKey: ConnectorTavily, BaseURL: "https://tavily.test",
	}); err != nil {
		t.Fatalf("keyless Connector Configuration was rejected: %v", err)
	}

	models := map[string]agentrun.ModelProviderConfig{
		ModelProviderDeepSeek: {
			ProviderKey: ModelProviderDeepSeek,
			BaseURL:     "https://deepseek.test",
			Model:       "deepseek-chat",
			APIKey:      "key",
		},
	}
	connectors := make(map[string]agentrun.ConnectorConfig)
	for _, key := range ConnectorKeys() {
		connectors[key] = agentrun.ConnectorConfig{ConnectorKey: key, BaseURL: "https://connector.test"}
	}
	configuration, err := BuildRuntimeConfiguration(models, connectors)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.ModelProvider.ProviderKey != ModelProviderDeepSeek || len(configuration.Connectors) != 7 {
		t.Fatalf("configuration = %#v", configuration)
	}

	delete(connectors, ConnectorTavily)
	_, err = BuildRuntimeConfiguration(models, connectors)
	if err == nil || !strings.Contains(err.Error(), ConnectorTavily) {
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

func TestConfigurationBaseURLMustBeUsableAndProtectCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{name: "HTTPS", baseURL: "https://api.example.test/v1"},
		{name: "loopback HTTP hostname", baseURL: "http://localhost:9080/v1"},
		{name: "loopback HTTP IPv4", baseURL: "http://127.0.0.1:9080/v1"},
		{name: "loopback HTTP IPv6", baseURL: "http://[::1]:9080/v1"},
		{name: "relative", baseURL: "/v1/search", wantErr: true},
		{name: "hostless", baseURL: "https:///v1/search", wantErr: true},
		{name: "unsupported scheme", baseURL: "ftp://api.example.test/v1", wantErr: true},
		{name: "remote plaintext", baseURL: "http://api.example.test/v1", wantErr: true},
		{name: "embedded credentials", baseURL: "https://secret:password@api.example.test/v1", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateConnectorConfig(agentrun.ConnectorConfig{
				ConnectorKey: ConnectorCLSTelegraph, BaseURL: test.baseURL,
			})
			if test.wantErr && err == nil {
				t.Fatalf("ValidateConnectorConfig(%q) accepted", test.baseURL)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("ValidateConnectorConfig(%q) error = %v", test.baseURL, err)
			}
			if err != nil && strings.Contains(err.Error(), "secret:password") {
				t.Fatalf("error leaked URL credentials: %v", err)
			}
		})
	}
	if err := ValidateConnectorConfigForEnvironment(agentrun.ConnectorConfig{
		ConnectorKey: ConnectorCLSTelegraph, BaseURL: "http://127.0.0.1:9080/v1",
	}, "uat"); err == nil {
		t.Fatal("UAT accepted a plaintext loopback Connector URL")
	}
}
