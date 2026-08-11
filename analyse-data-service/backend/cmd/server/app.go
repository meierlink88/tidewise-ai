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
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventtagcatalog"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidencepublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchgraph"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	adminquerydata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/adminquery"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/dbmigration"
	eventpublicationdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/eventpublication"
	evidencepublicationdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/evidencepublication"
	neo4jdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/neo4j"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
	researchdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/research"
	researchanalysiscontextdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/researchanalysiscontext"
	researchgraphdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/researchgraph"
	researchpublicationdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/researchpublication"
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
	evidencePublications, err := evidencepublication.NewService(evidencepublicationdata.NewRepository(db))
	if err != nil {
		var neo4jCloseErr error
		if neo4jHealth != nil {
			neo4jCloseErr = neo4jHealth.Close(context.Background())
		}
		return nil, nil, errors.Join(
			fmt.Errorf("configure Evidence Publication service: %w", err),
			neo4jCloseErr,
			db.Close(),
		)
	}

	application := service.NewDataService(service.Dependencies{
		EventPublications:       eventpublication.NewService(eventpublicationdata.NewRepository(db)),
		EvidencePublications:    evidencePublications,
		EventTagCatalog:         eventtagcatalog.NewService(postgres.NewEventTagCatalogRepository(db)),
		EventSemantics:          eventsemantics.NewService(postgres.NewEventSemanticsStore(db)),
		ResearchThemeImports:    researchpublication.NewService(researchpublicationdata.NewRepository(db)),
		Research:                research.NewService(researchdata.NewRepository(db), time.Now),
		ResearchAnalysisContext: researchanalysiscontext.NewService(researchanalysiscontextdata.NewRepository(db)),
		ResearchGraph:           researchgraph.NewService(researchgraphdata.NewRepository(db)),
		Admin:                   adminquery.NewService(adminquerydata.NewRepository(db)),
		RuntimeHealth:           runtimehealth.New(neo4jHealth, time.Now),
	})
	httpServer := server.NewHTTPServer(config, application, authenticator, logger)

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
