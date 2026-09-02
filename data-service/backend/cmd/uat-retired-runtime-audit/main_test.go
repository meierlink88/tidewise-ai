package main

import (
	"testing"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
)

func TestValidateTargetRejectsAnythingExceptCurrentUATDataIdentity(t *testing.T) {
	valid := conf.Config{
		App: conf.AppConfig{Env: conf.EnvUAT},
		Database: conf.DatabaseConfig{
			Name:    currentDatabase,
			User:    currentRole,
			SSLMode: "require",
		},
	}
	if err := validateTarget(valid); err != nil {
		t.Fatalf("validateTarget(valid) error = %v", err)
	}

	for name, mutate := range map[string]func(*conf.Config){
		"environment": func(cfg *conf.Config) { cfg.App.Env = conf.EnvLocal },
		"database":    func(cfg *conf.Config) { cfg.Database.Name = retiredDatabase },
		"role":        func(cfg *conf.Config) { cfg.Database.User = retiredRole },
		"ssl mode":    func(cfg *conf.Config) { cfg.Database.SSLMode = "disable" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if err := validateTarget(candidate); err == nil {
				t.Fatal("validateTarget() error = nil, want rejection")
			}
		})
	}
}
