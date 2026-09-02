package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/runtimeaudit"
)

const (
	currentDatabase = "tidewise_uat"
	currentRole     = "tidewise_uat"
	retiredDatabase = "tidewise_ai_server"
	retiredRole     = "agentrun_uat"
)

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		log.Printf("UAT retired runtime audit failed: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, output io.Writer) (runErr error) {
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := validateTarget(cfg); err != nil {
		return fmt.Errorf("validate target: %w", err)
	}

	operationContext, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	database, err := data.OpenPostgres(operationContext, cfg)
	if err != nil {
		return errors.New("connect to current UAT Data database: connection failed")
	}
	defer func() {
		runErr = errors.Join(runErr, database.Close())
	}()
	store, err := runtimeaudit.NewStore(database)
	if err != nil {
		return fmt.Errorf("configure runtime audit store: %w", err)
	}
	report, err := store.Inspect(operationContext, retiredDatabase, retiredRole)
	if err != nil {
		return fmt.Errorf("inspect retired database objects: %w", err)
	}
	if report.CurrentDatabase != currentDatabase || report.CurrentRole != currentRole {
		return fmt.Errorf("connected identity must be %s on %s", currentRole, currentDatabase)
	}
	if err := json.NewEncoder(output).Encode(report); err != nil {
		return fmt.Errorf("encode audit report: %w", err)
	}
	return nil
}

func validateTarget(cfg conf.Config) error {
	if cfg.App.Env != conf.EnvUAT {
		return fmt.Errorf("APP_ENV must be uat")
	}
	if cfg.Database.Name != currentDatabase || cfg.Database.User != currentRole {
		return fmt.Errorf("audit must connect as %s to %s", currentRole, currentDatabase)
	}
	if cfg.Database.SSLMode != "require" {
		return fmt.Errorf("audit requires ssl_mode=require")
	}
	return nil
}
