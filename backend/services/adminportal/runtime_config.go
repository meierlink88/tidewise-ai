package adminportal

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

// RuntimeConfig contains only Admin process, browser authentication, and Data
// API settings. It cannot carry PostgreSQL or migration configuration.
type RuntimeConfig struct {
	App         config.AppConfig
	Server      config.ServerConfig
	AdminToken  string
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
		return RuntimeConfig{}, fmt.Errorf("read Admin config file %s: %w", configPath, err)
	}
	var fileConfig struct {
		App    config.AppConfig    `yaml:"app"`
		Server config.ServerConfig `yaml:"server"`
	}
	if err := yaml.Unmarshal(contents, &fileConfig); err != nil {
		return RuntimeConfig{}, fmt.Errorf("parse Admin config file %s: %w", configPath, err)
	}
	fileConfig.App.Name = ServiceName
	fileConfig.App.Env = environment
	if fileConfig.Server.Host == "" || fileConfig.Server.Port <= 0 || fileConfig.Server.Port > 65535 || fileConfig.Server.ReadTimeoutSeconds <= 0 || fileConfig.Server.WriteTimeoutSeconds <= 0 {
		return RuntimeConfig{}, fmt.Errorf("Admin server config requires a valid host, port, and positive read/write timeouts")
	}

	runtime := RuntimeConfig{
		App:        fileConfig.App,
		Server:     fileConfig.Server,
		AdminToken: strings.TrimSpace(os.Getenv("ADMIN_API_TOKEN")),
		DataService: DataServiceRuntimeConfig{
			BaseURL:       strings.TrimSpace(os.Getenv("DATA_SERVICE_BASE_URL")),
			IdentityToken: strings.TrimSpace(os.Getenv("DATA_SERVICE_ADMIN_TOKEN")),
			Timeout:       DataServiceTimeout,
		},
	}
	if runtime.AdminToken == "" {
		return RuntimeConfig{}, fmt.Errorf("ADMIN_API_TOKEN is required")
	}
	if runtime.DataService.BaseURL == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_BASE_URL is required")
	}
	if runtime.DataService.IdentityToken == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_ADMIN_TOKEN is required")
	}
	return runtime, nil
}

func (c RuntimeConfig) ServiceConfig() config.Config {
	return config.Config{App: c.App, Server: c.Server}
}
