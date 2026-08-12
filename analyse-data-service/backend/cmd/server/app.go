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
	entitybiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entity"
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/event"
	eventsemanticbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
	evidencebiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidence"
	rawdocumentbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/rawdocument"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/dbmigration"
	entitydata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/entity"
	eventdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/event"
	eventsemanticdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/eventsemantic"
	evidencedata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/evidence"
	rawdocumentdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/rawdocument"
	researchdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/server"
	eventservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service/event"
	eventsemanticservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service/eventsemantic"
	evidenceservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service/evidence"
	rawdocumentservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service/rawdocument"
	researchservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service/research"
	runtimehealthservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service/runtimehealth"
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
	db, err := data.OpenPostgres(databaseContext, config)
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

	closeBuildResources := func(buildErr error) error {
		return errors.Join(buildErr, db.Close())
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
	rawDocumentStore, err := rawdocumentdata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure RawDocument store: %w", err))
	}
	rawDocumentUseCase, err := rawdocumentbiz.NewUseCase(rawDocumentStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure RawDocument use case: %w", err))
	}
	entityStore, err := entitydata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Entity store: %w", err))
	}
	entityUseCase, err := entitybiz.NewUseCase(entityStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Entity use case: %w", err))
	}
	researchStore, err := researchdata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Research store: %w", err))
	}
	researchUseCase, err := research.NewUseCase(
		researchStore,
		researchStore,
		eventUseCase,
		eventSemanticUseCase,
		entityUseCase,
		time.Now,
	)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Research use case: %w", err))
	}

	runtimeHealthApplication, err := runtimehealthservice.NewService(runtimehealth.New(time.Now))
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Runtime Health API service: %w", err))
	}
	researchApplication, err := researchservice.NewService(researchUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Research API service: %w", err))
	}
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
	rawDocumentApplication, err := rawdocumentservice.NewService(rawDocumentUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure RawDocument API service: %w", err))
	}
	httpServer, err := server.NewHTTPServer(config, runtimeHealthApplication, researchApplication, eventApplication, eventSemanticApplication, evidenceApplication, rawDocumentApplication, authenticator, logger)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure HTTP server: %w", err))
	}

	return newApp(httpServer, logger), func(ctx context.Context) error {
		return db.Close()
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
