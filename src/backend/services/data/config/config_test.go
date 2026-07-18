package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadReadsOnlyDataServiceConfiguration(t *testing.T) {
	dir := writeTestConfig(t, fullConfigYAML())
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")
	t.Setenv("TIDEWISE_DATABASE_URL", "postgres://data:secret@localhost:5432/tidewise_local?sslmode=disable")
	t.Setenv("DATA_SERVICE_AGENT_TOKEN", "agent-token")
	t.Setenv("DATA_SERVICE_MINIAPP_TOKEN", "miniapp-token")
	t.Setenv("DATA_SERVICE_ADMIN_TOKEN", "admin-token")
	t.Setenv("AGENT_PLATFORM_API_KEY", "must-not-be-loaded")
	t.Setenv("JWT_SECRET", "must-not-be-loaded")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Name != ServiceName || cfg.App.Env != EnvLocal {
		t.Fatalf("app = %#v", cfg.App)
	}
	if cfg.Database.Name != "tidewise_local" || cfg.Migration.Directory != "migrations" || !cfg.Neo4j.Enabled {
		t.Fatalf("Data configuration = %#v/%#v/%#v", cfg.Database, cfg.Migration, cfg.Neo4j)
	}
	if cfg.Secrets.DatabaseURL == "" || cfg.Secrets.DataServiceAgentToken == "" || cfg.Secrets.DataServiceMiniappToken == "" || cfg.Secrets.DataServiceAdminToken == "" {
		t.Fatalf("Data secrets were not loaded: %#v", cfg.Secrets)
	}
}

func TestLoadDefaultsToLocalAndRejectsUnknownEnvironment(t *testing.T) {
	dir := writeTestConfig(t, fullConfigYAML())
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Env != EnvLocal {
		t.Fatalf("environment = %q", cfg.App.Env)
	}
	t.Setenv("APP_ENV", "sandbox")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted an unsupported environment")
	}
}

func TestLoadRejectsMissingDataConfiguration(t *testing.T) {
	dir := writeTestConfig(t, "app:\n  name: ignored\n")
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted incomplete Data configuration")
	}
}

func TestLoadRejectsInvalidEnabledNeo4jConfiguration(t *testing.T) {
	invalid := strings.Replace(fullConfigYAML(), "uri: bolt://localhost:7687", "uri: ''", 1)
	dir := writeTestConfig(t, invalid)
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "neo4j.uri is required") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestSecretsAreNotSerialized(t *testing.T) {
	cfg := Config{Secrets: SecretConfig{
		DatabaseURL:             "postgres://data:secret@localhost/db",
		DatabasePassword:        "database-secret",
		DataServiceAgentToken:   "agent-token",
		DataServiceMiniappToken: "miniapp-token",
		DataServiceAdminToken:   "admin-token",
	}}
	content, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"database-secret", "agent-token", "miniapp-token", "admin-token"} {
		if strings.Contains(string(content), secret) {
			t.Fatalf("serialized configuration leaked %q", secret)
		}
	}
}

func TestPostgresURLPrefersInjectedURLAndCanBuildFromFields(t *testing.T) {
	injected := Config{Secrets: SecretConfig{DatabaseURL: "postgres://override:secret@localhost:5432/override?sslmode=disable"}}
	got, err := injected.PostgresURL()
	if err != nil || got != injected.Secrets.DatabaseURL {
		t.Fatalf("injected PostgresURL = %q, %v", got, err)
	}

	configured := Config{
		Database: DatabaseConfig{Host: "db.local", Port: 5432, Name: "tidewise_local", User: "tidewise", SSLMode: "disable", ConnectTimeoutSeconds: 7},
		Secrets:  SecretConfig{DatabasePassword: "test-password"},
	}
	got, err = configured.PostgresURL()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"postgres://tidewise:test-password@db.local:5432/tidewise_local", "connect_timeout=7", "sslmode=disable"} {
		if !strings.Contains(got, want) {
			t.Fatalf("PostgresURL() = %q, want %q", got, want)
		}
	}
}

func writeTestConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.local.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func fullConfigYAML() string {
	return `app:
  name: ignored
server:
  host: 127.0.0.1
  port: 8081
  read_timeout_seconds: 5
  write_timeout_seconds: 15
log:
  level: debug
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
neo4j:
  enabled: true
  uri: bolt://localhost:7687
  database: neo4j
  username_env: NEO4J_USERNAME
  password_env: NEO4J_PASSWORD
  connect_timeout_seconds: 5
  max_connection_pool_size: 10
migration:
  directory: migrations
  auto_apply: false
  lock_key: tidewise_schema_migration
`
}
