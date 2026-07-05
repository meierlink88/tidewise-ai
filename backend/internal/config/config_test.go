package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSupportedEnvironments(t *testing.T) {
	t.Setenv("TIDEWISE_CONFIG_DIR", filepath.Join("..", "..", "config"))

	for _, env := range []Environment{EnvLocal, EnvUAT, EnvProd} {
		t.Run(string(env), func(t *testing.T) {
			t.Setenv("APP_ENV", string(env))

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.App.Env != env {
				t.Fatalf("cfg.App.Env = %q, want %q", cfg.App.Env, env)
			}
			if cfg.App.Name != "tidewise-api" {
				t.Fatalf("cfg.App.Name = %q", cfg.App.Name)
			}
		})
	}
}

func TestLoadDefaultsToLocal(t *testing.T) {
	t.Setenv("TIDEWISE_CONFIG_DIR", filepath.Join("..", "..", "config"))
	t.Setenv("APP_ENV", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Env != EnvLocal {
		t.Fatalf("cfg.App.Env = %q, want %q", cfg.App.Env, EnvLocal)
	}
}

func TestLoadRejectsUnknownEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "sandbox")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRejectsMissingRequiredConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", string(EnvLocal))

	content := []byte("app:\n  name: tidewise-api\n")
	if err := os.WriteFile(filepath.Join(dir, "config.local.yaml"), content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadReadsInjectedSecretNames(t *testing.T) {
	t.Setenv("TIDEWISE_CONFIG_DIR", filepath.Join("..", "..", "config"))
	t.Setenv("APP_ENV", string(EnvLocal))
	t.Setenv("AGENT_PLATFORM_API_KEY", "test-agent-key")
	t.Setenv("DATABASE_PASSWORD", "test-database-password")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("PAYMENT_SECRET", "test-payment-secret")
	t.Setenv("CLOUD_SECRET", "test-cloud-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Secrets.AgentPlatformAPIKey == "" ||
		cfg.Secrets.DatabasePassword == "" ||
		cfg.Secrets.JWTSecret == "" ||
		cfg.Secrets.PaymentSecret == "" ||
		cfg.Secrets.CloudSecret == "" {
		t.Fatal("expected injected secret placeholders to be loaded")
	}
}
