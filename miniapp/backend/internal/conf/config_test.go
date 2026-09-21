package conf

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
	t.Setenv("DATA_SERVICE_TOKEN", "data-service-token")
	t.Setenv("DATABASE_PASSWORD", "must-not-be-loaded")
	t.Setenv("TIDEWISE_DATABASE_URL", "postgres://must-not-be-loaded")

	runtime, err := LoadRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.App.Name != ServiceName || runtime.DataService.BaseURL != "http://data.internal:8081" || runtime.DataService.IdentityToken != "data-service-token" || runtime.DataService.Timeout != 5*time.Second {
		t.Fatalf("runtime = %#v", runtime)
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
	t.Setenv("DATA_SERVICE_TOKEN", "")

	if _, err := LoadRuntimeConfig(); err == nil {
		t.Fatal("LoadRuntimeConfig() error = nil without Data Service Token")
	}
}

func TestOptionalUserServiceSettingsArePaired(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.local.yaml"), []byte("server:\n  host: 127.0.0.1\n  port: 9012\n  read_timeout_seconds: 5\n  write_timeout_seconds: 10\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "local")
	t.Setenv("TIDEWISE_CONFIG_DIR", dir)
	t.Setenv("DATA_SERVICE_BASE_URL", "http://data:9011")
	t.Setenv("DATA_SERVICE_TOKEN", "data-token")
	for _, tc := range []struct {
		url, token string
		valid      bool
	}{{"", "", true}, {"http://user:9015", "", false}, {"", "user-token", false}, {"http://user:9015", "user-token", true}} {
		t.Setenv("USER_SERVICE_BASE_URL", tc.url)
		t.Setenv("USER_SERVICE_TOKEN", tc.token)
		_, err := LoadRuntimeConfig()
		if (err == nil) != tc.valid {
			t.Fatal("User Service pairing mismatch")
		}
	}
}
