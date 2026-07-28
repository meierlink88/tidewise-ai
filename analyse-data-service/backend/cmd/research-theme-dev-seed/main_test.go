package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
)

func TestLoadBatchUsesFrozenImportContract(t *testing.T) {
	batch, err := loadBatch(filepath.Join("..", "..", "data", "research_themes", "local_homepage.json"))
	if err != nil {
		t.Fatal(err)
	}
	if batch.AnalysisBatchID != "20260728T-theme-reason-tree-v1-dev" || len(batch.Themes) != 1 {
		t.Fatalf("batch = %#v", batch)
	}
	if batch.Themes[0].ThemeKey != "ai-optical-module-demand" {
		t.Fatalf("first theme key = %q", batch.Themes[0].ThemeKey)
	}
}

func TestValidateLocalTarget(t *testing.T) {
	valid := []conf.Config{
		{App: conf.AppConfig{Env: conf.EnvLocal}, Database: conf.DatabaseConfig{Host: "postgres", Name: "tidewise_local"}},
		{App: conf.AppConfig{Env: conf.EnvLocal}, Database: conf.DatabaseConfig{Host: "127.0.0.1", Name: "tidewise_local"}},
	}
	for _, cfg := range valid {
		if err := validateLocalTarget(cfg); err != nil {
			t.Fatalf("valid local target error = %v", err)
		}
	}

	invalid := []conf.Config{
		{App: conf.AppConfig{Env: conf.EnvUAT}, Database: conf.DatabaseConfig{Host: "postgres", Name: "tidewise_local"}},
		{App: conf.AppConfig{Env: conf.EnvProd}, Database: conf.DatabaseConfig{Host: "db.prod", Name: "tidewise_prod"}},
		{App: conf.AppConfig{Env: conf.EnvLocal}, Database: conf.DatabaseConfig{Host: "postgres", Name: "shared_local"}},
		{App: conf.AppConfig{Env: conf.EnvLocal}, Database: conf.DatabaseConfig{Host: "db.prod", Name: "tidewise_local"}},
		{App: conf.AppConfig{Env: conf.EnvLocal}, Database: conf.DatabaseConfig{Host: "localhost", Name: "tidewise_prod"}},
	}
	for index, cfg := range invalid {
		if err := validateLocalTarget(cfg); err == nil || !strings.Contains(err.Error(), "local") {
			t.Fatalf("invalid target %d error = %v", index, err)
		}
	}
}
