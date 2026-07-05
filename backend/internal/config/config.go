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
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Name string `yaml:"name"`
}

type RedisConfig struct {
	Address string `yaml:"address"`
}

type SecurityConfig struct {
	JWTIssuer string `yaml:"jwt_issuer"`
}

type SecretConfig struct {
	AgentPlatformAPIKey string
	DatabasePassword    string
	JWTSecret           string
	PaymentSecret       string
	CloudSecret         string
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
		DatabasePassword:    os.Getenv("DATABASE_PASSWORD"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		PaymentSecret:       os.Getenv("PAYMENT_SECRET"),
		CloudSecret:         os.Getenv("CLOUD_SECRET"),
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
	if c.Redis.Address == "" {
		return fmt.Errorf("redis.address is required")
	}
	if c.Security.JWTIssuer == "" {
		return fmt.Errorf("security.jwt_issuer is required")
	}

	return nil
}

func (s ServerConfig) Address() string {
	return net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))
}
