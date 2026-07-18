package miniapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadRuntimeConfigRequiresOnlyBFFAndDataServiceSettings(t *testing.T) {
	configDir := t.TempDir()
	configBody := []byte("app:\n  name: ignored-shared-name\nserver:\n  host: 127.0.0.1\n  port: 18082\n  read_timeout_seconds: 5\n  write_timeout_seconds: 10\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.local.yaml"), configBody, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "local")
	t.Setenv("TIDEWISE_CONFIG_DIR", configDir)
	t.Setenv("DATA_SERVICE_BASE_URL", "http://data.internal:8081")
	t.Setenv("DATA_SERVICE_MINIAPP_TOKEN", "miniapp-identity")
	t.Setenv("DATABASE_PASSWORD", "must-not-be-loaded")
	t.Setenv("TIDEWISE_DATABASE_URL", "postgres://must-not-be-loaded")

	runtime, err := LoadRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.App.Name != ServiceName || runtime.DataService.BaseURL != "http://data.internal:8081" || runtime.DataService.IdentityToken != "miniapp-identity" || runtime.DataService.Timeout != 5*time.Second {
		t.Fatalf("runtime = %#v", runtime)
	}
	serviceConfig := runtime.ServiceConfig()
	if serviceConfig.Database.Host != "" || serviceConfig.Secrets.DatabaseURL != "" || serviceConfig.Secrets.DatabasePassword != "" {
		t.Fatalf("Miniapp service config carries Data DB credential: %#v/%#v", serviceConfig.Database, serviceConfig.Secrets)
	}
}

func TestLoadRuntimeConfigFailsClosedWithoutDataServiceIdentity(t *testing.T) {
	configDir := t.TempDir()
	configBody := []byte("app:\n  name: ignored\nserver:\n  host: 127.0.0.1\n  port: 18082\n  read_timeout_seconds: 5\n  write_timeout_seconds: 10\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.local.yaml"), configBody, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "local")
	t.Setenv("TIDEWISE_CONFIG_DIR", configDir)
	t.Setenv("DATA_SERVICE_BASE_URL", "http://data.internal:8081")
	t.Setenv("DATA_SERVICE_MINIAPP_TOKEN", "")

	if _, err := LoadRuntimeConfig(); err == nil {
		t.Fatal("LoadRuntimeConfig() error = nil without Miniapp Data Service identity")
	}
}
