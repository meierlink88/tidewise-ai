package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"gopkg.in/yaml.v3"
)

type Environment = runtimeconfig.Environment
type AppConfig = runtimeconfig.AppConfig
type ServerConfig = runtimeconfig.ServerConfig

const (
	EnvLocal = runtimeconfig.EnvLocal
	EnvUAT   = runtimeconfig.EnvUAT
	EnvProd  = runtimeconfig.EnvProd
)

const ServiceName = "data"

type Config struct {
	App       AppConfig       `yaml:"app"`
	Server    ServerConfig    `yaml:"server"`
	Log       LogConfig       `yaml:"log"`
	Database  DatabaseConfig  `yaml:"database"`
	Neo4j     Neo4jConfig     `yaml:"neo4j"`
	Migration MigrationConfig `yaml:"migration"`
	Secrets   SecretConfig    `yaml:"-"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type DatabaseConfig struct {
	Host                   string `yaml:"host"`
	Port                   int    `yaml:"port"`
	Name                   string `yaml:"name"`
	User                   string `yaml:"user"`
	SSLMode                string `yaml:"ssl_mode"`
	MaxOpenConns           int    `yaml:"max_open_conns"`
	MaxIdleConns           int    `yaml:"max_idle_conns"`
	ConnMaxLifetimeSeconds int    `yaml:"conn_max_lifetime_seconds"`
	ConnectTimeoutSeconds  int    `yaml:"connect_timeout_seconds"`
}

type Neo4jConfig struct {
	Enabled               bool   `yaml:"enabled"`
	URI                   string `yaml:"uri"`
	Database              string `yaml:"database"`
	UsernameEnv           string `yaml:"username_env"`
	PasswordEnv           string `yaml:"password_env"`
	ConnectTimeoutSeconds int    `yaml:"connect_timeout_seconds"`
	MaxConnectionPoolSize int    `yaml:"max_connection_pool_size"`
}

type MigrationConfig struct {
	Directory string `yaml:"directory"`
	AutoApply bool   `yaml:"auto_apply"`
	LockKey   string `yaml:"lock_key"`
}

type SecretConfig struct {
	DatabaseURL                       string
	DatabasePassword                  string
	DataServiceAgentToken             string
	DataServiceResearchPublisherToken string
	DataServiceMiniappToken           string
	DataServiceAdminToken             string
}

func Load() (Config, error) {
	env, err := runtimeconfig.ResolveEnvironment(os.Getenv("APP_ENV"))
	if err != nil {
		return Config{}, err
	}

	configDir := runtimeconfig.ResolveConfigDir(os.Getenv("TIDEWISE_CONFIG_DIR"), "services/data/config")
	configPath := filepath.Join(configDir, fmt.Sprintf("config.%s.yaml", env))

	content, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %s: %w", configPath, err)
	}

	cfg.App.Name = ServiceName
	cfg.App.Env = env
	cfg.Secrets = SecretConfig{
		DatabaseURL:                       firstEnv("TIDEWISE_DATABASE_URL", "DATABASE_URL"),
		DatabasePassword:                  os.Getenv("DATABASE_PASSWORD"),
		DataServiceAgentToken:             os.Getenv("DATA_SERVICE_AGENT_TOKEN"),
		DataServiceResearchPublisherToken: os.Getenv("DATA_SERVICE_RESEARCH_PUBLISHER_TOKEN"),
		DataServiceMiniappToken:           os.Getenv("DATA_SERVICE_MINIAPP_TOKEN"),
		DataServiceAdminToken:             os.Getenv("DATA_SERVICE_ADMIN_TOKEN"),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	if err := cfg.validateRuntimeSecrets(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validateRuntimeSecrets() error {
	if c.App.Env != EnvUAT {
		return nil
	}
	if c.Secrets.DatabaseURL == "" {
		return fmt.Errorf("TIDEWISE_DATABASE_URL is required in uat")
	}
	parsed, err := url.ParseRequestURI(c.Secrets.DatabaseURL)
	if err != nil || parsed.Scheme != "postgres" || parsed.Hostname() == "" || parsed.User == nil || parsed.Path == "" || parsed.Path == "/" {
		return fmt.Errorf("uat database url must be a complete postgres URL")
	}
	query := parsed.Query()
	if query.Get("sslmode") != "require" {
		return fmt.Errorf("uat database url must use sslmode=require")
	}
	return nil
}

func (c Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if _, err := runtimeconfig.ResolveEnvironment(string(c.App.Env)); err != nil {
		return err
	}
	if err := c.Server.Validate(); err != nil {
		return err
	}
	if c.Log.Level == "" {
		return fmt.Errorf("log.level is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("database.port must be between 1 and 65535")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database.name is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}
	if c.Database.SSLMode == "" {
		return fmt.Errorf("database.ssl_mode is required")
	}
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database.max_open_conns must be positive")
	}
	if c.Database.MaxIdleConns <= 0 {
		return fmt.Errorf("database.max_idle_conns must be positive")
	}
	if c.Database.ConnMaxLifetimeSeconds <= 0 {
		return fmt.Errorf("database.conn_max_lifetime_seconds must be positive")
	}
	if c.Database.ConnectTimeoutSeconds <= 0 {
		return fmt.Errorf("database.connect_timeout_seconds must be positive")
	}
	if c.Neo4j.Enabled {
		if c.Neo4j.URI == "" {
			return fmt.Errorf("neo4j.uri is required when neo4j is enabled")
		}
		parsed, err := url.ParseRequestURI(c.Neo4j.URI)
		if err != nil || (parsed.Scheme != "bolt" && parsed.Scheme != "neo4j" && parsed.Scheme != "neo4j+s" && parsed.Scheme != "bolt+s") {
			return fmt.Errorf("neo4j.uri must be a valid Neo4j URI")
		}
		if c.Neo4j.Database == "" {
			return fmt.Errorf("neo4j.database is required when neo4j is enabled")
		}
		if c.Neo4j.UsernameEnv == "" {
			return fmt.Errorf("neo4j.username_env is required when neo4j is enabled")
		}
		if c.Neo4j.PasswordEnv == "" {
			return fmt.Errorf("neo4j.password_env is required when neo4j is enabled")
		}
		if c.Neo4j.ConnectTimeoutSeconds <= 0 {
			return fmt.Errorf("neo4j.connect_timeout_seconds must be positive when neo4j is enabled")
		}
		if c.Neo4j.MaxConnectionPoolSize <= 0 {
			return fmt.Errorf("neo4j.max_connection_pool_size must be positive when neo4j is enabled")
		}
	}
	if c.Migration.Directory == "" {
		return fmt.Errorf("migration.directory is required")
	}
	if c.Migration.LockKey == "" {
		return fmt.Errorf("migration.lock_key is required")
	}
	return nil
}

func (c Config) PostgresURL() (string, error) {
	if c.Secrets.DatabaseURL != "" {
		if _, err := url.ParseRequestURI(c.Secrets.DatabaseURL); err != nil {
			return "", fmt.Errorf("database url must be a valid URL: %w", err)
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

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
