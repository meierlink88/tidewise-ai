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
	t.Setenv("ADMIN_API_TOKEN", "test-admin-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Secrets.AgentPlatformAPIKey == "" ||
		cfg.Secrets.DatabaseURL == "" ||
		cfg.Secrets.DatabasePassword == "" ||
		cfg.Secrets.JWTSecret == "" ||
		cfg.Secrets.PaymentSecret == "" ||
		cfg.Secrets.CloudSecret == "" ||
		cfg.Secrets.AdminAPIToken == "" {
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
	if cfg.Ingestion.SchedulerTickSeconds <= 0 {
		t.Fatal("cfg.Ingestion.SchedulerTickSeconds must be positive")
	}
	if cfg.Ingestion.SchedulerTimezone != "Asia/Shanghai" {
		t.Fatalf("cfg.Ingestion.SchedulerTimezone = %q, want Asia/Shanghai", cfg.Ingestion.SchedulerTimezone)
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
	if !cfg.Neo4j.Enabled {
		t.Fatal("local neo4j should be enabled")
	}
	if cfg.Neo4j.URI == "" {
		t.Fatal("cfg.Neo4j.URI is empty")
	}
	if cfg.Neo4j.Database == "" {
		t.Fatal("cfg.Neo4j.Database is empty")
	}
	if cfg.Neo4j.UsernameEnv == "" {
		t.Fatal("cfg.Neo4j.UsernameEnv is empty")
	}
	if cfg.Neo4j.PasswordEnv == "" {
		t.Fatal("cfg.Neo4j.PasswordEnv is empty")
	}
	if cfg.Neo4j.ConnectTimeoutSeconds <= 0 {
		t.Fatal("cfg.Neo4j.ConnectTimeoutSeconds must be positive")
	}
	if cfg.Neo4j.MaxConnectionPoolSize <= 0 {
		t.Fatal("cfg.Neo4j.MaxConnectionPoolSize must be positive")
	}
}

func TestLoadSupportsNeo4jDisabledByDefaultForUATAndProd(t *testing.T) {
	t.Setenv("TIDEWISE_CONFIG_DIR", filepath.Join("..", "..", "config"))

	for _, env := range []Environment{EnvUAT, EnvProd} {
		t.Run(string(env), func(t *testing.T) {
			t.Setenv("APP_ENV", string(env))

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.Neo4j.Enabled {
				t.Fatalf("%s neo4j should be disabled by default", env)
			}
		})
	}
}

func TestLoadRejectsInvalidEnabledNeo4jConfig(t *testing.T) {
	base := fullConfigYAML()

	tests := []struct {
		name    string
		neo4j   string
		wantErr string
	}{
		{
			name: "missing uri",
			neo4j: `neo4j:
  enabled: true
  database: neo4j
  username_env: NEO4J_USERNAME
  password_env: NEO4J_PASSWORD
  connect_timeout_seconds: 5
  max_connection_pool_size: 10
`,
			wantErr: "neo4j.uri is required when neo4j is enabled",
		},
		{
			name: "missing username env",
			neo4j: `neo4j:
  enabled: true
  uri: bolt://localhost:7687
  database: neo4j
  password_env: NEO4J_PASSWORD
  connect_timeout_seconds: 5
  max_connection_pool_size: 10
`,
			wantErr: "neo4j.username_env is required when neo4j is enabled",
		},
		{
			name: "invalid timeout",
			neo4j: `neo4j:
  enabled: true
  uri: bolt://localhost:7687
  database: neo4j
  username_env: NEO4J_USERNAME
  password_env: NEO4J_PASSWORD
  connect_timeout_seconds: 0
  max_connection_pool_size: 10
`,
			wantErr: "neo4j.connect_timeout_seconds must be positive when neo4j is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("TIDEWISE_CONFIG_DIR", dir)
			t.Setenv("APP_ENV", string(EnvLocal))

			content := strings.Replace(base, "__NEO4J__", tt.neo4j, 1)
			if err := os.WriteFile(filepath.Join(dir, "config.local.yaml"), []byte(content), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
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
			AdminAPIToken:       "admin-token",
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
		"admin-token",
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

func fullConfigYAML() string {
	return `app:
  name: tidewise-api
server:
  host: 127.0.0.1
  port: 8080
  read_timeout_seconds: 5
  write_timeout_seconds: 10
log:
  level: debug
agent_platform:
  base_url: http://localhost:9000
  callback_path: /api/v1/agent/callbacks
database:
  host: localhost
  port: 5432
  name: tidewise_local
  user: tidewise
  ssl_mode: disable
  max_open_conns: 10
  max_idle_conns: 5
  conn_max_lifetime_seconds: 300
  connect_timeout_seconds: 5
redis:
  address: localhost:6379
__NEO4J__migration:
  directory: migrations
  auto_apply: true
  lock_key: tidewise_schema_migration
ingestion:
  default_timeout_seconds: 10
  batch_size: 50
  scheduler_tick_seconds: 30
  scheduler_timezone: Asia/Shanghai
object_store:
  provider: local
  local_path: .data/raw-objects
rate_limit:
  default_requests_per_minute: 60
security:
  jwt_issuer: tidewise-local
`
}
