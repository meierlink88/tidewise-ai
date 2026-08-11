package conf

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

const ServiceName = "data"
const ServiceVersion = "1.0.0"

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

type Config struct {
	App         AppConfig         `yaml:"app"`
	Server      ServerConfig      `yaml:"server"`
	Log         LogConfig         `yaml:"log"`
	Database    DatabaseConfig    `yaml:"database"`
	Neo4jHealth Neo4jHealthConfig `yaml:"neo4j_health"`
	Migration   MigrationConfig   `yaml:"migration"`
	Secrets     SecretConfig      `yaml:"-"`
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

type MigrationConfig struct {
	Directory string `yaml:"directory"`
	AutoApply bool   `yaml:"auto_apply"`
	LockKey   string `yaml:"lock_key"`
}

type Neo4jHealthConfig struct {
	URI            string `yaml:"uri"`
	Database       string `yaml:"database"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type SecretConfig struct {
	DatabasePassword    string
	ServiceToken        string
	Neo4jHealthUsername string
	Neo4jHealthPassword string
}

func Load() (Config, error) {
	cfg, err := loadConfiguration()
	if err != nil {
		return Config{}, err
	}
	if err := cfg.validateRuntimeSecrets(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LoadDatabaseOperation loads Data-owned database configuration without requiring
// the HTTP service identity token. It is intended for short-lived offline commands.
func LoadDatabaseOperation() (Config, error) {
	cfg, err := loadConfiguration()
	if err != nil {
		return Config{}, err
	}
	if cfg.Secrets.DatabasePassword == "" {
		return Config{}, fmt.Errorf("TIDEWISW_DB_PASSWORD is required for database operations")
	}
	if cfg.App.Env == EnvUAT && cfg.Database.SSLMode != "require" {
		return Config{}, fmt.Errorf("uat database configuration must use ssl_mode=require")
	}
	cfg.Secrets.ServiceToken = ""
	return cfg, nil
}

func loadConfiguration() (Config, error) {
	env, err := resolveEnvironment(os.Getenv("APP_ENV"))
	if err != nil {
		return Config{}, err
	}

	configDir := resolveConfigDir(os.Getenv("TIDEWISE_CONFIG_DIR"))
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
		DatabasePassword:    os.Getenv("TIDEWISW_DB_PASSWORD"),
		ServiceToken:        os.Getenv("DATA_SERVICE_TOKEN"),
		Neo4jHealthUsername: os.Getenv("DATA_NEO4J_HEALTH_USERNAME"),
		Neo4jHealthPassword: os.Getenv("DATA_NEO4J_HEALTH_PASSWORD"),
	}
	if host := os.Getenv("TIDEWISE_DB_HOST"); host != "" {
		cfg.Database.Host = host
	}
	if uri := os.Getenv("DATA_NEO4J_HEALTH_URI"); uri != "" {
		cfg.Neo4jHealth.URI = uri
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validateRuntimeSecrets() error {
	if c.Secrets.ServiceToken == "" {
		return fmt.Errorf("DATA_SERVICE_TOKEN is required")
	}
	configuredNeo4jFields := 0
	for _, value := range []string{c.Secrets.Neo4jHealthUsername, c.Secrets.Neo4jHealthPassword} {
		if value != "" {
			configuredNeo4jFields++
		}
	}
	if configuredNeo4jFields != 0 && configuredNeo4jFields != 2 {
		return fmt.Errorf("DATA_NEO4J_HEALTH_USERNAME and DATA_NEO4J_HEALTH_PASSWORD must be configured together")
	}
	if c.App.Env != EnvUAT {
		return nil
	}
	if c.Secrets.DatabasePassword == "" {
		return fmt.Errorf("TIDEWISW_DB_PASSWORD is required in uat")
	}
	if c.Database.SSLMode != "require" {
		return fmt.Errorf("uat database configuration must use ssl_mode=require")
	}
	if configuredNeo4jFields != 2 {
		return fmt.Errorf("separate Data Neo4j health probe credentials are required in uat")
	}
	return nil
}

func (c Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if _, err := resolveEnvironment(string(c.App.Env)); err != nil {
		return err
	}
	if err := c.Server.Validate(); err != nil {
		return err
	}
	if c.Log.Level == "" {
		return fmt.Errorf("log.level is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("postgres.host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("postgres.port must be between 1 and 65535")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("postgres.name is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("postgres.user is required")
	}
	if c.Database.SSLMode == "" {
		return fmt.Errorf("postgres.ssl_mode is required")
	}
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("postgres.max_open_conns must be positive")
	}
	if c.Database.MaxIdleConns <= 0 {
		return fmt.Errorf("postgres.max_idle_conns must be positive")
	}
	if c.Database.ConnMaxLifetimeSeconds <= 0 {
		return fmt.Errorf("postgres.conn_max_lifetime_seconds must be positive")
	}
	if c.Database.ConnectTimeoutSeconds <= 0 {
		return fmt.Errorf("postgres.connect_timeout_seconds must be positive")
	}
	neo4jURI, neo4jErr := url.Parse(c.Neo4jHealth.URI)
	if neo4jErr != nil || neo4jURI.Host == "" || neo4jURI.User != nil ||
		!validNeo4jScheme(neo4jURI.Scheme) || c.Neo4jHealth.Database == "" || c.Neo4jHealth.TimeoutSeconds <= 0 {
		return fmt.Errorf("neo4j_health uri, database and positive timeout_seconds are required")
	}
	if c.Migration.Directory == "" {
		return fmt.Errorf("migration.directory is required")
	}
	if c.Migration.LockKey == "" {
		return fmt.Errorf("migration.lock_key is required")
	}
	return nil
}

func validNeo4jScheme(scheme string) bool {
	switch scheme {
	case "bolt", "bolt+s", "bolt+ssc", "neo4j", "neo4j+s", "neo4j+ssc":
		return true
	default:
		return false
	}
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
