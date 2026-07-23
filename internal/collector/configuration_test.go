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

func TestProviderBaseURLMustBeUsableAndProtectCredentials(t *testing.T) {
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
			err := ValidateProviderConfig(agentrun.ProviderConfig{
				Key: ProviderCLSTelegraph, BaseURL: test.baseURL,
			})
			if test.wantErr && err == nil {
				t.Fatalf("ValidateProviderConfig(%q) accepted", test.baseURL)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("ValidateProviderConfig(%q) error = %v", test.baseURL, err)
			}
			if err != nil && strings.Contains(err.Error(), "secret:password") {
				t.Fatalf("error leaked URL credentials: %v", err)
			}
		})
	}
	if err := ValidateProviderConfigForEnvironment(agentrun.ProviderConfig{
		Key: ProviderCLSTelegraph, BaseURL: "http://127.0.0.1:9080/v1",
	}, "uat"); err == nil {
		t.Fatal("UAT accepted a plaintext loopback Provider URL")
	}
}
