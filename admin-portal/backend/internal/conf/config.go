package conf

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DataServiceTimeout = 5 * time.Second
const ServiceName = "adminportal"
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

// RuntimeConfig contains only Admin process, browser authentication, and
// downstream API settings. It cannot carry PostgreSQL or migration configuration.
type RuntimeConfig struct {
	App                      AppConfig
	Server                   ServerConfig
	AdminToken               string
	AllowedOrigin            string
	RawEvidencePublicBaseURL string
	DataService              DataServiceRuntimeConfig
}

func LoadRuntimeConfig() (RuntimeConfig, error) {
	environment, err := resolveEnvironment(os.Getenv("APP_ENV"))
	if err != nil {
		return RuntimeConfig{}, err
	}
	configPath := filepath.Join(resolveConfigDir(os.Getenv("TIDEWISE_CONFIG_DIR")), fmt.Sprintf("config.%s.yaml", environment))
	contents, err := os.ReadFile(configPath)
	if err != nil {
		return RuntimeConfig{}, fmt.Errorf("read Admin config file %s: %w", configPath, err)
	}
	var fileConfig struct {
		App    AppConfig    `yaml:"app"`
		Server ServerConfig `yaml:"server"`
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
		AdminToken:    strings.TrimSpace(os.Getenv("ADMIN_SERVICE_TOKEN")),
		AllowedOrigin: strings.TrimSpace(os.Getenv("ADMIN_ALLOWED_ORIGIN")),
		RawEvidencePublicBaseURL: strings.TrimSpace(
			os.Getenv("RAW_EVIDENCE_PUBLIC_BASE_URL"),
		),
		DataService: DataServiceRuntimeConfig{
			BaseURL:       strings.TrimSpace(os.Getenv("DATA_SERVICE_BASE_URL")),
			IdentityToken: strings.TrimSpace(os.Getenv("DATA_SERVICE_TOKEN")),
			Timeout:       DataServiceTimeout,
		},
	}
	if runtime.AdminToken == "" {
		return RuntimeConfig{}, fmt.Errorf("ADMIN_SERVICE_TOKEN is required")
	}
	if err := validateAllowedOrigin(runtime.AllowedOrigin); err != nil {
		return RuntimeConfig{}, err
	}
	if err := validateRawEvidencePublicBaseURL(runtime.RawEvidencePublicBaseURL); err != nil {
		return RuntimeConfig{}, err
	}
	if runtime.DataService.BaseURL == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_BASE_URL is required")
	}
	if runtime.DataService.IdentityToken == "" {
		return RuntimeConfig{}, fmt.Errorf("DATA_SERVICE_TOKEN is required")
	}
	return runtime, nil
}

func validateRawEvidencePublicBaseURL(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("RAW_EVIDENCE_PUBLIC_BASE_URL must be an http(s) origin without credentials, path, query, or fragment")
	}
	return nil
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
