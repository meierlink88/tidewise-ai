package main

import (
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
)

func TestValidateLocalTarget(t *testing.T) {
	valid := []config.Config{
		{App: config.AppConfig{Env: config.EnvLocal}, Database: config.DatabaseConfig{Host: "postgres", Name: "tidewise_local"}},
		{App: config.AppConfig{Env: config.EnvLocal}, Secrets: config.SecretConfig{DatabaseURL: "postgres://user:secret@127.0.0.1:5432/tidewise_local?sslmode=disable"}},
	}
	for _, cfg := range valid {
		if err := validateLocalTarget(cfg); err != nil {
			t.Fatalf("valid local target error = %v", err)
		}
	}

	invalid := []config.Config{
		{App: config.AppConfig{Env: config.EnvUAT}, Database: config.DatabaseConfig{Host: "postgres", Name: "tidewise_local"}},
		{App: config.AppConfig{Env: config.EnvProd}, Database: config.DatabaseConfig{Host: "db.prod", Name: "tidewise_prod"}},
		{App: config.AppConfig{Env: config.EnvLocal}, Database: config.DatabaseConfig{Host: "postgres", Name: "shared_local"}},
		{App: config.AppConfig{Env: config.EnvLocal}, Secrets: config.SecretConfig{DatabaseURL: "postgres://user:secret@db.prod:5432/tidewise_local"}},
		{App: config.AppConfig{Env: config.EnvLocal}, Secrets: config.SecretConfig{DatabaseURL: "postgres://user:secret@localhost:5432/tidewise_prod"}},
	}
	for index, cfg := range invalid {
		if err := validateLocalTarget(cfg); err == nil || !strings.Contains(err.Error(), "local") {
			t.Fatalf("invalid target %d error = %v", index, err)
		}
	}
}
