package conf

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLoadDefaultsToDevAndUsesFixedServicePort(t *testing.T) {
	setRequiredRuntimeEnvironment(t)
	dir := t.TempDir()
	writeConfig(t, dir, "dev", configYAML("dev", 9080, "disable"))
	t.Setenv("APP_ENV", "")
	t.Setenv("AGENTRUN_CONFIG_DIR", dir)
	t.Setenv("AGENTRUN_DB_PASSWORD", "dev-password")
	t.Setenv("AGENTRUN_SERVICE_TOKEN", "dev-token")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Name != ServiceName || cfg.App.Env != EnvDev {
		t.Fatalf("app = %#v", cfg.App)
	}
	if got := cfg.Server.Address(); got != "0.0.0.0:9080" {
		t.Fatalf("server address = %q, want 0.0.0.0:9080", got)
	}
	if cfg.Artifact.Root != "data" {
		t.Fatalf("artifact root = %q, want data", cfg.Artifact.Root)
	}
	if cfg.EventFact.ModelTimeoutSeconds != 180 {
		t.Fatalf("Event Fact model timeout = %d, want 180", cfg.EventFact.ModelTimeoutSeconds)
	}
	databaseURL, err := cfg.PostgresURL()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"postgres://agentrun:dev-password@localhost:5432/tidewise_ai_server", "connect_timeout=5", "sslmode=disable"} {
		if !strings.Contains(databaseURL, want) {
			t.Fatalf("PostgresURL() = %q, want %q", databaseURL, want)
		}
	}
}

func TestLoadRequiresServiceTokensAndDeploymentTimezone(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "dev", configYAML("dev", 9080, "disable"))
	t.Setenv("APP_ENV", "dev")
	t.Setenv("AGENTRUN_CONFIG_DIR", dir)
	t.Setenv("AGENTRUN_SERVICE_TOKEN", "")
	t.Setenv("DATA_SERVICE_TOKEN", "data-service-token")
	t.Setenv("TZ", "Asia/Shanghai")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AGENTRUN_SERVICE_TOKEN") {
		t.Fatalf("Load() error = %v, want missing AgentRun Service Token rejection", err)
	}

	t.Setenv("AGENTRUN_SERVICE_TOKEN", "service-token")
	t.Setenv("DATA_SERVICE_TOKEN", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DATA_SERVICE_TOKEN") {
		t.Fatalf("Load() error = %v, want missing Data Service Token rejection", err)
	}

	t.Setenv("DATA_SERVICE_TOKEN", "data-service-token")
	t.Setenv("TZ", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TZ") {
		t.Fatalf("Load() error = %v, want missing deployment timezone rejection", err)
	}

	t.Setenv("TZ", "Not/A-Timezone")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TZ") {
		t.Fatalf("Load() error = %v, want invalid deployment timezone rejection", err)
	}

	t.Setenv("TZ", "Local")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TZ") {
		t.Fatalf("Load() error = %v, want host-local timezone rejection", err)
	}

	t.Setenv("TZ", "Asia/Shanghai")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Location == nil || cfg.Location.String() != "Asia/Shanghai" {
		t.Fatalf("deployment location = %v, want Asia/Shanghai", cfg.Location)
	}
}

func TestLoadUATRequiresPasswordAndEncryptedYAMLDatabaseConfiguration(t *testing.T) {
	setRequiredRuntimeEnvironment(t)
	dir := t.TempDir()
	writeConfig(t, dir, "uat", configYAML("uat", 9080, "require"))
	t.Setenv("APP_ENV", "uat")
	t.Setenv("AGENTRUN_CONFIG_DIR", dir)

	t.Setenv("AGENTRUN_DB_PASSWORD", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AGENTRUN_DB_PASSWORD") {
		t.Fatalf("Load() error = %v, want missing database password rejection", err)
	}
	t.Setenv("AGENTRUN_DB_PASSWORD", "database-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() rejected valid UAT configuration: %v", err)
	}
	if cfg.App.Env != EnvUAT || cfg.Server.Port != 9080 {
		t.Fatalf("UAT configuration = %#v", cfg)
	}
	dsn, err := cfg.PostgresURL()
	if err != nil || !strings.Contains(dsn, "database-secret") || !strings.Contains(dsn, "sslmode=require") {
		t.Fatalf("PostgresURL() = %q, %v", dsn, err)
	}
}

