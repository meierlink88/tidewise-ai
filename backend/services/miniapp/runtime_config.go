package miniapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"gopkg.in/yaml.v3"
)

const DataServiceTimeout = 5 * time.Second

type DataServiceRuntimeConfig struct {
	BaseURL       string
	IdentityToken string
	Timeout       time.Duration
}

// RuntimeConfig contains only the Miniapp process and Data API settings. It
// intentionally cannot carry PostgreSQL connection or migration settings.
type RuntimeConfig struct {
	App         config.AppConfig
	Server      config.ServerConfig
	DataService DataServiceRuntimeConfig
}

func LoadRuntimeConfig() (RuntimeConfig, error) {
	environment, err := config.ResolveEnvironment(os.Getenv("APP_ENV"))
	if err != nil {
		return RuntimeConfig{}, err
	}
	configPath := filepath.Join(config.ResolveConfigDir(os.Getenv("TIDEWISE_CONFIG_DIR")), fmt.Sprintf("config.%s.yaml", environment))
	contents, err := os.ReadFile(configPath)
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("read Miniapp config file %s: %w", configPath, err)
	}
	var fileConfig struct {
		App    config.AppConfig    `yaml:"app"`
		Server config.ServerConfig `yaml:"server"`
	}
	if err := yaml.Unmarshal(contents, &fileConfig); err != nil {
		return RuntimeConfig{}, fmt.Errorf("parse Miniapp config file %s: %w", configPath, err)
	}
	fileConfig.App.Name = ServiceName
	fileConfig.App.Env = environment
	if fileConfig.Server.Host == "" || fileConfig.Server.Port <= 0 || fileConfig.Server.Port > 65535 || fileConfig.Server.ReadTimeoutSeconds <= 0 || fileConfig.Server.WriteTimeoutSeconds <= 0 {
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

func (c RuntimeConfig) ServiceConfig() config.Config {
	return config.Config{App: c.App, Server: c.Server}
}
