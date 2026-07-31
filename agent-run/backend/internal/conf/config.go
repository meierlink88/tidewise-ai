package conf

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"

	"gopkg.in/yaml.v3"
)

type Environment string

const (
	EnvDev Environment = "dev"
	EnvUAT Environment = "uat"
)

const (
	ServiceName = "agentrun"
	ServicePort = 9080
)

type Config struct {
	App       AppConfig       `yaml:"app"`
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Artifact  ArtifactConfig  `yaml:"artifact"`
	Data      DataConfig      `yaml:"data"`
	EventFact EventFactConfig `yaml:"event_fact"`
	Secrets   SecretConfig    `yaml:"-"`
	Location  *time.Location  `yaml:"-"`
}

type AppConfig struct {
	Name string      `yaml:"name"`
	Env  Environment `yaml:"env"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Host                  string `yaml:"host"`
	Port                  int    `yaml:"port"`
	Name                  string `yaml:"name"`
	User                  string `yaml:"user"`
	SSLMode               string `yaml:"ssl_mode"`
	ConnectTimeoutSeconds int    `yaml:"connect_timeout_seconds"`
}

type ArtifactConfig struct {
	Root string `yaml:"root"`
}

type DataConfig struct {
	BaseURL          string `yaml:"base_url"`
	TimeoutSeconds   int    `yaml:"timeout_seconds"`
	MaxResponseBytes int64  `yaml:"max_response_bytes"`
}

type EventFactConfig struct {
	ReconcileIntervalSeconds int `yaml:"reconcile_interval_seconds"`
	ModelTimeoutSeconds      int `yaml:"model_timeout_seconds"`
}

type SecretConfig struct {
	DatabasePassword string
	ServiceToken     string
	DataServiceToken string
}

func Load() (Config, error) {
	cfg, err := loadConfiguration(true)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.validateRuntimeSecrets(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LoadDatabaseOperation loads AgentRun-owned database configuration without
// requiring service identity tokens or the long-running process timezone.
func LoadDatabaseOperation() (Config, error) {
	cfg, err := loadConfiguration(false)
	if err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(cfg.Secrets.DatabasePassword) == "" {
		return Config{}, fmt.Errorf(
			"AGENTRUN_DB_PASSWORD is required for database operations",
		)
	}
	if cfg.App.Env == EnvUAT && cfg.Database.SSLMode != "require" {
		return Config{}, fmt.Errorf(
			"uat database configuration must use ssl_mode=require",
		)
	}
	cfg.Secrets.ServiceToken = ""
	cfg.Secrets.DataServiceToken = ""
	return cfg, nil
}

func loadConfiguration(requireTimezone bool) (Config, error) {
	environment, err := resolveEnvironment(os.Getenv("APP_ENV"))
	if err != nil {
		return Config{}, err
	}
	configDir := os.Getenv("AGENTRUN_CONFIG_DIR")
	if configDir == "" {
		configDir = "configs"
	}
	configPath := filepath.Join(configDir, fmt.Sprintf("config.%s.yaml", environment))
	content, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %s: %w", configPath, err)
	}
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %s: %w", configPath, err)
	}
	if cfg.App.Env != environment {
		return Config{}, fmt.Errorf("app.env must match selected APP_ENV %s", environment)
	}
	cfg.Secrets = SecretConfig{
		DatabasePassword: os.Getenv("AGENTRUN_DB_PASSWORD"),
		ServiceToken:     os.Getenv("AGENTRUN_SERVICE_TOKEN"),
		DataServiceToken: os.Getenv("DATA_SERVICE_TOKEN"),
	}
	if baseURL := strings.TrimSpace(os.Getenv("AGENTRUN_DATA_BASE_URL")); baseURL != "" {
		cfg.Data.BaseURL = baseURL
	}
	if requireTimezone {
		timezone := strings.TrimSpace(os.Getenv("TZ"))
		if timezone == "" || timezone == "Local" {
			return Config{}, fmt.Errorf("TZ is required")
		}
		cfg.Location, err = time.LoadLocation(timezone)
		if err != nil {
			return Config{}, fmt.Errorf("TZ must name a valid IANA timezone")
		}
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func resolveEnvironment(value string) (Environment, error) {
	if value == "" {
		return EnvDev, nil
	}
	environment := Environment(value)
	switch environment {
	case EnvDev, EnvUAT:
		return environment, nil
	default:
		return "", fmt.Errorf("unsupported APP_ENV %q", value)
	}
}

func (c Config) Validate() error {
	if c.App.Name != ServiceName {
		return fmt.Errorf("app.name must be %s", ServiceName)
	}
	if _, err := resolveEnvironment(string(c.App.Env)); err != nil {
		return err
	}
	if err := c.Server.Validate(); err != nil {
		return err
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("database.port must be between 1 and 65535")
	}
	if c.Database.Name != "tidewise_ai_server" {
		return fmt.Errorf("database.name must be tidewise_ai_server")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}
	if c.Database.SSLMode == "" {
		return fmt.Errorf("database.ssl_mode is required")
	}
	if c.Database.ConnectTimeoutSeconds <= 0 {
		return fmt.Errorf("database.connect_timeout_seconds must be positive")
	}
	if c.Artifact.Root == "" {
		return fmt.Errorf("artifact.root is required")
	}
	dataURL, err := url.Parse(c.Data.BaseURL)
	if err != nil || dataURL.Scheme == "" || dataURL.Host == "" ||
		(dataURL.Scheme != "http" && dataURL.Scheme != "https") ||
		(dataURL.Path != "" && dataURL.Path != "/") ||
		dataURL.User != nil || dataURL.RawQuery != "" || dataURL.Fragment != "" {
		return fmt.Errorf("data.base_url must be an absolute HTTP URL without credentials")
	}
	if c.Data.TimeoutSeconds <= 0 {
		return fmt.Errorf("data.timeout_seconds must be positive")
	}
	if c.Data.MaxResponseBytes <= 0 {
		return fmt.Errorf("data.max_response_bytes must be positive")
	}
	if c.EventFact.ReconcileIntervalSeconds <= 0 {
		return fmt.Errorf("event_fact.reconcile_interval_seconds must be positive")
	}
	if c.EventFact.ModelTimeoutSeconds <= 0 {
		return fmt.Errorf("event_fact.model_timeout_seconds must be positive")
	}
	return nil
}

func (c Config) validateRuntimeSecrets() error {
	if strings.TrimSpace(c.Secrets.ServiceToken) == "" {
		return fmt.Errorf("AGENTRUN_SERVICE_TOKEN is required")
	}
	if strings.TrimSpace(c.Secrets.DataServiceToken) == "" {
		return fmt.Errorf("DATA_SERVICE_TOKEN is required")
	}
	if c.App.Env != EnvUAT {
		return nil
	}
	if c.Secrets.DatabasePassword == "" {
		return fmt.Errorf("AGENTRUN_DB_PASSWORD is required in uat")
	}
	if c.Database.SSLMode != "require" {
		return fmt.Errorf("uat database configuration must use ssl_mode=require")
	}
	return nil
}

func (c Config) PostgresURL() (string, error) {
	values := url.Values{}
	values.Set("sslmode", c.Database.SSLMode)
	values.Set("connect_timeout", fmt.Sprintf("%d", c.Database.ConnectTimeoutSeconds))
	dsn := url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(c.Database.Host, fmt.Sprintf("%d", c.Database.Port)),
		Path:     c.Database.Name,
		RawQuery: values.Encode(),
	}
	if c.Secrets.DatabasePassword != "" {
		dsn.User = url.UserPassword(c.Database.User, c.Secrets.DatabasePassword)
	} else {
		dsn.User = url.User(c.Database.User)
	}
	return dsn.String(), nil
}

func (c ServerConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("server.host is required")
	}
	if c.Port != ServicePort {
		return fmt.Errorf("server.port must be 9080")
	}
	return nil
}

func (c ServerConfig) Address() string {
	return net.JoinHostPort(c.Host, fmt.Sprintf("%d", c.Port))
}
