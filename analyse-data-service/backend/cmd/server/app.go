package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/transport"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/adminquery"
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/event"
	eventsemanticbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
	evidencebiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidence"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchgraph"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	adminquerydata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/adminquery"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/dbmigration"
	eventdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/event"
	eventsemanticdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/eventsemantic"
	evidencedata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/evidence"
	neo4jdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/neo4j"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
	researchdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/research"
	researchanalysiscontextdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/researchanalysiscontext"
	researchgraphdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/researchgraph"
	researchpublicationdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/server"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service"
	eventservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service/event"
	eventsemanticservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service/eventsemantic"
	evidenceservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service/evidence"
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

	var neo4jHealth *neo4jdata.HealthProbe
	if config.Secrets.Neo4jHealthUsername != "" {
		neo4jHealth, err = neo4jdata.NewHealthProbe(neo4jdata.HealthConfig{
			URI: config.Neo4jHealth.URI, Database: config.Neo4jHealth.Database,
			Username: config.Secrets.Neo4jHealthUsername, Password: config.Secrets.Neo4jHealthPassword,
			Timeout: time.Duration(config.Neo4jHealth.TimeoutSeconds) * time.Second,
		})
		if err != nil {
			_ = db.Close()
			return nil, nil, fmt.Errorf("configure Data Neo4j health probe: %w", err)
		}
	}
	closeBuildResources := func(buildErr error) error {
		var neo4jCloseErr error
		if neo4jHealth != nil {
			neo4jCloseErr = neo4jHealth.Close(context.Background())
		}
		return errors.Join(buildErr, neo4jCloseErr, db.Close())
	}
	evidenceStore, err := evidencedata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Evidence store: %w", err))
	}
	evidenceUseCase, err := evidencebiz.NewUseCase(evidenceStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Evidence use case: %w", err))
	}
	eventStore, err := eventdata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Event store: %w", err))
	}
	eventUseCase, err := eventbiz.NewUseCase(eventStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Event use case: %w", err))
	}
	eventSemanticStore, err := eventsemanticdata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Event Semantic store: %w", err))
	}
	eventSemanticUseCase, err := eventsemanticbiz.NewUseCase(eventSemanticStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Event Semantic use case: %w", err))
	}

	application := service.NewDataService(service.Dependencies{
		ResearchThemeImports:    researchpublication.NewService(researchpublicationdata.NewRepository(db)),
		Research:                research.NewService(researchdata.NewRepository(db), time.Now),
		ResearchAnalysisContext: researchanalysiscontext.NewService(researchanalysiscontextdata.NewRepository(db)),
		ResearchGraph:           researchgraph.NewService(researchgraphdata.NewRepository(db)),
		Admin:                   adminquery.NewService(adminquerydata.NewRepository(db)),
		RuntimeHealth:           runtimehealth.New(neo4jHealth, time.Now),
	})
	evidenceApplication, err := evidenceservice.NewService(evidenceUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Evidence API service: %w", err))
	}
	eventApplication, err := eventservice.NewService(eventUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Event API service: %w", err))
	}
	eventSemanticApplication, err := eventsemanticservice.NewService(eventSemanticUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Event Semantic API service: %w", err))
	}
	httpServer, err := server.NewHTTPServer(config, application, eventApplication, eventSemanticApplication, evidenceApplication, authenticator, logger)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure HTTP server: %w", err))
	}

	return newApp(httpServer, logger), func(ctx context.Context) error {
		var neo4jErr error
		if neo4jHealth != nil {
			neo4jErr = neo4jHealth.Close(ctx)
		}
		return errors.Join(neo4jErr, db.Close())
	}, nil
}

func buildAuthenticator(config conf.Config) (*server.Authenticator, error) {
	credentials := []server.Credential{
		{
			Secret: config.Secrets.ServiceToken,
			Principal: v1.Principal{Identity: "tidewise-internal-service", Scopes: []string{
				server.ScopeReviewedEventImport,
				server.ScopeRawEvidenceImport,
				server.ScopeEvidenceImport,
				server.ScopeEventTagRead,
				server.ScopeEventSemanticsRead,
				server.ScopeEventSemanticsWrite,
				server.ScopeResearchImport,
				server.ScopeResearchRead,
				server.ScopeAdminRead,
			}},
		},
	}
	authenticator, err := server.NewAuthenticator(credentials)
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
