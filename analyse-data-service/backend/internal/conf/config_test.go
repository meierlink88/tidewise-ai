package conf

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
	t.Setenv("TIDEWISW_DB_PASSWORD", "database-secret")
	t.Setenv("TIDEWISE_DB_HOST", "postgres.infrastructure.internal")
	t.Setenv("DATA_SERVICE_TOKEN", "service-token")
	t.Setenv("DATA_NEO4J_HEALTH_URI", "bolt://neo4j.internal:7687")
	t.Setenv("DATA_NEO4J_HEALTH_USERNAME", "health-reader")
	t.Setenv("DATA_NEO4J_HEALTH_PASSWORD", "health-secret")
	t.Setenv("AGENT_PLATFORM_API_KEY", "must-not-be-loaded")
	t.Setenv("JWT_SECRET", "must-not-be-loaded")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Name != ServiceName || cfg.App.Env != EnvLocal {
		t.Fatalf("app = %#v", cfg.App)
	}
	if cfg.Database.Host != "postgres.infrastructure.internal" || cfg.Database.Name != "tidewise_local" || cfg.Migration.Directory != "migrations" {
		t.Fatalf("Data configuration = %#v/%#v", cfg.Database, cfg.Migration)
	}
	if cfg.Secrets.DatabasePassword != "database-secret" || cfg.Secrets.ServiceToken != "service-token" {
		t.Fatalf("Data secrets were not loaded: %#v", cfg.Secrets)
	}
	if cfg.Secrets.Neo4jHealthUsername != "health-reader" || cfg.Secrets.Neo4jHealthPassword != "health-secret" {
		t.Fatalf("Neo4j health credentials were not loaded: %#v", cfg.Secrets)
	}
	if cfg.Neo4jHealth.URI != "bolt://neo4j.internal:7687" || cfg.Neo4jHealth.Database != "neo4j" || cfg.Neo4jHealth.TimeoutSeconds != 1 {
		t.Fatalf("Neo4j health target = %#v", cfg.Neo4jHealth)
	}
}

func TestLoadDatabaseOperationRequiresOnlyDatabasePassword(t *testing.T) {
	dir := writeTestConfig(t, fullConfigYAML())
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")
	t.Setenv("TIDEWISW_DB_PASSWORD", "database-secret")
	t.Setenv("DATA_SERVICE_TOKEN", "must-not-be-loaded")

	cfg, err := LoadDatabaseOperation()
	if err != nil {
		t.Fatalf("LoadDatabaseOperation() error = %v", err)
	}
	if cfg.Secrets.DatabasePassword != "database-secret" || cfg.Secrets.ServiceToken != "" {
		t.Fatalf("database operation secrets = %#v", cfg.Secrets)
	}

	t.Setenv("TIDEWISW_DB_PASSWORD", "")
	if _, err := LoadDatabaseOperation(); err == nil ||
		!strings.Contains(err.Error(), "TIDEWISW_DB_PASSWORD") {
		t.Fatalf("LoadDatabaseOperation() error = %v, want missing database password", err)
	}
}

func TestLoadDatabaseOperationRequiresEncryptedUATDatabase(t *testing.T) {
	uatConfig := strings.Replace(fullConfigYAML(), "env: local", "env: uat", 1)
	dir := writeTestConfig(t, uatConfig)
	if err := os.Rename(filepath.Join(dir, "config.local.yaml"), filepath.Join(dir, "config.uat.yaml")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "uat")
	t.Setenv("TIDEWISW_DB_PASSWORD", "database-secret")

	if _, err := LoadDatabaseOperation(); err == nil || !strings.Contains(err.Error(), "ssl_mode=require") {
		t.Fatalf("LoadDatabaseOperation() error = %v, want encrypted UAT rejection", err)
	}
}

func TestLoadDefaultsToLocalAndRejectsUnknownEnvironment(t *testing.T) {
	dir := writeTestConfig(t, fullConfigYAML())
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "")
	t.Setenv("DATA_SERVICE_TOKEN", "service-token")
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

func TestSecretsAreNotSerialized(t *testing.T) {
	cfg := Config{Secrets: SecretConfig{
		DatabasePassword:    "database-secret",
		ServiceToken:        "service-token",
		Neo4jHealthPassword: strings.Repeat("n", 24),
	}}
	content, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"database-secret", "service-token", "neo4j-health-secret"} {
		if strings.Contains(string(content), secret) {
			t.Fatalf("serialized configuration leaked %q", secret)
		}
	}
}

func TestPostgresURLBuildsFromNonSecretFieldsAndPassword(t *testing.T) {
	configured := Config{
		Database: DatabaseConfig{Host: "db.local", Port: 5432, Name: "tidewise_local", User: "tidewise", SSLMode: "disable", ConnectTimeoutSeconds: 7},
		Secrets:  SecretConfig{DatabasePassword: "test-password"},
	}
	got, err := configured.PostgresURL()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"postgres://tidewise:test-password@db.local:5432/tidewise_local", "connect_timeout=7", "sslmode=disable"} {
		if !strings.Contains(got, want) {
			t.Fatalf("PostgresURL() = %q, want %q", got, want)
		}
	}
}

func TestLoadUATRequiresPasswordAndEncryptedYAMLDatabaseConfiguration(t *testing.T) {
	uatConfig := strings.Replace(fullConfigYAML(), "env: local", "env: uat", 1)
	uatConfig = strings.Replace(uatConfig, "ssl_mode: disable", "ssl_mode: require", 1)
	dir := writeTestConfig(t, uatConfig)
	if err := os.Rename(filepath.Join(dir, "config.local.yaml"), filepath.Join(dir, "config.uat.yaml")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "uat")
	t.Setenv("DATA_SERVICE_TOKEN", "service-token")
	t.Setenv("TIDEWISW_DB_PASSWORD", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TIDEWISW_DB_PASSWORD") {
		t.Fatalf("Load() error = %v, want missing database password rejection", err)
	}
	t.Setenv("TIDEWISW_DB_PASSWORD", "database-secret")
	t.Setenv("DATA_NEO4J_HEALTH_URI", "bolt://neo4j.internal:7687")
	t.Setenv("DATA_NEO4J_HEALTH_USERNAME", "health-reader")
	t.Setenv("DATA_NEO4J_HEALTH_PASSWORD", "health-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() rejected valid UAT database configuration: %v", err)
	}
	dsn, err := cfg.PostgresURL()
	if err != nil || !strings.Contains(dsn, "database-secret") || !strings.Contains(dsn, "sslmode=require") {
		t.Fatalf("PostgresURL() = %q, %v", dsn, err)
	}
}

func TestLoadRejectsPartialNeo4jHealthCredentials(t *testing.T) {
	dir := writeTestConfig(t, fullConfigYAML())
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")
	t.Setenv("DATA_SERVICE_TOKEN", "service-token")
	t.Setenv("DATA_NEO4J_HEALTH_USERNAME", "health-reader")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "must be configured together") {
		t.Fatalf("Load() error = %v, want partial Neo4j health configuration rejection", err)
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
neo4j_health:
  uri: bolt://127.0.0.1:7687
  database: neo4j
  timeout_seconds: 1
migration:
  directory: migrations
  auto_apply: false
  lock_key: tidewise_schema_migration
`
}
