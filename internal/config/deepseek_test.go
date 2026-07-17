package config

import (
	"strings"
	"testing"
	"time"
)

func TestDeepSeekConfigUsesSafeDefaults(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")
	t.Setenv("DEEPSEEK_MODEL", "")
	t.Setenv("DEEPSEEK_BASE_URL", "")
	t.Setenv("DEEPSEEK_TIMEOUT", "")

	config, err := LoadDeepSeekConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.APIKey != "test-key" || config.Model != "deepseek-chat" || config.BaseURL != "" || config.Timeout != 30*time.Second {
		t.Fatalf("unexpected config: %+v", config)
	}
}

func TestDeepSeekConfigAcceptsOverrides(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", " test-key ")
	t.Setenv("DEEPSEEK_MODEL", " deepseek-reasoner ")
	t.Setenv("DEEPSEEK_BASE_URL", " https://deepseek.test/v1 ")
	t.Setenv("DEEPSEEK_TIMEOUT", "45s")

	config, err := LoadDeepSeekConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.APIKey != "test-key" || config.Model != "deepseek-reasoner" || config.BaseURL != "https://deepseek.test/v1" || config.Timeout != 45*time.Second {
		t.Fatalf("unexpected config: %+v", config)
	}
}

func TestDeepSeekConfigRejectsMissingKeyWithoutLeakingValues(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "  ")
	t.Setenv("DEEPSEEK_TIMEOUT", "secret-invalid-timeout")

	_, err := LoadDeepSeekConfig()
	if err == nil {
		t.Fatal("expected missing key error")
	}
	if strings.Contains(err.Error(), "secret-invalid-timeout") {
		t.Fatalf("error leaked configuration value: %v", err)
	}
}

func TestDeepSeekConfigRejectsInvalidTimeoutWithoutLeakingValues(t *testing.T) {
	for _, value := range []string{"secret-invalid-timeout", "0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("DEEPSEEK_API_KEY", "secret-api-key")
			t.Setenv("DEEPSEEK_TIMEOUT", value)
			_, err := LoadDeepSeekConfig()
			if err == nil {
				t.Fatal("expected timeout error")
			}
			if strings.Contains(err.Error(), value) || strings.Contains(err.Error(), "secret-api-key") {
				t.Fatalf("error leaked sensitive configuration: %v", err)
			}
		})
	}
}
