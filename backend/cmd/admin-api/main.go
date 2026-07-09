package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/apps/adminapi"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/database"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/dbmigration"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	migrationCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Database.ConnectTimeoutSeconds)*time.Second)
	defer cancel()

	migrationReport, err := dbmigration.CheckPostgres(migrationCtx, cfg, cfg.Migration.AutoApply)
	if err != nil {
		log.Fatalf("check migrations: %v", err)
	}
	if !cfg.Migration.AutoApply && len(migrationReport.Pending) > 0 {
		log.Fatalf("pending migrations exist and migration.auto_apply is false")
	}

	db, err := database.Open(context.Background(), cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	repo := repositories.NewPostgresRepository(db)
	server := newAdminServer(cfg, adminapi.NewRouter(repo, cfg.Secrets.AdminAPIToken))

	log.Printf("starting %s admin api on %s in %s", cfg.App.Name, cfg.Server.Address(), cfg.App.Env)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func newAdminServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}
}
