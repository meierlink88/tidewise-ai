package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"gopkg.in/yaml.v3"
)

const DataServiceTimeout = 5 * time.Second
const ServiceName = "adminportal"

type DataServiceRuntimeConfig struct {
	BaseURL       string
	IdentityToken string
	Timeout       time.Duration
}

// RuntimeConfig contains only Admin process, browser authentication, and Data
// API settings. It cannot carry PostgreSQL or migration configuration.
type RuntimeConfig struct {
	App           runtimeconfig.AppConfig
	Server        runtimeconfig.ServerConfig
	AdminToken    string
	AllowedOrigin string
	DataService   DataServiceRuntimeConfig
}

func LoadRuntimeConfig() (RuntimeConfig, error) {
	environment, err := runtimeconfig.ResolveEnvironment(os.Getenv("APP_ENV"))
	if err != nil {
		return RuntimeConfig{}, err
	}
	configPath := filepath.Join(runtimeconfig.ResolveConfigDir(os.Getenv("TIDEWISE_CONFIG_DIR"), "services/adminportal/config"), fmt.Sprintf("config.%s.yaml", environment))
	contents, err := os.ReadFile(configPath)
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("read Admin config file %s: %w", configPath, err)
	}
	var fileConfig struct {
		App    runtimeconfig.AppConfig    `yaml:"app"`
		Server runtimeconfig.ServerConfig `yaml:"server"`
	}
	if err := yaml.Unmarshal(contents, &fileConfig); err != nil {
		return RuntimeConfig{}, fmt.Errorf("parse Admin config file %s: %w", configPath, err)
	}
	fileConfig.App.Name = ServiceName
	fileConfig.App.Env = environment
	if err := fileConfig.Server.Validate(); err != nil {
		return RuntimeConfig{}, fmt.Errorf("Admin server config requires a valid host, port, and positive read/write timeouts")
	}

	runtime := RuntimeConfig{
		App:           fileConfig.App,
		Server:        fileConfig.Server,
		AdminToken:    strings.TrimSpace(os.Getenv("ADMIN_API_TOKEN")),
		AllowedOrigin: strings.TrimSpace(os.Getenv("ADMIN_ALLOWED_ORIGIN")),
		DataService: DataServiceRuntimeConfig{
			BaseURL:       strings.TrimSpace(os.Getenv("DATA_SERVICE_BASE_URL")),
			IdentityToken: strings.TrimSpace(os.Getenv("DATA_SERVICE_ADMIN_TOKEN")),
			Timeout:       DataServiceTimeout,
		},
	}
	if runtime.AdminToken == "" {
		return RuntimeConfig{}, fmt.Errorf("ADMIN_API_TOKEN is required")
	}
	if err := validateAllowedOrigin(runtime.AllowedOrigin); err != nil {
		return RuntimeConfig{}, err
	}
	if runtime.DataService.BaseURL == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_BASE_URL is required")
	}
	if runtime.DataService.IdentityToken == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_ADMIN_TOKEN is required")
	}
	return runtime, nil
}

func validateAllowedOrigin(value string) error {
	if value == "" {
		return fmt.Errorf("ADMIN_ALLOWED_ORIGIN is required")
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("ADMIN_ALLOWED_ORIGIN must be an http(s) origin without path, query, or fragment")
	}
	return nil
}
