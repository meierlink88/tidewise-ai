package config

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
	t.Setenv("AGENTRUN_DATABASE_URL", "")
	t.Setenv("AGENTRUN_DATABASE_PASSWORD", "dev-password")
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

func TestLoadRequiresAdminTokenAndDeploymentTimezone(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "dev", configYAML("dev", 9080, "disable"))
	t.Setenv("APP_ENV", "dev")
	t.Setenv("AGENTRUN_CONFIG_DIR", dir)
	t.Setenv("AGENTRUN_SERVICE_TOKEN", "service-token")
	t.Setenv("AGENTRUN_ADMIN_TOKEN", "")
	t.Setenv("TZ", "Asia/Shanghai")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AGENTRUN_ADMIN_TOKEN") {
		t.Fatalf("Load() error = %v, want missing Admin Token rejection", err)
	}

	t.Setenv("AGENTRUN_ADMIN_TOKEN", "admin-token")
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

func TestLoadUATRequiresEncryptedDatabaseURL(t *testing.T) {
	setRequiredRuntimeEnvironment(t)
	dir := t.TempDir()
	writeConfig(t, dir, "uat", configYAML("uat", 9080, "require"))
	t.Setenv("APP_ENV", "uat")
	t.Setenv("AGENTRUN_CONFIG_DIR", dir)

	for _, databaseURL := range []string{
		"",
		"postgres://agentrun:secret@rds.internal:5432/tidewise_ai_server?sslmode=disable",
	} {
		t.Setenv("AGENTRUN_DATABASE_URL", databaseURL)
		if _, err := Load(); err == nil {
			t.Fatalf("Load() accepted unsafe UAT database URL %q", databaseURL)
		}
	}

	t.Setenv("AGENTRUN_DATABASE_URL", "postgres://agentrun:secret@rds.internal:5432/tidewise_ai_server?sslmode=require")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() rejected valid UAT configuration: %v", err)
	}
	if cfg.App.Env != EnvUAT || cfg.Server.Port != 9080 {
		t.Fatalf("UAT configuration = %#v", cfg)
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
	configDir := "."
	t.Setenv("AGENTRUN_CONFIG_DIR", configDir)
	t.Setenv("AGENTRUN_DATABASE_PASSWORD", "dev-password")
	t.Setenv("AGENTRUN_SERVICE_TOKEN", "service-token")

	for _, test := range []struct {
		environment Environment
		databaseURL string
	}{
		{environment: EnvDev},
		{environment: EnvUAT, databaseURL: "postgres://agentrun:secret@rds.internal:5432/tidewise_ai_server?sslmode=require"},
	} {
		t.Run(string(test.environment), func(t *testing.T) {
			t.Setenv("APP_ENV", string(test.environment))
			t.Setenv("AGENTRUN_DATABASE_URL", test.databaseURL)
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
	t.Setenv("AGENTRUN_ADMIN_TOKEN", "admin-token")
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
`
}
