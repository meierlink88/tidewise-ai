package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
)

func TestLoadBatchUsesFrozenImportContract(t *testing.T) {
	batch, err := loadBatch(filepath.Join("..", "..", "..", "..", "data", "research_themes", "local_homepage.json"))
	if err != nil {
		t.Fatal(err)
	}
	if batch.AnalysisBatchID != "20260718T-v6-72h-validation-home-dev" || len(batch.Themes) != 3 {
		t.Fatalf("batch = %#v", batch)
	}
	if batch.Themes[0].ThemeKey != "ai-application-commercialization" {
		t.Fatalf("first theme key = %q", batch.Themes[0].ThemeKey)
	}
}

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
