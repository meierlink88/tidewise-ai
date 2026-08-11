package conf

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DataServiceTimeout = 5 * time.Second
const ServiceName = "miniapp"
const ServiceVersion = "1.0.0"

type Environment string

const (
	EnvLocal Environment = "local"
	EnvUAT   Environment = "uat"
	EnvProd  Environment = "prod"
)

type AppConfig struct {
	Name string      `yaml:"name"`
	Env  Environment `yaml:"env"`
}

type ServerConfig struct {
	Host                string `yaml:"host"`
	Port                int    `yaml:"port"`
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"`
}

type DataServiceRuntimeConfig struct {
	BaseURL       string
	IdentityToken string
	Timeout       time.Duration
}

// RuntimeConfig contains only the Miniapp process and Data API settings. It
// intentionally cannot carry PostgreSQL connection or migration settings.
type RuntimeConfig struct {
	App         AppConfig
	Server      ServerConfig
	DataService DataServiceRuntimeConfig
}

func LoadRuntimeConfig() (RuntimeConfig, error) {
	environment, err := resolveEnvironment(os.Getenv("APP_ENV"))
	if err != nil {
		return RuntimeConfig{}, err
	}
	configPath := filepath.Join(resolveConfigDir(os.Getenv("TIDEWISE_CONFIG_DIR")), fmt.Sprintf("config.%s.yaml", environment))
	contents, err := os.ReadFile(configPath)
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("read Miniapp config file %s: %w", configPath, err)
	}
	var fileConfig struct {
		App    AppConfig    `yaml:"app"`
		Server ServerConfig `yaml:"server"`
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
		IdentityToken: strings.TrimSpace(os.Getenv("DATA_SERVICE_TOKEN")),
		Timeout:       DataServiceTimeout,
	}
	if dataService.BaseURL == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_BASE_URL is required")
	}
	if dataService.IdentityToken == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_TOKEN is required")
	}
	return RuntimeConfig{App: fileConfig.App, Server: fileConfig.Server, DataService: dataService}, nil
}

func resolveEnvironment(value string) (Environment, error) {
	if value == "" {
		return EnvLocal, nil
	}
	environment := Environment(value)
	switch environment {
	case EnvLocal, EnvUAT, EnvProd:
		return environment, nil
	default:
		return "", fmt.Errorf("unsupported APP_ENV %q", value)
	}
}

func resolveConfigDir(explicit string) string {
	if explicit != "" {
		return explicit
	}
	return "/app/configs"
}

func (c ServerConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("server.host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.ReadTimeoutSeconds <= 0 {
		return fmt.Errorf("server.read_timeout_seconds must be positive")
	}
	if c.WriteTimeoutSeconds <= 0 {
		return fmt.Errorf("server.write_timeout_seconds must be positive")
	}
	return nil
}

func (c ServerConfig) Address() string {
	return net.JoinHostPort(c.Host, fmt.Sprintf("%d", c.Port))
}

func errorsForInvalidServerConfig() error {
	return fmt.Errorf("Miniapp server config requires a valid host, port, and positive read/write timeouts")
}
