package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	eventapp "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/apps/miniappapi"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/database"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/dbmigration"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
	"github.com/meierlink88/tidewise-ai/backend/services/data"
	"github.com/meierlink88/tidewise-ai/backend/services/data/internalapi"
	"github.com/meierlink88/tidewise-ai/backend/services/data/rawimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/rawimport/postgresstore"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	authenticator, err := buildAuthenticator(cfg)
	if err != nil {
		log.Fatalf("configure service identities: %v", err)
	}

	connectTimeout := time.Duration(cfg.Database.ConnectTimeoutSeconds) * time.Second
	databaseCtx, cancelDatabase := context.WithTimeout(context.Background(), connectTimeout)
	db, err := database.Open(databaseCtx, cfg)
	cancelDatabase()
	if err != nil {
		log.Fatalf("open Data PostgreSQL: %v", err)
	}
	defer db.Close()
	readinessCtx, cancelReadiness := context.WithTimeout(context.Background(), connectTimeout)
	_, err = dbmigration.RequirePostgresReadyReadOnly(readinessCtx, db, cfg.Migration.Directory)
	cancelReadiness()
	if err != nil {
		log.Fatalf("check read-only migration readiness: %v", err)
	}
	repository := repositories.NewPostgresRepository(db)
	api := internalapi.NewHandler(internalapi.Dependencies{
		Authenticator:  authenticator,
		RawImports:     rawimport.NewService(postgresstore.New(db), time.Now),
		ReviewedEvents: eventapp.NewService(repository),
		Research:       miniappapi.NewResearchService(repository, time.Now),
		Admin:          repository,
		SourceMetadata: repository,
	})

	server := data.NewServer(cfg, api)
	log.Printf("starting %s on %s in %s", data.ServiceName, server.Addr, cfg.App.Env)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func buildAuthenticator(cfg config.Config) (*internalapi.Authenticator, error) {
	credentials := []internalapi.Credential{
		{
			Secret: cfg.Secrets.DataServiceAgentToken,
			Principal: internalapi.Principal{Identity: "agent-run", Scopes: []string{
				internalapi.ScopeRawImport,
				internalapi.ScopeReviewedEventImport,
				internalapi.ScopeSourceMetadataRead,
			}},
		},
		{
			Secret:    cfg.Secrets.DataServiceMiniappToken,
			Principal: internalapi.Principal{Identity: "miniapp-bff", Scopes: []string{internalapi.ScopeResearchRead}},
		},
		{
			Secret:    cfg.Secrets.DataServiceAdminToken,
			Principal: internalapi.Principal{Identity: "admin-portal-bff", Scopes: []string{internalapi.ScopeAdminRead}},
		},
	}
	authenticator, err := internalapi.NewAuthenticator(credentials)
	if err != nil {
		return nil, fmt.Errorf("build Data authenticator: %w", err)
	}
	return authenticator, nil
}
