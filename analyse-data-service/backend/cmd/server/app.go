package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/transport"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/adminquery"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanchorimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	adminquerydata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/adminquery"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/dbmigration"
	eventpublicationdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/eventpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
	researchdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/research"
	researchanchorimportdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/researchanchorimport"
	researchthemeimportdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/server"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service"
)

const applicationStopTimeout = 10 * time.Second
const resourceCleanupTimeout = 5 * time.Second

func buildApp(config conf.Config, logger *slog.Logger) (*kratos.App, func(context.Context) error, error) {
	authenticator, err := buildAuthenticator(config)
	if err != nil {
		return nil, nil, err
	}

	connectTimeout := time.Duration(config.Database.ConnectTimeoutSeconds) * time.Second
	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), connectTimeout)
	db, err := postgres.Open(databaseContext, config)
	cancelDatabase()
	if err != nil {
		return nil, nil, fmt.Errorf("open Data PostgreSQL: %w", err)
	}

	readinessContext, cancelReadiness := context.WithTimeout(context.Background(), connectTimeout)
	_, err = dbmigration.RequirePostgresReadyReadOnly(readinessContext, db, config.Migration.Directory)
	cancelReadiness()
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("check read-only migration readiness: %w", err)
	}

	application := service.NewDataService(service.Dependencies{
		EventPublications:     eventpublication.NewService(eventpublicationdata.NewRepository(db)),
		ResearchThemeImports:  researchthemeimport.NewService(researchthemeimportdata.NewRepository(db)),
		ResearchAnchorImports: researchanchorimport.NewService(researchanchorimportdata.NewRepository(db)),
		Research:              research.NewService(researchdata.NewRepository(db), time.Now),
		Admin:                 adminquery.NewService(adminquerydata.NewRepository(db)),
	})
	httpServer := server.NewHTTPServer(config, application, authenticator, logger)

	return newApp(httpServer, logger), func(context.Context) error {
		return db.Close()
	}, nil
}

func buildAuthenticator(config conf.Config) (*service.Authenticator, error) {
	credentials := []service.Credential{
		{
			Secret: config.Secrets.DataServiceAgentToken,
			Principal: service.Principal{Identity: "agent-run", Scopes: []string{
				service.ScopeReviewedEventImport,
			}},
		},
		{
			Secret: config.Secrets.DataServiceResearchPublisherToken,
			Principal: service.Principal{
				Identity: "research-theme-publisher",
				Scopes:   []string{service.ScopeResearchImport},
			},
		},
		{
			Secret: config.Secrets.DataServiceMiniappToken,
			Principal: service.Principal{
				Identity: "miniapp-bff",
				Scopes:   []string{service.ScopeResearchRead},
			},
		},
		{
			Secret: config.Secrets.DataServiceAdminToken,
			Principal: service.Principal{
				Identity: "admin-portal-bff",
				Scopes:   []string{service.ScopeAdminRead},
			},
		},
	}
	authenticator, err := service.NewAuthenticator(credentials)
	if err != nil {
		return nil, fmt.Errorf("build Data authenticator: %w", err)
	}
	return authenticator, nil
}

func newApp(httpServer transport.Server, logger *slog.Logger) *kratos.App {
	return kratos.New(
		kratos.Name(conf.ServiceName),
		kratos.Version(conf.ServiceVersion),
		kratos.Logger(logger),
		kratos.StopTimeout(applicationStopTimeout),
		kratos.Server(httpServer),
	)
}

func runApplication(app interface{ Run() error }, cleanup func(context.Context) error) error {
	runErr := app.Run()
	cleanupErr := runCleanup(cleanup, resourceCleanupTimeout)
	return errors.Join(runErr, cleanupErr)
}

func runCleanup(cleanup func(context.Context) error, timeout time.Duration) error {
	if cleanup == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- cleanup(ctx)
	}()

	select {
	case err := <-result:
		if err != nil {
			return fmt.Errorf("cleanup Data resources: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cleanup Data resources: %w", ctx.Err())
	}
}
