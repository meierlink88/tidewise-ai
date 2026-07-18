package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"gopkg.in/yaml.v3"
)

const DataServiceTimeout = 5 * time.Second
const ServiceName = "miniapp"

type DataServiceRuntimeConfig struct {
	BaseURL       string
	IdentityToken string
	Timeout       time.Duration
}

// RuntimeConfig contains only the Miniapp process and Data API settings. It
// intentionally cannot carry PostgreSQL connection or migration settings.
type RuntimeConfig struct {
	App         runtimeconfig.AppConfig
	Server      runtimeconfig.ServerConfig
	DataService DataServiceRuntimeConfig
}

func LoadRuntimeConfig() (RuntimeConfig, error) {
	environment, err := runtimeconfig.ResolveEnvironment(os.Getenv("APP_ENV"))
	if err != nil {
		return RuntimeConfig{}, err
	}
	configPath := filepath.Join(runtimeconfig.ResolveConfigDir(os.Getenv("TIDEWISE_CONFIG_DIR"), "services/miniapp/config"), fmt.Sprintf("config.%s.yaml", environment))
	contents, err := os.ReadFile(configPath)
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("read Miniapp config file %s: %w", configPath, err)
	}
	var fileConfig struct {
		App    runtimeconfig.AppConfig    `yaml:"app"`
		Server runtimeconfig.ServerConfig `yaml:"server"`
	}
	if err := yaml.Unmarshal(contents, &fileConfig); err != nil {
		return RuntimeConfig{}, fmt.Errorf("parse Miniapp config file %s: %w", configPath, err)
	}
	fileConfig.App.Name = ServiceName
	fileConfig.App.Env = environment
	if err := fileConfig.Server.Validate(); err != nil {
		return RuntimeConfig{}, errorsForInvalidServerConfig()
	}
	dataService := DataServiceRuntimeConfig{
		BaseURL:       strings.TrimSpace(os.Getenv("DATA_SERVICE_BASE_URL")),
		IdentityToken: strings.TrimSpace(os.Getenv("DATA_SERVICE_MINIAPP_TOKEN")),
		Timeout:       DataServiceTimeout,
	}
	if dataService.BaseURL == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_BASE_URL is required")
	}
	if dataService.IdentityToken == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_MINIAPP_TOKEN is required")
	}
	return RuntimeConfig{App: fileConfig.App, Server: fileConfig.Server, DataService: dataService}, nil
}

func errorsForInvalidServerConfig() error {
	return fmt.Errorf("Miniapp server config requires a valid host, port, and positive read/write timeouts")
}
