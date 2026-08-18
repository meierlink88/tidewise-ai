package conf

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadRuntimeConfigRequiresAdminAndDataServiceSettings(t *testing.T) {
	configDir := writeRuntimeConfig(t)
	t.Setenv("APP_ENV", "local")
	t.Setenv("TIDEWISE_CONFIG_DIR", configDir)
	t.Setenv("DATA_SERVICE_BASE_URL", "http://data.internal:8081")
	t.Setenv("DATA_SERVICE_TOKEN", "data-service-token")
	t.Setenv("ADMIN_SERVICE_TOKEN", "admin-service-token")
	t.Setenv("ADMIN_ALLOWED_ORIGIN", "http://127.0.0.1:5174")
	t.Setenv("TIDEWISW_DB_PASSWORD", "must-not-be-loaded")

	runtime, err := LoadRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.App.Name != ServiceName ||
		runtime.DataService.BaseURL != "http://data.internal:8081" ||
		runtime.DataService.IdentityToken != "data-service-token" ||
		runtime.AdminToken != "admin-service-token" ||
		runtime.AllowedOrigin != "http://127.0.0.1:5174" ||
		runtime.DataService.Timeout != 5*time.Second {
		t.Fatalf("runtime = %#v", runtime)
	}
}

func TestLoadRuntimeConfigFailsClosedWithoutRequiredServiceIdentity(t *testing.T) {
	for _, missing := range []string{
		"DATA_SERVICE_BASE_URL",
		"DATA_SERVICE_TOKEN",
		"ADMIN_SERVICE_TOKEN",
		"ADMIN_ALLOWED_ORIGIN",
	} {
		t.Run(missing, func(t *testing.T) {
			configDir := writeRuntimeConfig(t)
			t.Setenv("APP_ENV", "local")
			t.Setenv("TIDEWISE_CONFIG_DIR", configDir)
			t.Setenv("DATA_SERVICE_BASE_URL", "http://data.internal:8081")
			t.Setenv("DATA_SERVICE_TOKEN", "data-service-token")
			t.Setenv("ADMIN_SERVICE_TOKEN", "admin-service-token")
			t.Setenv("ADMIN_ALLOWED_ORIGIN", "http://127.0.0.1:5174")
			t.Setenv(missing, "")
			if _, err := LoadRuntimeConfig(); err == nil {
				t.Fatalf("LoadRuntimeConfig() error = nil without %s", missing)
			}
		})
	}
}

func TestLoadRuntimeConfigRejectsWildcardOrPathOrigin(t *testing.T) {
	for _, origin := range []string{"*", "http://uat.example.test/path"} {
		t.Run(origin, func(t *testing.T) {
			t.Setenv("APP_ENV", "local")
			t.Setenv("TIDEWISE_CONFIG_DIR", writeRuntimeConfig(t))
			t.Setenv("DATA_SERVICE_BASE_URL", "http://data.internal:9011")
			t.Setenv("DATA_SERVICE_TOKEN", "data-service-token")
			t.Setenv("ADMIN_SERVICE_TOKEN", "admin-service-token")
			t.Setenv("ADMIN_ALLOWED_ORIGIN", origin)
			if _, err := LoadRuntimeConfig(); err == nil {
				t.Fatalf("LoadRuntimeConfig() accepted origin %q", origin)
			}
		})
	}
}

func writeRuntimeConfig(t *testing.T) string {
	t.Helper()
	configDir := t.TempDir()
	configBody := []byte("app:\n  name: ignored-shared-name\nserver:\n  host: 127.0.0.1\n  port: 18083\n  read_timeout_seconds: 5\n  write_timeout_seconds: 10\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.local.yaml"), configBody, 0o600); err != nil {
		t.Fatal(err)
	}
	return configDir
}
