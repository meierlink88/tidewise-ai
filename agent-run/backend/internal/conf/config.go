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
	App      AppConfig      `yaml:"app"`
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Artifact ArtifactConfig `yaml:"artifact"`
	Secrets  SecretConfig   `yaml:"-"`
	Location *time.Location `yaml:"-"`
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

type SecretConfig struct {
	DatabaseURL      string
	DatabasePassword string
	ServiceToken     string
	AdminToken       string
}

func Load() (Config, error) {
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
		DatabaseURL:      os.Getenv("AGENTRUN_DATABASE_URL"),
		DatabasePassword: os.Getenv("AGENTRUN_DATABASE_PASSWORD"),
		ServiceToken:     os.Getenv("AGENTRUN_SERVICE_TOKEN"),
		AdminToken:       os.Getenv("AGENTRUN_ADMIN_TOKEN"),
	}
	timezone := strings.TrimSpace(os.Getenv("TZ"))
	if timezone == "" || timezone == "Local" {
		return Config{}, fmt.Errorf("TZ is required")
	}
	cfg.Location, err = time.LoadLocation(timezone)
	if err != nil {
		return Config{}, fmt.Errorf("TZ must name a valid IANA timezone")
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	if err := cfg.validateRuntimeSecrets(); err != nil {
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
	return nil
}

func (c Config) validateRuntimeSecrets() error {
	if strings.TrimSpace(c.Secrets.AdminToken) == "" {
		return fmt.Errorf("AGENTRUN_ADMIN_TOKEN is required")
	}
	if c.App.Env != EnvUAT {
		return nil
	}
	if c.Secrets.DatabaseURL == "" {
		return fmt.Errorf("AGENTRUN_DATABASE_URL is required in uat")
	}
	parsed, err := parsePostgresURL(c.Secrets.DatabaseURL)
	if err != nil {
		return fmt.Errorf("uat database url must be a complete postgres URL")
	}
	if parsed.Query().Get("sslmode") != "require" {
		return fmt.Errorf("uat database url must use sslmode=require")
	}
	return nil
}

func (c Config) PostgresURL() (string, error) {
	if c.Secrets.DatabaseURL != "" {
		if _, err := parsePostgresURL(c.Secrets.DatabaseURL); err != nil {
			return "", fmt.Errorf("database url must be a complete postgres URL")
		}
		return c.Secrets.DatabaseURL, nil
	}
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

func parsePostgresURL(value string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme != "postgres" || parsed.Hostname() == "" || parsed.User == nil || parsed.Path == "" || parsed.Path == "/" {
		return nil, fmt.Errorf("invalid postgres URL")
	}
	return parsed, nil
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