func TestLoadDatabaseOperationRequiresOnlyDatabaseCredentials(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "uat", configYAML("uat", 9080, "require"))
	t.Setenv("APP_ENV", "uat")
	t.Setenv("AGENTRUN_CONFIG_DIR", dir)
	t.Setenv("AGENTRUN_DB_PASSWORD", "database-secret")
	t.Setenv("AGENTRUN_SERVICE_TOKEN", "")
	t.Setenv("DATA_SERVICE_TOKEN", "")
	t.Setenv("TZ", "")

	cfg, err := LoadDatabaseOperation()
	if err != nil {
		t.Fatalf("LoadDatabaseOperation() error = %v", err)
	}
	if cfg.App.Env != EnvUAT ||
		cfg.Secrets.DatabasePassword != "database-secret" ||
		cfg.Secrets.ServiceToken != "" ||
		cfg.Secrets.DataServiceToken != "" {
		t.Fatalf("database operation configuration = %#v", cfg)
	}

	t.Setenv("AGENTRUN_DB_PASSWORD", "")
	if _, err := LoadDatabaseOperation(); err == nil ||
		!strings.Contains(err.Error(), "AGENTRUN_DB_PASSWORD") {
		t.Fatalf(
			"LoadDatabaseOperation() error = %v, want missing database password",
			err,
		)
	}
}

func TestLoadRejectsNonStandardServicePort(t *testing.T) {
	setRequiredRuntimeEnvironment(t)
	dir := t.TempDir()
	writeConfig(t, dir, "dev", configYAML("dev", 8080, "disable"))
	t.Setenv("APP_ENV", "dev")
	t.Setenv("AGENTRUN_CONFIG_DIR", dir)

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "server.port must be 9080") {
		t.Fatalf("Load() error = %v, want fixed-port rejection", err)
	}
}

func TestLoadRejectsConfigurationIdentityMismatch(t *testing.T) {
	setRequiredRuntimeEnvironment(t)
	dir := t.TempDir()
	writeConfig(t, dir, "dev", strings.Replace(configYAML("uat", 9080, "disable"), "name: agentrun", "name: wrong-service", 1))
	t.Setenv("APP_ENV", "dev")
	t.Setenv("AGENTRUN_CONFIG_DIR", dir)

	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted a configuration whose app identity does not match the selected service and environment")
	}
}

func TestCheckedInEnvironmentConfigurations(t *testing.T) {
	setRequiredRuntimeEnvironment(t)
	configDir := filepath.Join("..", "..", "configs")
	t.Setenv("AGENTRUN_CONFIG_DIR", configDir)
	t.Setenv("AGENTRUN_DB_PASSWORD", "database-secret")
	t.Setenv("AGENTRUN_SERVICE_TOKEN", "service-token")

	for _, test := range []struct {
		environment Environment
	}{
		{environment: EnvDev},
		{environment: EnvUAT},
	} {
		t.Run(string(test.environment), func(t *testing.T) {
			t.Setenv("APP_ENV", string(test.environment))
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.App.Env != test.environment || cfg.Server.Port != ServicePort {
				t.Fatalf("configuration = %#v", cfg)
			}
		})
	}
}

func setRequiredRuntimeEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("AGENTRUN_SERVICE_TOKEN", "service-token")
	t.Setenv("DATA_SERVICE_TOKEN", "data-service-token")
	t.Setenv("TZ", "Asia/Shanghai")
}

func writeConfig(t *testing.T, dir, environment, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "config."+environment+".yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func configYAML(environment string, port int, sslMode string) string {
	return `app:
  name: agentrun
  env: ` + environment + `
server:
  host: 0.0.0.0
  port: ` + strconv.Itoa(port) + `
database:
  host: localhost
  port: 5432
  name: tidewise_ai_server
  user: agentrun
  ssl_mode: ` + sslMode + `
  connect_timeout_seconds: 5
artifact:
  root: data
data:
  base_url: http://127.0.0.1:9011
  timeout_seconds: 10
  max_response_bytes: 1048576
event_fact:
  reconcile_interval_seconds: 60
  model_timeout_seconds: 180
semantic_retrieval:
  qdrant_url: http://127.0.0.1:6333
  embedding_base_url: https://dashscope.aliyuncs.com/compatible-mode/v1
  embedding_model: text-embedding-v4
  entity_collection: entity_semantic_v1
  variable_collection: variable_definition_semantic_v1
  vector_size: 1024
  timeout_seconds: 10
  max_response_bytes: 8388608
`
}
