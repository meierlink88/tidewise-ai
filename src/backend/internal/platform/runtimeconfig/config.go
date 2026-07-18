// Package runtimeconfig provides business-free process configuration primitives.
package runtimeconfig

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

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

func ResolveEnvironment(value string) (Environment, error) {
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

func ResolveConfigDir(explicit string, serviceRelativeDir string) string {
	if explicit != "" {
		return explicit
	}
	if _, err := os.Stat(serviceRelativeDir); err == nil {
		return serviceRelativeDir
	}
	return filepath.Join("src", "backend", serviceRelativeDir)
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
