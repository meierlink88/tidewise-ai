package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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
	t.Setenv("TIDEWISE_DATABASE_URL", "postgres://test-user:test-password@localhost:5432/tidewise_local?sslmode=disable")
	t.Setenv("DATABASE_PASSWORD", "test-database-password")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("PAYMENT_SECRET", "test-payment-secret")
	t.Setenv("CLOUD_SECRET", "test-cloud-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Secrets.AgentPlatformAPIKey == "" ||
		cfg.Secrets.DatabaseURL == "" ||
		cfg.Secrets.DatabasePassword == "" ||
		cfg.Secrets.JWTSecret == "" ||
		cfg.Secrets.PaymentSecret == "" ||
		cfg.Secrets.CloudSecret == "" {
		t.Fatal("expected injected secret placeholders to be loaded")
	}
}

func TestLoadReadsOperationalConfig(t *testing.T) {
	t.Setenv("TIDEWISE_CONFIG_DIR", filepath.Join("..", "..", "config"))
	t.Setenv("APP_ENV", string(EnvLocal))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Migration.Directory == "" {
		t.Fatal("cfg.Migration.Directory is empty")
	}
	if !cfg.Migration.AutoApply {
		t.Fatal("local migration auto apply should be enabled")
	}
	if cfg.Ingestion.DefaultTimeoutSeconds <= 0 {
		t.Fatal("cfg.Ingestion.DefaultTimeoutSeconds must be positive")
	}
	if cfg.ObjectStore.LocalPath == "" {
		t.Fatal("cfg.ObjectStore.LocalPath is empty")
	}
	if cfg.RateLimit.DefaultRequestsPerMinute <= 0 {
		t.Fatal("cfg.RateLimit.DefaultRequestsPerMinute must be positive")
	}
	if cfg.Database.User == "" {
		t.Fatal("cfg.Database.User is empty")
	}
	if cfg.Database.SSLMode == "" {
		t.Fatal("cfg.Database.SSLMode is empty")
	}
	if cfg.Database.MaxOpenConns <= 0 {
		t.Fatal("cfg.Database.MaxOpenConns must be positive")
	}
	if cfg.Database.ConnectTimeoutSeconds <= 0 {
		t.Fatal("cfg.Database.ConnectTimeoutSeconds must be positive")
	}
}

func TestProdMigrationAutoApplyDisabledByDefault(t *testing.T) {
	t.Setenv("TIDEWISE_CONFIG_DIR", filepath.Join("..", "..", "config"))
	t.Setenv("APP_ENV", string(EnvProd))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Migration.AutoApply {
		t.Fatal("prod migration auto apply should be disabled by default")
	}
}

func TestSecretsAreNotSerializedToYAML(t *testing.T) {
	cfg := Config{
		App: AppConfig{
			Name: "tidewise-api",
			Env:  EnvLocal,
		},
		Secrets: SecretConfig{
			AgentPlatformAPIKey: "agent-secret",
			DatabaseURL:         "postgres://user:database-secret@localhost/db",
			DatabasePassword:    "database-secret",
			JWTSecret:           "jwt-secret",
			PaymentSecret:       "payment-secret",
			CloudSecret:         "cloud-secret",
		},
	}

	content, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	serialized := string(content)
	for _, secret := range []string{
		"agent-secret",
		"postgres://user:database-secret@localhost/db",
		"database-secret",
		"jwt-secret",
		"payment-secret",
		"cloud-secret",
	} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("serialized config leaked secret %q", secret)
		}
	}
}

func TestDatabaseConnectionStringUsesInjectedURLFirst(t *testing.T) {
	cfg := Config{
		Database: DatabaseConfig{
			Host:                  "localhost",
			Port:                  5432,
			Name:                  "tidewise_local",
			User:                  "tidewise",
			SSLMode:               "disable",
			ConnectTimeoutSeconds: 5,
		},
		Secrets: SecretConfig{
			DatabaseURL: "postgres://override:secret@localhost:5432/override?sslmode=disable",
		},
	}

	dsn, err := cfg.PostgresURL()
	if err != nil {
		t.Fatalf("PostgresURL() error = %v", err)
	}

	if dsn != cfg.Secrets.DatabaseURL {
		t.Fatalf("PostgresURL() = %q, want injected URL", dsn)
	}
}

func TestDatabaseConnectionStringBuildsFromConfigAndPassword(t *testing.T) {
	cfg := Config{
		Database: DatabaseConfig{
			Host:                  "db.local",
			Port:                  5432,
			Name:                  "tidewise_local",
			User:                  "tidewise",
			SSLMode:               "disable",
			ConnectTimeoutSeconds: 7,
		},
		Secrets: SecretConfig{
			DatabasePassword: "test-password",
		},
	}

	dsn, err := cfg.PostgresURL()
	if err != nil {
		t.Fatalf("PostgresURL() error = %v", err)
	}

	for _, want := range []string{
		"postgres://tidewise:test-password@db.local:5432/tidewise_local",
		"connect_timeout=7",
		"sslmode=disable",
	} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("PostgresURL() = %q, want to contain %q", dsn, want)
		}
	}
}
