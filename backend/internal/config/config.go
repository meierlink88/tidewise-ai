package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Environment string

const (
	EnvLocal Environment = "local"
	EnvUAT   Environment = "uat"
	EnvProd  Environment = "prod"
)

type Config struct {
	App           AppConfig           `yaml:"app"`
	Server        ServerConfig        `yaml:"server"`
	Log           LogConfig           `yaml:"log"`
	AgentPlatform AgentPlatformConfig `yaml:"agent_platform"`
	Database      DatabaseConfig      `yaml:"database"`
	Redis         RedisConfig         `yaml:"redis"`
	Migration     MigrationConfig     `yaml:"migration"`
	Ingestion     IngestionConfig     `yaml:"ingestion"`
	ObjectStore   ObjectStoreConfig   `yaml:"object_store"`
	RateLimit     RateLimitConfig     `yaml:"rate_limit"`
	Security      SecurityConfig      `yaml:"security"`
	Secrets       SecretConfig        `yaml:"-"`
}

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

type LogConfig struct {
	Level string `yaml:"level"`
}

type AgentPlatformConfig struct {
	BaseURL      string `yaml:"base_url"`
	CallbackPath string `yaml:"callback_path"`
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

type RedisConfig struct {
	Address string `yaml:"address"`
}

type MigrationConfig struct {
	Directory string `yaml:"directory"`
	AutoApply bool   `yaml:"auto_apply"`
	LockKey   string `yaml:"lock_key"`
}

type IngestionConfig struct {
	DefaultTimeoutSeconds int    `yaml:"default_timeout_seconds"`
	BatchSize             int    `yaml:"batch_size"`
	SchedulerTickSeconds  int    `yaml:"scheduler_tick_seconds"`
	SchedulerTimezone     string `yaml:"scheduler_timezone"`
}

type ObjectStoreConfig struct {
	Provider  string `yaml:"provider"`
	LocalPath string `yaml:"local_path"`
}

type RateLimitConfig struct {
	DefaultRequestsPerMinute int `yaml:"default_requests_per_minute"`
}

type SecurityConfig struct {
	JWTIssuer string `yaml:"jwt_issuer"`
}

type SecretConfig struct {
	AgentPlatformAPIKey string
	DatabaseURL         string
	DatabasePassword    string
	JWTSecret           string
	PaymentSecret       string
	CloudSecret         string
	AdminAPIToken       string
}

func Load() (Config, error) {
	env, err := ResolveEnvironment(os.Getenv("APP_ENV"))
	if err != nil {
		return Config{}, err
	}

	configDir := ResolveConfigDir(os.Getenv("TIDEWISE_CONFIG_DIR"))
	configPath := filepath.Join(configDir, fmt.Sprintf("config.%s.yaml", env))

	content, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %s: %w", configPath, err)
	}

	cfg.App.Env = env
	cfg.Secrets = SecretConfig{
		AgentPlatformAPIKey: os.Getenv("AGENT_PLATFORM_API_KEY"),
		DatabaseURL:         firstEnv("TIDEWISE_DATABASE_URL", "DATABASE_URL"),
		DatabasePassword:    os.Getenv("DATABASE_PASSWORD"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		PaymentSecret:       os.Getenv("PAYMENT_SECRET"),
		CloudSecret:         os.Getenv("CLOUD_SECRET"),
		AdminAPIToken:       os.Getenv("ADMIN_API_TOKEN"),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func ResolveEnvironment(value string) (Environment, error) {
	if value == "" {
		return EnvLocal, nil
	}

	env := Environment(value)
	switch env {
	case EnvLocal, EnvUAT, EnvProd:
		return env, nil
	default:
		return "", fmt.Errorf("unsupported APP_ENV %q", value)
	}
}

func ResolveConfigDir(value string) string {
	if value != "" {
		return value
	}

	if _, err := os.Stat("config"); err == nil {
		return "config"
	}

	return filepath.Join("backend", "config")
}

func (c Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if _, err := ResolveEnvironment(string(c.App.Env)); err != nil {
		return err
	}
	if c.Server.Host == "" {
		return fmt.Errorf("server.host is required")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Server.ReadTimeoutSeconds <= 0 {
		return fmt.Errorf("server.read_timeout_seconds must be positive")
	}
	if c.Server.WriteTimeoutSeconds <= 0 {
		return fmt.Errorf("server.write_timeout_seconds must be positive")
	}
	if c.Log.Level == "" {
		return fmt.Errorf("log.level is required")
	}
	if _, err := url.ParseRequestURI(c.AgentPlatform.BaseURL); err != nil {
		return fmt.Errorf("agent_platform.base_url must be a valid URL: %w", err)
	}
	if c.AgentPlatform.CallbackPath == "" || c.AgentPlatform.CallbackPath[0] != '/' {
		return fmt.Errorf("agent_platform.callback_path must start with /")
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
	if c.Redis.Address == "" {
		return fmt.Errorf("redis.address is required")
	}
	if c.Migration.Directory == "" {
		return fmt.Errorf("migration.directory is required")
	}
	if c.Migration.LockKey == "" {
		return fmt.Errorf("migration.lock_key is required")
	}
	if c.Ingestion.DefaultTimeoutSeconds <= 0 {
		return fmt.Errorf("ingestion.default_timeout_seconds must be positive")
	}
	if c.Ingestion.BatchSize <= 0 {
		return fmt.Errorf("ingestion.batch_size must be positive")
	}
	if c.Ingestion.SchedulerTickSeconds <= 0 {
		return fmt.Errorf("ingestion.scheduler_tick_seconds must be positive")
	}
	if c.Ingestion.SchedulerTimezone == "" {
		return fmt.Errorf("ingestion.scheduler_timezone is required")
	}
	if c.ObjectStore.Provider == "" {
		return fmt.Errorf("object_store.provider is required")
	}
	if c.ObjectStore.Provider == "local" && c.ObjectStore.LocalPath == "" {
		return fmt.Errorf("object_store.local_path is required when provider is local")
	}
	if c.RateLimit.DefaultRequestsPerMinute <= 0 {
		return fmt.Errorf("rate_limit.default_requests_per_minute must be positive")
	}
	if c.Security.JWTIssuer == "" {
		return fmt.Errorf("security.jwt_issuer is required")
	}

	return nil
}

func (s ServerConfig) Address() string {
	return net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))
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
